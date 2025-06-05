package config

import (
	"os"
	"sync"
)

var (
	detectedDarkMode     bool
	detectedDarkModeOnce sync.Once
)

// InitializeThemeDetection performs terminal theme detection before TUI starts
func InitializeThemeDetection() {
	detectedDarkModeOnce.Do(func() {
		// Skip if override is set
		if override := os.Getenv("SIMTOOL_THEME_MODE"); override != "" {
			return
		}
		
		// Try to query terminal background
		bgColor, err := QueryTerminalBackgroundColor()
		if err == nil && bgColor != "" {
			isDark := IsColorDark(bgColor)
			// Store the result in an environment variable for later use
			if isDark {
				os.Setenv("SIMTOOL_DETECTED_MODE", "dark")
			} else {
				os.Setenv("SIMTOOL_DETECTED_MODE", "light")
			}
		}
	})
}

// GetDetectedMode returns the previously detected mode
func GetDetectedMode() string {
	return os.Getenv("SIMTOOL_DETECTED_MODE")
}