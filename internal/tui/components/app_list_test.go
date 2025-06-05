package components

import (
	"strings"
	"testing"

	"simtool/internal/simulator"
)

func TestNewAppList(t *testing.T) {
	al := NewAppList(80, 24)
	if al.Width != 80 {
		t.Errorf("Expected width 80, got %d", al.Width)
	}
	if al.Height != 24 {
		t.Errorf("Expected height 24, got %d", al.Height)
	}
}

func TestAppListRender(t *testing.T) {
	tests := []struct {
		name        string
		apps        []simulator.App
		cursor      int
		searchQuery string
		expected    []string
	}{
		{
			name:        "empty list",
			apps:        []simulator.App{},
			cursor:      0,
			searchQuery: "",
			expected:    []string{"No apps installed"},
		},
		{
			name:        "empty list with search",
			apps:        []simulator.App{},
			cursor:      0,
			searchQuery: "test",
			expected:    []string{"No apps match your search"},
		},
		{
			name: "single app",
			apps: []simulator.App{
				{
					Name:     "TestApp",
					BundleID: "com.test.app",
					Version:  "1.0",
					Size:     1024 * 1024, // 1 MB
				},
			},
			cursor:      0,
			searchQuery: "",
			expected:    []string{"TestApp", "com.test.app", "v1.0", "1.0 MB"},
		},
		{
			name: "multiple apps with selection",
			apps: []simulator.App{
				{
					Name:     "App1",
					BundleID: "com.test.app1",
					Version:  "1.0",
					Size:     1024,
				},
				{
					Name:     "App2",
					BundleID: "com.test.app2",
					Version:  "2.0",
					Size:     2048,
				},
			},
			cursor:      1,
			searchQuery: "",
			expected:    []string{"App1", "App2", "▶", "com.test.app2"},
		},
		{
			name: "app without version",
			apps: []simulator.App{
				{
					Name:     "NoVersionApp",
					BundleID: "com.test.noversion",
					Version:  "",
					Size:     512,
				},
			},
			cursor:      0,
			searchQuery: "",
			expected:    []string{"NoVersionApp", "com.test.noversion", "512 B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			al := NewAppList(80, 24)
			al.Update(tt.apps, tt.cursor, 0, false, tt.searchQuery, "iPhone 15")
			
			result := al.Render()

			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected result to contain %q, but it didn't\nGot: %s", exp, result)
				}
			}
		})
	}
}

func TestAppListGetTitle(t *testing.T) {
	al := NewAppList(80, 24)

	tests := []struct {
		name        string
		apps        []simulator.App
		searchQuery string
		simName     string
		totalCount  int
		expected    string
	}{
		{
			name: "no search",
			apps: []simulator.App{
				{Name: "App1"},
				{Name: "App2"},
			},
			searchQuery: "",
			simName:     "iPhone 15",
			totalCount:  2,
			expected:    "iPhone 15 Apps (2)",
		},
		{
			name: "with search",
			apps: []simulator.App{
				{Name: "TestApp"},
			},
			searchQuery: "test",
			simName:     "iPhone 15",
			totalCount:  5,
			expected:    "iPhone 15 Apps (1 of 5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			al.Update(tt.apps, 0, 0, false, tt.searchQuery, tt.simName)
			result := al.GetTitle(tt.totalCount)
			if result != tt.expected {
				t.Errorf("Expected title %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAppListGetFooter(t *testing.T) {
	al := NewAppList(80, 24)

	tests := []struct {
		name       string
		searchMode bool
		expected   string
	}{
		{
			name:       "normal mode",
			searchMode: false,
			expected:   "↑/k: up • ↓/j: down • →/l: files • space: open in Finder • /: search • ←/h: back • q: quit",
		},
		{
			name:       "search mode",
			searchMode: true,
			expected:   "ESC: exit search • ↑/↓: navigate • →/Enter: select",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			al.Update([]simulator.App{}, 0, 0, tt.searchMode, "", "iPhone 15")
			result := al.GetFooter()
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected footer to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAppListGetStatus(t *testing.T) {
	al := NewAppList(80, 24)

	tests := []struct {
		name        string
		searchMode  bool
		searchQuery string
		expected    string
	}{
		{
			name:        "no status",
			searchMode:  false,
			searchQuery: "",
			expected:    "",
		},
		{
			name:        "search mode with query",
			searchMode:  true,
			searchQuery: "test",
			expected:    "Search: test",
		},
		{
			name:        "search mode without query",
			searchMode:  true,
			searchQuery: "",
			expected:    "Search: (type to filter)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			al.Update([]simulator.App{}, 0, 0, tt.searchMode, tt.searchQuery, "iPhone 15")
			result := al.GetStatus()
			if tt.expected == "" && result != "" {
				t.Errorf("Expected empty status, got %q", result)
			} else if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("Expected status to contain %q, got %q", tt.expected, result)
			}
		})
	}
}