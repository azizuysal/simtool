package simulator

import (
	"encoding/json"
	"fmt"
	"testing"
)

// MockCommandExecutor implements CommandExecutor for testing
type MockCommandExecutor struct {
	ExecuteFunc func(name string, args ...string) ([]byte, error)
	RunFunc     func(name string, args ...string) error
}

func (m *MockCommandExecutor) Execute(name string, args ...string) ([]byte, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(name, args...)
	}
	return nil, nil
}

func (m *MockCommandExecutor) Run(name string, args ...string) error {
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return nil
}

func TestSimctlFetcher_Fetch(t *testing.T) {
	mockExecutor := &MockCommandExecutor{}
	fetcher := NewFetcherWithExecutor(mockExecutor)

	// Mock successful fetch
	mockExecutor.ExecuteFunc = func(name string, args ...string) ([]byte, error) {
		if name == "xcrun" && args[0] == "simctl" && args[1] == "list" && args[2] == "devices" {
			output := SimctlOutput{
				Devices: map[string][]Simulator{
					"iOS 17.0": {
						{
							UDID:        "123",
							Name:        "iPhone 15",
							State:       "Booted",
							IsAvailable: true,
						},
					},
				},
			}
			return json.Marshal(output)
		}
		if name == "xcrun" && args[0] == "simctl" && args[1] == "listapps" {
			return []byte(`CFBundleIdentifier = "com.example.app";`), nil
		}
		return nil, fmt.Errorf("unexpected command: %s %v", name, args)
	}

	items, err := fetcher.Fetch()
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}
	if items[0].Name != "iPhone 15" {
		t.Errorf("Expected name iPhone 15, got %s", items[0].Name)
	}
	if items[0].AppCount != 1 {
		t.Errorf("Expected 1 app, got %d", items[0].AppCount)
	}
}

func TestSimctlFetcher_Boot(t *testing.T) {
	mockExecutor := &MockCommandExecutor{}
	fetcher := NewFetcherWithExecutor(mockExecutor)

	bootCalled := false
	openCalled := false

	mockExecutor.ExecuteFunc = func(name string, args ...string) ([]byte, error) {
		if name == "xcrun" && args[0] == "simctl" && args[1] == "boot" && args[2] == "123" {
			bootCalled = true
			return []byte{}, nil
		}
		return nil, fmt.Errorf("unexpected command: %s %v", name, args)
	}

	mockExecutor.RunFunc = func(name string, args ...string) error {
		if name == "open" && args[0] == "-a" && args[1] == "Simulator" {
			openCalled = true
			return nil
		}
		return fmt.Errorf("unexpected command: %s %v", name, args)
	}

	err := fetcher.Boot("123")
	if err != nil {
		t.Errorf("Boot() error = %v", err)
	}

	if !bootCalled {
		t.Error("Expected boot command to be called")
	}
	if !openCalled {
		t.Error("Expected open command to be called")
	}
}

func TestParseSimulatorJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantLen int
		wantErr bool
		check   func(t *testing.T, items []Item)
	}{
		{
			name: "valid JSON with simulators",
			input: []byte(`{
				"devices": {
					"iOS 17.0": [
						{
							"udid": "12345",
							"name": "iPhone 15",
							"state": "Booted",
							"isAvailable": true,
							"deviceTypeIdentifier": "com.apple.CoreSimulator.SimDeviceType.iPhone-15"
						}
					],
					"iOS 16.4": [
						{
							"udid": "67890",
							"name": "iPhone 14",
							"state": "Shutdown",
							"isAvailable": true,
							"deviceTypeIdentifier": "com.apple.CoreSimulator.SimDeviceType.iPhone-14"
						}
					]
				}
			}`),
			wantLen: 2,
			wantErr: false,
			check: func(t *testing.T, items []Item) {
				// Items might be in any order, so check both exist
				var found12345, found67890 bool
				for _, item := range items {
					if item.UDID == "12345" {
						found12345 = true
						if item.Runtime != "iOS 17.0" {
							t.Errorf("Expected runtime for UDID 12345 to be iOS 17.0, got %s", item.Runtime)
						}
					}
					if item.UDID == "67890" {
						found67890 = true
						if item.Runtime != "iOS 16.4" {
							t.Errorf("Expected runtime for UDID 67890 to be iOS 16.4, got %s", item.Runtime)
						}
					}
				}
				if !found12345 {
					t.Error("UDID 12345 not found in results")
				}
				if !found67890 {
					t.Error("UDID 67890 not found in results")
				}
			},
		},
		{
			name: "skip unavailable simulators",
			input: []byte(`{
				"devices": {
					"iOS 17.0": [
						{
							"udid": "12345",
							"name": "iPhone 15",
							"state": "Booted",
							"isAvailable": false
						},
						{
							"udid": "67890",
							"name": "iPhone 14",
							"state": "Shutdown",
							"isAvailable": true
						}
					]
				}
			}`),
			wantLen: 1,
			wantErr: false,
			check: func(t *testing.T, items []Item) {
				if items[0].UDID != "67890" {
					t.Errorf("Expected UDID to be 67890, got %s", items[0].UDID)
				}
			},
		},
		{
			name: "skip non-iOS runtimes",
			input: []byte(`{
				"devices": {
					"watchOS 10.0": [
						{
							"udid": "12345",
							"name": "Apple Watch",
							"state": "Booted",
							"isAvailable": true
						}
					],
					"iOS 17.0": [
						{
							"udid": "67890",
							"name": "iPhone 15",
							"state": "Shutdown",
							"isAvailable": true
						}
					]
				}
			}`),
			wantLen: 1,
			wantErr: false,
			check: func(t *testing.T, items []Item) {
				if items[0].Runtime != "iOS 17.0" {
					t.Errorf("Expected runtime to be iOS 17.0, got %s", items[0].Runtime)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{invalid json`),
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "empty devices",
			input:   []byte(`{"devices": {}}`),
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := parseSimulatorJSON(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseSimulatorJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(items) != tt.wantLen {
				t.Errorf("parseSimulatorJSON() returned %d items, want %d", len(items), tt.wantLen)
			}

			if tt.check != nil && !tt.wantErr {
				tt.check(t, items)
			}
		})
	}
}

func TestParseRuntimeVersion(t *testing.T) {
	tests := []struct {
		name     string
		runtime  string
		expected string
	}{
		{
			name:     "iOS version",
			runtime:  "iOS 17.0",
			expected: "17.0",
		},
		{
			name:     "iOS version with patch",
			runtime:  "iOS 16.4.1",
			expected: "16.4.1",
		},
		{
			name:     "watchOS version",
			runtime:  "watchOS 10.0",
			expected: "10.0",
		},
		{
			name:     "invalid format",
			runtime:  "Invalid",
			expected: "Invalid",
		},
		{
			name:     "empty string",
			runtime:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRuntimeVersion(tt.runtime)
			if result != tt.expected {
				t.Errorf("parseRuntimeVersion(%s) = %v, want %v", tt.runtime, result, tt.expected)
			}
		})
	}
}

func TestJSONMarshaling(t *testing.T) {
	// Test that our structures can be properly marshaled/unmarshaled
	original := SimctlOutput{
		Devices: map[string][]Simulator{
			"iOS 17.0": {
				{
					UDID:                 "12345",
					Name:                 "iPhone 15",
					State:                "Booted",
					IsAvailable:          true,
					DeviceTypeIdentifier: "com.apple.CoreSimulator.SimDeviceType.iPhone-15",
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SimctlOutput
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Devices) != 1 {
		t.Errorf("Expected 1 runtime, got %d", len(decoded.Devices))
	}

	devices, ok := decoded.Devices["iOS 17.0"]
	if !ok {
		t.Fatal("iOS 17.0 runtime not found")
	}

	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	if devices[0].UDID != "12345" {
		t.Errorf("Expected UDID 12345, got %s", devices[0].UDID)
	}
}
