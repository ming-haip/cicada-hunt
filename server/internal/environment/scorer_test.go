package environment

import (
	"math"
	"testing"
	"time"

	"github.com/cicada-hunt/server/internal/models"
)

func TestGetSeasonalFactor(t *testing.T) {
	// Northern hemisphere: peak June-August
	tests := []struct {
		month    time.Month
		lat      float64
		expected float64
	}{
		{6, 39.9, 1.0},  // Beijing, June → peak
		{7, 39.9, 1.0},  // July → peak
		{8, 39.9, 1.0},  // August → peak
		{5, 39.9, 0.6},  // May → transitional
		{9, 39.9, 0.6},  // September → transitional
		{4, 39.9, 0.2},  // April → off-season
		{10, 39.9, 0.2}, // October → off-season
		{1, 39.9, 0.05}, // January → winter
		{12, 39.9, 0.05}, // December → winter
	}

	for _, tt := range tests {
		now := time.Date(2026, tt.month, 15, 12, 0, 0, 0, time.UTC)
		got := GetSeasonalFactor(now, tt.lat)
		if math.Abs(got-tt.expected) > 0.01 {
			t.Errorf("SeasonalFactor(month=%d, lat=%.1f) = %.2f, want %.2f",
				tt.month, tt.lat, got, tt.expected)
		}
	}
}

func TestGetSeasonalFactor_SouthernHemisphere(t *testing.T) {
	// Southern hemisphere: peak December-February
	now := time.Date(2026, 12, 15, 12, 0, 0, 0, time.UTC)
	got := GetSeasonalFactor(now, -33.9) // Sydney
	if math.Abs(got-1.0) > 0.01 {
		t.Errorf("Southern December should be peak, got %.2f", got)
	}

	now = time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	got = GetSeasonalFactor(now, -33.9)
	if math.Abs(got-0.05) > 0.01 {
		t.Errorf("Southern July should be winter (0.05), got %.2f", got)
	}
}

func TestGetTimeFactor(t *testing.T) {
	tests := []struct {
		hour     int
		expected float64
	}{
		{22, 1.0}, // night peak
		{2, 1.0},  // night peak
		{5, 1.0},  // edge of night
		{6, 0.7},  // early morning
		{7, 0.7},
		{12, 0.3}, // midday
		{15, 0.3}, // afternoon
		{18, 0.8}, // dusk
		{19, 0.8},
		{20, 1.0}, // night starts
	}

	for _, tt := range tests {
		got := GetTimeFactor(tt.hour)
		if math.Abs(got-tt.expected) > 0.01 {
			t.Errorf("TimeFactor(hour=%d) = %.2f, want %.2f", tt.hour, got, tt.expected)
		}
	}
}

func TestGetWeatherFactor(t *testing.T) {
	// Optimal conditions
	w := &models.WeatherContext{
		TemperatureC:  28,
		RainLast24hMm: 10, // recent rain
		IsRaining:     false,
	}
	got := GetWeatherFactor(w)
	if got < 1.0 {
		t.Errorf("Optimal weather should give >= 1.0, got %.2f", got)
	}

	// Too cold
	w2 := &models.WeatherContext{
		TemperatureC:  5,
		RainLast24hMm: 0,
		IsRaining:     false,
	}
	got2 := GetWeatherFactor(w2)
	if got2 > 0.15 {
		t.Errorf("5°C should give very low score, got %.2f", got2)
	}

	// Too hot
	w3 := &models.WeatherContext{
		TemperatureC:  40,
		RainLast24hMm: 0,
		IsRaining:     false,
	}
	got3 := GetWeatherFactor(w3)
	if got3 > 0.7 {
		t.Errorf("40°C should give reduced score, got %.2f", got3)
	}
}

func TestGetTerrainFactor(t *testing.T) {
	// Low elevation, gentle slope = ideal
	got := GetTerrainFactor(200, 5)
	if got != 1.0 {
		t.Errorf("Low elevation + gentle slope should be 1.0, got %.2f", got)
	}

	// High elevation
	got = GetTerrainFactor(3500, 5)
	if got != 0.0 {
		t.Errorf("Very high elevation should be 0, got %.2f", got)
	}

	// Steep slope
	got = GetTerrainFactor(200, 35)
	if math.Abs(got-0.1) > 0.01 {
		t.Errorf("Steep slope should be 0.1, got %.2f", got)
	}
}

func TestScorer_CalculateDensity(t *testing.T) {
	weatherClient := NewMockWeatherClient()
	scorer := &Scorer{
		TreeScorer:    NewDefaultTreeScorer(),
		SoilScorer:    NewDefaultSoilScorer(),
		WeatherClient: weatherClient,
	}

	env := &models.EnvironmentFactors{
		H3CellLv9:     "8928308280fffff",
		TreeScore:     0.8,
		SoilScore:     0.7,
		ElevationM:    100,
		SlopeDeg:      3,
		ImperviousPct: 10,
		IsUrban:       false,
	}

	now := time.Date(2026, 7, 15, 22, 0, 0, 0, time.UTC) // July night
	result := scorer.CalculateDensity("test_cell", env, now, 39.9)

	// With tree=0.8, soil=0.7, peak season, night, good weather → expect high density
	if result.FinalDensity < 50 {
		t.Errorf("July night should have high density, got %.1f", result.FinalDensity)
	}
	if result.FinalDensity > 200 {
		t.Errorf("Density should not exceed max, got %.1f", result.FinalDensity)
	}

	// Check breakdown adds up
	weightedSum :=
		result.TreeScore*weightTree +
			result.SoilScore*weightSoil +
			result.SeasonalScore*weightSeasonal +
			result.TimeScore*weightTime +
			result.WeatherScore*weightWeather +
			result.TerrainScore*weightTerrain +
			result.UrbanScore*weightUrban

	expectedDensity := MaxNymphDensity * weightedSum
	if math.Abs(result.FinalDensity-expectedDensity) > 0.1 {
		t.Errorf("Density mismatch: got %.2f, expected %.2f from breakdown",
			result.FinalDensity, expectedDensity)
	}
}
