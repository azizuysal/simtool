# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go project called `simtool` that uses the Bubble Tea framework to create a terminal UI application. The application displays a list of iOS simulators installed on the system and allows navigation with arrow keys or vim-style j/k keys.

## Project Structure

```
simtool/
├── cmd/simtool/        # Application entry point
├── internal/
│   ├── config/         # Configuration management
│   │   ├── config.go      # Config loading and merging
│   │   ├── styles.go      # Style generation from config
│   │   ├── theme.go       # Theme color extraction
│   │   ├── detect.go      # Terminal theme detection
│   │   ├── terminal_query.go # macOS system theme detection
│   │   └── terminal_osc.go   # OSC escape sequence queries
│   ├── simulator/      # Simulator types and fetching logic
│   │   ├── simulator.go   # Core types and interfaces
│   │   ├── fetcher.go     # xcrun simctl integration
│   │   ├── app.go         # App information and listing
│   │   ├── files.go       # File browsing and operations
│   │   ├── files_darwin.go # macOS-specific file operations
│   │   ├── files_other.go  # Stub for other platforms
│   │   └── viewer.go      # File content viewing and syntax highlighting
│   ├── tui/           # Terminal UI components
│   │   ├── model.go       # Bubble Tea model
│   │   ├── update.go      # Message handling
│   │   ├── view.go        # Main view orchestrator
│   │   ├── viewport.go    # Scrolling logic
│   │   ├── keys.go        # Key bindings
│   │   └── components/    # Reusable UI components
│   │       ├── layout.go         # Base layout (title, content, footer)
│   │       ├── simulator_list.go # Simulator list view
│   │       ├── app_list.go       # App list view
│   │       ├── file_list.go      # File browser view
│   │       ├── database_table_list.go    # Database table list view
│   │       ├── database_table_content.go # Database table content view  
│   │       └── file_viewer/      # File viewer components
│   │           ├── viewer.go     # Main file viewer
│   │           ├── text.go       # Text file viewer with syntax highlighting
│   │           ├── image.go      # Image file viewer
│   │           ├── binary.go     # Binary file viewer
│   │           ├── archive.go    # Archive file viewer
│   │           └── database.go   # Database file viewer
│   └── ui/            # UI styles and formatting
│       ├── styles.go      # Lipgloss styles (theme-based)
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

1. **internal/config**: Configuration management
   - Loads user configuration from `~/.config/simtool/config.toml`
   - Merges user settings with defaults
   - Extracts colors from syntax highlighting themes to create cohesive UI
   - Detects terminal dark/light mode using OSC queries and system settings
   - Generates lipgloss styles dynamically from theme colors
   - Supports TOML format for human-friendly editing

2. **internal/simulator**: Core business logic
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
   - Provides SQLite database browsing with table navigation and data viewing

2. **internal/tui**: Terminal UI logic (Bubble Tea MVU pattern)
   - Model: Application state (simulators, apps, files, cursor, viewport)
   - Update: Handles messages and state updates
   - View: Main view orchestrator using component system
   - Components: Reusable UI components with consistent layout
     - Layout: Base responsive layout with title, content box, and footer
     - Views: Separated simulator list, app list, file list, database table list, and database table content components
     - File viewers: Type-specific renderers for text, image, binary, archive, and database files
   - Viewport: Manages scrolling logic for all views
   - Responsive design that adapts to terminal window size

3. **internal/ui**: UI styling and formatting
   - Centralized Lipgloss styles (theme-based, no hardcoded colors)
   - All colors derived from the active syntax highlighting theme
   - Formatting utilities
   - Reusable UI components

4. **cmd/simtool**: Application entry point
   - Terminal theme detection initialization
   - Command-line flag handling
   - Config file generation (`--generate-config`)
   - Config path display (`--show-config-path`)
   - Theme listing (`--list-themes`)
   - Minimal main function
   - Sets up dependencies
   - Runs the TUI application

## Key Design Decisions

- **Interface-based design**: Simulator fetcher is an interface for easy testing
- **Package separation**: Clear boundaries between UI, business logic, and presentation
- **Component-based UI**: Reusable UI components with consistent layout and behavior
- **Responsive design**: All views adapt to terminal window size
- **Clean architecture**: Dependencies flow inward (UI → TUI → Simulator)
- **Separation of concerns**: Each component handles its own rendering, state, and behavior

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
  - Support for TypeScript, TSX, JavaScript, JSX, Swift, Objective-C, Objective-C++, and 100+ more languages
  - Theme-aware colors that adapt to terminal dark/light mode
  - ANSI escape sequences for terminal rendering
- Images: Terminal-based previews for PNG, JPEG, GIF, BMP, TIFF, WebP
- SVG files: ASCII art previews with viewBox information
- Binary files: Hex dump format with offset and ASCII preview
- Archives: Tree structure view for ZIP, JAR, WAR, EAR, IPA, APK, AAR files
- Database files: SQLite database browser with two-stage navigation
  - Table list view showing all tables with row counts and column information
  - Table content view with paginated data display
  - Proper column alignment using rune-aware width calculations
  - Smart handling of binary/non-printable data with box character (□) substitution
  - Lazy loading with pagination for large tables
- Lazy loading for large files with dynamic chunk loading

### Navigation & UI
- Navigate with arrow keys (↑/↓) or vim keys (j/k)
- Move between views with arrow keys (←/→) or vim keys (h/l)
- Selected items highlighted with theme-based colors
- Responsive layout with consistent structure:
  - Centered title at top with padding
  - Rounded content box with padding
  - Status line and centered footer at bottom
- Smooth viewport scrolling for long lists
- Centered key legends on all views
- Status messages displayed in dedicated status area
- Theme-aware colored search and filter status indicators
- Theme-aware colored warnings (e.g., SVG rendering limitations)
- All UI colors automatically adapt to selected theme
- Press 'q' or Ctrl+C to quit

## Configuration

The application supports theme-based customization through TOML configuration:

- Config location: `~/.config/simtool/config.toml` (or `$XDG_CONFIG_HOME/simtool/config.toml`)
- Generate example: `simtool --generate-config`
- All UI colors are derived from the selected syntax highlighting theme
- No hardcoded colors - everything is theme-based
- Supports 60+ built-in themes from the chroma library
- Automatic dark/light mode detection:
  - OSC escape sequence queries (supported by some terminals)
  - macOS system appearance as fallback
  - Environment variable override (`SIMTOOL_THEME_MODE`)
- Theme colors are intelligently extracted to create a cohesive color scheme
- Contrast adjustments ensure readability in both light and dark themes

## Key Dependencies

- `github.com/charmbracelet/bubbletea` - Terminal UI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/alecthomas/chroma/v2` - Syntax highlighting library
- `github.com/mattn/go-sqlite3` - SQLite database driver for database file viewing
- `github.com/BurntSushi/toml` - TOML configuration parsing