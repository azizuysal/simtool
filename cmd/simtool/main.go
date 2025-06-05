package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/alecthomas/chroma/v2/styles"
	"simtool/internal/config"
	"simtool/internal/simulator"
	"simtool/internal/tui"
)

func main() {
	// Detect terminal theme before starting TUI
	config.InitializeThemeDetection()
	
	// Define command-line flags
	var (
		generateConfig = flag.Bool("generate-config", false, "Generate example configuration file")
		showConfigPath = flag.Bool("show-config-path", false, "Show configuration file path")
		listThemes     = flag.Bool("list-themes", false, "List available syntax highlighting themes")
	)
	
	flag.Parse()
	
	// Handle config-related flags
	if *generateConfig {
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
	
	if *showConfigPath {
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, ".config")
		}
		configPath := filepath.Join(configDir, "simtool", "config.toml")
		fmt.Printf("Configuration file path: %s\n", configPath)
		return
	}
	
	if *listThemes {
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
	defer f.Close()

	// Create simulator fetcher
	fetcher := simulator.NewFetcher()

	// Create and run the TUI application
	model := tui.New(fetcher)
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %s", err)
		os.Exit(1)
	}
}