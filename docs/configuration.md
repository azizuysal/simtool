# Configuration Guide

SimTool uses a TOML configuration file for customization. This guide covers all configuration options.

## Configuration File Location

SimTool looks for configuration in:
- `~/.config/simtool/config.toml`
- `$XDG_CONFIG_HOME/simtool/config.toml` (if XDG_CONFIG_HOME is set)

## Quick Start

Generate an example configuration:
```bash
simtool --generate-config
```

This creates a well-commented example config at `~/.config/simtool/config.example.toml`.

## Configuration Options

### Startup Settings

```toml
[startup]
# Initial view when starting SimTool
# Options: "simulator_list" (default) or "all_apps"
initial_view = "simulator_list"
```

- `simulator_list`: Start with the simulator list (default)
- `all_apps`: Start with all apps from all simulators

### Theme Configuration

```toml
[theme]
# Theme mode: "auto", "dark", or "light"
mode = "auto"

# Theme for dark mode
dark_theme = "github-dark"

# Theme for light mode  
light_theme = "github"
```

#### Theme Mode Options

- `auto`: Automatically detect terminal theme (default)
- `dark`: Always use dark theme
- `light`: Always use light theme

#### Available Themes

SimTool includes 60+ syntax highlighting themes. Popular ones include:

**Dark themes:**
- `github-dark`, `monokai`, `dracula`, `nord`, `onedark`, `solarized-dark`

**Light themes:**
- `github`, `solarized-light`, `gruvbox-light`, `tango`, `vs`

List all available themes:
```bash
simtool --list-themes
```

### Keyboard Shortcuts

All keyboard shortcuts are customizable. Each action can have multiple keys assigned.

```toml
[keys]
# Navigation
up = ["up", "k"]
down = ["down", "j"] 
left = ["left", "h"]
right = ["right", "l"]

# Quick navigation
home = ["home"]
end = ["end", "G"]

# Actions
quit = ["q", "ctrl+c"]
filter = ["f"]
search = ["/"]
escape = ["esc"]
backspace = ["backspace"]

# Simulator/App actions
boot = [" "]  # Space: boot simulator
open = [" "]  # Space: open in Finder
platform = ["p"]  # Cycle all, iOS, Android in Devices and All Apps

enter = ["enter"]
```

To disable a shortcut, set it to an empty array:
```toml
filter = []  # Disables the filter shortcut
```

`platform` affects the Devices and All Apps views. It cycles all platforms, iOS, and Android; it does not change the `--platform` command-line selection used when SimTool starts.

## Environment Variables

### Theme Override

Force a specific theme mode:
```bash
SIMTOOL_THEME_MODE=dark simtool
SIMTOOL_THEME_MODE=light simtool
```

This overrides both the config file and automatic detection.

## Example Configurations

### Minimal Config

```toml
[theme]
mode = "dark"
dark_theme = "monokai"
```

### Vim-Style Navigation Only

```toml
[keys]
up = ["k"]
down = ["j"]
left = ["h"]
right = ["l"]
home = ["g"]
end = ["G"]
quit = ["q"]
search = ["/"]
escape = ["esc"]
boot = ["space"]
open = ["o"]
```

### Start with All Apps View

```toml
[startup]
initial_view = "all_apps"

[theme]
mode = "auto"
```

## Terminal-Specific Configuration

### VS Code Terminal

VS Code's terminal doesn't support theme detection. Use:

```toml
[theme]
mode = "dark"  # or "light" based on your VS Code theme
```

### iTerm2

iTerm2 works best with:
```toml
[theme]
mode = "auto"
```

### WezTerm

WezTerm has excellent OSC support:
```toml
[theme]
mode = "auto"  # Will detect theme changes instantly
```

## Troubleshooting

### Theme not changing

1. Check if your terminal supports OSC queries
2. Try setting mode explicitly: `mode = "dark"`
3. Use environment variable: `SIMTOOL_THEME_MODE=dark`

### Keybindings not working

1. Some terminals capture certain keys (like Cmd+K)
2. Try alternative bindings
3. Check for conflicts with terminal shortcuts

### Config not loading

1. Verify file location: `simtool --show-config-path`
2. Check TOML syntax, section names, and theme names
3. Look for error messages when starting SimTool

## Configuration Best Practices

1. Start with the generated example config
2. Only include settings you want to change
3. Test changes incrementally
4. Keep backups of working configurations
5. Use version control for your config file
