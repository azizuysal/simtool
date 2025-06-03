package simulator

import (
	"os"
	"path/filepath"
	"strings"
	"regexp"
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

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "pure ASCII text",
			data:     []byte("Hello, World!\nThis is a test."),
			expected: true,
		},
		{
			name:     "UTF-8 text",
			data:     []byte("Hello, 世界! 🌍"),
			expected: false, // Current implementation only considers ASCII printable
		},
		{
			name:     "text with some control chars",
			data:     []byte("Line 1\nLine 2\tTabbed\rCarriage return"),
			expected: true,
		},
		{
			name:     "binary data",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
			expected: false,
		},
		{
			name:     "mostly binary with some text",
			data:     append([]byte("Hello"), []byte{0x00, 0x01, 0x02, 0x03, 0xFF}...),
			expected: false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: true,
		},
		{
			name:     "just newlines",
			data:     []byte("\n\n\n"),
			expected: true,
		},
		{
			name:     "high threshold of non-text",
			data:     func() []byte {
				// Create data that's just under 30% non-text
				data := make([]byte, 100)
				for i := 0; i < 70; i++ {
					data[i] = 'A'
				}
				for i := 70; i < 100; i++ {
					data[i] = 0x00
				}
				return data
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextContent(tt.data)
			if result != tt.expected {
				t.Errorf("isTextContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Match ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func TestGetSyntaxHighlightedLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		ext      string
		expected string
	}{
		{
			name:     "go file",
			line:     "func main() { fmt.Println(\"Hello\") }",
			ext:      ".go",
			expected: "func main() { fmt.Println(\"Hello\") }",
		},
		{
			name:     "javascript file",
			line:     "console.log('test');",
			ext:      ".js",
			expected: "console.log('test');",
		},
		{
			name:     "unknown extension",
			line:     "some text",
			ext:      ".xyz",
			expected: "some text",
		},
		{
			name:     "empty line",
			line:     "",
			ext:      ".go",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSyntaxHighlightedLine(tt.line, tt.ext)
			
			// For empty lines, expect no change
			if tt.line == "" {
				if result != tt.expected {
					t.Errorf("GetSyntaxHighlightedLine() = %v, want %v", result, tt.expected)
				}
				return
			}
			
			// For non-empty lines with known extensions, expect ANSI escape codes
			if tt.ext == ".go" || tt.ext == ".js" || tt.ext == ".py" || tt.ext == ".json" {
				// Check that syntax highlighting was applied (contains ANSI escape codes)
				if !strings.Contains(result, "\033[") && !strings.Contains(result, "[38;") {
					t.Errorf("Expected syntax highlighting for %s file, but got plain text: %v", tt.ext, result)
				}
				// Also check that the original content is preserved (minus ANSI codes)
				strippedResult := stripANSI(result)
				if strippedResult != tt.line {
					t.Errorf("Content changed after highlighting: got %v, want %v", strippedResult, tt.line)
				}
			} else {
				// For unknown extensions, expect plain text
				if result != tt.expected {
					t.Errorf("GetSyntaxHighlightedLine() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestDetectFileTypeWithContent(t *testing.T) {
	// Create temporary files with different content
	tmpDir := t.TempDir()

	// Create a text file with .bin extension to test content detection
	binTextFile := filepath.Join(tmpDir, "text.bin")
	os.WriteFile(binTextFile, []byte("This is actually a text file"), 0644)

	// Create a binary file with .txt extension
	txtBinaryFile := filepath.Join(tmpDir, "binary.txt")
	os.WriteFile(txtBinaryFile, []byte{0x00, 0x01, 0x02, 0x03, 0xFF}, 0644)

	// Create an empty file
	emptyFile := filepath.Join(tmpDir, "empty.dat")
	os.WriteFile(emptyFile, []byte{}, 0644)

	tests := []struct {
		name     string
		path     string
		expected FileType
	}{
		{
			name:     "text content with binary extension",
			path:     binTextFile,
			expected: FileTypeBinary, // DetectFileType checks extension first, then content
		},
		{
			name:     "binary content with text extension",
			path:     txtBinaryFile,
			expected: FileTypeBinary,
		},
		{
			name:     "empty file",
			path:     emptyFile,
			expected: FileTypeBinary, // Empty .dat files default to binary
		},
		{
			name:     "image by extension without file",
			path:     "/nonexistent/image.gif",
			expected: FileTypeImage,
		},
		{
			name:     "svg image",
			path:     "/nonexistent/diagram.svg",
			expected: FileTypeImage,
		},
		{
			name:     "webp image",
			path:     "/nonexistent/photo.webp",
			expected: FileTypeImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFileType(tt.path)
			if result != tt.expected {
				t.Errorf("DetectFileType(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}