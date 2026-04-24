package simulator

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/chroma/v2/styles"
)

// resetChromaInit clears the cached chroma state so initChromaStyle
// runs its body again on the next call. Only valid in tests that do
// not run in parallel.
//
// sync.Once embeds sync.noCopy, so we can't snapshot it by value;
// instead we zero it and let the cleanup also zero it after the test,
// which is sound because no other goroutine touches these package
// variables in the test binary.
func resetChromaInit(t *testing.T) {
	t.Helper()
	prevStyle := chromaStyle
	prevFormatter := termFormatter

	initOnce = sync.Once{}
	chromaStyle = nil
	termFormatter = nil

	t.Cleanup(func() {
		initOnce = sync.Once{}
		chromaStyle = prevStyle
		termFormatter = prevFormatter
	})
}

// captureLog redirects the stdlib logger output to a buffer for the
// duration of the test, so callers can assert on log messages produced
// during initChromaStyle.
func captureLog(t *testing.T) *strings.Builder {
	t.Helper()
	buf := &strings.Builder{}
	prev := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prev)
		log.SetFlags(prevFlags)
	})
	return buf
}

// writeConfig writes a config.toml under an XDG_CONFIG_HOME-rooted
// simtool directory and points XDG_CONFIG_HOME there. Returns the path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "simtool")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestInitChromaStyle_BrokenConfigLogsErrorAndUsesDefaults(t *testing.T) {
	// An unparseable config.toml causes config.Load to return
	// (defaults, error). initChromaStyle must surface the error via the
	// log package and still leave chromaStyle populated (non-nil).
	writeConfig(t, "this is not [ valid toml")
	resetChromaInit(t)
	buf := captureLog(t)

	initChromaStyle()

	if chromaStyle == nil {
		t.Fatal("chromaStyle is nil after init with broken config; want a usable style")
	}
	out := buf.String()
	if !strings.Contains(out, "config load failed") {
		t.Errorf("log output = %q, want it to mention 'config load failed'", out)
	}
}

func TestInitChromaStyle_UnknownThemeFallsBackToGithubDark(t *testing.T) {
	// A valid config selecting a theme name chroma does not recognize
	// must trigger the logged fallback to github-dark.
	body := `
[theme]
mode = "dark"
dark_theme = "does-not-exist-theme-xyz"
`
	writeConfig(t, body)
	resetChromaInit(t)
	buf := captureLog(t)

	initChromaStyle()

	if chromaStyle == nil {
		t.Fatal("chromaStyle is nil after init; want github-dark fallback")
	}
	githubDark := styles.Get("github-dark")
	if chromaStyle != githubDark {
		t.Errorf("chromaStyle = %v, want github-dark fallback %v", chromaStyle, githubDark)
	}
	out := buf.String()
	if !strings.Contains(out, `theme "does-not-exist-theme-xyz" not found`) {
		t.Errorf("log output = %q, want it to mention the missing theme name", out)
	}
}

func TestInitChromaStyle_ValidThemeIsSelected(t *testing.T) {
	body := `
[theme]
mode = "dark"
dark_theme = "monokai"
`
	writeConfig(t, body)
	resetChromaInit(t)

	initChromaStyle()

	want := styles.Get("monokai")
	if want == nil {
		t.Skip("monokai not registered in this chroma build")
	}
	if chromaStyle != want {
		t.Errorf("chromaStyle = %v, want monokai %v", chromaStyle, want)
	}
	if termFormatter == nil {
		t.Error("termFormatter is nil after init")
	}
}
