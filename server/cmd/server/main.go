// Package main is the entry point for the Cicada Hunt game server.
//
// It initializes all dependencies (config, database, cache, services)
// and starts the HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cicada-hunt/server/config"
	"github.com/cicada-hunt/server/internal/api"
	"github.com/cicada-hunt/server/internal/environment"
	"github.com/cicada-hunt/server/internal/generation"
	"github.com/cicada-hunt/server/internal/service"
	"github.com/cicada-hunt/server/internal/spatial"
	"github.com/cicada-hunt/server/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🦗 Cicada Hunt Server starting...")

	// =========================================
	// 1. Load configuration
	// =========================================
	cfg := config.Load()
	log.Printf("Config: env=%s port=%d maxDailyDigs=%d",
		cfg.Env, cfg.ServerPort, cfg.MaxDailyDigs)

	// =========================================
	// 2. Initialize database connection pool
	// =========================================
	ctx := context.Background()

	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		p, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Printf("WARNING: Database pool creation failed: %v", err)
		} else if err := p.Ping(ctx); err != nil {
			log.Printf("WARNING: Database ping failed: %v", err)
			p.Close()
		} else {
			pool = p
			defer pool.Close()
			log.Println("PostgreSQL connected")
		}
	}
	if pool == nil {
		log.Println("Running in DB-less mode — using in-memory stubs")
	}

	// =========================================
	// 3. Initialize Redis connection
	// =========================================
	var redisAdapter *store.RedisAdapter
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: Redis connection failed: %v", err)
		log.Println("Running in cache-less mode — cell data regenerated each request")
		rdb.Close()
		rdb = nil
	} else {
		defer rdb.Close()
		redisAdapter = store.NewRedisAdapter(rdb)
		log.Println("Redis connected")
	}

	// =========================================
	// 4. Initialize environment scoring engine
	// =========================================
	weatherClient := environment.NewMockWeatherClient()
	weatherClient.DefaultTemp = 28.0
	weatherClient.DefaultHumidity = 65.0

	treeScorer := environment.NewDefaultTreeScorer()
	soilScorer := environment.NewDefaultSoilScorer()

	scorer := &environment.Scorer{
		TreeScorer:    treeScorer,
		SoilScorer:    soilScorer,
		WeatherClient: weatherClient,
	}
	log.Println("Environment scorer ready")

	// =========================================
	// 5. Initialize nymph generation engine
	// =========================================
	var (
		nymphStorePtr  *store.NymphStore
		cacheManager   *service.CacheManager
		envStorePtr    *store.EnvironmentStore
	)

	if pool != nil {
		nymphStorePtr = store.NewNymphStore(pool)
		envStorePtr = store.NewEnvironmentStore(pool)
		log.Println("Using PostgreSQL persistence")
	} else {
		log.Println("Using in-memory nymph store (not persistent)")
	}

	if redisAdapter != nil {
		cacheManager = service.NewCacheManager(redisAdapter)
		log.Println("Using Redis caching")
	}

	// Convert to interfaces (nil pointers become nil interfaces)
	var nymphStore service.NymphStore
	if nymphStorePtr != nil {
		nymphStore = nymphStorePtr
	}
	var envStore generation.EnvironmentStore
	if envStorePtr != nil {
		envStore = envStorePtr
	}
	var cacheInterface service.NymphCache
	if cacheManager != nil {
		cacheInterface = cacheManager
	}

	densityCalc := generation.NewDensityCalculator(scorer, envStore)
	nymphSvc := service.NewNymphService(densityCalc, nymphStore, cacheInterface)
	log.Println("Nymph service ready")

	// =========================================
	// 7. Start background refresh (ecosystem recovery)
	// =========================================
	refreshMgr := generation.NewRefreshManager(densityCalc)
	go refreshMgr.Start(ctx)
	defer refreshMgr.Stop()

	// =========================================
	// 8. Seed initial environment data for nearby area
	// =========================================
	go seedInitialCells(ctx, densityCalc)

	// =========================================
	// 9. Initialize HTTP handlers and router
	// =========================================
	nymphHandler := api.NewNymphHandler(nymphSvc, cfg)
	heatmapHandler := api.NewHeatmapHandler()
	playerHandler := api.NewPlayerHandler()

	// Determine mobile app directory (from env or default relative path)
	mobileDir := os.Getenv("MOBILE_DIR")
	if mobileDir == "" {
		// Try default paths
		for _, p := range []string{"../mobile", "./mobile", "../../mobile"} {
			if _, err := os.Stat(p); err == nil {
				mobileDir = p
				break
			}
		}
	}
	if mobileDir != "" {
		log.Printf("Serving mobile PWA from: %s", mobileDir)
	}

	router := api.NewRouter(nymphHandler, heatmapHandler, playerHandler, mobileDir)

	// =========================================
	// 10. Start HTTP server
	// =========================================
	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on http://localhost%s", cfg.Addr())
		log.Printf("API base: http://localhost%s/api/v1", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// =========================================
	// 11. Graceful shutdown
	// =========================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %v, shutting down...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// seedInitialCells pre-warms the environment data for a few cells near the default location.
func seedInitialCells(ctx context.Context, dc *generation.DensityCalculator) {
	// Beijing area representative cells
	defaultLat, defaultLng := 39.9042, 116.4074
	cells, err := spatial.CellsInRadius(defaultLat, defaultLng, 500, spatial.GridLevelDefault)
	if err != nil {
		return
	}

	log.Printf("Seeding %d initial cells near Beijing...", len(cells))
	now := time.Now()

	for i, cellID := range cells {
		if i >= 50 {
			break // Don't seed too many at startup
		}
		lat, lng := spatial.CellToLatLng(cellID)
		_, err := dc.GetCellDensity(cellID, lat, lng, now)
		if err != nil {
			log.Printf("Seed cell %s failed: %v", cellID, err)
		}
	}

	log.Println("Initial cell seeding complete")
}
