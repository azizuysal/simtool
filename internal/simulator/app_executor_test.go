package simulator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeExecutor is a CommandExecutor that returns programmable
// responses keyed by the joined command line. Tests swap it into
// defaultExecutor to avoid spawning real subprocesses.
type fakeExecutor struct {
	responses map[string]fakeResult
	// calls records every call, useful for assertions.
	calls []string
}

type fakeResult struct {
	out []byte
	err error
}

func (f *fakeExecutor) key(name string, args []string) string {
	return name + " " + strings.Join(args, " ")
}

func (f *fakeExecutor) Execute(name string, args ...string) ([]byte, error) {
	key := f.key(name, args)
	f.calls = append(f.calls, key)
	r, ok := f.responses[key]
	if !ok {
		return nil, fmt.Errorf("fakeExecutor: no canned response for %q", key)
	}
	return r.out, r.err
}

func (f *fakeExecutor) Run(name string, args ...string) error {
	_, err := f.Execute(name, args...)
	return err
}

// withFakeExecutor swaps defaultExecutor for the duration of t and
// restores it on cleanup.
func withFakeExecutor(t *testing.T, fake *fakeExecutor) {
	t.Helper()
	original := defaultExecutor
	defaultExecutor = fake
	t.Cleanup(func() { defaultExecutor = original })
}

// ---------- readAppInfo ----------

func TestReadAppInfo_Success(t *testing.T) {
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"plutil -convert json -o - /apps/MyApp.app/Info.plist": {
				out: []byte(`{
					"CFBundleDisplayName": "My App",
					"CFBundleName": "MyApp",
					"CFBundleIdentifier": "com.example.myapp",
					"CFBundleShortVersionString": "1.2.3"
				}`),
			},
		},
	}
	withFakeExecutor(t, fake)

	info := readAppInfo("/apps/MyApp.app")
	if info == nil {
		t.Fatal("readAppInfo returned nil, want info")
	}
	// AppInfo.DisplayName prefers CFBundleDisplayName over CFBundleName.
	if info.DisplayName != "My App" {
		t.Errorf("DisplayName = %q, want 'My App'", info.DisplayName)
	}
	if info.BundleID != "com.example.myapp" {
		t.Errorf("BundleID = %q, want 'com.example.myapp'", info.BundleID)
	}
	if info.Version != "1.2.3" {
		t.Errorf("Version = %q, want '1.2.3'", info.Version)
	}
}

func TestReadAppInfo_FallsBackToCFBundleName(t *testing.T) {
	// Only CFBundleName is present — DisplayName should get it.
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"plutil -convert json -o - /fallback/Info.plist": {
				out: []byte(`{
					"CFBundleName": "FallbackApp",
					"CFBundleIdentifier": "com.example.fallback",
					"CFBundleShortVersionString": "0.1"
				}`),
			},
		},
	}
	withFakeExecutor(t, fake)

	info := readAppInfo("/fallback")
	if info == nil {
		t.Fatal("readAppInfo returned nil")
	}
	if info.DisplayName != "FallbackApp" {
		t.Errorf("DisplayName = %q, want 'FallbackApp' (fallback from CFBundleName)", info.DisplayName)
	}
}

func TestReadAppInfo_PlutilError(t *testing.T) {
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"plutil -convert json -o - /bad/Info.plist": {err: errors.New("plutil: No such file")},
		},
	}
	withFakeExecutor(t, fake)

	if info := readAppInfo("/bad"); info != nil {
		t.Errorf("readAppInfo() = %+v, want nil on plutil error", info)
	}
}

func TestReadAppInfo_InvalidJSON(t *testing.T) {
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"plutil -convert json -o - /app/Info.plist": {out: []byte("not { valid json")},
		},
	}
	withFakeExecutor(t, fake)

	if info := readAppInfo("/app"); info != nil {
		t.Errorf("readAppInfo() = %+v, want nil on invalid JSON", info)
	}
}

func TestReadAppInfo_PartialFields(t *testing.T) {
	// Only bundle ID is set; display name and version absent.
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"plutil -convert json -o - /partial/Info.plist": {
				out: []byte(`{"CFBundleIdentifier": "com.partial"}`),
			},
		},
	}
	withFakeExecutor(t, fake)

	info := readAppInfo("/partial")
	if info == nil {
		t.Fatal("want non-nil info for partial data")
	}
	if info.BundleID != "com.partial" {
		t.Errorf("BundleID = %q", info.BundleID)
	}
	if info.DisplayName != "" {
		t.Errorf("DisplayName = %q, want empty", info.DisplayName)
	}
	if info.Version != "" {
		t.Errorf("Version = %q, want empty", info.Version)
	}
}

// ---------- getAppsFromListApps ----------

func TestGetAppsFromListApps_Success(t *testing.T) {
	// Plist-style output as produced by `xcrun simctl listapps`.
	plist := `{
    "com.example.app1" =     {
        CFBundleDisplayName = "App One";
        CFBundleShortVersionString = "1.0";
        Path = /tmp/app1.app;
        DataContainer = "/tmp/container1";
    };
    "com.example.app2" =     {
        CFBundleDisplayName = "App Two";
        CFBundleShortVersionString = "2.0";
        Path = /tmp/app2.app;
        DataContainer = "/tmp/container2";
    };
    "com.apple.internal" =     {
        CFBundleDisplayName = "Apple Internal";
        CFBundleShortVersionString = "1.0";
        Path = /tmp/apple.app;
        DataContainer = "/tmp/apple-container";
    };
}`
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"xcrun simctl listapps DEVICE-UDID": {out: []byte(plist)},
		},
	}
	withFakeExecutor(t, fake)

	apps, err := getAppsFromListApps("DEVICE-UDID")
	if err != nil {
		t.Fatalf("getAppsFromListApps: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("len(apps) = %d, want 2 (com.apple.* should be filtered)", len(apps))
	}

	// Sorted alphabetically by name: "App One", "App Two"
	if apps[0].BundleID != "com.example.app1" {
		t.Errorf("apps[0].BundleID = %q, want com.example.app1", apps[0].BundleID)
	}
	if apps[0].Name != "App One" {
		t.Errorf("apps[0].Name = %q, want 'App One'", apps[0].Name)
	}
	if apps[0].Version != "1.0" {
		t.Errorf("apps[0].Version = %q, want 1.0", apps[0].Version)
	}
	if apps[0].Path != "/tmp/app1.app" {
		t.Errorf("apps[0].Path = %q", apps[0].Path)
	}
	if apps[0].Container != "/tmp/container1" {
		t.Errorf("apps[0].Container = %q", apps[0].Container)
	}

	if apps[1].BundleID != "com.example.app2" {
		t.Errorf("apps[1].BundleID = %q, want com.example.app2", apps[1].BundleID)
	}
}

func TestGetAppsFromListApps_ExecutorError(t *testing.T) {
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"xcrun simctl listapps BROKEN": {err: errors.New("simctl: device not found")},
		},
	}
	withFakeExecutor(t, fake)

	_, err := getAppsFromListApps("BROKEN")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list apps") {
		t.Errorf("error = %q, want 'failed to list apps' prefix", err.Error())
	}
}

func TestGetAppsFromListApps_EmptyOutput(t *testing.T) {
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"xcrun simctl listapps UDID": {out: []byte("{}")},
		},
	}
	withFakeExecutor(t, fake)

	apps, err := getAppsFromListApps("UDID")
	if err != nil {
		t.Fatalf("getAppsFromListApps: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("want 0 apps, got %d", len(apps))
	}
}

// ---------- findDataContainer ----------

func TestFindDataContainer_Match(t *testing.T) {
	// Lay out a fake data dir with three candidate containers.
	dataPath := t.TempDir()
	for _, name := range []string{"aaaa", "bbbb", "cccc"} {
		if err := os.MkdirAll(filepath.Join(dataPath, name), 0750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	fake := &fakeExecutor{responses: map[string]fakeResult{}}
	// Canned metadata: only "bbbb" matches the bundle ID we're searching for.
	for _, c := range []struct {
		name       string
		identifier string
	}{
		{"aaaa", "com.other.app"},
		{"bbbb", "com.example.target"},
		{"cccc", "com.noise.thing"},
	} {
		key := fmt.Sprintf("plutil -convert json -o - %s/%s/.com.apple.mobile_container_manager.metadata.plist",
			dataPath, c.name)
		fake.responses[key] = fakeResult{
			out: []byte(fmt.Sprintf(`{"MCMMetadataIdentifier": %q}`, c.identifier)),
		}
	}
	withFakeExecutor(t, fake)

	got := findDataContainer(dataPath, "com.example.target")
	want := filepath.Join(dataPath, "bbbb")
	if got != want {
		t.Errorf("findDataContainer() = %q, want %q", got, want)
	}
}

func TestFindDataContainer_NoMatch(t *testing.T) {
	dataPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataPath, "container1"), 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			fmt.Sprintf("plutil -convert json -o - %s/container1/.com.apple.mobile_container_manager.metadata.plist", dataPath): {
				out: []byte(`{"MCMMetadataIdentifier": "com.other.app"}`),
			},
		},
	}
	withFakeExecutor(t, fake)

	got := findDataContainer(dataPath, "com.not.there")
	if got != "" {
		t.Errorf("findDataContainer() = %q, want empty", got)
	}
}

func TestFindDataContainer_MissingDataDir(t *testing.T) {
	// Nonexistent path → returns empty string without panic.
	got := findDataContainer(filepath.Join(t.TempDir(), "does-not-exist"), "com.whatever")
	if got != "" {
		t.Errorf("findDataContainer() = %q, want empty", got)
	}
}

func TestFindDataContainer_SkipsPlutilErrors(t *testing.T) {
	dataPath := t.TempDir()
	for _, name := range []string{"aaaa", "bbbb"} {
		if err := os.MkdirAll(filepath.Join(dataPath, name), 0750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	fake := &fakeExecutor{responses: map[string]fakeResult{}}
	// First container fails plutil (simulating a corrupt metadata file),
	// second succeeds and matches.
	fake.responses[fmt.Sprintf("plutil -convert json -o - %s/aaaa/.com.apple.mobile_container_manager.metadata.plist", dataPath)] = fakeResult{
		err: errors.New("plutil: failed"),
	}
	fake.responses[fmt.Sprintf("plutil -convert json -o - %s/bbbb/.com.apple.mobile_container_manager.metadata.plist", dataPath)] = fakeResult{
		out: []byte(`{"MCMMetadataIdentifier": "com.example.target"}`),
	}
	withFakeExecutor(t, fake)

	got := findDataContainer(dataPath, "com.example.target")
	want := filepath.Join(dataPath, "bbbb")
	if got != want {
		t.Errorf("findDataContainer() = %q, want %q", got, want)
	}
}

// ---------- GetAppsForSimulator dispatcher ----------

func TestGetAppsForSimulator_RunningUsesListApps(t *testing.T) {
	fake := &fakeExecutor{
		responses: map[string]fakeResult{
			"xcrun simctl listapps UDID": {
				out: []byte(`{
    "com.example.app" =     {
        CFBundleDisplayName = "Running App";
        CFBundleShortVersionString = "1.0";
        Path = /tmp/app.app;
        DataContainer = "/tmp/data";
    };
}`),
			},
		},
	}
	withFakeExecutor(t, fake)

	apps, err := GetAppsForSimulator("UDID", true)
	if err != nil {
		t.Fatalf("GetAppsForSimulator: %v", err)
	}
	if len(apps) != 1 || apps[0].BundleID != "com.example.app" {
		t.Errorf("apps = %+v, want one entry with bundle com.example.app", apps)
	}

	// Verify the executor was called with listapps (not via data-dir walking).
	foundListApps := false
	for _, c := range fake.calls {
		if strings.HasPrefix(c, "xcrun simctl listapps") {
			foundListApps = true
			break
		}
	}
	if !foundListApps {
		t.Error("expected xcrun simctl listapps to be called for running simulators")
	}
}
