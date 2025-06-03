# SimTool

![Coverage](https://img.shields.io/badge/coverage-37%25-orange)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/simtool)](https://goreportcard.com/report/github.com/yourusername/simtool)
[![Build Status](https://github.com/yourusername/simtool/workflows/Tests/badge.svg)](https://github.com/yourusername/simtool/actions)

A terminal UI application for managing iOS simulators on macOS.

## Features

- List all available iOS simulators sorted alphabetically
- View simulator status and installed app count
- Boot simulators directly from the UI
- Browse apps installed on simulators
- Navigate app data container files
- View file contents with syntax highlighting
- Display images with terminal-based previews
- View binary files in hex dump format
- Open files and folders in Finder
- Navigate with arrow keys or vim-style keys
- Smooth scrolling for long lists
- Clean, modern UI with rounded borders

## Requirements

- macOS with Xcode installed
- Go 1.24.3 or later

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd simtool

# Build the application
make build

# Or install to $GOPATH/bin
make install
```

## Usage

```bash
# Run directly
./simtool

# Or if installed
simtool
```

### Keyboard Shortcuts

#### Navigation
- `↑` / `k` - Move up
- `↓` / `j` - Move down
- `→` / `l` - Enter/view (apps, files, or file content)
- `←` / `h` - Go back
- `Home` - Jump to first item
- `End` - Jump to last item
- `q` / `Ctrl+C` - Quit

#### Actions
- `space` - Boot simulator (in simulator list) or open in Finder (in app/file lists)
- `→` / `l` - View apps (simulator list), view files (app list), or view file content (file list)

### Display Information

#### Simulator List
- **Line 1**: Simulator name
- **Line 2**: iOS version • Running/Not Running • Number of installed apps

#### App List
- **Line 1**: App name
- **Line 2**: Bundle ID • Version • Size

#### File Browser
- **Line 1**: File/folder name (folders end with /)
- **Line 2**: Size • Created date • Modified date

#### File Viewer
- **Text files**: Line numbers with content, lazy-loaded in chunks
- **Images**: Metadata box and terminal-based preview
- **Binary files**: Hex dump with ASCII representation

## Development

### Project Structure

```
simtool/
├── cmd/simtool/        # Application entry point
├── internal/
│   ├── simulator/      # Simulator types and fetching logic
│   ├── tui/           # Terminal UI components
│   └── ui/            # UI styles and formatting
└── Makefile           # Build automation
```

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Format code
make fmt

# Clean build artifacts
make clean
```

## License

[Add your license here]