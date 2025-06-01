package simulator

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Fetcher is responsible for fetching simulator information
type Fetcher interface {
	Fetch() ([]Item, error)
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
				items = append(items, Item{
					Simulator: sim,
					Runtime:   runtimeName,
				})
			}
		}
	}

	return items, nil
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