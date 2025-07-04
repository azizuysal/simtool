# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.2] - 2025-07-04

### Fixed
- Pass HOMEBREW_TAP_GITHUB_TOKEN to GoReleaser environment
- Enable automatic Homebrew formula updates

## [1.0.1] - 2025-07-04

### Changed
- Added download count badge to README
- Removed prerelease filter from version badge
- Test automatic Homebrew formula updates

## [1.0.0] - 2025-07-04

### Added
- Initial release of SimTool
- List all iOS simulators with status indicators
- Browse installed apps with details (bundle ID, version, size, last modified)
- Navigate app container files and directories
- View file contents with appropriate rendering:
  - Syntax highlighting for 100+ programming languages
  - Image previews in terminal (PNG, JPEG, GIF, WebP, BMP, TIFF)
  - SVG preview with ASCII art
  - Hex dump for binary files
  - Archive contents viewer (ZIP, JAR, IPA, APK)
  - SQLite database browser with table navigation
  - Automatic binary plist conversion to XML
- Boot simulators directly from the UI
- Open apps and files in Finder
- Search functionality for simulators and apps
- Filter to show only simulators with apps
- "All Apps" view to see apps from all simulators
- Configurable keyboard shortcuts
- Dynamic theme switching based on terminal appearance
- 60+ syntax highlighting themes
- Lazy loading for large files
- Configuration via TOML file

### Features
- Cross-terminal compatibility
- Responsive design that adapts to window size
- Vim-style navigation keys
- Status indicators and loading states
- Breadcrumb navigation for file browsing
- Human-friendly date formatting
- Smart file type detection

### Technical
- Built with Bubble Tea TUI framework
- Uses `xcrun simctl` for simulator operations
- Chroma for syntax highlighting
- Clean architecture with separated concerns
- Comprehensive test coverage
- Cross-platform file operations