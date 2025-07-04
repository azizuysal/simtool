package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
	"github.com/azizuysal/simtool/internal/tui"
)

const appName = "simtool"

// Build variables - these are set via ldflags during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	// Detect terminal theme before starting TUI
	config.InitializeThemeDetection()
	
	// Define command-line flags
	var (
		generateConfig bool
		showConfigPath bool
		listThemes     bool
		showHelp       bool
		showVersion    bool
		startWithApps  bool
	)
	
	flag.BoolVar(&generateConfig, "generate-config", false, "Generate example configuration file")
	flag.BoolVar(&generateConfig, "g", false, "Generate example configuration file")
	
	flag.BoolVar(&showConfigPath, "show-config-path", false, "Show configuration file path")
	flag.BoolVar(&showConfigPath, "c", false, "Show configuration file path")
	
	flag.BoolVar(&listThemes, "list-themes", false, "List available syntax highlighting themes")
	flag.BoolVar(&listThemes, "l", false, "List available syntax highlighting themes")
	
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information")
	
	flag.BoolVar(&startWithApps, "apps", false, "Start with all apps view instead of simulator list")
	flag.BoolVar(&startWithApps, "a", false, "Start with all apps view instead of simulator list")
	
	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", appName)
		fmt.Fprintf(os.Stderr, "A terminal UI application for managing iOS simulators on macOS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -a, --apps                Start with all apps view instead of simulator list\n")
		fmt.Fprintf(os.Stderr, "  -g, --generate-config     Generate example configuration file\n")
		fmt.Fprintf(os.Stderr, "  -c, --show-config-path    Show configuration file path\n")
		fmt.Fprintf(os.Stderr, "  -l, --list-themes         List available syntax highlighting themes\n")
		fmt.Fprintf(os.Stderr, "  -h, --help                Show help message\n")
		fmt.Fprintf(os.Stderr, "  -v, --version             Show version information\n")
	}
	
	flag.Parse()
	
	// Handle help flag
	if showHelp {
		flag.Usage()
		return
	}
	
	// Handle version flag
	if showVersion {
		fmt.Printf("%s version %s\n", appName, version)
		if commit != "none" {
			fmt.Printf("  commit: %s\n", commit)
		}
		if date != "unknown" {
			fmt.Printf("  built:  %s\n", date)
		}
		if builtBy != "unknown" {
			fmt.Printf("  by:     %s\n", builtBy)
		}
		return
	}
	
	// Handle config-related flags
	if generateConfig {
		if err := config.SaveExample(); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating config: %v\n", err)
			os.Exit(1)
		}
		
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(configDir, "simtool")
		
		fmt.Printf("Example configuration file created at: %s/config.example.toml\n", configDir)
		fmt.Println("Copy it to config.toml and customize as needed.")
		return
	}
	
	if showConfigPath {
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, ".config")
		}
		configPath := filepath.Join(configDir, "simtool", "config.toml")
		fmt.Printf("Configuration file path: %s\n", configPath)
		return
	}
	
	if listThemes {
		fmt.Println("Available syntax highlighting themes:")
		fmt.Println()
		
		themes := styles.Names()
		categories := map[string][]string{
			"Dark themes": {
				"monokai", "dracula", "github-dark", "nord", "onedark", 
				"solarized-dark", "gruvbox", "vim", "paraiso-dark",
			},
			"Light themes": {
				"github", "solarized-light", "gruvbox-light", "tango",
				"monokailight", "paraiso-light", "pygments",
			},
			"High contrast": {
				"contrast", "fruity", "native",
			},
		}
		
		// Print categorized themes
		for category, themeList := range categories {
			fmt.Printf("%s:\n", category)
			for _, theme := range themeList {
				// Check if theme exists
				for _, available := range themes {
					if available == theme {
						fmt.Printf("  - %s\n", theme)
						break
					}
				}
			}
			fmt.Println()
		}
		
		// Print all other themes
		fmt.Println("Other themes:")
		printed := make(map[string]bool)
		for _, category := range categories {
			for _, theme := range category {
				printed[theme] = true
			}
		}
		
		for _, theme := range themes {
			if !printed[theme] {
				fmt.Printf("  - %s\n", theme)
			}
		}
		
		fmt.Println("\nTo use a theme, add it to your config file:")
		fmt.Println("[syntax]")
		fmt.Println("theme = \"theme-name\"")
		return
	}
	
	// Set up debug logging
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("failed to create debug log: %s", err)
	}
	defer func() { _ = f.Close() }()

	// Create simulator fetcher
	fetcher := simulator.NewFetcher()

	// Create and run the TUI application
	model := tui.New(fetcher, startWithApps)
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %s", err)
		os.Exit(1)
	}
}