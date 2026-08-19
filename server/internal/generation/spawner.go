package generation

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/cicada-hunt/server/internal/models"
	"github.com/google/uuid"
)

// SpawnConfig controls the spawning parameters for a batch generation.
type SpawnConfig struct {
	MinSpacingM float64 // minimum distance between nymphs (default 3m)
	MaxAttempts int     // max placement attempts per nymph (default 30)
	CellAreaKm2 float64 // area of the cell being populated
}

// DefaultSpawnConfig returns sensible defaults for nymph spawning.
func DefaultSpawnConfig() SpawnConfig {
	return SpawnConfig{
		MinSpacingM: 3.0,
		MaxAttempts: 30,
	}
}

// SpawnResult holds the output of a spawn batch.
type SpawnResult struct {
	Spawns     []*models.NymphSpawn `json:"spawns"`
	TotalCount int                  `json:"total_count"`
	CellID     string               `json:"cell_id"`
	Generated  bool                 `json:"generated"` // false = loaded from cache
}

// GenerateNymphs creates a batch of nymph spawns for a given H3 cell.
//
// Algorithm: Weighted Poisson-disc sampling
// 1. Build a heatmap within the cell based on tree positions
// 2. Use rejection sampling with minimum spacing constraint
// 3. Assign individual attributes (species, size, depth, quality)
func GenerateNymphs(
	cellID string,
	targetCount int,
	treeData []TreeLocation,
	cfg SpawnConfig,
	now time.Time,
) (*SpawnResult, error) {

	if targetCount <= 0 {
		return &SpawnResult{
			Spawns:     []*models.NymphSpawn{},
			TotalCount: 0,
			CellID:     cellID,
			Generated:  true,
		}, nil
	}

	// Cap generation to prevent runaway
	if targetCount > 50 {
		targetCount = 50
	}

	// 1. Build internal heatmap for the cell
	heatmap := buildCellHeatmap(cellID, treeData)

	// 2. Generate spawn points using weighted rejection sampling
	points := generateSpawnPoints(cellID, targetCount, heatmap, cfg)

	// 3. Create nymph objects with randomized attributes
	spawns := make([]*models.NymphSpawn, len(points))
	for i, pt := range points {
		spawns[i] = createNymphSpawn(cellID, i, pt, now)
	}

	return &SpawnResult{
		Spawns:     spawns,
		TotalCount: len(spawns),
		CellID:     cellID,
		Generated:  true,
	}, nil
}

// TreeLocation represents a tree within an H3 cell.
type TreeLocation struct {
	Lat        float64
	Lng        float64
	Species    string
	Preference float64 // cicada preference score for this tree
	CanopyRadiusM float64
}

// cellHeatmap is a simplified internal heatmap for spawn point selection.
type cellHeatmap struct {
	centerLat float64
	centerLng float64
	radiusKm  float64 // cell radius approximation
	trees     []TreeLocation
}

func buildCellHeatmap(cellID string, trees []TreeLocation) *cellHeatmap {
	// In production: use spatial.CellToLatLng and spatial.CellBoundary
	_ = cellID
	return &cellHeatmap{
		centerLat: 39.9,
		centerLng: 116.4,
		radiusKm:  0.2, // ~400m cell edge → ~200m radius
		trees:     trees,
	}
}

// spawnPoint is a candidate nymph position within a cell.
type spawnPoint struct {
	Lat     float64
	Lng     float64
	Density float64 // local heatmap density at this point
}

func generateSpawnPoints(
	cellID string,
	targetCount int,
	heatmap *cellHeatmap,
	cfg SpawnConfig,
) []spawnPoint {
	_ = cellID

	var points []spawnPoint
	attempts := 0
	maxTotalAttempts := targetCount * cfg.MaxAttempts

	for len(points) < targetCount && attempts < maxTotalAttempts {
		attempts++

		// Generate a candidate position, biased toward trees if available
		candidate := generateCandidate(heatmap)

		// Check minimum spacing
		if !hasMinSpacing(candidate, points, cfg.MinSpacingM) {
			continue
		}

		// Acceptance probability based on local density
		acceptProb := candidate.Density
		if acceptProb < rand.Float64() {
			continue
		}

		points = append(points, candidate)
	}

	return points
}

// generateCandidate creates a candidate spawn position biased toward tree locations.
func generateCandidate(heatmap *cellHeatmap) spawnPoint {
	// If trees are available, place points near them (Gaussian kernel)
	if len(heatmap.trees) > 0 {
		// Weighted random tree selection
		totalWeight := 0.0
		for _, t := range heatmap.trees {
			totalWeight += t.Preference
		}

		if totalWeight > 0 {
			r := rand.Float64() * totalWeight
			cumulative := 0.0
			for _, t := range heatmap.trees {
				cumulative += t.Preference
				if r <= cumulative {
					// Place near this tree with Gaussian offset
					offsetM := math.Abs(rand.NormFloat64() * t.CanopyRadiusM)
					if offsetM > t.CanopyRadiusM*2 {
						offsetM = t.CanopyRadiusM * 2
					}
					angle := rand.Float64() * 2 * math.Pi

					// Approximate meter→degree conversion
					latOffset := (offsetM / 111320.0) * math.Cos(angle)
					lngOffset := (offsetM / (111320.0 * math.Cos(heatmap.centerLat*math.Pi/180.0))) * math.Sin(angle)

					return spawnPoint{
						Lat:     t.Lat + latOffset,
						Lng:     t.Lng + lngOffset,
						Density: t.Preference,
					}
				}
			}
		}
	}

	// Fallback: random position within cell bounds
	radiusDeg := heatmap.radiusKm / 111.0
	latOffset := (rand.Float64()*2 - 1) * radiusDeg
	lngOffset := (rand.Float64()*2 - 1) * radiusDeg

	return spawnPoint{
		Lat:     heatmap.centerLat + latOffset,
		Lng:     heatmap.centerLng + lngOffset,
		Density: 0.1, // low density without trees
	}
}

// hasMinSpacing checks if a candidate point respects minimum spacing from all existing points.
func hasMinSpacing(candidate spawnPoint, existing []spawnPoint, minDistM float64) bool {
	for _, p := range existing {
		// Approximate distance in meters
		dLat := (candidate.Lat - p.Lat) * 111320.0
		dLng := (candidate.Lng - p.Lng) * 111320.0 * math.Cos(candidate.Lat*math.Pi/180.0)
		dist := math.Sqrt(dLat*dLat + dLng*dLng)

		if dist < minDistM {
			return false
		}
	}
	return true
}

// createNymphSpawn creates a single NymphSpawn with randomized attributes.
func createNymphSpawn(cellID string, index int, pt spawnPoint, now time.Time) *models.NymphSpawn {
	species := weightedSpeciesSelection()
	cfg := models.NymphSpeciesConfigs[species]

	// Randomize size within species range
	sizeCm := cfg.SizeRangeMin + rand.Float64()*(cfg.SizeRangeMax-cfg.SizeRangeMin)
	sizeCm = math.Round(sizeCm*10) / 10

	// Randomize depth
	depthCm := cfg.DepthRangeMin + rand.Float64()*(cfg.DepthRangeMax-cfg.DepthRangeMin)
	// Bias toward preferred depth
	depthCm = (depthCm + cfg.PreferredDepth) / 2
	depthCm = math.Round(depthCm)

	// Quality: 1-5 stars, weighted toward 3
	quality := weightedQuality(species)

	// Rare mutation: 0.2% chance
	isRare := rand.Float64() < 0.002

	// Estimated value
	value := cfg.BaseValue * float64(quality) / 3.0
	if isRare {
		value *= 5
	}
	value = math.Round(value*100) / 100

	return &models.NymphSpawn{
		ID:          fmt.Sprintf("nym_%s_%d_%s", cellID[:8], index, uuid.New().String()[:6]),
		Lat:         math.Round(pt.Lat*1e7) / 1e7,
		Lng:         math.Round(pt.Lng*1e7) / 1e7,
		DepthCm:     depthCm,
		H3CellLv9:   cellID,
		H3CellLv11:  "", // filled by caller
		Species:     species,
		SpeciesName: cfg.Name,
		SizeCm:      sizeCm,
		WeightG:     math.Round(cfg.BaseWeight*1000*(sizeCm/((cfg.SizeRangeMin+cfg.SizeRangeMax)/2))*100) / 100,
		Quality:     quality,
		IsRare:      isRare,
		ValueEst:    value,
		Status:      models.NymphStatusActive,
		CreatedAt:   now,
		RefreshedAt: now,
	}
}

// weightedSpeciesSelection picks a nymph species based on spawn weights.
func weightedSpeciesSelection() models.NymphSpecies {
	total := models.TotalSpawnWeight()
	r := rand.Intn(total)

	cumulative := 0
	for species, cfg := range models.NymphSpeciesConfigs {
		cumulative += cfg.SpawnWeight
		if r < cumulative {
			return species
		}
	}

	return models.NymphBlackCicada // fallback
}

// weightedQuality assigns a quality rating weighted toward middle values.
func weightedQuality(species models.NymphSpecies) int {
	// Gaussian-like distribution centered on 3
	cfg := models.NymphSpeciesConfigs[species]

	// Base quality shifts with rarity
	baseMean := 3.0
	switch cfg.Rarity {
	case 5:
		baseMean = 4.0
	case 4:
		baseMean = 3.5
	case 3:
		baseMean = 3.2
	case 2:
		baseMean = 3.0
	default:
		baseMean = 2.8
	}

	// Weighted random: roll until acceptance
	roll := rand.Intn(5) + 1 // 1-5
	_ = baseMean              // Placeholder for more sophisticated distribution

	// Simple weighted approach
	weights := []int{5, 20, 40, 25, 10} // 1★:5%, 2★:20%, 3★:40%, 4★:25%, 5★:10%
	totalW := 0
	for _, w := range weights {
		totalW += w
	}
	r := rand.Intn(totalW)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			_ = roll
			return i + 1
		}
	}

	return 3
}
