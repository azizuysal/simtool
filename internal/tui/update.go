package tui

import tea "github.com/charmbracelet/bubbletea"

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
	}

	return m, nil
}