// Package service implements the business logic layer for the cicada-hunt application.
package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/cicada-hunt/server/internal/generation"
	"github.com/cicada-hunt/server/internal/models"
	"github.com/cicada-hunt/server/internal/spatial"
	"github.com/google/uuid"
)

// NymphStore defines the persistence interface for nymph data.
type NymphStore interface {
	GetNymphsByCell(ctx context.Context, cellID string) ([]*models.NymphSpawn, error)
	SaveNymphs(ctx context.Context, nymphs []*models.NymphSpawn) error
	MarkNymphConsumed(ctx context.Context, nymphID, playerID string) error
	GetNymphByID(ctx context.Context, nymphID string) (*models.NymphSpawn, error)
}

// NymphCache defines the caching interface for nymph data.
type NymphCache interface {
	GetCellNymphs(ctx context.Context, cellID string) ([]*models.NymphSpawn, error)
	SetCellNymphs(ctx context.Context, cellID string, nymphs []*models.NymphSpawn, ttl time.Duration) error
	GetCellDensity(ctx context.Context, cellID string) (float64, error)
	SetCellDensity(ctx context.Context, cellID string, density float64, ttl time.Duration) error
	IncrPlayerDailyDigs(ctx context.Context, playerID string) (int64, error)
	GetPlayerDailyDigs(ctx context.Context, playerID string) (int64, error)
	SetGenCooldown(ctx context.Context, cellID string, ttl time.Duration) error
	CheckGenCooldown(ctx context.Context, cellID string) (bool, error)
}

// NymphService orchestrates the nymph spawning and digging workflows.
type NymphService struct {
	densityCalc  *generation.DensityCalculator
	nymphStore   NymphStore
	cache        NymphCache

	// Daily limits
	maxDailyDigs   int64
	maxNymphsPerCell int
}

// HasCache returns whether the cache layer is available.
func (s *NymphService) HasCache() bool {
	return s.cache != nil
}

// HasStore returns whether the persistence layer is available.
func (s *NymphService) HasStore() bool {
	return s.nymphStore != nil
}

// NewNymphService creates a new nymph service.
func NewNymphService(
	dc *generation.DensityCalculator,
	store NymphStore,
	cache NymphCache,
) *NymphService {
	return &NymphService{
		densityCalc:      dc,
		nymphStore:       store,
		cache:            cache,
		maxDailyDigs:     50,
		maxNymphsPerCell: 50,
	}
}

// QueryNearbyNymphs returns all active nymphs within a radius of the player.
func (s *NymphService) QueryNearbyNymphs(
	ctx context.Context,
	lat, lng float64,
	radiusM float64,
	maxResults int,
) (*models.NymphQueryResponse, error) {

	// 1. Get H3 cells covering the query area
	cells, err := spatial.CellsInRadius(lat, lng, radiusM, spatial.GridLevelDefault)
	if err != nil {
		return nil, fmt.Errorf("spatial query: %w", err)
	}

	// 2. Limit the number of cells processed
	if len(cells) > 20 {
		cells = cells[:20]
	}

	// 3. Collect nymphs from each cell
	now := time.Now()
	var allNymphs []*models.NymphSpawn
	var densityInfos []models.CellDensityInfo

	for _, cellID := range cells {
		// Get or generate nymphs for this cell
		nymphs, err := s.getOrGenerateCellNymphs(ctx, cellID, lat, lng, now)
		if err != nil {
			log.Printf("[NymphService] cell %s generation failed: %v", cellID, err)
			continue
		}

		// Filter to only active nymphs within radius
		for _, n := range nymphs {
			if n.Status == models.NymphStatusActive {
				dist := spatial.HaversineDistance(lat, lng, n.Lat, n.Lng)
				if dist <= radiusM {
					allNymphs = append(allNymphs, n)
				}
			}
		}

		// Collect density info
		var density float64
		if s.HasCache() {
			density, _ = s.cache.GetCellDensity(ctx, cellID)
		}
		densityInfos = append(densityInfos, models.CellDensityInfo{
			H3CellLv9:   cellID,
			CurrDensity: density,
		})
	}

	// 4. Sort by distance and truncate
	sortByDistance(allNymphs, lat, lng)
	if maxResults > 0 && len(allNymphs) > maxResults {
		allNymphs = allNymphs[:maxResults]
	}

	return &models.NymphQueryResponse{
		Nymphs:      allNymphs,
		DensityInfo: densityInfos,
		TotalInArea: len(allNymphs),
	}, nil
}

// DigNymph processes a player digging a specific nymph.
func (s *NymphService) DigNymph(
	ctx context.Context,
	playerID string,
	nymphID string,
	playerLat, playerLng float64,
	distanceM, deviationCm, angleDeg float64,
	toolUsed string,
) (*models.DigResponse, error) {

	resp := &models.DigResponse{}

	// 1. Check daily limit (if cache available)
	if s.HasCache() {
		dailyDigs, err := s.cache.GetPlayerDailyDigs(ctx, playerID)
		if err == nil && dailyDigs >= s.maxDailyDigs {
			resp.FailReason = "今日挖掘次数已用完，明天再来吧！"
			resp.FailReasonCode = "daily_limit_reached"
			return resp, nil
		}
	}

	// 2. Get the nymph (if store available)
	var nymph *models.NymphSpawn
	if s.HasStore() {
		var err error
		nymph, err = s.nymphStore.GetNymphByID(ctx, nymphID)
		if err != nil || nymph == nil {
			resp.FailReason = "找不到这只知了猴，可能已经被挖走了"
			resp.FailReasonCode = "nymph_not_found"
			return resp, nil
		}
	} else {
		resp.FailReason = "服务暂不可用，请稍后再试"
		resp.FailReasonCode = "service_unavailable"
		return resp, nil
	}

	if nymph.Status != models.NymphStatusActive {
		resp.FailReason = "这只知了猴已经被挖走了"
		resp.FailReasonCode = "nymph_already_consumed"
		return resp, nil
	}

	// 3. Distance check
	if distanceM > 2.0 {
		resp.FailReason = fmt.Sprintf("距离太远了（%.1fm），请靠近到2m以内", distanceM)
		resp.FailReasonCode = "too_far"
		return resp, nil
	}

	// 4. Direction check
	if angleDeg > 45 {
		resp.FailReason = "请将手机对准知了猴所在的方向"
		resp.FailReasonCode = "wrong_direction"
		return resp, nil
	}

	// 5. Calculate success rate
	tool := getToolStats(toolUsed)
	successRate := calculateDigSuccessRate(deviationCm, tool)

	resp.SuccessRate = round2Float(successRate)

	if rand.Float64() > successRate {
		resp.FailReason = fmt.Sprintf("差一点就挖到了！（成功率%.0f%%）", successRate*100)
		resp.FailReasonCode = "missed"
		return resp, nil
	}

	// 6. Success! Mark nymph as consumed
	if s.HasStore() {
		err := s.nymphStore.MarkNymphConsumed(ctx, nymphID, playerID)
		if err != nil {
			return nil, fmt.Errorf("mark consumed: %w", err)
		}
	}

	// 7. Increment daily counter
	if s.HasCache() {
		s.cache.IncrPlayerDailyDigs(ctx, playerID)
	}

	// 8. Build reward
	coinReward := int64(nymph.ValueEst)
	expReward := int64(nymph.Quality * 10)

	resp.Success = true
	resp.Nymph = nymph
	resp.CoinReward = coinReward
	resp.ExpReward = expReward

	return resp, nil
}

// getOrGenerateCellNymphs retrieves nymphs for a cell, generating them if needed.
func (s *NymphService) getOrGenerateCellNymphs(
	ctx context.Context,
	cellID string,
	refLat, refLng float64,
	now time.Time,
) ([]*models.NymphSpawn, error) {

	// 1. Check cache (if available)
	if s.HasCache() {
		nymphs, err := s.cache.GetCellNymphs(ctx, cellID)
		if err == nil && len(nymphs) > 0 {
			return nymphs, nil
		}

		// Check cooldown
		onCooldown, _ := s.cache.CheckGenCooldown(ctx, cellID)
		if onCooldown {
			return nil, nil
		}
	}

	// 2. Calculate density
	density, err := s.densityCalc.GetCellDensity(cellID, refLat, refLng, now)
	if err != nil {
		return nil, err
	}

	if density.FinalDensity <= 0.01 {
		return nil, nil // unsuitable area
	}

	// 3. Calculate target count
	cellAreaKm2 := spatial.CellAreaKm2(spatial.GridLevelDefault)
	targetCount := int(math.Round(density.FinalDensity * cellAreaKm2))

	if targetCount == 0 {
		return nil, nil
	}
	if targetCount > s.maxNymphsPerCell {
		targetCount = s.maxNymphsPerCell
	}

	// 4. Generate spawns
	cfg := generation.DefaultSpawnConfig()
	cfg.CellAreaKm2 = cellAreaKm2

	result, err := generation.GenerateNymphs(cellID, targetCount, nil, cfg, now)
	if err != nil {
		return nil, err
	}

	// 5. Persist (if store available)
	if s.HasStore() {
		if err := s.nymphStore.SaveNymphs(ctx, result.Spawns); err != nil {
			log.Printf("[NymphService] failed to persist nymphs for cell %s: %v", cellID, err)
		}
	}

	// 6. Cache (if available)
	if s.HasCache() {
		s.cache.SetCellNymphs(ctx, cellID, result.Spawns, 10*time.Minute)
		s.cache.SetCellDensity(ctx, cellID, density.FinalDensity, 1*time.Hour)
		s.cache.SetGenCooldown(ctx, cellID, 5*time.Minute)
	}

	return result.Spawns, nil
}

// Helper functions

func sortByDistance(nymphs []*models.NymphSpawn, lat, lng float64) {
	for i := 0; i < len(nymphs); i++ {
		for j := i + 1; j < len(nymphs); j++ {
			di := spatial.HaversineDistance(lat, lng, nymphs[i].Lat, nymphs[i].Lng)
			dj := spatial.HaversineDistance(lat, lng, nymphs[j].Lat, nymphs[j].Lng)
			if di > dj {
				nymphs[i], nymphs[j] = nymphs[j], nymphs[i]
			}
		}
	}
}

func calculateDigSuccessRate(deviationCm float64, tool models.ToolStats) float64 {
	baseRate := tool.Accuracy
	penalty := math.Pow(deviationCm/30.0, 2) // quadratic penalty
	rate := baseRate * (1.0 - penalty)
	return math.Max(0.05, math.Min(rate, 1.0))
}

func getToolStats(toolName string) models.ToolStats {
	tools := models.DefaultToolStats()
	if t, ok := tools[toolName]; ok {
		return t
	}
	return tools["bare_hand"]
}

func round2Float(v float64) float64 {
	return math.Round(v*100) / 100
}

// Discard unused imports
var _ = uuid.New
