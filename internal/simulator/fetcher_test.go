package simulator

import (
	"encoding/json"
	"errors"
	"testing"
)

// mockFetcher implements the Fetcher interface for testing
type mockFetcher struct {
	simulators []Item
	err        error
	bootCalled bool
	bootUDID   string
	bootErr    error
}

func (m *mockFetcher) Fetch() ([]Item, error) {
	return m.simulators, m.err
}

func (m *mockFetcher) Boot(udid string) error {
	m.bootCalled = true
	m.bootUDID = udid
	return m.bootErr
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
				if items[0].UDID != "12345" {
					t.Errorf("Expected first UDID to be 12345, got %s", items[0].UDID)
				}
				if items[0].Runtime != "iOS 17.0" {
					t.Errorf("Expected first runtime to be iOS 17.0, got %s", items[0].Runtime)
				}
				if items[1].UDID != "67890" {
					t.Errorf("Expected second UDID to be 67890, got %s", items[1].UDID)
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

func TestFetcherInterface(t *testing.T) {
	// Test that mockFetcher implements the Fetcher interface
	var _ Fetcher = &mockFetcher{}

	// Test Fetch method
	mock := &mockFetcher{
		simulators: []Item{
			{
				Simulator: Simulator{
					UDID: "123",
					Name: "iPhone 15",
					State: "Booted",
				},
				Runtime: "iOS 17.0",
				AppCount: 0,
			},
		},
		err: nil,
	}

	items, err := mock.Fetch()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	// Test Fetch with error
	mock.err = errors.New("fetch error")
	_, err = mock.Fetch()
	if err == nil || err.Error() != "fetch error" {
		t.Errorf("Expected fetch error, got %v", err)
	}

	// Test Boot method
	mock.bootErr = nil
	err = mock.Boot("test-udid")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !mock.bootCalled {
		t.Error("Expected Boot to be called")
	}
	if mock.bootUDID != "test-udid" {
		t.Errorf("Expected boot UDID to be test-udid, got %s", mock.bootUDID)
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