package tui

import (
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/simulator"
)

// ViewState represents the current view
type ViewState int

const (
	SimulatorListView ViewState = iota
	AppListView
)

// Model represents the application state
type Model struct {
	// Common state
	viewState     ViewState
	err           error
	height        int
	width         int
	statusMessage string
	fetcher       simulator.Fetcher
	
	// Simulator list state
	simulators    []simulator.Item
	simCursor     int
	simViewport   int
	booting       bool
	
	// App list state
	selectedSim   *simulator.Item
	apps          []simulator.App
	appCursor     int
	appViewport   int
	loadingApps   bool
}

// New creates a new Model with the given fetcher
func New(fetcher simulator.Fetcher) Model {
	return Model{
		fetcher: fetcher,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchSimulatorsCmd(m.fetcher),
		tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
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

// bootSimulatorMsg is sent when a simulator boot is attempted
type bootSimulatorMsg struct {
	udid string
	err  error
}

// bootSimulatorCmd boots a simulator asynchronously
func (m Model) bootSimulatorCmd(udid string) tea.Cmd {
	return func() tea.Msg {
		err := m.fetcher.Boot(udid)
		return bootSimulatorMsg{udid: udid, err: err}
	}
}

// fetchAppsMsg is sent when apps are fetched
type fetchAppsMsg struct {
	apps []simulator.App
	err  error
}

// tickMsg is sent periodically to refresh simulator status
type tickMsg time.Time

// fetchAppsCmd fetches apps for a simulator
func (m Model) fetchAppsCmd(sim simulator.Item) tea.Cmd {
	return func() tea.Msg {
		apps, err := simulator.GetAppsForSimulator(sim.UDID, sim.IsRunning())
		return fetchAppsMsg{apps: apps, err: err}
	}
}