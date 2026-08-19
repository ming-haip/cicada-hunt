// Package spatial provides H3-based geospatial indexing and query utilities.
package spatial

import (
	"fmt"
	"math"
	"strconv"

	"github.com/uber/h3-go/v4"
)

// GridLevelDefault is the default H3 resolution for nymph spawning (Lv9, ~400m edge).
const GridLevelDefault = 9

// GridLevelFine is the fine H3 resolution for individual nymph placement (Lv11, ~70m edge).
const GridLevelFine = 11

// GridLevelCoarse is the coarse resolution for density heatmaps (Lv7, ~3km edge).
const GridLevelCoarse = 7

// GridLevelMicro is the micro level for caching individual tree positions (Lv13, ~20m edge).
const GridLevelMicro = 13

// CellAreaKm2 returns the area of an H3 cell at the given resolution in km².
func CellAreaKm2(resolution int) float64 {
	return h3.HexagonAreaAvgKm2(resolution)
}

// ---------------------------------------------------------------------------
// H3 Cell <-> String conversions (Cell is int64 in h3-go v4)
// ---------------------------------------------------------------------------

// cellToString converts an H3 Cell to its hexadecimal string representation.
func cellToString(c h3.Cell) string {
	return fmt.Sprintf("%x", int64(c))
}

// stringToCell converts a hexadecimal string to an H3 Cell.
func stringToCell(s string) (h3.Cell, error) {
	val, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid H3 cell: %s: %w", s, err)
	}
	return h3.Cell(val), nil
}

// ---------------------------------------------------------------------------
// Core geospatial functions
// ---------------------------------------------------------------------------

// LatLngToCell converts a lat/lng coordinate to an H3 cell index string at the given resolution.
func LatLngToCell(lat, lng float64, resolution int) string {
	cell := h3.NewLatLng(lat, lng).Cell(resolution)
	return cellToString(cell)
}

// CellToLatLng returns the center coordinate of an H3 cell.
func CellToLatLng(cellID string) (float64, float64) {
	cell, err := stringToCell(cellID)
	if err != nil {
		return 0, 0
	}
	latLng := cell.LatLng()
	return latLng.Lat, latLng.Lng
}

// CellsInRadius returns all H3 cells at the given resolution that intersect a radius circle.
func CellsInRadius(lat, lng float64, radiusM float64, resolution int) ([]string, error) {
	center := h3.NewLatLng(lat, lng)
	centerCell := center.Cell(resolution)

	// Estimate k-ring needed: radius / cell_edge_length
	// At Lv9, edge ~400m → radius 200m → k=1; radius 500m → k=2
	radiusKm := radiusM / 1000.0
	edgeKm := math.Sqrt(h3.HexagonAreaAvgKm2(resolution) / 2.6) // rough edge
	k := int(math.Ceil(radiusKm / edgeKm))
	if k < 1 {
		k = 1
	}
	if k > 10 {
		k = 10
	}

	// GridDisk gives cells in k-ring (ordered by distance)
	rings := centerCell.GridDiskDistances(k)

	var result []string
	for ringDist, ring := range rings {
		for _, cell := range ring {
			cellCenter := cell.LatLng()
			dist := haversineDistance(lat, lng, cellCenter.Lat, cellCenter.Lng)
			if dist <= radiusM {
				_ = ringDist
				result = append(result, cellToString(cell))
			}
		}
	}

	return result, nil
}

// Neighbors returns the 6 neighboring H3 cells at the same resolution.
func Neighbors(cellID string) ([]string, error) {
	cell, err := stringToCell(cellID)
	if err != nil {
		return nil, err
	}

	// GridDisk(1) returns center + 6 neighbors (k-ring 1)
	rings := cell.GridDiskDistances(1)
	var result []string
	for _, ring := range rings {
		for _, c := range ring {
			if c != cell {
				result = append(result, cellToString(c))
			}
		}
	}
	return result, nil
}

// ParentCell returns the parent cell at a coarser resolution.
func ParentCell(cellID string, parentResolution int) (string, error) {
	cell, err := stringToCell(cellID)
	if err != nil {
		return "", err
	}
	parent := cell.Parent(parentResolution)
	return cellToString(parent), nil
}

// ChildrenCells returns all child cells at a finer resolution within a parent cell.
func ChildrenCells(cellID string, childResolution int) ([]string, error) {
	cell, err := stringToCell(cellID)
	if err != nil {
		return nil, err
	}
	children := cell.Children(childResolution)

	result := make([]string, len(children))
	for i, c := range children {
		result[i] = cellToString(c)
	}
	return result, nil
}

// CellBoundary returns the vertex coordinates of an H3 cell's hexagonal boundary.
func CellBoundary(cellID string) ([]GeoCoord, error) {
	cell, err := stringToCell(cellID)
	if err != nil {
		return nil, err
	}
	boundary := cell.Boundary()

	result := make([]GeoCoord, len(boundary))
	for i, coord := range boundary {
		result[i] = GeoCoord{Lat: coord.Lat, Lng: coord.Lng}
	}
	return result, nil
}

// GeoCoord is a latitude/longitude coordinate pair.
type GeoCoord struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// IsValidH3Cell checks if a string is a valid H3 index.
func IsValidH3Cell(cellID string) bool {
	cell, err := stringToCell(cellID)
	if err != nil {
		return false
	}
	return cell.IsValid()
}

// DistanceBetweenCells returns the approximate distance in meters between two H3 cell centers.
func DistanceBetweenCells(cellA, cellB string) (float64, error) {
	latA, lngA := CellToLatLng(cellA)
	latB, lngB := CellToLatLng(cellB)
	return haversineDistance(latA, lngA, latB, lngB), nil
}

// haversineDistance calculates the great-circle distance in meters between two points.
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusM * c
}

// HaversineDistance is the public wrapper for distance calculation.
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	return haversineDistance(lat1, lng1, lat2, lng2)
}

// Bearing calculates the initial bearing from point 1 to point 2 in degrees.
func Bearing(lat1, lng1, lat2, lng2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	y := math.Sin(dLng) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLng)

	bearing := math.Atan2(y, x) * 180.0 / math.Pi
	return math.Mod(bearing+360.0, 360.0)
}

// CellGrid represents a grid of H3 cells with their environmental data.
type CellGrid struct {
	Cells map[string]*CellInfo `json:"cells"`
}

// CellInfo contains spatial metadata for an H3 cell.
type CellInfo struct {
	CellID     string     `json:"cell_id"`
	Resolution int        `json:"resolution"`
	Center     GeoCoord   `json:"center"`
	Boundary   []GeoCoord `json:"boundary"`
	AreaKm2    float64    `json:"area_km2"`
}

// NewCellInfo creates a CellInfo from an H3 cell ID.
func NewCellInfo(cellID string, resolution int) (*CellInfo, error) {
	lat, lng := CellToLatLng(cellID)
	boundary, err := CellBoundary(cellID)
	if err != nil {
		return nil, err
	}

	return &CellInfo{
		CellID:     cellID,
		Resolution: resolution,
		Center:     GeoCoord{Lat: lat, Lng: lng},
		Boundary:   boundary,
		AreaKm2:    CellAreaKm2(resolution),
	}, nil
}
