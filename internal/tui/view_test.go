package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
	"github.com/azizuysal/simtool/internal/ui"
)

func TestView(t *testing.T) {
	// Create a default config for all tests
	defaultConfig := config.Default()

	tests := []struct {
		name      string
		model     Model
		wantError bool
		contains  []string
	}{
		{
			name: "error state",
			model: Model{
				err:    simulator.ErrSimulatorNotFound,
				config: defaultConfig,
			},
			wantError: true,
			contains:  []string{"Error:", "simulator not found"},
		},
		{
			name: "simulator list view",
			model: Model{
				viewState: SimulatorListView,
				simList: simListState{
					simulators: []simulator.Item{
						{
							Simulator: simulator.Simulator{
								Name:  "iPhone 15",
								State: "Booted",
							},
							Runtime: "iOS 17.0",
						},
					},
				},
				height: 30,
				width:  80,
				config: defaultConfig,
			},
			contains: []string{"Devices"},
		},
		{
			name: "app list view",
			model: Model{
				viewState: AppListView,
				appList: appListState{
					selectedSim: &simulator.Item{
						Simulator: simulator.Simulator{Name: "iPhone 15"},
					},
					apps: []simulator.App{
						{Name: "TestApp", BundleID: "com.test.app"},
					},
				},
				height: 30,
				width:  80,
				config: defaultConfig,
			},
			contains: []string{"iPhone 15 Apps"},
		},
		{
			name: "file list view",
			model: Model{
				viewState: FileListView,
				fileList: fileListState{
					selectedApp: &simulator.App{
						Name: "TestApp",
					},
					files: []simulator.FileInfo{
						{Name: "test.txt", IsDirectory: false},
					},
				},
				height: 30,
				width:  80,
				config: defaultConfig,
			},
			contains: []string{"TestApp Files"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.View().Content

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
		config: config.Default(),
		simList: simListState{
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
		},
	}

	// Test SimulatorListView
	model.viewState = SimulatorListView
	view := model.View().Content
	if !strings.Contains(view, "Devices") {
		t.Error("SimulatorListView should show Devices")
	}

	// Test AppListView
	model.viewState = AppListView
	model.appList.selectedSim = &model.simList.simulators[0]
	model.appList.apps = []simulator.App{
		{Name: "TestApp", BundleID: "com.test.app", Size: 1024},
	}
	view = model.View().Content
	if !strings.Contains(view, "iPhone 15 Apps") {
		t.Error("AppListView should show simulator name")
	}

	// Test FileListView
	model.viewState = FileListView
	model.fileList.selectedApp = &model.appList.apps[0]
	model.fileList.files = []simulator.FileInfo{
		{Name: "Documents", IsDirectory: true, Size: 0},
	}
	view = model.View().Content
	if !strings.Contains(view, "TestApp Files") {
		t.Error("FileListView should show app name")
	}
}

func TestViewLoadingStates(t *testing.T) {
	model := Model{
		height: 30,
		width:  80,
		config: config.Default(),
	}

	// Test loading simulators
	model.viewState = SimulatorListView
	model.simList.loading = true
	view := model.View().Content
	if !strings.Contains(view, "Loading simulators...") {
		t.Error("Should show loading message when loading simulators")
	}

	// Test loading apps
	model.viewState = AppListView
	model.appList.loading = true
	model.appList.selectedSim = &simulator.Item{
		Simulator: simulator.Simulator{Name: "iPhone 15"},
	}
	view = model.View().Content
	if !strings.Contains(view, "Loading apps...") {
		t.Error("Should show loading message when loading apps")
	}

	// Test loading files
	model.viewState = FileListView
	model.fileList.loading = true
	model.fileList.selectedApp = &simulator.App{Name: "TestApp"}
	view = model.View().Content
	if !strings.Contains(view, "Loading files...") {
		t.Error("Should show loading message when loading files")
	}
}

func TestViewSearchMode(t *testing.T) {
	model := Model{
		height: 30,
		width:  80,
		config: config.Default(),
		simList: simListState{
			simulators: []simulator.Item{
				{
					Simulator: simulator.Simulator{Name: "iPhone 15"},
					Runtime:   "iOS 17.0",
				},
			},
			searchMode:  true,
			searchQuery: "iPhone",
		},
	}

	view := model.View().Content
	if !strings.Contains(view, "Search: iPhone") {
		t.Error("Should show search query in search mode")
	}
}

func TestViewFilterMode(t *testing.T) {
	model := Model{
		height: 30,
		width:  80,
		config: config.Default(),
		simList: simListState{
			simulators: []simulator.Item{
				{
					Simulator: simulator.Simulator{Name: "iPhone 15"},
					Runtime:   "iOS 17.0",
					AppCount:  2,
				},
			},
			filterActive: true,
		},
	}

	view := model.View().Content
	if !strings.Contains(view, "Filter: Showing only simulators with apps") {
		t.Error("Should show filter status when filter is active")
	}
}

func TestViewRendersAtNarrowDimensions(t *testing.T) {
	model := Model{
		viewState: FileViewerView,
		width:     16,
		height:    8,
		config:    config.Default(),
		fileViewer: fileViewerState{
			file:    &simulator.FileInfo{Name: "notes.txt", Path: "/tmp/notes.txt"},
			content: &simulator.FileContent{Type: simulator.FileTypeText, Lines: []string{"a line that is wider than the viewport"}},
		},
	}

	if content := model.View().Content; content == "" {
		t.Error("View() returned empty content")
	}
}

func TestPlatformFooterFitsWithLastListItemVisible(t *testing.T) {
	if err := ui.InitializeStyles(config.Default()); err != nil {
		t.Fatal(err)
	}
	for _, state := range []ViewState{SimulatorListView, AllAppsView} {
		for _, width := range []int{80, 120} {
			for _, height := range []int{22, 24, 30} {
				model := Model{viewState: state, width: width, height: height, config: config.Default(), keyMap: config.NewKeyMap(config.DefaultKeys())}
				for range 20 {
					model.simList.simulators = append(model.simList.simulators, simulator.Item{Simulator: simulator.Simulator{Name: "Device"}})
					model.allApps.apps = append(model.allApps.apps, simulator.App{Name: "App"})
				}
				model.simList.simulators[19].Name = "LAST-ITEM"
				model.allApps.apps[19].Name = "LAST-ITEM"
				updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
				content := updated.(Model).View().Content
				for _, label := range []string{"p: platform", "quit", "LAST-ITEM"} {
					if !strings.Contains(content, label) {
						t.Errorf("view %d at %dx%d is missing %q", state, width, height, label)
					}
				}
				if actual := lipgloss.Height(content); actual > height {
					t.Errorf("view %d at %dx%d has %d rows", state, width, height, actual)
				}
				for _, line := range strings.Split(content, "\n") {
					if actual := lipgloss.Width(line); actual > width {
						t.Errorf("view %d at %dx%d has a %d-column line", state, width, height, actual)
					}
				}
			}
		}
	}
}

func TestAllAppsViewShowsBackgroundRefreshFailure(t *testing.T) {
	model := Model{
		viewState:     AllAppsView,
		width:         80,
		height:        24,
		config:        config.Default(),
		statusMessage: "Simulator refresh failed: simctl unavailable",
	}

	if content := model.View().Content; !strings.Contains(content, model.statusMessage) {
		t.Errorf("View() did not show status %q", model.statusMessage)
	}
}

func TestFileViewerFitsInitializedLayout(t *testing.T) {
	if err := ui.InitializeStyles(config.Default()); err != nil {
		t.Fatal(err)
	}

	lines := make([]string, 37)
	lines[0] = "ASCII-" + strings.Repeat("x", 200)
	lines[1] = "CJK-" + strings.Repeat("界", 100)
	for i := 2; i < len(lines)-1; i++ {
		lines[i] = "intermediate row"
	}
	lines[len(lines)-1] = "LAST-ROW-MARKER"

	model := Model{
		viewState: FileViewerView,
		width:     110,
		height:    30,
		config:    config.Default(),
		keyMap:    config.NewKeyMap(config.DefaultKeys()),
		fileViewer: fileViewerState{
			file: &simulator.FileInfo{Name: "notes.txt", Path: "/tmp/notes.txt"},
			content: &simulator.FileContent{
				Type:       simulator.FileTypeText,
				TotalLines: len(lines),
				Lines:      lines,
			},
		},
	}

	content := model.View().Content
	if height := lipgloss.Height(content); height > model.height {
		t.Errorf("rendered height = %d, want <= %d", height, model.height)
	}
	for lineNumber, line := range strings.Split(content, "\n") {
		if width := lipgloss.Width(line); width > model.width {
			t.Errorf("line %d width = %d, want <= %d", lineNumber+1, width, model.width)
		}
	}
	renderedLines := strings.Split(content, "\n")
	for i, line := range renderedLines {
		if strings.Contains(line, "ASCII-") {
			if i+2 >= len(renderedLines) || !strings.Contains(renderedLines[i+1], "CJK-") || !strings.Contains(renderedLines[i+2], "intermediate row") {
				t.Fatal("long ASCII or CJK line wrapped into multiple display rows")
			}
		}
	}

	for range 37 {
		updated, _ := model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
		model = updated.(Model)
	}
	content = model.View().Content
	if !strings.Contains(content, "LAST-ROW-MARKER") {
		t.Fatalf("last text row is not reachable through Update scrolling (viewport %d)", model.fileViewer.contentViewport)
	}
	if !strings.Contains(content, "scroll up") {
		t.Fatal("file viewer footer disappeared after scrolling")
	}

	files := make([]simulator.FileInfo, 37)
	for i := range files {
		files[i] = simulator.FileInfo{Name: "file.txt"}
	}
	files[len(files)-1].Name = "last-file.txt"
	fileListModel := Model{
		viewState: FileListView,
		width:     110,
		height:    30,
		config:    config.Default(),
		keyMap:    config.NewKeyMap(config.DefaultKeys()),
		fileList: fileListState{
			selectedApp: &simulator.App{Name: "Example"},
			files:       files,
		},
	}
	updated, _ := fileListModel.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if content := updated.(Model).View().Content; !strings.Contains(content, "▶ last-file.txt") {
		t.Fatal("last file-list row is not reachable through Update navigation")
	}
}

func TestListNavigationKeepsTheLastSelectionVisible(t *testing.T) {
	if err := ui.InitializeStyles(config.Default()); err != nil {
		t.Fatal(err)
	}
	apps := make([]simulator.App, 12)
	sims := make([]simulator.Item, 12)
	files := make([]simulator.FileInfo, 12)
	for i := range apps {
		apps[i] = simulator.App{Name: "item", BundleID: "com.example.app"}
		sims[i] = simulator.Item{Simulator: simulator.Simulator{Name: "item"}}
		files[i] = simulator.FileInfo{Name: "item.txt", Size: 1024}
	}
	apps[11].Name = "last-item"
	sims[11].Name = "last-item"
	files[11].Name = "last-item"

	for _, height := range []int{24, 30} {
		for _, state := range []ViewState{SimulatorListView, AppListView, AllAppsView, FileListView} {
			model := Model{
				viewState: state,
				width:     110,
				height:    height,
				config:    config.Default(),
				keyMap:    config.NewKeyMap(config.DefaultKeys()),
				simList:   simListState{simulators: sims},
				appList:   appListState{apps: apps, selectedSim: &sims[0]},
				allApps:   allAppsState{apps: apps},
				fileList:  fileListState{files: files, selectedApp: &apps[0]},
			}
			updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
			content := updated.(Model).View().Content
			if !strings.Contains(content, "▶ last-item") {
				t.Errorf("view %v at height %d hides the last selected item", state, height)
			}
			if renderedHeight := lipgloss.Height(content); renderedHeight > height {
				t.Errorf("view %v at height %d renders %d rows", state, height, renderedHeight)
			}
		}
	}
}
