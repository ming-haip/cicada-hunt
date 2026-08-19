package generation

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cicada-hunt/server/internal/environment"
	"github.com/cicada-hunt/server/internal/models"
)

// TestConcurrentCellGeneration verifies that multiple goroutines can generate
// different cells simultaneously without data races or deadlocks.
func TestConcurrentCellGeneration(t *testing.T) {
	weatherClient := environment.NewMockWeatherClient()
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: weatherClient,
	}
	dc := NewDensityCalculator(scorer, nil)

	cells := []struct{ lat, lng float64 }{
		{39.90, 116.40},
		{39.91, 116.41},
		{39.92, 116.42},
		{39.93, 116.43},
		{39.94, 116.44},
		{39.95, 116.45},
		{39.96, 116.46},
		{39.97, 116.47},
		{39.98, 116.48},
		{39.99, 116.49},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(cells))

	now := time.Now()
	for i, c := range cells {
		wg.Add(1)
		go func(idx int, lat, lng float64) {
			defer wg.Done()
			cellID := fmt.Sprintf("test_cell_conc_%d", idx)
			result, err := dc.GetCellDensity(cellID, lat, lng, now)
			if err != nil {
				errCh <- fmt.Errorf("cell %d: %w", idx, err)
				return
			}
			if result.FinalDensity < 0 || result.FinalDensity > 250 {
				errCh <- fmt.Errorf("cell %d: density out of range: %.1f", idx, result.FinalDensity)
			}
		}(i, c.lat, c.lng)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}

// TestConcurrentSpawnGeneration tests parallel nymph generation.
func TestConcurrentSpawnGeneration(t *testing.T) {
	cfg := DefaultSpawnConfig()
	now := time.Now()

	var wg sync.WaitGroup
	errCh := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cellID := fmt.Sprintf("cell_conc_spawn_%d", idx)
			result, err := GenerateNymphs(cellID, 10, nil, cfg, now)
			if err != nil {
				errCh <- fmt.Errorf("cell %s: %w", cellID, err)
				return
			}
			if result.TotalCount != 10 {
				errCh <- fmt.Errorf("cell %s: expected 10, got %d", cellID, result.TotalCount)
			}
			ids := make(map[string]bool)
			for _, n := range result.Spawns {
				if ids[n.ID] {
					errCh <- fmt.Errorf("cell %s: duplicate nymph ID %s", cellID, n.ID)
				}
				ids[n.ID] = true
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}

// TestConcurrentCicadaAI tests concurrent AI evaluation safety.
func TestConcurrentCicadaAI(t *testing.T) {
	now := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ai := NewCicadaAI()
			ctx := &CicadaAIContext{
				Cicada: &models.CicadaSpawn{
					ID:           fmt.Sprintf("conc_cicada_%d", idx),
					Lat:          39.9042,
					Lng:          116.4074,
					CurrentState: models.CicadaStateResting,
					AlertDistM:   4.0,
					FleeDistM:    15.0,
					Agility:       0.3,
				},
				NearestPlayer: &PlayerInfo{
					PlayerID: "p1",
					Lat:      39.9042 + float64(idx)*0.0001,
					Lng:      116.4074,
					SpeedMS:  1.0,
				},
				Now: now,
			}
			_ = ai.Update(ctx)
		}(i)
	}
	wg.Wait()
}
