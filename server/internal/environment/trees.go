package environment

import (
	"math"

	"github.com/cicada-hunt/server/internal/models"
)

// DefaultTreeScorer provides tree suitability scores using OSM data with a fallback.
type DefaultTreeScorer struct {
	// cellTreeScores caches pre-computed tree scores per H3 cell (Lv9).
	cellTreeScores map[string]float64
	// cellDominantTrees caches the dominant tree species per cell.
	cellDominantTrees map[string]string
}

// NewDefaultTreeScorer creates a new tree scorer with optional preloaded data.
func NewDefaultTreeScorer() *DefaultTreeScorer {
	return &DefaultTreeScorer{
		cellTreeScores:    make(map[string]float64),
		cellDominantTrees: make(map[string]string),
	}
}

// LoadCellTreeData loads pre-computed tree scores from the environment database.
func (ts *DefaultTreeScorer) LoadCellTreeData(cellID string, treeScore float64, dominantSpecies string) {
	ts.cellTreeScores[cellID] = treeScore
	ts.cellDominantTrees[cellID] = dominantSpecies
}

// GetTreeScore returns the tree suitability score for a location.
// Uses preloaded OSM/remote-sensing data, falling back to NDVI-based estimation.
func (ts *DefaultTreeScorer) GetTreeScore(lat, lng float64) float64 {
	// In production, this queries the pre-computed cell_environment table.
	// For now, return a sensible default that varies by NDVI estimation.

	// Use the lat/lng to estimate general vegetation presence.
	// Urban cores at certain coordinates → lower; parks → higher.
	// This is a placeholder; real implementation uses cell_environment DB.

	return estimateTreeScoreFromPosition(lat, lng)
}

// GetDominantSpecies returns the dominant tree species at a location.
func (ts *DefaultTreeScorer) GetDominantSpecies(lat, lng float64) string {
	// In production, queries the environment database.
	return "default_tree"
}

// GetTreeScoreForSpecies returns the cicada preference score for a given tree species.
func GetTreeScoreForSpecies(species string) float64 {
	if score, ok := models.TreePreferenceScore[species]; ok {
		return score
	}
	return models.TreePreferenceScore["default_tree"] // 0.40
}

// estimateTreeScoreFromPosition provides a rough tree score based on geographic position.
// This is a fallback when no OSM/remote-sensing data is available.
// REAL IMPLEMENTATION: query cell_environment table in PostgreSQL.
func estimateTreeScoreFromPosition(lat, lng float64) float64 {
	// General heuristic:
	// - Temperate latitudes (30-50°) → highest potential for deciduous trees
	// - Tropical latitudes (<23°) → mixed, depends on local ecology
	// - Very high latitudes (>60°) → low
	// - Extreme deserts → very low

	absLat := math.Abs(lat)

	baseScore := 0.5

	// Latitude adjustment
	switch {
	case absLat > 70:
		baseScore *= 0.05 // Arctic — essentially no trees
	case absLat > 60:
		baseScore *= 0.15 // tundra/taiga — very few deciduous trees
	case absLat > 50:
		baseScore *= 0.5
	case absLat >= 25 && absLat <= 45:
		baseScore *= 1.2 // optimal temperate zone
	case absLat < 20:
		baseScore *= 0.7 // tropics — can vary widely
	}

	return math.Max(0, math.Min(baseScore, 1.0))
}

// GetTreeScoreForCell returns the preloaded tree score for a specific H3 cell.
func (ts *DefaultTreeScorer) GetTreeScoreForCell(cellID string) float64 {
	if score, ok := ts.cellTreeScores[cellID]; ok {
		return score
	}
	return 0.4 // default moderate score
}
