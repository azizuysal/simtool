package tui

import (
	"os"
	"testing"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/simulator"
)

func TestUpdateThemeChange(t *testing.T) {
	model := Model{
		currentThemeMode: "light",
		simulators: []simulator.Item{
			{Simulator: simulator.Simulator{Name: "iPhone 15"}},
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

func TestUpdateTickMsg(t *testing.T) {
	// Save original env var
	originalOverride := os.Getenv("SIMTOOL_THEME_MODE")
	defer func() {
		if originalOverride != "" {
			os.Setenv("SIMTOOL_THEME_MODE", originalOverride)
		} else {
			os.Unsetenv("SIMTOOL_THEME_MODE")
		}
	}()

	// Set override to prevent actual theme detection
	os.Setenv("SIMTOOL_THEME_MODE", "dark")

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