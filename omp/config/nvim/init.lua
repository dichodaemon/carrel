-- Bootstrap lazy.nvim
local lazypath = vim.fn.stdpath('data') .. '/lazy/lazy.nvim'
if not vim.loop.fs_stat(lazypath) then
  vim.fn.system({
    'git', 'clone', '--filter=blob:none',
    'https://github.com/folke/lazy.nvim.git',
    '--branch=stable',
    lazypath,
  })
end
vim.opt.rtp:prepend(lazypath)

-------------------------------------------------------------------------------
-- Options (from your old vimrc)
-------------------------------------------------------------------------------
vim.g.mapleader = ','

vim.opt.number        = true
vim.opt.autoindent    = true
vim.opt.tabstop       = 8
vim.opt.softtabstop   = 2
vim.opt.shiftwidth    = 2
vim.opt.expandtab     = true
vim.opt.wrap          = false
vim.opt.autoread      = true
vim.opt.incsearch     = true
vim.opt.hlsearch      = true
vim.opt.ignorecase    = true
vim.opt.smartcase     = true
vim.opt.scrolloff     = 2
vim.opt.hidden        = true
vim.opt.wildmenu      = true
vim.opt.wildmode      = { 'list:longest', 'full' }
vim.opt.list          = true
vim.opt.listchars     = { tab = '··', trail = '~', extends = '#', nbsp = '~' }
vim.opt.colorcolumn   = '81'
vim.opt.laststatus    = 3  -- global statusline (Neovim-only)
vim.opt.directory     = vim.fn.stdpath('data') .. '/swap//'
vim.opt.backspace     = { 'indent', 'eol', 'start' }

-- Reload config on save
vim.api.nvim_create_autocmd('BufWritePost', {
  pattern = vim.env.MYVIMRC,
  command = 'source <afile>',
})

-- Per-filetype column width
vim.api.nvim_create_autocmd('FileType', {
  pattern = { 'c', 'cpp' },
  callback = function() vim.opt_local.colorcolumn = '101' end,
})

-------------------------------------------------------------------------------
-- Keymaps
-------------------------------------------------------------------------------
-- Clear search highlight
vim.keymap.set('n', '<leader>/', '<cmd>nohlsearch<cr>')
-- clang-format entire file
vim.keymap.set('n', '<leader>cf', '<cmd>lua vim.lsp.buf.format()<cr>')
-- LSP go-to (replaces YCM + ctags)
vim.keymap.set('n', '<leader>gt', '<cmd>lua vim.lsp.buf.definition()<cr>')
vim.keymap.set('n', '<leader>gr', '<cmd>lua vim.lsp.buf.references()<cr>')
vim.keymap.set('n', '<leader>rn', '<cmd>lua vim.lsp.buf.rename()<cr>')
vim.keymap.set('n', 'K',          '<cmd>lua vim.lsp.buf.hover()<cr>')

-------------------------------------------------------------------------------
-- Plugins
-------------------------------------------------------------------------------
require('lazy').setup({

  -- Treesitter
  {
    'nvim-treesitter/nvim-treesitter',
    build = ':TSUpdate',
    config = function()
      require('nvim-treesitter').setup()
    end,
  },

  -- Colorscheme
  {
    'NLKNguyen/papercolor-theme',
    lazy = false,
    priority = 1000,
    config = function()
      vim.opt.termguicolors = true
      vim.opt.background = 'light'
      vim.cmd.colorscheme('PaperColor')
      -- vim.cmd.colorscheme('Github')
      vim.api.nvim_set_hl(0, 'LineNr', { fg = '#ffffff', bg = '#777777', bold = true })
    end,
  },

  -- Statusline
  {
    'nvim-lualine/lualine.nvim',
    dependencies = { 'nvim-tree/nvim-web-devicons' },
    opts = {
      options = {
        theme = 'papercolor_light',
        globalstatus = true,
      },
      sections = {
        lualine_c = { { 'filename', path = 1 } },  -- relative path
      },
    },
  },

  -- Fuzzy finder (replaces Unite + command-t)
  {
    'nvim-telescope/telescope.nvim',
    dependencies = { 'nvim-lua/plenary.nvim' },
    keys = {
      { '<leader>t',  '<cmd>Telescope find_files<cr>' },
      { '<space>/',   '<cmd>Telescope live_grep<cr>' },
      { '<space>o',   '<cmd>Telescope lsp_document_symbols<cr>' },
      { '<leader>lj', '<cmd>Telescope buffers<cr>' },
      { '<space>qf',  '<cmd>Telescope quickfix<cr>' },
      { '<space>ll',  '<cmd>Telescope loclist<cr>' },
    },
  },

  -- Git (keep fugitive)
  { 'tpope/vim-fugitive' },
  { 'tpope/vim-unimpaired' },

  -- Commenting (replaces NERDCommenter)
  {
    'numToStr/Comment.nvim',
    opts = {},
  },

  -- LSP + Mason (replaces YCM + ctags)
  {
    'williamboman/mason.nvim',
    opts = {},
  },
  {
    'williamboman/mason-lspconfig.nvim',
    dependencies = { 'williamboman/mason.nvim', 'neovim/nvim-lspconfig' },
    opts = {
      ensure_installed = { 'clangd', 'pyright' },
      automatic_installation = true,
    },
  },
  {
    'neovim/nvim-lspconfig',
    dependencies = { 'williamboman/mason-lspconfig.nvim' },
    config = function()
      vim.lsp.config('clangd', {})
      vim.lsp.config('pyright', {})
      vim.lsp.enable({ 'clangd', 'pyright' })
    end,
  },

  -- Completion (replaces YCM completion UI)
  {
    'hrsh7th/nvim-cmp',
    dependencies = {
      'hrsh7th/cmp-nvim-lsp',
      'hrsh7th/cmp-buffer',
      'hrsh7th/cmp-path',
      'L3MON4D3/LuaSnip',
      'saadparwaiz1/cmp_luasnip',
    },
    config = function()
      local cmp = require('cmp')
      cmp.setup({
        snippet = {
          expand = function(args) require('luasnip').lsp_expand(args.body) end,
        },
        completion = {
          -- Don't pop up automatically; require explicit <C-Space> to trigger.
          autocomplete = false,
        },
        mapping = cmp.mapping.preset.insert({
          ['<C-Space>'] = cmp.mapping.complete(),
          -- Only confirm when an item was explicitly selected (select = false).
          -- If nothing is selected, <CR> falls through to a normal newline.
          ['<CR>']      = cmp.mapping.confirm({ select = false }),
          ['<Tab>']     = cmp.mapping.select_next_item(),
          ['<S-Tab>']   = cmp.mapping.select_prev_item(),
        }),
        sources = cmp.config.sources({
          { name = 'nvim_lsp' },
          { name = 'luasnip' },
          { name = 'buffer' },
          { name = 'path' },
        }),
      })
    end,
  },

  -- Formatter (replaces %!clang-format)
  {
    'stevearc/conform.nvim',
    opts = {
      formatters_by_ft = {
        c   = { 'clang_format' },
        cpp = { 'clang_format' },
        python = { 'black' },
      },
      format_on_save = { timeout_ms = 500, lsp_fallback = true },
    },
  },

  -- Linting (replaces vim-cpplint + vim-flake8)
  {
    'mfussenegger/nvim-lint',
    config = function()
      require('lint').linters_by_ft = {
        cpp    = { 'cpplint' },
        python = { 'flake8' },
      }
      vim.api.nvim_create_autocmd({ 'BufWritePost' }, {
        callback = function() require('lint').try_lint() end,
      })
    end,
  },

})
