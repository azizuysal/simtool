package config

import (
	"fmt"
	"os"
	"strings"
	
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// ThemeColors holds all colors derived from a chroma theme
type ThemeColors struct {
	// Base colors
	Background string
	Foreground string
	
	// UI element colors
	Selection      string // For selected items
	SelectionText  string // Text on selected items
	Border         string // For borders
	HeaderBg       string // Header background
	HeaderFg       string // Header foreground
	
	// Status colors  
	Success  string // Success/running status
	Error    string // Error messages
	Warning  string // Warnings
	Info     string // Info messages
	
	// Content colors
	Primary   string // Primary content (names)
	Secondary string // Secondary content (details)
	Muted     string // Muted text (footer, etc)
	Accent    string // Accent color (folders, special items)
}

// ExtractThemeColors extracts colors from a chroma theme
func ExtractThemeColors(themeName string) (*ThemeColors, error) {
	theme := styles.Get(themeName)
	if theme == nil || theme == styles.Fallback {
		return nil, fmt.Errorf("theme %q not found", themeName)
	}
	
	tc := &ThemeColors{}
	isDark := !isLightTheme(theme)
	
	// For any missing colors, we'll use a fallback theme
	var fallbackTheme *chroma.Style
	if isDark {
		fallbackTheme = styles.Get("github-dark")
	} else {
		fallbackTheme = styles.Get("github")
	}
	
	// Extract base colors
	bg := theme.Get(chroma.Background)
	if bg.Background.IsSet() {
		tc.Background = colorToHex(bg.Background)
	} else {
		// Use fallback theme's background
		fbg := fallbackTheme.Get(chroma.Background)
		if fbg.Background.IsSet() {
			tc.Background = colorToHex(fbg.Background)
		} else {
			tc.Background = tc.Background // This should never happen with monokai/github
		}
	}
	
	// Extract foreground
	text := theme.Get(chroma.Text)
	if text.Colour.IsSet() {
		tc.Foreground = colorToHex(text.Colour)
	} else {
		// Some themes (like github light) don't set Text.Colour
		// For light themes, default text is usually black/very dark
		if isLightTheme(theme) {
			// Check NameClass which is often the darkest color in light themes
			nameClass := theme.Get(chroma.NameClass)
			if nameClass.Colour.IsSet() {
				tc.Foreground = colorToHex(nameClass.Colour)
			} else {
				// Default to black for light themes
				tc.Foreground = "#000000"
			}
		} else {
			// For dark themes, check if background has a foreground color
			if bg.Colour.IsSet() {
				tc.Foreground = colorToHex(bg.Colour)
			} else {
				// Use fallback theme
				ftext := fallbackTheme.Get(chroma.Text)
				if ftext.Colour.IsSet() {
					tc.Foreground = colorToHex(ftext.Colour)
				} else {
					fbg := fallbackTheme.Get(chroma.Background)
					if fbg.Colour.IsSet() {
						tc.Foreground = colorToHex(fbg.Colour)
					} else {
						tc.Foreground = "#e6edf3" // github-dark default
					}
				}
			}
		}
	}
	
	// Selection colors - use keyword color as base
	keyword := theme.Get(chroma.Keyword)
	if keyword.Colour.IsSet() {
		tc.Selection = colorToHex(keyword.Colour)
		tc.SelectionText = tc.Background // Inverse for contrast
	} else {
		// Use fallback theme
		fkeyword := fallbackTheme.Get(chroma.Keyword)
		if fkeyword.Colour.IsSet() {
			tc.Selection = colorToHex(fkeyword.Colour)
		} else {
			tc.Selection = tc.Foreground // Last resort
		}
		tc.SelectionText = tc.Background
	}
	
	// Border color - temporarily use a placeholder
	// Will be set after secondary color is calculated
	tc.Border = ""
	
	// Header colors - use selection colors
	tc.HeaderBg = tc.Selection
	tc.HeaderFg = tc.SelectionText
	
	// Success - use string color (usually green)
	str := theme.Get(chroma.String)
	if str.Colour.IsSet() {
		tc.Success = colorToHex(str.Colour)
	} else {
		fstr := fallbackTheme.Get(chroma.String)
		if fstr.Colour.IsSet() {
			tc.Success = colorToHex(fstr.Colour)
		} else {
			tc.Success = tc.Foreground
		}
	}
	
	// Error - use error token color
	err := theme.Get(chroma.Error)
	if err.Colour.IsSet() {
		tc.Error = colorToHex(err.Colour)
	} else {
		ferr := fallbackTheme.Get(chroma.Error)
		if ferr.Colour.IsSet() {
			tc.Error = colorToHex(ferr.Colour)
		} else {
			tc.Error = tc.Foreground
		}
	}
	
	// Warning - use name.constant or number color (often orange/yellow)
	constant := theme.Get(chroma.NameConstant)
	if constant.Colour.IsSet() {
		tc.Warning = colorToHex(constant.Colour)
	} else {
		num := theme.Get(chroma.Number)
		if num.Colour.IsSet() {
			tc.Warning = colorToHex(num.Colour)
		} else {
			// Use fallback
			fconstant := fallbackTheme.Get(chroma.NameConstant)
			if fconstant.Colour.IsSet() {
				tc.Warning = colorToHex(fconstant.Colour)
			} else {
				fnum := fallbackTheme.Get(chroma.Number)
				if fnum.Colour.IsSet() {
					tc.Warning = colorToHex(fnum.Colour)
				} else {
					tc.Warning = tc.Foreground
				}
			}
		}
	}
	
	// Info - use name.function color (often blue/cyan)
	fn := theme.Get(chroma.NameFunction)
	if fn.Colour.IsSet() {
		tc.Info = colorToHex(fn.Colour)
	} else {
		ffn := fallbackTheme.Get(chroma.NameFunction)
		if ffn.Colour.IsSet() {
			tc.Info = colorToHex(ffn.Colour)
		} else {
			tc.Info = tc.Foreground
		}
	}
	
	// Primary content - use foreground but ensure contrast
	tc.Primary = tc.Foreground
	
	// Secondary content - use comment color but ensure readability
	comment := theme.Get(chroma.Comment)
	if comment.Colour.IsSet() {
		tc.Secondary = colorToHex(comment.Colour)
		// For light themes, ensure secondary text is dark enough
		if isLightTheme(theme) && comment.Colour.Brightness() > 0.7 {
			// Too light, darken it
			tc.Secondary = colorToHex(comment.Colour.Brighten(-0.4))
		}
		tc.Muted = tc.Secondary
	} else {
		// Use fallback theme's comment color
		fcomment := fallbackTheme.Get(chroma.Comment)
		if fcomment.Colour.IsSet() {
			tc.Secondary = colorToHex(fcomment.Colour)
			tc.Muted = tc.Secondary
		} else {
			// Last resort - slightly faded foreground
			if text.Colour.IsSet() {
				tc.Secondary = colorToHex(text.Colour.Brighten(-0.3))
			} else {
				tc.Secondary = tc.Foreground
			}
			tc.Muted = tc.Secondary
		}
	}
	
	// Accent - use name.class or name.namespace (often bright colors)
	class := theme.Get(chroma.NameClass)
	if class.Colour.IsSet() {
		tc.Accent = colorToHex(class.Colour)
		// For light themes, ensure accent is dark enough
		if isLightTheme(theme) && class.Colour.Brightness() > 0.6 {
			tc.Accent = colorToHex(class.Colour.Brighten(-0.5))
		}
	} else {
		// Use fallback theme's class color
		fclass := fallbackTheme.Get(chroma.NameClass)
		if fclass.Colour.IsSet() {
			tc.Accent = colorToHex(fclass.Colour)
		} else {
			tc.Accent = tc.Info
		}
	}
	
	// Set border color to secondary (which is already contrast-adjusted)
	tc.Border = tc.Secondary
	
	return tc, nil
}

// colorToHex converts a chroma color to hex string
func colorToHex(c chroma.Colour) string {
	if !c.IsSet() {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", 
		uint8(c.Red()), 
		uint8(c.Green()), 
		uint8(c.Blue()))
}

// isLightTheme determines if a theme is light based on background brightness
func isLightTheme(theme *chroma.Style) bool {
	bg := theme.Get(chroma.Background)
	if bg.Background.IsSet() {
		return bg.Background.Brightness() > 0.5
	}
	
	// Check theme name as fallback
	name := getThemeName(theme)
	lightThemes := []string{"github", "solarized-light", "tango", "monokailight", 
		"paraiso-light", "pygments", "gruvbox-light", "vs", "visual-studio"}
	
	for _, light := range lightThemes {
		if strings.Contains(strings.ToLower(name), light) {
			return true
		}
	}
	
	return false
}

// getThemeName finds the name of a theme by checking against all registered themes
func getThemeName(theme *chroma.Style) string {
	for _, name := range styles.Names() {
		if styles.Get(name) == theme {
			return name
		}
	}
	return "unknown"
}

// DetectTerminalDarkMode attempts to detect if terminal is in dark mode
func DetectTerminalDarkMode() bool {
	// First check for explicit override
	if override := os.Getenv("SIMTOOL_THEME_MODE"); override != "" {
		switch strings.ToLower(override) {
		case "light":
			return false
		case "dark":
			return true
		}
	}
	
	// Check if we already detected the mode before TUI started
	if detected := GetDetectedMode(); detected != "" {
		return detected == "dark"
	}
	
	// Check COLORFGBG if available
	colorScheme := os.Getenv("COLORFGBG")
	if colorScheme != "" {
		parts := strings.Split(colorScheme, ";")
		if len(parts) >= 2 {
			// Background color index 15 or 7 usually means light background
			bgIndex := strings.TrimSpace(parts[1])
			switch bgIndex {
			case "15", "7":
				return false // Light mode
			case "0", "8":
				return true // Dark mode
			}
		}
	}
	
	// Try OS-specific detection
	result := QueryTerminalBackground()
	if result == "light" {
		return false
	} else if result == "dark" {
		return true
	}
	
	// Default to dark mode if detection fails
	return true
}

// ConvertToLipglossColor converts a hex color string to lipgloss.Color
func ConvertToLipglossColor(hex string) lipgloss.Color {
	return lipgloss.Color(hex)
}