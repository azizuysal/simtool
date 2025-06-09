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
			m.allApps,
			m.allAppsCursor,
			m.allAppsViewport,
			m.width,
			m.height,
			m.allAppsSearchMode,
			m.allAppsSearchQuery,
			m.loadingAllApps,
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
	simList.Update(filteredSims, m.simCursor, m.simViewport, m.filterActive, m.simSearchMode, m.simSearchQuery, &m.config.Keys)

	// Get title
	title = simList.GetTitle(len(m.simulators))

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.loadingSimulators {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", simList.Render(), false)
	}

	// Get footer
	footer = simList.GetFooter()

	// Get status
	if m.loadingSimulators {
		status = ui.LoadingStyle().Render("Loading simulators...")
	} else if m.statusMessage != "" {
		if strings.Contains(m.statusMessage, "Error") || strings.Contains(m.statusMessage, "No apps installed") {
			status = ui.ErrorStyle().Render(m.statusMessage)
		} else if strings.Contains(m.statusMessage, "successfully") {
			status = ui.FooterStyle().Copy().Foreground(ui.SuccessColor()).Render(m.statusMessage)
		} else {
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	} else {
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
	if m.selectedSim != nil {
		simName = m.selectedSim.Name
	}
	appList.Update(filteredApps, m.appCursor, m.appViewport, m.appSearchMode, m.appSearchQuery, simName, &m.config.Keys)

	// Get title
	title = appList.GetTitle(len(m.apps))

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.loadingApps {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", appList.Render(), false)
	}

	// Get footer
	footer = appList.GetFooter()

	// Get status
	if m.loadingApps {
		status = ui.LoadingStyle().Render("Loading apps...")
	} else if m.statusMessage != "" {
		if strings.Contains(m.statusMessage, "Error") {
			status = ui.ErrorStyle().Render(m.statusMessage)
		} else {
			status = ui.FooterStyle().Render(m.statusMessage)
		}
	} else {
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
	fileList.Update(m.files, m.fileCursor, m.fileViewport, m.selectedApp, m.breadcrumbs, &m.config.Keys)

	// Get title
	title = fileList.GetTitle()

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.loadingFiles {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", fileList.Render(), false)
	}

	// Get footer
	footer = fileList.GetFooter()

	// Get status
	if m.loadingFiles {
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
	viewer.Update(m.viewingFile, m.fileContent, m.contentViewport, m.contentOffset, m.svgWarning, &m.config.Keys)

	// Get title
	title = viewer.GetTitle()

	// Get content
	// Create content box for all file types
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.loadingContent {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		rawContent := viewer.Render()
		content = contentBox.Render("", rawContent, false)
	}

	// Get footer
	footer = viewer.GetFooter()

	// Get status
	if m.loadingContent {
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
	tableList.Update(m.databaseInfo, m.viewingDatabase, m.tableCursor, m.tableViewport, &m.config.Keys)

	// Get title
	title = tableList.GetTitle()

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.loadingDatabase {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", tableList.Render(), false)
	}

	// Get footer
	footer = tableList.GetFooter()

	// Get status
	if m.loadingDatabase {
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
	tableContent.Update(m.selectedTable, m.tableData, m.viewingDatabase, m.tableDataViewport, m.tableDataOffset, &m.config.Keys)

	// Get title
	title = tableContent.GetTitle()

	// Get content
	// Create content box
	contentBox := components.NewContentBox(contentWidth, contentHeight)
	if m.loadingTableData {
		// Show empty content while loading
		content = contentBox.Render("", "", false)
	} else {
		content = contentBox.Render("", tableContent.Render(), false)
	}

	// Get footer
	footer = tableContent.GetFooter()

	// Get status
	if m.loadingTableData {
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