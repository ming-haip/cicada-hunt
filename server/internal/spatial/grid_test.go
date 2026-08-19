package spatial

import (
	"math"
	"testing"
)

func TestLatLngToCell_Roundtrip(t *testing.T) {
	lat, lng := 39.9042, 116.4074
	cellID := LatLngToCell(lat, lng, GridLevelDefault)

	if cellID == "" {
		t.Fatal("Cell ID should not be empty")
	}

	// Convert back
	lat2, lng2 := CellToLatLng(cellID)

	// Center should be close to original (within ~200m for Lv9)
	dist := HaversineDistance(lat, lng, lat2, lng2)
	if dist > 300 {
		t.Errorf("Round-trip distance too large: %.0fm", dist)
	}
}

func TestIsValidH3Cell(t *testing.T) {
	// Valid cell
	lat, lng := 39.9, 116.4
	cellID := LatLngToCell(lat, lng, 9)
	if !IsValidH3Cell(cellID) {
		t.Errorf("Generated cell %s should be valid", cellID)
	}

	// Invalid cells
	if IsValidH3Cell("") {
		t.Error("Empty string should be invalid")
	}
	if IsValidH3Cell("not_a_cell") {
		t.Error("Random string should be invalid")
	}
	if IsValidH3Cell("zzzz") {
		t.Error("Invalid hex should be invalid")
	}
}

func TestCellsInRadius(t *testing.T) {
	lat, lng := 39.9042, 116.4074
	radiusM := 200.0

	cells, err := CellsInRadius(lat, lng, radiusM, GridLevelDefault)
	if err != nil {
		t.Fatalf("CellsInRadius error: %v", err)
	}

	if len(cells) == 0 {
		t.Error("Should return at least 1 cell")
	}

	// All returned cells should be within radius
	for _, cellID := range cells {
		cLat, cLng := CellToLatLng(cellID)
		dist := HaversineDistance(lat, lng, cLat, cLng)
		if dist > radiusM+500 { // +500m tolerance for cell center vs edge
			t.Errorf("Cell %s center at %.4f,%.4f is %.0fm from query point (radius=%.0fm)",
				cellID, cLat, cLng, dist, radiusM)
		}
	}
}

func TestNeighbors(t *testing.T) {
	lat, lng := 39.9, 116.4
	cellID := LatLngToCell(lat, lng, GridLevelDefault)

	neighbors, err := Neighbors(cellID)
	if err != nil {
		t.Fatalf("Neighbors error: %v", err)
	}

	if len(neighbors) != 6 {
		t.Errorf("Expected 6 neighbors, got %d", len(neighbors))
	}

	// Each neighbor should be valid
	for _, n := range neighbors {
		if !IsValidH3Cell(n) {
			t.Errorf("Neighbor %s should be valid", n)
		}
	}
}

func TestParentAndChildren(t *testing.T) {
	lat, lng := 39.9, 116.4
	cellID := LatLngToCell(lat, lng, GridLevelFine) // Lv11

	// Get parent at Lv9
	parent, err := ParentCell(cellID, GridLevelDefault)
	if err != nil {
		t.Fatalf("ParentCell error: %v", err)
	}
	if !IsValidH3Cell(parent) {
		t.Error("Parent cell should be valid")
	}

	// Get children of parent at Lv11
	children, err := ChildrenCells(parent, GridLevelFine)
	if err != nil {
		t.Fatalf("ChildrenCells error: %v", err)
	}
	if len(children) < 4 {
		t.Errorf("Expected at least 4 children, got %d", len(children))
	}

	// One of the children should be the original cell
	found := false
	for _, c := range children {
		if c == cellID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Original cell should be among children of its parent")
	}
}

func TestHaversineDistance(t *testing.T) {
	// Beijing → Shanghai: ~1068 km
	bjLat, bjLng := 39.9042, 116.4074
	shLat, shLng := 31.2304, 121.4737

	dist := HaversineDistance(bjLat, bjLng, shLat, shLng)

	// Expected: ~1,068,000 meters
	if dist < 1_000_000 || dist > 1_150_000 {
		t.Errorf("Beijing→Shanghai distance: %.0fm (expected ~1,068,000m)", dist)
	}

	// Same point should be 0
	distZero := HaversineDistance(39.9, 116.4, 39.9, 116.4)
	if distZero != 0 {
		t.Errorf("Same point distance should be 0, got %.2f", distZero)
	}

	// ~1 degree latitude = ~111km
	dist1deg := HaversineDistance(39.0, 116.0, 40.0, 116.0)
	if dist1deg < 110_000 || dist1deg > 113_000 {
		t.Errorf("1° latitude should be ~111,320m, got %.0fm", dist1deg)
	}
}

func TestBearing(t *testing.T) {
	// Due North
	bearing := Bearing(39.0, 116.0, 40.0, 116.0)
	if math.Abs(bearing-0.0) > 1.0 {
		t.Errorf("North bearing should be ~0°, got %.1f°", bearing)
	}

	// Due East
	bearing = Bearing(39.0, 116.0, 39.0, 117.0)
	if math.Abs(bearing-90.0) > 2.0 {
		t.Errorf("East bearing should be ~90°, got %.1f°", bearing)
	}

	// Due South
	bearing = Bearing(40.0, 116.0, 39.0, 116.0)
	if math.Abs(bearing-180.0) > 2.0 {
		t.Errorf("South bearing should be ~180°, got %.1f°", bearing)
	}
}

func TestCellArea(t *testing.T) {
	area := CellAreaKm2(GridLevelDefault) // Lv9
	if area < 0.01 || area > 1.0 {
		t.Errorf("Lv9 cell area should be ~0.1 km², got %.4f km²", area)
	}

	area2 := CellAreaKm2(GridLevelCoarse) // Lv7
	if area2 < area {
		t.Error("Lv7 cell should be larger than Lv9")
	}
}
