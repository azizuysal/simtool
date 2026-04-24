package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigDir_UsesXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir, err := getConfigDir()
	if err != nil {
		t.Fatalf("getConfigDir: %v", err)
	}
	want := filepath.Join(xdg, "simtool")
	if dir != want {
		t.Errorf("getConfigDir = %q, want %q", dir, want)
	}
}

func TestGetConfigDir_FallsBackToHomeConfig(t *testing.T) {
	// Unset XDG_CONFIG_HOME and point HOME at a tempdir so we can
	// verify the fallback branch without touching the real home dir.
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := getConfigDir()
	if err != nil {
		t.Fatalf("getConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config", "simtool")
	if dir != want {
		t.Errorf("getConfigDir = %q, want %q", dir, want)
	}
}

func TestGetConfigPath_AppendsConfigToml(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	path, err := getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath: %v", err)
	}
	want := filepath.Join(xdg, "simtool", "config.toml")
	if path != want {
		t.Errorf("getConfigPath = %q, want %q", path, want)
	}
}

func TestSaveExample_WritesFileAndContent(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	if err := SaveExample(); err != nil {
		t.Fatalf("SaveExample: %v", err)
	}

	// The file must exist under the simtool config dir.
	examplePath := filepath.Join(xdg, "simtool", "config.example.toml")
	info, err := os.Stat(examplePath)
	if err != nil {
		t.Fatalf("stat example file: %v", err)
	}
	if info.IsDir() {
		t.Fatal("example path is a directory, want a file")
	}

	// Content sanity check — the example should at minimum declare the
	// three top-level TOML sections the Config struct recognizes.
	body, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	for _, marker := range []string{"[theme]", "[startup]", "[keys]"} {
		if !strings.Contains(string(body), marker) {
			t.Errorf("example file missing section %q", marker)
		}
	}
}

func TestSaveExample_CreatesConfigDirIfMissing(t *testing.T) {
	// The parent simtool directory does not exist yet — SaveExample
	// must create it (with user-only permissions per the implementation).
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	simtoolDir := filepath.Join(xdg, "simtool")
	if _, err := os.Stat(simtoolDir); !os.IsNotExist(err) {
		t.Fatalf("simtool dir already exists before SaveExample: %v", err)
	}

	if err := SaveExample(); err != nil {
		t.Fatalf("SaveExample: %v", err)
	}

	info, err := os.Stat(simtoolDir)
	if err != nil {
		t.Fatalf("simtool dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("simtool path exists but is not a directory")
	}
}
