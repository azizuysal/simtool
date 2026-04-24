package simulator

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srwiley/oksvg"
)

// writeTestPNG writes a solid-color PNG of the given dimensions.
func writeTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			t.Fatalf("close png: %v", cerr)
		}
	}()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

func TestReadImageInfo_PNGNoPreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "solid.png")
	writeTestPNG(t, path, 40, 30)

	// maxPreviewHeight <= 15 → no preview generated.
	info, err := readImageInfo(path, 10, 80)
	if err != nil {
		t.Fatalf("readImageInfo: %v", err)
	}
	if info.Format != "png" {
		t.Errorf("Format = %q, want png", info.Format)
	}
	if info.Width != 40 || info.Height != 30 {
		t.Errorf("dimensions = %dx%d, want 40x30", info.Width, info.Height)
	}
	if info.Size <= 0 {
		t.Errorf("Size = %d, want > 0", info.Size)
	}
	if info.Preview != nil {
		t.Error("Preview should be nil when maxPreviewHeight <= 15")
	}
}

func TestReadImageInfo_PNGWithPreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.png")
	writeTestPNG(t, path, 200, 100)

	info, err := readImageInfo(path, 40, 80)
	if err != nil {
		t.Fatalf("readImageInfo: %v", err)
	}
	if info.Preview == nil {
		t.Fatal("Preview is nil, want non-nil for large maxPreviewHeight")
	}
	if info.Preview.Width <= 0 || info.Preview.Height <= 0 {
		t.Errorf("preview dims = %dx%d, want positive", info.Preview.Width, info.Preview.Height)
	}
	if len(info.Preview.Rows) != info.Preview.Height {
		t.Errorf("len(Rows) = %d, want %d", len(info.Preview.Rows), info.Preview.Height)
	}
	// Each row should contain ANSI escape codes (true-color output).
	for i, row := range info.Preview.Rows {
		if !strings.Contains(row, "\x1b[") {
			t.Errorf("Rows[%d] missing ANSI escape: %q", i, row)
			break
		}
	}
}

func TestReadImageInfo_NonexistentFile(t *testing.T) {
	_, err := readImageInfo(filepath.Join(t.TempDir(), "missing.png"), 10, 80)
	if err == nil {
		t.Fatal("readImageInfo on missing file returned nil error")
	}
}

func TestReadImageInfo_NotAnImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake.png")
	// .png extension but the content isn't a valid image.
	if err := os.WriteFile(path, []byte("not an image at all"), 0600); err != nil {
		t.Fatalf("write fake png: %v", err)
	}

	_, err := readImageInfo(path, 40, 80)
	if err == nil {
		t.Fatal("readImageInfo on invalid image returned nil error")
	}
	if !strings.Contains(err.Error(), "not a valid image") {
		t.Errorf("error = %q, want wrapping 'not a valid image'", err.Error())
	}
}

func TestReadImageInfo_SVGByExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icon.svg")
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg width="64" height="64" viewBox="0 0 64 64" xmlns="http://www.w3.org/2000/svg">
  <rect width="64" height="64" fill="blue"/>
</svg>`
	if err := os.WriteFile(path, []byte(svg), 0600); err != nil {
		t.Fatalf("write svg: %v", err)
	}

	info, err := readImageInfo(path, 10, 80)
	if err != nil {
		t.Fatalf("readImageInfo: %v", err)
	}
	if info.Format != "svg" {
		t.Errorf("Format = %q, want svg (routed through readSVGInfo)", info.Format)
	}
	if info.Width != 64 || info.Height != 64 {
		t.Errorf("dimensions = %dx%d, want 64x64", info.Width, info.Height)
	}
}

func TestReadImageInfo_SVGByContentWithoutExtension(t *testing.T) {
	// File has no extension; content sniffing must route to SVG path.
	path := filepath.Join(t.TempDir(), "iconfile")
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg width="32" height="32" viewBox="0 0 32 32" xmlns="http://www.w3.org/2000/svg">
  <circle cx="16" cy="16" r="10" fill="red"/>
</svg>`
	if err := os.WriteFile(path, []byte(svg), 0600); err != nil {
		t.Fatalf("write svg: %v", err)
	}

	info, err := readImageInfo(path, 10, 80)
	if err != nil {
		t.Fatalf("readImageInfo: %v", err)
	}
	if info.Format != "svg" {
		t.Errorf("Format = %q, want svg (content-sniffed)", info.Format)
	}
	if info.Width != 32 || info.Height != 32 {
		t.Errorf("dimensions = %dx%d, want 32x32", info.Width, info.Height)
	}
}

func TestReadImageInfo_NoExtensionNonImageFallsThrough(t *testing.T) {
	// File has no extension, no SVG signature — should try to decode
	// as an image and fail with a "not a valid image" error.
	path := filepath.Join(t.TempDir(), "plainfile")
	if err := os.WriteFile(path, []byte("just some text, not an image"), 0600); err != nil {
		t.Fatalf("write plain: %v", err)
	}

	_, err := readImageInfo(path, 10, 80)
	if err == nil {
		t.Fatal("readImageInfo on non-image unextensioned file returned nil error")
	}
	if !strings.Contains(err.Error(), "not a valid image") {
		t.Errorf("error = %q, want 'not a valid image'", err.Error())
	}
}

func TestGenerateImagePreview_ShapeAndContent(t *testing.T) {
	// Build a tiny gradient image and verify the preview's shape and
	// that each emitted row carries ANSI escape sequences.
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 12), G: uint8(y * 12), B: 128, A: 255})
		}
	}

	preview := generateImagePreview(img, 40, 10)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	if preview.Height <= 0 {
		t.Errorf("preview.Height = %d, want > 0", preview.Height)
	}
	if preview.Width <= 0 {
		t.Errorf("preview.Width = %d, want > 0", preview.Width)
	}
	if len(preview.Rows) != preview.Height {
		t.Errorf("len(Rows) = %d, want %d", len(preview.Rows), preview.Height)
	}
	// Each row must contain the half-block character '▀' with a preceding
	// ANSI sequence.
	for i, row := range preview.Rows {
		if !bytes.Contains([]byte(row), []byte("\x1b[")) {
			t.Errorf("Rows[%d] has no ANSI escape: %q", i, row)
		}
		if !strings.Contains(row, "▀") {
			t.Errorf("Rows[%d] has no upper-half-block: %q", i, row)
		}
	}
}

func TestReadSVGInfo_UnreadablePath(t *testing.T) {
	// Pointing readSVGInfo at a directory makes os.ReadFile fail,
	// exercising the early-return error branch.
	dir := t.TempDir()
	if _, err := readSVGInfo(dir, 0, 30, 80); err == nil {
		t.Fatal("readSVGInfo on a directory returned nil error, want read error")
	}
}

func TestReadSVGInfo_LargeWidthConstrainedTriggersScaleDown(t *testing.T) {
	// A very wide SVG whose intrinsic dimensions exceed the preview's
	// character budget * 10 forces the scale-down branch, and the
	// width:height aspect ratio exceeds the preview ratio so the
	// width-constrained arm runs.
	path := filepath.Join(t.TempDir(), "wide.svg")
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg width="4000" height="1000" viewBox="0 0 4000 1000" xmlns="http://www.w3.org/2000/svg">
  <rect width="4000" height="1000" fill="green"/>
</svg>`
	if err := os.WriteFile(path, []byte(svg), 0600); err != nil {
		t.Fatalf("write svg: %v", err)
	}

	// availableWidth≈76 → *10=760 < 4000; aspect 4.0 > 76/26 ≈ 2.92
	// → width-constrained arm.
	info, err := readSVGInfo(path, 0, 30, 80)
	if err != nil {
		t.Fatalf("readSVGInfo: %v", err)
	}
	if info.Preview == nil {
		t.Fatal("Preview is nil, want non-nil")
	}
}

func TestReadSVGInfo_LargeHeightConstrainedTriggersScaleDown(t *testing.T) {
	// A very tall SVG whose aspect ratio is SMALLER than the preview's
	// ratio → height-constrained arm of the scale-down.
	path := filepath.Join(t.TempDir(), "tall.svg")
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg width="1000" height="4000" viewBox="0 0 1000 4000" xmlns="http://www.w3.org/2000/svg">
  <rect width="1000" height="4000" fill="purple"/>
</svg>`
	if err := os.WriteFile(path, []byte(svg), 0600); err != nil {
		t.Fatalf("write svg: %v", err)
	}

	info, err := readSVGInfo(path, 0, 30, 80)
	if err != nil {
		t.Fatalf("readSVGInfo: %v", err)
	}
	if info.Preview == nil {
		t.Fatal("Preview is nil, want non-nil")
	}
}

func TestRasterizeSVG_ClampsOversizedDimensions(t *testing.T) {
	// Parse any valid SVG into an icon, then ask rasterizeSVG for a
	// canvas bigger than maxSVGRasterDimension. The helper must clamp
	// silently and still return a valid image without error.
	svg := `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="black"/></svg>`
	icon, err := oksvg.ReadIconStream(bytes.NewReader([]byte(svg)))
	if err != nil {
		t.Fatalf("oksvg parse: %v", err)
	}

	img, err := rasterizeSVG(icon, maxSVGRasterDimension*4, maxSVGRasterDimension*4)
	if err != nil {
		t.Fatalf("rasterizeSVG: %v", err)
	}
	if img == nil {
		t.Fatal("rasterizeSVG returned nil image")
	}
	b := img.Bounds()
	if b.Dx() > maxSVGRasterDimension || b.Dy() > maxSVGRasterDimension {
		t.Errorf("rasterized bounds %dx%d exceed max %d",
			b.Dx(), b.Dy(), maxSVGRasterDimension)
	}
}

func TestExtractSVGDimensions(t *testing.T) {
	tests := []struct {
		name       string
		svg        string
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "width and height attributes",
			svg:        `<svg width="120" height="80"></svg>`,
			wantWidth:  120,
			wantHeight: 80,
		},
		{
			name:       "viewBox when width/height missing",
			svg:        `<svg viewBox="0 0 200 150" xmlns="http://www.w3.org/2000/svg"></svg>`,
			wantWidth:  200,
			wantHeight: 150,
		},
		{
			name:       "no dimensions falls back to defaults",
			svg:        `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			wantWidth:  defaultSVGDimension,
			wantHeight: defaultSVGDimension,
		},
		{
			name:       "non-numeric width falls back to default, height parsed",
			svg:        `<svg width="100%" height="200"></svg>`,
			wantWidth:  defaultSVGDimension,
			wantHeight: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := extractSVGDimensions(tt.svg)
			if w != tt.wantWidth || h != tt.wantHeight {
				t.Errorf("extractSVGDimensions() = (%d, %d), want (%d, %d)", w, h, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}
