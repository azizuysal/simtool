package tui

import (
	"errors"
	"os"
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
