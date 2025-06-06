package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// SimulatorList renders the simulator list view
type SimulatorList struct {
	Width         int
	Height        int
	Simulators    []simulator.Item
	Cursor        int
	Viewport      int
	FilterActive  bool
	SearchMode    bool
	SearchQuery   string
}

// NewSimulatorList creates a new simulator list renderer
func NewSimulatorList(width, height int) *SimulatorList {
	return &SimulatorList{
		Width:  width,
		Height: height,
	}
}

// Update updates the list data
func (sl *SimulatorList) Update(simulators []simulator.Item, cursor, viewport int, filterActive, searchMode bool, searchQuery string) {
	sl.Simulators = simulators
	sl.Cursor = cursor
	sl.Viewport = viewport
	sl.FilterActive = filterActive
	sl.SearchMode = searchMode
	sl.SearchQuery = searchQuery
}

// Render renders the simulator list content
func (sl *SimulatorList) Render() string {
	if len(sl.Simulators) == 0 {
		return ui.DetailStyle().Render("No simulators found")
	}

	// Calculate items per screen
	itemsPerScreen := sl.calculateItemsPerScreen()
	startIdx := sl.Viewport
	endIdx := sl.Viewport + itemsPerScreen
	if endIdx > len(sl.Simulators) {
		endIdx = len(sl.Simulators)
	}

	// Render the list
	return sl.renderList(startIdx, endIdx)
}

// GetTitle returns the title for the simulator list
func (sl *SimulatorList) GetTitle(totalCount int) string {
	title := fmt.Sprintf("iOS Simulators (%d", len(sl.Simulators))
	if sl.FilterActive || sl.SearchQuery != "" {
		title += fmt.Sprintf(" of %d)", totalCount)
	} else {
		title += ")"
	}
	return title
}

// GetFooter returns the footer for the simulator list
func (sl *SimulatorList) GetFooter() string {
	footer := ""
	if sl.SearchMode {
		footer = "ESC: exit search • ↑/↓: navigate • →/Enter: select"
	} else {
		footer = "↑/k: up • ↓/j: down • →/l: apps • space: run • f: filter • /: search • q: quit"
	}

	// Add scroll info
	itemsPerScreen := sl.calculateItemsPerScreen()
	scrollInfo := ui.FormatScrollInfo(sl.Viewport, itemsPerScreen, len(sl.Simulators))
	return footer + scrollInfo
}

// GetStatus returns the status message for the simulator list
func (sl *SimulatorList) GetStatus() string {
	if sl.SearchMode {
		searchStatus := fmt.Sprintf("Search: %s", sl.SearchQuery)
		if sl.SearchQuery == "" {
			searchStatus = "Search: (type to filter)"
		}
		return ui.SearchStyle().Render(searchStatus)
	} else if sl.FilterActive {
		return ui.SearchStyle().Render("Filter: Showing only simulators with apps")
	}
	return ""
}

// calculateItemsPerScreen calculates how many items fit on screen
func (sl *SimulatorList) calculateItemsPerScreen() int {
	// Each item takes 3 lines (name + details + blank line)
	// ContentBox will clip content to Height-2, so account for that
	availableHeight := sl.Height - 2
	itemsPerScreen := availableHeight / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	return itemsPerScreen
}

// renderList renders the visible simulators
func (sl *SimulatorList) renderList(startIdx, endIdx int) string {
	var s strings.Builder
	innerWidth := sl.Width - 4 // Account for padding

	for i := startIdx; i < endIdx; i++ {
		sim := sl.Simulators[i]

		// Format app count text
		appCountText := ""
		if sim.AppCount > 0 {
			appCountText = fmt.Sprintf(" • %d app", sim.AppCount)
			if sim.AppCount > 1 {
				appCountText += "s"
			}
		} else {
			appCountText = " • 0 apps"
		}

		if i == sl.Cursor {
			// Selected item
			line1 := fmt.Sprintf("▶ %s", sim.Name)
			line2 := fmt.Sprintf("  %s • %s%s", sim.Runtime, sim.StateDisplay(), appCountText)

			// Pad to full width
			line1 = ui.PadLine(line1, innerWidth)
			line2 = ui.PadLine(line2, innerWidth)

			s.WriteString(ui.SelectedStyle().Render(line1))
			s.WriteString("\n")
			s.WriteString(ui.SelectedStyle().Render(line2))
		} else {
			// Non-selected item
			var nameStyle, detailStyle lipgloss.Style
			if sim.IsRunning() {
				nameStyle = ui.ListItemStyle().Copy().Inherit(ui.NameStyle()).Inherit(ui.BootedStyle())
				detailStyle = ui.ListItemStyle().Copy().Inherit(ui.BootedStyle())
			} else {
				nameStyle = ui.ListItemStyle().Copy().Inherit(ui.NameStyle())
				detailStyle = ui.ListItemStyle().Copy().Inherit(ui.DetailStyle())
			}

			s.WriteString(nameStyle.Render(sim.Name))
			s.WriteString("\n")
			s.WriteString(detailStyle.Render(sim.Runtime + " • " + sim.StateDisplay() + appCountText))
		}

		if i < endIdx-1 {
			s.WriteString("\n\n")
		}
	}

	return s.String()
}