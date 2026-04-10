package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTOML creates a temp config file with the given body and
// returns the path.
func writeTOML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadFromPath_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := loadFromPath(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err != nil {
		t.Fatalf("loadFromPath: %v", err)
	}
	if cfg.Theme.Mode != "auto" {
		t.Errorf("theme.mode = %q, want 'auto' (default)", cfg.Theme.Mode)
	}
}

func TestLoadFromPath_ValidOverride(t *testing.T) {
	path := writeTOML(t, `
[theme]
mode = "dark"
dark_theme = "dracula"

[startup]
initial_view = "all_apps"
`)
	cfg, err := loadFromPath(path)
	if err != nil {
		t.Fatalf("loadFromPath: %v", err)
	}
	if cfg.Theme.Mode != "dark" {
		t.Errorf("theme.mode = %q, want 'dark'", cfg.Theme.Mode)
	}
	if cfg.Theme.DarkTheme != "dracula" {
		t.Errorf("theme.dark_theme = %q, want 'dracula'", cfg.Theme.DarkTheme)
	}
	if cfg.Startup.InitialView != "all_apps" {
		t.Errorf("startup.initial_view = %q, want 'all_apps'", cfg.Startup.InitialView)
	}
	// Untouched fields should retain their default values.
	if cfg.Theme.LightTheme != "github" {
		t.Errorf("theme.light_theme = %q, want 'github' (default)", cfg.Theme.LightTheme)
	}
}

func TestLoadFromPath_UnknownTopLevelKeyRejected(t *testing.T) {
	path := writeTOML(t, `
[theme]
mode = "auto"

[unknown_section]
foo = "bar"
`)
	cfg, err := loadFromPath(path)
	if err == nil {
		t.Fatal("expected error for unknown section, got nil")
	}
	if !strings.Contains(err.Error(), "unknown keys") {
		t.Errorf("error = %q, want to mention 'unknown keys'", err.Error())
	}
	if !strings.Contains(err.Error(), "unknown_section") {
		t.Errorf("error = %q, want to name the unknown section", err.Error())
	}
	// Defaults should still be returned alongside the error.
	if cfg == nil || cfg.Theme.Mode != "auto" {
		t.Error("defaults should be returned even when an error occurs")
	}
}

func TestLoadFromPath_UnknownFieldRejected(t *testing.T) {
	path := writeTOML(t, `
[theme]
mode = "auto"
moed = "typo"
`)
	_, err := loadFromPath(path)
	if err == nil {
		t.Fatal("expected error for typo field")
	}
	if !strings.Contains(err.Error(), "moed") {
		t.Errorf("error = %q, want to name the typo field", err.Error())
	}
}

func TestLoadFromPath_InvalidThemeMode(t *testing.T) {
	path := writeTOML(t, `
[theme]
mode = "maybe"
`)
	_, err := loadFromPath(path)
	if err == nil {
		t.Fatal("expected error for invalid theme.mode")
	}
	if !strings.Contains(err.Error(), "theme.mode") {
		t.Errorf("error = %q, want to mention 'theme.mode'", err.Error())
	}
	if !strings.Contains(err.Error(), "maybe") {
		t.Errorf("error = %q, want to echo the invalid value", err.Error())
	}
}

func TestLoadFromPath_InvalidInitialView(t *testing.T) {
	path := writeTOML(t, `
[startup]
initial_view = "dashboard"
`)
	_, err := loadFromPath(path)
	if err == nil {
		t.Fatal("expected error for invalid startup.initial_view")
	}
	if !strings.Contains(err.Error(), "startup.initial_view") {
		t.Errorf("error = %q, want to mention 'startup.initial_view'", err.Error())
	}
}

func TestLoadFromPath_MultipleValidationErrors(t *testing.T) {
	path := writeTOML(t, `
[theme]
mode = "bright"

[startup]
initial_view = "dashboard"
`)
	_, err := loadFromPath(path)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "theme.mode") || !strings.Contains(msg, "startup.initial_view") {
		t.Errorf("error = %q, want both errors reported together", msg)
	}
}

func TestLoadFromPath_MalformedTOML(t *testing.T) {
	path := writeTOML(t, `this is = not valid [toml`)
	cfg, err := loadFromPath(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "decoding") {
		t.Errorf("error = %q, want 'decoding' in message", err.Error())
	}
	if cfg == nil {
		t.Error("defaults should still be returned")
	}
}

func TestValidate_Defaults(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Errorf("Default() failed validation: %v", err)
	}
}

func TestValidate_EmptyFieldsAllowed(t *testing.T) {
	// Empty strings are allowed because merge() will fill them from
	// the default; they represent "not set in user config" rather
	// than "invalid".
	cfg := &Config{}
	if err := cfg.Validate(); err != nil {
		t.Errorf("empty config failed validation: %v", err)
	}
}
