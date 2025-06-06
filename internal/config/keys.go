package config

import "strings"

// KeysConfig represents keyboard shortcut configuration
type KeysConfig struct {
	// Navigation
	Up    []string `toml:"up"`
	Down  []string `toml:"down"`
	Left  []string `toml:"left"`
	Right []string `toml:"right"`
	Home  []string `toml:"home"`
	End   []string `toml:"end"`
	
	// Actions
	Quit   []string `toml:"quit"`
	Boot   []string `toml:"boot"`   // Boot simulator
	Open   []string `toml:"open"`   // Open in Finder
	Filter []string `toml:"filter"` // Toggle filter
	Search []string `toml:"search"` // Start search
	Escape []string `toml:"escape"` // Exit search/cancel
	Enter  []string `toml:"enter"`  // Select/confirm
	
	// Search mode
	Backspace []string `toml:"backspace"` // Delete character in search
}

// DefaultKeys returns the default keyboard shortcuts
func DefaultKeys() KeysConfig {
	return KeysConfig{
		// Navigation
		Up:    []string{"up", "k"},
		Down:  []string{"down", "j"},
		Left:  []string{"left", "h"},
		Right: []string{"right", "l"},
		Home:  []string{"home"},
		End:   []string{"end"},
		
		// Actions
		Quit:   []string{"q", "ctrl+c"},
		Boot:   []string{" "}, // space
		Open:   []string{" "}, // space (context-dependent)
		Filter: []string{"f"},
		Search: []string{"/"},
		Escape: []string{"esc"},
		Enter:  []string{"enter"},
		
		// Search mode
		Backspace: []string{"backspace"},
	}
}

// KeyMap provides a fast lookup for key bindings
type KeyMap struct {
	// Maps from key string to action
	bindings map[string]string
}

// NewKeyMap creates a key mapping from the configuration
func NewKeyMap(keys KeysConfig) *KeyMap {
	km := &KeyMap{
		bindings: make(map[string]string),
	}
	
	// Build the reverse mapping
	km.addBindings("up", keys.Up)
	km.addBindings("down", keys.Down)
	km.addBindings("left", keys.Left)
	km.addBindings("right", keys.Right)
	km.addBindings("home", keys.Home)
	km.addBindings("end", keys.End)
	km.addBindings("quit", keys.Quit)
	km.addBindings("boot", keys.Boot)
	km.addBindings("open", keys.Open)
	km.addBindings("filter", keys.Filter)
	km.addBindings("search", keys.Search)
	km.addBindings("escape", keys.Escape)
	km.addBindings("enter", keys.Enter)
	km.addBindings("backspace", keys.Backspace)
	
	return km
}

// addBindings adds multiple key bindings for an action
func (km *KeyMap) addBindings(action string, keys []string) {
	for _, key := range keys {
		if key != "" {
			km.bindings[key] = action
		}
	}
}

// GetAction returns the action for a given key, or empty string if not bound
func (km *KeyMap) GetAction(key string) string {
	return km.bindings[key]
}

// IsKey checks if a key is bound to a specific action
func (km *KeyMap) IsKey(key string, action string) bool {
	return km.bindings[key] == action
}

// FormatKeys formats multiple keys for display in the UI
// e.g. ["up", "k"] -> "↑/k"
func FormatKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	
	formatted := make([]string, 0, len(keys))
	for _, key := range keys {
		// Convert common keys to symbols for better display
		switch key {
		case "up":
			formatted = append(formatted, "↑")
		case "down":
			formatted = append(formatted, "↓")
		case "left":
			formatted = append(formatted, "←")
		case "right":
			formatted = append(formatted, "→")
		case " ":
			formatted = append(formatted, "space")
		case "ctrl+c":
			formatted = append(formatted, "Ctrl+C")
		case "esc":
			formatted = append(formatted, "ESC")
		case "enter":
			formatted = append(formatted, "Enter")
		case "backspace":
			formatted = append(formatted, "Backspace")
		case "home":
			formatted = append(formatted, "Home")
		case "end":
			formatted = append(formatted, "End")
		default:
			formatted = append(formatted, key)
		}
	}
	
	// Join with slash
	return strings.Join(formatted, "/")
}

// FormatKeyAction formats a key binding with its action label
// e.g. "up", "up" -> "↑/k: up"
func (kc *KeysConfig) FormatKeyAction(action string, label string) string {
	var keys []string
	
	switch action {
	case "up":
		keys = kc.Up
	case "down":
		keys = kc.Down
	case "left":
		keys = kc.Left
	case "right":
		keys = kc.Right
	case "home":
		keys = kc.Home
	case "end":
		keys = kc.End
	case "quit":
		keys = kc.Quit
	case "boot":
		keys = kc.Boot
	case "open":
		keys = kc.Open
	case "filter":
		keys = kc.Filter
	case "search":
		keys = kc.Search
	case "escape":
		keys = kc.Escape
	case "enter":
		keys = kc.Enter
	case "backspace":
		keys = kc.Backspace
	}
	
	if len(keys) == 0 {
		return ""
	}
	
	return FormatKeys(keys) + ": " + label
}