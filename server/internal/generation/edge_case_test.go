package generation

import (
	"math"
	"testing"
	"time"

	"github.com/cicada-hunt/server/internal/environment"
	"github.com/cicada-hunt/server/internal/models"
	"github.com/cicada-hunt/server/internal/spatial"
)

// ================================================================
// Environment Scorer Edge Cases
// ================================================================

func TestScorer_EdgeCases(t *testing.T) {
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: environment.NewMockWeatherClient(),
	}
	dc := NewDensityCalculator(scorer, nil)

	t.Run("zero_coordinates", func(t *testing.T) {
		// Equator + prime meridian (Gulf of Guinea)
		result, err := dc.GetCellDensity("test_zero", 0, 0, time.Now())
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.FinalDensity < 0 {
			t.Errorf("Density should not be negative: %.1f", result.FinalDensity)
		}
	})

	t.Run("extreme_north", func(t *testing.T) {
		// Svalbard (78°N) — should be much lower than max
		result, err := dc.GetCellDensity("test_north", 78.0, 15.0, time.Now())
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		// At 78°N: tree≈0.03, soil≈0.05 (permafrost), seasonal≈0.10
		// Expected density well below temperate regions
		if result.FinalDensity >= 80 {
			t.Errorf("Arctic should be well below temperate, got %.1f", result.FinalDensity)
		}
		t.Logf("Arctic density: %.1f nymphs/km² (tree=%.2f soil=%.2f seasonal=%.2f)",
			result.FinalDensity, result.TreeScore, result.SoilScore, result.SeasonalScore)
	})

	t.Run("deep_winter", func(t *testing.T) {
		// Beijing in January
		now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
		result, err := dc.GetCellDensity("test_winter", 39.9, 116.4, now)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.SeasonalScore > 0.1 {
			t.Errorf("Winter should have low seasonal score, got %.2f", result.SeasonalScore)
		}
	})

	t.Run("peak_summer_midnight", func(t *testing.T) {
		// Beijing in July at 2am — optimal conditions
		now := time.Date(2026, 7, 15, 2, 0, 0, 0, time.UTC)
		result, err := dc.GetCellDensity("test_peak", 39.9, 116.4, now)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.SeasonalScore < 0.9 {
			t.Errorf("July should be peak season, got %.2f", result.SeasonalScore)
		}
		if result.TimeScore < 0.9 {
			t.Errorf("2am should be peak time, got %.2f", result.TimeScore)
		}
	})

	t.Run("subzero_temperature", func(t *testing.T) {
		w := &models.WeatherContext{TemperatureC: -10, RainLast24hMm: 0, IsRaining: false}
		got := environment.GetWeatherFactor(w)
		if got > 0.15 {
			t.Errorf("-10°C should be near-zero, got %.2f", got)
		}
	})

	t.Run("scorching_temperature", func(t *testing.T) {
		w := &models.WeatherContext{TemperatureC: 45, RainLast24hMm: 0, IsRaining: false}
		got := environment.GetWeatherFactor(w)
		if got > 0.7 {
			t.Errorf("45°C should be reduced, got %.2f", got)
		}
	})

	t.Run("heavy_rain_bonus", func(t *testing.T) {
		w := &models.WeatherContext{TemperatureC: 28, RainLast24hMm: 20, IsRaining: false}
		got := environment.GetWeatherFactor(w)
		if got <= 1.0 {
			t.Errorf("Heavy rain should give bonus (>1.0), got %.2f", got)
		}
	})
}

// ================================================================
// Density Edge Cases
// ================================================================

func TestDensityCalculator_NilStore(t *testing.T) {
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: environment.NewMockWeatherClient(),
	}
	dc := NewDensityCalculator(scorer, nil) // nil store — should not panic

	result, err := dc.GetCellDensity("test_nil_store", 39.9, 116.4, time.Now())
	if err != nil {
		t.Fatalf("Nil store should not cause error: %v", err)
	}
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.FinalDensity < 0 {
		t.Errorf("Density should not be negative with nil store")
	}
	t.Logf("Density with nil store: %.1f nymphs/km²", result.FinalDensity)
}

func TestDensityCalculator_BatchWithErrors(t *testing.T) {
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: environment.NewMockWeatherClient(),
	}
	dc := NewDensityCalculator(scorer, nil)

	cells := []string{"valid_cell_1", "valid_cell_2"}
	results, err := dc.GetCellDensitiesBatch(cells, time.Now())
	if err != nil {
		t.Fatalf("Batch error: %v", err)
	}
	if len(results) == 0 {
		t.Error("Should have results for valid cells")
	}
}

// ================================================================
// Spawner Edge Cases
// ================================================================

func TestSpawner_NegativeCount(t *testing.T) {
	cfg := DefaultSpawnConfig()
	result, err := GenerateNymphs("test_neg", -5, nil, cfg, time.Now())
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("Negative count should produce 0, got %d", result.TotalCount)
	}
}

func TestSpawner_SingleNymph(t *testing.T) {
	cfg := DefaultSpawnConfig()
	result, err := GenerateNymphs("test_single", 1, nil, cfg, time.Now())
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("Expected exactly 1, got %d", result.TotalCount)
	}
	if len(result.Spawns) != 1 {
		t.Errorf("Expected 1 spawn, got %d", len(result.Spawns))
	}
	if result.Spawns[0].ID == "" {
		t.Error("Spawn should have an ID")
	}
}

func TestSpawner_SameCellTwice(t *testing.T) {
	cfg := DefaultSpawnConfig()
	now := time.Now()

	// Generate twice for the same cell
	r1, _ := GenerateNymphs("same_cell", 5, nil, cfg, now)
	r2, _ := GenerateNymphs("same_cell", 5, nil, cfg, now)

	// IDs should be unique across both generations
	ids := make(map[string]bool)
	for _, n := range r1.Spawns {
		ids[n.ID] = true
	}
	for _, n := range r2.Spawns {
		if ids[n.ID] {
			t.Errorf("Duplicate ID across generations: %s", n.ID)
		}
		ids[n.ID] = true
	}
}

// ================================================================
// Spatial Edge Cases
// ================================================================

func TestSpatial_InvalidCells(t *testing.T) {
	if spatial.IsValidH3Cell("") {
		t.Error("Empty string should be invalid")
	}
	if spatial.IsValidH3Cell("not_a_hex_string") {
		t.Error("Invalid hex should be rejected")
	}
	if spatial.IsValidH3Cell("zzzzzzzzzzzzzzz") {
		t.Error("Non-hex chars should be rejected")
	}
}

func TestSpatial_DistanceZero(t *testing.T) {
	dist := spatial.HaversineDistance(39.9, 116.4, 39.9, 116.4)
	if dist != 0 {
		t.Errorf("Same point should be 0m, got %.6f", dist)
	}
}

func TestSpatial_AntipodalDistance(t *testing.T) {
	// Beijing vs approximate antipode (near Chile/Argentina)
	dist := spatial.HaversineDistance(39.9, 116.4, -39.9, -63.6)
	// Should be roughly half the Earth's circumference (~20,000 km)
	if dist < 19000000 || dist > 21000000 {
		t.Errorf("Antipodal distance should be ~20,000 km, got %.0f km", dist/1000)
	}
}

func TestSpatial_LargeRadius(t *testing.T) {
	// Large radius should return many cells but not error
	cells, err := spatial.CellsInRadius(39.9, 116.4, 5000, spatial.GridLevelCoarse)
	if err != nil {
		t.Fatalf("Large radius error: %v", err)
	}
	if len(cells) == 0 {
		t.Error("5km radius should return some cells")
	}
	t.Logf("5000m at Lv7: %d cells", len(cells))
}

// ================================================================
// Cicada AI Edge Cases
// ================================================================

func TestCicadaAI_NoPlayer(t *testing.T) {
	ai := NewCicadaAI()
	ctx := &CicadaAIContext{
		Cicada: &models.CicadaSpawn{
			ID:           "solo_cicada",
			Lat:          39.9,
			Lng:          116.4,
			CurrentState: models.CicadaStateResting,
			AlertDistM:   4.0,
			Agility:      0.3,
		},
		NearestPlayer: nil, // no player
		Now:           time.Now(),
	}
	// Should not panic — could return nil (no state change) or spontaneous fly
	event := ai.Update(ctx)
	// No player → should either do nothing or spontaneous fly
	if event != nil && event.NewState != models.CicadaStateFlying {
		t.Errorf("Without player, only expected Flying (spontaneous), got %s", event.NewState)
	}
}

func TestCicadaAI_FlightPathEndpoints(t *testing.T) {
	path := GenerateFlightPath(39.9, 116.4, 5.0, 39.905, 116.41, 3.0, models.FlightTypeCasual)

	// Evaluate many points along the path
	for _, tParam := range []float64{0, 0.1, 0.25, 0.5, 0.75, 0.9, 1.0} {
		lat, lng, alt := path.EvaluatePosition(tParam)
		if math.IsNaN(lat) || math.IsNaN(lng) || math.IsNaN(alt) {
			t.Errorf("NaN at t=%.2f: lat=%.4f lng=%.4f alt=%.1f", tParam, lat, lng, alt)
		}
		if alt < 0 {
			t.Errorf("Altitude should not go below 0: %.1f at t=%.2f", alt, tParam)
		}
	}
}

func TestCicadaAI_AllFlightTypes(t *testing.T) {
	tests := []models.FlightType{
		models.FlightTypeCasual,
		models.FlightTypeEvasive,
		models.FlightTypePanic,
	}

	for _, ft := range tests {
		t.Run(string(ft), func(t *testing.T) {
			path := GenerateFlightPath(39.9, 116.4, 5.0, 39.91, 116.42, 4.0, ft)
			if path.Duration <= 0 {
				t.Errorf("Duration should be positive for %s", ft)
			}
			if path.FlightType != ft {
				t.Errorf("FlightType mismatch: %s vs %s", path.FlightType, ft)
			}
		})
	}
}

// ================================================================
// Recovery Rate Edge Cases
// ================================================================

func TestRecoveryRate_EdgeCases(t *testing.T) {
	tests := []struct {
		curr, base float64
		want       float64
	}{
		{0, 200, 30},           // completely depleted → 15% recovery
		{100, 200, 115},        // half → recover 15% of gap (15)
		{199, 200, 199.15},     // almost full → tiny recovery
		{200, 200, 200},        // at capacity → no change
		{0, 0, 0},              // zero base → zero
		{250, 200, 200},        // over-full → clamp to base
	}

	for _, tt := range tests {
		got := CalculateRecoveredDensity(tt.curr, tt.base)
		if math.Abs(got-tt.want) > 0.15 {
			t.Errorf("Recovery(%.0f, %.0f) = %.2f, want %.2f", tt.curr, tt.base, got, tt.want)
		}
	}
}

// ================================================================
// Cicada Spawn Edge Cases
// ================================================================

func TestCicadaSpawn_ZeroCount(t *testing.T) {
	cfg := DefaultCicadaSpawnConfig()
	result, err := GenerateCicadas(39.9, 116.4, 0, nil, cfg, time.Now())
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("Zero count should produce 0, got %d", result.TotalCount)
	}
}

func TestCicadaSpawn_ManyTrees(t *testing.T) {
	cfg := DefaultCicadaSpawnConfig()
	// Create many trees with varying preferences
	trees := make([]TreeLocation, 20)
	for i := range trees {
		trees[i] = TreeLocation{
			Lat:           39.9 + float64(i)*0.0001,
			Lng:           116.4 + float64(i)*0.0001,
			Species:       "poplar",
			Preference:    0.3 + float64(i)*0.03,
			CanopyRadiusM: 5.0,
		}
	}

	result, err := GenerateCicadas(39.9, 116.4, 15, trees, cfg, time.Now())
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.TotalCount == 0 {
		t.Error("Should generate cicadas with trees")
	}
	t.Logf("Generated %d cicadas across %d trees", result.TotalCount, len(trees))
}
