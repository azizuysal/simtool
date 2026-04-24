package config

import (
	"testing"
)

func TestDefaultKeys(t *testing.T) {
	d := DefaultKeys()

	checks := []struct {
		name    string
		got     []string
		want    []string
		atLeast int
	}{
		{"Up", d.Up, []string{"up", "k"}, 0},
		{"Down", d.Down, []string{"down", "j"}, 0},
		{"Left", d.Left, []string{"left", "h"}, 0},
		{"Right", d.Right, []string{"right", "l"}, 0},
		{"Home", d.Home, []string{"home"}, 0},
		{"End", d.End, []string{"end"}, 0},
		{"Quit", d.Quit, []string{"q", "ctrl+c"}, 0},
		{"Boot", d.Boot, []string{" "}, 0},
		{"Open", d.Open, []string{" "}, 0},
		{"Filter", d.Filter, []string{"f"}, 0},
		{"Search", d.Search, []string{"/"}, 0},
		{"Escape", d.Escape, []string{"esc"}, 0},
		{"Enter", d.Enter, []string{"enter"}, 0},
		{"Backspace", d.Backspace, []string{"backspace"}, 0},
	}

	for _, c := range checks {
		if len(c.got) != len(c.want) {
			t.Errorf("%s: len=%d, want %d (got %v)", c.name, len(c.got), len(c.want), c.got)
			continue
		}
		for i := range c.got {
			if c.got[i] != c.want[i] {
				t.Errorf("%s[%d] = %q, want %q", c.name, i, c.got[i], c.want[i])
			}
		}
	}
}

func TestNewKeyMap_ResolvesAllActions(t *testing.T) {
	km := NewKeyMap(DefaultKeys())

	cases := []struct {
		key    string
		action string
	}{
		{"up", "up"}, {"k", "up"},
		{"down", "down"}, {"j", "down"},
		{"left", "left"}, {"h", "left"},
		{"right", "right"}, {"l", "right"},
		{"home", "home"},
		{"end", "end"},
		{"q", "quit"}, {"ctrl+c", "quit"},
		{" ", "open"}, // Open is declared AFTER Boot in NewKeyMap, so "open" wins on collision
		{"f", "filter"},
		{"/", "search"},
		{"esc", "escape"},
		{"enter", "enter"},
		{"backspace", "backspace"},
	}
	for _, c := range cases {
		if got := km.GetAction(c.key); got != c.action {
			t.Errorf("GetAction(%q) = %q, want %q", c.key, got, c.action)
		}
	}
}

func TestKeyMap_UnknownKeyReturnsEmpty(t *testing.T) {
	km := NewKeyMap(DefaultKeys())
	if got := km.GetAction("this-key-is-not-bound"); got != "" {
		t.Errorf("GetAction of unknown key = %q, want empty", got)
	}
}

func TestKeyMap_IsKey(t *testing.T) {
	km := NewKeyMap(DefaultKeys())
	if !km.IsKey("k", "up") {
		t.Error(`IsKey("k", "up") = false, want true`)
	}
	if km.IsKey("k", "down") {
		t.Error(`IsKey("k", "down") = true, want false`)
	}
	if km.IsKey("not-bound", "anything") {
		t.Error("IsKey on unknown key returned true")
	}
}

func TestKeyMap_EmptyBindingDoesNotMatch(t *testing.T) {
	// Empty-string bindings must be silently skipped by addBindings,
	// otherwise the empty string (which is what msg.String() returns
	// for some key events) would be bound to an arbitrary action.
	kc := DefaultKeys()
	kc.Quit = []string{"", "q"}
	km := NewKeyMap(kc)

	if got := km.GetAction(""); got != "" {
		t.Errorf("empty key resolved to action %q, want empty", got)
	}
	if got := km.GetAction("q"); got != "quit" {
		t.Errorf(`GetAction("q") = %q, want "quit"`, got)
	}
}

func TestFormatKeys(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"arrow symbols", []string{"up", "down", "left", "right"}, "↑/↓/←/→"},
		{"space rendered as word", []string{" "}, "space"},
		{"ctrl+c casing", []string{"ctrl+c"}, "Ctrl+C"},
		{"esc uppercased", []string{"esc"}, "ESC"},
		{"enter titlecased", []string{"enter"}, "Enter"},
		{"backspace titlecased", []string{"backspace"}, "Backspace"},
		{"home/end titlecased", []string{"home", "end"}, "Home/End"},
		{"default passthrough", []string{"f", "/"}, "f//"},
		{"empty slice", []string{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FormatKeys(c.in); got != c.want {
				t.Errorf("FormatKeys(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestKeysConfig_FormatKeyAction(t *testing.T) {
	kc := DefaultKeys()

	cases := []struct {
		action string
		label  string
		want   string
	}{
		{"up", "move up", "↑/k: move up"},
		{"down", "move down", "↓/j: move down"},
		{"left", "back", "←/h: back"},
		{"right", "enter", "→/l: enter"},
		{"home", "top", "Home: top"},
		{"end", "bottom", "End: bottom"},
		{"quit", "quit", "q/Ctrl+C: quit"},
		{"boot", "boot", "space: boot"},
		{"open", "open", "space: open"},
		{"filter", "filter", "f: filter"},
		{"search", "search", "/: search"},
		{"escape", "cancel", "ESC: cancel"},
		{"enter", "select", "Enter: select"},
		{"backspace", "delete", "Backspace: delete"},
		{"unknown-action", "x", ""},
	}
	for _, c := range cases {
		t.Run(c.action, func(t *testing.T) {
			if got := kc.FormatKeyAction(c.action, c.label); got != c.want {
				t.Errorf("FormatKeyAction(%q, %q) = %q, want %q", c.action, c.label, got, c.want)
			}
		})
	}
}

func TestKeysConfig_FormatKeyAction_EmptyBinding(t *testing.T) {
	// An action with an empty binding list should format to empty —
	// callers use this to skip over disabled bindings in help text.
	kc := DefaultKeys()
	kc.Filter = nil
	if got := kc.FormatKeyAction("filter", "filter"); got != "" {
		t.Errorf("FormatKeyAction with empty Filter = %q, want empty", got)
	}
}
