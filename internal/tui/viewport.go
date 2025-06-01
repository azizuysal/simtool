package tui

// CalculateItemsPerScreen calculates how many items fit on screen
func CalculateItemsPerScreen(height int) int {
	// Each item takes 2 lines + 1 line spacing = 3 lines
	// Reserve 8 lines for header, footer, and border
	itemsPerScreen := (height - 8) / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	return itemsPerScreen
}

// updateViewport adjusts the viewport to keep cursor visible
func (m *Model) updateViewport() {
	itemsPerScreen := CalculateItemsPerScreen(m.height)

	// Adjust viewport to keep cursor visible
	if m.cursor < m.viewport {
		m.viewport = m.cursor
	} else if m.cursor >= m.viewport+itemsPerScreen {
		m.viewport = m.cursor - itemsPerScreen + 1
	}

	// Ensure viewport doesn't go beyond bounds
	maxViewport := len(m.simulators) - itemsPerScreen
	if maxViewport < 0 {
		maxViewport = 0
	}
	if m.viewport > maxViewport {
		m.viewport = maxViewport
	}
	if m.viewport < 0 {
		m.viewport = 0
	}
}