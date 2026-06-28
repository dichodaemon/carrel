---
description: "Format, structure, naming, and best practices for WezTerm configuration entries."
globs:
  - 'omp/config/wezterm/*.lua'
  - 'omp/config/wezterm/*.terminfo'
---

# WezTerm Entry Type

WezTerm entries are Lua configuration files deployed into the WezTerm terminal emulator environment. They define appearance (colors, fonts, window chrome), behavior (key bindings, hyperlink rules, event handlers), and multiplexer settings.

WezTerm config is deployed by `carrel os-setup` (container) or `carrel host-setup` (host) — not by `carrel run`.

## Format

- **File extension:** `.lua` (Lua) or `.terminfo` (terminal capability database)
- **Location:** `omp/config/wezterm/<name>.lua`
- **Frontmatter:** None. WezTerm files are plain Lua scripts evaluated by the WezTerm runtime.

### Standard Entry Points

| File | Purpose |
|---|---|
| `wezterm.lua` | Main configuration file; returns a config table |
| `mux-server.lua` | Multiplexer server configuration |
| `wezterm.terminfo` | Custom terminal capability definitions |
| `overrides.lua` | Per-machine overrides (in `config/local/wezterm/`) |

WezTerm loads `wezterm.lua` from its config directory on startup. Other Lua files are required from `wezterm.lua` via `require` or `dofile`.

## Naming Conventions

- **Snake_case:** `wezterm.lua`, `mux-server.lua`, `hyperlink-rules.lua`
- **Descriptive of content:** `keybindings.lua`, `color-scheme.lua`, `tab-bar.lua`
- **`wezterm.lua` is the required entry point** — it must exist and return a config table
- **Avoid generic names:** `config.lua`, `settings.lua`

## Content Best Practices

- **Return a config table.** The main `wezterm.lua` MUST return a table. Other files MAY return tables that are merged into the main config:
  ```lua
  return {
    color_scheme = 'PaperColor Light (base16)',
    font_size = 14.0,
  }
  ```
- **Use `wez` global for API access.** WezTerm injects the `wez` module globally. Access features through it:
  ```lua
  local wez = require 'wezterm'
  ```
- **Event handlers use `wez.on()`.** Register event callbacks for `gui-startup`, `open-uri`, `window-config-reloaded`, etc.
- **Local overrides pattern.** Support per-machine overrides by merging a local file:
  ```lua
  local overrides = os.getenv('HOME') .. '/.config/carrel/overrides.lua'
  local function try_merge(path)
    local ok, result = pcall(dofile, path)
    if ok and result then
      for k, v in pairs(result) do config[k] = v end
    end
  end
  try_merge(overrides)
  ```
- **Hyperlink rules.** Use `wez.default_hyperlink_rules()` as a base and extend with `table.insert()`:
  ```lua
  local rules = wez.default_hyperlink_rules()
  table.insert(rules, {
    regex = 'https?://example\\.com/[^ ]+',
    format = '$0',
  })
  ```
- **Document sections with comments.** Each logical section (key bindings, colors, hyperlinks, events) SHOULD have a banner comment and keep lines under ~100 columns.
- **Test config reload.** WezTerm supports live config reload (default key: `CTRL+SHIFT+R`). Config changes SHOULD be safe to reload without restarting the multiplexer.
- **Keep terminal compatibility.** Custom terminfo entries SHOULD extend, not replace, standard terminal capabilities.

## Deployment Behavior

1. **Registration:** `carrel config add wezterm <name> --file=omp/config/wezterm/<name>.lua`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:**
   - Container: `carrel os-setup` deploys to the container's WezTerm config directory
   - Host: `carrel host-setup` deploys to the host's WezTerm config directory (typically `~/.config/wezterm/`)
4. **Loading:** WezTerm reads `wezterm.lua` at startup. Changes take effect on next WezTerm launch or after `CTRL+SHIFT+R` reload.

WezTerm config is NOT deployed by `carrel run`.

## Validation

Use the `validate-wezterm` skill to check:
- File has `.lua` or `.terminfo` extension
- Lua syntax is valid (`luac -p <file>` passes or equivalent checking)
- `wezterm.lua` returns a table (not nil)
- No syntax errors in event handler registrations
- Referenced local override paths use `os.getenv('HOME')` not hardcoded paths
- File is at the correct path under `omp/config/wezterm/`

## Example

```lua
-- Tab bar appearance customization for WezTerm.
local wez = require 'wezterm'

local config = {}

config.use_fancy_tab_bar = false
config.tab_bar_at_bottom = false
config.hide_tab_bar_if_only_one_tab = true

return config
```
