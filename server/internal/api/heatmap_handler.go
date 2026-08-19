package api

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// HeatmapHandler serves pre-rendered heatmap tiles showing nymph density.
type HeatmapHandler struct {
	// In production: cache rendered tiles in Redis/CDN
	tileCache map[string][]byte
}

// NewHeatmapHandler creates a new heatmap handler.
func NewHeatmapHandler() *HeatmapHandler {
	return &HeatmapHandler{
		tileCache: make(map[string][]byte),
	}
}

// ServeTile handles GET /api/v1/heatmap/{z}/{x}/{y}.png
// Returns a 256x256 PNG tile showing nymph density as a heatmap overlay.
func (h *HeatmapHandler) ServeTile(w http.ResponseWriter, r *http.Request) {
	zStr := chi.URLParam(r, "z")
	xStr := chi.URLParam(r, "x")
	yStr := chi.URLParam(r, "y")

	z, err := strconv.Atoi(zStr)
	if err != nil || z < 0 || z > 18 {
		http.Error(w, "invalid zoom level", http.StatusBadRequest)
		return
	}
	x, err := strconv.Atoi(xStr)
	if err != nil {
		http.Error(w, "invalid x coordinate", http.StatusBadRequest)
		return
	}
	y, err := strconv.Atoi(yStr)
	if err != nil {
		http.Error(w, "invalid y coordinate", http.StatusBadRequest)
		return
	}

	// Check cache
	cacheKey := tileKey(z, x, y)
	if cached, ok := h.tileCache[cacheKey]; ok {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(cached)
		return
	}

	// Generate the tile
	img := h.generateTile(z, x, y)

	// Encode to PNG
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if err := png.Encode(w, img); err != nil {
		log.Printf("[Heatmap] encode error: %v", err)
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

// generateTile renders a single 256x256 heatmap tile.
// In production, this queries the cell_environment table for density data
// and renders using a green→yellow→red color ramp.
func (h *HeatmapHandler) generateTile(z, x, y int) *image.RGBA {
	// TODO: Query cell densities for the tile's geographic bounds
	// For now, render a demo pattern showing the tile coordinates

	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	// Demo: generate a pattern based on tile coordinates
	for py := 0; py < 256; py++ {
		for px := 0; px < 256; px++ {
			// Simulate density variation based on position
			noise := math.Sin(float64(px)/50.0+float64(x)*0.5) *
				math.Cos(float64(py)/50.0+float64(y)*0.5)
			density := (noise + 1.0) / 2.0 // normalize to 0-1

			c := densityToColor(density)
			img.Set(px, py, c)
		}
	}

	return img
}

// densityToColor maps a density value (0-1) to a heatmap color.
// 0.0 → transparent, 0.3 → green, 0.6 → yellow, 1.0 → red
func densityToColor(density float64) color.RGBA {
	if density < 0.1 {
		return color.RGBA{0, 0, 0, 0} // transparent
	}

	alpha := uint8(math.Min(density*255, 200))

	switch {
	case density < 0.3:
		// Green gradient
		t := (density - 0.1) / 0.2
		return color.RGBA{
			R: uint8(100 * t),
			G: uint8(200 * (0.5 + t*0.5)),
			B: uint8(50 * t),
			A: alpha,
		}
	case density < 0.6:
		// Green → Yellow
		t := (density - 0.3) / 0.3
		return color.RGBA{
			R: uint8(100 + 155*t),
			G: uint8(200),
			B: uint8(50 * (1 - t)),
			A: alpha,
		}
	default:
		// Yellow → Red
		t := (density - 0.6) / 0.4
		return color.RGBA{
			R: 255,
			G: uint8(200 * (1 - t)),
			B: uint8(30 * (1 - t)),
			A: alpha,
		}
	}
}

func tileKey(z, x, y int) string {
	return strconv.Itoa(z) + "/" + strconv.Itoa(x) + "/" + strconv.Itoa(y)
}
