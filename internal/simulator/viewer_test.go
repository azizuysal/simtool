package simulator

import (
	"strings"
	"testing"
)

func TestFormatHexDump(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		offset   int64
		wantLines int
		check    func(t *testing.T, lines []string)
	}{
		{
			name:   "simple hex dump",
			data:   []byte("Hello, World!"),
			offset: 0,
			wantLines: 1,
			check: func(t *testing.T, lines []string) {
				if len(lines) != 1 {
					t.Errorf("Expected 1 line, got %d", len(lines))
					return
				}
				// Check format: offset | hex | ascii
				if !strings.Contains(lines[0], "00000000") {
					t.Error("Line should start with offset 00000000")
				}
				if !strings.Contains(lines[0], "48 65 6c 6c 6f") {
					t.Error("Line should contain hex values for 'Hello'")
				}
				if !strings.Contains(lines[0], "Hello, World!") {
					t.Error("Line should contain ASCII representation")
				}
			},
		},
		{
			name:   "multiple lines",
			data:   []byte("This is a longer string that spans multiple lines in hex dump"),
			offset: 0,
			wantLines: 4,
			check: func(t *testing.T, lines []string) {
				if len(lines) != 4 {
					t.Errorf("Expected 4 lines, got %d", len(lines))
					return
				}
				// Check second line offset
				if !strings.Contains(lines[1], "00000010") {
					t.Error("Second line should start with offset 00000010")
				}
			},
		},
		{
			name:   "with offset",
			data:   []byte("Test data"),
			offset: 0x100,
			wantLines: 1,
			check: func(t *testing.T, lines []string) {
				if len(lines) != 1 {
					t.Errorf("Expected 1 line, got %d", len(lines))
					return
				}
				if !strings.Contains(lines[0], "00000100") {
					t.Error("Line should start with offset 00000100")
				}
			},
		},
		{
			name:   "binary data with non-printable chars",
			data:   []byte{0x00, 0x01, 0x02, 0x03, 0x41, 0x42, 0x43, 0xff},
			offset: 0,
			wantLines: 1,
			check: func(t *testing.T, lines []string) {
				if len(lines) != 1 {
					t.Errorf("Expected 1 line, got %d", len(lines))
					return
				}
				// Non-printable chars should be dots in ASCII section
				asciiPart := lines[0][60:] // ASCII section starts at column 60
				if !strings.Contains(asciiPart, "....ABC.") {
					t.Errorf("ASCII section should show dots for non-printable chars, got: %s", asciiPart)
				}
			},
		},
		{
			name:      "empty data",
			data:      []byte{},
			offset:    0,
			wantLines: 0,
		},
		{
			name:   "exact 16 bytes",
			data:   []byte("1234567890123456"),
			offset: 0,
			wantLines: 1,
			check: func(t *testing.T, lines []string) {
				if len(lines) != 1 {
					t.Errorf("Expected 1 line, got %d", len(lines))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := FormatHexDump(tt.data, tt.offset)
			
			if len(lines) != tt.wantLines {
				t.Errorf("FormatHexDump() returned %d lines, want %d", len(lines), tt.wantLines)
				for i, line := range lines {
					t.Logf("Line %d: %s", i, line)
				}
			}

			if tt.check != nil && len(lines) > 0 {
				tt.check(t, lines)
			}
		})
	}
}

func TestImagePreviewGeneration(t *testing.T) {
	// Test basic image info structure
	info := &ImageInfo{
		Format: "png",
		Width:  100,
		Height: 100,
		Size:   1024,
		Preview: &ImagePreview{
			Width:  50,
			Height: 25,
			Rows:   []string{"test row 1", "test row 2"},
		},
	}

	if info.Format != "png" {
		t.Errorf("Expected format 'png', got %s", info.Format)
	}

	if info.Preview.Width != 50 {
		t.Errorf("Expected preview width 50, got %d", info.Preview.Width)
	}

	if len(info.Preview.Rows) != 2 {
		t.Errorf("Expected 2 preview rows, got %d", len(info.Preview.Rows))
	}
}

func TestFileContentStructure(t *testing.T) {
	// Test text file content
	textContent := &FileContent{
		Type:       FileTypeText,
		Lines:      []string{"line 1", "line 2", "line 3"},
		TotalLines: 100,
	}

	if textContent.Type != FileTypeText {
		t.Errorf("Expected FileTypeText, got %v", textContent.Type)
	}

	if len(textContent.Lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(textContent.Lines))
	}

	// Test binary file content
	binaryContent := &FileContent{
		Type:         FileTypeBinary,
		BinaryData:   []byte{0x00, 0x01, 0x02, 0x03},
		BinaryOffset: 256,
		TotalSize:    1024,
	}

	if binaryContent.Type != FileTypeBinary {
		t.Errorf("Expected FileTypeBinary, got %v", binaryContent.Type)
	}

	if len(binaryContent.BinaryData) != 4 {
		t.Errorf("Expected 4 bytes, got %d", len(binaryContent.BinaryData))
	}

	if binaryContent.BinaryOffset != 256 {
		t.Errorf("Expected offset 256, got %d", binaryContent.BinaryOffset)
	}

	// Test image file content
	imageContent := &FileContent{
		Type: FileTypeImage,
		ImageInfo: &ImageInfo{
			Format: "jpeg",
			Width:  800,
			Height: 600,
			Size:   4096,
		},
	}

	if imageContent.Type != FileTypeImage {
		t.Errorf("Expected FileTypeImage, got %v", imageContent.Type)
	}

	if imageContent.ImageInfo.Width != 800 {
		t.Errorf("Expected width 800, got %d", imageContent.ImageInfo.Width)
	}
}