package generation

import (
	"math/rand"

	"github.com/cicada-hunt/server/internal/models"
)

// AttributeRoller generates individual nymph attributes.
type AttributeRoller struct {
	rng *rand.Rand
}

// NewAttributeRoller creates a new attribute roller.
func NewAttributeRoller() *AttributeRoller {
	return &AttributeRoller{
		rng: rand.New(rand.NewSource(rand.Int63())),
	}
}

// RollAttributes generates a complete set of attributes for a nymph at a given position.
func (ar *AttributeRoller) RollAttributes(lat, lng, depthCm float64) *models.NymphSpawn {
	species := ar.rollSpecies()

	cfg := models.NymphSpeciesConfigs[species]

	sizeCm := ar.rollSize(cfg)
	quality := ar.rollQuality(cfg)
	isRare := ar.rollRare()

	value := cfg.BaseValue * float64(quality) / 3.0
	if isRare {
		value *= 5
	}
	value = round2(value)

	return &models.NymphSpawn{
		Lat:         round7(lat),
		Lng:         round7(lng),
		DepthCm:     round1(depthCm),
		Species:     species,
		SpeciesName: cfg.Name,
		SizeCm:      round1(sizeCm),
		WeightG:     round2(cfg.BaseWeight * 1000 * (sizeCm / avgSizeRange(cfg))),
		Quality:     quality,
		IsRare:      isRare,
		ValueEst:    value,
	}
}

// rollSpecies selects a species by weighted random.
func (ar *AttributeRoller) rollSpecies() models.NymphSpecies {
	total := models.TotalSpawnWeight()
	r := ar.rng.Intn(total)

	cumulative := 0
	// Iterate in a deterministic order
	order := []models.NymphSpecies{
		models.NymphBlackCicada,
		models.NymphPlatypleura,
		models.NymphOncotympana,
		models.NymphMeimuna,
		models.NymphMongolian,
		models.NymphGrassCicada,
		models.NymphGoldenCicada,
	}

	for _, species := range order {
		cumulative += models.NymphSpeciesConfigs[species].SpawnWeight
		if r < cumulative {
			return species
		}
	}

	return models.NymphBlackCicada
}

// rollSize returns a randomized size within the species range.
func (ar *AttributeRoller) rollSize(cfg models.NymphSpeciesConfig) float64 {
	return cfg.SizeRangeMin + ar.rng.Float64()*(cfg.SizeRangeMax-cfg.SizeRangeMin)
}

// rollQuality assigns a quality rating (1-5 stars).
// Distribution: 1★:5%, 2★:20%, 3★:40%, 4★:25%, 5★:10%
func (ar *AttributeRoller) rollQuality(cfg models.NymphSpeciesConfig) int {
	// Shift distribution based on rarity
	weights := []int{5, 20, 40, 25, 10}

	// Adjust for rarity
	switch cfg.Rarity {
	case 5:
		weights = []int{0, 5, 20, 40, 35} // legendary: way higher quality floor
	case 4:
		weights = []int{0, 10, 30, 40, 20}
	case 3:
		weights = []int{3, 15, 40, 30, 12}
	case 2:
		weights = []int{5, 20, 40, 25, 10}
	default:
		weights = []int{10, 25, 40, 20, 5}
	}

	total := 0
	for _, w := range weights {
		total += w
	}

	r := ar.rng.Intn(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return i + 1
		}
	}

	return 3
}

// rollRare determines if the nymph is a rare mutation (0.2% chance).
func (ar *AttributeRoller) rollRare() bool {
	return ar.rng.Float64() < 0.002
}

// Collection-related helpers

// CanCompleteCollection checks if a player has completed a species collection.
func CanCompleteCollection(collected []models.NymphSpecies) (bool, models.NymphSpecies) {
	allSpecies := []models.NymphSpecies{
		models.NymphBlackCicada,
		models.NymphPlatypleura,
		models.NymphOncotympana,
		models.NymphMeimuna,
		models.NymphMongolian,
		models.NymphGrassCicada,
		models.NymphGoldenCicada,
	}

	collectedSet := make(map[models.NymphSpecies]bool)
	for _, s := range collected {
		collectedSet[s] = true
	}

	for _, s := range allSpecies {
		if !collectedSet[s] {
			return false, s // missing this one
		}
	}

	return true, "" // complete
}

// CalculateNymphValue computes the estimated market value of a nymph.
func CalculateNymphValue(species models.NymphSpecies, sizeCm float64, quality int, isRare bool) float64 {
	cfg := models.NymphSpeciesConfigs[species]

	value := cfg.BaseValue

	// Size bonus: +20% for max-size, -20% for min-size
	sizeRange := cfg.SizeRangeMax - cfg.SizeRangeMin
	if sizeRange > 0 {
		normalizedSize := (sizeCm - cfg.SizeRangeMin) / sizeRange
		sizeMultiplier := 0.8 + normalizedSize*0.4
		value *= sizeMultiplier
	}

	// Quality multiplier
	value *= float64(quality) / 3.0

	// Rare bonus
	if isRare {
		value *= 5
	}

	return round2(value)
}

// FormatSpeciesName returns the Chinese display name for a species.
func FormatSpeciesName(species models.NymphSpecies) string {
	if cfg, ok := models.NymphSpeciesConfigs[species]; ok {
		return cfg.Name
	}
	return string(species)
}

// Helper functions

func avgSizeRange(cfg models.NymphSpeciesConfig) float64 {
	return (cfg.SizeRangeMin + cfg.SizeRangeMax) / 2.0
}

func round7(v float64) float64 {
	return float64(int64(v*1e7+0.5)) / 1e7
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}

