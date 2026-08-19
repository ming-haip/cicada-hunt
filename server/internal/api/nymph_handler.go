package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/cicada-hunt/server/config"
	"github.com/cicada-hunt/server/internal/service"
	"github.com/go-chi/chi/v5"
)

// NymphHandler handles HTTP requests for nymph-related operations.
type NymphHandler struct {
	nymphService *service.NymphService
	cfg          *config.Config
}

// NewNymphHandler creates a new nymph handler.
func NewNymphHandler(ns *service.NymphService, cfg *config.Config) *NymphHandler {
	return &NymphHandler{
		nymphService: ns,
		cfg:          cfg,
	}
}

// QueryNearby handles GET /api/v1/nymphs
// Query params: lat, lng, radius (optional, default 100m), limit (optional, default 20)
func (h *NymphHandler) QueryNearby(w http.ResponseWriter, r *http.Request) {
	playerID := GetPlayerID(r.Context())

	// Parse query parameters
	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_lat", "lat parameter is required and must be a number")
		return
	}

	lng, err := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_lng", "lng parameter is required and must be a number")
		return
	}

	radiusM := h.cfg.MaxQueryRadiusM
	if r := r.URL.Query().Get("radius"); r != "" {
		if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 && parsed <= h.cfg.MaxQueryRadiusM {
			radiusM = parsed
		}
	}

	limit := h.cfg.MaxResultsPerQuery
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= h.cfg.MaxResultsPerQuery {
			limit = parsed
		}
	}

	log.Printf("[API] Player %s querying nymphs at (%.5f, %.5f) radius=%.0fm limit=%d",
		playerID, lat, lng, radiusM, limit)

	resp, err := h.nymphService.QueryNearbyNymphs(r.Context(), lat, lng, radiusM, limit)
	if err != nil {
		log.Printf("[API] QueryNearby error: %v", err)
		writeError(w, http.StatusInternalServerError, "query_failed", "Failed to query nearby nymphs")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetNymph handles GET /api/v1/nymphs/{nymphID}
func (h *NymphHandler) GetNymph(w http.ResponseWriter, r *http.Request) {
	nymphID := chi.URLParam(r, "nymphID")
	if nymphID == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "nymph ID is required")
		return
	}

	// TODO: Implement single nymph lookup via NymphService
	writeJSON(w, http.StatusOK, map[string]string{
		"nymph_id": nymphID,
		"message":  "single nymph lookup pending",
	})
}

// DigNymph handles POST /api/v1/nymphs/{nymphID}/dig
func (h *NymphHandler) DigNymph(w http.ResponseWriter, r *http.Request) {
	playerID := GetPlayerID(r.Context())
	nymphID := chi.URLParam(r, "nymphID")

	if nymphID == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "nymph ID is required")
		return
	}

	// Parse request body
	var req struct {
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		DistanceM   float64 `json:"distance_m"`
		DeviationCm float64 `json:"deviation_cm"`
		AngleDeg    float64 `json:"angle_deg"`
		ToolUsed    string  `json:"tool_used"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	if req.ToolUsed == "" {
		req.ToolUsed = "bare_hand"
	}

	log.Printf("[API] Player %s digging nymph %s at (%.5f, %.5f) dist=%.1fm dev=%.1fcm angle=%.1f° tool=%s",
		playerID, nymphID, req.Lat, req.Lng, req.DistanceM, req.DeviationCm, req.AngleDeg, req.ToolUsed)

	resp, err := h.nymphService.DigNymph(
		r.Context(),
		playerID, nymphID,
		req.Lat, req.Lng,
		req.DistanceM, req.DeviationCm, req.AngleDeg,
		req.ToolUsed,
	)
	if err != nil {
		log.Printf("[API] DigNymph error: %v", err)
		writeError(w, http.StatusInternalServerError, "dig_failed", "Failed to process digging action")
		return
	}

	if !resp.Success {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
