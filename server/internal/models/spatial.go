package models

// GeoCoord represents a geographic coordinate.
type GeoCoord struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// CellDensityInfo contains the density information for an H3 cell.
type CellDensityInfo struct {
	H3CellLv9    string  `json:"h3_cell_lv9"`
	BaseDensity  float64 `json:"base_density"`  // nymphs per km²
	CurrDensity  float64 `json:"curr_density"`  // current (after depletion)
	TreeScore    float64 `json:"tree_score"`
	SoilScore    float64 `json:"soil_score"`
	IsHotspot    bool    `json:"is_hotspot"`
	RecommendIdx float64 `json:"recommend_idx"` // 0-1 recommendation
}

// QueryBounds defines a geographic query rectangle or circle.
type QueryBounds struct {
	CenterLat float64
	CenterLng float64
	RadiusM   float64
	MaxResult int
}

// HeatmapTileRequest parameterizes a heatmap tile generation request.
type HeatmapTileRequest struct {
	Z int `json:"z"` // zoom level
	X int `json:"x"` // tile x
	Y int `json:"y"` // tile y
}

// NymphQueryResponse wraps nymph query results with density metadata.
type NymphQueryResponse struct {
	Nymphs      []*NymphSpawn      `json:"nymphs"`
	DensityInfo []CellDensityInfo `json:"density_info"`
	TotalInArea int               `json:"total_in_area"`
}

// DigResponse reports the result of a digging action.
type DigResponse struct {
	Success         bool        `json:"success"`
	Nymph           *NymphSpawn `json:"nymph,omitempty"`
	FailReason      string      `json:"fail_reason,omitempty"`
	FailReasonCode  string      `json:"fail_reason_code,omitempty"`
	SuccessRate     float64     `json:"success_rate,omitempty"`
	CoinReward      int64       `json:"coin_reward,omitempty"`
	ExpReward       int64       `json:"exp_reward,omitempty"`
	NewToolUnlocked string      `json:"new_tool_unlocked,omitempty"`
}
