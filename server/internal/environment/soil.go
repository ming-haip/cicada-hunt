package environment

import (
	"math"

	"github.com/cicada-hunt/server/internal/models"
)

// DefaultSoilScorer provides soil suitability scores.
type DefaultSoilScorer struct {
	cellSoilScores map[string]float64
	cellSoilTypes  map[string]string
}

// NewDefaultSoilScorer creates a new soil scorer.
func NewDefaultSoilScorer() *DefaultSoilScorer {
	return &DefaultSoilScorer{
		cellSoilScores: make(map[string]float64),
		cellSoilTypes:  make(map[string]string),
	}
}

// LoadSoilData loads pre-computed soil data for a cell.
func (ss *DefaultSoilScorer) LoadSoilData(cellID string, soilScore float64, soilType string) {
	ss.cellSoilScores[cellID] = soilScore
	ss.cellSoilTypes[cellID] = soilType
}

// GetSoilScore returns the soil suitability score for a location.
func (ss *DefaultSoilScorer) GetSoilScore(lat, lng float64) float64 {
	// In production: query ISRIC SoilGrids or China CAS soil database.
	// For now, use a latitude-adjusted default.

	absLat := math.Abs(lat)

	// Permafrost and extreme cold regions: very low suitability
	if absLat > 70 {
		return 0.05
	}
	if absLat > 60 {
		return 0.15
	}
	if absLat > 55 {
		return 0.35
	}

	return 0.7 // default loam-like assumption for temperate regions
}

// GetSoilScoreForCell returns the preloaded soil score for a cell.
func (ss *DefaultSoilScorer) GetSoilScoreForCell(cellID string) float64 {
	if score, ok := ss.cellSoilScores[cellID]; ok {
		return score
	}
	return 0.65 // default
}

// GetSoilType returns the soil type at a location.
func (ss *DefaultSoilScorer) GetSoilType(lat, lng float64) string {
	// In production: query soil database.
	return "loam" // default assumption
}

// GetSoilScoreForType returns the cicada preference score for a given soil type.
func GetSoilScoreForType(soilType string) float64 {
	if score, ok := models.SoilPreferenceScore[soilType]; ok {
		return score
	}
	return 0.4
}

// CalculateImperviousAdjustment adjusts the soil score based on impervious surface percentage.
// High impervious surface (concrete, asphalt) → nymphs cannot emerge.
func CalculateImperviousAdjustment(soilScore, imperviousPct float64) float64 {
	// imperviousPct = 0 → full soil exposure (no adjustment)
	// imperviousPct = 100 → complete concrete (score = 0)

	permeableRatio := 1.0 - (imperviousPct / 100.0)
	return soilScore * permeableRatio
}
