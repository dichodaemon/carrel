---
name: validate-wezterm
description: >
  Validate a wezterm entry's content, naming, and cross-references.
  Use before `carrel config add wezterm` or to audit existing wezterm config.
  Usage: /validate-wezterm <name> [--source=<source-alias>]
---

# Validate WezTerm

Validate a WezTerm configuration entry for structural correctness and
completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered
WezTerm config files.

## What to Validate

### Content structure

- The file is valid Lua with WezTerm API calls.
- The file returns a configuration table at the top level.
- Required WezTerm imports use `wezterm` module (e.g.,
  `local wezterm = require 'wezterm'`).
- Configuration keys use valid WezTerm option names from the
  documented API.
- Event handlers (`wezterm.on`, `wezterm.action`) use correct callback
  signatures.
- Font configuration references fonts that are installed or fall back
  gracefully.

### Naming conventions

- The entry name is kebab-case.
- The filename uses lowercase with hyphens and a `.lua` extension.
- Common names: `wezterm.lua`, `keybindings.lua`, `appearance.lua`.

### Cross-references

- Referenced external commands (e.g., `bat`, `helix`, `nvim`) are
  available in the target environment.
- Font names match installed fonts or have valid fallback chains.
- Color scheme references resolve to built-in or bundled schemes.

## How to Validate

### Pre-registration (new wezterm config)

1. Verify the file is at `omp/config/wezterm/<name>.lua` or the
   target-specific equivalent.
2. Check that the filename uses kebab-case and the `.lua` extension.
3. Parse the Lua file: confirm it returns a table (not `nil`).
4. Check that all `wezterm.on` handlers have valid event names.
5. Verify external command references are documented and available.
6. Check font fallback chains for completeness.

### Post-hoc audit (existing wezterm config)

1. Run `carrel config view wezterm <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:wezterm:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify the deployed file's Lua syntax is still valid.
5. Check that referenced external commands and fonts are still
   available.
