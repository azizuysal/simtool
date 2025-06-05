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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.View()

			if tt.wantError {
				if !strings.Contains(got, "Error:") {
					t.Errorf("View() error case should contain 'Error:', got %v", got)
				}
			}

			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("View() should contain %q, got %v", want, got)
				}
			}
		})
	}
}

func TestViewStateIntegration(t *testing.T) {
	// Test that view correctly renders based on viewState
	model := Model{
		height: 30,
		width:  80,
		simulators: []simulator.Item{
			{
				Simulator: simulator.Simulator{
					Name:  "iPhone 15",
					State: "Booted",
					UDID:  "test-udid",
				},
				Runtime:  "iOS 17.0",
				AppCount: 2,
			},
		},
	}

	// Test SimulatorListView
	model.viewState = SimulatorListView
	view := model.View()
	if !strings.Contains(view, "iOS Simulators") {
		t.Error("SimulatorListView should show iOS Simulators")
	}

	// Test AppListView
	model.viewState = AppListView
	model.selectedSim = &model.simulators[0]
	model.apps = []simulator.App{
		{Name: "TestApp", BundleID: "com.test.app", Size: 1024},
	}
	view = model.View()
	if !strings.Contains(view, "iPhone 15 Apps") {
		t.Error("AppListView should show simulator name")
	}

	// Test FileListView
	model.viewState = FileListView
	model.selectedApp = &model.apps[0]
	model.files = []simulator.FileInfo{
		{Name: "Documents", IsDirectory: true, Size: 0},
	}
	view = model.View()
	if !strings.Contains(view, "TestApp Files") {
		t.Error("FileListView should show app name")
	}
}

func TestViewLoadingStates(t *testing.T) {
	model := Model{
		height: 30,
		width:  80,
	}

	// Test loading simulators
	model.viewState = SimulatorListView
	model.loadingSimulators = true
	view := model.View()
	if !strings.Contains(view, "Loading simulators...") {
		t.Error("Should show loading message when loading simulators")
	}

	// Test loading apps
	model.viewState = AppListView
	model.loadingApps = true
	model.selectedSim = &simulator.Item{
		Simulator: simulator.Simulator{Name: "iPhone 15"},
	}
	view = model.View()
	if !strings.Contains(view, "Loading apps...") {
		t.Error("Should show loading message when loading apps")
	}

	// Test loading files
	model.viewState = FileListView
	model.loadingFiles = true
	model.selectedApp = &simulator.App{Name: "TestApp"}
	view = model.View()
	if !strings.Contains(view, "Loading files...") {
		t.Error("Should show loading message when loading files")
	}
}

func TestViewSearchMode(t *testing.T) {
	model := Model{
		height: 30,
		width:  80,
		simulators: []simulator.Item{
			{
				Simulator: simulator.Simulator{Name: "iPhone 15"},
				Runtime:   "iOS 17.0",
			},
		},
		simSearchMode:  true,
		simSearchQuery: "iPhone",
	}

	view := model.View()
	if !strings.Contains(view, "Search: iPhone") {
		t.Error("Should show search query in search mode")
	}
}

func TestViewFilterMode(t *testing.T) {
	model := Model{
		height: 30,
		width:  80,
		simulators: []simulator.Item{
			{
				Simulator: simulator.Simulator{Name: "iPhone 15"},
				Runtime:   "iOS 17.0",
				AppCount:  2,
			},
		},
		filterActive: true,
	}

	view := model.View()
	if !strings.Contains(view, "Filter: Showing only simulators with apps") {
		t.Error("Should show filter status when filter is active")
	}
}