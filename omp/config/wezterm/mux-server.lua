-- Carula mux server configuration.
-- Used by wezterm-mux-server inside the container. Rendering (fonts,
-- colors, keybindings) is handled by the client; this file only
-- configures server-side behavior.

local wez = require 'wezterm'

return {
  default_prog = { '/usr/bin/zsh' },
  default_cwd = '/workspace',
}
