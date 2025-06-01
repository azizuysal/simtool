package simulator

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Fetcher is responsible for fetching simulator information
type Fetcher interface {
	Fetch() ([]Item, error)
	Boot(udid string) error
}

// SimctlFetcher fetches simulators using xcrun simctl
type SimctlFetcher struct{}

// NewFetcher creates a new simulator fetcher
func NewFetcher() Fetcher {
	return &SimctlFetcher{}
}

// Fetch retrieves all available iOS simulators
func (f *SimctlFetcher) Fetch() ([]Item, error) {
	cmd := exec.Command("xcrun", "simctl", "list", "devices", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run simctl: %w", err)
	}

	var simctlOutput SimctlOutput
	if err := json.Unmarshal(output, &simctlOutput); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var items []Item
	for runtime, sims := range simctlOutput.Devices {
		runtimeName := formatRuntime(runtime)
		for _, sim := range sims {
			if sim.IsAvailable {
				appCount := f.getAppCount(sim.UDID)
				items = append(items, Item{
					Simulator: sim,
					Runtime:   runtimeName,
					AppCount:  appCount,
				})
			}
		}
	}

	// Sort simulators by name
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

// Boot starts the simulator with the given UDID
func (f *SimctlFetcher) Boot(udid string) error {
	// First boot the simulator
	cmd := exec.Command("xcrun", "simctl", "boot", udid)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already booted
		if strings.Contains(string(output), "Unable to boot device in current state: Booted") {
			// If already booted, just open the Simulator app
			return f.openSimulatorApp()
		}
		return fmt.Errorf("failed to boot simulator: %s", string(output))
	}
	
	// Open the Simulator app to show the UI
	return f.openSimulatorApp()
}

// openSimulatorApp opens the Simulator application
func (f *SimctlFetcher) openSimulatorApp() error {
	cmd := exec.Command("open", "-a", "Simulator")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open Simulator app: %w", err)
	}
	return nil
}

// getAppCount returns the number of installed apps on a simulator
func (f *SimctlFetcher) getAppCount(udid string) int {
	// Only count apps for booted simulators
	cmd := exec.Command("xcrun", "simctl", "listapps", udid)
	output, err := cmd.Output()
	if err != nil {
		// If error (e.g., simulator not booted), return -1 to indicate unknown
		return -1
	}

	// Parse the plist-style output
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	
	userAppCount := 0
	for _, line := range lines {
		// Look for CFBundleIdentifier lines
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CFBundleIdentifier = ") {
			bundleID := strings.Trim(strings.TrimPrefix(line, "CFBundleIdentifier = "), `";`)
			// Count non-Apple apps
			if !strings.HasPrefix(bundleID, "com.apple.") {
				userAppCount++
			}
		}
	}

	return userAppCount
}

// formatRuntime converts runtime identifier to user-friendly format
func formatRuntime(runtime string) string {
	// Remove prefix
	runtimeName := strings.Replace(runtime, "com.apple.CoreSimulator.SimRuntime.", "", 1)
	// Format iOS versions
	runtimeName = strings.Replace(runtimeName, "iOS-", "iOS ", 1)
	runtimeName = strings.Replace(runtimeName, "-", ".", -1)
	return runtimeName
}