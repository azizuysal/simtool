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

	switch m.viewState {
	case SimulatorListView:
		updateViewportForList(&m.simCursor, &m.simViewport, len(m.simulators), itemsPerScreen)
	case AppListView:
		updateViewportForList(&m.appCursor, &m.appViewport, len(m.apps), itemsPerScreen)
	}
}

// updateViewportForList updates viewport for any list
func updateViewportForList(cursor, viewport *int, totalItems, itemsPerScreen int) {
	// Adjust viewport to keep cursor visible
	if *cursor < *viewport {
		*viewport = *cursor
	} else if *cursor >= *viewport+itemsPerScreen {
		*viewport = *cursor - itemsPerScreen + 1
	}

	// Ensure viewport doesn't go beyond bounds
	maxViewport := totalItems - itemsPerScreen
	if maxViewport < 0 {
		maxViewport = 0
	}
	if *viewport > maxViewport {
		*viewport = maxViewport
	}
	if *viewport < 0 {
		*viewport = 0
	}
}