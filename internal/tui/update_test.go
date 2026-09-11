package tui

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
)

func TestUpdateThemeChange(t *testing.T) {
	model := Model{
		currentThemeMode: "light",
		simList: simListState{
			simulators: []simulator.Item{
				{Simulator: simulator.Simulator{Name: "iPhone 15"}},
			},
		},
		height: 30,
		width:  80,
	}

	// Test theme change message
	msg := themeChangedMsg{newMode: "dark"}
	updated, cmd := model.Update(msg)
	updatedModel := updated.(Model)

	if updatedModel.currentThemeMode != "dark" {
		t.Errorf("Expected theme mode to be updated to 'dark', got %q", updatedModel.currentThemeMode)
	}

	// cmd should be nil for theme change
	if cmd != nil {
		t.Error("Expected no command for theme change")
	}
}

func TestSearchBackspaceRemovesWholeRune(t *testing.T) {
	tests := []struct {
		name  string
		model Model
	}{
		{
			name:  "simulators",
			model: Model{viewState: SimulatorListView, simList: simListState{searchMode: true, searchQuery: "café"}},
		},
		{
			name:  "apps",
			model: Model{viewState: AppListView, appList: appListState{searchMode: true, searchQuery: "café"}},
		},
		{
			name:  "all apps",
			model: Model{viewState: AllAppsView, allApps: allAppsState{searchMode: true, searchQuery: "café"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.model.height = 24
			tt.model.width = 80
			tt.model.config = config.Default()
			tt.model.keyMap = config.NewKeyMap(tt.model.config.Keys)
			updated, _ := tt.model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
			model := updated.(Model)

			switch model.viewState {
			case SimulatorListView:
				if model.simList.searchQuery != "caf" {
					t.Errorf("searchQuery = %q, want caf", model.simList.searchQuery)
				}
			case AppListView:
				if model.appList.searchQuery != "caf" {
					t.Errorf("searchQuery = %q, want caf", model.appList.searchQuery)
				}
			case AllAppsView:
				if model.allApps.searchQuery != "caf" {
					t.Errorf("searchQuery = %q, want caf", model.allApps.searchQuery)
				}
			}
		})
	}
}

func TestSimulatorRefreshFailurePreservesActiveFileViewer(t *testing.T) {
	model := Model{
		viewState: FileViewerView,
		width:     80,
		height:    24,
		fileViewer: fileViewerState{
			file:    &simulator.FileInfo{Name: "notes.txt", Path: "/tmp/notes.txt"},
			content: &simulator.FileContent{Type: simulator.FileTypeText, Lines: []string{"keep this open"}},
		},
	}

	updated, cmd := model.handleFetchSimulators(fetchSimulatorsMsg{err: errors.New("simctl unavailable")})
	if cmd == nil {
		t.Fatal("expected status clear command")
	}
	if updated.viewState != FileViewerView {
		t.Errorf("viewState = %v, want FileViewerView", updated.viewState)
	}
	if updated.fileViewer.file == nil || updated.fileViewer.file.Name != "notes.txt" {
		t.Fatal("file viewer content was cleared")
	}
	if updated.err != nil {
		t.Errorf("err = %v, want nil for a background refresh failure", updated.err)
	}
	if updated.statusMessage != "Simulator refresh failed: simctl unavailable" {
		t.Errorf("statusMessage = %q", updated.statusMessage)
	}
}

func TestPrepareRemoteFileUsesLocalPreviewPath(t *testing.T) {
	fetcher := &mockFetcher{preparePath: "/tmp/preview.db"}
	model := Model{
		viewState: FileListView,
		fetcher:   fetcher,
		fileList: fileListState{
			currentPath: "/@android/emulator/package",
			files:       []simulator.FileInfo{{Name: "app.db", Path: "/@android/emulator/package/app.db"}},
			loading:     true,
			preparing:   true,
		},
	}
	file := simulator.FileInfo{Name: "app.db", Path: "/@android/emulator/package/app.db"}
	msg := model.prepareFileCmd(file)().(prepareFileMsg)
	updated, cmd := model.handlePrepareFile(msg)

	if updated.viewState != DatabaseTableListView {
		t.Errorf("viewState = %v, want DatabaseTableListView", updated.viewState)
	}
	if updated.dbTables.file == nil || updated.dbTables.file.Path != file.Path || updated.dbTables.file.PreviewPath() != "/tmp/preview.db" {
		t.Errorf("prepared file = %+v", updated.dbTables.file)
	}
	if cmd == nil {
		t.Fatal("expected database fetch command")
	}
}

func TestPreparingFileShowsStatusAndIgnoresRepeatedSelection(t *testing.T) {
	model := Model{
		viewState: FileListView,
		config:    config.Default(),
		fileList: fileListState{
			files:     []simulator.FileInfo{{Name: "notes.txt", Path: "/@android/notes.txt"}},
			loading:   true,
			preparing: true,
		},
	}
	_, _, _, status := model.renderFileListView()
	if !strings.Contains(status, "Preparing file...") {
		t.Fatalf("status = %q, want preparation message", status)
	}
	_, cmd := model.handleFileListKey("right")
	if cmd != nil {
		t.Fatal("repeated selection must not start a second preparation")
	}
}

func TestNavigatingDuringFilePreparationAllowsAnotherSelection(t *testing.T) {
	for _, action := range []string{"up", "down", "home", "end"} {
		t.Run(action, func(t *testing.T) {
			model := Model{
				viewState: FileListView,
				fetcher:   &mockFetcher{},
				width:     80,
				height:    24,
				fileList: fileListState{
					files: []simulator.FileInfo{
						{Name: "first.txt", Path: "/@android/first.txt"},
						{Name: "second.txt", Path: "/@android/second.txt"},
					},
				},
			}
			if action == "up" || action == "home" {
				model.fileList.cursor = 1
			}
			started, prepare := model.handleFileListKey("right")
			moved, _ := started.(Model).handleFileListKey(action)
			updated, cmd := moved.(Model).handlePrepareFile(prepare().(prepareFileMsg))
			if cmd != nil || updated.viewState != FileListView || updated.fileList.loading || updated.fileList.preparing {
				t.Fatal("navigation must discard the old preview and leave the file list usable")
			}
			selected, next := updated.handleFileListKey("right")
			if next == nil || !selected.(Model).fileList.preparing {
				t.Fatal("the newly selected file must be openable")
			}
		})
	}
}

func TestReturningToParentDuringPreparationIgnoresTheOldPreview(t *testing.T) {
	file := simulator.FileInfo{Name: "notes.txt", Path: "/@android/data/files/notes.txt"}
	model := Model{
		viewState: FileListView,
		fileList: fileListState{
			basePath:    "/@android/data",
			currentPath: "/@android/data/files",
			breadcrumbs: []string{"files"},
			files:       []simulator.FileInfo{file},
			loading:     true,
			preparing:   true,
		},
	}
	parent, _ := model.handleFileListKey("left")
	updated, cmd := parent.(Model).handlePrepareFile(prepareFileMsg{file: file, path: "/tmp/notes.txt"})
	if cmd != nil || updated.viewState != FileListView || updated.fileList.currentPath != model.fileList.basePath || !updated.fileList.loading || updated.fileList.preparing {
		t.Fatal("the old preview must not replace the pending parent directory listing")
	}
}

func TestFetchFilesAccessErrorReturnsToTheCorrectParent(t *testing.T) {
	t.Run("nested folder", func(t *testing.T) {
		model := Model{
			viewState: FileListView,
			fileList: fileListState{
				selectedApp: &simulator.App{SimulatorUDID: "android-1"},
				basePath:    "/@android/android-1/com.example",
				currentPath: "/@android/android-1/com.example/data",
				breadcrumbs: []string{"data"},
				loading:     true,
			},
		}
		updated, cmd := model.handleFetchFiles(fetchFilesMsg{path: model.fileList.currentPath, err: errors.New("permission denied")})
		if cmd == nil || updated.viewState != FileListView || updated.fileList.currentPath != updated.fileList.basePath || !updated.fileList.loading {
			t.Fatalf("nested error did not return to parent: %+v", updated.fileList)
		}
		if !strings.Contains(updated.statusMessage, "Access restricted") {
			t.Fatalf("status = %q", updated.statusMessage)
		}
	})

	t.Run("root all apps", func(t *testing.T) {
		model := Model{
			viewState: FileListView,
			fileList: fileListState{
				selectedApp: &simulator.App{SimulatorUDID: "android-1"},
				currentPath: "/@android/android-1/com.example",
				loading:     true,
			},
		}
		updated, _ := model.handleFetchFiles(fetchFilesMsg{path: model.fileList.currentPath, err: errors.New("permission denied")})
		if updated.viewState != AllAppsView || updated.fileList.selectedApp != nil {
			t.Fatalf("root all-apps error = view %v file list %+v", updated.viewState, updated.fileList)
		}
	})
}

func TestStaleFileContentDoesNotReplaceTheActiveFile(t *testing.T) {
	active := simulator.FileInfo{Name: "second.txt", Path: "/tmp/second.txt"}
	model := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			file:          &active,
			contentOffset: 5,
			loading:       true,
		},
	}
	updated, cmd := model.handleFetchFileContent(fetchFileContentMsg{
		path:    "/tmp/first.txt",
		offset:  0,
		content: &simulator.FileContent{Type: simulator.FileTypeText, Lines: []string{"old"}},
	})
	if cmd != nil || updated.fileViewer.content != nil || !updated.fileViewer.loading {
		t.Fatal("stale file content changed the active viewer")
	}
}

func TestPlatformFilterAndCtrlCInSearch(t *testing.T) {
	model := Model{
		viewState: SimulatorListView,
		config:    config.Default(),
		keyMap:    config.NewKeyMap(config.DefaultKeys()),
		simList: simListState{simulators: []simulator.Item{
			{Simulator: simulator.Simulator{Name: "iPhone", Platform: "ios"}},
			{Simulator: simulator.Simulator{Name: "Pixel", Platform: "android"}},
		}},
	}

	updated, _ := model.handleSimulatorListKey("platform")
	filtered := updated.(Model).getFilteredAndSearchedSimulators()
	if len(filtered) != 1 || filtered[0].Platform != "ios" {
		t.Errorf("iOS platform filter = %+v", filtered)
	}

	model.simList.searchMode = true
	quitModel, cmd := model.Update(tea.KeyPressMsg{Code: 'c', Text: "c", Mod: tea.ModCtrl})
	if quitModel == nil || cmd == nil {
		t.Fatal("Ctrl+C should quit from search mode")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("Ctrl+C command = %T, want tea.QuitMsg", cmd())
	}
}

func TestUpdateTickMsg(t *testing.T) {
	// Save original env var
	originalOverride := os.Getenv("SIMTOOL_THEME_MODE")
	defer func() {
		if originalOverride != "" {
			_ = os.Setenv("SIMTOOL_THEME_MODE", originalOverride)
		} else {
			_ = os.Unsetenv("SIMTOOL_THEME_MODE")
		}
	}()

	// Set override to prevent actual theme detection
	_ = os.Setenv("SIMTOOL_THEME_MODE", "dark")

	model := Model{
		currentThemeMode: "dark",
		viewState:        SimulatorListView,
		fetcher:          &mockFetcher{},
	}

	// Test tick message
	msg := tickMsg(time.Now())
	updated, cmd := model.Update(msg)
	updatedModel := updated.(Model)

	// Model should remain unchanged
	if updatedModel.currentThemeMode != "dark" {
		t.Errorf("Theme mode should not change with override set")
	}

	// Should return commands for refresh and next tick
	if cmd == nil {
		t.Error("Expected commands from tick update")
	}
}

func TestUpdateWindowSizeMsg(t *testing.T) {
	model := Model{
		height: 24,
		width:  80,
	}

	// Test window resize
	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 40,
	}

	updated, cmd := model.Update(msg)
	updatedModel := updated.(Model)

	if updatedModel.width != 100 {
		t.Errorf("Expected width to be 100, got %d", updatedModel.width)
	}

	if updatedModel.height != 40 {
		t.Errorf("Expected height to be 40, got %d", updatedModel.height)
	}

	// No command expected for window resize
	if cmd != nil {
		t.Error("Expected no command for window resize")
	}
}

func TestOfflineRefreshDuringBootPreservesDeviceList(t *testing.T) {
	model := Model{viewState: SimulatorListView, simList: simListState{booting: true, simulators: []simulator.Item{{Simulator: simulator.Simulator{Name: "Pixel", Platform: "android"}}}}}
	updated, cmd := model.handleFetchSimulators(fetchSimulatorsMsg{err: errors.New("emulator is offline during startup")})
	if updated.err != nil || len(updated.simList.simulators) != 1 || !updated.simList.booting || cmd == nil {
		t.Fatal("transient offline refresh replaced the booting device list")
	}
}
