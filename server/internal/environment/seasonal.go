package environment

import (
	"time"

	"github.com/cicada-hunt/server/internal/models"
)

// SeasonalCalendar provides seasonal context for cicada activity.
type SeasonalCalendar struct {
	// PeakStart/End defines the peak cicada emergence season.
	PeakStartMonth int // e.g. 6 (June)
	PeakEndMonth   int // e.g. 8 (August)
}

// DefaultSeasonalCalendar returns the default calendar for northern temperate regions.
func DefaultSeasonalCalendar() *SeasonalCalendar {
	return &SeasonalCalendar{
		PeakStartMonth: 6,
		PeakEndMonth:   8,
	}
}

// GetCurrentSeasonalInfo returns the seasonal context for a given time and location.
func (sc *SeasonalCalendar) GetCurrentSeasonalInfo(now time.Time, lat float64) *models.SeasonalInfo {
	month := int(now.Month())
	isNorthern := lat >= 0

	info := &models.SeasonalInfo{
		Month:         month,
		Season:        models.SeasonName(month, isNorthern),
		IsNorthernHem: isNorthern,
	}

	// Determine seasonal factor
	info.SeasonalFactor = GetSeasonalFactor(now, lat)

	// Determine if it's peak season
	effectiveMonth := month
	if !isNorthern {
		effectiveMonth = (month + 6) % 12
		if effectiveMonth == 0 {
			effectiveMonth = 12
		}
	}

	info.IsPeakSeason = effectiveMonth >= sc.PeakStartMonth && effectiveMonth <= sc.PeakEndMonth
	info.IsTransitional = effectiveMonth == sc.PeakStartMonth-1 || effectiveMonth == sc.PeakEndMonth+1

	return info
}

// GetCicadaActiveHours returns the active hours for adult cicadas based on species and season.
func GetCicadaActiveHours(species string, now time.Time) (startHour, endHour int) {
	// Most cicadas are diurnal (active during daylight)
	// Some species (e.g., Mongolian cicada) are crepuscular/nocturnal

	switch species {
	case "mongolian":
		return 19, 23 // evening-active
	case "grass_cicada":
		return 4, 7 // dawn-active
	case "meimuna":
		return 17, 22 // late afternoon to evening
	default:
		// Default: daylight hours
		return 6, 20
	}
}

// IsCicadaActive checks if a cicada would be active (singing, visible) at the given time.
func IsCicadaActive(species string, now time.Time) bool {
	start, end := GetCicadaActiveHours(species, now)
	hour := now.Hour()

	if start <= end {
		return hour >= start && hour <= end
	}
	// Wraps around midnight (e.g., 19:00 - 02:00)
	return hour >= start || hour <= end
}
