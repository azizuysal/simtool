# SimTool v1.0.0 - Initial Release 🎉

We're excited to announce the first stable release of SimTool - a beautiful and powerful Terminal UI for managing iOS Simulators!

## 🌟 Highlights

SimTool brings the power of iOS Simulator management to your terminal with an intuitive interface, rich file viewing capabilities, and seamless navigation. Whether you're debugging apps, exploring container data, or managing multiple simulators, SimTool makes it fast and enjoyable.

## ✨ Key Features

### 📱 Simulator Management
- List all iOS simulators with real-time status indicators
- Boot simulators directly from the TUI
- Smart filtering to show only simulators with apps
- Lightning-fast search by name, runtime, or state

### 🗂️ App Browsing
- Browse installed apps with detailed metadata
- View bundle IDs, versions, sizes, and modification dates
- "All Apps" view to see apps across all simulators
- Quick Finder access to app containers

### 📁 File Explorer
- Navigate app containers with an intuitive file browser
- Breadcrumb navigation for easy orientation
- Smart file type detection and previews
- Open any file or folder in Finder

### 🎨 Rich File Viewing
- **Syntax Highlighting**: 100+ languages with theme-aware colors
- **Image Preview**: View PNG, JPEG, GIF, WebP, BMP, TIFF in terminal
- **Database Browser**: Explore SQLite databases interactively
- **Archive Viewer**: Peek inside ZIP, JAR, IPA, APK files
- **Binary Files**: Automatic hex dump with ASCII preview
- **Plist Support**: Auto-converts binary plists to readable XML

### ⚡ Performance & UX
- Vim-style keyboard navigation
- 60+ beautiful syntax themes
- Auto dark/light mode switching
- Responsive design for any terminal size
- Lazy loading for large files
- Configurable keyboard shortcuts

## 📋 System Requirements
- macOS 10.15 or later
- Xcode Command Line Tools
- Go 1.24.4 or later (for building from source)

## 🚀 Installation

### Homebrew (Recommended)
```bash
brew tap azizuysal/simtool
brew install simtool
```

### Direct Download
Download the universal binary from the [releases page](https://github.com/azizuysal/simtool/releases/tag/v1.0.0).

### Build from Source
```bash
go install github.com/azizuysal/simtool/cmd/simtool@v1.0.0
```

## 🎮 Quick Start
```bash
# Launch SimTool
simtool

# Start with all apps view
simtool --apps
```

## 📖 Documentation
- [Installation Guide](https://github.com/azizuysal/simtool/blob/main/docs/installation.md)
- [Configuration Guide](https://github.com/azizuysal/simtool/blob/main/docs/configuration.md)
- [Development Guide](https://github.com/azizuysal/simtool/blob/main/docs/development.md)

## 🤝 Contributing
We welcome contributions! Please see our [Contributing Guide](https://github.com/azizuysal/simtool/blob/main/CONTRIBUTING.md) for details.

## 🙏 Acknowledgments
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm
- Syntax highlighting powered by [Chroma](https://github.com/alecthomas/chroma)
- Inspired by the iOS development community's need for better terminal tools

## 📝 License
MIT License - see [LICENSE](https://github.com/azizuysal/simtool/blob/main/LICENSE) for details.

---

**Full Changelog**: https://github.com/azizuysal/simtool/commits/v1.0.0

Thank you to everyone who helped test the beta versions and provided valuable feedback! 🙌