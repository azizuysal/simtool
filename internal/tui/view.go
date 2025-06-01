package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return ui.ErrorStyle.Render("Error: " + m.err.Error())
	}

	switch m.viewState {
	case SimulatorListView:
		return m.viewSimulatorList()
	case AppListView:
		return m.viewAppList()
	default:
		return m.viewSimulatorList()
	}
}

// viewSimulatorList renders the simulator list view
func (m Model) viewSimulatorList() string {
	if len(m.simulators) == 0 {
		return "Loading simulators..."
	}

	var s strings.Builder

	// Header
	headerText := fmt.Sprintf("iOS Simulators (%d)", len(m.simulators))
	s.WriteString(ui.FormatHeader(headerText, m.width))

	// Calculate visible range
	itemsPerScreen := CalculateItemsPerScreen(m.height)
	startIdx := m.simViewport
	endIdx := m.simViewport + itemsPerScreen
	if endIdx > len(m.simulators) {
		endIdx = len(m.simulators)
	}

	// Calculate content width
	contentWidth := m.width - 6 // Account for borders and padding
	if contentWidth < 50 {
		contentWidth = 50
	}

	// Build list content
	listContent := m.renderSimulatorList(startIdx, endIdx, contentWidth)
	
	// Pad content to fill the screen height
	paddedContent := m.padContentToHeight(listContent, itemsPerScreen)

	// Apply border and center
	borderedList := ui.BorderStyle.Width(contentWidth).Render(paddedContent)
	s.WriteString(m.centerContent(borderedList))

	// Status message
	if m.statusMessage != "" {
		s.WriteString("\n")
		statusStyle := ui.FooterStyle.Copy()
		if strings.Contains(m.statusMessage, "Error") || strings.Contains(m.statusMessage, "No apps installed") {
			statusStyle = ui.ErrorStyle
		} else if strings.Contains(m.statusMessage, "successfully") {
			statusStyle = statusStyle.Foreground(lipgloss.Color("42"))
		}
		
		// Center the status message
		if m.width > lipgloss.Width(m.statusMessage) {
			leftPadding := (m.width - lipgloss.Width(m.statusMessage)) / 2
			s.WriteString(strings.Repeat(" ", leftPadding))
		}
		s.WriteString(statusStyle.Render(m.statusMessage))
		s.WriteString("\n")
	} else {
		s.WriteString("\n\n")
	}

	// Footer
	footerText := "↑/k: up • ↓/j: down • enter/→: apps • r: run • q: quit"
	scrollInfo := ui.FormatScrollInfo(m.simViewport, itemsPerScreen, len(m.simulators))
	s.WriteString(ui.FormatFooter(footerText+scrollInfo, 
		lipgloss.Width(strings.Split(borderedList, "\n")[0]), m.width))

	return s.String()
}

// renderSimulatorList renders the visible simulators
func (m Model) renderSimulatorList(startIdx, endIdx int, contentWidth int) string {
	var listContent strings.Builder
	innerWidth := contentWidth - 4 // Account for padding

	for i := startIdx; i < endIdx; i++ {
		sim := m.simulators[i]

		// Format app count text
		appCountText := ""
		if sim.AppCount > 0 {
			appCountText = fmt.Sprintf(" • %d app", sim.AppCount)
			if sim.AppCount > 1 {
				appCountText += "s"
			}
		} else if sim.AppCount == 0 {
			// Show "0 apps" for both running and non-running simulators
			appCountText = " • 0 apps"
		}

		if i == m.simCursor {
			// Selected item
			line1 := fmt.Sprintf("▶ %s", sim.Name)
			line2 := fmt.Sprintf("  %s • %s%s", sim.Runtime, sim.StateDisplay(), appCountText)

			// Pad to full width
			line1 = ui.PadLine(line1, innerWidth)
			line2 = ui.PadLine(line2, innerWidth)

			listContent.WriteString(ui.SelectedStyle.Render(line1))
			listContent.WriteString("\n")
			listContent.WriteString(ui.SelectedStyle.Render(line2))
		} else {
			// Non-selected item
			var nameStyle, detailStyle lipgloss.Style
			if sim.IsRunning() {
				nameStyle = ui.ListItemStyle.Copy().Inherit(ui.NameStyle).Inherit(ui.BootedStyle)
				detailStyle = ui.ListItemStyle.Copy().Inherit(ui.BootedStyle)
			} else {
				nameStyle = ui.ListItemStyle.Copy().Inherit(ui.NameStyle)
				detailStyle = ui.ListItemStyle.Copy().Inherit(ui.DetailStyle)
			}

			listContent.WriteString(nameStyle.Render(sim.Name))
			listContent.WriteString("\n")
			listContent.WriteString(detailStyle.Render(sim.Runtime + " • " + sim.StateDisplay() + appCountText))
		}

		if i < endIdx-1 {
			listContent.WriteString("\n\n")
		}
	}

	return listContent.String()
}

// centerContent centers content horizontally
func (m Model) centerContent(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	contentWidth := lipgloss.Width(lines[0])
	if m.width <= contentWidth {
		return content
	}

	var result strings.Builder
	leftPadding := (m.width - contentWidth) / 2
	paddingStr := strings.Repeat(" ", leftPadding)

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(paddingStr)
		result.WriteString(line)
	}

	return result.String()
}

// viewAppList renders the app list view
func (m Model) viewAppList() string {
	if m.loadingApps {
		return "Loading apps..."
	}
	
	if m.selectedSim == nil {
		return "No simulator selected"
	}

	var s strings.Builder

	// Header with simulator name
	headerText := fmt.Sprintf("%s Apps (%d)", m.selectedSim.Name, len(m.apps))
	s.WriteString(ui.FormatHeader(headerText, m.width))

	// Calculate visible range
	itemsPerScreen := CalculateItemsPerScreen(m.height)
	startIdx := m.appViewport
	endIdx := m.appViewport + itemsPerScreen
	if endIdx > len(m.apps) {
		endIdx = len(m.apps)
	}

	// Calculate content width
	contentWidth := m.width - 6
	if contentWidth < 50 {
		contentWidth = 50
	}

	// Build app list content
	var listContent strings.Builder
	innerWidth := contentWidth - 4

	if len(m.apps) == 0 {
		listContent.WriteString(ui.DetailStyle.Render("No apps installed"))
	} else {
		for i := startIdx; i < endIdx; i++ {
			app := m.apps[i]

			// Format app details
			sizeText := simulator.FormatSize(app.Size)
			detailText := fmt.Sprintf("%s • %s", app.BundleID, sizeText)
			if app.Version != "" {
				detailText = fmt.Sprintf("%s • v%s • %s", app.BundleID, app.Version, sizeText)
			}

			if i == m.appCursor {
				// Selected item
				line1 := fmt.Sprintf("▶ %s", app.Name)
				line2 := fmt.Sprintf("  %s", detailText)

				// Pad to full width
				line1 = ui.PadLine(line1, innerWidth)
				line2 = ui.PadLine(line2, innerWidth)

				listContent.WriteString(ui.SelectedStyle.Render(line1))
				listContent.WriteString("\n")
				listContent.WriteString(ui.SelectedStyle.Render(line2))
			} else {
				// Non-selected item
				listContent.WriteString(ui.ListItemStyle.Copy().Inherit(ui.NameStyle).Render(app.Name))
				listContent.WriteString("\n")
				listContent.WriteString(ui.ListItemStyle.Copy().Inherit(ui.DetailStyle).Render(detailText))
			}

			if i < endIdx-1 {
				listContent.WriteString("\n\n")
			}
		}
	}
	
	// Pad content to fill the screen height
	paddedContent := m.padContentToHeight(listContent.String(), itemsPerScreen)

	// Apply border and center
	borderedList := ui.BorderStyle.Width(contentWidth).Render(paddedContent)
	s.WriteString(m.centerContent(borderedList))

	s.WriteString("\n\n")

	// Footer
	footerText := "↑/k: up • ↓/j: down • ←: back • q: quit"
	scrollInfo := ui.FormatScrollInfo(m.appViewport, itemsPerScreen, len(m.apps))
	s.WriteString(ui.FormatFooter(footerText+scrollInfo,
		lipgloss.Width(strings.Split(borderedList, "\n")[0]), m.width))

	return s.String()
}

// padContentToHeight pads content to fill the expected height
func (m Model) padContentToHeight(content string, itemsPerScreen int) string {
	lines := strings.Split(content, "\n")
	currentLines := len(lines)
	
	// Calculate expected lines: itemsPerScreen * 3 (2 lines per item + 1 blank line)
	// Subtract 1 because the last item doesn't have a trailing blank line
	expectedLines := itemsPerScreen * 3 - 1
	if itemsPerScreen > 0 && expectedLines < 2 {
		expectedLines = 2 // Minimum of 2 lines
	}
	
	// If content is already at or above expected height, return as is
	if currentLines >= expectedLines {
		return content
	}
	
	// Add empty lines to reach expected height
	var result strings.Builder
	result.WriteString(content)
	for i := currentLines; i < expectedLines; i++ {
		result.WriteString("\n")
	}
	
	return result.String()
}