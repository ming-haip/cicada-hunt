package models

import "time"

// Player represents a game player's profile and inventory.
type Player struct {
	ID            string    `json:"id"`
	Nickname      string    `json:"nickname"`
	Level         int       `json:"level"`
	Exp           int64     `json:"exp"`
	GoldCoins     int64     `json:"gold_coins"`
	Diamonds      int       `json:"diamonds"`

	// Stats
	TotalDigs     int `json:"total_digs"`
	TotalCatches  int `json:"total_catches"`
	TotalNymphs   int `json:"total_nymphs"`
	RareNymphs    int `json:"rare_nymphs"`
	LegendaryCaptures int `json:"legendary_captures"`

	// Daily limits (stored in Redis)
	// TodayDigs, TodayCatches computed at query time

	// Equipment
	CurrentShovel string `json:"current_shovel"` // e.g. "small_shovel"
	CurrentNet    string `json:"current_net"`    // e.g. "basic_net"
	CurrentRadar  string `json:"current_radar"`  // e.g. "basic_radar"

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DigEvent represents a single digging action by a player.
type DigEvent struct {
	PlayerID   string    `json:"player_id"`
	NymphID    string    `json:"nymph_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	DistanceM  float64   `json:"distance_m"`  // distance to nymph
	DeviationCm float64  `json:"deviation_cm"` // AR precision deviation
	AngleDeg   float64   `json:"angle_deg"`   // bearing deviation
	ToolUsed   string    `json:"tool_used"`
	Success    bool      `json:"success"`
	DigTime    time.Time `json:"dig_time"`
}

// CatchEvent represents a single cicada catching action.
type CatchEvent struct {
	PlayerID    string    `json:"player_id"`
	CicadaID    string    `json:"cicada_id"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	DistanceM   float64   `json:"distance_m"`
	AngleDeg    float64   `json:"angle_deg"`
	NetUsed     string    `json:"net_used"`
	SwingSpeed  float64   `json:"swing_speed_ms"`
	Success     bool      `json:"success"`
	CicadaEvaded bool     `json:"cicada_evaded"`
	CatchTime   time.Time `json:"catch_time"`
}

// ToolStats defines the properties of a tool (shovel, net, radar).
type ToolStats struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"` // "shovel", "net", "radar"
	Level        int     `json:"level"`
	Efficiency   float64 `json:"efficiency"`    // 1.0 = baseline
	Accuracy     float64 `json:"accuracy"`      // 0-1
	MaxReachM    float64 `json:"max_reach_m"`   // max reach distance
	DigsRequired int     `json:"digs_required"` // for shovels
	ConeAngleDeg float64 `json:"cone_angle_deg"` // for nets
	Price        int64   `json:"price"`
	Description  string  `json:"description"`
}

// DefaultToolStats returns the starter tool configurations.
func DefaultToolStats() map[string]ToolStats {
	return map[string]ToolStats{
		"bare_hand": {
			Name:         "徒手",
			Type:         "shovel",
			Level:        0,
			Efficiency:   1.0,
			Accuracy:     0.60,
			MaxReachM:    1.0,
			DigsRequired: 12,
			Price:        0,
			Description:  "直接用手挖掘，效率较低",
		},
		"small_shovel": {
			Name:         "小铲子",
			Type:         "shovel",
			Level:        1,
			Efficiency:   1.5,
			Accuracy:     0.80,
			MaxReachM:    1.5,
			DigsRequired: 8,
			Price:        50,
			Description:  "基础挖掘工具，提升挖掘速度与成功率",
		},
		"pro_shovel": {
			Name:         "专业挖掘铲",
			Type:         "shovel",
			Level:        2,
			Efficiency:   2.2,
			Accuracy:     0.95,
			MaxReachM:    2.0,
			DigsRequired: 5,
			Price:        200,
			Description:  "专业级挖掘铲，精准高效",
		},
		"basic_net": {
			Name:          "基础捕蝉网",
			Type:          "net",
			Level:         0,
			Efficiency:    1.0,
			Accuracy:      0.65,
			MaxReachM:     2.0,
			ConeAngleDeg:  30,
			Price:         0,
			Description:   "入门捕蝉网，适合捕捉普通蝉类",
		},
		"telescopic_net": {
			Name:          "伸缩捕蝉网",
			Type:          "net",
			Level:         1,
			Efficiency:    1.3,
			Accuracy:      0.75,
			MaxReachM:     3.5,
			ConeAngleDeg:  25,
			Price:         150,
			Description:   "可伸缩设计，触及更高树枝",
		},
		"carbon_fiber_net": {
			Name:          "碳纤维竞技网",
			Type:          "net",
			Level:         2,
			Efficiency:    1.6,
			Accuracy:      0.90,
			MaxReachM:     4.0,
			ConeAngleDeg:  20,
			Price:         500,
			Description:   "轻量碳纤维，静音挥网不惊动蝉",
		},
		"basic_radar": {
			Name:         "基础雷达",
			Type:         "radar",
			Level:        0,
			Efficiency:   1.0,
			MaxReachM:    200,
			Price:        0,
			Description:  "探测半径200m，波束宽度15°",
		},
	}
}
