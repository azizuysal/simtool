package config

import (
	"testing"
)

func TestParseOSC11Response(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "rgb with ST terminator",
			in:   "\x1b]11;rgb:ffff/ffff/ffff\x1b\\",
			want: "rgb:ffff/ffff/ffff",
		},
		{
			name: "rgb with BEL terminator",
			in:   "\x1b]11;rgb:1a1b/2a2b/3a3b\x07",
			want: "rgb:1a1b/2a2b/3a3b",
		},
		{
			name: "rgb without terminator consumes to end",
			in:   "leading-junk\x1b]11;rgb:00/00/00",
			want: "rgb:00/00/00",
		},
		{
			name: "no rgb marker returns empty",
			in:   "some unrelated terminal output",
			want: "",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseOSC11Response(c.in); got != c.want {
				t.Errorf("parseOSC11Response(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestIsColorDark_8BitChannels(t *testing.T) {
	// theme_test.go exercises the common 16-bit-per-channel path.
	// These cases cover the 8-bit channel normalization branch —
	// IsColorDark shifts only when a channel exceeds 255.
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"pure black 8-bit", "rgb:00/00/00", true},
		{"pure white 8-bit", "rgb:ff/ff/ff", false},
		{"dark red 8-bit", "rgb:40/00/00", true},
		{"bright yellow 8-bit", "rgb:ff/ff/00", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsColorDark(c.in); got != c.want {
				t.Errorf("IsColorDark(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestIsColorDark_WrongChannelCount(t *testing.T) {
	// Not three slash-separated channels → the implementation defaults
	// to "dark" for safety on ambiguous input.
	if !IsColorDark("rgb:ff/ff") {
		t.Error("IsColorDark on 2-channel input returned false, want true (safe default)")
	}
}
