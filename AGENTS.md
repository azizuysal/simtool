# Agent instructions for SimTool

SimTool is a macOS terminal application for browsing iOS simulators and Android
emulators, their installed apps, and app files. Preserve the macOS 13 minimum and
both Intel and Apple Silicon builds. Physical devices are outside the current scope.

## Working in this repository

- Inspect the relevant code and configuration before editing. Make the smallest
  correct change that fits the existing architecture.
- Use `.mise.toml` for tool versions and `go.mod` / `go.sum` for dependencies.
  Run managed tools through `mise exec --`; do not install project tools globally.
- Read `Makefile`, `.golangci.yml`, and the applicable workflows before changing
  development or release commands. Builds require CGO for SQLite.
- Fix root causes. Do not add lint suppressions, `#nosec` annotations, or new
  exclusions to hide errors. Do not weaken tests to make them pass.
- Preserve unrelated worktree changes. Do not add secrets, credentials, or
  generated binaries to Git. Do not use emoji in code, documentation, or messages.
- Update user documentation only when the change makes existing instructions
  inaccurate. Keep release history in `CHANGELOG.md`; avoid repeating historical
  compatibility warnings elsewhere.

## Code map

- `cmd/simtool/main.go`: CLI options, startup, signal handling, and final cleanup.
- `internal/simulator/`: device discovery, app and file access, viewers, and
  Android session management. `Browser` and `DeviceBrowser` in `browser.go`
  coordinate platform-specific operations and resource ownership.
- `internal/tui/`: Bubble Tea model, commands, messages, navigation, and view state.
- `internal/tui/components/`: lists, viewers, layout, and footer rendering.
- `internal/config/`: TOML configuration, key bindings, and theme settings.
- `internal/ui/`: shared styles and display formatting.
- `docs/` and `.github/CONTRIBUTING.md`: user and contributor instructions.

## Behavior to preserve

- Both platforms are available by default. `--platform all|ios|android` selects
  the initial platform; the configurable `platform` action defaults to `p` in
  the Devices and All Apps views.
- Keep blocking device and file operations in Bubble Tea commands. Ignore stale
  asynchronous results after navigation by checking their request context.
- Key bindings live in `internal/config/keys.go`. Derive footer hints from the
  configured bindings, respect disabled actions, and show only active controls.
  Keep configuration examples and shortcut documentation consistent.
- Measure rendered terminal cell widths, including styled text. Check narrow
  layouts with populated lists so the footer and last item remain visible.
- Preserve iOS behavior when changing shared Android paths. iOS requires Xcode
  and `simctl`; Android requires SDK `adb`, `emulator`, and configured AVDs.

## Android lifecycle and data safety

- Discovery must not boot emulators. Start headless sessions only when needed,
  and stop only emulators started by SimTool. Leave pre-existing sessions running.
- Preserve cleanup on normal quit, cancellation, handled signals, and startup
  failures. Close `DeviceBrowser` before process exit; do not bypass cleanup with
  `os.Exit` or fatal logging while resources are owned.
- Do not kill the shared ADB server, automatically enable root, wipe AVD data,
  or force-stop apps to read their files. Use `run-as` or existing root access.
- Android file paths are virtual. Use the prepared local preview path for
  viewers, while retaining the original path for navigation and device access.
- ADB remote commands pass through a shell. Validate package names and quote
  path arguments with the existing helper; local `exec.Command` alone does not
  protect remote shell arguments. Preserve path and symlink access boundaries.
- Bound file copies and use private temporary directories and files. SQLite
  inspection copies must account for WAL/journal files and copy consistency;
  do not describe them as guaranteed transactional backups of live databases.
- Keep the Finder WebDAV bridge read-only, bound to loopback, and authenticated
  with a session capability. Unmount before removing temporary mount directories;
  report cleanup failures instead of deleting a still-mounted tree.

## Verification

Run commands from the repository root:

```sh
mise exec -- make fmt
mise exec -- go test ./...
mise exec -- go test -race ./...
mise exec -- make lint
mise exec -- make build
```

Use focused tests during development and scale final checks to the change.
Bug fixes need a regression test for the actual behavior. For release or build
changes, also run `mise exec -- make build-all` and the vulnerability check pinned
in `.github/workflows/lint.yml`. A successful build does not prove device behavior.

Android integration tests are opt-in via `SIMTOOL_ANDROID_AVD`; use a configured,
stopped AVD and verify cleanup without disturbing existing sessions. Native
Finder mount tests are opt-in via `SIMTOOL_TEST_NATIVE_WEBDAV=1` on macOS.
See `docs/development.md` for prerequisites and commands.

If sandbox restrictions block tool caches, use task-specific writable caches and
`GOTOOLCHAIN=local` with the installed mise version. Do not change the project
toolchain to work around an environment problem. Remove task-created scratch
artifacts when finished.

## Git and releases

- Commit, push, tag, and publish only when the user authorizes those actions.
  Once authorized, complete them without requesting the same approval again.
- Use conventional commits without assistant attribution or co-author trailers.
- Whenever pushing, push to both `origin` (GitHub) and `proxmox` (Gitea), and
  verify both remotes received the intended commits and any authorized tags.
- Release configuration lives in `.goreleaser.yml` and
  `.github/workflows/release.yml`. Homebrew publishing targets the separate
  `azizuysal/homebrew-tap` repository; verify workflow and tap results after a release.
- Follow `.github/SECURITY.md` for GitHub private vulnerability reporting.
