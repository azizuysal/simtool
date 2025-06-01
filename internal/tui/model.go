package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/simulator"
)

// Model represents the application state
type Model struct {
	simulators []simulator.Item
	cursor     int
	err        error
	height     int
	width      int
	viewport   int // The index of the first visible item
	fetcher    simulator.Fetcher
}

// New creates a new Model with the given fetcher
func New(fetcher simulator.Fetcher) Model {
	return Model{
		fetcher: fetcher,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return fetchSimulatorsCmd(m.fetcher)
}

// fetchSimulatorsMsg is sent when simulators are fetched
type fetchSimulatorsMsg struct {
	simulators []simulator.Item
	err        error
}

// fetchSimulatorsCmd fetches simulators asynchronously
func fetchSimulatorsCmd(fetcher simulator.Fetcher) tea.Cmd {
	return func() tea.Msg {
		sims, err := fetcher.Fetch()
		return fetchSimulatorsMsg{simulators: sims, err: err}
	}
}