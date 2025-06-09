# Installation Guide

SimTool can be installed using several methods. Choose the one that works best for you.

## Prerequisites

- **macOS** (required - SimTool uses iOS simulator functionality)
- **Xcode** or **Xcode Command Line Tools** installed
- **iOS Simulators** installed via Xcode

## Installation Methods

### Homebrew (Recommended)

```bash
# Add the tap
brew tap azizuysal/simtool

# Install simtool
brew install simtool
```

To update:
```bash
brew upgrade simtool
```

### Go Install

If you have Go installed:

```bash
go install github.com/azizuysal/simtool/cmd/simtool@latest
```

Make sure `$GOPATH/bin` is in your PATH.

### Download Binary

1. Go to the [Releases](https://github.com/azizuysal/simtool/releases) page
2. Download the appropriate binary for your system:
   - `simtool-darwin-amd64` for Intel Macs
   - `simtool-darwin-arm64` for Apple Silicon Macs
3. Make it executable:
   ```bash
   chmod +x simtool-darwin-*
   ```
4. Move to your PATH:
   ```bash
   sudo mv simtool-darwin-* /usr/local/bin/simtool
   ```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/azizuysal/simtool.git
cd simtool

# Build
make build

# Install
make install
```

## Verify Installation

```bash
simtool --version
```

## First Run

Simply run:
```bash
simtool
```

Or start with all apps view:
```bash
simtool --apps
```

## Troubleshooting

### "xcrun: error: unable to find utility 'simctl'"

Install Xcode Command Line Tools:
```bash
xcode-select --install
```

### No simulators found

Make sure you have iOS simulators installed:
1. Open Xcode
2. Go to Preferences → Platforms
3. Install iOS simulators

### Permission denied

If you get permission errors, make sure the binary is executable:
```bash
chmod +x $(which simtool)
```

## Uninstallation

### Homebrew
```bash
brew uninstall simtool
brew untap azizuysal/simtool
```

### Manual
```bash
rm /usr/local/bin/simtool
rm -rf ~/.config/simtool
```

## Next Steps

- See [Configuration](configuration.md) to customize SimTool
- Check the [README](../README.md) for usage examples