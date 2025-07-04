package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration
type Config struct {
	Theme   ThemeConfig   `toml:"theme"`
	Keys    KeysConfig    `toml:"keys"`
	Startup StartupConfig `toml:"startup"`
	// Future: Display options, etc.
}

// ThemeConfig defines theme configuration
type ThemeConfig struct {
	// Force dark or light mode ("dark", "light", or "auto")
	Mode string `toml:"mode"`
	
	// Themes for each mode
	DarkTheme  string `toml:"dark_theme"`  // Theme to use in dark mode
	LightTheme string `toml:"light_theme"` // Theme to use in light mode
}

// StartupConfig defines startup behavior
type StartupConfig struct {
	// Initial view to show on startup: "simulator_list" (default) or "all_apps"
	InitialView string `toml:"initial_view"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Theme: ThemeConfig{
			Mode:       "auto",
			DarkTheme:  "github-dark",
			LightTheme: "github",
		},
		Keys: DefaultKeys(),
		Startup: StartupConfig{
			InitialView: "simulator_list",
		},
	}
}

// Load loads configuration from the standard config path
func Load() (*Config, error) {
	cfg := Default()
	
	// Get config path
	configPath, err := getConfigPath()
	if err != nil {
		return cfg, fmt.Errorf("getting config path: %w", err)
	}
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No user config, return defaults
		return cfg, nil
	}
	
	// Load user config
	userCfg := &Config{}
	if _, err := toml.DecodeFile(configPath, userCfg); err != nil {
		return cfg, fmt.Errorf("decoding config file: %w", err)
	}
	
	// Merge user config with defaults
	cfg.merge(userCfg)
	
	return cfg, nil
}

// SaveExample saves an example configuration file
func SaveExample() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("getting config dir: %w", err)
	}
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	
	examplePath := filepath.Join(configDir, "config.example.toml")
	
	file, err := os.Create(examplePath)
	if err != nil {
		return fmt.Errorf("creating example file: %w", err)
	}
	defer func() { _ = file.Close() }()
	
	// Write example with comments
	example := `# SimTool Configuration File
# Copy this file to config.toml and customize as needed

[theme]
# Theme mode: "auto" (default), "dark", or "light"
# - "auto": Automatically detect based on terminal
# - "dark": Always use dark theme
# - "light": Always use light theme
mode = "auto"

# Theme for dark mode (or when mode="dark")
dark_theme = "github-dark"

# Theme for light mode (or when mode="light")  
light_theme = "github"

# Run 'simtool --list-themes' to see all available themes

# Popular dark themes:
# - "github-dark" (default) - GitHub's official dark theme
# - "dracula" - popular dark theme with purple accents  
# - "monokai" - vibrant colors on dark background
# - "nord" - arctic, north-bluish color palette
# - "onedark" - Atom One Dark theme
# - "solarized-dark" - precision colors for machines and people
# - "gruvbox" - retro groove color scheme

# Popular light themes:
# - "github" (default) - GitHub's light theme
# - "solarized-light" - light variant of solarized
# - "gruvbox-light" - light variant of gruvbox
# - "tango" - GNOME Tango theme
# - "vs" - Visual Studio theme

[startup]
# Initial view to show when app starts
# Options: "simulator_list" (default) or "all_apps"
initial_view = "simulator_list"

# Set to "all_apps" to start with all apps from all simulators
# This is equivalent to using the --apps/-a command-line flag

[keys]
# Keyboard shortcuts configuration
# Each action can have multiple keys assigned
# Use key names as recognized by Bubble Tea (https://github.com/charmbracelet/bubbletea)

# Navigation keys
up = ["up", "k"]           # Move cursor up
down = ["down", "j"]       # Move cursor down
left = ["left", "h"]       # Go back / navigate left
right = ["right", "l"]     # Enter / navigate right
home = ["home"]            # Jump to first item
end = ["end"]              # Jump to last item

# Action keys
quit = ["q", "ctrl+c"]     # Quit the application
boot = [" "]               # Boot simulator (space key)
open = [" "]               # Open in Finder (space key, context-dependent)
filter = ["f"]             # Toggle filter (simulator list only)
search = ["/"]             # Start search mode
escape = ["esc"]           # Exit search mode / cancel
enter = ["enter"]          # Select / confirm

# Search mode keys
backspace = ["backspace"]  # Delete character in search

# Examples of customization:
# - Use only arrow keys: up = ["up"], down = ["down"]
# - Use only vim keys: up = ["k"], down = ["j"]
# - Add custom keys: quit = ["q", "ctrl+c", "ctrl+d"]
# - Disable a shortcut: filter = []
`
	
	if _, err := file.WriteString(example); err != nil {
		return fmt.Errorf("writing example file: %w", err)
	}
	
	return nil
}

// GetActiveTheme returns the active theme name based on config and terminal mode
func (c *Config) GetActiveTheme() string {
	switch c.Theme.Mode {
	case "dark":
		return c.Theme.DarkTheme
	case "light":
		return c.Theme.LightTheme
	case "auto":
		fallthrough
	default:
		// Auto-detect terminal mode using live detection for dynamic switching
		if DetectTerminalDarkModeLive() {
			return c.Theme.DarkTheme
		}
		return c.Theme.LightTheme
	}
}

// merge merges user config into the default config
func (c *Config) merge(user *Config) {
	// Merge theme settings
	if user.Theme.Mode != "" {
		c.Theme.Mode = user.Theme.Mode
	}
	if user.Theme.DarkTheme != "" {
		c.Theme.DarkTheme = user.Theme.DarkTheme
	}
	if user.Theme.LightTheme != "" {
		c.Theme.LightTheme = user.Theme.LightTheme
	}
	
	// Merge startup settings
	if user.Startup.InitialView != "" {
		c.Startup.InitialView = user.Startup.InitialView
	}
	
	// Merge key settings - only override if user has specified keys
	if len(user.Keys.Up) > 0 {
		c.Keys.Up = user.Keys.Up
	}
	if len(user.Keys.Down) > 0 {
		c.Keys.Down = user.Keys.Down
	}
	if len(user.Keys.Left) > 0 {
		c.Keys.Left = user.Keys.Left
	}
	if len(user.Keys.Right) > 0 {
		c.Keys.Right = user.Keys.Right
	}
	if len(user.Keys.Home) > 0 {
		c.Keys.Home = user.Keys.Home
	}
	if len(user.Keys.End) > 0 {
		c.Keys.End = user.Keys.End
	}
	if len(user.Keys.Quit) > 0 {
		c.Keys.Quit = user.Keys.Quit
	}
	if len(user.Keys.Boot) > 0 {
		c.Keys.Boot = user.Keys.Boot
	}
	if len(user.Keys.Open) > 0 {
		c.Keys.Open = user.Keys.Open
	}
	if len(user.Keys.Filter) > 0 {
		c.Keys.Filter = user.Keys.Filter
	}
	if len(user.Keys.Search) > 0 {
		c.Keys.Search = user.Keys.Search
	}
	if len(user.Keys.Escape) > 0 {
		c.Keys.Escape = user.Keys.Escape
	}
	if len(user.Keys.Enter) > 0 {
		c.Keys.Enter = user.Keys.Enter
	}
	if len(user.Keys.Backspace) > 0 {
		c.Keys.Backspace = user.Keys.Backspace
	}
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "simtool"), nil
	}
	
	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	
	return filepath.Join(home, ".config", "simtool"), nil
}

// getConfigPath returns the configuration file path
func getConfigPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	
	return filepath.Join(configDir, "config.toml"), nil
}