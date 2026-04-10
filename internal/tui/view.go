package tui

import (
	"strings"

	"github.com/azizuysal/simtool/internal/tui/components"
	"github.com/azizuysal/simtool/internal/tui/components/file_viewer"
	"github.com/azizuysal/simtool/internal/ui"
)

// View renders the UI using the component system
func (m Model) View() string {
	// Handle errors
	if m.err != nil && m.viewState != AllAppsView {
		return ui.ErrorStyle().Render("Error: " + m.err.Error())
	}

	// Special handling for AllAppsView which returns complete layout
	if m.viewState == AllAppsView {
		return components.AllAppsListView(
			m.allApps.apps,
			m.allApps.cursor,
			m.allApps.viewport,
			m.width,
			m.height,
			m.allApps.searchMode,
			m.allApps.searchQuery,
			m.allApps.loading,
			m.err,
			&m.config.Keys,
		)
	}

	// Create layout
	layout := components.NewLayout(m.width, m.height)

	// Get view-specific content
	var title, content, footer, status string

	switch m.viewState {
	case SimulatorListView:
		title, content, footer, status = m.renderSimulatorListView()
	case AppListView:
		title, content, footer, status = m.renderAppListView()
	case FileListView:
		title, content, footer, status = m.renderFileListView()
	case FileViewerView:
		title, content, footer, status = m.renderFileViewerView()
	case DatabaseTableListView:
		title, content, footer, status = m.renderDatabaseTableListView()
	case DatabaseTableContentView:
		title, content, footer, status = m.renderDatabaseTableContentView()
	default:
		title, content, footer, status = m.renderSimulatorListView()
	}

	// Render with layout
	return layout.Render(title, content, footer, status)
}

// renderSimulatorListView renders the simulator list using components
func (m Model) renderSimulatorListView() (title, content, footer, status string) {
	// Get filtered simulators
	filteredSims := m.getFilteredAndSearchedSimulators()

	// Calculate available space for content
	// Title takes ~4 lines (padding + title + padding)
	// Footer takes ~4 lines (padding + status + footer + padding)
	contentHeight := m.height - 8
	contentWidth := m.width - 6 // Account for side margins

	// Create simulator list component
	simList := components.NewSimulatorList(contentWidth, contentHeight)
	simList.Update(filteredSims, m.simList.cursor, m.simList.viewport, m.simList.filterActive, m.simList.searchMode, m.simList.searchQuery, &m.config.Keys)

	// Get title
	title = simList.GetTitle(len(m.simList.simulators))

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.simList.loading {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", simList.Render(), false)
	}

	// Get footer
	footer = simList.GetFooter()

	// Get status
	switch {
	case m.simList.loading:
		status = ui.LoadingStyle().Render("Loading simulators...")
	case m.statusMessage != "":
		switch {
		case strings.Contains(m.statusMessage, "Error") || strings.Contains(m.statusMessage, "No apps installed"):
			status = ui.ErrorStyle().Render(m.statusMessage)
		case strings.Contains(m.statusMessage, "successfully"):
			status = ui.FooterStyle().Foreground(ui.SuccessColor()).Render(m.statusMessage)
		default:
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	default:
		status = simList.GetStatus()
	}

	return
}

// renderAppListView renders the app list using components
func (m Model) renderAppListView() (title, content, footer, status string) {
	// Get filtered apps
	filteredApps := m.getFilteredAndSearchedApps()

	// Calculate available space
	contentHeight := m.height - 8
	contentWidth := m.width - 6

	// Create app list component
	appList := components.NewAppList(contentWidth, contentHeight)
	simName := ""
	if m.appList.selectedSim != nil {
		simName = m.appList.selectedSim.Name
	}
	appList.Update(filteredApps, m.appList.cursor, m.appList.viewport, m.appList.searchMode, m.appList.searchQuery, simName, &m.config.Keys)

	// Get title
	title = appList.GetTitle(len(m.appList.apps))

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.appList.loading {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", appList.Render(), false)
	}

	// Get footer
	footer = appList.GetFooter()

	// Get status
	switch {
	case m.appList.loading:
		status = ui.LoadingStyle().Render("Loading apps...")
	case m.statusMessage != "":
		if strings.Contains(m.statusMessage, "Error") {
			status = ui.ErrorStyle().Render(m.statusMessage)
		} else {
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	default:
		status = appList.GetStatus()
	}

	return
}

// renderFileListView renders the file list using components
func (m Model) renderFileListView() (title, content, footer, status string) {
	// Calculate available space
	contentHeight := m.height - 8
	contentWidth := m.width - 6

	// Create file list component
	fileList := components.NewFileList(contentWidth, contentHeight)
	fileList.Update(m.fileList.files, m.fileList.cursor, m.fileList.viewport, m.fileList.selectedApp, m.fileList.breadcrumbs, &m.config.Keys)

	// Get title
	title = fileList.GetTitle()

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.fileList.loading {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", fileList.Render(), false)
	}

	// Get footer
	footer = fileList.GetFooter()

	// Get status
	if m.fileList.loading {
		status = ui.LoadingStyle().Render("Loading files...")
	} else if m.statusMessage != "" {
		if strings.Contains(m.statusMessage, "Error") {
			status = ui.ErrorStyle().Render(m.statusMessage)
		} else {
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	}

	return
}

// renderFileViewerView renders the file viewer using components
func (m Model) renderFileViewerView() (title, content, footer, status string) {
	// Calculate available space for content
	contentHeight := m.height - 8
	contentWidth := m.width - 6

	// Create file viewer component with content dimensions
	viewer := file_viewer.NewFileViewer(contentWidth, contentHeight)
	viewer.Update(m.fileViewer.file, m.fileViewer.content, m.fileViewer.contentViewport, m.fileViewer.contentOffset, m.fileViewer.svgWarning, &m.config.Keys)

	// Get title
	title = viewer.GetTitle()

	// Get content
	// Create content box for all file types
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.fileViewer.loading {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		rawContent := viewer.Render()
		content = contentBox.Render("", rawContent, false)
	}

	// Get footer
	footer = viewer.GetFooter()

	// Get status
	if m.fileViewer.loading {
		status = ui.LoadingStyle().Render("Loading file...")
	} else if m.statusMessage != "" {
		if strings.Contains(m.statusMessage, "Error") {
			status = ui.ErrorStyle().Render(m.statusMessage)
		} else {
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	} else if viewerStatus := viewer.GetStatus(); viewerStatus != "" {
		status = viewerStatus
	}

	return
}

// renderDatabaseTableListView renders the database table list using components
func (m Model) renderDatabaseTableListView() (title, content, footer, status string) {
	// Calculate available space
	contentHeight := m.height - 8
	contentWidth := m.width - 6

	// Create database table list component
	tableList := components.NewDatabaseTableList(contentWidth, contentHeight)
	tableList.Update(m.dbTables.info, m.dbTables.file, m.dbTables.cursor, m.dbTables.viewport, &m.config.Keys)

	// Get title
	title = tableList.GetTitle()

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.dbTables.loading {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", tableList.Render(), false)
	}

	// Get footer
	footer = tableList.GetFooter()

	// Get status
	if m.dbTables.loading {
		status = ui.LoadingStyle().Render("Loading database...")
	} else if m.statusMessage != "" {
		if strings.Contains(m.statusMessage, "Error") {
			status = ui.ErrorStyle().Render(m.statusMessage)
		} else {
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	}

	return
}

// renderDatabaseTableContentView renders individual table content using components
func (m Model) renderDatabaseTableContentView() (title, content, footer, status string) {
	// Calculate available space
	contentHeight := m.height - 8
	contentWidth := m.width - 6

	// Create database table content component
	tableContent := components.NewDatabaseTableContent(contentWidth, contentHeight)
	tableContent.Update(m.dbContent.table, m.dbContent.data, m.dbTables.file, m.dbContent.viewport, m.dbContent.offset, &m.config.Keys)

	// Get title
	title = tableContent.GetTitle()

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.dbContent.loading {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", tableContent.Render(), false)
	}

	// Get footer
	footer = tableContent.GetFooter()

	// Get status
	if m.dbContent.loading {
		status = ui.LoadingStyle().Render("Loading table data...")
	} else if m.statusMessage != "" {
		if strings.Contains(m.statusMessage, "Error") {
			status = ui.ErrorStyle().Render(m.statusMessage)
		} else {
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	}

	return
}
