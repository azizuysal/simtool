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
