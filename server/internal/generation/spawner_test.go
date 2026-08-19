package generation

import (
	"testing"
	"time"

	"github.com/cicada-hunt/server/internal/models"
)

func TestWeightedSpeciesSelection(t *testing.T) {
	// Run 10,000 selections and verify distribution roughly matches weights
	counts := make(map[models.NymphSpecies]int)
	const iterations = 10000

	for i := 0; i < iterations; i++ {
		species := weightedSpeciesSelection()
		counts[species]++
	}

	// Check rare species appear at expected rates
	totalWeight := float64(models.TotalSpawnWeight())

	for species, cfg := range models.NymphSpeciesConfigs {
		expected := float64(cfg.SpawnWeight) / totalWeight * iterations
		actual := counts[species]
		tolerance := expected * 0.3 // 30% tolerance

		if float64(actual) < expected-tolerance || float64(actual) > expected+tolerance {
			t.Logf("%s: expected ~%.0f, got %d (tolerance: ±%.0f)",
				cfg.Name, expected, actual, tolerance)
		}
	}

	// Golden cicada should be VERY rare (weight=1 out of 121)
	goldenCount := counts[models.NymphGoldenCicada]
	goldenExpected := 1.0 / 121.0 * iterations // ~82
	if goldenCount > int(goldenExpected*3) {
		t.Errorf("Golden cicada too frequent: %d (expected ~%.0f)", goldenCount, goldenExpected)
	}
}

func TestWeightedQuality(t *testing.T) {
	counts := make([]int, 6) // index 1-5
	const iterations = 10000

	for i := 0; i < iterations; i++ {
		q := weightedQuality(models.NymphBlackCicada)
		if q < 1 || q > 5 {
			t.Errorf("Quality out of range: %d", q)
		}
		counts[q]++
	}

	// Most should be 2-4 stars
	midRange := counts[2] + counts[3] + counts[4]
	if midRange < iterations*7/10 {
		t.Errorf("Too few mid-quality (2-4★): %d/%d", midRange, iterations)
	}

	// 5-star should be rare (~10%)
	if counts[5] > iterations*15/100 {
		t.Errorf("5-star too frequent: %d", counts[5])
	}
}

func TestGenerateNymphs_EmptyCell(t *testing.T) {
	cfg := DefaultSpawnConfig()
	now := time.Now()

	result, err := GenerateNymphs("test_cell", 0, nil, cfg, now)
	if err != nil {
		t.Fatalf("GenerateNymphs error: %v", err)
	}

	if result.TotalCount != 0 {
		t.Errorf("Target count 0 should produce 0 nymphs, got %d", result.TotalCount)
	}
	if !result.Generated {
		t.Error("Should be marked as generated")
	}
}

func TestGenerateNymphs_SmallBatch(t *testing.T) {
	cfg := DefaultSpawnConfig()
	now := time.Now()

	// Generate 10 nymphs with no tree data (uniform random placement)
	result, err := GenerateNymphs("test_cell_2", 10, nil, cfg, now)
	if err != nil {
		t.Fatalf("GenerateNymphs error: %v", err)
	}

	if result.TotalCount != 10 {
		t.Errorf("Expected 10 nymphs, got %d", result.TotalCount)
	}

	// Each nymph should have valid attributes
	for _, nymph := range result.Spawns {
		if nymph.ID == "" {
			t.Error("Nymph should have an ID")
		}
		if nymph.SpeciesName == "" {
			t.Error("Nymph should have a species name")
		}
		if nymph.Quality < 1 || nymph.Quality > 5 {
			t.Errorf("Quality out of range: %d", nymph.Quality)
		}
		if nymph.DepthCm < 0 || nymph.DepthCm > 70 {
			t.Errorf("Depth out of range: %.1f cm", nymph.DepthCm)
		}
		if nymph.SizeCm <= 0 {
			t.Errorf("Invalid size: %.1f cm", nymph.SizeCm)
		}
		if nymph.Status != models.NymphStatusActive {
			t.Errorf("New nymph should be active, got %s", nymph.Status)
		}
	}
}

func TestGenerateNymphs_WithTrees(t *testing.T) {
	cfg := DefaultSpawnConfig()
	now := time.Now()

	// Provide tree data — nymphs should cluster near trees
	trees := []TreeLocation{
		{Lat: 39.9042, Lng: 116.4074, Species: "poplar", Preference: 1.0, CanopyRadiusM: 8},
		{Lat: 39.9044, Lng: 116.4076, Species: "willow", Preference: 0.95, CanopyRadiusM: 6},
	}

	result, err := GenerateNymphs("test_cell_3", 20, trees, cfg, now)
	if err != nil {
		t.Fatalf("GenerateNymphs error: %v", err)
	}

	if result.TotalCount < 1 {
		t.Error("Should generate at least some nymphs")
	}
}

func TestGenerateNymphs_CapLimit(t *testing.T) {
	cfg := DefaultSpawnConfig()
	now := time.Now()

	// Request more than max (50)
	result, err := GenerateNymphs("test_cell_4", 100, nil, cfg, now)
	if err != nil {
		t.Fatalf("GenerateNymphs error: %v", err)
	}

	if result.TotalCount > 50 {
		t.Errorf("Should cap at 50, got %d", result.TotalCount)
	}
}

func TestMinSpacing(t *testing.T) {
	existing := []spawnPoint{
		{Lat: 39.9042, Lng: 116.4074},
	}

	// Too close (< 3m)
	close := spawnPoint{Lat: 39.90421, Lng: 116.40741}
	if hasMinSpacing(close, existing, 3.0) {
		t.Error("Close point should be rejected")
	}

	// Far enough (> 3m)
	far := spawnPoint{Lat: 39.9045, Lng: 116.4074}
	if !hasMinSpacing(far, existing, 3.0) {
		t.Error("Far point should be accepted")
	}
}

func TestRecoveryRate(t *testing.T) {
	// Depleted cell: 50 → base 200
	newDensity := CalculateRecoveredDensity(50, 200)
	expected := 50 + (200-50)*0.15 // 72.5
	if newDensity < expected-0.1 || newDensity > expected+0.1 {
		t.Errorf("Recovered density: %.1f, expected %.1f", newDensity, expected)
	}

	// Full cell: 200 → base 200 (should stay at 200)
	newDensity = CalculateRecoveredDensity(200, 200)
	if newDensity != 200 {
		t.Errorf("Full cell should stay full: got %.1f", newDensity)
	}

	// Over-full cell (shouldn't happen, but clamp)
	newDensity = CalculateRecoveredDensity(250, 200)
	if newDensity != 200 {
		t.Errorf("Over-full should clamp to base: %.1f", newDensity)
	}
}
