package generation

import (
	"math"
	"testing"
	"time"

	"github.com/cicada-hunt/server/internal/models"
)

func TestCicadaAI_RestingToAlert(t *testing.T) {
	ai := NewCicadaAI()
	now := time.Now()

	cicada := &models.CicadaSpawn{
		ID:            "test_cicada_1",
		Lat:           39.9042,
		Lng:           116.4074,
		CurrentState:  models.CicadaStateResting,
		AlertDistM:    8.0, // large alert range for reliable test
		FleeDistM:     15.0,
		Agility:       0.3,
	}

	// Player at ~5m (within 8m alert, but outside 8*0.3=2.4m flee range)
	ctx := &CicadaAIContext{
		Cicada: cicada,
		NearestPlayer: &PlayerInfo{
			PlayerID: "player1",
			Lat:      39.90425, // ~5.5m away, within alert
			Lng:      116.4074,
			SpeedMS:  1.0,
		},
		Now: now,
	}

	event := ai.Update(ctx)
	if event == nil {
		t.Fatal("Expected a state change event, got nil")
	}
	if event.NewState != models.CicadaStateAlert {
		t.Errorf("Expected ALERT state, got %s", event.NewState)
	}
}

func TestCicadaAI_AlertToFlee(t *testing.T) {
	ai := NewCicadaAI()
	now := time.Now()

	cicada := &models.CicadaSpawn{
		ID:            "test_cicada_2",
		Lat:           39.9042,
		Lng:           116.4074,
		CurrentState:  models.CicadaStateAlert,
		AlertDistM:    4.0,
		FleeDistM:     15.0,
		Agility:       0.6,
	}

	// Player very close (within 30% alert distance) → should FLEE
	ctx := &CicadaAIContext{
		Cicada: cicada,
		NearestPlayer: &PlayerInfo{
			PlayerID: "player1",
			Lat:      39.90421, // ~1m away (within 4*0.3=1.2m)
			Lng:      116.4074,
			SpeedMS:  1.0,
		},
		Now: now,
	}

	event := ai.Update(ctx)
	if event == nil {
		t.Fatal("Expected a state change event, got nil")
	}
	if event.NewState != models.CicadaStateFlying {
		t.Errorf("Expected FLYING state, got %s", event.NewState)
	}
}

func TestCicadaAI_NetSwingStartles(t *testing.T) {
	ai := NewCicadaAI()
	now := time.Now()

	cicada := &models.CicadaSpawn{
		ID:            "test_cicada_3",
		Lat:           39.9042,
		Lng:           116.4074,
		CurrentState:  models.CicadaStateResting,
		AlertDistM:    4.0,
		Agility:       0.3,
	}

	// Player swings net nearby → STARTLED
	ctx := &CicadaAIContext{
		Cicada: cicada,
		NearestPlayer: &PlayerInfo{
			PlayerID:   "player1",
			Lat:        39.90421,
			Lng:        116.4074,
			IsSwinging: true,
		},
		Now: now,
	}

	event := ai.Update(ctx)
	if event == nil {
		t.Fatal("Expected a state change event, got nil")
	}
	// Should either transition through Startled to Flying, or directly
	if event.NewState == models.CicadaStateStartled {
		t.Log("Transitioned to STARTLED (expected)")
	} else if event.NewState == models.CicadaStateFlying || event.NewState == models.CicadaStateAlert {
		t.Logf("Transitioned to %s (also valid from resting with net swing)", event.NewState)
	} else {
		t.Errorf("Unexpected state: %s", event.NewState)
	}
}

func TestCicadaAI_StartledCooldown(t *testing.T) {
	ai := NewCicadaAI()
	now := time.Now()

	cicada := &models.CicadaSpawn{
		ID:            "test_cicada_4",
		Lat:           39.9042,
		Lng:           116.4074,
		CurrentState:  models.CicadaStateResting,
		AlertDistM:    4.0,
		Agility:       0.5,
	}

	// Mark as startled (within cooldown)
	ai.startledUntil[cicada.ID] = now.Add(30 * time.Second)

	ctx := &CicadaAIContext{
		Cicada: cicada,
		NearestPlayer: &PlayerInfo{
			PlayerID: "player1",
			Lat:      39.90421,
			Lng:      116.4074,
		},
		Now: now,
	}

	event := ai.Update(ctx)
	if event != nil {
		t.Error("No event expected during startled cooldown")
	}

	// After cooldown expires
	ctx.Now = now.Add(31 * time.Second)
	event = ai.Update(ctx)
	if event == nil {
		t.Error("Expected event after cooldown expires")
	}
}

func TestCicadaFlightPath_Casual(t *testing.T) {
	path := GenerateFlightPath(
		39.9042, 116.4074, 5.0,
		39.9045, 116.4080, 4.0,
		models.FlightTypeCasual,
	)

	// Evaluate at t=0, 0.5, 1.0
	lat0, lng0, alt0 := path.EvaluatePosition(0)
	if math.Abs(lat0-39.9042) > 0.0001 || math.Abs(lng0-116.4074) > 0.0001 {
		t.Errorf("Position at t=0 should be start: %.4f, %.4f", lat0, lng0)
	}

	lat1, lng1, _ := path.EvaluatePosition(1)
	if math.Abs(lat1-39.9045) > 0.0001 || math.Abs(lng1-116.4080) > 0.0001 {
		t.Errorf("Position at t=1 should be end: %.4f, %.4f (got %.4f, %.4f)", 39.9045, 116.4080, lat1, lng1)
	}

	// Mid-point should be above start (casual flight arcs up)
	_, _, altMid := path.EvaluatePosition(0.5)
	if altMid <= alt0 {
		t.Errorf("Mid-flight altitude (%.1f) should be above start (%.1f)", altMid, alt0)
	}

	// Duration should be in 3-8s range for casual
	if path.Duration < 3 || path.Duration > 8 {
		t.Errorf("Casual duration %.1f outside 3-8s range", path.Duration)
	}
}

func TestCicadaFlightPath_Panic(t *testing.T) {
	path := GenerateFlightPath(
		39.9042, 116.4074, 5.0,
		39.9050, 116.4090, 6.0,
		models.FlightTypePanic,
	)

	// Panic should be fast (1-2s)
	if path.Duration < 0.8 || path.Duration > 2.5 {
		t.Errorf("Panic duration %.1f outside 1-2s range", path.Duration)
	}

	// First control point should shoot upward (panic response)
	if path.Control1[2] < 5 {
		t.Errorf("Panic should shoot upward rapidly, ctrl1.z=%.1f", path.Control1[2])
	}
}

func TestDetectionRisk(t *testing.T) {
	// Very close + fast = high risk
	risk := calculateDetectionRisk(1.0, 5.0, 0.5, 4.0)
	if risk < 0.7 {
		t.Errorf("Close+fast should be high risk, got %.2f", risk)
	}

	// Far + slow = low risk
	risk = calculateDetectionRisk(10.0, 0.5, 0.5, 4.0)
	if risk > 0.3 {
		t.Errorf("Far+slow should be low risk, got %.2f", risk)
	}

	// Agile cicada detects sooner
	riskLowAgility := calculateDetectionRisk(3.0, 2.0, 0.2, 4.0)
	riskHighAgility := calculateDetectionRisk(3.0, 2.0, 0.9, 4.0)
	if riskHighAgility <= riskLowAgility {
		t.Errorf("High agility cicada should detect sooner: agile=%.2f, slow=%.2f",
			riskHighAgility, riskLowAgility)
	}
}

func TestCicadaSpawner_BasicGeneration(t *testing.T) {
	cfg := DefaultCicadaSpawnConfig()
	now := time.Now()

	// Generate 10 cicadas near Beijing (with ad-hoc fallback trees)
	result, err := GenerateCicadas(39.9042, 116.4074, 10, nil, cfg, now)
	if err != nil {
		t.Fatalf("GenerateCicadas error: %v", err)
	}

	// With 10 ad-hoc trees and max 3 per tree, we should get at least 8
	if result.TotalCount < 8 {
		t.Errorf("Expected at least 8 cicadas, got %d", result.TotalCount)
	}
	if result.TotalCount > 10 {
		t.Errorf("Expected at most 10 cicadas, got %d", result.TotalCount)
	}
	t.Logf("Generated %d cicadas", result.TotalCount)

	for _, c := range result.Cicadas {
		if c.ID == "" {
			t.Error("Cicada should have an ID")
		}
		if c.SpeciesName == "" {
			t.Error("Cicada should have a species name")
		}
		if c.HeightM <= 0 {
			t.Errorf("Cicada height should be positive: %.1f", c.HeightM)
		}
		if c.AlertDistM <= 0 {
			t.Errorf("Alert distance should be positive: %.1f", c.AlertDistM)
		}
	}
}

func TestCicadaSpawner_CapLimit(t *testing.T) {
	cfg := DefaultCicadaSpawnConfig()
	now := time.Now()

	result, _ := GenerateCicadas(39.9, 116.4, 100, nil, cfg, now)
	if result.TotalCount > 30 {
		t.Errorf("Should cap at 30, got %d", result.TotalCount)
	}
}
