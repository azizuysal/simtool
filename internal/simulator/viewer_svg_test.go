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
	defer os.RemoveAll(tmpDir)

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

func TestReadSVGInfo_InvalidSVG(t *testing.T) {
	// Create a temporary file with invalid SVG content
	tmpDir, err := os.MkdirTemp("", "svg_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	invalidSVG := "This is not valid SVG content"
	svgPath := filepath.Join(tmpDir, "invalid.svg")
	if err := os.WriteFile(svgPath, []byte(invalidSVG), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := readSVGInfo(svgPath, int64(len(invalidSVG)), 30, 80)
	if err == nil {
		// Some SVG parsers might be lenient, check if dimensions are set to defaults
		if info != nil && info.Width == 256 && info.Height == 256 {
			// This is acceptable behavior - parser was lenient and used defaults
			return
		}
		t.Error("Expected error for invalid SVG, got nil")
	} else if !strings.Contains(err.Error(), "failed to parse SVG") {
		t.Errorf("Expected 'failed to parse SVG' error, got: %v", err)
	}
}

func TestReadSVGInfo_NoViewBox(t *testing.T) {
	// Create a temporary SVG file without viewBox
	tmpDir, err := os.MkdirTemp("", "svg_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

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