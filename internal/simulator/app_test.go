package simulator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// setupSimulatorHome builds the expected HOME-rooted data-dir layout
// for a simulator and returns the bundle-root path where app
// directories live.
func setupSimulatorHome(t *testing.T, udid string) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	bundleRoot := filepath.Join(home,
		"Library/Developer/CoreSimulator/Devices", udid,
		"data/Containers/Bundle/Application")
	if err := os.MkdirAll(bundleRoot, 0750); err != nil {
		t.Fatalf("mkdir bundleRoot: %v", err)
	}
	return bundleRoot
}

func TestGetAppsFromDataDir_MissingDeviceReturnsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	apps, err := getAppsFromDataDir("NO-SUCH-DEVICE")
	if err != nil {
		t.Fatalf("getAppsFromDataDir: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("len(apps) = %d, want 0 for missing device", len(apps))
	}
}

func TestGetAppsFromDataDir_EmptyDir(t *testing.T) {
	_ = setupSimulatorHome(t, "UDID-EMPTY")

	apps, err := getAppsFromDataDir("UDID-EMPTY")
	if err != nil {
		t.Fatalf("getAppsFromDataDir: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("len(apps) = %d, want 0 for empty dir", len(apps))
	}
}

func TestGetAppsFromDataDir_WithAppUsesInfoPlist(t *testing.T) {
	udid := "UDID-APPS"
	bundleRoot := setupSimulatorHome(t, udid)

	// Lay out one app bundle.
	appContainer := filepath.Join(bundleRoot, "app-uuid-1")
	appBundle := filepath.Join(appContainer, "Example.app")
	if err := os.MkdirAll(appBundle, 0750); err != nil {
		t.Fatalf("mkdir app bundle: %v", err)
	}
	// Write a dummy file inside the bundle so size calculation is nonzero.
	if err := os.WriteFile(filepath.Join(appBundle, "binary"), []byte("0123456789"), 0600); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	// The Info.plist file itself isn't parsed by our code — plutil is —
	// so it's enough to create the file so the path exists.
	plistPath := filepath.Join(appBundle, "Info.plist")
	if err := os.WriteFile(plistPath, []byte("irrelevant"), 0600); err != nil {
		t.Fatalf("write plist: %v", err)
	}

	// And a data-container directory that findDataContainer should match.
	dataRoot := filepath.Join(filepath.Dir(filepath.Dir(bundleRoot)), "Data/Application")
	dataContainer := filepath.Join(dataRoot, "data-uuid-1")
	if err := os.MkdirAll(dataContainer, 0750); err != nil {
		t.Fatalf("mkdir data container: %v", err)
	}
	metadataPath := filepath.Join(dataContainer, ".com.apple.mobile_container_manager.metadata.plist")
	if err := os.WriteFile(metadataPath, []byte("irrelevant"), 0600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	// Stub plutil: answer the Info.plist read with real app info, and
	// the container metadata read with the matching bundle ID.
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			fmt.Sprintf("plutil -convert json -o - %s", plistPath): {
				out: []byte(`{
					"CFBundleDisplayName": "Example",
					"CFBundleIdentifier": "com.example.app",
					"CFBundleShortVersionString": "1.2.3"
				}`),
			},
			fmt.Sprintf("plutil -convert json -o - %s", metadataPath): {
				out: []byte(`{"MCMMetadataIdentifier": "com.example.app"}`),
			},
		},
	}
	withFakeExecutor(t, fake)

	apps, err := getAppsFromDataDir(udid)
	if err != nil {
		t.Fatalf("getAppsFromDataDir: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("len(apps) = %d, want 1", len(apps))
	}
	got := apps[0]
	if got.Name != "Example" {
		t.Errorf("Name = %q, want 'Example'", got.Name)
	}
	if got.BundleID != "com.example.app" {
		t.Errorf("BundleID = %q, want 'com.example.app'", got.BundleID)
	}
	if got.Version != "1.2.3" {
		t.Errorf("Version = %q, want '1.2.3'", got.Version)
	}
	if got.Path != appBundle {
		t.Errorf("Path = %q, want %q", got.Path, appBundle)
	}
	if got.Container != dataContainer {
		t.Errorf("Container = %q, want %q", got.Container, dataContainer)
	}
	if got.Size < 10 {
		t.Errorf("Size = %d, want >= 10 (bundle contains a 10-byte file)", got.Size)
	}
	if got.ModTime.IsZero() {
		t.Error("ModTime is zero, want non-zero")
	}
}

func TestGetAppsFromDataDir_AppWithoutUsableInfoPlist(t *testing.T) {
	// Info.plist exists but plutil errors; the name should fall back to
	// the bundle's basename minus ".app" and BundleID should read "Unknown".
	udid := "UDID-NOPLIST"
	bundleRoot := setupSimulatorHome(t, udid)

	appContainer := filepath.Join(bundleRoot, "app-uuid-2")
	appBundle := filepath.Join(appContainer, "NoPlistApp.app")
	if err := os.MkdirAll(appBundle, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	plistPath := filepath.Join(appBundle, "Info.plist")
	if err := os.WriteFile(plistPath, []byte("x"), 0600); err != nil {
		t.Fatalf("write plist: %v", err)
	}

	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			fmt.Sprintf("plutil -convert json -o - %s", plistPath): {
				err: fmt.Errorf("plutil: refused"),
			},
		},
	}
	withFakeExecutor(t, fake)

	apps, err := getAppsFromDataDir(udid)
	if err != nil {
		t.Fatalf("getAppsFromDataDir: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("len(apps) = %d, want 1", len(apps))
	}
	if apps[0].Name != "NoPlistApp" {
		t.Errorf("Name = %q, want fallback 'NoPlistApp'", apps[0].Name)
	}
	if apps[0].BundleID != "Unknown" {
		t.Errorf("BundleID = %q, want 'Unknown'", apps[0].BundleID)
	}
}

func TestGetAppsFromDataDir_SkipsContainersWithoutDotApp(t *testing.T) {
	udid := "UDID-NODOTAPP"
	bundleRoot := setupSimulatorHome(t, udid)

	// Container directory that does NOT contain a *.app subdirectory —
	// should be silently skipped.
	noisy := filepath.Join(bundleRoot, "app-uuid-noise")
	if err := os.MkdirAll(filepath.Join(noisy, "not-an-app-dir"), 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	apps, err := getAppsFromDataDir(udid)
	if err != nil {
		t.Fatalf("getAppsFromDataDir: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("len(apps) = %d, want 0 when no *.app is present", len(apps))
	}
}

func TestGetAppsFromDataDir_SortsByName(t *testing.T) {
	udid := "UDID-SORT"
	bundleRoot := setupSimulatorHome(t, udid)

	// Two apps whose directory order does NOT match display-name order,
	// so the result must be sorted alphabetically by Name.
	for _, spec := range []struct {
		dir      string
		bundleID string
		name     string
	}{
		{"container-1", "com.example.zebra", "Zebra"},
		{"container-2", "com.example.alpha", "Alpha"},
	} {
		appBundle := filepath.Join(bundleRoot, spec.dir, spec.name+".app")
		if err := os.MkdirAll(appBundle, 0750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		plistPath := filepath.Join(appBundle, "Info.plist")
		if err := os.WriteFile(plistPath, []byte("x"), 0600); err != nil {
			t.Fatalf("write plist: %v", err)
		}
	}

	// Stub plutil for both bundles.
	fake := &fakeExecutor{responses: map[string]fakeResult{}}
	for _, spec := range []struct {
		dir      string
		bundleID string
		name     string
	}{
		{"container-1", "com.example.zebra", "Zebra"},
		{"container-2", "com.example.alpha", "Alpha"},
	} {
		plistPath := filepath.Join(bundleRoot, spec.dir, spec.name+".app", "Info.plist")
		fake.responses[fmt.Sprintf("plutil -convert json -o - %s", plistPath)] = fakeResult{
			out: []byte(fmt.Sprintf(`{"CFBundleDisplayName": %q, "CFBundleIdentifier": %q, "CFBundleShortVersionString": "1.0"}`,
				spec.name, spec.bundleID)),
		}
	}
	withFakeExecutor(t, fake)

	apps, err := getAppsFromDataDir(udid)
	if err != nil {
		t.Fatalf("getAppsFromDataDir: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("len(apps) = %d, want 2", len(apps))
	}
	if apps[0].Name != "Alpha" || apps[1].Name != "Zebra" {
		t.Errorf("sort order = [%q, %q], want [Alpha, Zebra]", apps[0].Name, apps[1].Name)
	}
}
