---
description: >
  Format, structure, naming, and best practices for Helix editor configuration entries.
  Helix config uses TOML and is deployed via `carrel os-setup` (container)
  or `carrel host-setup` (host). Configuration is loaded when Helix starts.
  Use `carrel config add helix <name> --file=omp/config/helix/<name>.toml` to register.
  Validate with `validate-helix` skill before registering.
globs:
  - 'omp/config/helix/*.toml'
---

# Helix Entry Type

Helix entries are TOML configuration files deployed into the Helix editor environment. They define theme, key bindings, language server settings, editor behavior, and file-type associations.

Helix config is deployed by `carrel os-setup` (container) or `carrel host-setup` (host) — not by `carrel run`.

## Format

- **File extension:** `.toml`
- **Location:** `omp/config/helix/<name>.toml`
- **Frontmatter:** None. Helix files are plain TOML evaluated by the Helix runtime.
- **Main entry point:** `omp/config/helix/config.toml` — Helix loads this file from its config directory

### Standard Configuration Sections

| TOML Section | Purpose |
|---|---|
| `theme` | Color theme name (top-level string) |
| `[editor]` | Editor behavior: cursor shape, line numbers, indentation, rulers |
| `[editor.cursor-shape]` | Cursor appearance per mode |
| `[editor.indent-guides]` | Indentation guide rendering |
| `[editor.statusline]` | Status line layout |
| `[editor.lsp]` | Language server display settings |
| `[editor.whitespace]` | Whitespace character rendering |
| `[keys.normal]` | Key bindings for normal mode |
| `[keys.insert]` | Key bindings for insert mode |
| `[keys.select]` | Key bindings for select mode |

## Naming Conventions

- **`config.toml`** is the required main entry point — Helix loads this from its config directory
- **Additional modular files:** `languages.toml` (language server configs), `themes.toml` (custom themes)
- **Avoid generic names:** `settings.toml`, `editor.toml`

## Content Best Practices

- **Start with `theme`.** The color theme is the top-level key:
  ```toml
  theme = "papercolor-light"
  ```
- **Key bindings are mode-specific.** Bindings go under the appropriate mode section. Use Helix's key notation:
  ```toml
  [keys.normal]
  C-left  = "move_prev_word_start"
  C-right = "move_next_word_end"

  [keys.select]
  C-left  = "extend_prev_word_start"
  C-right = "extend_next_word_end"
  ```
- **Use Helix command names.** Key bindings map to Helix commands, not arbitrary strings. Run `:help` in Helix or check the documentation for the command reference.
- **Language servers.** Configure LSP settings under `[editor.lsp]` for display, and language-specific settings under `[[language]]` arrays (in `languages.toml`):
  ```toml
  [[language]]
  name = "rust"
  language-servers = ["rust-analyzer"]
  ```
- **Indentation.** Configure per-filetype indentation in `languages.toml`:
  ```toml
  [[language]]
  name = "python"
  indent = { tab-width = 4, unit = "    " }
  ```
- **Keep files modular.** Don't put language server configs in `config.toml`. Use `languages.toml` for LSP settings and keep `config.toml` for editor behavior.
- **Minimal configuration.** Helix has sensible defaults. Only override what you need — resist the urge to configure everything.
- **Test with `hx --health`.** After changes, run `hx --health <language>` to verify language server integration.

## Deployment Behavior

1. **Registration:** `carrel config add helix <name> --file=omp/config/helix/<name>.toml`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:**
   - Container: `carrel os-setup` deploys to the container's Helix config directory
   - Host: `carrel host-setup` deploys to the host's Helix config directory (typically `~/.config/helix/`)
4. **Loading:** Helix reads `config.toml` at startup. Changes take effect on next Helix launch.

Helix config is NOT deployed by `carrel run`.

## Validation

Use the `validate-helix` skill to check:
- File has `.toml` extension
- TOML syntax is valid (parses without errors)
- Key bindings use valid Helix key notation
- Command names are recognized Helix commands
- `config.toml` top-level `theme` is a string (not a table)
- No duplicate key bindings in the same mode section
- File is at the correct path under `omp/config/helix/`

## Example

```toml
# Helix editor configuration
theme = "papercolor-light"

[editor]
line-number = "relative"
mouse = false
auto-format = true
bufferline = "multiple"

[editor.cursor-shape]
normal = "block"
insert = "bar"
select = "underline"

[editor.indent-guides]
render = true
character = "▏"

[keys.normal]
C-left  = "move_prev_word_start"
C-right = "move_next_word_end"
A-left  = "jump_backward"
A-right = "jump_forward"

[keys.select]
C-left  = "extend_prev_word_start"
C-right = "extend_next_word_end"
```
