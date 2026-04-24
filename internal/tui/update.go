package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/azizuysal/simtool/internal/simulator"
	"github.com/azizuysal/simtool/internal/ui"
)

const (
	// textChunkBackStep is how many text lines the viewer jumps back
	// when scrolling past the top of the loaded chunk and needs to
	// re-fetch the preceding window.
	textChunkBackStep = 200
	// binaryChunkBackStep is the equivalent for hex-dump viewing,
	// expressed in hex lines (each HexBytesPerLine bytes).
	binaryChunkBackStep = 256
)

// clearStatusMsg is sent to clear the status message
type clearStatusMsg struct{}

// clearStatusAfter returns a command that clears the status message
// after d. Extracted from the repeated tea.Tick pattern that was
// previously inlined at every error/notification site.
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// flashStatus sets m.statusMessage and returns a command that will
// clear the message after d. Callers that need to batch the clear
// with other commands (e.g. a refresh after a successful boot)
// should set the status themselves and use clearStatusAfter directly.
func (m Model) flashStatus(msg string, d time.Duration) (Model, tea.Cmd) {
	m.statusMessage = msg
	return m, clearStatusAfter(d)
}

// Update handles messages and updates the model. Non-trivial per-message
// logic is delegated to handle* methods below; the dispatcher only knows
// which handler to call for which message type.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m.updateViewport(), nil
	case fetchSimulatorsMsg:
		return m.handleFetchSimulators(msg)
	case fetchAppsMsg:
		return m.handleFetchApps(msg)
	case fetchAllAppsMsg:
		return m.handleFetchAllApps(msg)
	case bootSimulatorMsg:
		return m.handleBootSimulator(msg)
	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil
	case openInFinderMsg:
		if msg.err != nil {
			return m.flashStatus(fmt.Sprintf("Error opening in Finder: %v", msg.err), 3*time.Second)
		}
		return m, nil
	case tickMsg:
		return m.handleTick()
	case themeChangedMsg:
		return m.handleThemeChanged(msg)
	case fetchFilesMsg:
		return m.handleFetchFiles(msg)
	case fetchDatabaseInfoMsg:
		return m.handleFetchDatabaseInfo(msg)
	case fetchTableDataMsg:
		return m.handleFetchTableData(msg)
	case fetchFileContentMsg:
		return m.handleFetchFileContent(msg)
	}
	return m, nil
}

// handleFetchSimulators processes the result of a simulator list fetch,
// clamping the cursor into the new range and refreshing the viewport.
func (m Model) handleFetchSimulators(msg fetchSimulatorsMsg) (Model, tea.Cmd) {
	m.simList.simulators = msg.simulators
	m.err = msg.err
	m.simList.loading = false
	if m.simList.cursor >= len(m.simList.simulators) {
		m.simList.cursor = len(m.simList.simulators) - 1
	}
	if m.simList.cursor < 0 && len(m.simList.simulators) > 0 {
		m.simList.cursor = 0
	}
	return m.updateViewport(), nil
}

// handleFetchApps processes the result of a per-simulator app list
// fetch. Errors and empty results both return the user to the simulator
// list with a flash message.
func (m Model) handleFetchApps(msg fetchAppsMsg) (Model, tea.Cmd) {
	m.appList.apps = msg.apps
	m.appList.loading = false
	if msg.err != nil {
		m.viewState = SimulatorListView
		m.appList.selectedSim = nil
		return m.flashStatus(fmt.Sprintf("Error loading apps: %v", msg.err), 3*time.Second)
	}
	if len(msg.apps) == 0 {
		m.viewState = SimulatorListView
		m.appList.selectedSim = nil
		return m.flashStatus("No apps installed on this simulator", 2*time.Second)
	}
	m.appList.cursor = 0
	m.appList.viewport = 0
	return m.updateViewport(), nil
}

// handleFetchAllApps processes the result of the combined all-apps
// fetch. Errors are surfaced via m.err rather than a flash so the main
// list view can render an error state.
func (m Model) handleFetchAllApps(msg fetchAllAppsMsg) (Model, tea.Cmd) {
	m.allApps.apps = msg.apps
	m.allApps.loading = false
	if msg.err != nil {
		m.err = msg.err
	} else {
		m.allApps.cursor = 0
		m.allApps.viewport = 0
	}
	return m.updateViewport(), nil
}

// handleBootSimulator processes the result of a boot command, batching
// a simulator refresh and status clear on success.
func (m Model) handleBootSimulator(msg bootSimulatorMsg) (Model, tea.Cmd) {
	m.simList.booting = false
	if msg.err != nil {
		return m.flashStatus(fmt.Sprintf("Error: %v", msg.err), 3*time.Second)
	}
	m.statusMessage = "Simulator booted successfully!"
	return m, tea.Batch(
		fetchSimulatorsCmd(m.fetcher),
		clearStatusAfter(3*time.Second),
	)
}

// handleTick runs on the 2-second periodic tick: refreshes simulator
// state, re-schedules the next tick, and opportunistically polls the
// terminal theme for a live switch.
func (m Model) handleTick() (Model, tea.Cmd) {
	cmds := []tea.Cmd{
		fetchSimulatorsCmd(m.fetcher),
		tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	}
	if cmd := m.checkThemeChange(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// handleThemeChanged reloads chroma styles and updates the cached
// theme mode when the terminal switches light/dark.
func (m Model) handleThemeChanged(msg themeChangedMsg) (Model, tea.Cmd) {
	m.currentThemeMode = msg.newMode
	if err := ui.ReloadStyles(); err != nil {
		return m.flashStatus(fmt.Sprintf("Failed to reload theme: %v", err), 2*time.Second)
	}
	return m, nil
}

// handleFetchFiles processes the result of a directory listing fetch,
// restoring cursor/viewport positions saved when the user drilled into
// the directory so going back to a parent lands on the previous entry.
func (m Model) handleFetchFiles(msg fetchFilesMsg) (Model, tea.Cmd) {
	m.fileList.files = msg.files
	m.fileList.loading = false
	if msg.err != nil {
		m.viewState = AppListView
		m.fileList.selectedApp = nil
		m.fileList.currentPath = ""
		return m.flashStatus(fmt.Sprintf("Error loading files: %v", msg.err), 3*time.Second)
	}
	if m.fileList.cursorMemory != nil {
		if cursor, ok := m.fileList.cursorMemory[m.fileList.currentPath]; ok {
			m.fileList.cursor = cursor
			if m.fileList.cursor >= len(m.fileList.files) {
				m.fileList.cursor = len(m.fileList.files) - 1
			}
			if m.fileList.cursor < 0 {
				m.fileList.cursor = 0
			}
		} else {
			m.fileList.cursor = 0
		}
		if viewport, ok := m.fileList.viewportMemory[m.fileList.currentPath]; ok {
			m.fileList.viewport = viewport
		} else {
			m.fileList.viewport = 0
		}
	} else {
		m.fileList.cursor = 0
		m.fileList.viewport = 0
	}
	return m.updateViewport(), nil
}

// handleFetchDatabaseInfo processes the result of a SQLite schema read.
func (m Model) handleFetchDatabaseInfo(msg fetchDatabaseInfoMsg) (Model, tea.Cmd) {
	m.dbTables.loading = false
	if msg.err != nil {
		m.viewState = FileListView
		m.dbTables.file = nil
		return m.flashStatus(fmt.Sprintf("Error loading database: %v", msg.err), 3*time.Second)
	}
	m.dbTables.info = msg.dbInfo
	return m.updateViewport(), nil
}

// handleFetchTableData processes a page of SQLite row data.
func (m Model) handleFetchTableData(msg fetchTableDataMsg) (Model, tea.Cmd) {
	m.dbContent.loading = false
	if msg.err != nil {
		return m.flashStatus(fmt.Sprintf("Error loading table data: %v", msg.err), 3*time.Second)
	}
	m.dbContent.data = msg.data
	m.dbContent.offset = msg.offset
	return m.updateViewport(), nil
}

// handleFetchFileContent processes a chunk of file content for the
// viewer: re-syncs the hex-dump offset for binary files and scans SVG
// source for features the rasterizer can't render.
func (m Model) handleFetchFileContent(msg fetchFileContentMsg) (Model, tea.Cmd) {
	m.fileViewer.loading = false
	if msg.err != nil {
		m.viewState = FileListView
		m.fileViewer.file = nil
		return m.flashStatus(fmt.Sprintf("Error loading file: %v", msg.err), 3*time.Second)
	}
	m.fileViewer.content = msg.content
	if m.fileViewer.content.Type == simulator.FileTypeBinary {
		m.fileViewer.contentOffset = int(m.fileViewer.content.BinaryOffset / simulator.HexBytesPerLine)
	}
	m.fileViewer.svgWarning = detectSVGWarning(m.fileViewer.file)
	return m.updateViewport(), nil
}

// detectSVGWarning returns a non-empty warning string if the given file
// is an SVG whose source references features (embedded raster images,
// filters, foreign objects) that the rasterizer cannot render. Returns
// "" for non-SVG files, missing files, or SVGs with no such features.
func detectSVGWarning(file *simulator.FileInfo) string {
	if file == nil || strings.ToLower(filepath.Ext(file.Path)) != ".svg" {
		return ""
	}
	data, err := os.ReadFile(file.Path)
	if err != nil {
		return ""
	}
	svg := string(data)
	var features []string
	if strings.Contains(svg, "data:image/") ||
		strings.Contains(svg, "xlink:href=\"data:") ||
		strings.Contains(svg, "xlink:href=\"http") {
		features = append(features, "embedded images")
	}
	if strings.Contains(svg, "<filter") || strings.Contains(svg, "filter=") {
		features = append(features, "filters")
	}
	if strings.Contains(svg, "<foreignObject") {
		features = append(features, "foreign objects")
	}
	if len(features) == 0 {
		return ""
	}
	return fmt.Sprintf("Warning: SVG contains unsupported features (%s). Preview may be incomplete.",
		strings.Join(features, ", "))
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode input first
	if m.simList.searchMode && m.viewState == SimulatorListView {
		return m.handleSimulatorSearchInput(msg)
	}
	if m.appList.searchMode && m.viewState == AppListView {
		return m.handleAppSearchInput(msg)
	}
	if m.allApps.searchMode && m.viewState == AllAppsView {
		return m.handleAllAppsSearchInput(msg)
	}

	action := m.keyMap.GetAction(msg.String())

	// Global quit (ignored in search mode)
	if action == "quit" {
		if m.simList.searchMode || m.appList.searchMode || m.allApps.searchMode {
			return m, nil
		}
		return m, tea.Quit
	}

	// Navigation actions clear any pending status message before dispatch.
	if action == "up" || action == "down" || action == "home" || action == "end" {
		m.statusMessage = ""
	}

	switch m.viewState {
	case SimulatorListView:
		return m.handleSimulatorListKey(action)
	case AppListView:
		return m.handleAppListKey(action)
	case AllAppsView:
		return m.handleAllAppsKey(action)
	case FileListView:
		return m.handleFileListKey(action)
	case FileViewerView:
		return m.handleFileViewerKey(action)
	case DatabaseTableListView:
		return m.handleDatabaseTableListKey(action)
	case DatabaseTableContentView:
		return m.handleDatabaseTableContentKey(action)
	}
	return m, nil
}

// handleSimulatorListKey handles key actions in the simulator list view.
func (m Model) handleSimulatorListKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "right":
		filteredSims := m.getFilteredSimulators()
		if len(filteredSims) > 0 && m.simList.cursor < len(filteredSims) {
			sim := filteredSims[m.simList.cursor]
			m.appList.selectedSim = &sim
			m.viewState = AppListView
			m.appList.loading = true
			return m, m.fetchAppsCmd(sim)
		}
	case "up":
		if m.simList.cursor > 0 {
			m.simList.cursor--
			m = m.updateViewport()
		}
	case "down":
		filteredSims := m.getFilteredAndSearchedSimulators()
		if m.simList.cursor < len(filteredSims)-1 {
			m.simList.cursor++
			m = m.updateViewport()
		}
	case "home":
		m.simList.cursor = 0
		m.simList.viewport = 0
	case "end":
		filteredSims := m.getFilteredAndSearchedSimulators()
		m.simList.cursor = len(filteredSims) - 1
		if m.simList.cursor < 0 {
			m.simList.cursor = 0
		}
		m = m.updateViewport()
	case "filter":
		m.simList.filterActive = !m.simList.filterActive
		// Reset cursor when toggling filter
		m.simList.cursor = 0
		m.simList.viewport = 0
		m = m.updateViewport()
	case "boot", "open":
		filteredSims := m.getFilteredSimulators()
		if len(filteredSims) > 0 && m.simList.cursor < len(filteredSims) {
			sim := filteredSims[m.simList.cursor]
			if !sim.IsRunning() && !m.simList.booting {
				m.simList.booting = true
				m.statusMessage = fmt.Sprintf("Booting %s...", sim.Name)
				return m, m.bootSimulatorCmd(sim.UDID)
			} else if sim.IsRunning() {
				return m.flashStatus("Simulator is already running", 2*time.Second)
			}
		}
	case "search":
		m.simList.searchMode = true
		m.simList.searchQuery = ""
		// Reset cursor to 0 when starting search
		m.simList.cursor = 0
		m.simList.viewport = 0
		m = m.updateViewport()
	}
	return m, nil
}

// handleAppListKey handles key actions in the app list view.
func (m Model) handleAppListKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "left":
		m.viewState = SimulatorListView
		m.appList = appListState{}
		m = m.updateViewport()
	case "right":
		if len(m.appList.apps) > 0 {
			app := m.appList.apps[m.appList.cursor]
			m.fileList.selectedApp = &app
			m.viewState = FileListView
			m.fileList.loading = true
			m.fileList.currentPath = app.Container
			m.fileList.basePath = app.Container
			m.fileList.breadcrumbs = []string{}
			m.fileList.cursorMemory = make(map[string]int)
			m.fileList.viewportMemory = make(map[string]int)
			return m, m.fetchFilesCmd(app.Container)
		}
	case "up":
		if m.appList.cursor > 0 {
			m.appList.cursor--
			m = m.updateViewport()
		}
	case "down":
		filteredApps := m.getFilteredAndSearchedApps()
		if m.appList.cursor < len(filteredApps)-1 {
			m.appList.cursor++
			m = m.updateViewport()
		}
	case "home":
		m.appList.cursor = 0
		m.appList.viewport = 0
	case "end":
		filteredApps := m.getFilteredAndSearchedApps()
		m.appList.cursor = len(filteredApps) - 1
		if m.appList.cursor < 0 {
			m.appList.cursor = 0
		}
		m = m.updateViewport()
	case "boot", "open":
		if len(m.appList.apps) > 0 {
			app := m.appList.apps[m.appList.cursor]
			if app.Container != "" {
				// Open the app's container in Finder
				return m, m.openInFinderCmd(app.Container)
			}
		}
	case "search":
		m.appList.searchMode = true
		m.appList.searchQuery = ""
		// Reset cursor to 0 when starting search
		m.appList.cursor = 0
		m.appList.viewport = 0
		m = m.updateViewport()
	}
	return m, nil
}

// handleAllAppsKey handles key actions in the combined all-apps view.
func (m Model) handleAllAppsKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "right":
		filteredApps := m.getFilteredAndSearchedAllApps()
		if len(filteredApps) > 0 && m.allApps.cursor < len(filteredApps) {
			app := filteredApps[m.allApps.cursor]
			m.fileList.selectedApp = &app
			m.viewState = FileListView
			m.fileList.loading = true
			m.fileList.currentPath = app.Container
			m.fileList.basePath = app.Container
			m.fileList.breadcrumbs = []string{}
			m.fileList.cursorMemory = make(map[string]int)
			m.fileList.viewportMemory = make(map[string]int)
			return m, m.fetchFilesCmd(app.Container)
		}
	case "up":
		if m.allApps.cursor > 0 {
			m.allApps.cursor--
			m = m.updateViewport()
		}
	case "down":
		filteredApps := m.getFilteredAndSearchedAllApps()
		if m.allApps.cursor < len(filteredApps)-1 {
			m.allApps.cursor++
			m = m.updateViewport()
		}
	case "boot", "open":
		filteredApps := m.getFilteredAndSearchedAllApps()
		if len(filteredApps) > 0 && m.allApps.cursor < len(filteredApps) {
			app := filteredApps[m.allApps.cursor]
			if app.Container != "" {
				// Open the app's container in Finder
				return m, m.openInFinderCmd(app.Container)
			}
		}
	case "search":
		m.allApps.searchMode = true
		m.allApps.searchQuery = ""
		// Reset cursor to 0 when starting search
		m.allApps.cursor = 0
		m.allApps.viewport = 0
		m = m.updateViewport()
	}
	return m, nil
}

// handleFileListKey handles key actions in the file list view.
func (m Model) handleFileListKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "left":
		if len(m.fileList.breadcrumbs) > 0 {
			// Go up one directory level
			m.fileList.breadcrumbs = m.fileList.breadcrumbs[:len(m.fileList.breadcrumbs)-1]
			newPath := m.fileList.basePath
			if len(m.fileList.breadcrumbs) > 0 {
				newPath = filepath.Join(append([]string{m.fileList.basePath}, m.fileList.breadcrumbs...)...)
			}
			m.fileList.currentPath = newPath
			m.fileList.loading = true
			return m, m.fetchFilesCmd(newPath)
		}
		// At root level, go back to app list or all apps view.
		// Reading selectedApp before the fileList clear is deliberate:
		// it decides which view we return to.
		nextView := AppListView
		if m.fileList.selectedApp != nil && m.fileList.selectedApp.SimulatorUDID != "" {
			nextView = AllAppsView
		}
		m.viewState = nextView
		m.fileList = fileListState{}
		m = m.updateViewport()
	case "right":
		if len(m.fileList.files) > 0 {
			file := m.fileList.files[m.fileList.cursor]
			if file.IsDirectory {
				// Save current cursor position before drilling in
				if m.fileList.cursorMemory == nil {
					m.fileList.cursorMemory = make(map[string]int)
					m.fileList.viewportMemory = make(map[string]int)
				}
				m.fileList.cursorMemory[m.fileList.currentPath] = m.fileList.cursor
				m.fileList.viewportMemory[m.fileList.currentPath] = m.fileList.viewport

				// Drill into the directory
				m.fileList.breadcrumbs = append(m.fileList.breadcrumbs, file.Name)
				m.fileList.currentPath = file.Path
				m.fileList.loading = true
				return m, m.fetchFilesCmd(file.Path)
			}
			// Check if it's a database file
			fileType := simulator.DetectFileType(file.Path)
			if fileType == simulator.FileTypeDatabase {
				// View database tables
				m.dbTables.file = &file
				m.viewState = DatabaseTableListView
				m.dbTables.loading = true
				m.dbTables.cursor = 0
				m.dbTables.viewport = 0
				return m, m.fetchDatabaseInfoCmd(file.Path)
			}
			// View the file
			m.fileViewer.file = &file
			m.viewState = FileViewerView
			m.fileViewer.loading = true
			m.fileViewer.contentOffset = 0
			m.fileViewer.contentViewport = 0
			return m, m.fetchFileContentCmd(file.Path, 0)
		}
	case "up":
		if m.fileList.cursor > 0 {
			m.fileList.cursor--
			m = m.updateViewport()
		}
	case "down":
		if m.fileList.cursor < len(m.fileList.files)-1 {
			m.fileList.cursor++
			m = m.updateViewport()
		}
	case "home":
		m.fileList.cursor = 0
		m.fileList.viewport = 0
	case "end":
		m.fileList.cursor = len(m.fileList.files) - 1
		m = m.updateViewport()
	case "boot", "open":
		if len(m.fileList.files) > 0 {
			file := m.fileList.files[m.fileList.cursor]
			// Open in Finder - for files, this will reveal them in their containing folder
			return m, m.openInFinderCmd(file.Path)
		}
	}
	return m, nil
}

// handleFileViewerKey handles key actions in the file viewer view.
func (m Model) handleFileViewerKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "left":
		m.viewState = FileListView
		m.fileViewer = fileViewerState{}
		m = m.updateViewport()
	case "up":
		if m.fileViewer.content == nil {
			return m, nil
		}
		switch m.fileViewer.content.Type {
		case simulator.FileTypeText:
			if m.fileViewer.contentViewport > 0 {
				m.fileViewer.contentViewport--
			} else if m.fileViewer.contentOffset > 0 {
				// Need to load previous chunk
				newOffset := m.fileViewer.contentOffset - textChunkBackStep
				if newOffset < 0 {
					newOffset = 0
				}
				m.fileViewer.contentOffset = newOffset
				m.fileViewer.loading = true
				return m, m.fetchFileContentCmd(m.fileViewer.file.Path, newOffset)
			}
		case simulator.FileTypeImage:
			if m.fileViewer.contentViewport > 0 {
				m.fileViewer.contentViewport--
			}
		case simulator.FileTypeBinary:
			if m.fileViewer.contentViewport > 0 {
				m.fileViewer.contentViewport--
			} else if m.fileViewer.contentOffset > 0 {
				// Need to load previous chunk
				newOffset := m.fileViewer.contentOffset - binaryChunkBackStep
				if newOffset < 0 {
					newOffset = 0
				}
				m.fileViewer.contentOffset = newOffset
				m.fileViewer.loading = true
				// Convert line offset to hex-dump row offset
				return m, m.fetchFileContentCmd(m.fileViewer.file.Path, newOffset/simulator.HexBytesPerLine)
			}
		case simulator.FileTypeArchive:
			// Allow scrolling through archive entries
			if m.fileViewer.content.ArchiveInfo != nil && m.fileViewer.contentViewport > 0 {
				m.fileViewer.contentViewport--
			}
		}
	case "down":
		if m.fileViewer.content == nil {
			return m, nil
		}
		switch m.fileViewer.content.Type {
		case simulator.FileTypeText:
			itemsPerScreen := CalculateItemsPerScreen(m.height) - 5 // Account for header
			maxViewport := len(m.fileViewer.content.Lines) - itemsPerScreen
			if maxViewport < 0 {
				maxViewport = 0
			}

			if m.fileViewer.contentViewport < maxViewport {
				m.fileViewer.contentViewport++
			} else if m.fileViewer.contentOffset+len(m.fileViewer.content.Lines) < m.fileViewer.content.TotalLines {
				// Need to load more content
				newOffset := m.fileViewer.contentOffset + len(m.fileViewer.content.Lines)
				m.fileViewer.contentOffset = newOffset
				m.fileViewer.contentViewport = 0 // Reset viewport for new chunk
				m.fileViewer.loading = true
				return m, m.fetchFileContentCmd(m.fileViewer.file.Path, newOffset)
			}
		case simulator.FileTypeImage:
			// For images, calculate based on total content lines
			if m.fileViewer.content.ImageInfo != nil && m.fileViewer.content.ImageInfo.Preview != nil {
				// Calculate total lines (metadata + preview)
				totalLines := 8 + len(m.fileViewer.content.ImageInfo.Preview.Rows) // ~8 lines for metadata
				itemsPerScreen := CalculateItemsPerScreen(m.height) - 5
				maxViewport := totalLines - itemsPerScreen
				if maxViewport < 0 {
					maxViewport = 0
				}
				if m.fileViewer.contentViewport < maxViewport {
					m.fileViewer.contentViewport++
				}
			}
		case simulator.FileTypeBinary:
			// Allow scrolling through binary files with lazy loading
			hexLines := simulator.FormatHexDump(m.fileViewer.content.BinaryData, m.fileViewer.content.BinaryOffset)
			itemsPerScreen := CalculateItemsPerScreen(m.height) - 5 // Account for header
			maxViewport := len(hexLines) - itemsPerScreen
			if maxViewport < 0 {
				maxViewport = 0
			}

			if m.fileViewer.contentViewport < maxViewport {
				m.fileViewer.contentViewport++
			} else {
				// Check if we need to load more data
				currentEndByte := m.fileViewer.content.BinaryOffset + int64(len(m.fileViewer.content.BinaryData))
				if currentEndByte < m.fileViewer.content.TotalSize {
					// Load next chunk
					newOffset := m.fileViewer.contentOffset + len(hexLines)
					m.fileViewer.contentOffset = newOffset
					m.fileViewer.contentViewport = 0 // Reset viewport for new chunk
					m.fileViewer.loading = true
					// Load with line offset (total lines from start)
					return m, m.fetchFileContentCmd(m.fileViewer.file.Path, newOffset)
				}
			}
		case simulator.FileTypeArchive:
			// Allow scrolling through archive entries (now 1 line per entry)
			if m.fileViewer.content.ArchiveInfo != nil {
				itemsPerScreen := CalculateItemsPerScreen(m.height) - 3 // Header takes 3 lines
				maxViewport := len(m.fileViewer.content.ArchiveInfo.Entries) - itemsPerScreen
				if maxViewport < 0 {
					maxViewport = 0
				}
				if m.fileViewer.contentViewport < maxViewport {
					m.fileViewer.contentViewport++
				}
			}
		}
	}
	return m, nil
}

// handleDatabaseTableListKey handles key actions in the database table list view.
func (m Model) handleDatabaseTableListKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "left":
		m.viewState = FileListView
		m.dbTables = dbTableListState{}
		m = m.updateViewport()
	case "right":
		if m.dbTables.info != nil && len(m.dbTables.info.Tables) > 0 && m.dbTables.cursor < len(m.dbTables.info.Tables) {
			table := m.dbTables.info.Tables[m.dbTables.cursor]
			m.dbContent.table = &table
			m.viewState = DatabaseTableContentView
			m.dbContent.loading = true
			m.dbContent.offset = 0
			m.dbContent.viewport = 0
			// Load first page of table data (50 rows)
			return m, m.fetchTableDataCmd(m.dbTables.file.Path, table.Name, 0, 50)
		}
	case "up":
		if m.dbTables.cursor > 0 {
			m.dbTables.cursor--
			m = m.updateViewport()
		}
	case "down":
		if m.dbTables.info != nil && m.dbTables.cursor < len(m.dbTables.info.Tables)-1 {
			m.dbTables.cursor++
			m = m.updateViewport()
		}
	}
	return m, nil
}

// handleDatabaseTableContentKey handles key actions in the table content view.
func (m Model) handleDatabaseTableContentKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "left":
		m.viewState = DatabaseTableListView
		m.dbContent = dbTableContentState{}
		m = m.updateViewport()
	case "up":
		if m.dbContent.viewport > 0 {
			m.dbContent.viewport--
		}
	case "down":
		// Allow scrolling through table data with lazy loading
		itemsPerScreen := CalculateItemsPerScreen(m.height) - 8 // Account for header and table headers
		maxViewport := len(m.dbContent.data) - itemsPerScreen
		if maxViewport < 0 {
			maxViewport = 0
		}

		if m.dbContent.viewport < maxViewport {
			m.dbContent.viewport++
		} else if m.dbContent.table != nil && m.dbContent.offset+len(m.dbContent.data) < int(m.dbContent.table.RowCount) {
			// Need to load more data
			newOffset := m.dbContent.offset + len(m.dbContent.data)
			m.dbContent.offset = newOffset
			m.dbContent.viewport = 0 // Reset viewport for new chunk
			m.dbContent.loading = true
			return m, m.fetchTableDataCmd(m.dbTables.file.Path, m.dbContent.table.Name, newOffset, 50)
		}
	}
	return m, nil
}

// getFilteredSimulators returns simulators based on the current filter state
func (m Model) getFilteredSimulators() []simulator.Item {
	if !m.simList.filterActive {
		return m.simList.simulators
	}

	// Filter to show only simulators with apps
	var filtered []simulator.Item
	for _, sim := range m.simList.simulators {
		if sim.AppCount > 0 {
			filtered = append(filtered, sim)
		}
	}
	return filtered
}

// handleSimulatorSearchInput handles keyboard input when in simulator search mode
func (m Model) handleSimulatorSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	action := m.keyMap.GetAction(key)

	switch action {
	case "escape":
		// Exit search mode
		m.simList.searchMode = false
		m.simList.searchQuery = ""
		m.simList.cursor = 0
		m.simList.viewport = 0
		m.statusMessage = ""
		m = m.updateViewport()
		return m, nil

	case "backspace":
		// Remove last character from search query
		if len(m.simList.searchQuery) > 0 {
			m.simList.searchQuery = m.simList.searchQuery[:len(m.simList.searchQuery)-1]
			m.simList.cursor = 0
			m.simList.viewport = 0
			m = m.updateViewport()
		}
		return m, nil

	case "up":
		// Navigate in search results
		if m.simList.cursor > 0 {
			m.simList.cursor--
			m = m.updateViewport()
		}
		return m, nil

	case "down":
		// Navigate in search results
		filteredSims := m.getFilteredAndSearchedSimulators()
		if m.simList.cursor < len(filteredSims)-1 {
			m.simList.cursor++
			m = m.updateViewport()
		}
		return m, nil

	case "enter", "right":
		// Select simulator while in search
		filteredSims := m.getFilteredAndSearchedSimulators()
		if len(filteredSims) > 0 && m.simList.cursor < len(filteredSims) {
			sim := filteredSims[m.simList.cursor]
			m.appList.selectedSim = &sim
			m.viewState = AppListView
			m.appList.loading = true
			// Exit search mode
			m.simList.searchMode = false
			m.simList.searchQuery = ""
			m.statusMessage = ""
			return m, m.fetchAppsCmd(sim)
		}
		return m, nil

	case "boot", "open":
		// Space is allowed in search
		m.simList.searchQuery += " "
		m.simList.cursor = 0
		m.simList.viewport = 0
		m = m.updateViewport()
		return m, nil

	default:
		// Add any single character to search query (including h, j, k, l, q, etc.)
		if len(msg.String()) == 1 {
			m.simList.searchQuery += msg.String()
			m.simList.cursor = 0
			m.simList.viewport = 0
			m = m.updateViewport()
		}
		return m, nil
	}
}

// handleAppSearchInput handles keyboard input when in app search mode
func (m Model) handleAppSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	action := m.keyMap.GetAction(key)

	switch action {
	case "escape":
		// Exit search mode
		m.appList.searchMode = false
		m.appList.searchQuery = ""
		m.appList.cursor = 0
		m.appList.viewport = 0
		m.statusMessage = ""
		m = m.updateViewport()
		return m, nil

	case "backspace":
		// Remove last character from search query
		if len(m.appList.searchQuery) > 0 {
			m.appList.searchQuery = m.appList.searchQuery[:len(m.appList.searchQuery)-1]
			m.appList.cursor = 0
			m.appList.viewport = 0
			m = m.updateViewport()
		}
		return m, nil

	case "up":
		// Navigate in search results
		if m.appList.cursor > 0 {
			m.appList.cursor--
			m = m.updateViewport()
		}
		return m, nil

	case "down":
		// Navigate in search results
		filteredApps := m.getFilteredAndSearchedApps()
		if m.appList.cursor < len(filteredApps)-1 {
			m.appList.cursor++
			m = m.updateViewport()
		}
		return m, nil

	case "enter", "right":
		// Select app while in search
		filteredApps := m.getFilteredAndSearchedApps()
		if len(filteredApps) > 0 && m.appList.cursor < len(filteredApps) {
			app := filteredApps[m.appList.cursor]
			m.fileList.selectedApp = &app
			m.viewState = FileListView
			m.fileList.loading = true
			m.fileList.currentPath = app.Container
			m.fileList.basePath = app.Container
			m.fileList.breadcrumbs = []string{}
			m.fileList.cursorMemory = make(map[string]int)
			m.fileList.viewportMemory = make(map[string]int)
			// Exit search mode
			m.appList.searchMode = false
			m.appList.searchQuery = ""
			m.statusMessage = ""
			return m, m.fetchFilesCmd(app.Container)
		}
		return m, nil

	case "boot", "open":
		// Space is allowed in search
		m.appList.searchQuery += " "
		m.appList.cursor = 0
		m.appList.viewport = 0
		m = m.updateViewport()
		return m, nil

	default:
		// Add any single character to search query (including h, j, k, l, q, etc.)
		if len(msg.String()) == 1 {
			m.appList.searchQuery += msg.String()
			m.appList.cursor = 0
			m.appList.viewport = 0
			m = m.updateViewport()
		}
		return m, nil
	}
}

// getFilteredAndSearchedSimulators returns simulators based on both filter and search
func (m Model) getFilteredAndSearchedSimulators() []simulator.Item {
	// First apply the app filter
	filtered := m.getFilteredSimulators()

	// If no search query, return filtered results
	if m.simList.searchQuery == "" {
		return filtered
	}

	// Apply search filter
	var searched []simulator.Item
	query := strings.ToLower(m.simList.searchQuery)

	for _, sim := range filtered {
		// Search in name, runtime, and state
		if strings.Contains(strings.ToLower(sim.Name), query) ||
			strings.Contains(strings.ToLower(sim.Runtime), query) ||
			strings.Contains(strings.ToLower(sim.State), query) {
			searched = append(searched, sim)
		}
	}

	return searched
}

// getFilteredAndSearchedApps returns apps based on search query
func (m Model) getFilteredAndSearchedApps() []simulator.App {
	// If no search query, return all apps
	if m.appList.searchQuery == "" {
		return m.appList.apps
	}

	// Apply search filter
	var searched []simulator.App
	query := strings.ToLower(m.appList.searchQuery)

	for _, app := range m.appList.apps {
		// Search in name, bundle ID, and version
		if strings.Contains(strings.ToLower(app.Name), query) ||
			strings.Contains(strings.ToLower(app.BundleID), query) ||
			strings.Contains(strings.ToLower(app.Version), query) {
			searched = append(searched, app)
		}
	}

	return searched
}

// handleAllAppsSearchInput handles keyboard input when in all apps search mode
func (m Model) handleAllAppsSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	action := m.keyMap.GetAction(key)

	switch action {
	case "escape":
		// Exit search mode
		m.allApps.searchMode = false
		m.allApps.searchQuery = ""
		m.allApps.cursor = 0
		m.allApps.viewport = 0
		m.statusMessage = ""
		m = m.updateViewport()
		return m, nil

	case "backspace":
		// Remove last character from search query
		if len(m.allApps.searchQuery) > 0 {
			m.allApps.searchQuery = m.allApps.searchQuery[:len(m.allApps.searchQuery)-1]
			m.allApps.cursor = 0
			m.allApps.viewport = 0
			m = m.updateViewport()
		}
		return m, nil

	case "up":
		// Navigate in search results
		if m.allApps.cursor > 0 {
			m.allApps.cursor--
			m = m.updateViewport()
		}
		return m, nil

	case "down":
		// Navigate in search results
		filteredApps := m.getFilteredAndSearchedAllApps()
		if m.allApps.cursor < len(filteredApps)-1 {
			m.allApps.cursor++
			m = m.updateViewport()
		}
		return m, nil

	case "enter", "right":
		// Select app while in search
		filteredApps := m.getFilteredAndSearchedAllApps()
		if len(filteredApps) > 0 && m.allApps.cursor < len(filteredApps) {
			app := filteredApps[m.allApps.cursor]
			m.fileList.selectedApp = &app
			m.viewState = FileListView
			m.fileList.loading = true
			m.fileList.currentPath = app.Container
			m.fileList.basePath = app.Container
			m.fileList.breadcrumbs = []string{}
			m.fileList.cursorMemory = make(map[string]int)
			m.fileList.viewportMemory = make(map[string]int)
			// Exit search mode
			m.allApps.searchMode = false
			m.allApps.searchQuery = ""
			m.statusMessage = ""
			return m, m.fetchFilesCmd(app.Container)
		}
		return m, nil

	default:
		// Add any single character to search query (including h, j, k, l, q, etc.)
		if len(msg.String()) == 1 {
			m.allApps.searchQuery += msg.String()
			m.allApps.cursor = 0
			m.allApps.viewport = 0
			m = m.updateViewport()
		}
		return m, nil
	}
}

// getFilteredAndSearchedAllApps returns all apps based on search query
func (m Model) getFilteredAndSearchedAllApps() []simulator.App {
	// If no search query, return all apps
	if m.allApps.searchQuery == "" {
		return m.allApps.apps
	}

	// Apply search filter
	var searched []simulator.App
	query := strings.ToLower(m.allApps.searchQuery)

	for _, app := range m.allApps.apps {
		// Search in name, bundle ID, version, and simulator name
		if strings.Contains(strings.ToLower(app.Name), query) ||
			strings.Contains(strings.ToLower(app.BundleID), query) ||
			strings.Contains(strings.ToLower(app.Version), query) ||
			strings.Contains(strings.ToLower(app.SimulatorName), query) {
			searched = append(searched, app)
		}
	}

	return searched
}
