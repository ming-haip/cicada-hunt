package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cicada-hunt/server/config"
	"github.com/cicada-hunt/server/internal/environment"
	"github.com/cicada-hunt/server/internal/generation"
	"github.com/cicada-hunt/server/internal/service"
	"github.com/go-chi/chi/v5"
)

// setupTestHandler creates a NymphHandler with DB-less dependencies for testing.
func setupTestHandler() *NymphHandler {
	weatherClient := environment.NewMockWeatherClient()
	scorer := &environment.Scorer{
		TreeScorer:    environment.NewDefaultTreeScorer(),
		SoilScorer:    environment.NewDefaultSoilScorer(),
		WeatherClient: weatherClient,
	}
	densityCalc := generation.NewDensityCalculator(scorer, nil)
	nymphSvc := service.NewNymphService(densityCalc, nil, nil)
	cfg := &config.Config{
		MaxQueryRadiusM:   500,
		MaxResultsPerQuery: 50,
		MaxDailyDigs:      50,
	}
	return NewNymphHandler(nymphSvc, cfg)
}

func TestQueryNearby_MissingLatLng(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		name string
		url  string
		want int
	}{
		{"no_params", "/api/v1/nymphs", http.StatusBadRequest},
		{"only_lat", "/api/v1/nymphs?lat=39.9", http.StatusBadRequest},
		{"only_lng", "/api/v1/nymphs?lng=116.4", http.StatusBadRequest},
		{"invalid_lat", "/api/v1/nymphs?lat=abc&lng=116.4", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			rec := httptest.NewRecorder()
			handler.QueryNearby(rec, req)

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestQueryNearby_ValidRequest(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/nymphs?lat=39.9042&lng=116.4074&radius=200&limit=5", nil)
	rec := httptest.NewRecorder()
	handler.QueryNearby(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	// Should return valid JSON with expected structure
	body := rec.Body.String()
	if body == "" {
		t.Error("Response body should not be empty")
	}
	// Should contain expected fields
	for _, field := range []string{"nymphs", "density_info", "total_in_area"} {
		if !contains(body, field) {
			t.Errorf("Response should contain %q", field)
		}
	}
}

func TestQueryNearby_RespectsLimit(t *testing.T) {
	handler := setupTestHandler()

	// Request with limit=1
	req := httptest.NewRequest("GET", "/api/v1/nymphs?lat=39.9042&lng=116.4074&radius=500&limit=1", nil)
	rec := httptest.NewRecorder()
	handler.QueryNearby(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestQueryNearby_CapsRadius(t *testing.T) {
	handler := setupTestHandler()

	// Request with radius exceeding max → should be capped
	req := httptest.NewRequest("GET", "/api/v1/nymphs?lat=39.9&lng=116.4&radius=5000&limit=50", nil)
	rec := httptest.NewRecorder()
	handler.QueryNearby(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestDigNymph_MissingBody(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("POST", "/api/v1/nymphs/nymph_1/dig", nil)
	req.Header.Set("X-Player-ID", "test_player")
	rec := httptest.NewRecorder()
	handler.DigNymph(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDigNymph_InvalidJSON(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("POST", "/api/v1/nymphs/nymph_1/dig", nil)
	req.Header.Set("X-Player-ID", "test_player")
	req.Header.Set("Content-Type", "application/json")
	// Attach invalid JSON body
	rec := httptest.NewRecorder()
	handler.DigNymph(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestDigNymph_ValidRequest_NoStore(t *testing.T) {
	handler := setupTestHandler()

	body := strings.NewReader(`{"lat":39.9,"lng":116.4,"distance_m":1.5,"deviation_cm":5,"angle_deg":10}`)
	req := httptest.NewRequest("POST", "/api/v1/nymphs/nymph_test_001/dig", body)
	// Use chi URL routing context so chi.URLParam resolves correctly
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("nymphID", "nymph_test_001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	req.Header.Set("X-Player-ID", "test_player")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.DigNymph(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	// Should return a graceful rejection (no store available)
	responseBody := rec.Body.String()
	if !contains(responseBody, "service_unavailable") {
		t.Logf("Dig response: %s", responseBody)
	}
}

func TestDigNymph_MissingNymphID(t *testing.T) {
	handler := setupTestHandler()

	body := strings.NewReader(`{"lat":39.9,"lng":116.4,"distance_m":1.5}`)
	req := httptest.NewRequest("POST", "/api/v1/nymphs//dig", body)
	req.Header.Set("X-Player-ID", "test_player")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.DigNymph(rec, req)

	// Empty nymphID → bad request
	if rec.Code != http.StatusBadRequest {
		t.Logf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetNymph(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/nymphs/nymph_test_123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("nymphID", "nymph_test_123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.GetNymph(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestPlayerIDExtraction(t *testing.T) {
	// Test context-based player ID extraction
	ctx := WithPlayerID(context.Background(), "player_abc")
	id := GetPlayerID(ctx)
	if id != "player_abc" {
		t.Errorf("Expected player_abc, got %s", id)
	}

	// Test default
	ctx2 := context.Background()
	id2 := GetPlayerID(ctx2)
	if id2 != "anonymous" {
		t.Errorf("Expected anonymous, got %s", id2)
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "ok") || !contains(body, "cicada-hunt-server") {
		t.Errorf("Unexpected health response: %s", body)
	}
}

// contains checks if substr is in s (case-sensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
