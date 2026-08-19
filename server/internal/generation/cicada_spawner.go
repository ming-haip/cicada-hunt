package generation

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/cicada-hunt/server/internal/models"
	"github.com/google/uuid"
)

// CicadaSpawnConfig controls the spawning of adult cicadas on trees.
type CicadaSpawnConfig struct {
	MaxCicadasPerTree int     // max cicadas on a single tree
	MaxSpawnRadiusM   float64 // max search radius for suitable trees
	MinTreeHeightM    float64 // minimum tree height for spawning
}

// DefaultCicadaSpawnConfig returns sensible defaults for adult cicada spawning.
func DefaultCicadaSpawnConfig() CicadaSpawnConfig {
	return CicadaSpawnConfig{
		MaxCicadasPerTree: 3,
		MaxSpawnRadiusM:   200,
		MinTreeHeightM:    2.0,
	}
}

// CicadaSpawnResult holds the output of an adult cicada spawn batch.
type CicadaSpawnResult struct {
	Cicadas    []*models.CicadaSpawn `json:"cicadas"`
	TotalCount int                   `json:"total_count"`
}

// GenerateCicadas creates adult cicada spawns near a given location.
//
// Unlike nymphs (which are tied to H3 cells), adult cicadas spawn on
// specific trees within a radius. Each cicada has AI behavior state.
func GenerateCicadas(
	lat, lng float64,
	targetCount int,
	trees []TreeLocation,
	cfg CicadaSpawnConfig,
	now time.Time,
) (*CicadaSpawnResult, error) {

	if targetCount <= 0 {
		return &CicadaSpawnResult{Cicadas: []*models.CicadaSpawn{}, TotalCount: 0}, nil
	}

	if targetCount > 30 {
		targetCount = 30 // cap
	}

	// Filter suitable trees
	var suitableTrees []TreeLocation
	for _, tree := range trees {
		if tree.Preference > 0.3 {
			suitableTrees = append(suitableTrees, tree)
		}
	}

	// If no tree data, create ad-hoc tree positions (fallback)
	if len(suitableTrees) == 0 {
		suitableTrees = generateAdHocTrees(lat, lng, 10)
	}

	spawns := make([]*models.CicadaSpawn, 0, targetCount)
	usedTrees := make(map[int]int) // tree index → count

	for i := 0; i < targetCount; i++ {
		// Pick a random tree (weighted by preference)
		treeIdx := weightedTreeIndex(suitableTrees)

		// Check per-tree limit
		if usedTrees[treeIdx] >= cfg.MaxCicadasPerTree {
			continue
		}
		usedTrees[treeIdx]++

		tree := suitableTrees[treeIdx]
		cicada := createCicadaSpawn(tree, i, now)
		spawns = append(spawns, cicada)
	}

	return &CicadaSpawnResult{Cicadas: spawns, TotalCount: len(spawns)}, nil
}

// createCicadaSpawn creates a single adult cicada on a tree.
func createCicadaSpawn(tree TreeLocation, index int, now time.Time) *models.CicadaSpawn {
	species := selectCicadaSpecies()
	cfg := models.CicadaSpeciesConfigs[species]

	// Height: within species preference range, capped by tree height
	treeHeight := math.Max(tree.CanopyRadiusM*3, 3.0)
	maxHeight := math.Min(cfg.MaxHeightM, treeHeight*0.9)
	minHeight := math.Max(cfg.MinHeightM, 1.0)
	heightM := minHeight + rand.Float64()*(maxHeight-minHeight)
	heightM = math.Round(heightM*10) / 10

	// Position: on the tree
	// Add small offset from tree center
	offsetM := rand.Float64() * tree.CanopyRadiusM * 0.8
	angle := rand.Float64() * 2 * math.Pi
	latOffset := (offsetM / 111320.0) * math.Cos(angle)
	lngOffset := (offsetM / (111320.0 * math.Cos(tree.Lat*math.Pi/180.0))) * math.Sin(angle)

	// Size variation
	sizeCm := cfg.BaseSize * (0.8 + rand.Float64()*0.4)
	sizeCm = math.Round(sizeCm*10) / 10

	// Male/female (only males sing)
	isMale := rand.Float64() < 0.6 // ~60% male

	// Value
	value := cfg.BaseValue * (sizeCm / cfg.BaseSize)
	if isMale {
		value *= 1.1 // males slightly more valuable (singing, easier to find)
	}
	value = math.Round(value*100) / 100

	return &models.CicadaSpawn{
		ID:             fmt.Sprintf("cic_%s_%d", uuid.New().String()[:8], index),
		Lat:            math.Round((tree.Lat+latOffset)*1e7) / 1e7,
		Lng:            math.Round((tree.Lng+lngOffset)*1e7) / 1e7,
		AltitudeM:      heightM + 1.0, // tree base + height
		TreeID:         fmt.Sprintf("tree_%d", index),
		TreeSpecies:    tree.Species,
		HeightM:        heightM,
		FoliageDensity: 0.3 + rand.Float64()*0.5,
		Species:        species,
		SpeciesName:    cfg.Name,
		SizeCm:         sizeCm,
		IsMale:         isMale,
		Rarity:         cfg.Rarity,
		CurrentState:   models.CicadaStateResting,
		AlertDistM:     cfg.AlertDistM,
		FleeDistM:      cfg.FleeDistM,
		FlightSpeed:    cfg.BaseFlightSpeed * (0.9 + rand.Float64()*0.2),
		Agility:        cfg.Agility * (0.9 + rand.Float64()*0.2),
		Status:         models.CicadaStateResting,
		ValueEst:       value,
		CreatedAt:      now,
		SpawnedAt:      now,
	}
}

// selectCicadaSpecies picks an adult cicada species by weighted random.
func selectCicadaSpecies() models.CicadaSpecies {
	total := 0
	for _, cfg := range models.CicadaSpeciesConfigs {
		total += cfg.SpawnWeight
	}

	r := rand.Intn(total)
	cumulative := 0

	order := []models.CicadaSpecies{
		models.CicadaBlackCicada,
		models.CicadaPlatypleura,
		models.CicadaOncotympana,
		models.CicadaMeimuna,
		models.CicadaMongolian,
		models.CicadaGrassCicada,
		models.CicadaGoldenCicada,
	}

	for _, species := range order {
		cumulative += models.CicadaSpeciesConfigs[species].SpawnWeight
		if r < cumulative {
			return species
		}
	}

	return models.CicadaBlackCicada
}

// weightedTreeIndex selects a tree index biased by preference score.
func weightedTreeIndex(trees []TreeLocation) int {
	total := 0.0
	for _, t := range trees {
		total += t.Preference
	}
	if total == 0 {
		return rand.Intn(len(trees))
	}

	r := rand.Float64() * total
	cumulative := 0.0
	for i, t := range trees {
		cumulative += t.Preference
		if r <= cumulative {
			return i
		}
	}
	return len(trees) - 1
}

// generateAdHocTrees creates synthetic tree positions as fallback.
func generateAdHocTrees(lat, lng float64, count int) []TreeLocation {
	trees := make([]TreeLocation, count)
	species := []string{"poplar", "willow", "elm", "locust", "oak", "maple", "pine"}

	for i := 0; i < count; i++ {
		offsetM := rand.Float64() * 100
		angle := rand.Float64() * 2 * math.Pi
		latOffset := (offsetM / 111320.0) * math.Cos(angle)
		lngOffset := (offsetM / (111320.0 * math.Cos(lat*math.Pi/180.0))) * math.Sin(angle)

		sp := species[rand.Intn(len(species))]
		trees[i] = TreeLocation{
			Lat:           lat + latOffset,
			Lng:           lng + lngOffset,
			Species:       sp,
			Preference:    models.TreePreferenceScore[sp],
			CanopyRadiusM: 3 + rand.Float64()*8,
		}
	}

	return trees
}
