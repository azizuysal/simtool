package ui

import (
	"log"
	"simtool/internal/config"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Global styles instance
	styles *config.Styles
	
	// Legacy color constants (for backward compatibility)
	SuccessColor lipgloss.Color

	// Style variables for backward compatibility
	SelectedStyle lipgloss.Style
	NormalStyle   lipgloss.Style
	BootedStyle   lipgloss.Style
	ShutdownStyle lipgloss.Style
	HeaderStyle   lipgloss.Style
	ErrorStyle    lipgloss.Style
	SearchStyle   lipgloss.Style
	NameStyle     lipgloss.Style
	DetailStyle   lipgloss.Style
	BorderStyle   lipgloss.Style
	ListItemStyle lipgloss.Style
	FooterStyle   lipgloss.Style
	FolderStyle   lipgloss.Style
	StatusStyle   lipgloss.Style
	LoadingStyle  lipgloss.Style
)

func init() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: failed to load config, using defaults: %v", err)
		cfg = config.Default()
	}
	
	// Generate styles from config
	styles = cfg.GenerateStyles()
	
	// Map to legacy variables for backward compatibility
	// Get colors from the extracted theme
	colors, _ := config.ExtractThemeColors(cfg.GetActiveTheme())
	if colors != nil {
		SuccessColor = config.ConvertToLipglossColor(colors.Success)
	} else {
		// Extract from github-dark as absolute fallback
		githubDarkColors, _ := config.ExtractThemeColors("github-dark")
		if githubDarkColors != nil {
			SuccessColor = config.ConvertToLipglossColor(githubDarkColors.Success)
		} else {
			SuccessColor = lipgloss.Color("") // No color if all fails
		}
	}
	
	SelectedStyle = styles.Selected
	NormalStyle = styles.Normal
	BootedStyle = styles.Booted
	ShutdownStyle = styles.Shutdown
	HeaderStyle = styles.Header
	ErrorStyle = styles.Error
	SearchStyle = styles.Search
	NameStyle = styles.Name
	DetailStyle = styles.Detail
	BorderStyle = styles.Border
	ListItemStyle = styles.ListItem
	FooterStyle = styles.Footer
	FolderStyle = styles.Folder
	StatusStyle = styles.Status
	LoadingStyle = styles.Loading
}

// ReloadStyles reloads styles from configuration
// This can be called if the config file changes
func ReloadStyles() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	
	// Regenerate styles
	styles = cfg.GenerateStyles()
	
	// Update legacy variables
	// Get colors from the extracted theme
	colors, _ := config.ExtractThemeColors(cfg.GetActiveTheme())
	if colors != nil {
		SuccessColor = config.ConvertToLipglossColor(colors.Success)
	} else {
		// Extract from github-dark as absolute fallback
		githubDarkColors, _ := config.ExtractThemeColors("github-dark")
		if githubDarkColors != nil {
			SuccessColor = config.ConvertToLipglossColor(githubDarkColors.Success)
		} else {
			SuccessColor = lipgloss.Color("") // No color if all fails
		}
	}
	
	SelectedStyle = styles.Selected
	NormalStyle = styles.Normal
	BootedStyle = styles.Booted
	ShutdownStyle = styles.Shutdown
	HeaderStyle = styles.Header
	ErrorStyle = styles.Error
	SearchStyle = styles.Search
	NameStyle = styles.Name
	DetailStyle = styles.Detail
	BorderStyle = styles.Border
	ListItemStyle = styles.ListItem
	FooterStyle = styles.Footer
	FolderStyle = styles.Folder
	StatusStyle = styles.Status
	LoadingStyle = styles.Loading
	
	return nil
}