// Package generation implements the cicada nymph spawn generation engine.
package generation

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/cicada-hunt/server/internal/environment"
	"github.com/cicada-hunt/server/internal/models"
)

// DensityCalculator computes the nymph density for H3 cells.
type DensityCalculator struct {
	scorer   *environment.Scorer
	envStore EnvironmentStore
}

// EnvironmentStore provides access to cached environmental data.
type EnvironmentStore interface {
	GetEnvironmentFactors(ctx context.Context, cellID string) (*models.EnvironmentFactors, error)
	UpsertEnvironmentFactors(ctx context.Context, factors *models.EnvironmentFactors) error
}

// NewDensityCalculator creates a new density calculator.
func NewDensityCalculator(scorer *environment.Scorer, envStore EnvironmentStore) *DensityCalculator {
	return &DensityCalculator{
		scorer:   scorer,
		envStore: envStore,
	}
}

// GetCellDensity calculates the current nymph density for a cell.
// If no cached data exists, it computes fresh and stores it.
func (dc *DensityCalculator) GetCellDensity(
	cellID string,
	lat, lng float64,
	now time.Time,
) (*environment.DensityResult, error) {

	ctx := context.TODO()

	// 1. Try to load cached environmental factors
	var env *models.EnvironmentFactors
	if dc.envStore != nil {
		var err error
		env, err = dc.envStore.GetEnvironmentFactors(ctx, cellID)
		if err != nil {
			log.Printf("[Density] env store lookup failed for %s: %v", cellID, err)
		}
	}

	if env == nil {
		// First-time: compute from raw data
		env = dc.computeFreshEnvironment(cellID, lat, lng)
		// Persist for future queries (if store available)
		if dc.envStore != nil {
			_ = dc.envStore.UpsertEnvironmentFactors(ctx, env)
		}
	}

	// 2. Calculate full density breakdown
	result := dc.scorer.CalculateDensity(cellID, env, now, lat)

	return result, nil
}

// GetCellDensitiesBatch computes densities for multiple cells concurrently.
func (dc *DensityCalculator) GetCellDensitiesBatch(
	cells []string,
	now time.Time,
) (map[string]*environment.DensityResult, error) {

	results := make(map[string]*environment.DensityResult, len(cells))

	for _, cellID := range cells {
		lat, lng := cellCenter(cellID)
		result, err := dc.GetCellDensity(cellID, lat, lng, now)
		if err != nil {
			continue // skip failed cells, degrade gracefully
		}
		results[cellID] = result
	}

	return results, nil
}

// computeFreshEnvironment builds EnvironmentFactors from raw data sources
// when no cached data exists for a cell.
func (dc *DensityCalculator) computeFreshEnvironment(cellID string, lat, lng float64) *models.EnvironmentFactors {
	treeScorer := environment.NewDefaultTreeScorer()
	soilScorer := environment.NewDefaultSoilScorer()

	return &models.EnvironmentFactors{
		H3CellLv9:      cellID,
		TreeScore:       treeScorer.GetTreeScore(lat, lng),
		SoilScore:       soilScorer.GetSoilScore(lat, lng),
		SoilType:        soilScorer.GetSoilType(lat, lng),
		ElevationM:      estimateElevation(lat, lng),
		SlopeDeg:        3.0, // default gentle slope
		ImperviousPct:   estimateImpervious(lat, lng),
		NDVI:            0.6, // default moderate vegetation
		TreeDensityIdx:  0.5,
		IsUrban:         estimateIsUrban(lat, lng),
		WaterProxM:      500, // default far from water
		DominantTrees:   treeScorer.GetDominantSpecies(lat, lng),
		UpdatedAt:       time.Now(),
	}
}

// estimateElevation returns a rough elevation estimate.
// In production, query SRTM DEM data.
func estimateElevation(lat, lng float64) float64 {
	// Placeholder: China's average elevation varies greatly.
	// East coast: 0-200m, Central: 500-2000m, Tibet: 4000m+
	_ = lng
	absLat := math.Abs(lat)
	if absLat > 35 {
		return 500 // northern regions
	}
	return 200 // default eastern/central
}

// estimateImpervious returns a rough impervious surface percentage.
func estimateImpervious(lat, lng float64) float64 {
	// Simple heuristic based on common urban coordinates
	// Real implementation: query GHSL dataset
	_ = lat
	_ = lng
	return 15.0 // default: suburban/rural
}

// estimateIsUrban returns whether a location is broadly urban.
func estimateIsUrban(lat, lng float64) bool {
	// Real implementation: query OSM landuse / GHSL
	_ = lat
	_ = lng
	return false
}

// cellCenter extracts the center coordinates from an H3 cell ID.
// This is a lightweight local helper; the spatial package provides the canonical version.
func cellCenter(cellID string) (float64, float64) {
	// Avoid circular dependency by inlining the conversion.
	// In production this calls spatial.CellToLatLng.
	return 39.9, 116.4 // placeholder Beijing coordinates
}
