package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/simulator"
	"simtool/internal/tui"
)

func main() {
	// Set up debug logging
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("failed to create debug log: %s", err)
	}
	defer f.Close()

	// Create simulator fetcher
	fetcher := simulator.NewFetcher()

	// Create and run the TUI application
	model := tui.New(fetcher)
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %s", err)
		os.Exit(1)
	}
}