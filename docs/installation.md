# Installation Guide

SimTool can be installed using several methods. Choose the one that works best for you.

## Prerequisites

- macOS 13.0 or later
- For iOS browsing: full Xcode with an iOS Simulator runtime. Xcode Command Line Tools alone do not include `simctl`.
- For Android browsing: Android SDK `platform-tools` and `emulator`, configured Android Virtual Devices, and `ANDROID_HOME` when the SDK is not on the standard path. Android SDK build-tools `aapt2` is optional; package IDs are shown when it is unavailable.

Install only the platform tooling you plan to browse.

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

# Restrict the initial device set; the default is all platforms
simtool --platform android
```

## Android browsing

Android support works with Android Virtual Devices. Physical devices are outside the current scope. SimTool can start stopped AVDs headlessly to browse apps or load All Apps. An AVD that SimTool started is released after collecting All Apps metadata and is restarted if you browse it later. During normal browsing, owned emulators stay available until SimTool exits, receives Ctrl+C, SIGTERM, or SIGHUP, or booting fails. Emulators that were already running are left alone.

Private app data requires a debuggable app with `run-as`, or an ADB session that already has root access. SimTool never roots or wipes devices, or force-stops Android apps. Android app size is installed APK bytes; directory size and created time are unavailable and shown as unknown. Android Finder access is an authenticated loopback, read-only WebDAV mount using macOS `mount_webdav` and live ADB access, with no macFUSE dependency. SimTool removes the mount and its private preview copies on exit.

Previews transfer at most 256 MiB per Android file and refresh when reopened. SQLite copies include WAL or journal files, change detection, and `PRAGMA quick_check`; they are not [transactional live backups](https://www.sqlite.org/howtocorrupt.html#backup_or_restore_while_a_transaction_is_active). Stop app activity and retry if the database changes during copying.

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
