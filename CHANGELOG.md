# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.1] - 2026-04-24

### Fixed
- Files dominated by non-ASCII UTF-8 content (CJK, accented Latin, emoji) were classified as binary and rendered as hex dumps. `isTextContent` now iterates at the rune level with `unicode.IsPrint` so any valid UTF-8 text file is recognized as text.

### Changed
- `initChromaStyle` no longer silently swallows config load errors or unknown theme names. Both paths still fall back to `github-dark` for rendering, but the underlying cause is now logged to the tea debug log (`$XDG_CACHE_HOME/simtool/debug.log` or OS equivalent) so users can discover why their chosen theme didn't load.

### Internal
- Removed three layers of unreachable defensive fallbacks in `initChromaStyle` (dead `terminal256`/`terminal` formatter fallbacks, a lowercased style re-lookup redundant with chroma's case-insensitive registry, and a hardcoded `github-dark` assignment on config error that discarded the valid defaults returned by `config.Load`).
- Split the ~200-line `Update()` message dispatcher in `internal/tui/update.go` into ten per-message-type handlers. Extracted the inline SVG unsupported-features scan into a free `detectSVGWarning` helper. Zero behavior change.
- Test coverage push: `internal/simulator` 62.6% → 83.7%, `internal/config` 28.0% → 62.0%. New tests cover `readArchiveInfo`, `readBinaryFile`, `readImageInfo`, `readTextFile`, `detectContentLanguage`, `getAppsFromDataDir`, `FetchSimulators`, `Boot`, `getAppCountFromDataDir`, `initChromaStyle` branches, `DefaultKeys`/`NewKeyMap`/`FormatKeys`, `GenerateStyles`, `parseOSC11Response`, and `SaveExample`/`getConfigDir`/`getConfigPath`.
- Tightened a pre-existing SVG test that asserted an error message the production code never emits. It now exercises the real contract (malformed XML → parse-error message stored in `info.Preview.Rows`).

## [1.1.0] - 2026-04-10

### Added
- TOML config schema validation at load time: unknown keys (typos, stale schemas) and invalid enum values are now reported as descriptive errors instead of being silently ignored
- CycloneDX SBOM generation for each release archive via GoReleaser
- Cosign keyless signatures on release checksums via GitHub Actions OIDC (Sigstore); `checksums.txt.sig` and `checksums.txt.pem` sidecars published alongside each release. Verification command documented in README
- `govulncheck` dependency vulnerability scan in CI, alongside the existing `golangci-lint` job

### Changed
- Debug log moved from the process working directory to `$XDG_CACHE_HOME/simtool/debug.log` (or OS equivalent), with 0700 parent and 0600 file permissions. Running `simtool` from arbitrary directories no longer leaves `debug.log` scattered around.
- User config directory (`~/.config/simtool/`) permissions tightened from 0755 to 0700
- File viewer scroll indicator now shows accurate `(start-end of total)` line counts for archives, databases, and images; previously used rough heuristics that could be off by a factor of 2 or more depending on content
- Upgraded Go toolchain to 1.26.2 (pinned via `.mise.toml`); bumped direct dependencies including `mattn/go-sqlite3` 1.14.17 → 1.14.42, `alecthomas/chroma/v2` 2.18.0 → 2.23.1, `charmbracelet/bubbletea` 1.3.5 → 1.3.10, `BurntSushi/toml` 1.5.0 → 1.6.0
- CI workflows now read the Go version from `go.mod` as a single source of truth instead of four hardcoded copies; `lint.yml` and `codeql.yml` moved to `ubuntu-latest` (linting doesn't need macOS)

### Fixed
- Multi-byte characters (emoji, CJK, accents) in text files were cut mid-codepoint when long lines were truncated for display. Truncation is now rune-aware.
- SVG rasterize panic would print to stdout and corrupt the TUI rendering surface. The panic handler now propagates the error through a named return.
- Database viewer silently treated missing files as empty databases because `?mode=ro` was not honored by `go-sqlite3` without the `file:` URI prefix. Now returns a descriptive error.
- Database table names were interpolated into SQL queries without escaping embedded double quotes, so a file containing a crafted table name could break out of the identifier quoting. Now escaped via a `quoteSQLiteIdentifier` helper that doubles embedded quotes per SQL standard.
- `DetectTerminalDarkModeLive` was called from the periodic tick handler and from goroutines spawned by `tea.Cmd` callbacks without synchronization. Now mutex-protected with a 2-second TTL cache.
- Boot-simulator error messages dropped the underlying error chain (`%s` instead of `%w`), breaking `errors.Is` / `errors.As` against the returned error

### Internal
- Structural refactors with no user-visible impact: split 1502-line `viewer.go` into 7 per-format files; split 496-line `handleKeyPress` into 7 per-view-state handlers; grouped 56 `Model` fields into 7 per-view substates so back-navigation clears become single-line zero-value resets; unified `Component` interface across list/viewer TUI components; extracted `flashStatus` / `clearStatusAfter` helpers replacing 11 inlined tick-clear patterns; lifted magic numbers (chunk sizes, scan buffers, SVG dimensions, lazy-load offsets) to named constants
- Expanded the linter set from 5 linters to 14: added errorlint, gocritic, gosec, misspell, nilerr, revive, sqlclosecheck, unconvert, bodyclose. Migrated `.golangci.yml` to golangci-lint v2 format. Enforced `gofmt` + `goimports` via `gci` with strict stdlib/external/local section order
- Test coverage went from 29% total to roughly 60%: database viewer 0% → 84–100% per function, file_viewer package 0% → 92.9%, handleKeyPress dispatcher and per-state handlers 0% → 85–100%. New tests for `readAppInfo`, `findDataContainer`, and `getAppsFromListApps` via a swappable `CommandExecutor`
- Removed dead `parseAppListJSON` function and its tests; the production path uses an inline plist-style parser

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