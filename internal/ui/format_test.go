package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFormatHeader(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		contains []string
	}{
		{
			name:     "standard header",
			text:     "Test Header",
			width:    80,
			contains: []string{"Test Header"},
		},
		{
			name:     "narrow width",
			text:     "Header",
			width:    30,
			contains: []string{"Header"},
		},
		{
			name:     "empty header",
			text:     "",
			width:    50,
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatHeader(tt.text, tt.width)
			
			// Check that the header is centered
			lines := strings.Split(result, "\n")
			if len(lines) > 1 && tt.text != "" {
				// The header is on the second line (first line is top margin)
				headerLine := lines[1]
				if !strings.Contains(headerLine, tt.text) && tt.text != "" {
					t.Errorf("Header should contain text '%s'", tt.text)
				}
			}
			
			// Check for expected content
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected header to contain '%s'", expected)
				}
			}
		})
	}
}

func TestFormatFooter(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		contentWidth int
		screenWidth  int
		checkCenter  bool
	}{
		{
			name:         "standard footer",
			text:         "Press q to quit",
			contentWidth: 60,
			screenWidth:  80,
			checkCenter:  true,
		},
		{
			name:         "long footer text",
			text:         "↑/k: up • ↓/j: down • →/l: enter • ←/h: back • q: quit",
			contentWidth: 70,
			screenWidth:  80,
			checkCenter:  true,
		},
		{
			name:         "narrow screen",
			text:         "Footer",
			contentWidth: 20,
			screenWidth:  30,
			checkCenter:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFooter(tt.text, tt.contentWidth, tt.screenWidth)
			
			if !strings.Contains(result, tt.text) {
				t.Errorf("Footer should contain text '%s'", tt.text)
			}
			
			// Check centering
			if tt.checkCenter && tt.screenWidth > lipgloss.Width(tt.text) {
				if !strings.HasPrefix(result, " ") && tt.screenWidth > len(tt.text)+10 {
					t.Log("Footer might not be properly centered")
				}
			}
		})
	}
}

func TestFormatScrollInfo(t *testing.T) {
	tests := []struct {
		name         string
		viewport     int
		itemsPerPage int
		totalItems   int
		expected     string
	}{
		{
			name:         "first page",
			viewport:     0,
			itemsPerPage: 10,
			totalItems:   50,
			expected:     " (1-10 of 50) ↓",
		},
		{
			name:         "middle page",
			viewport:     20,
			itemsPerPage: 10,
			totalItems:   50,
			expected:     " (21-30 of 50) ↑↓",
		},
		{
			name:         "last page partial",
			viewport:     40,
			itemsPerPage: 10,
			totalItems:   45,
			expected:     " (41-45 of 45) ↑",
		},
		{
			name:         "single item",
			viewport:     0,
			itemsPerPage: 10,
			totalItems:   1,
			expected:     "",
		},
		{
			name:         "empty list",
			viewport:     0,
			itemsPerPage: 10,
			totalItems:   0,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatScrollInfo(tt.viewport, tt.itemsPerPage, tt.totalItems)
			if result != tt.expected {
				t.Errorf("FormatScrollInfo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPadLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		width    int
		expected int // expected length
	}{
		{
			name:     "short line",
			line:     "Hello",
			width:    20,
			expected: 20,
		},
		{
			name:     "exact width",
			line:     "12345",
			width:    5,
			expected: 5,
		},
		{
			name:     "line longer than width",
			line:     "This is a very long line",
			width:    10,
			expected: 24, // Original length, not truncated
		},
		{
			name:     "empty line",
			line:     "",
			width:    15,
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLine(tt.line, tt.width)
			if len(result) != tt.expected {
				t.Errorf("PadLine() length = %d, want %d", len(result), tt.expected)
			}
			
			// Check that padding is spaces
			if len(tt.line) < tt.width {
				expectedPadding := tt.width - len(tt.line)
				actualPadding := strings.Count(result[len(tt.line):], " ")
				if actualPadding != expectedPadding {
					t.Errorf("Expected %d spaces for padding, got %d", expectedPadding, actualPadding)
				}
			}
		})
	}
}