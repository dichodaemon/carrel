local wez = require 'wezterm'
local mux = wez.mux
local color_scheme_name = 'PaperColor Light (base16)'
-- local color_scheme_name = 'Github (base16)'
local color_scheme =  wez.color.get_builtin_schemes()[color_scheme_name]
local new_brights = color_scheme.brights
new_brights[1] = '#555555'
new_brights[8] = '#ffffff'

-------------------------------------------------------------------------------
-- Hyperlink rules
-------------------------------------------------------------------------------
local hyperlink_rules = wez.default_hyperlink_rules()

-- .md files → markless
table.insert(hyperlink_rules, {
  regex = [=[([^\s"'<>]+\.md)(:\d+)?]=],
  format = 'markless://$0',
})

-- bare www. URLs (WezTerm defaults only match https?://, not bare www. domains)
table.insert(hyperlink_rules, {
  regex = [=[(www\.[^\s"'<>]+)]=],
  format = 'https://$1',
})

-- code files → bat viewer (CTRL+click) or helix editor (CTRL+ALT+click)
-- Requires '/' in the matched token: this prunes commit hashes, tags, and
-- other dot-dense non-path tokens before the extension alternation is tried,
-- keeping the DFA state space low on git push/commit output.
table.insert(hyperlink_rules, {
  regex = [=[([^\s"'<>]*/[^\s"'<>]*\.(R|Rmd|S|asm|bash|bat|bazel|bzl|c|capnp|cc|cfg|cjs|cl|clj|cljc|cljs|cmake|cmd|comp|conf|containerfile|cpp|cr|cs|css|csv|csx|cts|cu|cue|cuh|cxx|dart|db|desktop|dhall|diff|dockerfile|dot|dsql|edn|eex|elm|env|erb|erl|ex|exs|f|f03|f90|f95|feature|fish|for|frag|fs|fsi|fsscript|fsx|gemspec|geom|gleam|glsl|go|gql|gradle|graphql|groovy|h|h++|hcl|heex|hh|hpp|hrl|hs|htm|html|hxx|inc|ini|inl|ipy|jav|java|jinja|jinja2|jl|js|json|json5|jsonc|jsonl|jsonnet|jsx|just|ksh|kt|kts|less|lhs|libsonnet|log|lua|luau|mak|make|markdown|md|mdx|mjs|mk|ml|mli|mount|mts|nim|nimble|nims|nix|njk|odin|org|pas|patch|php|php3|php4|php5|phtml|pkl|pl|ply|pm|pp|prisma|proto|prql|ps1|psd1|psm1|pug|pxd|pxi|py|py3|pyi|pyt|pyw|pyx|r|rake|rb|rbi|regex|rmd|robot|rs|rst|ru|s|sass|sbt|sc|scala|scss|service|sh|shtml|smali|smithy|socket|sol|sql|sqlite3|sv|svg|svh|swift|target|task|tcl|tesc|tese|tex|tf|tfvars|thrift|timer|toml|ts|tsx|twig|txt|typ|v|vala|vert|vhd|vhdl|vim|vsh|vv|wgsl|xaml|xhtml|xml|yaml|yara|yml|zig|zon|zsh))(:\d+)?(:\d+)?]=],
  format = 'bat://$0',
})

-------------------------------------------------------------------------------
-- Notification helper — uses tab-bar right status (works without DBUS)
-------------------------------------------------------------------------------
local function notify(window, message, timeout_s)
  window:set_right_status('  ' .. message .. '  ')
  wez.time.call_after(timeout_s or 3, function()
    window:set_right_status('')
  end)
end

-- Per-tab pane caches: tab_id → pane_id
-- Persist for the lifetime of the WezTerm process (reset only on config reload).
local markless_pane_by_tab = {}
local bat_pane_by_tab = {}

-- Flag: next bat:// open-uri should go to helix instead (set by CTRL+ALT+click)
local open_in_editor = false
wez.on('open-uri-editor-mode', function() open_in_editor = true end)

-------------------------------------------------------------------------------
-- Event: open-uri
-------------------------------------------------------------------------------
wez.on('open-uri', function(window, pane, uri)
  -- Only intercept our custom schemes; open everything else in the system browser
  if not (uri:match('^bat://') or uri:match('^markless://')) then
    wez.open_with(uri)
    return true
  end

  -- Capture and reset the editor-mode flag before any async work
  local editor_mode = open_in_editor
  open_in_editor = false

  local ok, err = pcall(function()

    -- bat:// — view in bat (CTRL+click) or edit in helix (CTRL+ALT+click)
    if uri:match('^bat://') then
      local path = uri:gsub('^bat://', '')

      -- Parse :line:col or :line suffix
      local filepath, line = path:match('^(.+):(%d+):%d+$')
      if not filepath then
        filepath, line = path:match('^(.+):(%d+)$')
      end
      if not filepath then
        filepath = path
      end

      -- If the path looks like a URL (e.g. our regex matched https://foo/bar.js),
      -- fall through to the system browser regardless of mode.
      if filepath:match('^https?://') or filepath:match('^ftp://') then
        wez.open_with(filepath)
        return
      end

      if editor_mode then
        -- CTRL+ALT+click: open in helix pane (reuse or split)
        local hx_pane
        for _, p in ipairs(pane:tab():panes()) do
          local fg = (p:get_foreground_process_name() or ''):lower()
          local title = (p:get_title() or ''):lower()
          if fg:match('hx') or fg:match('helix') or title:match('hx') or title:match('helix') then
            hx_pane = p
            break
          end
        end

        local hx_cmd = line
          and ('hx ' .. filepath .. ':' .. line .. '\r')
          or  ('hx ' .. filepath .. '\r')

        if hx_pane then
          -- ESC first (exit insert mode), then :open after a short delay.
          -- Sending them together trips bracketed-paste, which eats the 'o' in ':open'.
          hx_pane:send_text('\x1b')
          wez.time.call_after(0.05, function()
            hx_pane:send_text(':open ' .. filepath .. (line and (':' .. line) or '') .. '\r')
          end)
          hx_pane:activate()
        else
          -- No helix pane: split a new one vertically and start hx
          window:perform_action(
            wez.action.SplitPane {
              direction = 'Right',
              size = { Percent = 40 },
              command = { domain = 'CurrentPaneDomain' },
            },
            pane
          )
          wez.time.call_after(0.1, function()
            window:active_pane():send_text(hx_cmd)
          end)
        end
        return
      end

      -- CTRL+click: open in bat viewer pane (reuse or split)
      local tab_id = pane:tab():tab_id()
      local bat_pane
      local cached_id = bat_pane_by_tab[tab_id]
      if cached_id then
        for _, p in ipairs(pane:tab():panes()) do
          if p:pane_id() == cached_id then bat_pane = p; break end
        end
        if not bat_pane then bat_pane_by_tab[tab_id] = nil end
      end

      local bat_cmd
      if line then
        bat_cmd = "bat --pager 'less -R +" .. line .. "' --highlight-line " .. line .. ' ' .. filepath .. '\r'
      else
        bat_cmd = 'bat --paging always ' .. filepath .. '\r'
      end

      if bat_pane then
        bat_pane:send_text('q')
        bat_pane:activate()
        wez.time.call_after(0.2, function()
          bat_pane:send_text(bat_cmd)
        end)
      else
        window:perform_action(
          wez.action.SplitPane {
            direction = 'Right',
            size = { Percent = 40 },
            command = { domain = 'CurrentPaneDomain' },
          },
          pane
        )
        wez.time.call_after(0.1, function()
          local new_pane = window:active_pane()
          bat_pane_by_tab[tab_id] = new_pane:pane_id()
          new_pane:send_text(bat_cmd)
        end)
      end
    end

    -- markless:// — open .md file in markless pane (reuse or split)
    if uri:match('^markless://') then
      local filepath = uri:gsub('^markless://', '')

      local tab_id = pane:tab():tab_id()

      -- Find the markless pane we previously created for this tab.
      -- get_foreground_process_name() is empty for SSH-domain panes, so we
      -- track pane IDs ourselves in markless_pane_by_tab.
      local markless_pane
      local cached_id = markless_pane_by_tab[tab_id]
      if cached_id then
        for _, p in ipairs(pane:tab():panes()) do
          if p:pane_id() == cached_id then
            markless_pane = p
            break
          end
        end
        if not markless_pane then
          -- Pane was closed; clear stale entry
          markless_pane_by_tab[tab_id] = nil
        end
      end
      if markless_pane then
        -- Send 'q' alone first; if we concatenate the next command immediately,
        -- the 'e' in 'markless' arrives while markless is still in raw mode and
        -- triggers editor mode. Wait 300ms for the process to fully exit.
        markless_pane:send_text('q')
        markless_pane:activate()
        wez.time.call_after(0.3, function()
          markless_pane:send_text('markless --image-mode kitty ' .. filepath .. '\r')
        end)
      else
        -- Split a shell in the same domain, then send the markless command
        window:perform_action(
          wez.action.SplitPane {
            direction = 'Right',
            size = { Percent = 40 },
            command = { domain = 'CurrentPaneDomain' },
          },
          pane
        )
        wez.time.call_after(0.1, function()
          local new_pane = window:active_pane()
          markless_pane_by_tab[tab_id] = new_pane:pane_id()
          new_pane:send_text('markless --image-mode kitty ' .. filepath .. '\r')
        end)
      end
    end

  end)

  if not ok then
    notify(window, 'WezTerm error: ' .. tostring(err), 4)
  end

  return true  -- always suppress default handling for our schemes
end)

-------------------------------------------------------------------------------
-- gui-startup
-------------------------------------------------------------------------------
wez.on('gui-startup', function()
  -- Always spawn the mux daemon. wezterm start --daemonize is idempotent:
  -- it connects to an existing healthy mux, and replaces a frozen one.
  -- The socket-existence check added in 596931a was unnecessary and caused
  -- a startup deadlock when probing the mux from inside its own event handler.
  wez.background_child_process{ 'wezterm', 'start', '--daemonize' }
end)

local config = {
  default_domain = 'unix',
  -- Font
  font = wez.font 'JetBrains Mono',
  font_size = 14.0,

  -- Window
  window_padding = { left = 12, right = 12, top = 8, bottom = 8 },
  initial_cols = 220,
  initial_rows = 50,

  -- Tab bar
  hide_tab_bar_if_only_one_tab = true,

  -- Color scheme
  color_scheme = color_scheme_name,

  colors = {
    brights = new_brights,
  },

  -- Hyperlink rules
  hyperlink_rules = hyperlink_rules,

  -- Multiplexer keybinds
  keys = {
    -- Pass CTRL+T through to the shell (fzf file finder)
    { key = 't', mods = 'CTRL', action = wez.action.SendKey { key = 't', mods = 'CTRL' } },

    -- Splits
    { key = 'd', mods = 'CTRL|SHIFT', action = wez.action.SplitHorizontal { domain = 'CurrentPaneDomain' } },
    { key = 'e', mods = 'CTRL|SHIFT', action = wez.action.SplitVertical   { domain = 'CurrentPaneDomain' } },

    -- Navigate panes
    { key = 'LeftArrow',  mods = 'CTRL|SHIFT', action = wez.action.ActivatePaneDirection 'Left' },
    { key = 'RightArrow', mods = 'CTRL|SHIFT', action = wez.action.ActivatePaneDirection 'Right' },
    { key = 'UpArrow',    mods = 'CTRL|SHIFT', action = wez.action.ActivatePaneDirection 'Up' },
    { key = 'DownArrow',  mods = 'CTRL|SHIFT', action = wez.action.ActivatePaneDirection 'Down' },

    -- Maximize/zoom pane (toggle)
    { key = 'm', mods = 'CTRL|SHIFT', action = wez.action.TogglePaneZoomState },

    -- Close current pane
    { key = 'q', mods = 'CTRL|SHIFT', action = wez.action.CloseCurrentPane { confirm = false } },

    -- Resize panes
    { key = 'LeftArrow',  mods = 'ALT|SHIFT', action = wez.action.AdjustPaneSize { 'Left',  5 } },
    { key = 'RightArrow', mods = 'ALT|SHIFT', action = wez.action.AdjustPaneSize { 'Right', 5 } },
    { key = 'UpArrow',    mods = 'ALT|SHIFT', action = wez.action.AdjustPaneSize { 'Up',    5 } },
    { key = 'DownArrow',  mods = 'ALT|SHIFT', action = wez.action.AdjustPaneSize { 'Down',  5 } },

    -- Tabs
    { key = 't', mods = 'CTRL|SHIFT', action = wez.action.SpawnTab 'CurrentPaneDomain' },
    -- (CTRL+T is passed through to the shell for fzf)
    { key = 'w', mods = 'CTRL|SHIFT', action = wez.action.CloseCurrentTab { confirm = false } },
    { key = 'Tab', mods = 'CTRL',     action = wez.action.ActivateTabRelative(1) },
    { key = 'Tab', mods = 'CTRL|SHIFT', action = wez.action.ActivateTabRelative(-1) },
  },

  mouse_bindings = {
    -- CTRL+click → bat viewer
    {
      event = { Up = { streak = 1, button = 'Left' } },
      mods = 'CTRL',
      action = wez.action.OpenLinkAtMouseCursor,
    },
    -- CTRL+ALT+click → helix editor
    -- Down sets the flag; Up fires OpenLinkAtMouseCursor (open-uri reads the flag)
    {
      event = { Down = { streak = 1, button = 'Left' } },
      mods = 'CTRL|ALT',
      action = wez.action.EmitEvent 'open-uri-editor-mode',
    },
    {
      event = { Up = { streak = 1, button = 'Left' } },
      mods = 'CTRL|ALT',
      action = wez.action.OpenLinkAtMouseCursor,
    },
  },

  term = 'wezterm',
  enable_kitty_keyboard = false,
  enable_kitty_graphics = true,

  unix_domains = {
    { name = 'unix' },
    -- {
    --   name = 'carrel',
    --   proxy_command = { 'ssh', '-T', '<user>@<host-address>', '/home/<user>/code/carrel/bin/proxy' },
    -- },
  },
  set_environment_variables = {
    EDITOR = 'hx',
  },
}

-------------------------------------------------------------------------------
-- Local overrides: machine-specific secrets (IPs, ssh_domains, usernames).
-- Place overrides in config/local/wezterm/overrides.lua (gitignored).
-- The file should return a table; its keys are shallow-merged into config.
-------------------------------------------------------------------------------
local overrides_path = os.getenv('HOME') .. '/.config/carrel/overrides.lua'
-- Also check the sibling local/ tree relative to this config file's directory.
-- Inside the container config_dir is /workspace/carrel/config/wezterm; on a
-- laptop it is wherever the symlink lives (e.g. ~/.config/wezterm).
local sibling_overrides = wez.config_dir .. '/../local/wezterm/overrides.lua'

local function try_merge(path)
  local ok, result = pcall(dofile, path)
  if ok and type(result) == 'table' then
    for k, v in pairs(result) do
      config[k] = v
    end
  end
end

try_merge(overrides_path)
try_merge(sibling_overrides)

return config
