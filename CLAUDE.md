# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go project called `simtool` that uses the Bubble Tea framework to create a terminal UI application. The application displays a list of iOS simulators installed on the system and allows navigation with arrow keys or vim-style j/k keys.

## Development Commands

### Building the Application
```bash
go build
```

### Running the Application
```bash
go run main.go

# Or build and run
go build
./simtool
```

Note: This is a TUI application that requires a proper terminal environment. It won't run properly in environments without TTY support.

### Managing Dependencies
```bash
# Download dependencies
go mod download

# Add new dependencies
go get <package-name>

# Update dependencies
go mod tidy
```

## Architecture

The application follows the Bubble Tea Model-View-Update (MVU) pattern:

- **Model**: Holds the application state including:
  - List of simulators with their runtime info
  - Current cursor position
  - Error state
- **Init**: Fetches iOS simulators using `xcrun simctl` command
- **Update**: Handles keyboard input and simulator data updates
- **View**: Renders the simulator list with selection highlighting

The application uses:
- `xcrun simctl list devices --json` to fetch simulator data
- Alternate screen mode for full-screen terminal UI
- Debug logging to `debug.log` file
- Bubble Tea framework for terminal UI handling
- Lipgloss for styling and colors

## Features

- Lists all available iOS simulators grouped by runtime
- Navigate with arrow keys (↑/↓) or vim keys (j/k)
- Visual indication of booted simulators (green text)
- Selected simulator highlighted with background color
- Press 'q' or Ctrl+C to quit

## Key Dependencies

- `github.com/charmbracelet/bubbletea` - Terminal UI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling (indirect dependency)