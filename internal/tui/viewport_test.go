package tui

import (
	"testing"
)

func TestCalculateItemsPerScreen(t *testing.T) {
	tests := []struct {
		name           string
		terminalHeight int
		expected       int
	}{
		{
			name:           "standard terminal height",
			terminalHeight: 30,
			expected:       7, // (30 - 8) / 3 = 7.33, rounded down to 7
		},
		{
			name:           "small terminal",
			terminalHeight: 15,
			expected:       2, // (15 - 8) / 3 = 2.33, rounded down to 2
		},
		{
			name:           "very small terminal",
			terminalHeight: 10,
			expected:       1, // (10 - 8) / 3 = 0.66, but minimum is 1
		},
		{
			name:           "large terminal",
			terminalHeight: 60,
			expected:       17, // (60 - 8) / 3 = 17.33, rounded down to 17
		},
		{
			name:           "edge case - exactly fits",
			terminalHeight: 23,
			expected:       5, // (23 - 8) / 3 = 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateItemsPerScreen(tt.terminalHeight)
			if result != tt.expected {
				t.Errorf("CalculateItemsPerScreen(%d) = %d, want %d", 
					tt.terminalHeight, result, tt.expected)
			}
		})
	}
}

func TestCalculateSimulatorViewport(t *testing.T) {
	tests := []struct {
		name              string
		currentViewport   int
		currentCursor     int
		totalItems        int
		terminalHeight    int
		expectedViewport  int
	}{
		{
			name:              "cursor in view - no scroll",
			currentViewport:   0,
			currentCursor:     2,
			totalItems:        20,
			terminalHeight:    30,
			expectedViewport:  0,
		},
		{
			name:              "cursor below view - scroll down",
			currentViewport:   0,
			currentCursor:     8,
			totalItems:        20,
			terminalHeight:    30,
			expectedViewport:  2, // Cursor at 8, items per screen 7, so viewport should be 2 (8 - 7 + 1)
		},
		{
			name:              "cursor above view - scroll up",
			currentViewport:   5,
			currentCursor:     3,
			totalItems:        20,
			terminalHeight:    30,
			expectedViewport:  3,
		},
		{
			name:              "at bottom of list",
			currentViewport:   10,
			currentCursor:     19,
			totalItems:        20,
			terminalHeight:    30,
			expectedViewport:  13, // 20 - 7 = 13
		},
		{
			name:              "single item list",
			currentViewport:   0,
			currentCursor:     0,
			totalItems:        1,
			terminalHeight:    30,
			expectedViewport:  0,
		},
		{
			name:              "viewport exceeds max",
			currentViewport:   15,
			currentCursor:     19,
			totalItems:        20,
			terminalHeight:    30,
			expectedViewport:  13, // Should clamp to max viewport
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSimulatorViewport(
				tt.currentViewport,
				tt.currentCursor,
				tt.totalItems,
				tt.terminalHeight,
			)
			if result != tt.expectedViewport {
				t.Errorf("CalculateSimulatorViewport() = %d, want %d", 
					result, tt.expectedViewport)
			}
		})
	}
}

func TestCalculateFileListViewport(t *testing.T) {
	tests := []struct {
		name              string
		currentViewport   int
		currentCursor     int
		totalItems        int
		terminalHeight    int
		hasHeader         bool
		hasBreadcrumbs    bool
		expectedViewport  int
	}{
		{
			name:              "with header and breadcrumbs",
			currentViewport:   0,
			currentCursor:     5,
			totalItems:        20,
			terminalHeight:    30,
			hasHeader:         true,
			hasBreadcrumbs:    true,
			expectedViewport:  1, // Fewer items visible due to header space
		},
		{
			name:              "with header only",
			currentViewport:   0,
			currentCursor:     5,
			totalItems:        20,
			terminalHeight:    30,
			hasHeader:         true,
			hasBreadcrumbs:    false,
			expectedViewport:  0, // More space available
		},
		{
			name:              "small terminal with header",
			currentViewport:   0,
			currentCursor:     2,
			totalItems:        10,
			terminalHeight:    20,
			hasHeader:         true,
			hasBreadcrumbs:    true,
			expectedViewport:  1,
		},
		{
			name:              "cursor at bottom",
			currentViewport:   5,
			currentCursor:     19,
			totalItems:        20,
			terminalHeight:    30,
			hasHeader:         true,
			hasBreadcrumbs:    false,
			expectedViewport:  15, // Adjusted for available items
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate header lines
			headerLines := 6 // Base header lines
			if tt.hasBreadcrumbs {
				headerLines += 2
			}
			
			result := CalculateFileListViewport(
				tt.currentViewport,
				tt.currentCursor,
				tt.totalItems,
				tt.terminalHeight,
				headerLines,
			)
			
			// Verify result is within valid range
			availableHeight := tt.terminalHeight - 8 - headerLines
			actualFileItems := availableHeight / 3
			if actualFileItems < 1 {
				actualFileItems = 1
			}
			
			maxViewport := tt.totalItems - actualFileItems
			if maxViewport < 0 {
				maxViewport = 0
			}
			
			if result > maxViewport {
				t.Errorf("CalculateFileListViewport() = %d exceeds max viewport %d", 
					result, maxViewport)
			}
			
			// Verify cursor is visible
			if tt.currentCursor < result || tt.currentCursor >= result + actualFileItems {
				t.Errorf("Cursor %d not visible in viewport %d-%d", 
					tt.currentCursor, result, result + actualFileItems - 1)
			}
		})
	}
}