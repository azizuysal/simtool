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
  - Supports TypeScript, TSX, JavaScript, JSX, Swift, Objective-C, Objective-C++, and many more
  - Theme-aware syntax colors that adapt to dark/light mode
- **Property list files**: Automatic handling of both XML and binary plist formats
  - Binary plists automatically converted to XML for viewing using macOS plutil
  - XML syntax highlighting for all plist files
  - Clear indication when viewing converted binary plists
- **Images**: Terminal-based previews for PNG, JPEG, GIF, BMP, TIFF, WebP
- **SVG files**: ASCII art previews with dimension information
- **Binary files**: Hex dump format with offset and ASCII preview
- **Archives**: Tree structure view for ZIP, JAR, WAR, EAR, IPA, APK, AAR files
- **Database files**: SQLite database browser with two-stage navigation
  - Table list view showing all tables in the database
  - Table content view with paginated data, column headers, and proper alignment
  - Smart handling of binary data with box character (□) substitution
  - Rune-aware column width calculation for proper multi-byte character support
- Lazy loading for large files with automatic chunking

### User Interface
- Clean, modern TUI with rounded borders and centered layouts
- Navigate with arrow keys or vim-style keys (h,j,k,l)
- Smooth viewport scrolling for long lists
- Visual selection indicator with theme-aware colors
- Centered key legends showing available shortcuts
- Dedicated status area for search and filter indicators
- Theme-aware color scheme that adapts to dark/light mode
- Dynamic theme switching - automatically updates when terminal theme changes
- All UI colors derived from syntax highlighting themes for consistency

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

# Show help
simtool --help
simtool -h

# Show version
simtool --version
simtool -v

# Generate example configuration file
simtool --generate-config
simtool -g

# Show configuration file path
simtool --show-config-path
simtool -c

# List available syntax highlighting themes
simtool --list-themes
simtool -l
```

## Configuration

SimTool automatically adapts its color scheme based on your selected theme. All UI colors are derived from the syntax highlighting theme for a cohesive appearance.

### Theme Selection

SimTool supports 60+ built-in themes and can automatically switch between dark and light themes based on your terminal.

Configuration file location: `~/.config/simtool/config.toml` (or `$XDG_CONFIG_HOME/simtool/config.toml`)

### Terminal Theme Detection

SimTool automatically detects your terminal's theme and dynamically updates its color scheme:
1. **Live theme detection** - UI colors update automatically when you change your terminal theme
2. **OSC escape sequences** - Supported by some terminals like WezTerm for accurate theme detection
3. **macOS system appearance** - Used as a fallback when OSC queries aren't supported
4. **Environment variable override** - `SIMTOOL_THEME_MODE` for manual control

Note: Theme switching works best with terminals that support OSC queries. For terminals that don't (VS Code, iTerm2), use the environment variable or config file to set the theme.

**To manually set the theme mode:**

1. **Using configuration file** (recommended):
   ```toml
   [theme]
   mode = "light"  # or "dark"
   ```

2. **Using environment variable**:
   ```bash
   # For light theme
   SIMTOOL_THEME_MODE=light simtool
   
   # For dark theme  
   SIMTOOL_THEME_MODE=dark simtool
   ```

3. **Create an alias** for convenience:
   ```bash
   # Add to your shell profile
   alias simtool-light='SIMTOOL_THEME_MODE=light simtool'
   alias simtool-dark='SIMTOOL_THEME_MODE=dark simtool'
   ```

1. Generate an example configuration:
   ```bash
   simtool --generate-config
   ```

2. Create your config:
   ```bash
   cd ~/.config/simtool
   cp config.example.toml config.toml
   ```

### Configuration Options

```toml
[theme]
# Theme mode: "auto", "dark", or "light"
mode = "auto"

# Theme for dark mode (or when mode="dark")
dark_theme = "github-dark"

# Theme for light mode (or when mode="light")
light_theme = "github"
```

**Examples:**
- Always use dark theme: `mode = "dark"` with your preferred `dark_theme`
- Always use light theme: `mode = "light"` with your preferred `light_theme`
- Auto-switch based on terminal: `mode = "auto"` (default)

### Popular Themes

**Dark themes:**
- `github-dark` - GitHub's dark theme (default)
- `monokai` - Vibrant colors on dark background
- `dracula` - Dark theme with purple accents
- `nord` - Arctic, north-bluish palette
- `onedark` - Atom One Dark
- `solarized-dark` - Precision colors

**Light themes:**
- `github` - GitHub's light theme (default)
- `solarized-light` - Solarized light variant
- `gruvbox-light` - Retro groove (light)
- `tango` - GNOME Tango

View all available themes:
```bash
simtool --list-themes
```

### Keyboard Shortcuts

SimTool supports fully customizable keyboard shortcuts. The default shortcuts are:

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

#### Customizing Shortcuts

You can customize any keyboard shortcut in your config file. See the `[keys]` section in the example configuration:

```toml
[keys]
# Each action can have multiple keys assigned
up = ["up", "k"]           # Move cursor up
down = ["down", "j"]       # Move cursor down
left = ["left", "h"]       # Go back / navigate left
right = ["right", "l"]     # Enter / navigate right
# ... etc
```

Example customizations:
- Use only arrow keys: `up = ["up"]`, `down = ["down"]`
- Use only vim keys: `up = ["k"]`, `down = ["j"]`
- Add custom keys: `quit = ["q", "ctrl+c", "ctrl+d"]`
- Disable a shortcut: `filter = []`

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
- **Property list files**: XML syntax highlighting with automatic binary plist conversion
- **Images**: Metadata box and terminal-based preview
- **Binary files**: Hex dump with ASCII representation
- **ZIP archives**: Tree view with file/folder counts and overall compression ratio
- **SVG files**: Metadata and terminal-based preview using ASCII art
- **Database files**: 
  - Table list with schema information
  - Table content viewer with proper column alignment
  - Pagination controls showing current data range (e.g., "1-50 of 1000")

## Development

### Project Structure

```
simtool/
├── cmd/simtool/        # Application entry point
│   └── main.go
├── internal/
│   ├── config/         # Configuration and theme management
│   │   ├── config.go      # Config loading and theme selection
│   │   ├── config_test.go
│   │   ├── detect.go      # Terminal theme detection
│   │   ├── detect_test.go
│   │   ├── paths.go       # Config file paths
│   │   ├── styles.go      # Style generation from themes
│   │   ├── terminal_osc.go    # OSC escape sequences
│   │   ├── terminal_query.go  # Terminal background detection
│   │   ├── theme.go       # Theme color extraction
│   │   └── theme_test.go
│   ├── simulator/      # Simulator types and fetching logic
│   │   ├── simulator.go   # Core types
│   │   ├── fetcher.go     # xcrun simctl integration
│   │   ├── app.go         # App information
│   │   ├── files.go       # File operations
│   │   ├── viewer.go      # File content viewing
│   │   └── *_test.go      # Test files
│   ├── tui/           # Terminal UI components  
│   │   ├── model.go       # Bubble Tea model
│   │   ├── update.go      # Message handling
│   │   ├── view.go        # View orchestrator
│   │   ├── viewport.go    # Scrolling logic
│   │   ├── keys.go        # Key bindings
│   │   ├── components/    # Reusable UI components
│   │   └── *_test.go      # Test files
│   └── ui/            # UI styles and formatting
│       ├── styles.go      # Dynamic style functions
│       ├── styles_test.go
│       ├── format.go      # Formatting helpers
│       └── format_test.go
├── scripts/
│   └── coverage-badge.sh
├── .gitignore
├── CLAUDE.md          # AI assistant guidance
├── go.mod
├── go.sum
├── Makefile           # Build automation
└── README.md
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