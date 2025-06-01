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
}

type SimulatorItem struct {
	Simulator
	Runtime string
}

var (
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("86")).
			Foreground(lipgloss.Color("230"))
	
	normalStyle = lipgloss.NewStyle()
	
	bootedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))
	
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginBottom(1)
	
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
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
			}
		case "down", "j":
			if m.cursor < len(m.simulators)-1 {
				m.cursor++
			}
		}
	
	case fetchSimulatorsMsg:
		m.simulators = msg.simulators
		m.err = msg.err
		if m.cursor >= len(m.simulators) {
			m.cursor = len(m.simulators) - 1
		}
		if m.cursor < 0 && len(m.simulators) > 0 {
			m.cursor = 0
		}
	}
	
	return m, nil
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
	
	for i, sim := range m.simulators {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		
		name := fmt.Sprintf("%-30s", sim.Name)
		runtime := fmt.Sprintf("%-20s", sim.Runtime)
		state := sim.State
		
		line := fmt.Sprintf("%s%s %s %s", cursor, name, runtime, state)
		
		if i == m.cursor {
			s.WriteString(selectedStyle.Render(line))
		} else if sim.State == "Booted" {
			s.WriteString(bootedStyle.Render(line))
		} else {
			s.WriteString(normalStyle.Render(line))
		}
		
		if i < len(m.simulators)-1 {
			s.WriteString("\n")
		}
	}
	
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("↑/k: up • ↓/j: down • q: quit"))
	
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
