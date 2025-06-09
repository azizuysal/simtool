# Development Guide

This guide covers setting up a development environment for SimTool and explains the codebase structure.

## Development Setup

### Prerequisites

- Go 1.21 or later
- macOS with Xcode
- Git
- Make (optional but recommended)

### Getting Started

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/simtool.git
   cd simtool
   go mod download
   ```

2. **Build and Run**
   ```bash
   make build
   ./simtool
   
   # Or directly:
   go run ./cmd/simtool
   ```

3. **Run Tests**
   ```bash
   make test
   
   # With coverage:
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

## Project Structure

```
simtool/
├── cmd/simtool/           # Application entry point
│   └── main.go           # CLI flags, initialization
├── internal/             # Internal packages (not importable)
│   ├── config/           # Configuration management
│   │   ├── config.go     # Config loading and structures
│   │   ├── keys.go       # Keyboard shortcut mapping
│   │   ├── theme.go      # Theme extraction and management
│   │   └── detect.go     # Terminal theme detection
│   ├── simulator/        # iOS simulator interaction
│   │   ├── simulator.go  # Core types and interfaces
│   │   ├── fetcher.go    # xcrun simctl wrapper
│   │   ├── app.go        # App information and operations
│   │   ├── files.go      # File system operations
│   │   └── viewer.go     # File content rendering
│   ├── tui/              # Terminal UI (Bubble Tea)
│   │   ├── model.go      # Application state
│   │   ├── update.go     # Message handling
│   │   ├── view.go       # Main view orchestration
│   │   ├── viewport.go   # Scrolling logic
│   │   └── components/   # Reusable UI components
│   └── ui/               # UI utilities
│       ├── styles.go     # Lipgloss style definitions
│       └── format.go     # Formatting helpers
├── docs/                 # Documentation
├── scripts/              # Build and utility scripts
└── Makefile             # Build automation
```

## Key Concepts

### Architecture

SimTool follows clean architecture principles:

1. **Separation of Concerns**: Business logic (simulator package) is separate from UI (tui package)
2. **Dependency Injection**: Interfaces allow for easy testing and mocking
3. **Component-Based UI**: Reusable components for consistent rendering

### Bubble Tea Framework

SimTool uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI:

- **Model**: Contains application state
- **Update**: Handles messages and updates state
- **View**: Renders the UI based on state

Example flow:
```go
// Model
type Model struct {
    simulators []simulator.Item
    cursor     int
}

// Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle key press
    }
    return m, nil
}

// View
func (m Model) View() string {
    // Render UI
}
```

### Adding a New Feature

1. **Plan the Feature**
   - Identify affected packages
   - Design data structures
   - Plan UI changes

2. **Implement Business Logic**
   - Add to simulator package if iOS-related
   - Create interfaces for testability

3. **Add UI Components**
   - Create new component in components/
   - Follow existing patterns

4. **Wire Everything Together**
   - Update model.go with new state
   - Add message types and handlers
   - Update view.go to render

5. **Add Tests**
   - Unit tests for business logic
   - Component tests for UI

### Testing

#### Unit Tests

```go
func TestGetAppsForSimulator(t *testing.T) {
    // Arrange
    mockFetcher := &MockFetcher{...}
    
    // Act
    apps, err := GetAppsForSimulator("udid", true)
    
    // Assert
    assert.NoError(t, err)
    assert.Len(t, apps, 2)
}
```

#### Mocking

Use interfaces for external dependencies:

```go
type Fetcher interface {
    Fetch() ([]Item, error)
    Boot(udid string) error
}

type MockFetcher struct {
    FetchFunc func() ([]Item, error)
}
```

### Code Style

1. **Go Standards**
   - Run `gofmt` before committing
   - Follow [Effective Go](https://golang.org/doc/effective_go)
   - Use meaningful variable names

2. **Project Conventions**
   - Interfaces end with "-er" (Fetcher, Viewer)
   - Error messages start with lowercase
   - Use table-driven tests

3. **Comments**
   - Document all exported types and functions
   - Explain "why", not "what"

### Debugging

1. **Enable Debug Logging**
   ```go
   // Add debug prints (remove before committing)
   fmt.Fprintf(os.Stderr, "DEBUG: %v\n", variable)
   ```

2. **Use Delve**
   ```bash
   dlv debug ./cmd/simtool
   ```

3. **Bubble Tea Debug Mode**
   ```go
   p := tea.NewProgram(model, tea.WithAltScreen())
   ```

## Common Tasks

### Adding a New View

1. Add ViewState constant
2. Create component in components/
3. Add case in view.go
4. Handle navigation in update.go

### Adding a Config Option

1. Update config.Config struct
2. Add to defaultConfig
3. Update config.example.toml
4. Document in configuration.md

### Adding a Keyboard Shortcut

1. Update KeysConfig in config/keys.go
2. Add default in defaultKeysConfig
3. Handle in handleKeyPress
4. Update documentation

## Performance Considerations

1. **Lazy Loading**: Load file content in chunks
2. **Caching**: Cache lexers for syntax highlighting
3. **Efficient Rendering**: Only render visible content
4. **Async Operations**: Use tea.Cmd for blocking operations

## Release Process

1. Update version in Makefile
2. Update CHANGELOG.md
3. Create git tag
4. Push tag (triggers CI release)

## Troubleshooting Development Issues

### Build Errors

```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

### Test Failures

```bash
# Run specific test
go test -v -run TestName ./internal/simulator

# Update golden files
go test ./... -update
```

### Terminal Issues

- Test in different terminals (iTerm2, Terminal.app, WezTerm)
- Check TERM environment variable
- Verify terminal capabilities

## Resources

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Chroma Syntax Highlighting](https://github.com/alecthomas/chroma)
- [xcrun simctl Documentation](https://developer.apple.com/documentation/xcode/simctl)

## Getting Help

- Check existing issues on GitHub
- Ask in discussions
- Read the test files for examples
- Explore the codebase with `go doc`