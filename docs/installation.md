# Installation Guide

SimTool can be installed using several methods. Choose the one that works best for you.

## Prerequisites

- macOS 13.0 or later
- Full Xcode with an iOS Simulator runtime. Xcode Command Line Tools alone do not include `simctl`.

## Installation Methods

### Homebrew (Recommended)

```bash
brew install azizuysal/tap/simtool
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
2. Download `simtool_<version>_darwin_all.tar.gz`.
3. Extract it:
   ```bash
   tar -xzf simtool_<version>_darwin_all.tar.gz
   ```
4. Move to your PATH:
   ```bash
   sudo mv simtool /usr/local/bin/simtool
   ```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/azizuysal/simtool.git
cd simtool

mise install
mise exec -- make build

# Install
mise exec -- make install
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

Install full Xcode, then select it:
```bash
sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer
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
brew untap azizuysal/tap
```

### Manual
```bash
rm /usr/local/bin/simtool
rm -rf ~/.config/simtool
```

## Next Steps

- See [Configuration](configuration.md) to customize SimTool
- Check the [README](../README.md) for usage examples
