package models

import "time"

// EnvironmentFactors aggregates all environmental data for a single H3 cell.
type EnvironmentFactors struct {
	H3CellLv9 string  `json:"h3_cell_lv9"`
	TreeScore float64 `json:"tree_score"` // 0-1 tree suitability
	SoilScore float64 `json:"soil_score"` // 0-1 soil suitability
	SoilType  string  `json:"soil_type"`

	ElevationM     float64 `json:"elevation_m"`
	SlopeDeg       float64 `json:"slope_deg"`
	ImperviousPct  float64 `json:"impervious_pct"`  // % impervious surface
	NDVI           float64 `json:"ndvi"`            // -1 to 1
	TreeDensityIdx float64 `json:"tree_density_idx"` // 0-1

	IsUrban       bool    `json:"is_urban"`
	WaterProxM    float64 `json:"water_proximity_m"`
	DominantTrees string  `json:"dominant_trees"` // most common tree species

	UpdatedAt time.Time `json:"updated_at"`
}

// SeasonalInfo captures the current seasonal context.
type SeasonalInfo struct {
	Month           int     `json:"month"`
	Season          string  `json:"season"` // spring, summer, autumn, winter
	IsNorthernHem   bool    `json:"is_northern_hem"`
	SeasonalFactor  float64 `json:"seasonal_factor"`  // 0-1
	IsPeakSeason    bool    `json:"is_peak_season"`   // June-August
	IsTransitional  bool    `json:"is_transitional"`  // May, September
}

// TimeContext captures time-of-day information.
type TimeContext struct {
	Hour       int     `json:"hour"`
	IsNight    bool    `json:"is_night"`     // 20:00-05:00 peak emergence
	IsTwilight bool    `json:"is_twilight"`  // 18:00-20:00 or 05:00-06:00
	IsDaytime  bool    `json:"is_daytime"`
	TimeFactor float64 `json:"time_factor"`  // 0-1
}

// WeatherContext holds current weather data from external API.
type WeatherContext struct {
	TemperatureC  float64 `json:"temperature_c"`
	HumidityPct   float64 `json:"humidity_pct"`
	RainLast24hMm float64 `json:"rain_last_24h_mm"`
	IsRaining     bool    `json:"is_raining"`
	WindSpeedMS   float64 `json:"wind_speed_ms"`
	WeatherFactor float64 `json:"weather_factor"` // 0-1.5
}

// TreePreferenceScore maps tree species names to cicada preference scores (0-1).
var TreePreferenceScore = map[string]float64{
	"poplar":        1.00,
	"willow":        0.95,
	"elm":           0.85,
	"locust":        0.80,
	"apple":         0.80,
	"pear":          0.78,
	"peach":         0.75,
	"cherry":        0.70,
	"oak":           0.60,
	"maple":         0.50,
	"pine":          0.15,
	"cypress":       0.10,
	"bamboo":        0.05,
	"default_tree":  0.40,
}

// SoilPreferenceScore maps soil types to cicada nymph preference scores (0-1).
var SoilPreferenceScore = map[string]float64{
	"loam":       1.00,
	"sandy_loam": 0.85,
	"silt_loam":  0.80,
	"clay_loam":  0.65,
	"sandy":      0.40,
	"clay":       0.30,
	"gravel":     0.10,
	"bedrock":    0.00,
	"water":      0.00,
	"concrete":   0.00,
}

// SeasonNames maps hemisphere + month to season name.
func SeasonName(month int, isNorthern bool) string {
	if isNorthern {
		switch {
		case month >= 3 && month <= 5:
			return "spring"
		case month >= 6 && month <= 8:
			return "summer"
		case month >= 9 && month <= 11:
			return "autumn"
		default:
			return "winter"
		}
	}
	// Southern hemisphere: seasons inverted
	switch {
	case month >= 3 && month <= 5:
		return "autumn"
	case month >= 6 && month <= 8:
		return "winter"
	case month >= 9 && month <= 11:
		return "spring"
	default:
		return "summer"
	}
}
