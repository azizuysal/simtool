package simulator

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseAppListJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    []App
		wantErr bool
	}{
		{
			name: "valid app list",
			input: []byte(`{
				"com.example.app1": {
					"ApplicationType": "User",
					"Bundle": "/path/to/app1.app",
					"BundleContainer": "/path/to/container1",
					"CFBundleDisplayName": "App One",
					"CFBundleIdentifier": "com.example.app1",
					"CFBundleName": "AppOne",
					"CFBundleShortVersionString": "1.0.0",
					"DataContainer": "/path/to/data1",
					"Path": "/path/to/app1.app"
				},
				"com.example.app2": {
					"ApplicationType": "User",
					"Bundle": "/path/to/app2.app",
					"BundleContainer": "/path/to/container2",
					"CFBundleDisplayName": "App Two",
					"CFBundleIdentifier": "com.example.app2",
					"CFBundleName": "AppTwo",
					"CFBundleShortVersionString": "2.0.0",
					"DataContainer": "/path/to/data2",
					"Path": "/path/to/app2.app"
				}
			}`),
			want: []App{
				{
					Name:      "App One",
					BundleID:  "com.example.app1",
					Version:   "1.0.0",
					Path:      "/path/to/app1.app",
					Container: "/path/to/data1",
					Size:      0,
				},
				{
					Name:      "App Two",
					BundleID:  "com.example.app2",
					Version:   "2.0.0",
					Path:      "/path/to/app2.app",
					Container: "/path/to/data2",
					Size:      0,
				},
			},
			wantErr: false,
		},
		{
			name: "skip system apps",
			input: []byte(`{
				"com.apple.systemapp": {
					"ApplicationType": "System",
					"CFBundleDisplayName": "System App",
					"CFBundleIdentifier": "com.apple.systemapp"
				},
				"com.example.userapp": {
					"ApplicationType": "User",
					"CFBundleDisplayName": "User App",
					"CFBundleIdentifier": "com.example.userapp",
					"CFBundleShortVersionString": "1.0",
					"DataContainer": "/path/to/data"
				}
			}`),
			want: []App{
				{
					Name:      "User App",
					BundleID:  "com.example.userapp",
					Version:   "1.0",
					Container: "/path/to/data",
					Size:      0,
				},
			},
			wantErr: false,
		},
		{
			name: "handle missing display name",
			input: []byte(`{
				"com.example.app": {
					"ApplicationType": "User",
					"CFBundleName": "AppName",
					"CFBundleIdentifier": "com.example.app",
					"DataContainer": "/path/to/data"
				}
			}`),
			want: []App{
				{
					Name:      "AppName",
					BundleID:  "com.example.app",
					Version:   "",
					Container: "/path/to/data",
					Size:      0,
				},
			},
			wantErr: false,
		},
		{
			name: "handle missing bundle name",
			input: []byte(`{
				"com.example.app": {
					"ApplicationType": "User",
					"CFBundleIdentifier": "com.example.app",
					"DataContainer": "/path/to/data"
				}
			}`),
			want: []App{
				{
					Name:      "com.example.app",
					BundleID:  "com.example.app",
					Version:   "",
					Container: "/path/to/data",
					Size:      0,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{invalid json`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty app list",
			input:   []byte(`{}`),
			want:    []App{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAppListJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAppListJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// Sort both slices by BundleID for consistent comparison
			sortAppsByBundleID(got)
			sortAppsByBundleID(tt.want)
			
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAppListJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppInfoMarshaling(t *testing.T) {
	// Test that appInfo structure can be properly marshaled/unmarshaled
	original := map[string]appInfo{
		"com.example.app": {
			ApplicationType:             "User",
			Bundle:                      "/path/to/app.app",
			BundleContainer:             "/path/to/container",
			CFBundleDisplayName:         "Example App",
			CFBundleIdentifier:          "com.example.app",
			CFBundleName:                "ExampleApp",
			CFBundleShortVersionString:  "1.2.3",
			DataContainer:               "/path/to/data",
			Path:                        "/path/to/app.app",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]appInfo
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded) != 1 {
		t.Errorf("Expected 1 app, got %d", len(decoded))
	}

	app, ok := decoded["com.example.app"]
	if !ok {
		t.Fatal("App not found in decoded data")
	}

	if app.CFBundleDisplayName != "Example App" {
		t.Errorf("Expected display name 'Example App', got %s", app.CFBundleDisplayName)
	}

	if app.CFBundleShortVersionString != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got %s", app.CFBundleShortVersionString)
	}
}

// Helper function to sort apps by BundleID for testing
func sortAppsByBundleID(apps []App) {
	for i := 0; i < len(apps)-1; i++ {
		for j := i + 1; j < len(apps); j++ {
			if apps[i].BundleID > apps[j].BundleID {
				apps[i], apps[j] = apps[j], apps[i]
			}
		}
	}
}