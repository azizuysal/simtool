package components

import (
	"strings"
	"testing"
	"time"

	"simtool/internal/config"
	"simtool/internal/simulator"
)

func TestAllAppsListView(t *testing.T) {
	// Create test apps
	testApps := []simulator.App{
		{
			Name:          "Test App 1",
			BundleID:      "com.test.app1",
			Version:       "1.0.0",
			Size:          1024 * 1024, // 1 MB
			SimulatorName: "iPhone 14",
			ModTime:       time.Now().Add(-1 * time.Hour),
		},
		{
			Name:          "Test App 2",
			BundleID:      "com.test.app2",
			Version:       "2.0.0",
			Size:          2048 * 1024, // 2 MB
			SimulatorName: "iPhone 15",
			ModTime:       time.Now().Add(-24 * time.Hour),
		},
	}

	keys := config.DefaultKeys()

	tests := []struct {
		name         string
		allApps      []simulator.App
		cursor       int
		viewport     int
		width        int
		height       int
		searchMode   bool
		searchQuery  string
		loading      bool
		err          error
		expectInView []string
		notInView    []string
	}{
		{
			name:     "normal view with apps",
			allApps:  testApps,
			cursor:   0,
			viewport: 0,
			width:    80,
			height:   24,
			expectInView: []string{
				"All Apps (2)",
				"Test App 1",
				"com.test.app1",
				"iPhone 14",
			},
		},
		{
			name:     "loading state",
			loading:  true,
			width:    80,
			height:   24,
			expectInView: []string{
				"Loading all apps...",
			},
			notInView: []string{
				"Test App 1",
			},
		},
		{
			name:    "error state",
			err:     simulator.ErrSimulatorNotFound,
			width:   80,
			height:  24,
			expectInView: []string{
				"Error loading apps:",
			},
		},
		{
			name:     "empty apps list",
			allApps:  []simulator.App{},
			width:    80,
			height:   24,
			expectInView: []string{
				"No apps installed on any simulator",
			},
		},
		{
			name:        "search mode with results",
			allApps:     testApps,
			searchMode:  true,
			searchQuery: "app1",
			cursor:      0,
			viewport:    0,
			width:       80,
			height:      24,
			expectInView: []string{
				"All Apps (1 of 2)",
				"Search: app1",
				"Test App 1",
			},
			notInView: []string{
				"Test App 2",
			},
		},
		{
			name:        "search mode no results",
			allApps:     testApps,
			searchMode:  true,
			searchQuery: "nonexistent",
			width:       80,
			height:      24,
			expectInView: []string{
				"No apps match your search",
			},
		},
		{
			name:        "search by simulator name",
			allApps:     testApps,
			searchMode:  true,
			searchQuery: "iPhone 15",
			cursor:      0,
			viewport:    0,
			width:       80,
			height:      24,
			expectInView: []string{
				"Test App 2",
				"iPhone 15",
			},
			notInView: []string{
				"Test App 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AllAppsListView(
				tt.allApps,
				tt.cursor,
				tt.viewport,
				tt.width,
				tt.height,
				tt.searchMode,
				tt.searchQuery,
				tt.loading,
				tt.err,
				&keys,
			)

			// Check expected strings are in view
			for _, expected := range tt.expectInView {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected %q in view, but not found", expected)
				}
			}

			// Check strings that should not be in view
			for _, notExpected := range tt.notInView {
				if strings.Contains(result, notExpected) {
					t.Errorf("Did not expect %q in view, but found it", notExpected)
				}
			}
		})
	}
}

func TestFilterAllApps(t *testing.T) {
	testApps := []simulator.App{
		{
			Name:          "Calculator",
			BundleID:      "com.apple.calculator",
			Version:       "1.0",
			SimulatorName: "iPhone 14",
		},
		{
			Name:          "Notes",
			BundleID:      "com.apple.notes",
			Version:       "2.0",
			SimulatorName: "iPhone 15",
		},
		{
			Name:          "TestApp",
			BundleID:      "com.test.app",
			Version:       "1.0.0",
			SimulatorName: "iPad Air",
		},
	}

	tests := []struct {
		name          string
		apps          []simulator.App
		query         string
		expectedCount int
		expectedApps  []string
	}{
		{
			name:          "empty query returns all",
			apps:          testApps,
			query:         "",
			expectedCount: 3,
		},
		{
			name:          "search by name",
			apps:          testApps,
			query:         "calc",
			expectedCount: 1,
			expectedApps:  []string{"Calculator"},
		},
		{
			name:          "search by bundle ID",
			apps:          testApps,
			query:         "apple",
			expectedCount: 2,
			expectedApps:  []string{"Calculator", "Notes"},
		},
		{
			name:          "search by version",
			apps:          testApps,
			query:         "1.0",
			expectedCount: 2,
			expectedApps:  []string{"Calculator", "TestApp"},
		},
		{
			name:          "search by simulator name",
			apps:          testApps,
			query:         "ipad",
			expectedCount: 1,
			expectedApps:  []string{"TestApp"},
		},
		{
			name:          "case insensitive search",
			apps:          testApps,
			query:         "IPHONE",
			expectedCount: 2,
			expectedApps:  []string{"Calculator", "Notes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterAllApps(tt.apps, tt.query)

			if len(result) != tt.expectedCount {
				t.Errorf("filterAllApps() returned %d apps, expected %d", len(result), tt.expectedCount)
			}

			// Check specific apps if provided
			if tt.expectedApps != nil {
				for _, expectedApp := range tt.expectedApps {
					found := false
					for _, app := range result {
						if app.Name == expectedApp {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected app %q not found in filtered results", expectedApp)
					}
				}
			}
		})
	}
}

func TestBuildAllAppsFooter(t *testing.T) {
	keys := config.DefaultKeys()

	tests := []struct {
		name             string
		searchMode       bool
		appCount         int
		viewport         int
		itemsPerScreen   int
		expectContains   []string
		notExpectContains []string
	}{
		{
			name:       "normal mode",
			searchMode: false,
			appCount:   10,
			viewport:   0,
			itemsPerScreen: 5,
			expectContains: []string{
				"up",
				"down",
				"files",
				"open in Finder",
				"search",
				"quit",
			},
			notExpectContains: []string{
				"Type to search",
				"cancel",
			},
		},
		{
			name:       "search mode",
			searchMode: true,
			appCount:   10,
			viewport:   0,
			itemsPerScreen: 5,
			expectContains: []string{
				"Type to search",
				"navigate",
				"select",
			},
			notExpectContains: []string{
				"open in Finder",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAllAppsFooter(tt.searchMode, tt.appCount, &keys, tt.viewport, tt.itemsPerScreen)

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected footer to contain %q, but it didn't", expected)
				}
			}

			for _, notExpected := range tt.notExpectContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("Did not expect footer to contain %q, but it did", notExpected)
				}
			}
		})
	}
}