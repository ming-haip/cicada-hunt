package api

import (
	"net/http"

	"github.com/cicada-hunt/server/internal/models"
)

// PlayerHandler handles player profile and inventory requests.
type PlayerHandler struct {
	// playerStore would be injected here in the full implementation
}

// NewPlayerHandler creates a new player handler.
func NewPlayerHandler() *PlayerHandler {
	return &PlayerHandler{}
}

// GetProfile handles GET /api/v1/player
func (h *PlayerHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	playerID := GetPlayerID(r.Context())

	// TODO: Load from database
	// For now, return a stub profile
	profile := &models.Player{
		ID:        playerID,
		Nickname:  "Player",
		Level:     1,
		Exp:       0,
		GoldCoins: 100,
		Diamonds:  5,
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"player": profile,
	})
}

// GetInventory handles GET /api/v1/player/inventory
func (h *PlayerHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	playerID := GetPlayerID(r.Context())
	_ = playerID

	// Return default tool set
	tools := models.DefaultToolStats()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"player_id": playerID,
		"tools":     tools,
	})
}

// GetDailyStats handles GET /api/v1/player/daily-stats
func (h *PlayerHandler) GetDailyStats(w http.ResponseWriter, r *http.Request) {
	playerID := GetPlayerID(r.Context())

	// TODO: Query Redis for today's stats
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"player_id":     playerID,
		"today_digs":    0,
		"today_catches": 0,
		"daily_limit":   50,
		"remaining":     50,
	})
}
