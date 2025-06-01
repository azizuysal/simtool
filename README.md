# SimTool

A terminal UI application for managing iOS simulators on macOS.

## Features

- List all available iOS simulators
- View simulator status (Running/Not Running)
- Navigate with arrow keys or vim-style keys (j/k)
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

- `↑` / `k` - Move up
- `↓` / `j` - Move down
- `Home` - Jump to first simulator
- `End` - Jump to last simulator
- `q` / `Ctrl+C` - Quit

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