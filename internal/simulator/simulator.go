package simulator

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