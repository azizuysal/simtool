package tui

import (
	"testing"
	"simtool/internal/simulator"
)

func TestGetFilteredAndSearchedSimulators(t *testing.T) {
	tests := []struct {
		name           string
		simulators     []simulator.Item
		filterActive   bool
		searchQuery    string
		expectedCount  int
		expectedNames  []string
	}{
		{
			name: "no filter, no search",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 5},
				{Simulator: simulator.Simulator{Name: "iPhone 14", State: "Shutdown"}, Runtime: "iOS 16.0", AppCount: 0},
				{Simulator: simulator.Simulator{Name: "iPad Pro", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 3},
			},
			filterActive:  false,
			searchQuery:   "",
			expectedCount: 3,
			expectedNames: []string{"iPhone 15", "iPhone 14", "iPad Pro"},
		},
		{
			name: "filter active, no search",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 5},
				{Simulator: simulator.Simulator{Name: "iPhone 14", State: "Shutdown"}, Runtime: "iOS 16.0", AppCount: 0},
				{Simulator: simulator.Simulator{Name: "iPad Pro", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 3},
			},
			filterActive:  true,
			searchQuery:   "",
			expectedCount: 2,
			expectedNames: []string{"iPhone 15", "iPad Pro"},
		},
		{
			name: "no filter, search by name",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 5},
				{Simulator: simulator.Simulator{Name: "iPhone 14", State: "Shutdown"}, Runtime: "iOS 16.0", AppCount: 0},
				{Simulator: simulator.Simulator{Name: "iPad Pro", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 3},
			},
			filterActive:  false,
			searchQuery:   "iphone",
			expectedCount: 2,
			expectedNames: []string{"iPhone 15", "iPhone 14"},
		},
		{
			name: "filter and search combined",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 5},
				{Simulator: simulator.Simulator{Name: "iPhone 14", State: "Shutdown"}, Runtime: "iOS 16.0", AppCount: 0},
				{Simulator: simulator.Simulator{Name: "iPad Pro", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 3},
			},
			filterActive:  true,
			searchQuery:   "iphone",
			expectedCount: 1,
			expectedNames: []string{"iPhone 15"},
		},
		{
			name: "search by runtime",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 5},
				{Simulator: simulator.Simulator{Name: "iPhone 14", State: "Shutdown"}, Runtime: "iOS 16.0", AppCount: 0},
				{Simulator: simulator.Simulator{Name: "iPad Pro", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 3},
			},
			filterActive:  false,
			searchQuery:   "17.0",
			expectedCount: 2,
			expectedNames: []string{"iPhone 15", "iPad Pro"},
		},
		{
			name: "search by state",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 5},
				{Simulator: simulator.Simulator{Name: "iPhone 14", State: "Shutdown"}, Runtime: "iOS 16.0", AppCount: 0},
				{Simulator: simulator.Simulator{Name: "iPad Pro", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 3},
			},
			filterActive:  false,
			searchQuery:   "booted",
			expectedCount: 2,
			expectedNames: []string{"iPhone 15", "iPad Pro"},
		},
		{
			name: "case insensitive search",
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 5},
				{Simulator: simulator.Simulator{Name: "iPhone 14", State: "Shutdown"}, Runtime: "iOS 16.0", AppCount: 0},
				{Simulator: simulator.Simulator{Name: "iPad Pro", State: "Booted"}, Runtime: "iOS 17.0", AppCount: 3},
			},
			filterActive:  false,
			searchQuery:   "IPHONE",
			expectedCount: 2,
			expectedNames: []string{"iPhone 15", "iPhone 14"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				simulators:     tt.simulators,
				filterActive:   tt.filterActive,
				simSearchQuery: tt.searchQuery,
			}

			result := model.getFilteredAndSearchedSimulators()

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d simulators, got %d", tt.expectedCount, len(result))
			}

			for i, sim := range result {
				if i < len(tt.expectedNames) && sim.Name != tt.expectedNames[i] {
					t.Errorf("expected simulator %d to be %s, got %s", i, tt.expectedNames[i], sim.Name)
				}
			}
		})
	}
}

func TestGetFilteredAndSearchedApps(t *testing.T) {
	tests := []struct {
		name          string
		apps          []simulator.App
		searchQuery   string
		expectedCount int
		expectedNames []string
	}{
		{
			name: "no search",
			apps: []simulator.App{
				{Name: "Safari", BundleID: "com.apple.safari", Version: "1.0"},
				{Name: "Messages", BundleID: "com.apple.messages", Version: "2.0"},
				{Name: "Calendar", BundleID: "com.apple.calendar", Version: "1.5"},
			},
			searchQuery:   "",
			expectedCount: 3,
			expectedNames: []string{"Safari", "Messages", "Calendar"},
		},
		{
			name: "search by app name",
			apps: []simulator.App{
				{Name: "Safari", BundleID: "com.apple.safari", Version: "1.0"},
				{Name: "Messages", BundleID: "com.apple.messages", Version: "2.0"},
				{Name: "Calendar", BundleID: "com.apple.calendar", Version: "1.5"},
			},
			searchQuery:   "saf",
			expectedCount: 1,
			expectedNames: []string{"Safari"},
		},
		{
			name: "search by bundle ID",
			apps: []simulator.App{
				{Name: "Safari", BundleID: "com.apple.safari", Version: "1.0"},
				{Name: "Messages", BundleID: "com.apple.messages", Version: "2.0"},
				{Name: "Calendar", BundleID: "com.apple.calendar", Version: "1.5"},
			},
			searchQuery:   "messages",
			expectedCount: 1,
			expectedNames: []string{"Messages"},
		},
		{
			name: "search by version",
			apps: []simulator.App{
				{Name: "Safari", BundleID: "com.apple.safari", Version: "1.0"},
				{Name: "Messages", BundleID: "com.apple.messages", Version: "2.0"},
				{Name: "Calendar", BundleID: "com.apple.calendar", Version: "1.5"},
			},
			searchQuery:   "2.0",
			expectedCount: 1,
			expectedNames: []string{"Messages"},
		},
		{
			name: "partial match in bundle ID",
			apps: []simulator.App{
				{Name: "Safari", BundleID: "com.apple.safari", Version: "1.0"},
				{Name: "Messages", BundleID: "com.apple.messages", Version: "2.0"},
				{Name: "Calendar", BundleID: "com.apple.calendar", Version: "1.5"},
			},
			searchQuery:   "apple",
			expectedCount: 3,
			expectedNames: []string{"Safari", "Messages", "Calendar"},
		},
		{
			name: "case insensitive search",
			apps: []simulator.App{
				{Name: "Safari", BundleID: "com.apple.safari", Version: "1.0"},
				{Name: "Messages", BundleID: "com.apple.messages", Version: "2.0"},
				{Name: "Calendar", BundleID: "com.apple.calendar", Version: "1.5"},
			},
			searchQuery:   "SAFARI",
			expectedCount: 1,
			expectedNames: []string{"Safari"},
		},
		{
			name: "no matches",
			apps: []simulator.App{
				{Name: "Safari", BundleID: "com.apple.safari", Version: "1.0"},
				{Name: "Messages", BundleID: "com.apple.messages", Version: "2.0"},
				{Name: "Calendar", BundleID: "com.apple.calendar", Version: "1.5"},
			},
			searchQuery:   "chrome",
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				apps:           tt.apps,
				appSearchQuery: tt.searchQuery,
			}

			result := model.getFilteredAndSearchedApps()

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d apps, got %d", tt.expectedCount, len(result))
			}

			for i, app := range result {
				if i < len(tt.expectedNames) && app.Name != tt.expectedNames[i] {
					t.Errorf("expected app %d to be %s, got %s", i, tt.expectedNames[i], app.Name)
				}
			}
		})
	}
}