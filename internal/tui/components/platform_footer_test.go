package components

import (
	"strings"
	"testing"

	"github.com/azizuysal/simtool/internal/config"
)

func TestPlatformShortcutFooterUsesConfiguredBinding(t *testing.T) {
	for _, view := range []string{"devices", "all apps"} {
		for _, test := range []struct {
			name     string
			binding  []string
			search   bool
			defaults bool
			want     string
		}{
			{name: "default", binding: []string{"p"}, want: "p: platform"},
			{name: "custom", binding: []string{"ctrl+p"}, want: "ctrl+p: platform"},
			{name: "disabled"},
			{name: "search input", binding: []string{"p"}, search: true},
			{name: "no config", defaults: true, want: "p: platform"},
			{name: "no config in search", defaults: true, search: true},
		} {
			t.Run(view+"/"+test.name, func(t *testing.T) {
				keys := config.DefaultKeys()
				keys.Platform = test.binding
				keyConfig := &keys
				if test.defaults {
					keyConfig = nil
				}
				var footer string
				if view == "devices" {
					list := NewSimulatorList(80, 24)
					list.Update(nil, 0, 0, false, test.search, "", keyConfig)
					footer = list.GetFooter()
				} else {
					footer = buildAllAppsFooter(test.search, 0, keyConfig, 0, 4)
				}
				if test.want != "" {
					if !strings.Contains(footer, test.want) {
						t.Errorf("footer %q does not contain %q", footer, test.want)
					}
				} else if strings.Contains(footer, "platform") {
					t.Errorf("inactive shortcut advertised in %q", footer)
				}
			})
		}
	}
}

func TestSearchFootersOnlyAdvertiseActiveNonTextBindings(t *testing.T) {
	custom := config.DefaultKeys()
	custom.Up = nil
	custom.Down = []string{"ctrl+n", "j"}
	custom.Right = []string{"l"}
	custom.Enter = []string{"enter", "e"}
	custom.Escape = []string{"x"}
	for _, test := range []struct {
		name string
		keys config.KeysConfig
		want string
	}{
		{name: "default", keys: config.DefaultKeys(), want: "ESC: exit search • ↑/↓: navigate • →/Enter: select"},
		{name: "custom", keys: custom, want: "ctrl+n: navigate • Enter: select"},
		{name: "disabled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			devices := NewSimulatorList(80, 24)
			devices.Update(nil, 0, 0, false, true, "", &test.keys)
			apps := NewAppList(80, 24)
			apps.Keys = &test.keys
			apps.SearchMode = true
			for name, footer := range map[string]string{
				"devices":  devices.GetFooter(),
				"apps":     apps.GetFooter(),
				"all apps": buildAllAppsFooter(true, 0, &test.keys, 0, 4),
			} {
				want := test.want
				if name == "all apps" {
					want = strings.ReplaceAll(want, "exit search", "cancel")
					if want != "" {
						want = " • " + want
					}
					want = "Type to search" + want
				}
				if footer != want {
					t.Errorf("%s footer = %q, want %q", name, footer, want)
				}
			}
		})
	}
}
