package ui

import (
	"log"
	"github.com/azizuysal/simtool/internal/config"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Global styles instance
	styles *config.Styles
	
	// Legacy color constants (for backward compatibility)
	successColor lipgloss.Color
)

// Style getter functions that always return the current styles
// This ensures components always use the latest styles after reload

func SelectedStyle() lipgloss.Style {
	if styles != nil {
		return styles.Selected
	}
	return lipgloss.NewStyle()
}

func NormalStyle() lipgloss.Style {
	if styles != nil {
		return styles.Normal
	}
	return lipgloss.NewStyle()
}

func BootedStyle() lipgloss.Style {
	if styles != nil {
		return styles.Booted
	}
	return lipgloss.NewStyle()
}

func ShutdownStyle() lipgloss.Style {
	if styles != nil {
		return styles.Shutdown
	}
	return lipgloss.NewStyle()
}

func HeaderStyle() lipgloss.Style {
	if styles != nil {
		return styles.Header
	}
	return lipgloss.NewStyle()
}

func ErrorStyle() lipgloss.Style {
	if styles != nil {
		return styles.Error
	}
	return lipgloss.NewStyle()
}

func SearchStyle() lipgloss.Style {
	if styles != nil {
		return styles.Search
	}
	return lipgloss.NewStyle()
}

func NameStyle() lipgloss.Style {
	if styles != nil {
		return styles.Name
	}
	return lipgloss.NewStyle()
}

func DetailStyle() lipgloss.Style {
	if styles != nil {
		return styles.Detail
	}
	return lipgloss.NewStyle()
}

func BorderStyle() lipgloss.Style {
	if styles != nil {
		return styles.Border
	}
	return lipgloss.NewStyle()
}

func ListItemStyle() lipgloss.Style {
	if styles != nil {
		return styles.ListItem
	}
	return lipgloss.NewStyle()
}

func FooterStyle() lipgloss.Style {
	if styles != nil {
		return styles.Footer
	}
	return lipgloss.NewStyle()
}

func FolderStyle() lipgloss.Style {
	if styles != nil {
		return styles.Folder
	}
	return lipgloss.NewStyle()
}

func StatusStyle() lipgloss.Style {
	if styles != nil {
		return styles.Status
	}
	return lipgloss.NewStyle()
}

func LoadingStyle() lipgloss.Style {
	if styles != nil {
		return styles.Loading
	}
	return lipgloss.NewStyle()
}

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
		successColor = config.ConvertToLipglossColor(colors.Success)
	} else {
		// Extract from github-dark as absolute fallback
		githubDarkColors, _ := config.ExtractThemeColors("github-dark")
		if githubDarkColors != nil {
			successColor = config.ConvertToLipglossColor(githubDarkColors.Success)
		} else {
			successColor = lipgloss.Color("") // No color if all fails
		}
	}
}

// SuccessColor returns the success color
func SuccessColor() lipgloss.Color {
	return successColor
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
		successColor = config.ConvertToLipglossColor(colors.Success)
	} else {
		// Extract from github-dark as absolute fallback
		githubDarkColors, _ := config.ExtractThemeColors("github-dark")
		if githubDarkColors != nil {
			successColor = config.ConvertToLipglossColor(githubDarkColors.Success)
		} else {
			successColor = lipgloss.Color("") // No color if all fails
		}
	}
	
	return nil
}