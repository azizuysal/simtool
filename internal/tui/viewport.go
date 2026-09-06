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

// updateViewport adjusts the viewport to keep cursor visible. Takes a
// Model by value and returns the updated Model so callers follow the
// same "m = m.foo()" pattern as the rest of the package. See the note
// in model.go about receiver conventions.
func (m Model) updateViewport() Model {
	_, contentHeight := m.contentDimensions()
	itemsPerScreen := max(1, (contentHeight-2)/3)

	switch m.viewState {
	case SimulatorListView:
		updateViewportForList(&m.simList.cursor, &m.simList.viewport, len(m.simList.simulators), itemsPerScreen)
	case AllAppsView:
		updateViewportForList(&m.allApps.cursor, &m.allApps.viewport, len(m.allApps.apps), itemsPerScreen)
	case AppListView:
		updateViewportForList(&m.appList.cursor, &m.appList.viewport, len(m.appList.apps), itemsPerScreen)
	case FileListView:
		// Account for header inside content box
		headerLines := 5 // App name, details, separator, and two blank lines
		if len(m.fileList.breadcrumbs) > 0 {
			headerLines += 2 // Breadcrumb line + spacing
		}

		// Available height for file items
		availableHeight := contentHeight - 2 - headerLines

		// Each file item takes 3 lines (name + details + spacing)
		// But we need to ensure we don't count partial items
		actualFileItems := availableHeight / 3
		if actualFileItems < 1 {
			actualFileItems = 1
		}

		updateViewportForList(&m.fileList.cursor, &m.fileList.viewport, len(m.fileList.files), actualFileItems)
	}
	return m
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

// CalculateSimulatorViewport calculates the viewport position for simulator list
func CalculateSimulatorViewport(currentViewport, currentCursor, totalItems, terminalHeight int) int {
	itemsPerScreen := CalculateItemsPerScreen(terminalHeight)
	viewport := currentViewport
	cursor := currentCursor

	updateViewportForList(&cursor, &viewport, totalItems, itemsPerScreen)
	return viewport
}

// CalculateFileListViewport calculates the viewport position for file list
func CalculateFileListViewport(currentViewport, currentCursor, totalItems, terminalHeight, headerLines int) int {
	availableHeight := terminalHeight - 8 - headerLines
	actualFileItems := availableHeight / 3
	if actualFileItems < 1 {
		actualFileItems = 1
	}

	viewport := currentViewport
	cursor := currentCursor

	updateViewportForList(&cursor, &viewport, totalItems, actualFileItems)
	return viewport
}
