# SimTool

<p align="center">
  <img src="https://img.shields.io/badge/platform-macOS-blue" alt="macOS">
  <img src="https://img.shields.io/badge/go-1.27.1-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/github/license/azizuysal/simtool" alt="License">
  <img src="https://img.shields.io/github/v/release/azizuysal/simtool" alt="Release">
  <a href="https://codecov.io/gh/azizuysal/simtool"><img src="https://codecov.io/gh/azizuysal/simtool/branch/main/graph/badge.svg" alt="codecov"></a>
  <a href="https://sonarcloud.io/summary/new_code?id=azizuysal_simtool"><img src="https://sonarcloud.io/api/project_badges/measure?project=azizuysal_simtool&metric=alert_status" alt="Quality Gate Status"></a>
  <img src="https://img.shields.io/github/downloads/azizuysal/simtool/total" alt="Downloads">
</p>

<p align="center">
  <strong>A terminal UI for browsing iOS simulators and Android emulators</strong>
</p>

<p align="center">
  Navigate simulators and emulators, browse apps, explore files, and preview content from your terminal.
</p>

Android support uses configured Android Virtual Devices. Physical devices are outside the current scope.

![SimTool Demo](demo.gif)

## Features

### Simulator Management
- **List iOS simulators and Android emulators** with status indicators (running/stopped)
- **Boot selected devices** directly from the TUI
- **Smart filtering** to show only simulators with apps
- **Platform filtering** in Devices and All Apps (`p`: all, iOS, Android)
- **Real-time search** by name, runtime, state, or platform

### App Browsing
- **Browse installed apps** with detailed information
- **View app metadata**: Bundle ID, version, size, and available timestamps
- **All Apps view**: See apps from all simulators in one place
- **Open in Finder**: Access to iOS containers and authenticated, read-only Android folders
- **Search** across app metadata, simulator name, platform, and access status

### File Explorer
- **Navigate app containers** with an intuitive file browser
- **Breadcrumb navigation** for easy orientation
- **Smart file previews** based on content type
- **Android private-data access** when the app is debuggable with `run-as`, or ADB already has root access

### Rich File Viewing

<table>
<tr>
<td width="50%">

**📝 Text Files**
- Syntax highlighting for 100+ languages
- Automatic language detection
- Theme-aware colors
- Lazy loading for large files

</td>
<td width="50%">

**🖼️ Images**
- Terminal-based previews
- Support for PNG, JPEG, GIF, WebP, BMP, TIFF
- SVG rendering with ASCII art
- Automatic format detection

</td>
</tr>
<tr>
<td width="50%">

**📦 Archives**
- Browse ZIP, JAR, IPA, APK contents
- Tree structure visualization
- Compression statistics
- No extraction needed

</td>
<td width="50%">

**🗄️ Databases**
- SQLite browser with table navigation
- Paginated data viewing
- Schema inspection
- Column-aligned display

</td>
</tr>
</table>

### Additional Features
- **Property List Support**: Automatic binary plist → XML conversion
- **Binary File Viewer**: Hex dump with ASCII preview
- **Dynamic Theming**: 60+ themes, auto dark/light mode switching
- **Vim Navigation**: Full keyboard control with customizable shortcuts
- **Responsive Design**: Adapts to any terminal size
- **Lazy loading** for large content

## Requirements

- macOS 13.0 or later
- For iOS: full Xcode with an iOS Simulator runtime; the Command Line Tools alone do not include `simctl`
- For Android: Android SDK `platform-tools` and `emulator`, plus configured Android Virtual Devices. Set `ANDROID_HOME` when the SDK is not on the standard path. `aapt2` from Android SDK build-tools is optional; without it, apps are shown by package ID.
- Go 1.27.1, installed through [mise](https://mise.jdx.dev/), for building from source

You only need the platform tools for the platform you browse. iOS behavior is unchanged.

## Installation

### Homebrew (Recommended)
```bash
brew install azizuysal/tap/simtool
```

To update:

```bash
brew upgrade simtool
```

### Go Install
```bash
go install github.com/azizuysal/simtool/cmd/simtool@latest
```

### Download Binary
Download `simtool_<version>_darwin_all.tar.gz` from [Releases](https://github.com/azizuysal/simtool/releases), extract it with `tar -xzf`, and move `simtool` to a directory on your `PATH`.

### Build from Source
```bash
git clone https://github.com/azizuysal/simtool.git
cd simtool
mise install
mise exec -- make install
```

## Verifying releases

Release artifacts are signed with [Cosign](https://github.com/sigstore/cosign) using Sigstore's keyless signing. There are no long-lived keys — each signature's identity is tied to the specific GitHub Actions workflow run that produced the release, recorded in the public [Sigstore transparency log](https://search.sigstore.dev/).

New releases ship a `checksums.txt.sigstore.json` bundle alongside the archives. Signing the checksums file protects every artifact transitively via SHA-256.

To verify a new release:

```bash
# One-time: brew install cosign

# From the release page, download: the archive, checksums.txt, checksums.txt.sigstore.json
# Replace vX.Y.Z with the release tag you downloaded.
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity 'https://github.com/azizuysal/simtool/.github/workflows/release.yml@refs/tags/vX.Y.Z' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

# If verification succeeds, confirm the archive matches:
shasum -a 256 -c checksums.txt --ignore-missing
```

For the older v1.1.1 release, use its legacy `checksums.txt.pem` and `checksums.txt.sig` sidecars:

```bash
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity 'https://github.com/azizuysal/simtool/.github/workflows/release.yml@refs/tags/v1.1.1' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

Homebrew installs don't need manual verification — the tap formula pins each release to a specific SHA-256, so any tampering after the fact is caught by `brew install` itself.

In addition to signatures, each release includes a **CycloneDX SBOM** sidecar (`*.cdx.json`) cataloging every dependency baked into the binary.

## 📖 Usage

### Quick Start
```bash
# Launch SimTool
simtool

# Start with all apps view
simtool --apps

# Limit the initial device set to iOS or Android (default: all)
simtool --platform android
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate up/down |
| `←/→` or `h/l` | Go back/enter |
| `Space` | Boot simulator / Open in Finder |
| `/` | Search mode |
| `f` | Filter (simulators with apps only) |
| `p` | Cycle platform filter: all, iOS, Android |
| `q` | Quit |
| `Home/End` | Jump to top/bottom |

All shortcuts are [customizable](#configuration).

## ⚙️ Configuration

SimTool uses a TOML configuration file located at `~/.config/simtool/config.toml`.

```toml
# Start with all apps view by default
[startup]
initial_view = "all_apps"

# Theme configuration
[theme]
mode = "auto"  # auto, dark, or light
dark_theme = "dracula"
light_theme = "github"

# Custom key bindings
[keys]
up = ["up", "k"]
down = ["down", "j"]
quit = ["q", "ctrl+c"]
platform = ["p"]
```

### Android file access and previews

SimTool never roots, wipes, or force-stops Android apps. Private files are available only through a debuggable app's `run-as` access or an ADB session that already has root access; otherwise the TUI reports the restriction. Android app size is the installed APK bytes; directory size and created time are unavailable and shown as unknown. Android file previews are private local copies, limited to 256 MiB per file, and reopening a file refreshes its copy.

Finder access to Android folders uses an authenticated loopback, read-only WebDAV mount backed by live ADB access and provided by macOS. It does not use macFUSE. The mount and private previews are cleaned up when SimTool exits.

SQLite previews copy the database together with WAL or journal files, detect changes, and run `PRAGMA quick_check`. They are not [transactional live backups](https://www.sqlite.org/howtocorrupt.html#backup_or_restore_while_a_transaction_is_active). Stop app activity and retry if a changing database cannot be copied consistently; SimTool does not force-stop apps.

Generate an example configuration:
```bash
simtool --generate-config
```

See [Configuration Guide](docs/configuration.md) for all options.

## 🎨 Themes

SimTool includes 60+ beautiful syntax highlighting themes. Popular choices:

**Dark**: `dracula`, `monokai`, `github-dark`, `nord`, `tokyo-night`  
**Light**: `github`, `solarized-light`, `tango`, `papercolor-light`

List all themes:
```bash
simtool --list-themes
```


## Contributing

Contributions are welcome! Please read the [Contributing Guide](.github/CONTRIBUTING.md) for details on the development process.

### Development Setup
```bash
git clone https://github.com/azizuysal/simtool.git
cd simtool
mise install
mise exec -- make build
```

See [Development Guide](docs/development.md) for architecture details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) - The delightful TUI framework
- Syntax highlighting by [Chroma](https://github.com/alecthomas/chroma)
- Styled with [Lipgloss](https://github.com/charmbracelet/lipgloss)

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=azizuysal/simtool&type=Date)](https://star-history.com/#azizuysal/simtool&Date)

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/azizuysal">Aziz Uysal</a>
</p>
