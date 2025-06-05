package ui

import (
	"testing"
	
	"github.com/charmbracelet/lipgloss"
)

func TestStyleFunctions(t *testing.T) {
	// Test that all style functions return valid styles
	tests := []struct {
		name  string
		style func() lipgloss.Style
	}{
		{"SelectedStyle", SelectedStyle},
		{"NormalStyle", NormalStyle},
		{"BootedStyle", BootedStyle},
		{"ShutdownStyle", ShutdownStyle},
		{"HeaderStyle", HeaderStyle},
		{"ErrorStyle", ErrorStyle},
		{"SearchStyle", SearchStyle},
		{"NameStyle", NameStyle},
		{"DetailStyle", DetailStyle},
		{"BorderStyle", BorderStyle},
		{"ListItemStyle", ListItemStyle},
		{"FooterStyle", FooterStyle},
		{"FolderStyle", FolderStyle},
		{"StatusStyle", StatusStyle},
		{"LoadingStyle", LoadingStyle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := tt.style()
			// Simply verify that we get a style back without panicking
			// The actual style properties will depend on the loaded config
			_ = style.String()
		})
	}
}

func TestSuccessColor(t *testing.T) {
	// Test that SuccessColor returns a valid color
	color := SuccessColor()
	// The actual color will depend on the theme, but it should not be nil
	_ = color // Use the color to avoid compiler warning
}

func TestReloadStyles(t *testing.T) {
	// Test that ReloadStyles doesn't panic
	// We can't easily test the actual reload behavior without mocking the config
	err := ReloadStyles()
	// Error is expected if config file doesn't exist or is invalid
	// We're mainly testing that it doesn't panic
	_ = err
}

func TestStylesInitialization(t *testing.T) {
	// Test that styles are initialized properly
	// This is implicitly tested by the other tests, but we can be explicit
	
	// Get a style to ensure initialization has occurred
	style := HeaderStyle()
	
	// The style should be usable
	rendered := style.Render("Test")
	if rendered == "" {
		t.Error("Expected HeaderStyle to render some content")
	}
}