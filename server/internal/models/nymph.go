// Package models defines core data types for the cicada-hunt application.
package models

import (
	"time"
)

// NymphSpecies represents a cicada nymph species.
type NymphSpecies string

const (
	NymphBlackCicada  NymphSpecies = "black_cicada"  // 黑蚱蝉
	NymphPlatypleura  NymphSpecies = "platypleura"   // 蟪蛄
	NymphMeimuna      NymphSpecies = "meimuna"       // 寒蝉
	NymphMongolian    NymphSpecies = "mongolian"     // 蒙古寒蝉
	NymphGrassCicada  NymphSpecies = "grass_cicada"  // 草蝉
	NymphGoldenCicada NymphSpecies = "golden_cicada" // 周期蝉（金蝉）
	NymphOncotympana  NymphSpecies = "oncotympana"   // 鸣鸣蝉
)

// NymphStatus represents the lifecycle status of a nymph spawn point.
type NymphStatus string

const (
	NymphStatusActive   NymphStatus = "active"
	NymphStatusConsumed NymphStatus = "consumed"
	NymphStatusExpired  NymphStatus = "expired"
)

// NymphSpawn represents a single cicada nymph (知了猴) spawned in the game world.
type NymphSpawn struct {
	ID      string  `json:"id"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	DepthCm float64 `json:"depth_cm"`

	// H3 spatial indices
	H3CellLv9  string `json:"h3_cell_lv9"`
	H3CellLv11 string `json:"h3_cell_lv11"`

	// Attributes
	Species      NymphSpecies `json:"species"`
	SpeciesName  string       `json:"species_name"`
	SizeCm       float64      `json:"size_cm"`
	WeightG      float64      `json:"weight_g"`
	Quality      int          `json:"quality"` // 1-5 stars
	IsRare       bool         `json:"is_rare"`
	ValueEst     float64      `json:"estimated_value"`

	// Lifecycle
	Status     NymphStatus `json:"status"`
	ConsumedBy string      `json:"consumed_by,omitempty"`
	ConsumedAt *time.Time  `json:"consumed_at,omitempty"`

	// Metadata
	CreatedAt   time.Time `json:"created_at"`
	RefreshedAt time.Time `json:"refreshed_at"`
}

// NymphSpeciesConfig defines the generation parameters for a nymph species.
type NymphSpeciesConfig struct {
	Name            string
	Rarity          int     // 1-5
	BaseWeight      float64 // kg
	SizeRangeMin    float64 // cm
	SizeRangeMax    float64 // cm
	DepthRangeMin   float64 // cm
	DepthRangeMax   float64 // cm
	BaseValue       float64
	SpawnWeight     int // relative spawn weight
	PreferredDepth  float64 // cm
}

// NymphSpeciesConfigs maps each species to its configuration.
var NymphSpeciesConfigs = map[NymphSpecies]NymphSpeciesConfig{
	NymphBlackCicada: {
		Name:           "黑蚱蝉",
		Rarity:         1,
		BaseWeight:     0.8,
		SizeRangeMin:   2.5,
		SizeRangeMax:   4.5,
		DepthRangeMin:  10,
		DepthRangeMax:  40,
		BaseValue:      3.5,
		SpawnWeight:    50,
		PreferredDepth: 20,
	},
	NymphPlatypleura: {
		Name:           "蟪蛄",
		Rarity:         1,
		BaseWeight:     0.3,
		SizeRangeMin:   1.5,
		SizeRangeMax:   2.5,
		DepthRangeMin:  5,
		DepthRangeMax:  25,
		BaseValue:      2.0,
		SpawnWeight:    35,
		PreferredDepth: 12,
	},
	NymphOncotympana: {
		Name:           "鸣鸣蝉",
		Rarity:         2,
		BaseWeight:     0.6,
		SizeRangeMin:   2.8,
		SizeRangeMax:   3.8,
		DepthRangeMin:  8,
		DepthRangeMax:  35,
		BaseValue:      8.0,
		SpawnWeight:    20,
		PreferredDepth: 18,
	},
	NymphMeimuna: {
		Name:           "寒蝉",
		Rarity:         3,
		BaseWeight:     1.0,
		SizeRangeMin:   2.8,
		SizeRangeMax:   3.8,
		DepthRangeMin:  15,
		DepthRangeMax:  45,
		BaseValue:      15.0,
		SpawnWeight:    8,
		PreferredDepth: 25,
	},
	NymphMongolian: {
		Name:           "蒙古寒蝉",
		Rarity:         3,
		BaseWeight:     0.9,
		SizeRangeMin:   3.0,
		SizeRangeMax:   4.0,
		DepthRangeMin:  15,
		DepthRangeMax:  50,
		BaseValue:      30.0,
		SpawnWeight:    5,
		PreferredDepth: 30,
	},
	NymphGrassCicada: {
		Name:           "草蝉",
		Rarity:         4,
		BaseWeight:     0.2,
		SizeRangeMin:   1.0,
		SizeRangeMax:   2.0,
		DepthRangeMin:  3,
		DepthRangeMax:  15,
		BaseValue:      50.0,
		SpawnWeight:    3,
		PreferredDepth: 8,
	},
	NymphGoldenCicada: {
		Name:           "周期蝉（金蝉）",
		Rarity:         5,
		BaseWeight:     1.5,
		SizeRangeMin:   3.5,
		SizeRangeMax:   5.5,
		DepthRangeMin:  20,
		DepthRangeMax:  60,
		BaseValue:      500.0,
		SpawnWeight:    1,
		PreferredDepth: 35,
	},
}

// TotalSpawnWeight returns the sum of all spawn weights for weighted random selection.
func TotalSpawnWeight() int {
	total := 0
	for _, cfg := range NymphSpeciesConfigs {
		total += cfg.SpawnWeight
	}
	return total
}
