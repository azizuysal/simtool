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
	itemsPerScreen := CalculateItemsPerScreen(m.height)

	switch m.viewState {
	case SimulatorListView:
		// Calculate items per screen the same way SimulatorList does
		contentHeight := m.height - 8                // Same calculation as in view.go
		simItemsPerScreen := (contentHeight - 2) / 3 // Same as SimulatorList.calculateItemsPerScreen
		if simItemsPerScreen < 1 {
			simItemsPerScreen = 1
		}
		updateViewportForList(&m.simList.cursor, &m.simList.viewport, len(m.simList.simulators), simItemsPerScreen)
	case AllAppsView:
		// Each app entry takes 3 lines (name, bundle ID, simulator name)
		contentHeight := m.height - 8
		appItemsPerScreen := contentHeight / 3
		if appItemsPerScreen < 1 {
			appItemsPerScreen = 1
		}
		updateViewportForList(&m.allApps.cursor, &m.allApps.viewport, len(m.allApps.apps), appItemsPerScreen)
	case AppListView:
		updateViewportForList(&m.appList.cursor, &m.appList.viewport, len(m.appList.apps), itemsPerScreen)
	case FileListView:
		// Calculate available height for content box
		contentHeight := m.height - 8 // Title (4) + Footer (4)

		// Account for header inside content box
		headerLines := 6 // App name (1) + app details (1) + spacing (2) + separator (2)
		if len(m.fileList.breadcrumbs) > 0 {
			headerLines += 2 // Breadcrumb line + spacing
		}

		// Available height for file items
		availableHeight := contentHeight - headerLines

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
