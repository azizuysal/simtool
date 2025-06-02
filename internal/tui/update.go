package tui

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/simulator"
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
		// Periodically refresh simulator status
		return m, tea.Batch(
			fetchSimulatorsCmd(m.fetcher),
			tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}),
		)
		
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
		m.updateViewport()
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyCtrlC, KeyQuit:
		return m, tea.Quit

	case KeyLeft, KeyH:
		switch m.viewState {
		case AppListView:
			m.viewState = SimulatorListView
			m.apps = nil
			m.selectedSim = nil
			m.updateViewport()
		case FileViewerView:
			m.viewState = FileListView
			m.viewingFile = nil
			m.fileContent = nil
			m.contentOffset = 0
			m.contentViewport = 0
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
		
	case KeyEnter:
		// Enter key no longer used for viewing files

	case KeyRight, KeyL:
		switch m.viewState {
		case SimulatorListView:
			if len(m.simulators) > 0 {
				sim := m.simulators[m.simCursor]
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

	case KeyUp, KeyK:
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
					}
				case simulator.FileTypeBinary:
					if m.contentViewport > 0 {
						m.contentViewport--
					}
				}
			}
		}

	case KeyDown, KeyJ:
		// Clear status message on navigation
		m.statusMessage = ""
		switch m.viewState {
		case SimulatorListView:
			if m.simCursor < len(m.simulators)-1 {
				m.simCursor++
				m.updateViewport()
			}
		case AppListView:
			if m.appCursor < len(m.apps)-1 {
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
					}
				case simulator.FileTypeBinary:
					// Allow scrolling through binary files
					hexLines := simulator.FormatHexDump(m.fileContent.BinaryData, int64(m.contentOffset))
					itemsPerScreen := CalculateItemsPerScreen(m.height) - 5 // Account for header
					maxViewport := len(hexLines) - itemsPerScreen
					if maxViewport < 0 {
						maxViewport = 0
					}
					if m.contentViewport < maxViewport {
						m.contentViewport++
					}
				}
			}
		}

	case KeyHome:
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

	case KeyEnd:
		// Clear status message on navigation
		m.statusMessage = ""
		switch m.viewState {
		case SimulatorListView:
			m.simCursor = len(m.simulators) - 1
		case AppListView:
			m.appCursor = len(m.apps) - 1
		case FileListView:
			m.fileCursor = len(m.files) - 1
		}
		m.updateViewport()

	case KeySpace:
		switch m.viewState {
		case SimulatorListView:
			if len(m.simulators) > 0 {
				sim := m.simulators[m.simCursor]
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
	}

	return m, nil
}