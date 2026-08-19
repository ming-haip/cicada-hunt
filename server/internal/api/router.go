package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the HTTP router with all routes.
func NewRouter(
	nymphHandler *NymphHandler,
	heatmapHandler *HeatmapHandler,
	playerHandler *PlayerHandler,
	mobileDir string,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RecoverMiddleware)
	r.Use(LoggingMiddleware)
	r.Use(CORSMiddleware)
	r.Use(AuthMiddleware)

	// Health check
	r.Get("/health", healthHandler)
	r.Get("/api/v1/health", healthHandler)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Nymph endpoints
		r.Get("/nymphs", nymphHandler.QueryNearby)
		r.Get("/nymphs/{nymphID}", nymphHandler.GetNymph)
		r.Post("/nymphs/{nymphID}/dig", nymphHandler.DigNymph)

		// Cicada endpoints (stub)
		r.Get("/cicadas", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"message": "cicada endpoints coming soon"})
		})

		// Heatmap tiles
		r.Get("/heatmap/{z}/{x}/{y}.png", heatmapHandler.ServeTile)

		// Player endpoints
		r.Get("/player", playerHandler.GetProfile)
		r.Get("/player/inventory", playerHandler.GetInventory)
		r.Get("/player/daily-stats", playerHandler.GetDailyStats)

		// Environment info
		r.Get("/environment/cell/{cellID}", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"message": "environment endpoints coming soon"})
		})
	})

	// Serve mobile PWA (if mobileDir is provided)
	if mobileDir != "" {
		fileServer := http.FileServer(http.Dir(mobileDir))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, mobileDir+"/index.html")
		})
		r.Get("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, mobileDir+"/manifest.json")
		})
		r.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			http.ServeFile(w, r, mobileDir+"/sw.js")
		})
		r.Get("/css/*", func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix("/", fileServer).ServeHTTP(w, r)
		})
		r.Get("/js/*", func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix("/", fileServer).ServeHTTP(w, r)
		})
		r.Get("/icons/*", func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix("/", fileServer).ServeHTTP(w, r)
		})
	}

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"service": "cicada-hunt-server",
		"version": "0.1.0",
	})
}

// writeJSON is a helper to write JSON responses.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
