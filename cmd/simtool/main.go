package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
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
		if err := printThemes(os.Stdout); err != nil {
			log.Printf("failed to print themes: %s", err)
			os.Exit(1)
		}
		return
	}

	// Create simulator fetcher
	fetcher := simulator.NewFetcher()

	// Validate configuration and construct the TUI before creating the debug log.
	model, err := tui.New(fetcher, startWithApps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up debug logging. The file goes under the user cache directory
	// (e.g. ~/Library/Caches/simtool/debug.log on macOS) rather than the
	// process working directory, which would pollute wherever the user
	// invoked simtool and leak TUI state into unrelated project trees.
	logPath, err := debugLogPath()
	if err != nil {
		log.Fatalf("failed to resolve debug log path: %s", err)
	}
	f, err := tea.LogToFile(logPath, "debug")
	if err != nil {
		log.Fatalf("failed to create debug log: %s", err)
	}

	// Run the TUI application.
	p := tea.NewProgram(model)

	_, runErr := p.Run()
	_ = f.Close()
	if runErr != nil {
		log.Printf("Error running program: %s", runErr)
		os.Exit(1)
	}
}

func printThemes(w io.Writer) error {
	available := make(map[string]struct{}, len(styles.Names()))
	for _, theme := range styles.Names() {
		available[theme] = struct{}{}
	}

	categories := []struct {
		name   string
		themes []string
	}{
		{"Dark themes", []string{"monokai", "dracula", "github-dark", "nord", "onedark", "solarized-dark", "gruvbox", "vim", "paraiso-dark"}},
		{"Light themes", []string{"github", "solarized-light", "gruvbox-light", "tango", "monokailight", "paraiso-light", "pygments"}},
		{"High contrast", []string{"contrast", "fruity", "native"}},
	}

	lines := []string{"Available syntax highlighting themes:", ""}
	printed := make(map[string]struct{})
	for _, category := range categories {
		lines = append(lines, category.name+":")
		for _, theme := range category.themes {
			if _, ok := available[theme]; ok {
				lines = append(lines, "  - "+theme)
				printed[theme] = struct{}{}
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, "Other themes:")
	for _, theme := range styles.Names() {
		if _, ok := printed[theme]; !ok {
			lines = append(lines, "  - "+theme)
		}
	}

	lines = append(lines,
		"",
		"To use a theme, add it to your config file:",
		"[theme]",
		"dark_theme = \"theme-name\"",
		"light_theme = \"theme-name\"",
	)
	_, err := io.WriteString(w, strings.Join(lines, "\n")+"\n")
	return err
}

// debugLogPath returns the path for simtool's debug log file, ensuring
// the parent directory exists with user-only permissions.
func debugLogPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir, appName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "debug.log"), nil
}
