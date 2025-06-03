package simulator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestItemStateDisplay(t *testing.T) {
	tests := []struct {
		name     string
		item     Item
		expected string
	}{
		{
			name: "running simulator",
			item: Item{
				Simulator: Simulator{
					State: "Booted",
				},
			},
			expected: "Running",
		},
		{
			name: "shutdown simulator",
			item: Item{
				Simulator: Simulator{
					State: "Shutdown",
				},
			},
			expected: "Not Running",
		},
		{
			name: "unknown state",
			item: Item{
				Simulator: Simulator{
					State: "Unknown",
				},
			},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.StateDisplay()
			if result != tt.expected {
				t.Errorf("StateDisplay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestItemIsRunning(t *testing.T) {
	tests := []struct {
		name     string
		item     Item
		expected bool
	}{
		{
			name: "running simulator",
			item: Item{
				Simulator: Simulator{
					State: "Booted",
				},
			},
			expected: true,
		},
		{
			name: "shutdown simulator",
			item: Item{
				Simulator: Simulator{
					State: "Shutdown",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.IsRunning()
			if result != tt.expected {
				t.Errorf("IsRunning() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{
			name:     "zero bytes",
			size:     0,
			expected: "0 B",
		},
		{
			name:     "bytes",
			size:     512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			size:     1536,
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			size:     1048576,
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			size:     1073741824,
			expected: "1.0 GB",
		},
		{
			name:     "large value",
			size:     5368709120,
			expected: "5.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.size)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %v, want %v", tt.size, result, tt.expected)
			}
		})
	}
}

func TestFormatFileDate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		time     time.Time
		validate func(string) bool
	}{
		{
			name: "today's date",
			time: time.Now(),
			validate: func(s string) bool {
				return strings.Contains(s, "Today")
			},
		},
		{
			name: "this year but not today",
			time: time.Date(now.Year(), 1, 15, 9, 0, 0, 0, time.UTC),
			validate: func(s string) bool {
				return strings.Contains(s, "Jan 15") && !strings.Contains(s, ",")
			},
		},
		{
			name: "old date",
			time: time.Date(2020, 1, 15, 9, 0, 0, 0, time.UTC),
			validate: func(s string) bool {
				return s == "Jan 15, 2020"
			},
		},
		{
			name: "recent within week",
			time: now.AddDate(0, 0, -2),
			validate: func(s string) bool {
				// Should show day and time OR month day depending on logic
				return s != "" && (strings.Contains(s, ":") || strings.Contains(s, " "))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFileDate(tt.time)
			if !tt.validate(result) {
				t.Errorf("FormatFileDate() = %v, validation failed", result)
			}
		})
	}
}

func TestDetectFileType(t *testing.T) {
	// Create a temporary text file for testing
	tmpDir := t.TempDir()
	textFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(textFile, []byte("Hello, world!\nThis is a text file."), 0644)

	// Create a temporary binary file for testing
	binaryFile := filepath.Join(tmpDir, "test.bin")
	os.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}, 0644)

	tests := []struct {
		name     string
		path     string
		expected FileType
	}{
		{
			name:     "actual text file",
			path:     textFile,
			expected: FileTypeText,
		},
		{
			name:     "actual binary file",
			path:     binaryFile,
			expected: FileTypeBinary,
		},
		{
			name:     "image file - .png (by extension)",
			path:     "/nonexistent/image.png",
			expected: FileTypeImage,
		},
		{
			name:     "image file - .jpg (by extension)",
			path:     "/nonexistent/photo.jpg",
			expected: FileTypeImage,
		},
		{
			name:     "non-existent file defaults to binary",
			path:     "/nonexistent/file.txt",
			expected: FileTypeBinary,
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