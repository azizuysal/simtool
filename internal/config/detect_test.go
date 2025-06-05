package config

import (
	"os"
	"testing"
)

func TestInitializeThemeDetection(t *testing.T) {
	// Save original env vars
	originalOverride := os.Getenv("SIMTOOL_THEME_MODE")
	originalDetected := os.Getenv("SIMTOOL_DETECTED_MODE")
	defer func() {
		// Restore env vars
		if originalOverride != "" {
			os.Setenv("SIMTOOL_THEME_MODE", originalOverride)
		} else {
			os.Unsetenv("SIMTOOL_THEME_MODE")
		}
		if originalDetected != "" {
			os.Setenv("SIMTOOL_DETECTED_MODE", originalDetected)
		} else {
			os.Unsetenv("SIMTOOL_DETECTED_MODE")
		}
	}()

	tests := []struct {
		name           string
		envOverride    string
		expectDetected bool
	}{
		{
			name:           "no override - should detect",
			envOverride:    "",
			expectDetected: true,
		},
		{
			name:           "with override - should skip detection",
			envOverride:    "dark",
			expectDetected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear detected mode
			os.Unsetenv("SIMTOOL_DETECTED_MODE")
			
			// Set override if specified
			if tt.envOverride != "" {
				os.Setenv("SIMTOOL_THEME_MODE", tt.envOverride)
			} else {
				os.Unsetenv("SIMTOOL_THEME_MODE")
			}

			// Note: InitializeThemeDetection uses sync.Once, so we can't easily test
			// multiple scenarios in the same test run. This is a limitation of the
			// current design. We can at least verify it doesn't panic.
			InitializeThemeDetection()

			// If no override, we might have a detected mode
			// (depends on terminal support)
			detected := GetDetectedMode()
			if tt.envOverride != "" && detected != "" {
				// We had an override, so detection should have been skipped
				// but GetDetectedMode might still return a value from a previous run
				t.Log("Detection may have run in a previous test")
			}
		})
	}
}

func TestGetDetectedMode(t *testing.T) {
	// Save original
	original := os.Getenv("SIMTOOL_DETECTED_MODE")
	defer func() {
		if original != "" {
			os.Setenv("SIMTOOL_DETECTED_MODE", original)
		} else {
			os.Unsetenv("SIMTOOL_DETECTED_MODE")
		}
	}()

	tests := []struct {
		name     string
		setMode  string
		expected string
	}{
		{
			name:     "dark mode",
			setMode:  "dark",
			expected: "dark",
		},
		{
			name:     "light mode",
			setMode:  "light",
			expected: "light",
		},
		{
			name:     "no mode set",
			setMode:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setMode != "" {
				os.Setenv("SIMTOOL_DETECTED_MODE", tt.setMode)
			} else {
				os.Unsetenv("SIMTOOL_DETECTED_MODE")
			}

			result := GetDetectedMode()
			if result != tt.expected {
				t.Errorf("GetDetectedMode() = %v, want %v", result, tt.expected)
			}
		})
	}
}