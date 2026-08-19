package models

import "time"

// CicadaState represents the behavioral state of an adult cicada.
type CicadaState string

const (
	CicadaStateResting        CicadaState = "resting"
	CicadaStateRestingSinging CicadaState = "resting_singing"
	CicadaStateAlert          CicadaState = "alert"
	CicadaStateFlying         CicadaState = "flying"
	CicadaStateStartled       CicadaState = "startled"
	CicadaStateCaptured       CicadaState = "captured"
)

// FlightType categorizes the flight behavior of a cicada.
type FlightType string

const (
	FlightTypeCasual  FlightType = "casual"  // 随意飞行（自发换树）
	FlightTypeEvasive FlightType = "evasive" // 逃避飞行（被追踪）
	FlightTypePanic   FlightType = "panic"   // 恐慌飞行（受惊）
)

// CicadaSpecies represents an adult cicada species.
type CicadaSpecies string

const (
	CicadaBlackCicada  CicadaSpecies = "black_cicada"
	CicadaPlatypleura  CicadaSpecies = "platypleura"
	CicadaOncotympana  CicadaSpecies = "oncotympana"
	CicadaMeimuna      CicadaSpecies = "meimuna"
	CicadaMongolian    CicadaSpecies = "mongolian"
	CicadaGrassCicada  CicadaSpecies = "grass_cicada"
	CicadaGoldenCicada CicadaSpecies = "golden_cicada"
)

// CicadaSpawn represents an adult cicada that can be tracked and caught.
type CicadaSpawn struct {
	ID        string  `json:"id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	AltitudeM float64 `json:"altitude_m"`

	// Habitat
	TreeID         string  `json:"tree_id"`
	TreeSpecies    string  `json:"tree_species"`
	HeightM        float64 `json:"height_m"`        // height on tree
	FoliageDensity float64 `json:"foliage_density"` // 0-1

	// Attributes
	Species     CicadaSpecies `json:"species"`
	SpeciesName string        `json:"species_name"`
	SizeCm      float64       `json:"size_cm"`
	IsMale      bool          `json:"is_male"`
	Rarity      int           `json:"rarity"` // 1-5

	// Behavior
	CurrentState CicadaState `json:"current_state"`
	AlertDistM   float64     `json:"alert_distance_m"`   // alert radius
	FleeDistM    float64     `json:"flee_distance_m"`    // flee distance
	FlightSpeed  float64     `json:"flight_speed_ms"`    // m/s
	Agility      float64     `json:"agility"`            // 0-1 evasion skill

	// Lifecycle
	Status      CicadaState `json:"status"`
	CapturedBy  string      `json:"captured_by,omitempty"`
	CapturedAt  *time.Time  `json:"captured_at,omitempty"`
	StartledUntil *time.Time `json:"startled_until,omitempty"`

	// Economic
	ValueEst float64 `json:"estimated_value"`

	// Metadata
	CreatedAt   time.Time `json:"created_at"`
	SpawnedAt   time.Time `json:"spawned_at"`
}

// CicadaSpeciesConfig defines generation parameters for adult cicada species.
type CicadaSpeciesConfig struct {
	Name             string
	Rarity           int
	BaseSize         float64
	MinHeightM       float64
	MaxHeightM       float64
	BaseFlightSpeed  float64
	Agility          float64
	AlertDistM       float64
	FleeDistM        float64
	SpawnWeight      int
	ActiveHoursStart int
	ActiveHoursEnd   int
	BaseValue        float64
}

// CicadaSpeciesConfigs maps each adult cicada species to its generation config.
var CicadaSpeciesConfigs = map[CicadaSpecies]CicadaSpeciesConfig{
	CicadaBlackCicada: {
		Name:            "黑蚱蝉",
		Rarity:          1,
		BaseSize:        4.2,
		MinHeightM:      1.0,
		MaxHeightM:      4.0,
		BaseFlightSpeed: 2.5,
		Agility:          0.2,
		AlertDistM:      4.0,
		FleeDistM:       15.0,
		SpawnWeight:     45,
		ActiveHoursStart: 6,
		ActiveHoursEnd:   20,
		BaseValue:       5.0,
	},
	CicadaPlatypleura: {
		Name:            "蟪蛄",
		Rarity:          1,
		BaseSize:        2.2,
		MinHeightM:      0.5,
		MaxHeightM:      2.5,
		BaseFlightSpeed: 3.0,
		Agility:          0.3,
		AlertDistM:      3.0,
		FleeDistM:       10.0,
		SpawnWeight:     35,
		ActiveHoursStart: 5,
		ActiveHoursEnd:   21,
		BaseValue:       3.0,
	},
	CicadaOncotympana: {
		Name:            "鸣鸣蝉",
		Rarity:          2,
		BaseSize:        3.5,
		MinHeightM:      1.5,
		MaxHeightM:      5.0,
		BaseFlightSpeed: 4.0,
		Agility:          0.4,
		AlertDistM:      5.0,
		FleeDistM:       25.0,
		SpawnWeight:     25,
		ActiveHoursStart: 7,
		ActiveHoursEnd:   19,
		BaseValue:       12.0,
	},
	CicadaMeimuna: {
		Name:            "寒蝉",
		Rarity:          3,
		BaseSize:        3.2,
		MinHeightM:      2.0,
		MaxHeightM:      6.0,
		BaseFlightSpeed: 5.0,
		Agility:          0.6,
		AlertDistM:      8.0,
		FleeDistM:       40.0,
		SpawnWeight:     8,
		ActiveHoursStart: 17,
		ActiveHoursEnd:   22,
		BaseValue:       30.0,
	},
	CicadaMongolian: {
		Name:            "蒙古寒蝉",
		Rarity:          3,
		BaseSize:        3.5,
		MinHeightM:      3.0,
		MaxHeightM:      8.0,
		BaseFlightSpeed: 6.5,
		Agility:          0.7,
		AlertDistM:      10.0,
		FleeDistM:       50.0,
		SpawnWeight:     5,
		ActiveHoursStart: 19,
		ActiveHoursEnd:   23,
		BaseValue:       50.0,
	},
	CicadaGrassCicada: {
		Name:            "草蝉",
		Rarity:          4,
		BaseSize:        1.8,
		MinHeightM:      0.5,
		MaxHeightM:      2.0,
		BaseFlightSpeed: 8.0,
		Agility:          0.85,
		AlertDistM:      6.0,
		FleeDistM:       30.0,
		SpawnWeight:     3,
		ActiveHoursStart: 4,
		ActiveHoursEnd:   7,
		BaseValue:       100.0,
	},
	CicadaGoldenCicada: {
		Name:            "周期蝉（金蝉）",
		Rarity:          5,
		BaseSize:        3.0,
		MinHeightM:      1.0,
		MaxHeightM:      10.0,
		BaseFlightSpeed: 8.0,
		Agility:          0.95,
		AlertDistM:      15.0,
		FleeDistM:       100.0,
		SpawnWeight:     1,
		ActiveHoursStart: 0,
		ActiveHoursEnd:   24,
		BaseValue:       500.0,
	},
}
