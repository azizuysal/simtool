package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		} else {
			m.fileCursor = 0
			m.fileViewport = 0
			m.updateViewport()
		}
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
		case FileListView:
			m.viewState = AppListView
			m.files = nil
			m.selectedApp = nil
			m.currentPath = ""
			m.updateViewport()
		}

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
				return m, m.fetchFilesCmd(app.Container)
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
		if m.viewState == SimulatorListView && len(m.simulators) > 0 {
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
	}

	return m, nil
}