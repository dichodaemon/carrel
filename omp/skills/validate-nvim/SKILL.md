---
name: validate-nvim
description: >
  Validate a nvim entry's content, naming, and cross-references.
  Use before `carrel config add nvim` or to audit existing nvim config.
  Usage: /validate-nvim <name> [--source=<source-alias>]
---

# Validate Nvim

Validate a Neovim configuration entry for structural correctness and
completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered Neovim
config files.

## What to Validate

### Content structure

- The file is valid Lua compatible with Neovim's Lua API (or Vimscript
  if using legacy format).
- Lua files use the `vim.*` API namespace (e.g., `vim.opt`,
  `vim.keymap.set`, `vim.api.nvim_create_autocmd`).
- Plugin specifications reference valid plugin identifiers if using a
  plugin manager.
- Keybindings use valid Neovim key notation (`<C-n>`, `<leader>`,
  etc.).
- Option settings use correct Vim/Neovim option names.
- The config is structured as an `init.lua` entry point or a modular
  directory with `lua/` convention.

### Naming conventions

- The entry name is kebab-case.
- The filename uses lowercase with hyphens and a `.lua` extension.
- Common names: `init.lua`, `options.lua`, `keymaps.lua`,
  `plugins.lua`.

### Cross-references

- Referenced plugins are available or declared as dependencies.
- If the config sources other Lua files, those files exist.
- File paths reference valid directories in the Neovim config tree.
- Color scheme references match installed color schemes.

## How to Validate

### Pre-registration (new nvim config)

1. Verify the file path follows the source layout for nvim config.
2. Check that the filename uses kebab-case and the `.lua` extension.
3. Parse the Lua file: confirm it is syntactically valid (no parse
   errors on `vim.*` API calls).
4. Verify option names are valid Neovim options.
5. Check keybinding notation for correctness.
6. Validate plugin references against a known plugin registry.

### Post-hoc audit (existing nvim config)

1. Run `carrel config view nvim <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:nvim:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify the Lua syntax is still valid.
5. Check that referenced plugins, color schemes, and dependencies are
   still available.
