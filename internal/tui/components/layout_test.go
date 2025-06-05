package components

import (
	"strings"
	"testing"
)

func TestNewLayout(t *testing.T) {
	layout := NewLayout(80, 24)
	if layout.Width != 80 {
		t.Errorf("Expected width 80, got %d", layout.Width)
	}
	if layout.Height != 24 {
		t.Errorf("Expected height 24, got %d", layout.Height)
	}
}

func TestLayoutRender(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		height  int
		title   string
		content string
		footer  string
		status  string
		checks  []string
	}{
		{
			name:    "basic layout",
			width:   80,
			height:  24,
			title:   "Test Title",
			content: "Test content here",
			footer:  "Press q to quit",
			status:  "",
			checks:  []string{"Test Title", "Test content here", "Press q to quit"},
		},
		{
			name:    "layout with status",
			width:   80,
			height:  24,
			title:   "Test Title",
			content: "Test content",
			footer:  "Footer",
			status:  "Loading...",
			checks:  []string{"Test Title", "Test content", "Footer", "Loading..."},
		},
		{
			name:    "narrow layout",
			width:   40,
			height:  20,
			title:   "Title",
			content: "Content",
			footer:  "Footer",
			status:  "",
			checks:  []string{"Title", "Content", "Footer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := NewLayout(tt.width, tt.height)
			result := layout.Render(tt.title, tt.content, tt.footer, tt.status)

			for _, check := range tt.checks {
				if !strings.Contains(result, check) {
					t.Errorf("Expected result to contain %q, but it didn't", check)
				}
			}

			// Check that result has proper structure
			lines := strings.Split(result, "\n")
			if len(lines) < 3 {
				t.Error("Layout should have at least 3 lines")
			}
		})
	}
}

func TestContentBox(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		header    string
		content   string
		hasHeader bool
		checks    []string
	}{
		{
			name:      "content without header",
			width:     60,
			height:    20,
			header:    "",
			content:   "Main content goes here",
			hasHeader: false,
			checks:    []string{"Main content goes here"},
		},
		{
			name:      "content with header",
			width:     60,
			height:    20,
			header:    "File Information",
			content:   "File contents",
			hasHeader: true,
			checks:    []string{"File Information", "─", "File contents"},
		},
		{
			name:      "multi-line content",
			width:     60,
			height:    20,
			header:    "",
			content:   "Line 1\nLine 2\nLine 3",
			hasHeader: false,
			checks:    []string{"Line 1", "Line 2", "Line 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewContentBox(tt.width, tt.height)
			result := cb.Render(tt.header, tt.content, tt.hasHeader)

			for _, check := range tt.checks {
				if !strings.Contains(result, check) {
					t.Errorf("Expected result to contain %q, but it didn't", check)
				}
			}
		})
	}
}

func TestLayoutCentering(t *testing.T) {
	layout := NewLayout(100, 24)
	
	// Test centering with various content widths
	testCases := []struct {
		content  string
		expected int // expected left padding
	}{
		{"Short", 47}, // (100-6)/2 = 47 for "Short" (5 chars)
		{"A much longer string that should still be centered", 25},
	}

	for _, tc := range testCases {
		t.Run(tc.content, func(t *testing.T) {
			centered := layout.centerContent(tc.content)
			lines := strings.Split(centered, "\n")
			
			// Count leading spaces
			leadingSpaces := 0
			for _, char := range lines[0] {
				if char == ' ' {
					leadingSpaces++
				} else {
					break
				}
			}

			// Allow for some variance in centering calculation
			if leadingSpaces < tc.expected-2 || leadingSpaces > tc.expected+2 {
				t.Errorf("Expected approximately %d leading spaces, got %d", tc.expected, leadingSpaces)
			}
		})
	}
}