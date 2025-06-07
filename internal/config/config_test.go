package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	
	if cfg.Theme.Mode != "auto" {
		t.Errorf("Default theme mode should be 'auto', got %q", cfg.Theme.Mode)
	}
	
	if cfg.Theme.DarkTheme != "github-dark" {
		t.Errorf("Default dark theme should be 'github-dark', got %q", cfg.Theme.DarkTheme)
	}
	
	if cfg.Theme.LightTheme != "github" {
		t.Errorf("Default light theme should be 'github', got %q", cfg.Theme.LightTheme)
	}
	
	if cfg.Startup.InitialView != "simulator_list" {
		t.Errorf("Default initial view should be 'simulator_list', got %q", cfg.Startup.InitialView)
	}
}

func TestGetActiveTheme(t *testing.T) {
	// Save original env var
	originalOverride := os.Getenv("SIMTOOL_THEME_MODE")
	originalDetected := os.Getenv("SIMTOOL_DETECTED_MODE")
	defer func() {
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
		name         string
		mode         string
		envOverride  string
		envDetected  string
		darkTheme    string
		lightTheme   string
		expected     string
	}{
		{
			name:       "explicit dark mode",
			mode:       "dark",
			darkTheme:  "custom-dark",
			lightTheme: "custom-light",
			expected:   "custom-dark",
		},
		{
			name:       "explicit light mode",
			mode:       "light",
			darkTheme:  "custom-dark",
			lightTheme: "custom-light",
			expected:   "custom-light",
		},
		{
			name:        "auto mode with env override dark",
			mode:        "auto",
			envOverride: "dark",
			darkTheme:   "custom-dark",
			lightTheme:  "custom-light",
			expected:    "custom-dark",
		},
		{
			name:        "auto mode with env override light",
			mode:        "auto",
			envOverride: "light",
			darkTheme:   "custom-dark",
			lightTheme:  "custom-light",
			expected:    "custom-light",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envOverride != "" {
				os.Setenv("SIMTOOL_THEME_MODE", tt.envOverride)
			} else {
				os.Unsetenv("SIMTOOL_THEME_MODE")
			}
			
			if tt.envDetected != "" {
				os.Setenv("SIMTOOL_DETECTED_MODE", tt.envDetected)
			} else {
				os.Unsetenv("SIMTOOL_DETECTED_MODE")
			}

			cfg := &Config{
				Theme: ThemeConfig{
					Mode:       tt.mode,
					DarkTheme:  tt.darkTheme,
					LightTheme: tt.lightTheme,
				},
			}

			result := cfg.GetActiveTheme()
			if result != tt.expected {
				t.Errorf("GetActiveTheme() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "simtool")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Save original HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	t.Run("no config file - use defaults", func(t *testing.T) {
		cfg, err := Load()
		if err != nil {
			t.Errorf("Load() with no config file should not error: %v", err)
		}
		
		// Should have default values
		if cfg.Theme.Mode != "auto" {
			t.Errorf("Expected default theme mode 'auto', got %q", cfg.Theme.Mode)
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		// Create a config file
		configPath := filepath.Join(configDir, "config.toml")
		configContent := `[theme]
mode = "dark"
dark_theme = "dracula"
light_theme = "solarized-light"

[startup]
initial_view = "apps"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		cfg, err := Load()
		if err != nil {
			t.Errorf("Load() with valid config file should not error: %v", err)
		}
		
		if cfg.Theme.Mode != "dark" {
			t.Errorf("Expected theme mode 'dark', got %q", cfg.Theme.Mode)
		}
		if cfg.Theme.DarkTheme != "dracula" {
			t.Errorf("Expected dark theme 'dracula', got %q", cfg.Theme.DarkTheme)
		}
		if cfg.Theme.LightTheme != "solarized-light" {
			t.Errorf("Expected light theme 'solarized-light', got %q", cfg.Theme.LightTheme)
		}
		if cfg.Startup.InitialView != "apps" {
			t.Errorf("Expected initial view 'apps', got %q", cfg.Startup.InitialView)
		}
	})

	t.Run("invalid config file", func(t *testing.T) {
		// Create an invalid config file
		configPath := filepath.Join(configDir, "config.toml")
		err := os.WriteFile(configPath, []byte("invalid toml content {{{"), 0644)
		if err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		_, err = Load()
		if err == nil {
			t.Error("Load() with invalid config file should error")
		}
	})
}