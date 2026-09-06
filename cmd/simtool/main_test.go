package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestPrintThemesUsesThemeConfigurationSchema(t *testing.T) {
	var output bytes.Buffer

	if err := printThemes(&output); err != nil {
		t.Fatalf("printThemes: %v", err)
	}

	got := output.String()
	for _, want := range []string{
		"Available syntax highlighting themes:",
		"[theme]",
		"dark_theme = \"theme-name\"",
		"light_theme = \"theme-name\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("theme output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "[syntax]") {
		t.Errorf("theme output advertises unsupported syntax configuration:\n%s", got)
	}
}

func TestPrintThemesReturnsWriterError(t *testing.T) {
	want := errors.New("write failed")
	err := printThemes(errorWriter{err: want})
	if !errors.Is(err, want) {
		t.Errorf("printThemes error = %v, want %v", err, want)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
