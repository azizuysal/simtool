package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// clearStatusMsg is sent to clear the status message
type clearStatusMsg struct{}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.updateViewport()

	case fetchSimulatorsMsg:
		m.simulators = msg.simulators
		m.err = msg.err
		m.loadingSimulators = false
		if m.simCursor >= len(m.simulators) {
			m.simCursor = len(m.simulators) - 1
		}
		if m.simCursor < 0 && len(m.simulators) > 0 {
			m.simCursor = 0
		}
		m.updateViewport()

	case fetchAppsMsg:
		m.apps = msg.apps
		m.loadingApps = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading apps: %v", msg.err)
			m.viewState = SimulatorListView
			m.selectedSim = nil
			return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		} else if len(msg.apps) == 0 {
			m.statusMessage = "No apps installed on this simulator"
			m.viewState = SimulatorListView
			m.selectedSim = nil
			return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		} else {
			m.appCursor = 0
			m.appViewport = 0
			m.updateViewport()
		}

	case bootSimulatorMsg:
		m.booting = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		} else {
			m.statusMessage = "Simulator booted successfully!"
			// Refresh simulators to update status AND set up clear timer
			return m, tea.Batch(
				fetchSimulatorsCmd(m.fetcher),
				tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
					return clearStatusMsg{}
				}),
			)
		}

	case clearStatusMsg:
		m.statusMessage = ""

	case openInFinderMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error opening in Finder: %v", msg.err)
			return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}

	case tickMsg:
		// Periodically refresh simulator status and check theme
		cmds := []tea.Cmd{
			fetchSimulatorsCmd(m.fetcher),
			tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}),
		}
		
		// Check if theme mode has changed
		cmd := m.checkThemeChange()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		
		return m, tea.Batch(cmds...)
		
	case themeChangedMsg:
		// Theme has changed, update model and reload styles
		m.currentThemeMode = msg.newMode
		
		// Reload styles from config with new theme
		if err := ui.ReloadStyles(); err != nil {
			m.statusMessage = fmt.Sprintf("Failed to reload theme: %v", err)
			// Clear error message after a delay
			return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		
		// Simply return the model - styles will be picked up on next render
		return m, nil
		
	case fetchFilesMsg:
		m.files = msg.files
		m.loadingFiles = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading files: %v", msg.err)
			m.viewState = AppListView
			m.selectedApp = nil
			m.currentPath = ""
			return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		// Restore cursor position if we've been here before
		if m.cursorMemory != nil {
			if cursor, ok := m.cursorMemory[m.currentPath]; ok {
				m.fileCursor = cursor
				// Ensure cursor is within bounds
				if m.fileCursor >= len(m.files) {
					m.fileCursor = len(m.files) - 1
				}
				if m.fileCursor < 0 {
					m.fileCursor = 0
				}
			} else {
				m.fileCursor = 0
			}
			
			if viewport, ok := m.viewportMemory[m.currentPath]; ok {
				m.fileViewport = viewport
			} else {
				m.fileViewport = 0
			}
		} else {
			m.fileCursor = 0
			m.fileViewport = 0
		}
		m.updateViewport()
		
	case fetchDatabaseInfoMsg:
		m.loadingDatabase = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading database: %v", msg.err)
			m.viewState = FileListView
			m.viewingDatabase = nil
			return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		m.databaseInfo = msg.dbInfo
		m.updateViewport()
		
	case fetchTableDataMsg:
		m.loadingTableData = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading table data: %v", msg.err)
			return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		m.tableData = msg.data
		m.tableDataOffset = msg.offset
		m.updateViewport()
		
	case fetchFileContentMsg:
		m.loadingContent = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading file: %v", msg.err)
			m.viewState = FileListView
			m.viewingFile = nil
			return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		m.fileContent = msg.content
		// For binary files, update the content offset to match the loaded chunk
		if m.fileContent.Type == simulator.FileTypeBinary {
			m.contentOffset = int(m.fileContent.BinaryOffset / 16)
		}
		
		// Check for SVG with unsupported features
		m.svgWarning = ""
		if m.viewingFile != nil && strings.ToLower(filepath.Ext(m.viewingFile.Path)) == ".svg" {
			// Read file to check for unsupported features
			if data, err := os.ReadFile(m.viewingFile.Path); err == nil {
				svgStr := string(data)
				var unsupportedFeatures []string
				
				if strings.Contains(svgStr, "data:image/") || 
				   strings.Contains(svgStr, "xlink:href=\"data:") ||
				   strings.Contains(svgStr, "xlink:href=\"http") {
					unsupportedFeatures = append(unsupportedFeatures, "embedded images")
				}
				if strings.Contains(svgStr, "<filter") || strings.Contains(svgStr, "filter=") {
					unsupportedFeatures = append(unsupportedFeatures, "filters")
				}
				if strings.Contains(svgStr, "<foreignObject") {
					unsupportedFeatures = append(unsupportedFeatures, "foreign objects")
				}
				
				if len(unsupportedFeatures) > 0 {
					m.svgWarning = fmt.Sprintf("Warning: SVG contains unsupported features (%s). Preview may be incomplete.", 
						strings.Join(unsupportedFeatures, ", "))
				}
			}
		}
		
		m.updateViewport()
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode input first
	if m.simSearchMode && m.viewState == SimulatorListView {
		return m.handleSimulatorSearchInput(msg)
	}
	if m.appSearchMode && m.viewState == AppListView {
		return m.handleAppSearchInput(msg)
	}
	
	key := msg.String()
	action := m.keyMap.GetAction(key)
	
	switch action {
	case "quit":
		// Don't quit if in search mode
		if m.simSearchMode || m.appSearchMode {
			return m, nil
		}
		return m, tea.Quit

	case "left":
		switch m.viewState {
		case AppListView:
			m.viewState = SimulatorListView
			m.apps = nil
			m.selectedSim = nil
			// Clear app search mode
			m.appSearchMode = false
			m.appSearchQuery = ""
			m.updateViewport()
		case FileViewerView:
			m.viewState = FileListView
			m.viewingFile = nil
			m.fileContent = nil
			m.contentOffset = 0
			m.contentViewport = 0
			m.svgWarning = ""
			m.updateViewport()
		case DatabaseTableListView:
			m.viewState = FileListView
			m.viewingDatabase = nil
			m.databaseInfo = nil
			m.tableCursor = 0
			m.tableViewport = 0
			m.updateViewport()
		case DatabaseTableContentView:
			m.viewState = DatabaseTableListView
			m.selectedTable = nil
			m.tableData = nil
			m.tableDataOffset = 0
			m.tableDataViewport = 0
			m.updateViewport()
		case FileListView:
			if len(m.breadcrumbs) > 0 {
				// Go up one directory level
				m.breadcrumbs = m.breadcrumbs[:len(m.breadcrumbs)-1]
				newPath := m.basePath
				if len(m.breadcrumbs) > 0 {
					newPath = filepath.Join(append([]string{m.basePath}, m.breadcrumbs...)...)
				}
				m.currentPath = newPath
				m.loadingFiles = true
				return m, m.fetchFilesCmd(newPath)
			} else {
				// At root level, go back to app list
				m.viewState = AppListView
				m.files = nil
				m.selectedApp = nil
				m.currentPath = ""
				m.basePath = ""
				m.breadcrumbs = nil
				m.cursorMemory = nil
				m.viewportMemory = nil
				m.updateViewport()
			}
		}
		
	case "enter":
		// Enter key no longer used for viewing files

	case "right":
		switch m.viewState {
		case SimulatorListView:
			filteredSims := m.getFilteredSimulators()
			if len(filteredSims) > 0 && m.simCursor < len(filteredSims) {
				sim := filteredSims[m.simCursor]
				m.selectedSim = &sim
				m.viewState = AppListView
				m.loadingApps = true
				return m, m.fetchAppsCmd(sim)
			}
		case AppListView:
			if len(m.apps) > 0 {
				app := m.apps[m.appCursor]
				m.selectedApp = &app
				m.viewState = FileListView
				m.loadingFiles = true
				m.currentPath = app.Container
				m.basePath = app.Container
				m.breadcrumbs = []string{}
				m.cursorMemory = make(map[string]int)
				m.viewportMemory = make(map[string]int)
				return m, m.fetchFilesCmd(app.Container)
			}
		case FileListView:
			if len(m.files) > 0 {
				file := m.files[m.fileCursor]
				if file.IsDirectory {
					// Save current cursor position before drilling in
					if m.cursorMemory == nil {
						m.cursorMemory = make(map[string]int)
						m.viewportMemory = make(map[string]int)
					}
					m.cursorMemory[m.currentPath] = m.fileCursor
					m.viewportMemory[m.currentPath] = m.fileViewport
					
					// Drill into the directory
					m.breadcrumbs = append(m.breadcrumbs, file.Name)
					m.currentPath = file.Path
					m.loadingFiles = true
					return m, m.fetchFilesCmd(file.Path)
				} else {
					// Check if it's a database file
					fileType := simulator.DetectFileType(file.Path)
					if fileType == simulator.FileTypeDatabase {
						// View database tables
						m.viewingDatabase = &file
						m.viewState = DatabaseTableListView
						m.loadingDatabase = true
						m.tableCursor = 0
						m.tableViewport = 0
						return m, m.fetchDatabaseInfoCmd(file.Path)
					} else {
						// View the file
						m.viewingFile = &file
						m.viewState = FileViewerView
						m.loadingContent = true
						m.contentOffset = 0
						m.contentViewport = 0
						return m, m.fetchFileContentCmd(file.Path, 0)
					}
				}
			}
		case DatabaseTableListView:
			if m.databaseInfo != nil && len(m.databaseInfo.Tables) > 0 && m.tableCursor < len(m.databaseInfo.Tables) {
				table := m.databaseInfo.Tables[m.tableCursor]
				m.selectedTable = &table
				m.viewState = DatabaseTableContentView
				m.loadingTableData = true
				m.tableDataOffset = 0
				m.tableDataViewport = 0
				// Load first page of table data (50 rows)
				return m, m.fetchTableDataCmd(m.viewingDatabase.Path, table.Name, 0, 50)
			}
		}

	case "up":
		// Clear status message on navigation
		m.statusMessage = ""
		switch m.viewState {
		case SimulatorListView:
			if m.simCursor > 0 {
				m.simCursor--
				m.updateViewport()
			}
		case AppListView:
			if m.appCursor > 0 {
				m.appCursor--
				m.updateViewport()
			}
		case FileListView:
			if m.fileCursor > 0 {
				m.fileCursor--
				m.updateViewport()
			}
		case FileViewerView:
			if m.fileContent != nil {
				switch m.fileContent.Type {
				case simulator.FileTypeText:
					if m.contentViewport > 0 {
						m.contentViewport--
					} else if m.contentOffset > 0 {
						// Need to load previous chunk
						newOffset := m.contentOffset - 200 // Go back 200 lines
						if newOffset < 0 {
							newOffset = 0
						}
						m.contentOffset = newOffset
						m.loadingContent = true
						return m, m.fetchFileContentCmd(m.viewingFile.Path, newOffset)
					}
				case simulator.FileTypeImage:
					if m.contentViewport > 0 {
						m.contentViewport--
					}
				case simulator.FileTypeBinary:
					if m.contentViewport > 0 {
						m.contentViewport--
					} else if m.contentOffset > 0 {
						// Need to load previous chunk
						newOffset := m.contentOffset - 256 // Go back 256 lines (4KB)
						if newOffset < 0 {
							newOffset = 0
						}
						m.contentOffset = newOffset
						m.loadingContent = true
						// Load with line offset (offset / 16)
						return m, m.fetchFileContentCmd(m.viewingFile.Path, int(newOffset/16))
					}
				case simulator.FileTypeArchive:
					// Allow scrolling through archive entries
					if m.fileContent.ArchiveInfo != nil && m.contentViewport > 0 {
						m.contentViewport--
					}
				}
			}
		case DatabaseTableListView:
			if m.tableCursor > 0 {
				m.tableCursor--
				m.updateViewport()
			}
		case DatabaseTableContentView:
			if m.tableDataViewport > 0 {
				m.tableDataViewport--
			}
		}

	case "down":
		// Clear status message on navigation
		m.statusMessage = ""
		switch m.viewState {
		case SimulatorListView:
			filteredSims := m.getFilteredAndSearchedSimulators()
			if m.simCursor < len(filteredSims)-1 {
				m.simCursor++
				m.updateViewport()
			}
		case AppListView:
			filteredApps := m.getFilteredAndSearchedApps()
			if m.appCursor < len(filteredApps)-1 {
				m.appCursor++
				m.updateViewport()
			}
		case FileListView:
			if m.fileCursor < len(m.files)-1 {
				m.fileCursor++
				m.updateViewport()
			}
		case FileViewerView:
			if m.fileContent != nil {
				switch m.fileContent.Type {
				case simulator.FileTypeText:
					itemsPerScreen := CalculateItemsPerScreen(m.height) - 5 // Account for header
					maxViewport := len(m.fileContent.Lines) - itemsPerScreen
					if maxViewport < 0 {
						maxViewport = 0
					}
					
					if m.contentViewport < maxViewport {
						m.contentViewport++
					} else if m.contentOffset + len(m.fileContent.Lines) < m.fileContent.TotalLines {
						// Need to load more content
						newOffset := m.contentOffset + len(m.fileContent.Lines)
						m.contentOffset = newOffset
						m.contentViewport = 0 // Reset viewport for new chunk
						m.loadingContent = true
						return m, m.fetchFileContentCmd(m.viewingFile.Path, newOffset)
					}
				case simulator.FileTypeImage:
					// For images, calculate based on total content lines
					if m.fileContent.ImageInfo != nil && m.fileContent.ImageInfo.Preview != nil {
						// Calculate total lines (metadata + preview)
						totalLines := 8 + len(m.fileContent.ImageInfo.Preview.Rows) // ~8 lines for metadata
						itemsPerScreen := CalculateItemsPerScreen(m.height) - 5
						maxViewport := totalLines - itemsPerScreen
						if maxViewport < 0 {
							maxViewport = 0
						}
						if m.contentViewport < maxViewport {
							m.contentViewport++
						}
					}
				case simulator.FileTypeBinary:
					// Allow scrolling through binary files with lazy loading
					hexLines := simulator.FormatHexDump(m.fileContent.BinaryData, m.fileContent.BinaryOffset)
					itemsPerScreen := CalculateItemsPerScreen(m.height) - 5 // Account for header
					maxViewport := len(hexLines) - itemsPerScreen
					if maxViewport < 0 {
						maxViewport = 0
					}
					
					if m.contentViewport < maxViewport {
						m.contentViewport++
					} else {
						// Check if we need to load more data
						currentEndByte := m.fileContent.BinaryOffset + int64(len(m.fileContent.BinaryData))
						if currentEndByte < m.fileContent.TotalSize {
							// Load next chunk
							newOffset := m.contentOffset + len(hexLines)
							m.contentOffset = newOffset
							m.contentViewport = 0 // Reset viewport for new chunk
							m.loadingContent = true
							// Load with line offset (total lines from start)
							return m, m.fetchFileContentCmd(m.viewingFile.Path, newOffset)
						}
						}
					case simulator.FileTypeArchive:
						// Allow scrolling through archive entries (now 1 line per entry)
						if m.fileContent.ArchiveInfo != nil {
							itemsPerScreen := CalculateItemsPerScreen(m.height) - 3 // Header takes 3 lines
							maxViewport := len(m.fileContent.ArchiveInfo.Entries) - itemsPerScreen
							if maxViewport < 0 {
								maxViewport = 0
							}
							if m.contentViewport < maxViewport {
								m.contentViewport++
							}
						}
				}
			}
		case DatabaseTableListView:
			if m.databaseInfo != nil && m.tableCursor < len(m.databaseInfo.Tables)-1 {
				m.tableCursor++
				m.updateViewport()
			}
		case DatabaseTableContentView:
			// Allow scrolling through table data with lazy loading
			itemsPerScreen := CalculateItemsPerScreen(m.height) - 8 // Account for header and table headers
			maxViewport := len(m.tableData) - itemsPerScreen
			if maxViewport < 0 {
				maxViewport = 0
			}
			
			if m.tableDataViewport < maxViewport {
				m.tableDataViewport++
			} else if m.selectedTable != nil && m.tableDataOffset + len(m.tableData) < int(m.selectedTable.RowCount) {
				// Need to load more data
				newOffset := m.tableDataOffset + len(m.tableData)
				m.tableDataOffset = newOffset
				m.tableDataViewport = 0 // Reset viewport for new chunk
				m.loadingTableData = true
				return m, m.fetchTableDataCmd(m.viewingDatabase.Path, m.selectedTable.Name, newOffset, 50)
			}
		}

	case "home":
		// Clear status message on navigation
		m.statusMessage = ""
		switch m.viewState {
		case SimulatorListView:
			m.simCursor = 0
			m.simViewport = 0
		case AppListView:
			m.appCursor = 0
			m.appViewport = 0
		case FileListView:
			m.fileCursor = 0
			m.fileViewport = 0
		}

	case "end":
		// Clear status message on navigation
		m.statusMessage = ""
		switch m.viewState {
		case SimulatorListView:
			filteredSims := m.getFilteredAndSearchedSimulators()
			m.simCursor = len(filteredSims) - 1
			if m.simCursor < 0 {
				m.simCursor = 0
			}
		case AppListView:
			filteredApps := m.getFilteredAndSearchedApps()
			m.appCursor = len(filteredApps) - 1
			if m.appCursor < 0 {
				m.appCursor = 0
			}
		case FileListView:
			m.fileCursor = len(m.files) - 1
		}
		m.updateViewport()

	case "filter":
		if m.viewState == SimulatorListView {
			m.filterActive = !m.filterActive
			// Reset cursor when toggling filter
			m.simCursor = 0
			m.simViewport = 0
			m.updateViewport()
		}

	case "boot", "open":
		switch m.viewState {
		case SimulatorListView:
			filteredSims := m.getFilteredSimulators()
			if len(filteredSims) > 0 && m.simCursor < len(filteredSims) {
				sim := filteredSims[m.simCursor]
				if !sim.IsRunning() && !m.booting {
					m.booting = true
					m.statusMessage = fmt.Sprintf("Booting %s...", sim.Name)
					return m, m.bootSimulatorCmd(sim.UDID)
				} else if sim.IsRunning() {
					m.statusMessage = "Simulator is already running"
					return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
						return clearStatusMsg{}
					})
				}
			}
		case AppListView:
			if len(m.apps) > 0 {
				app := m.apps[m.appCursor]
				if app.Container != "" {
					// Open the app's container in Finder
					return m, m.openInFinderCmd(app.Container)
				}
			}
		case FileListView:
			if len(m.files) > 0 {
				file := m.files[m.fileCursor]
				// Open in Finder - for files, this will reveal them in their containing folder
				return m, m.openInFinderCmd(file.Path)
			}
		}
		
	case "search":
		switch m.viewState {
		case SimulatorListView:
			m.simSearchMode = true
			m.simSearchQuery = ""
			// Reset cursor to 0 when starting search
			m.simCursor = 0
			m.simViewport = 0
			m.updateViewport()
		case AppListView:
			m.appSearchMode = true
			m.appSearchQuery = ""
			// Reset cursor to 0 when starting search
			m.appCursor = 0
			m.appViewport = 0
			m.updateViewport()
		}
	}

	return m, nil
}

// getFilteredSimulators returns simulators based on the current filter state
func (m Model) getFilteredSimulators() []simulator.Item {
	if !m.filterActive {
		return m.simulators
	}
	
	// Filter to show only simulators with apps
	var filtered []simulator.Item
	for _, sim := range m.simulators {
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
		m.simSearchMode = false
		m.simSearchQuery = ""
		m.simCursor = 0
		m.simViewport = 0
		m.statusMessage = ""
		m.updateViewport()
		return m, nil
		
	case "backspace":
		// Remove last character from search query
		if len(m.simSearchQuery) > 0 {
			m.simSearchQuery = m.simSearchQuery[:len(m.simSearchQuery)-1]
			m.simCursor = 0
			m.simViewport = 0
			m.updateViewport()
		}
		return m, nil
		
	case "up":
		// Navigate in search results
		if m.simCursor > 0 {
			m.simCursor--
			m.updateViewport()
		}
		return m, nil
		
	case "down":
		// Navigate in search results
		filteredSims := m.getFilteredAndSearchedSimulators()
		if m.simCursor < len(filteredSims)-1 {
			m.simCursor++
			m.updateViewport()
		}
		return m, nil
		
	case "enter", "right":
		// Select simulator while in search
		filteredSims := m.getFilteredAndSearchedSimulators()
		if len(filteredSims) > 0 && m.simCursor < len(filteredSims) {
			sim := filteredSims[m.simCursor]
			m.selectedSim = &sim
			m.viewState = AppListView
			m.loadingApps = true
			// Exit search mode
			m.simSearchMode = false
			m.simSearchQuery = ""
			m.statusMessage = ""
			return m, m.fetchAppsCmd(sim)
		}
		return m, nil
		
	case "boot", "open":
		// Space is allowed in search
		m.simSearchQuery += " "
		m.simCursor = 0
		m.simViewport = 0
		m.updateViewport()
		return m, nil
		
	default:
		// Add any single character to search query (including h, j, k, l, q, etc.)
		if len(msg.String()) == 1 {
			m.simSearchQuery += msg.String()
			m.simCursor = 0
			m.simViewport = 0
			m.updateViewport()
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
		m.appSearchMode = false
		m.appSearchQuery = ""
		m.appCursor = 0
		m.appViewport = 0
		m.statusMessage = ""
		m.updateViewport()
		return m, nil
		
	case "backspace":
		// Remove last character from search query
		if len(m.appSearchQuery) > 0 {
			m.appSearchQuery = m.appSearchQuery[:len(m.appSearchQuery)-1]
			m.appCursor = 0
			m.appViewport = 0
			m.updateViewport()
		}
		return m, nil
		
	case "up":
		// Navigate in search results
		if m.appCursor > 0 {
			m.appCursor--
			m.updateViewport()
		}
		return m, nil
		
	case "down":
		// Navigate in search results
		filteredApps := m.getFilteredAndSearchedApps()
		if m.appCursor < len(filteredApps)-1 {
			m.appCursor++
			m.updateViewport()
		}
		return m, nil
		
	case "enter", "right":
		// Select app while in search
		filteredApps := m.getFilteredAndSearchedApps()
		if len(filteredApps) > 0 && m.appCursor < len(filteredApps) {
			app := filteredApps[m.appCursor]
			m.selectedApp = &app
			m.viewState = FileListView
			m.loadingFiles = true
			m.currentPath = app.Container
			m.basePath = app.Container
			m.breadcrumbs = []string{}
			m.cursorMemory = make(map[string]int)
			m.viewportMemory = make(map[string]int)
			// Exit search mode
			m.appSearchMode = false
			m.appSearchQuery = ""
			m.statusMessage = ""
			return m, m.fetchFilesCmd(app.Container)
		}
		return m, nil
		
	case "boot", "open":
		// Space is allowed in search
		m.appSearchQuery += " "
		m.appCursor = 0
		m.appViewport = 0
		m.updateViewport()
		return m, nil
		
	default:
		// Add any single character to search query (including h, j, k, l, q, etc.)
		if len(msg.String()) == 1 {
			m.appSearchQuery += msg.String()
			m.appCursor = 0
			m.appViewport = 0
			m.updateViewport()
		}
		return m, nil
	}
}

// getFilteredAndSearchedSimulators returns simulators based on both filter and search
func (m Model) getFilteredAndSearchedSimulators() []simulator.Item {
	// First apply the app filter
	filtered := m.getFilteredSimulators()
	
	// If no search query, return filtered results
	if m.simSearchQuery == "" {
		return filtered
	}
	
	// Apply search filter
	var searched []simulator.Item
	query := strings.ToLower(m.simSearchQuery)
	
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
	if m.appSearchQuery == "" {
		return m.apps
	}
	
	// Apply search filter
	var searched []simulator.App
	query := strings.ToLower(m.appSearchQuery)
	
	for _, app := range m.apps {
		// Search in name, bundle ID, and version
		if strings.Contains(strings.ToLower(app.Name), query) ||
			strings.Contains(strings.ToLower(app.BundleID), query) ||
			strings.Contains(strings.ToLower(app.Version), query) {
			searched = append(searched, app)
		}
	}
	
	return searched
}