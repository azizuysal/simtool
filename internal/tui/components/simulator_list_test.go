package components

import (
	"strings"
	"testing"

	"simtool/internal/simulator"
)

func TestNewSimulatorList(t *testing.T) {
	sl := NewSimulatorList(80, 24)
	if sl.Width != 80 {
		t.Errorf("Expected width 80, got %d", sl.Width)
	}
	if sl.Height != 24 {
		t.Errorf("Expected height 24, got %d", sl.Height)
	}
}

func TestSimulatorListRender(t *testing.T) {
	tests := []struct {
		name       string
		simulators []simulator.Item
		cursor     int
		viewport   int
		expected   []string
		notExpected []string
	}{
		{
			name:       "empty list",
			simulators: []simulator.Item{},
			cursor:     0,
			viewport:   0,
			expected:   []string{"No simulators found"},
		},
		{
			name: "single simulator",
			simulators: []simulator.Item{
				{
					Simulator: simulator.Simulator{
						Name:  "iPhone 15",
						State: "Booted",
					},
					Runtime:  "iOS 17.0",
					AppCount: 5,
				},
			},
			cursor:   0,
			viewport: 0,
			expected: []string{"iPhone 15", "iOS 17.0", "Running", "5 apps"},
		},
		{
			name: "multiple simulators with selection",
			simulators: []simulator.Item{
				{
					Simulator: simulator.Simulator{
						Name:  "iPhone 14",
						State: "Shutdown",
					},
					Runtime:  "iOS 16.0",
					AppCount: 0,
				},
				{
					Simulator: simulator.Simulator{
						Name:  "iPhone 15",
						State: "Booted",
					},
					Runtime:  "iOS 17.0",
					AppCount: 3,
				},
			},
			cursor:   1,
			viewport: 0,
			expected: []string{"iPhone 14", "iPhone 15", "▶", "3 apps"},
		},
		{
			name: "simulator with 1 app",
			simulators: []simulator.Item{
				{
					Simulator: simulator.Simulator{
						Name:  "iPad Pro",
						State: "Shutdown",
					},
					Runtime:  "iPadOS 17.0",
					AppCount: 1,
				},
			},
			cursor:   0,
			viewport: 0,
			expected: []string{"iPad Pro", "1 app"}, // Should be "1 app" not "1 apps"
			notExpected: []string{"1 apps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := NewSimulatorList(80, 24)
			sl.Update(tt.simulators, tt.cursor, tt.viewport, false, false, "")
			
			result := sl.Render()

			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected result to contain %q, but it didn't\nGot: %s", exp, result)
				}
			}

			for _, notExp := range tt.notExpected {
				if strings.Contains(result, notExp) {
					t.Errorf("Expected result NOT to contain %q, but it did", notExp)
				}
			}
		})
	}
}

func TestSimulatorListGetTitle(t *testing.T) {
	sl := NewSimulatorList(80, 24)

	tests := []struct {
		name         string
		simulators   []simulator.Item
		filterActive bool
		searchQuery  string
		totalCount   int
		expected     string
	}{
		{
			name: "no filter or search",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15"}},
				{Simulator: simulator.Simulator{Name: "iPhone 14"}},
			},
			filterActive: false,
			searchQuery:  "",
			totalCount:   2,
			expected:     "iOS Simulators (2)",
		},
		{
			name: "with filter active",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15"}, AppCount: 1},
			},
			filterActive: true,
			searchQuery:  "",
			totalCount:   3,
			expected:     "iOS Simulators (1 of 3)",
		},
		{
			name: "with search query",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15"}},
			},
			filterActive: false,
			searchQuery:  "iPhone",
			totalCount:   5,
			expected:     "iOS Simulators (1 of 5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl.Update(tt.simulators, 0, 0, tt.filterActive, false, tt.searchQuery)
			result := sl.GetTitle(tt.totalCount)
			if result != tt.expected {
				t.Errorf("Expected title %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSimulatorListGetFooter(t *testing.T) {
	sl := NewSimulatorList(80, 24)

	tests := []struct {
		name       string
		searchMode bool
		expected   string
	}{
		{
			name:       "normal mode",
			searchMode: false,
			expected:   "↑/k: up • ↓/j: down • →/l: apps • space: run • f: filter • /: search • q: quit",
		},
		{
			name:       "search mode",
			searchMode: true,
			expected:   "ESC: exit search • ↑/↓: navigate • →/Enter: select",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl.Update([]simulator.Item{}, 0, 0, false, tt.searchMode, "")
			result := sl.GetFooter()
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected footer to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSimulatorListGetStatus(t *testing.T) {
	sl := NewSimulatorList(80, 24)

	tests := []struct {
		name         string
		searchMode   bool
		searchQuery  string
		filterActive bool
		expected     string
	}{
		{
			name:         "no status",
			searchMode:   false,
			searchQuery:  "",
			filterActive: false,
			expected:     "",
		},
		{
			name:         "search mode with query",
			searchMode:   true,
			searchQuery:  "iPhone",
			filterActive: false,
			expected:     "Search: iPhone",
		},
		{
			name:         "search mode without query",
			searchMode:   true,
			searchQuery:  "",
			filterActive: false,
			expected:     "Search: (type to filter)",
		},
		{
			name:         "filter active",
			searchMode:   false,
			searchQuery:  "",
			filterActive: true,
			expected:     "Filter: Showing only simulators with apps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl.Update([]simulator.Item{}, 0, 0, tt.filterActive, tt.searchMode, tt.searchQuery)
			result := sl.GetStatus()
			if tt.expected == "" && result != "" {
				t.Errorf("Expected empty status, got %q", result)
			} else if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("Expected status to contain %q, got %q", tt.expected, result)
			}
		})
	}
}