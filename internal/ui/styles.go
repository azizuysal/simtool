package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/azizuysal/simtool/internal/config"
)

var (
	// Global styles instance
	styles *config.Styles

	successColor color.Color
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

func InitializeStyles(cfg *config.Config) error {
	colors, err := config.ExtractThemeColors(cfg.GetActiveTheme())
	if err != nil {
		return err
	}

	generatedStyles := cfg.GenerateStyles()
	styles = generatedStyles
	successColor = config.ConvertToLipglossColor(colors.Success)
	return nil
}

// SuccessColor returns the configured success color.
func SuccessColor() color.Color {
	return successColor
}

// ReloadStyles reloads styles from configuration
// This can be called if the config file changes
func ReloadStyles() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	return InitializeStyles(cfg)
}
