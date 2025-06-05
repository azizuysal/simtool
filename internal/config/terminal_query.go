package config

import (
	"os/exec"
	"strings"
)

// QueryTerminalBackground attempts to detect terminal background
func QueryTerminalBackground() string {
	// First try OSC query for accurate terminal color detection
	bgColor, err := QueryTerminalBackgroundColor()
	if err == nil && bgColor != "" {
		if IsColorDark(bgColor) {
			return "dark"
		}
		return "light"
	}
	
	// Fall back to macOS system dark mode as proxy
	return queryMacOSTerminalBackground()
}

// queryMacOSTerminalBackground queries terminal background on macOS
func queryMacOSTerminalBackground() string {
	// Check system dark mode as fallback
	if isSystemDarkMode() {
		return "dark"
	}
	return "light"
}

// isSystemDarkMode checks if macOS is in dark mode
func isSystemDarkMode() bool {
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, AppleInterfaceStyle is not set, meaning light mode
		return false
	}
	
	// If it returns "Dark", system is in dark mode
	return strings.TrimSpace(string(output)) == "Dark"
}

// QueryTerminalBackgroundLive attempts to detect terminal background for live updates
// This version is designed to work even when running inside a TUI
func QueryTerminalBackgroundLive() string {
	// For now, just use system detection since OSC queries don't work in TUI
	// In the future, we could try more sophisticated approaches
	return queryMacOSTerminalBackground()
}

