# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go project called `simtool` that uses the Bubble Tea framework to create a terminal UI application. The application displays a list of iOS simulators installed on the system and allows navigation with arrow keys or vim-style j/k keys.

## Project Structure

```
simtool/
├── cmd/simtool/        # Application entry point
├── internal/
│   ├── simulator/      # Simulator types and fetching logic
│   │   ├── simulator.go   # Core types and interfaces
│   │   ├── fetcher.go     # xcrun simctl integration
│   │   ├── app.go         # App information and listing
│   │   ├── files.go       # File browsing and operations
│   │   ├── files_darwin.go # macOS-specific file operations
│   │   ├── files_other.go  # Stub for other platforms
│   │   └── viewer.go      # File content viewing
│   ├── tui/           # Terminal UI components
│   │   ├── model.go       # Bubble Tea model
│   │   ├── update.go      # Message handling
│   │   ├── view.go        # Rendering logic
│   │   ├── view_file.go   # File viewer rendering
│   │   ├── viewport.go    # Scrolling logic
│   │   └── keys.go        # Key bindings
│   └── ui/            # UI styles and formatting
│       ├── styles.go      # Lipgloss styles
│       └── format.go      # Formatting helpers
└── Makefile           # Build automation
```

## Development Commands

### Building the Application
```bash
make build

# Or directly
go build -o simtool ./cmd/simtool
```

### Running the Application
```bash
make run

# Or directly
go run ./cmd/simtool

# Or after building
./simtool
```

Note: This is a TUI application that requires a proper terminal environment. It won't run properly in environments without TTY support.

### Testing
```bash
make test
```

### Other Commands
```bash
# Format code
make fmt

# Clean build artifacts
make clean

# Install to $GOPATH/bin
make install

# Download dependencies
make deps
```

## Architecture

The application follows clean architecture principles with clear separation of concerns:

### Packages

1. **internal/simulator**: Core business logic
   - Defines simulator types and interfaces
   - Fetches simulators via `xcrun simctl list devices --json`
   - Boots simulators and opens Simulator.app
   - Lists and manages installed apps
   - Browses app container files and directories
   - Reads file content with lazy loading for large files
   - Provides syntax highlighting for code files using chroma
   - Generates terminal-based image previews
   - Formats hex dumps for binary files
   - Displays ZIP archive contents with file listings
   - Generates SVG previews as ASCII art

2. **internal/tui**: Terminal UI logic (Bubble Tea MVU pattern)
   - Model: Application state (simulators, apps, files, cursor, viewport)
   - Update: Handles messages and state updates
   - View: Renders simulator list, app list, file browser, and file viewer
   - Viewport: Manages scrolling logic for all views
   - Lazy loading for text files and dynamic chunk loading

3. **internal/ui**: UI styling and formatting
   - Centralized Lipgloss styles
   - Formatting utilities
   - Reusable UI components

4. **cmd/simtool**: Application entry point
   - Minimal main function
   - Sets up dependencies
   - Runs the TUI application

## Key Design Decisions

- **Interface-based design**: Simulator fetcher is an interface for easy testing
- **Package separation**: Clear boundaries between UI, business logic, and presentation
- **Reusable components**: UI styles and formatting are centralized
- **Clean architecture**: Dependencies flow inward (UI → TUI → Simulator)

## Features

### Simulator Management
- Lists all iOS simulators sorted alphabetically by name
- Shows installed app count for each simulator (both running and shutdown)
- Visual indication of running simulators (green text)
- Boot simulators with 'space' key (opens Simulator.app)
- Filter simulators to show only those with installed apps (press 'f')
- Search simulators by name, runtime, or state (press '/')

### App Browsing
- Browse apps installed on each simulator
- View app details including bundle ID, version, and size
- Search apps by name, bundle ID, or version (press '/')
- Open app containers in Finder (press 'space')

### File Management
- Navigate app data container files and directories
- View file contents with appropriate rendering based on type
- Open files and folders in Finder (press 'space')
- Smart file type detection based on content and extension

### File Viewing
- Text files: Syntax highlighting for code files using chroma
- Images: Terminal-based previews for PNG, JPEG, GIF, BMP, TIFF, WebP
- SVG files: ASCII art previews with viewBox information
- Binary files: Hex dump format with offset and ASCII preview
- Archives: Tree structure view for ZIP, JAR, WAR, EAR, IPA, APK, AAR files
- Lazy loading for large files with dynamic chunk loading

### Navigation & UI
- Navigate with arrow keys (↑/↓) or vim keys (j/k)
- Move between views with arrow keys (←/→) or vim keys (h/l)
- Selected items highlighted with gray background
- Centered UI elements with rounded borders
- Smooth viewport scrolling for long lists
- Centered key legends on all views
- Status messages displayed in dedicated status area
- Blue colored search and filter status indicators
- Press 'q' or Ctrl+C to quit

## Key Dependencies

- `github.com/charmbracelet/bubbletea` - Terminal UI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/alecthomas/chroma/v2` - Syntax highlighting library