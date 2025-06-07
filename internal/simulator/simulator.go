package simulator

import (
	"fmt"
	"sort"
)

// Simulator represents an iOS simulator device
type Simulator struct {
	UDID                 string `json:"udid"`
	Name                 string `json:"name"`
	State                string `json:"state"`
	IsAvailable          bool   `json:"isAvailable"`
	DeviceTypeIdentifier string `json:"deviceTypeIdentifier"`
}

// Item represents a simulator with its runtime information
type Item struct {
	Simulator
	Runtime  string
	AppCount int
}

// DevicesByRuntime maps runtime identifiers to simulators
type DevicesByRuntime map[string][]Simulator

// SimctlOutput represents the JSON output from simctl
type SimctlOutput struct {
	Devices DevicesByRuntime `json:"devices"`
}

// IsRunning returns true if the simulator is booted
func (s *Simulator) IsRunning() bool {
	return s.State == "Booted"
}

// StateDisplay returns a user-friendly state description
func (s *Simulator) StateDisplay() string {
	switch s.State {
	case "Shutdown":
		return "Not Running"
	case "Booted":
		return "Running"
	default:
		return s.State
	}
}

// Common errors
var (
	ErrSimulatorNotFound = fmt.Errorf("simulator not found")
)

// GetAllApps returns all apps from all simulators with simulator info populated
func GetAllApps(fetcher Fetcher) ([]App, error) {
	items, err := fetcher.Fetch()
	if err != nil {
		return nil, fmt.Errorf("fetching simulators: %w", err)
	}
	
	allApps := make([]App, 0)
	
	for _, item := range items {
		apps, err := GetAppsForSimulator(item.UDID, item.IsRunning())
		if err != nil {
			// Skip simulators with errors
			continue
		}
		
		// Add simulator info to each app
		for i := range apps {
			apps[i].SimulatorName = item.Name
			apps[i].SimulatorUDID = item.UDID
		}
		
		allApps = append(allApps, apps...)
	}
	
	// Sort all apps by name, then by simulator name
	sort.Slice(allApps, func(i, j int) bool {
		if allApps[i].Name != allApps[j].Name {
			return allApps[i].Name < allApps[j].Name
		}
		return allApps[i].SimulatorName < allApps[j].SimulatorName
	})
	
	return allApps, nil
}