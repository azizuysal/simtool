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
		if m.cursor >= len(m.simulators) {
			m.cursor = len(m.simulators) - 1
		}
		if m.cursor < 0 && len(m.simulators) > 0 {
			m.cursor = 0
		}
		m.updateViewport()

	case bootSimulatorMsg:
		m.booting = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMessage = "Simulator booted successfully!"
			// Refresh simulators to update status
			return m, fetchSimulatorsCmd(m.fetcher)
		}
		// Clear status message after 3 seconds
		return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		m.statusMessage = ""
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyCtrlC, KeyQuit:
		return m, tea.Quit

	case KeyUp, KeyK:
		if m.cursor > 0 {
			m.cursor--
			m.updateViewport()
		}

	case KeyDown, KeyJ:
		if m.cursor < len(m.simulators)-1 {
			m.cursor++
			m.updateViewport()
		}

	case KeyHome:
		m.cursor = 0
		m.viewport = 0

	case KeyEnd:
		m.cursor = len(m.simulators) - 1
		m.updateViewport()

	case KeyRun:
		if len(m.simulators) > 0 && m.cursor < len(m.simulators) {
			sim := m.simulators[m.cursor]
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