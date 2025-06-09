package components

import (
	"fmt"
	"strings"

	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
	"github.com/azizuysal/simtool/internal/ui"
)

// AppList renders the app list view
type AppList struct {
	Width        int
	Height       int
	Apps         []simulator.App
	Cursor       int
	Viewport     int
	SearchMode   bool
	SearchQuery  string
	SimulatorName string
	Keys         *config.KeysConfig
}

// NewAppList creates a new app list renderer
func NewAppList(width, height int) *AppList {
	return &AppList{
		Width:  width,
		Height: height,
	}
}

// Update updates the list data
func (al *AppList) Update(apps []simulator.App, cursor, viewport int, searchMode bool, searchQuery, simName string, keys *config.KeysConfig) {
	al.Apps = apps
	al.Cursor = cursor
	al.Viewport = viewport
	al.SearchMode = searchMode
	al.SearchQuery = searchQuery
	al.SimulatorName = simName
	al.Keys = keys
}

// Render renders the app list content
func (al *AppList) Render() string {
	if len(al.Apps) == 0 {
		if al.SearchQuery != "" {
			return ui.DetailStyle().Render("No apps match your search")
		}
		return ui.DetailStyle().Render("No apps installed")
	}

	// Calculate items per screen
	itemsPerScreen := al.calculateItemsPerScreen()
	startIdx := al.Viewport
	endIdx := al.Viewport + itemsPerScreen
	if endIdx > len(al.Apps) {
		endIdx = len(al.Apps)
	}

	// Render the list
	return al.renderList(startIdx, endIdx)
}

// GetTitle returns the title for the app list
func (al *AppList) GetTitle(totalCount int) string {
	title := fmt.Sprintf("%s Apps (%d", al.SimulatorName, len(al.Apps))
	if al.SearchQuery != "" {
		title += fmt.Sprintf(" of %d)", totalCount)
	} else {
		title += ")"
	}
	return title
}

// GetFooter returns the footer for the app list
func (al *AppList) GetFooter() string {
	if al.Keys == nil {
		// Fallback to default if keys not set
		footer := ""
		if al.SearchMode {
			footer = "ESC: exit search • ↑/↓: navigate • →/Enter: select"
		} else {
			footer = "↑/k: up • ↓/j: down • →/l: files • space: open in Finder • /: search • ←/h: back • q: quit"
		}
		// Add scroll info
		itemsPerScreen := al.calculateItemsPerScreen()
		scrollInfo := ui.FormatScrollInfo(al.Viewport, itemsPerScreen, len(al.Apps))
		return footer + scrollInfo
	}
	
	// Build footer from configured keys
	var parts []string
	
	if al.SearchMode {
		if esc := al.Keys.FormatKeyAction("escape", "exit search"); esc != "" {
			parts = append(parts, esc)
		}
		if up := al.Keys.FormatKeyAction("up", "navigate"); up != "" {
			parts = append(parts, config.FormatKeys(al.Keys.Up)+"/"+config.FormatKeys(al.Keys.Down)+": navigate")
		}
		if right := al.Keys.FormatKeyAction("right", "select"); right != "" {
			enter := config.FormatKeys(al.Keys.Enter)
			if enter != "" {
				parts = append(parts, config.FormatKeys(al.Keys.Right)+"/"+enter+": select")
			} else {
				parts = append(parts, right)
			}
		}
	} else {
		if up := al.Keys.FormatKeyAction("up", "up"); up != "" {
			parts = append(parts, up)
		}
		if down := al.Keys.FormatKeyAction("down", "down"); down != "" {
			parts = append(parts, down)
		}
		if right := al.Keys.FormatKeyAction("right", "files"); right != "" {
			parts = append(parts, right)
		}
		if open := al.Keys.FormatKeyAction("open", "open in Finder"); open != "" {
			parts = append(parts, open)
		}
		if search := al.Keys.FormatKeyAction("search", "search"); search != "" {
			parts = append(parts, search)
		}
		if left := al.Keys.FormatKeyAction("left", "back"); left != "" {
			parts = append(parts, left)
		}
		if quit := al.Keys.FormatKeyAction("quit", "quit"); quit != "" {
			parts = append(parts, quit)
		}
	}
	
	footer := strings.Join(parts, " • ")
	
	// Add scroll info
	itemsPerScreen := al.calculateItemsPerScreen()
	scrollInfo := ui.FormatScrollInfo(al.Viewport, itemsPerScreen, len(al.Apps))
	return footer + scrollInfo
}

// GetStatus returns the status message for the app list
func (al *AppList) GetStatus() string {
	if al.SearchMode {
		searchStatus := fmt.Sprintf("Search: %s", al.SearchQuery)
		if al.SearchQuery == "" {
			searchStatus = "Search: (type to filter)"
		}
		return ui.SearchStyle().Render(searchStatus)
	}
	return ""
}

// calculateItemsPerScreen calculates how many items fit on screen
func (al *AppList) calculateItemsPerScreen() int {
	// Each item takes 3 lines (name + details + blank line)
	// Account for borders and padding
	availableHeight := al.Height - 2 // Border takes 2 lines
	itemsPerScreen := availableHeight / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	return itemsPerScreen
}

// renderList renders the visible apps
func (al *AppList) renderList(startIdx, endIdx int) string {
	var s strings.Builder
	innerWidth := al.Width - 4 // Account for padding

	for i := startIdx; i < endIdx; i++ {
		app := al.Apps[i]

		// Format app details
		sizeText := simulator.FormatSize(app.Size)
		modTimeText := simulator.FormatModTime(app.ModTime)
		detailText := fmt.Sprintf("%s • %s", app.BundleID, sizeText)
		if app.Version != "" {
			detailText = fmt.Sprintf("%s • v%s • %s", app.BundleID, app.Version, sizeText)
		}
		if modTimeText != "" {
			detailText = fmt.Sprintf("%s • %s", detailText, modTimeText)
		}

		if i == al.Cursor {
			// Selected item
			line1 := fmt.Sprintf("▶ %s", app.Name)
			line2 := fmt.Sprintf("  %s", detailText)

			// Pad to full width
			line1 = ui.PadLine(line1, innerWidth)
			line2 = ui.PadLine(line2, innerWidth)

			s.WriteString(ui.SelectedStyle().Render(line1))
			s.WriteString("\n")
			s.WriteString(ui.SelectedStyle().Render(line2))
		} else {
			// Non-selected item
			s.WriteString(ui.ListItemStyle().Copy().Inherit(ui.NameStyle()).Render(app.Name))
			s.WriteString("\n")
			s.WriteString(ui.ListItemStyle().Copy().Inherit(ui.DetailStyle()).Render(detailText))
		}

		if i < endIdx-1 {
			s.WriteString("\n\n")
		}
	}

	return s.String()
}