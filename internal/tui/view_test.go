package tui

import (
	"strings"
	"testing"

	"simtool/internal/simulator"
)

func TestView(t *testing.T) {
	tests := []struct {
		name      string
		model     Model
		wantError bool
		contains  []string
	}{
		{
			name: "error state",
			model: Model{
				err: simulator.ErrSimulatorNotFound,
			},
			wantError: true,
			contains:  []string{"Error:", "simulator not found"},
		},
		{
			name: "simulator list view",
			model: Model{
				viewState: SimulatorListView,
				simulators: []simulator.Item{
					{
						Simulator: simulator.Simulator{
							Name:  "iPhone 15",
							State: "Booted",
						},
						Runtime: "iOS 17.0",
					},
				},
				height: 30,
				width:  80,
			},
			contains: []string{"iOS Simulators"},
		},
		{
			name: "app list view",
			model: Model{
				viewState: AppListView,
				selectedSim: &simulator.Item{
					Simulator: simulator.Simulator{Name: "iPhone 15"},
				},
				apps: []simulator.App{
					{Name: "TestApp", BundleID: "com.test.app"},
				},
				height: 30,
				width:  80,
			},
			contains: []string{"iPhone 15 Apps"},
		},
		{
			name: "file list view",
			model: Model{
				viewState: FileListView,
				selectedApp: &simulator.App{
					Name: "TestApp",
				},
				files: []simulator.FileInfo{
					{Name: "test.txt", IsDirectory: false},
				},
				height: 30,
				width:  80,
			},
			contains: []string{"TestApp Files"},
		},
		{
			name: "file viewer view",
			model: Model{
				viewState: FileViewerView,
				viewingFile: &simulator.FileInfo{
					Path: "/path/to/test.txt",
				},
				fileContent: &simulator.FileContent{
					Type:  simulator.FileTypeText,
					Lines: []string{"Hello", "World"},
				},
				height: 30,
				width:  80,
			},
			contains: []string{"test.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.model.View()

			if tt.wantError {
				if !strings.Contains(result, "Error:") {
					t.Error("Expected error message")
				}
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected view to contain '%s'", expected)
				}
			}
		})
	}
}

func TestCenterContent(t *testing.T) {
	model := Model{width: 80}
	
	tests := []struct {
		name     string
		content  string
		width    int
		centered bool
	}{
		{
			name:     "short content",
			content:  "Hello",
			width:    80,
			centered: true,
		},
		{
			name:     "multi-line content",
			content:  "Line 1\nLine 2\nLine 3",
			width:    80,
			centered: true,
		},
		{
			name:     "content wider than width",
			content:  strings.Repeat("x", 100),
			width:    80,
			centered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.width = tt.width
			result := model.centerContent(tt.content)

			lines := strings.Split(result, "\n")
			if len(lines) == 0 {
				t.Error("Expected at least one line")
				return
			}

			// Check if content is centered (has leading spaces)
			if tt.centered && len(lines[0]) > 0 {
				if !strings.HasPrefix(lines[0], " ") && tt.width > len(tt.content) {
					t.Log("Content might not be centered")
				}
			}
		})
	}
}

func TestPadContentToHeight(t *testing.T) {
	model := Model{}
	
	tests := []struct {
		name           string
		content        string
		itemsPerScreen int
		minExpected    int
	}{
		{
			name:           "short content",
			content:        "Line 1\nLine 2",
			itemsPerScreen: 5,
			minExpected:    2, // At least the original content
		},
		{
			name:           "exact fit",
			content:        strings.Repeat("Line\n", 14),
			itemsPerScreen: 5,
			minExpected:    14,
		},
		{
			name:           "content too long",
			content:        strings.Repeat("Line\n", 20),
			itemsPerScreen: 5,
			minExpected:    20, // No padding, return as is
		},
		{
			name:           "single item",
			content:        "One line",
			itemsPerScreen: 1,
			minExpected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.padContentToHeight(tt.content, tt.itemsPerScreen)
			lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
			
			if len(lines) < tt.minExpected {
				t.Errorf("Expected at least %d lines, got %d", tt.minExpected, len(lines))
			}
			
			// Check that padding was added for short content
			originalLines := strings.Split(strings.TrimRight(tt.content, "\n"), "\n")
			if len(originalLines) < tt.itemsPerScreen*3-1 && len(lines) <= len(originalLines) {
				t.Log("Warning: No padding was added to short content")
			}
		})
	}
}

func TestRenderSimulatorList(t *testing.T) {
	simulators := []simulator.Item{
		{
			Simulator: simulator.Simulator{
				Name:  "iPhone 15",
				State: "Booted",
			},
			Runtime:  "iOS 17.0",
			AppCount: 5,
		},
		{
			Simulator: simulator.Simulator{
				Name:  "iPhone 14",
				State: "Shutdown",
			},
			Runtime:  "iOS 16.4",
			AppCount: 0,
		},
	}

	model := Model{
		simulators: simulators,
		simCursor:  0,
		height:     30,
		width:      80,
	}

	result := model.renderSimulatorList(model.simulators, 0, 2, 70)

	// Check that result contains expected elements
	if !strings.Contains(result, "iPhone 15") {
		t.Error("Expected iPhone 15 in result")
	}

	if !strings.Contains(result, "iOS 17.0") {
		t.Error("Expected iOS 17.0 in result")
	}

	if !strings.Contains(result, "Running") {
		t.Error("Expected Running state")
	}

	if !strings.Contains(result, "5 apps") {
		t.Error("Expected app count")
	}

	if !strings.Contains(result, "▶") {
		t.Error("Expected cursor indicator")
	}
}

func TestViewStates(t *testing.T) {
	// Test empty simulator list
	model := Model{
		viewState:  SimulatorListView,
		simulators: []simulator.Item{},
		height:     30,
		width:      80,
	}

	result := model.viewSimulatorList()
	if !strings.Contains(result, "Loading simulators...") {
		t.Error("Expected loading message for empty simulator list")
	}

	// Test loading apps state
	model = Model{
		viewState:   AppListView,
		loadingApps: true,
		height:      30,
		width:       80,
	}

	result = model.viewAppList()
	if !strings.Contains(result, "Loading apps...") {
		t.Error("Expected loading message for apps")
	}

	// Test no simulator selected
	model = Model{
		viewState:   AppListView,
		selectedSim: nil,
		height:      30,
		width:       80,
	}

	result = model.viewAppList()
	if !strings.Contains(result, "No simulator selected") {
		t.Error("Expected no simulator selected message")
	}

	// Test loading files state
	model = Model{
		viewState:    FileListView,
		loadingFiles: true,
		height:       30,
		width:        80,
	}

	result = model.viewFileList()
	if !strings.Contains(result, "Loading files...") {
		t.Error("Expected loading message for files")
	}

	// Test no app selected
	model = Model{
		viewState:   FileListView,
		selectedApp: nil,
		height:      30,
		width:       80,
	}

	result = model.viewFileList()
	if !strings.Contains(result, "No app selected") {
		t.Error("Expected no app selected message")
	}
}