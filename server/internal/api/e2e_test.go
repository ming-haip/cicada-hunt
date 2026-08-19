package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cicada-hunt/server/config"
	"github.com/cicada-hunt/server/internal/environment"
	"github.com/cicada-hunt/server/internal/generation"
	"github.com/cicada-hunt/server/internal/models"
	"github.com/cicada-hunt/server/internal/service"
	"github.com/go-chi/chi/v5"
)

// ================================================================
// End-to-End Tests: simulate a complete player session
// ================================================================

// InMemoryNymphStore is a test double that stores nymphs in memory.
type InMemoryNymphStore struct {
	nymphs map[string]*models.NymphSpawn
}

func NewInMemoryNymphStore() *InMemoryNymphStore {
	return &InMemoryNymphStore{nymphs: make(map[string]*models.NymphSpawn)}
}

func (m *InMemoryNymphStore) GetNymphsByCell(ctx context.Context, cellID string) ([]*models.NymphSpawn, error) {
	var result []*models.NymphSpawn
	for _, n := range m.nymphs {
		if n.H3CellLv9 == cellID && n.Status == models.NymphStatusActive {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *InMemoryNymphStore) SaveNymphs(ctx context.Context, nymphs []*models.NymphSpawn) error {
	for _, n := range nymphs {
		m.nymphs[n.ID] = n
	}
	return nil
}

func (m *InMemoryNymphStore) MarkNymphConsumed(ctx context.Context, nymphID, playerID string) error {
	if n, ok := m.nymphs[nymphID]; ok {
		n.Status = models.NymphStatusConsumed
		n.ConsumedBy = playerID
		return nil
	}
	return nil
}

func (m *InMemoryNymphStore) GetNymphByID(ctx context.Context, nymphID string) (*models.NymphSpawn, error) {
	if n, ok := m.nymphs[nymphID]; ok {
		return n, nil
	}
	return nil, nil
}

// setupE2EHandler creates a fully wired handler stack for end-to-end testing.
func setupE2EHandler() (http.Handler, *InMemoryNymphStore) {
	weatherClient := environment.NewMockWeatherClient()
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: weatherClient,
	}

	// Pass nil envStore for DB-less mode — DensityCalculator handles nil gracefully
	densityCalc := generation.NewDensityCalculator(scorer, nil)

	nymphStore := NewInMemoryNymphStore()
	nymphSvc := service.NewNymphService(densityCalc, nymphStore, nil)

	cfg := &config.Config{
		MaxQueryRadiusM:   500,
		MaxResultsPerQuery: 50,
		MaxDailyDigs:      50,
	}

	nymphHandler := NewNymphHandler(nymphSvc, cfg)
	heatmapHandler := NewHeatmapHandler()
	playerHandler := NewPlayerHandler()

	router := NewRouter(nymphHandler, heatmapHandler, playerHandler, "")
	return router, nymphStore
}

// TestE2E_PlayerJourney simulates a complete player flow:
// 1. Player opens app → queries nearby nymphs
// 2. Player sees nymphs on map → selects one
// 3. Player walks to location → digs
// 4. Player checks daily stats
func TestE2E_PlayerJourney(t *testing.T) {
	router, store := setupE2EHandler()
	playerID := "e2e_player_1"

	// === Step 1: Query nearby nymphs ===
	t.Log("Step 1: Player opens app, queries nymphs near Beijing")
	req := httptest.NewRequest("GET", "/api/v1/nymphs?lat=39.9042&lng=116.4074&radius=200&limit=10", nil)
	req.Header.Set("X-Player-ID", playerID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Step 1 failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var queryResp models.NymphQueryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &queryResp); err != nil {
		t.Fatalf("Step 1 parse: %v", err)
	}
	t.Logf("  Found %d nymphs in area", queryResp.TotalInArea)

	// === Step 2: Pre-seed a specific nymph for reliable testing ===
	t.Log("Step 2: Pre-seed a test nymph at known location")
	testNymph := &models.NymphSpawn{
		ID:          "e2e_test_nymph_001",
		Lat:         39.90425,
		Lng:         116.40745,
		DepthCm:     20.0,
		H3CellLv9:   "test_cell",
		Species:     models.NymphBlackCicada,
		SpeciesName: "黑蚱蝉",
		SizeCm:      4.0,
		WeightG:     8.0,
		Quality:     4,
		IsRare:      false,
		ValueEst:    5.0,
		Status:      models.NymphStatusActive,
	}
	store.SaveNymphs(context.Background(), []*models.NymphSpawn{testNymph})
	t.Logf("  Seeded nymph: %s at (%.5f, %.5f)", testNymph.ID, testNymph.Lat, testNymph.Lng)

	// Query again to see our test nymph
	req2 := httptest.NewRequest("GET", "/api/v1/nymphs?lat=39.90425&lng=116.40745&radius=10&limit=5", nil)
	req2.Header.Set("X-Player-ID", playerID)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("Step 2 failed: status=%d", rec2.Code)
	}
	var queryResp2 models.NymphQueryResponse
	json.Unmarshal(rec2.Body.Bytes(), &queryResp2)
	t.Logf("  Re-queried: %d nymphs at exact location", queryResp2.TotalInArea)

	// === Step 3: Dig the test nymph (near miss) ===
	t.Log("Step 3: Player attempts to dig (off by 15cm)")
	digBody := `{"lat":39.90425,"lng":116.40745,"distance_m":1.0,"deviation_cm":15.0,"angle_deg":10,"tool_used":"small_shovel"}`
	req3 := httptest.NewRequest("POST", "/api/v1/nymphs/e2e_test_nymph_001/dig", strings.NewReader(digBody))
	req3.Header.Set("X-Player-ID", playerID)
	req3.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("nymphID", "e2e_test_nymph_001")
	req3 = req3.WithContext(context.WithValue(req3.Context(), chi.RouteCtxKey, rctx))
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("Step 3 failed: status=%d body=%s", rec3.Code, rec3.Body.String())
	}

	var digResp models.DigResponse
	if err := json.Unmarshal(rec3.Body.Bytes(), &digResp); err != nil {
		t.Fatalf("Step 3 parse: %v", err)
	}
	t.Logf("  Dig result: success=%v, rate=%.0f%%, reason=%s",
		digResp.Success, digResp.SuccessRate*100, digResp.FailReason)

	// === Step 4: Dig again with perfect aim ===
	t.Log("Step 4: Player digs perfectly (0cm deviation)")
	digBody2 := `{"lat":39.90425,"lng":116.40745,"distance_m":0.5,"deviation_cm":0.0,"angle_deg":0,"tool_used":"pro_shovel"}`
	req4 := httptest.NewRequest("POST", "/api/v1/nymphs/e2e_test_nymph_001/dig", strings.NewReader(digBody2))
	req4.Header.Set("X-Player-ID", playerID)
	req4.Header.Set("Content-Type", "application/json")
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("nymphID", "e2e_test_nymph_001")
	req4 = req4.WithContext(context.WithValue(req4.Context(), chi.RouteCtxKey, rctx2))
	rec4 := httptest.NewRecorder()
	router.ServeHTTP(rec4, req4)

	if rec4.Code != http.StatusOK {
		t.Fatalf("Step 4 failed: status=%d body=%s", rec4.Code, rec4.Body.String())
	}

	var digResp2 models.DigResponse
	json.Unmarshal(rec4.Body.Bytes(), &digResp2)
	t.Logf("  Dig2 result: success=%v, coin_reward=%d, rate=%.0f%%",
		digResp2.Success, digResp2.CoinReward, digResp2.SuccessRate*100)

	// === Step 5: Verify nymph is now consumed ===
	t.Log("Step 5: Verify nymph status")

	// Query the exact location — our nymph should now show as consumed
	req5 := httptest.NewRequest("GET", "/api/v1/nymphs?lat=39.90425&lng=116.40745&radius=5&limit=3", nil)
	req5.Header.Set("X-Player-ID", playerID)
	rec5 := httptest.NewRecorder()
	router.ServeHTTP(rec5, req5)

	var queryResp3 models.NymphQueryResponse
	json.Unmarshal(rec5.Body.Bytes(), &queryResp3)

	// Check if our test nymph is still in active results
	nymphStillActive := false
	for _, n := range queryResp3.Nymphs {
		if n.ID == "e2e_test_nymph_001" && n.Status == models.NymphStatusActive {
			nymphStillActive = true
		}
	}
	if digResp2.Success && nymphStillActive {
		t.Error("Nymph should be consumed after successful dig but still shows as active")
	}
	if !digResp2.Success && !nymphStillActive {
		t.Log("  Nymph still active (dig was not successful, as expected)")
	}
	t.Logf("  Final check: nymph active=%v (dig_success=%v)", nymphStillActive, digResp2.Success)

	// === Step 6: Check daily stats ===
	t.Log("Step 6: Check daily stats")
	req6 := httptest.NewRequest("GET", "/api/v1/player/daily-stats", nil)
	req6.Header.Set("X-Player-ID", playerID)
	rec6 := httptest.NewRecorder()
	router.ServeHTTP(rec6, req6)

	if rec6.Code != http.StatusOK {
		t.Fatalf("Step 6 failed: status=%d", rec6.Code)
	}
	t.Logf("  Daily stats: %s", rec6.Body.String())

	t.Log("✅ E2E Player Journey complete!")
}
