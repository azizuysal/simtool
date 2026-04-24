package simulator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFileType_SVG(t *testing.T) {
	tests := []struct {
		filename string
		expected FileType
	}{
		{"icon.svg", FileTypeImage},
		{"logo.SVG", FileTypeImage},
		{"image.png", FileTypeImage},
		{"document.pdf", FileTypeBinary},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectFileType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectFileType(%s) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsImageFile_SVG(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"icon.svg", true},
		{"logo.SVG", true},
		{"image.png", true},
		{"image.jpg", true},
		{"document.txt", false},
		{"script.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsImageFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsImageFile(%s) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestReadSVGInfo(t *testing.T) {
	// Create a temporary SVG file for testing
	tmpDir, err := os.MkdirTemp("", "svg_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	svgContent := `<?xml version="1.0" encoding="UTF-8"?>
<svg width="100" height="100" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
  <circle cx="50" cy="50" r="40" fill="red"/>
</svg>`

	svgPath := filepath.Join(tmpDir, "test.svg")
	if err := os.WriteFile(svgPath, []byte(svgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test reading SVG info without preview
	info, err := readSVGInfo(svgPath, int64(len(svgContent)), 10, 80)
	if err != nil {
		t.Fatalf("readSVGInfo failed: %v", err)
	}

	if info.Format != "svg" {
		t.Errorf("Format = %s, want svg", info.Format)
	}
	if info.Width != 100 {
		t.Errorf("Width = %d, want 100", info.Width)
	}
	if info.Height != 100 {
		t.Errorf("Height = %d, want 100", info.Height)
	}
	if info.Size != int64(len(svgContent)) {
		t.Errorf("Size = %d, want %d", info.Size, len(svgContent))
	}
	if info.Preview != nil {
		t.Error("Preview should be nil for small maxPreviewHeight")
	}

	// Test with preview generation
	info, err = readSVGInfo(svgPath, int64(len(svgContent)), 30, 80)
	if err != nil {
		t.Fatalf("readSVGInfo with preview failed: %v", err)
	}

	if info.Preview == nil {
		t.Error("Preview should not be nil for large maxPreviewHeight")
	} else {
		if info.Preview.Height <= 0 {
			t.Error("Preview height should be greater than 0")
		}
		if info.Preview.Width <= 0 {
			t.Error("Preview width should be greater than 0")
		}
		if len(info.Preview.Rows) != info.Preview.Height {
			t.Errorf("Preview rows count = %d, want %d", len(info.Preview.Rows), info.Preview.Height)
		}
	}
}

func TestReadSVGInfo_MalformedXMLSurfacesErrorInPreview(t *testing.T) {
	// readSVGInfo's contract: it always returns (info, nil) and surfaces
	// parse/render failures inside info.Preview.Rows so the TUI can show
	// a user-visible error message instead of aborting the file view.
	//
	// oksvg is extremely lenient — arbitrary prose or non-svg XML parses
	// into an empty (but valid) icon, so to exercise the error branch
	// we need genuinely malformed XML.
	tmpDir, err := os.MkdirTemp("", "svg_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	malformed := `<svg><<<`
	svgPath := filepath.Join(tmpDir, "malformed.svg")
	if err := os.WriteFile(svgPath, []byte(malformed), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := readSVGInfo(svgPath, int64(len(malformed)), 30, 80)
	if err != nil {
		t.Fatalf("readSVGInfo returned error %v, want nil (failures go in the preview)", err)
	}
	if info == nil {
		t.Fatal("info is nil, want non-nil with error message in preview")
	}
	if info.Width != defaultSVGDimension || info.Height != defaultSVGDimension {
		t.Errorf("dimensions = %dx%d, want %[3]dx%[3]d (defaults)", info.Width, info.Height, defaultSVGDimension)
	}
	if info.Preview == nil {
		t.Fatal("Preview is nil, want a preview carrying the error message")
	}
	if len(info.Preview.Rows) != 1 {
		t.Fatalf("len(Preview.Rows) = %d, want 1 (single error line), got rows=%v", len(info.Preview.Rows), info.Preview.Rows)
	}
	row := info.Preview.Rows[0]
	if !strings.Contains(row, "SVG parsing error") && !strings.Contains(row, "SVG rendering error") {
		t.Errorf("Preview.Rows[0] = %q, want it to contain 'SVG parsing error' or 'SVG rendering error'", row)
	}
}

func TestReadSVGInfo_NoViewBox(t *testing.T) {
	// Create a temporary SVG file without viewBox
	tmpDir, err := os.MkdirTemp("", "svg_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	svgContent := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg">
  <circle cx="50" cy="50" r="40" fill="blue"/>
</svg>`

	svgPath := filepath.Join(tmpDir, "no_viewbox.svg")
	if err := os.WriteFile(svgPath, []byte(svgContent), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := readSVGInfo(svgPath, int64(len(svgContent)), 30, 80)
	if err != nil {
		t.Fatalf("readSVGInfo failed: %v", err)
	}

	// Should use default dimensions when viewBox is missing
	if info.Width != 256 {
		t.Errorf("Width = %d, want 256 (default)", info.Width)
	}
	if info.Height != 256 {
		t.Errorf("Height = %d, want 256 (default)", info.Height)
	}
}
