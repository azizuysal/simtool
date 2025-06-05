# SimTool

![Coverage](https://img.shields.io/badge/coverage-37%25-orange)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/simtool)](https://goreportcard.com/report/github.com/yourusername/simtool)
[![Build Status](https://github.com/yourusername/simtool/workflows/Tests/badge.svg)](https://github.com/yourusername/simtool/actions)

A terminal UI application for managing iOS simulators on macOS.

## Features

### Simulator Management
- List all available iOS simulators sorted alphabetically
- View simulator status (running/not running) with color indicators
- Display installed app count for each simulator
- Boot simulators directly from the UI with visual feedback
- Filter simulators to show only those with apps installed
- Search simulators by name, runtime, or state

### App Browsing
- Browse all apps installed on each simulator
- View detailed app information (bundle ID, version, size)
- Search apps by name, bundle ID, or version
- Open app data containers directly in Finder

### File Management
- Navigate app data container files and directories
- Browse hierarchical file structures with breadcrumb navigation
- View detailed file information (size, created/modified dates)
- Open files and folders in Finder for external editing
- Smart file type detection based on content and extension

### File Viewing
- **Text files**: Syntax highlighting for 100+ languages using chroma
- **Images**: Terminal-based previews for PNG, JPEG, GIF, BMP, TIFF, WebP
- **SVG files**: ASCII art previews with dimension information
- **Binary files**: Hex dump format with offset and ASCII preview
- **Archives**: Tree structure view for ZIP, JAR, WAR, EAR, IPA, APK, AAR files
- Lazy loading for large files with automatic chunking

### User Interface
- Clean, modern TUI with rounded borders and centered layouts
- Navigate with arrow keys or vim-style keys (h,j,k,l)
- Smooth viewport scrolling for long lists
- Visual selection indicator with gray background
- Centered key legends showing available shortcuts
- Dedicated status area for search and filter indicators
- Consistent blue color scheme for active states

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
- `f` - Toggle filter to show only simulators with apps (simulator list only)
- `/` - Start search mode (simulator and app lists)
- `ESC` - Exit search mode

#### Search Mode
- Type to filter results in real-time
- `↑` / `↓` - Navigate filtered results
- `→` / `Enter` - Select item
- `Backspace` - Delete last character
- `ESC` - Cancel search

### Display Information

#### Status Indicators
- **Search Mode**: Blue "Search: [query]" indicator in status area
- **Filter Active**: Blue "Filter: Showing only simulators with apps" in status area
- **Running Simulators**: Green text color
- **Selected Item**: Gray background highlight

#### Simulator List
- **Header**: "iOS Simulators (X)" or "(X of Y)" when filtered
- **Line 1**: Simulator name
- **Line 2**: iOS version • Running/Not Running • Number of installed apps

#### App List
- **Header**: "[Simulator Name] Apps (X)" or "(X of Y)" when searching
- **Line 1**: App name
- **Line 2**: Bundle ID • Version • Size

#### File Browser
- **Header**: "[App Name] Files" with breadcrumb navigation below
- **Line 1**: File/folder name (folders end with /)
- **Line 2**: Size • Created date • Modified date

#### File Viewer
- **Text files**: Line numbers with syntax-highlighted content, lazy-loaded in chunks
- **Images**: Metadata box and terminal-based preview
- **Binary files**: Hex dump with ASCII representation
- **ZIP archives**: Tree view with file/folder counts and overall compression ratio
- **SVG files**: Metadata and terminal-based preview using ASCII art

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