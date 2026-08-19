// Package environment calculates cicada habitat suitability scores
// from real-world environmental data sources.
package environment

import (
	"math"
	"time"

	"github.com/cicada-hunt/server/internal/models"
)

// ---------------------------------------------------------------------------
// Top-level scorer: aggregates all seven environmental factors
// ---------------------------------------------------------------------------

// Scorer computes the overall nymph density for an H3 cell.
type Scorer struct {
	TreeScorer    TreeScorer
	SoilScorer    SoilScorer
	WeatherClient WeatherClient
}

// DensityResult holds the full breakdown of a density calculation.
type DensityResult struct {
	CellID        string  `json:"cell_id"`
	FinalDensity  float64 `json:"final_density"`  // nymphs per km²
	MaxDensity    float64 `json:"max_density"`    // theoretical max
	TreeScore     float64 `json:"tree_score"`
	SoilScore     float64 `json:"soil_score"`
	SeasonalScore float64 `json:"seasonal_score"`
	TimeScore     float64 `json:"time_score"`
	WeatherScore  float64 `json:"weather_score"`
	TerrainScore  float64 `json:"terrain_score"`
	UrbanScore    float64 `json:"urban_score"`
	Breakdown     string  `json:"breakdown"`
}

const (
	// MaxNymphDensity is the theoretical maximum nymph density per km².
	// This represents a high-yield poplar forest in peak season.
	MaxNymphDensity = 200.0 // nymphs/km²

	// Factor weights sum to 1.0.
	weightTree     = 0.30
	weightSoil     = 0.20
	weightSeasonal = 0.18
	weightTime     = 0.15
	weightWeather  = 0.10
	weightTerrain  = 0.05
	weightUrban    = 0.02
)

// CalculateDensity computes the full density breakdown for a cell.
func (s *Scorer) CalculateDensity(
	cellID string,
	env *models.EnvironmentFactors,
	now time.Time,
	lat float64,
) *DensityResult {

	dr := &DensityResult{
		CellID:     cellID,
		MaxDensity: MaxNymphDensity,
	}

	// 1. Tree score (30%)
	dr.TreeScore = env.TreeScore

	// 2. Soil score (20%)
	dr.SoilScore = env.SoilScore

	// 3. Seasonal factor (18%)
	dr.SeasonalScore = GetSeasonalFactor(now, lat)

	// 4. Time-of-day factor (15%)
	dr.TimeScore = GetTimeFactor(now.Hour())

	// 5. Weather factor (10%)
	weather := s.WeatherClient.GetCurrentWeather(env.H3CellLv9)
	dr.WeatherScore = GetWeatherFactor(weather)

	// 6. Terrain factor (5%)
	dr.TerrainScore = GetTerrainFactor(env.ElevationM, env.SlopeDeg)

	// 7. Urban factor (2%)
	dr.UrbanScore = 1.0
	if env.IsUrban {
		dr.UrbanScore = 0.5 // urban areas halved
	}

	// Weighted sum
	weightedSum :=
		dr.TreeScore     * weightTree +
		dr.SoilScore     * weightSoil +
		dr.SeasonalScore * weightSeasonal +
		dr.TimeScore     * weightTime +
		dr.WeatherScore  * weightWeather +
		dr.TerrainScore  * weightTerrain +
		dr.UrbanScore    * weightUrban

	dr.FinalDensity = MaxNymphDensity * weightedSum

	// Clamp to valid range
	dr.FinalDensity = math.Max(0, math.Min(dr.FinalDensity, MaxNymphDensity*1.2))

	return dr
}

// GetSeasonalFactor returns the seasonal activity factor based on month, hemisphere, and latitude.
// Extreme latitudes (polar) have a much shorter active season.
func GetSeasonalFactor(now time.Time, lat float64) float64 {
	month := now.Month()
	isNorthern := lat >= 0
	absLat := math.Abs(lat)

	// Map month to effective season index
	var effectiveMonth int
	if isNorthern {
		effectiveMonth = int(month)
	} else {
		// Southern hemisphere: shift by 6 months
		effectiveMonth = (int(month) + 6) % 12
		if effectiveMonth == 0 {
			effectiveMonth = 12
		}
	}

	// Base seasonal factor by month
	var baseFactor float64
	switch {
	case effectiveMonth >= 6 && effectiveMonth <= 8:
		baseFactor = 1.0 // peak season (June-August)
	case effectiveMonth == 5 || effectiveMonth == 9:
		baseFactor = 0.6 // transitional (May, September)
	case effectiveMonth == 4 || effectiveMonth == 10:
		baseFactor = 0.2 // off-season edge
	default:
		baseFactor = 0.05 // winter dormancy
	}

	// Latitude correction: at very high latitudes, the season is shorter
	// and ground doesn't warm enough even in summer
	if absLat > 70 {
		return baseFactor * 0.1 // Arctic — almost no cicada activity ever
	}
	if absLat > 60 {
		return baseFactor * 0.3 // Subarctic — very short active window
	}
	if absLat > 50 {
		return baseFactor * 0.7 // Cool temperate — reduced
	}

	return baseFactor
}

// GetTimeFactor returns the time-of-day activity factor.
// Cicada nymphs primarily emerge at night (20:00-05:00 peak).
func GetTimeFactor(hour int) float64 {
	switch {
	case hour >= 20 || hour <= 5:
		return 1.0 // night — peak emergence
	case hour >= 6 && hour <= 8:
		return 0.7 // early morning residual
	case hour >= 18 && hour <= 19:
		return 0.8 // dusk — starting to become active
	default:
		return 0.3 // daytime — deep underground
	}
}

// GetWeatherFactor calculates the weather impact on nymph availability.
// Temperature sweet spot: 25-35°C. Recent rain provides a bonus.
func GetWeatherFactor(w *models.WeatherContext) float64 {
	if w == nil {
		return 0.8 // default assumption
	}

	score := 1.0

	// Temperature curve (optimal 25-35°C)
	switch {
	case w.TemperatureC < 10:
		score *= 0.1
	case w.TemperatureC < 18:
		score *= 0.3
	case w.TemperatureC < 25:
		score *= 0.7
	case w.TemperatureC <= 35:
		score *= 1.0 // optimal range
	default:
		score *= 0.6 // too hot
	}

	// Post-rain bonus: soft soil makes emergence easier
	if w.RainLast24hMm > 5.0 {
		score *= 1.3 // moderate rain boost
	}

	// Active rain slightly reduces score (player experience)
	if w.IsRaining {
		score *= 0.9
	}

	return math.Max(0, math.Min(score, 1.5))
}

// GetTerrainFactor returns suitability based on elevation and slope.
// Cicadas prefer low elevations (<1000m) and gentle slopes (<15°).
func GetTerrainFactor(elevationM, slopeDeg float64) float64 {
	score := 1.0

	switch {
	case elevationM > 3000:
		score *= 0.0
	case elevationM > 2000:
		score *= 0.2
	case elevationM > 1000:
		score *= 0.5
	}

	switch {
	case slopeDeg > 30:
		score *= 0.1
	case slopeDeg > 15:
		score *= 0.5
	}

	return score
}

// WeatherClient is the interface for fetching current weather data.
type WeatherClient interface {
	GetCurrentWeather(cellID string) *models.WeatherContext
}

// TreeScorer computes tree suitability scores for locations.
type TreeScorer interface {
	GetTreeScore(lat, lng float64) float64
	GetDominantSpecies(lat, lng float64) string
}

// SoilScorer computes soil suitability scores for locations.
type SoilScorer interface {
	GetSoilScore(lat, lng float64) float64
	GetSoilType(lat, lng float64) string
}
