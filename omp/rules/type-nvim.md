---
description: >
  Format, structure, naming, and best practices for Neovim (nvim) configuration entries.
  Nvim config uses Lua and is deployed via `carrel os-setup` (container)
  or `carrel host-setup` (host). Configuration is loaded when Neovim starts.
  Use `carrel config add nvim <name> --file=omp/config/nvim/<name>.lua` to register.
  Validate with `validate-nvim` skill before registering.
globs:
  - 'omp/config/nvim/*.lua'
  - 'omp/config/nvim/*.vim'
---

# Nvim Entry Type

Nvim entries are Lua (or legacy Vimscript) configuration files deployed into the Neovim editor environment. They define plugins, key mappings, editor settings, LSP configuration, and UI appearance.

Nvim config is deployed by `carrel os-setup` (container) or `carrel host-setup` (host) — not by `carrel run`.

## Format

- **File extension:** `.lua` (preferred, modern) or `.vim` (legacy Vimscript)
- **Location:** `omp/config/nvim/<name>.lua`
- **Frontmatter:** None. Nvim files are plain Lua scripts evaluated by the Neovim Lua runtime.
- **Standard directory layout:**
  ```
  omp/config/nvim/
  ├── init.lua          # Entry point — Neovim loads this first
  ├── lua/              # Lua modules (require-able)
  │   ├── plugins.lua   # Plugin manager configuration
  │   ├── mappings.lua  # Key mappings
  │   └── lsp.lua       # LSP configuration
  └── after/            # Late-loaded overrides
  ```

### Standard Entry Points

| File | Purpose |
|---|---|
| `init.lua` | Entry point loaded by Neovim at startup; boots the configuration |
| `init.vim` | Legacy Vimscript entry point (use `init.lua` for new config) |
| `lua/*.lua` | Modular Lua configuration files, `require`-d from `init.lua` |
| `after/*.lua` | Late-loaded overrides that run after plugins are loaded |

## Naming Conventions

- **`init.lua`** is the required main entry point
- **Modular files:** `plugins.lua`, `mappings.lua`, `options.lua`, `lsp.lua`, `autocmds.lua`
- **Directory-as-namespace:** `lua/` directory maps to Lua's `require` path. `lua/config/plugins.lua` → `require('config.plugins')`
- **Snake_case module names:** `lsp_config.lua`, `treesitter_config.lua`
- **Avoid generic names:** `config.lua`, `setup.lua`

## Content Best Practices

- **`init.lua` as bootstrapper.** `init.lua` SHOULD be minimal — require modular files, don't inline all configuration:
  ```lua
  -- init.lua
  require('options')
  require('mappings')
  require('plugins')
  require('lsp')
  ```
- **Use `vim.g`, `vim.opt`, `vim.keymap` APIs.** Prefer the Lua API over `vim.cmd()` for readability and type safety:
  ```lua
  vim.opt.number = true
  vim.opt.relativenumber = true
  vim.opt.tabstop = 4
  vim.opt.shiftwidth = 4

  vim.keymap.set('n', '<C-h>', '<C-w>h', { desc = 'Move to left window' })
  ```
- **Plugin management.** Use a declarative plugin manager. `lazy.nvim` is the current standard:
  ```lua
  require('lazy').setup({
    { 'nvim-treesitter/nvim-treesitter', build = ':TSUpdate' },
    { 'neovim/nvim-lspconfig' },
    { 'hrsh7th/nvim-cmp', dependencies = { 'hrsh7th/cmp-nvim-lsp' } },
  })
  ```
- **LSP setup.** Use `nvim-lspconfig` with `on_attach` callbacks for key mappings and capabilities:
  ```lua
  local lspconfig = require('lspconfig')
  local on_attach = function(client, bufnr)
    -- buffer-local keymaps
  end
  lspconfig.rust_analyzer.setup({ on_attach = on_attach })
  ```
- **Guard against missing plugins.** Wrap plugin-specific configuration in `pcall` or check availability:
  ```lua
  local ok, _ = pcall(require, 'some-plugin')
  if not ok then return end
  ```
- **Document key mappings.** Use the `desc` option in `vim.keymap.set` so key mappings show in `:Telescope keymaps` or `:WhichKey`.
- **Modular files, not monolithic config.** Each concern (plugins, mappings, LSP, options) gets its own file. This makes the config easier to navigate and maintain.
- **Test with `nvim --startuptime`.** Profile startup time to catch slow plugins or configuration. Target <100ms startup overhead for carrel-managed config.

## Deployment Behavior

1. **Registration:** `carrel config add nvim <name> --file=omp/config/nvim/<name>.lua`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:**
   - Container: `carrel os-setup` deploys to the container's Neovim config directory
   - Host: `carrel host-setup` deploys to the host's Neovim config directory (typically `~/.config/nvim/`)
4. **Loading:** Neovim reads `init.lua` (or `init.vim`) at startup. Lua files in `lua/` are loaded on demand via `require`. Changes take effect on next Neovim launch or after `:source %` for individual files.

Nvim config is NOT deployed by `carrel run`.

## Validation

Use the `validate-nvim` skill to check:
- File has `.lua` or `.vim` extension
- Lua syntax is valid (`luac -p <file>` passes or equivalent)
- `init.lua` exists if any nvim entries are registered
- `vim.keymap.set` calls have valid mode short-names (`n`, `i`, `v`, `x`, `t`, `c`)
- No unbounded autocommands that could cause performance issues
- Plugin specs are syntactically valid
- File is at the correct path under `omp/config/nvim/`

## Example

```lua
-- omp/config/nvim/lua/options.lua
-- Editor options for Neovim.

-- Line numbers
vim.opt.number = true
vim.opt.relativenumber = true

-- Indentation
vim.opt.tabstop = 2
vim.opt.shiftwidth = 2
vim.opt.expandtab = true
vim.opt.smartindent = true

-- Search
vim.opt.ignorecase = true
vim.opt.smartcase = true
vim.opt.hlsearch = false
vim.opt.incsearch = true

-- UI
vim.opt.termguicolors = true
vim.opt.signcolumn = 'yes'
vim.opt.cursorline = true
vim.opt.scrolloff = 8

-- Undo
vim.opt.undofile = true
vim.opt.undodir = vim.fn.stdpath('data') .. '/undo'
```
