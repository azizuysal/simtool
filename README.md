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

# Generate example configuration file
simtool --generate-config

# Show configuration file path
simtool --show-config-path

# List available syntax highlighting themes
simtool --list-themes
```

## Configuration

SimTool automatically adapts its color scheme based on your selected theme. All UI colors are derived from the syntax highlighting theme for a cohesive appearance.

### Theme Selection

SimTool supports 60+ built-in themes and can automatically switch between dark and light themes based on your terminal.

Configuration file location: `~/.config/simtool/config.toml` (or `$XDG_CONFIG_HOME/simtool/config.toml`)

### Terminal Theme Detection

SimTool attempts to automatically detect your terminal's theme using:
1. OSC escape sequences (supported by some terminals like WezTerm)
2. macOS system appearance as a fallback
3. Environment variable override (`SIMTOOL_THEME_MODE`)

Note: Many modern terminals (VS Code, iTerm2) don't support OSC queries for security reasons.

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
- **Database files**: 
  - Table list with schema information
  - Table content viewer with proper column alignment
  - Pagination controls showing current data range (e.g., "1-50 of 1000")

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