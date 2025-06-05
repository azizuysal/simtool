package config

import (
	"log"
	
	"github.com/charmbracelet/lipgloss"
)

// Styles holds all the application styles generated from config
type Styles struct {
	// Selection styles
	Selected lipgloss.Style
	Normal   lipgloss.Style
	
	// Status styles
	Booted   lipgloss.Style
	Shutdown lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
	
	// UI element styles
	Header  lipgloss.Style
	Border  lipgloss.Style
	Footer  lipgloss.Style
	
	// Content styles
	Name   lipgloss.Style
	Detail lipgloss.Style
	Folder lipgloss.Style
	
	// Search and status styles
	Search   lipgloss.Style
	Status   lipgloss.Style
	Loading  lipgloss.Style
	
	// List item style
	ListItem lipgloss.Style
}

// GenerateStyles creates lipgloss styles from the configuration
func (c *Config) GenerateStyles() *Styles {
	// Get the active theme name
	themeName := c.GetActiveTheme()
	
	// Extract colors from the theme
	colors, err := ExtractThemeColors(themeName)
	if err != nil {
		log.Printf("Warning: failed to extract colors from theme %q: %v", themeName, err)
		// Fallback to monokai
		colors, _ = ExtractThemeColors("monokai")
	}
	
	return &Styles{
		// Selection styles
		Selected: lipgloss.NewStyle().
			Background(ConvertToLipglossColor(colors.Selection)).
			Foreground(ConvertToLipglossColor(colors.SelectionText)),
		
		Normal: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Foreground)),
		
		// Status styles
		Booted: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Success)),
		
		Shutdown: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Secondary)),
		
		Error: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Error)),
		
		Success: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Success)),
		
		// UI element styles
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(ConvertToLipglossColor(colors.HeaderFg)).
			Background(ConvertToLipglossColor(colors.HeaderBg)).
			Padding(0, 2).
			MarginBottom(1),
		
		Border: lipgloss.NewStyle().
			Border(lipgloss.Border{
				Top:         "─",
				Bottom:      "─",
				Left:        "│",
				Right:       "│",
				TopLeft:     "╭",
				TopRight:    "╮",
				BottomLeft:  "╰",
				BottomRight: "╯",
			}).
			BorderForeground(ConvertToLipglossColor(colors.Border)).
			Padding(1, 2),
		
		Footer: lipgloss.NewStyle().
			Faint(true).
			Foreground(ConvertToLipglossColor(colors.Muted)),
		
		// Content styles
		Name: lipgloss.NewStyle().
			Bold(true).
			Foreground(ConvertToLipglossColor(colors.Primary)),
		
		Detail: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Secondary)),
		
		Folder: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Accent)).
			Bold(true),
		
		// Search and status styles
		Search: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Info)).
			Bold(true),
		
		Status: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Warning)).
			Bold(true),
		
		Loading: lipgloss.NewStyle().
			Foreground(ConvertToLipglossColor(colors.Info)).
			Bold(true),
		
		// List item style
		ListItem: lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2),
	}
}