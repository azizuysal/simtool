package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Simulator struct {
	UDID      string `json:"udid"`
	Name      string `json:"name"`
	State     string `json:"state"`
	IsAvailable bool `json:"isAvailable"`
	DeviceTypeIdentifier string `json:"deviceTypeIdentifier"`
}

type DevicesByRuntime map[string][]Simulator

type SimctlOutput struct {
	Devices DevicesByRuntime `json:"devices"`
}

type model struct {
	simulators []SimulatorItem
	cursor     int
	err        error
	height     int
	width      int
	viewport   int // The index of the first visible item
}

type SimulatorItem struct {
	Simulator
	Runtime string
}

var (
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("86")).
			Foreground(lipgloss.Color("230")).
			PaddingLeft(1)
	
	normalStyle = lipgloss.NewStyle().
			PaddingLeft(3)
	
	bootedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))
	
	shutdownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginBottom(1)
	
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
	
	nameStyle = lipgloss.NewStyle().
			Bold(true)
	
	detailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

func fetchSimulators() ([]SimulatorItem, error) {
	cmd := exec.Command("xcrun", "simctl", "list", "devices", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run simctl: %w", err)
	}

	var simctlOutput SimctlOutput
	if err := json.Unmarshal(output, &simctlOutput); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var items []SimulatorItem
	for runtime, sims := range simctlOutput.Devices {
		runtimeName := strings.Replace(runtime, "com.apple.CoreSimulator.SimRuntime.", "", 1)
		for _, sim := range sims {
			if sim.IsAvailable {
				items = append(items, SimulatorItem{
					Simulator: sim,
					Runtime:   runtimeName,
				})
			}
		}
	}
	
	return items, nil
}

type fetchSimulatorsMsg struct {
	simulators []SimulatorItem
	err        error
}

func fetchSimulatorsCmd() tea.Msg {
	sims, err := fetchSimulators()
	return fetchSimulatorsMsg{simulators: sims, err: err}
}

func (m model) Init() tea.Cmd {
	return fetchSimulatorsCmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateViewport()
			}
		case "down", "j":
			if m.cursor < len(m.simulators)-1 {
				m.cursor++
				m.updateViewport()
			}
		case "home":
			m.cursor = 0
			m.viewport = 0
		case "end":
			m.cursor = len(m.simulators) - 1
			m.updateViewport()
		}
	
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

func (m *model) updateViewport() {
	// Calculate how many items can fit on screen
	// Each item takes 2 lines + 1 line spacing = 3 lines
	// Reserve 5 lines for header and footer
	itemsPerScreen := (m.height - 5) / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	
	// Adjust viewport to keep cursor visible
	if m.cursor < m.viewport {
		m.viewport = m.cursor
	} else if m.cursor >= m.viewport+itemsPerScreen {
		m.viewport = m.cursor - itemsPerScreen + 1
	}
	
	// Ensure viewport doesn't go beyond bounds
	maxViewport := len(m.simulators) - itemsPerScreen
	if maxViewport < 0 {
		maxViewport = 0
	}
	if m.viewport > maxViewport {
		m.viewport = maxViewport
	}
	if m.viewport < 0 {
		m.viewport = 0
	}
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}
	
	if len(m.simulators) == 0 {
		return "Loading simulators..."
	}
	
	var s strings.Builder
	s.WriteString(headerStyle.Render(fmt.Sprintf("iOS Simulators (%d)", len(m.simulators))))
	s.WriteString("\n\n")
	
	// Calculate visible range
	itemsPerScreen := (m.height - 5) / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	
	startIdx := m.viewport
	endIdx := m.viewport + itemsPerScreen
	if endIdx > len(m.simulators) {
		endIdx = len(m.simulators)
	}
	
	// Only render visible simulators
	for i := startIdx; i < endIdx; i++ {
		sim := m.simulators[i]
		
		// Format runtime name to be more readable
		runtimeDisplay := strings.Replace(sim.Runtime, "iOS-", "iOS ", 1)
		runtimeDisplay = strings.Replace(runtimeDisplay, "-", ".", -1)
		
		// Format state
		stateDisplay := sim.State
		if sim.State == "Shutdown" {
			stateDisplay = "Not Running"
		} else if sim.State == "Booted" {
			stateDisplay = "Running"
		}
		
		// Build the two-line display
		var nameLineStyle, detailLineStyle lipgloss.Style
		
		if i == m.cursor {
			// Selected item styling
			nameLineStyle = selectedStyle.Copy().Inherit(nameStyle)
			detailLineStyle = selectedStyle.Copy()
			
			// Add cursor indicator
			s.WriteString(selectedStyle.Render("▶ " + sim.Name))
			s.WriteString("\n")
			s.WriteString(detailLineStyle.Render("  " + runtimeDisplay + " • " + stateDisplay))
		} else {
			// Non-selected item styling
			if sim.State == "Booted" {
				nameLineStyle = normalStyle.Copy().Inherit(nameStyle).Inherit(bootedStyle)
				detailLineStyle = normalStyle.Copy().Inherit(bootedStyle)
			} else {
				nameLineStyle = normalStyle.Copy().Inherit(nameStyle)
				detailLineStyle = normalStyle.Copy().Inherit(detailStyle)
			}
			
			s.WriteString(nameLineStyle.Render(sim.Name))
			s.WriteString("\n")
			s.WriteString(detailLineStyle.Render(runtimeDisplay + " • " + stateDisplay))
		}
		
		if i < endIdx-1 {
			s.WriteString("\n\n")
		}
	}
	
	// Add scroll indicators
	scrollInfo := ""
	if m.viewport > 0 && m.viewport+itemsPerScreen < len(m.simulators) {
		scrollInfo = fmt.Sprintf(" (%d-%d of %d) ↑↓", m.viewport+1, endIdx, len(m.simulators))
	} else if m.viewport > 0 {
		scrollInfo = fmt.Sprintf(" (%d-%d of %d) ↑", m.viewport+1, endIdx, len(m.simulators))
	} else if m.viewport+itemsPerScreen < len(m.simulators) {
		scrollInfo = fmt.Sprintf(" (%d-%d of %d) ↓", m.viewport+1, endIdx, len(m.simulators))
	}
	
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("↑/k: up • ↓/j: down • q: quit" + scrollInfo))
	
	return s.String()
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("err: %s", err)
	}
	defer f.Close()
	
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
