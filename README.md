# SimTool 🛠️

<p align="center">
  <img src="https://img.shields.io/badge/platform-macOS-blue" alt="macOS">
  <img src="https://img.shields.io/badge/go-%3E%3D1.24.4-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/github/license/azizuysal/simtool" alt="License">
  <img src="https://img.shields.io/github/v/release/azizuysal/simtool" alt="Release">
  <a href="https://codecov.io/gh/azizuysal/simtool"><img src="https://codecov.io/gh/azizuysal/simtool/branch/main/graph/badge.svg" alt="codecov"></a>
  <a href="https://sonarcloud.io/summary/new_code?id=azizuysal_simtool"><img src="https://sonarcloud.io/api/project_badges/measure?project=azizuysal_simtool&metric=alert_status" alt="Quality Gate Status"></a>
  <img src="https://img.shields.io/github/downloads/azizuysal/simtool/total" alt="Downloads">
</p>

<p align="center">
  <strong>A beautiful and powerful TUI for managing iOS Simulators</strong>
</p>

<p align="center">
  Navigate your iOS simulators, browse apps, explore files, and preview content—all from your terminal.
</p>

![SimTool Demo](demo.gif)

## ✨ Features

### 🚀 Simulator Management
- **List all iOS simulators** with status indicators (running/stopped)
- **Boot simulators** directly from the TUI
- **Smart filtering** to show only simulators with apps
- **Real-time search** by name, runtime, or state

### 📱 App Browsing  
- **Browse installed apps** with detailed information
- **View app metadata**: Bundle ID, version, size, last modified date
- **All Apps view**: See apps from all simulators in one place
- **Open in Finder**: Quick access to app containers
- **Lightning-fast search** across all app properties

### 📁 File Explorer
- **Navigate app containers** with an intuitive file browser
- **Breadcrumb navigation** for easy orientation
- **Smart file previews** based on content type
- **Quick Finder access** for any file or folder

### 🎨 Rich File Viewing

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

### ⚡ Additional Features
- **Property List Support**: Automatic binary plist → XML conversion
- **Binary File Viewer**: Hex dump with ASCII preview
- **Dynamic Theming**: 60+ themes, auto dark/light mode switching
- **Vim Navigation**: Full keyboard control with customizable shortcuts
- **Responsive Design**: Adapts to any terminal size
- **Lightning Fast**: Instant navigation and lazy loading

## 📋 Requirements

- macOS 10.15 or later
- Xcode Command Line Tools
- Go 1.24.4 or later (for building from source)

## 🚀 Installation

### Homebrew (Recommended)
```bash
brew tap azizuysal/tap
brew install simtool
```

### Go Install
```bash
go install github.com/azizuysal/simtool/cmd/simtool@latest
```

### Download Binary
Download from [Releases](https://github.com/azizuysal/simtool/releases) page.

### Build from Source
```bash
git clone https://github.com/azizuysal/simtool.git
cd simtool
make install
```

## 🔐 Verifying releases

Release artifacts are signed with [Cosign](https://github.com/sigstore/cosign) using Sigstore's keyless signing. There are no long-lived keys — each signature's identity is tied to the specific GitHub Actions workflow run that produced the release, recorded in the public [Sigstore transparency log](https://search.sigstore.dev/).

Every release ships with `checksums.txt.sig` and `checksums.txt.pem` sidecars alongside the archives. Signing the checksums file protects every artifact transitively via SHA-256.

To verify a downloaded release:

```bash
# One-time: brew install cosign

# From the release page, download: the archive, checksums.txt, checksums.txt.sig, checksums.txt.pem
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity-regexp '^https://github.com/azizuysal/simtool/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

# If verification succeeds, confirm the archive matches:
shasum -a 256 -c checksums.txt
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
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate up/down |
| `←/→` or `h/l` | Go back/enter |
| `Space` | Boot simulator / Open in Finder |
| `/` | Search mode |
| `f` | Filter (simulators with apps only) |
| `q` | Quit |
| `g/G` | Jump to top/bottom |

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
```

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


## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and development process.

### Development Setup
```bash
git clone https://github.com/azizuysal/simtool.git
cd simtool
go mod download
make build
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