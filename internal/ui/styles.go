package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color constants
	SuccessColor = lipgloss.Color("42") // Green for success messages

	// SelectedStyle is used for the currently selected item
	SelectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("255"))

	// NormalStyle is the default style
	NormalStyle = lipgloss.NewStyle()

	// BootedStyle is used for running simulators
	BootedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	// ShutdownStyle is used for shutdown simulators
	ShutdownStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// HeaderStyle is used for the main header
	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("63")).
		Padding(0, 2).
		MarginBottom(1)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	// SearchStyle is used for search status messages
	SearchStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")). // Blue color
		Bold(true)

	// NameStyle is used for simulator names
	NameStyle = lipgloss.NewStyle().
		Bold(true)

	// DetailStyle is used for simulator details
	DetailStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// BorderStyle defines the rounded border style
	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
		}).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	// ListItemStyle is used for list items
	ListItemStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2)

	// FooterStyle is used for the footer
	FooterStyle = lipgloss.NewStyle().
		Faint(true)

	// FolderStyle is used for folders/directories
	FolderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("87")).
		Bold(true)

	// StatusStyle is used for status messages in the status line
	StatusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")). // Orange
		Bold(true)

	// LoadingStyle is used for loading messages
	LoadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")). // Blue
		Bold(true)
)