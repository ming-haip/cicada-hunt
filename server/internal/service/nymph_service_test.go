package service

import (
	"context"
	"testing"
	"time"

	"github.com/cicada-hunt/server/internal/environment"
	"github.com/cicada-hunt/server/internal/generation"
	"github.com/cicada-hunt/server/internal/models"
	"github.com/cicada-hunt/server/internal/spatial"
)

// TestNymphService_QueryNearby_DBLessMode verifies the full query path works
// without database or Redis (pure in-memory generation).
func TestNymphService_QueryNearby_DBLessMode(t *testing.T) {
	// Setup (mimics main.go when DB/Redis unavailable)
	weatherClient := environment.NewMockWeatherClient()
	treeScorer := environment.NewDefaultTreeScorer()
	soilScorer := environment.NewDefaultSoilScorer()

	scorer := &environment.Scorer{
		TreeScorer:    treeScorer,
		SoilScorer:    soilScorer,
		WeatherClient: weatherClient,
	}

	densityCalc := generation.NewDensityCalculator(scorer, nil)

	// NymphService with nil store and nil cache (DB-less mode)
	svc := NewNymphService(densityCalc, nil, nil)

	ctx := context.Background()
	lat, lng := 39.9042, 116.4074

	// Query nearby nymphs
	resp, err := svc.QueryNearbyNymphs(ctx, lat, lng, 100, 10)
	if err != nil {
		t.Errorf("QueryNearbyNymphs returned error: %v", err)
		return
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	t.Logf("Found %d nymphs in total", resp.TotalInArea)

	// With no trees data, not all cells will get nymphs
	// But the query should complete gracefully
}

func TestNymphService_DigNymph_NoStore(t *testing.T) {
	weatherClient := environment.NewMockWeatherClient()
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: weatherClient,
	}
	densityCalc := generation.NewDensityCalculator(scorer, nil)
	svc := NewNymphService(densityCalc, nil, nil)

	ctx := context.Background()
	resp, err := svc.DigNymph(ctx, "player1", "nymph1", 39.9, 116.4, 1.5, 5.0, 10.0, "small_shovel")

	if err != nil {
		t.Errorf("DigNymph error: %v", err)
	}

	if resp.Success {
		t.Error("Dig should fail without store (nymph not found)")
	}

	t.Logf("Dig result: success=%v reason=%s", resp.Success, resp.FailReason)
}

func TestNymphService_GenerationFlow(t *testing.T) {
	weatherClient := environment.NewMockWeatherClient()
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: weatherClient,
	}
	densityCalc := generation.NewDensityCalculator(scorer, nil)
	_ = NewNymphService(densityCalc, nil, nil) // verify construction works

	// Get H3 cells near Beijing
	cells, err := spatial.CellsInRadius(39.9042, 116.4074, 200, spatial.GridLevelDefault)
	if err != nil {
		t.Fatalf("CellsInRadius error: %v", err)
	}
	if len(cells) == 0 {
		t.Fatal("Should find at least 1 cell")
	}
	t.Logf("Found %d cells in 200m radius", len(cells))

	now := time.Now()
	maxCells := len(cells)
	if maxCells > 3 {
		maxCells = 3
	}
	for _, cellID := range cells[:maxCells] { // Test up to 3 cells
		lat, lng := spatial.CellToLatLng(cellID)
		result, err := densityCalc.GetCellDensity(cellID, lat, lng, now)
		if err != nil {
			t.Errorf("GetCellDensity(%s) error: %v", cellID, err)
			continue
		}
		t.Logf("Cell %s: density=%.1f nymphs/km² (tree=%.2f soil=%.2f seasonal=%.2f)",
			cellID[:12], result.FinalDensity,
			result.TreeScore, result.SoilScore, result.SeasonalScore)

		if result.FinalDensity < 0 || result.FinalDensity > 250 {
			t.Errorf("Density out of range: %.1f", result.FinalDensity)
		}
	}
}

func TestDigSuccessRateCalculation(t *testing.T) {
	tool := models.ToolStats{Accuracy: 0.80}

	// Perfect aim → ~80% success
	rate := calculateDigSuccessRate(0.0, tool)
	if rate != 0.80 {
		t.Errorf("Perfect aim: expected 0.80, got %.2f", rate)
	}

	// 10cm off → reduced
	rate10 := calculateDigSuccessRate(10.0, tool)
	if rate10 >= 0.80 {
		t.Errorf("10cm deviation should reduce rate: got %.2f", rate10)
	}

	// 30cm off → very low
	rate30 := calculateDigSuccessRate(30.0, tool)
	if rate30 > 0.10 {
		t.Errorf("30cm deviation should be near 0: got %.2f", rate30)
	}
}
