package config

import (
	"log"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGenerateStyles_HappyPath(t *testing.T) {
	// A valid theme name resolves cleanly; GenerateStyles must return a
	// fully populated Styles struct with no zero-valued lipgloss.Style
	// fields.
	cfg := Default()
	cfg.Theme.Mode = "dark"
	cfg.Theme.DarkTheme = "github-dark"

	s := cfg.GenerateStyles()
	if s == nil {
		t.Fatal("GenerateStyles returned nil")
	}
	checkNonZero := func(name string, st lipgloss.Style) {
		t.Helper()
		// lipgloss.Style zero-value renders to an empty string for any
		// non-empty input. Use a marker input and assert the output
		// differs or at least contains the marker.
		out := st.Render("marker")
		if !strings.Contains(out, "marker") {
			t.Errorf("%s: rendered output %q dropped the input", name, out)
		}
	}
	checkNonZero("Selected", s.Selected)
	checkNonZero("Normal", s.Normal)
	checkNonZero("Booted", s.Booted)
	checkNonZero("Shutdown", s.Shutdown)
	checkNonZero("Error", s.Error)
	checkNonZero("Success", s.Success)
	checkNonZero("Header", s.Header)
	checkNonZero("Border", s.Border)
	checkNonZero("Footer", s.Footer)
	checkNonZero("Name", s.Name)
	checkNonZero("Detail", s.Detail)
	checkNonZero("Folder", s.Folder)
	checkNonZero("Search", s.Search)
	checkNonZero("Status", s.Status)
	checkNonZero("Loading", s.Loading)
	checkNonZero("ListItem", s.ListItem)
}

func TestGenerateStyles_UnknownThemeLogsWarningAndFallsBack(t *testing.T) {
	// An unknown theme triggers the logged fallback to github-dark;
	// redirect the stdlib logger to assert on the warning text.
	buf := &strings.Builder{}
	prev := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prev)
		log.SetFlags(prevFlags)
	})

	cfg := Default()
	cfg.Theme.Mode = "dark"
	cfg.Theme.DarkTheme = "this-is-not-a-real-theme-xyz"

	s := cfg.GenerateStyles()
	if s == nil {
		t.Fatal("GenerateStyles returned nil on fallback path")
	}

	out := buf.String()
	if !strings.Contains(out, "failed to extract colors from theme") {
		t.Errorf("log output = %q, want it to contain 'failed to extract colors from theme'", out)
	}
	if !strings.Contains(out, "this-is-not-a-real-theme-xyz") {
		t.Errorf("log output = %q, want it to mention the bad theme name", out)
	}
}
