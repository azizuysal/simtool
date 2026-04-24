package simulator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFetcher_UsesRealExecutor(t *testing.T) {
	f := NewFetcher()
	if f == nil {
		t.Fatal("NewFetcher returned nil")
	}
	concrete, ok := f.(*SimctlFetcher)
	if !ok {
		t.Fatalf("NewFetcher returned %T, want *SimctlFetcher", f)
	}
	if _, ok := concrete.executor.(*RealCommandExecutor); !ok {
		t.Errorf("default executor is %T, want *RealCommandExecutor", concrete.executor)
	}
}

func TestRealCommandExecutor_ExecuteAndRun(t *testing.T) {
	exec := &RealCommandExecutor{}

	// Execute returns stdout of a known benign command.
	out, err := exec.Execute("echo", "simtool-marker")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(string(out), "simtool-marker") {
		t.Errorf("Execute stdout = %q, want containing 'simtool-marker'", string(out))
	}

	// Run of the same command should succeed (exit 0).
	if err := exec.Run("true"); err != nil {
		t.Errorf("Run(true): %v", err)
	}

	// Run of a nonexistent binary should fail.
	if err := exec.Run("this-binary-absolutely-does-not-exist-xyz"); err == nil {
		t.Error("Run(nonexistent) returned nil error, want non-nil")
	}
}

func TestFetchSimulators_Success(t *testing.T) {
	mock := &MockCommandExecutor{
		ExecuteFunc: func(name string, args ...string) ([]byte, error) {
			if name != "xcrun" || args[0] != "simctl" || args[1] != "list" {
				return nil, fmt.Errorf("unexpected command: %s %v", name, args)
			}
			payload := SimctlOutput{
				Devices: map[string][]Simulator{
					"com.apple.CoreSimulator.SimRuntime.iOS-17-0": {
						{UDID: "A", Name: "iPhone 15", State: "Booted", IsAvailable: true},
						{UDID: "B", Name: "iPhone 14", State: "Shutdown", IsAvailable: false},
					},
				},
			}
			return json.Marshal(payload)
		},
	}
	f := NewFetcherWithExecutor(mock)

	sims, err := f.FetchSimulators()
	if err != nil {
		t.Fatalf("FetchSimulators: %v", err)
	}
	if len(sims) != 1 {
		t.Fatalf("len(sims) = %d, want 1 (unavailable sim filtered out)", len(sims))
	}
	if sims[0].UDID != "A" {
		t.Errorf("UDID = %q, want A", sims[0].UDID)
	}
}

func TestFetchSimulators_ExecError(t *testing.T) {
	mock := &MockCommandExecutor{
		ExecuteFunc: func(string, ...string) ([]byte, error) {
			return nil, errors.New("xcrun boom")
		},
	}
	f := NewFetcherWithExecutor(mock)

	if _, err := f.FetchSimulators(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchSimulators_BadJSON(t *testing.T) {
	mock := &MockCommandExecutor{
		ExecuteFunc: func(string, ...string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}
	f := NewFetcherWithExecutor(mock)

	if _, err := f.FetchSimulators(); err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
}

func TestFetcher_Boot_AlreadyBootedFallsBackToOpen(t *testing.T) {
	// simctl boot returns an error indicating the device is already in
	// the Booted state — Boot must swallow that and still call `open`.
	openCalled := false
	mock := &MockCommandExecutor{
		ExecuteFunc: func(name string, args ...string) ([]byte, error) {
			if name == "xcrun" && len(args) >= 2 && args[0] == "simctl" && args[1] == "boot" {
				return []byte("Unable to boot device in current state: Booted"),
					errors.New("exit status 149")
			}
			return nil, fmt.Errorf("unexpected Execute: %s %v", name, args)
		},
		RunFunc: func(name string, args ...string) error {
			if name == "open" && len(args) == 2 && args[0] == "-a" && args[1] == "Simulator" {
				openCalled = true
				return nil
			}
			return fmt.Errorf("unexpected Run: %s %v", name, args)
		},
	}
	f := NewFetcherWithExecutor(mock)

	if err := f.Boot("ALREADY-BOOTED"); err != nil {
		t.Fatalf("Boot: %v", err)
	}
	if !openCalled {
		t.Error("expected Simulator.app to be opened after Booted-state fallback")
	}
}

func TestFetcher_Boot_HardError(t *testing.T) {
	mock := &MockCommandExecutor{
		ExecuteFunc: func(string, ...string) ([]byte, error) {
			return []byte("some other failure"), errors.New("exit status 1")
		},
	}
	f := NewFetcherWithExecutor(mock)

	if err := f.Boot("X"); err == nil {
		t.Fatal("expected error for non-Booted-state failure, got nil")
	}
}

func TestFetcher_Boot_OpenFails(t *testing.T) {
	mock := &MockCommandExecutor{
		ExecuteFunc: func(string, ...string) ([]byte, error) {
			return []byte{}, nil
		},
		RunFunc: func(string, ...string) error {
			return errors.New("open failed")
		},
	}
	f := NewFetcherWithExecutor(mock)

	err := f.Boot("X")
	if err == nil {
		t.Fatal("expected open error to surface")
	}
	if !strings.Contains(err.Error(), "failed to open Simulator app") {
		t.Errorf("error = %q, want wrap of 'failed to open Simulator app'", err.Error())
	}
}

func TestGetAppCountFromDataDir_CountsDirEntries(t *testing.T) {
	// Redirect HOME to a tempdir, build the exact directory layout
	// getAppCountFromDataDir expects, then count.
	udid := "UDID-COUNT"
	home := t.TempDir()
	t.Setenv("HOME", home)

	bundleRoot := filepath.Join(home,
		"Library/Developer/CoreSimulator/Devices", udid,
		"data/Containers/Bundle/Application")
	if err := os.MkdirAll(bundleRoot, 0750); err != nil {
		t.Fatalf("mkdir bundleRoot: %v", err)
	}
	// Three app directories + one stray file (which must not be counted).
	for _, name := range []string{"app1", "app2", "app3"} {
		if err := os.MkdirAll(filepath.Join(bundleRoot, name), 0750); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(bundleRoot, "stray-file"), []byte("x"), 0600); err != nil {
		t.Fatalf("write stray: %v", err)
	}

	f := &SimctlFetcher{executor: &RealCommandExecutor{}}
	got := f.getAppCountFromDataDir(udid)
	if got != 3 {
		t.Errorf("getAppCountFromDataDir = %d, want 3", got)
	}
}

func TestGetAppCountFromDataDir_MissingDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := &SimctlFetcher{executor: &RealCommandExecutor{}}
	if got := f.getAppCountFromDataDir("NO-SUCH-UDID"); got != 0 {
		t.Errorf("getAppCountFromDataDir = %d, want 0 for missing dir", got)
	}
}

func TestFetcher_GetAppCount_FallsBackToDataDirOnListAppsError(t *testing.T) {
	// listapps fails — counter must fall back to filesystem scan.
	udid := "FALLBACK-UDID"
	home := t.TempDir()
	t.Setenv("HOME", home)

	bundleRoot := filepath.Join(home,
		"Library/Developer/CoreSimulator/Devices", udid,
		"data/Containers/Bundle/Application")
	if err := os.MkdirAll(filepath.Join(bundleRoot, "only-app"), 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	mock := &MockCommandExecutor{
		ExecuteFunc: func(name string, args ...string) ([]byte, error) {
			if name == "xcrun" && args[0] == "simctl" && args[1] == "listapps" {
				return nil, errors.New("simctl: device not booted")
			}
			return nil, fmt.Errorf("unexpected Execute: %s %v", name, args)
		},
	}
	f := &SimctlFetcher{executor: mock}

	if got := f.getAppCount(udid); got != 1 {
		t.Errorf("getAppCount = %d, want 1 (fell back to data-dir scan)", got)
	}
}

func TestFormatRuntime(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"com.apple.CoreSimulator.SimRuntime.iOS-17-0", "iOS 17.0"},
		{"com.apple.CoreSimulator.SimRuntime.iOS-16-4-1", "iOS 16.4.1"},
		{"com.apple.CoreSimulator.SimRuntime.watchOS-10-0", "watchOS.10.0"},
		{"unexpected", "unexpected"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := formatRuntime(tt.in); got != tt.want {
				t.Errorf("formatRuntime(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
