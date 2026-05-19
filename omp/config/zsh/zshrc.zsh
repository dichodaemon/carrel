# Carula default zshrc.
# Sourced inside the container. User extensions go in ~/.zshrc.local
# (e.g., mount from host or create inside the container).

# Powerlevel10k instant prompt (must stay near the top)
# ---------------------------------------------------------------------------
if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

# ---------------------------------------------------------------------------
# Theme
source /opt/powerlevel10k/powerlevel10k.zsh-theme

# Plugins — autosuggestions
source /usr/share/zsh-autosuggestions/zsh-autosuggestions.zsh

# ---------------------------------------------------------------------------
# Shell options
# ---------------------------------------------------------------------------
setopt histignorealldups sharehistory
bindkey -e

# Key bindings not covered by /etc/zsh/zshrc (which only binds unmodified keys
# via terminfo).  These are the modifier+key sequences sent by WezTerm/xterm.
bindkey "^[[1;5D" backward-word      # Ctrl+Left
bindkey "^[[1;5C" forward-word       # Ctrl+Right
bindkey "^[[1;3D" backward-word      # Alt+Left
bindkey "^[[1;3C" forward-word       # Alt+Right
bindkey "^[[3;5~" kill-word          # Ctrl+Delete
bindkey "^[[3~"   delete-char        # Delete

HISTSIZE=10000000
SAVEHIST=10000000
HISTFILE=~/.zsh_history

# ---------------------------------------------------------------------------
# Environment
# ---------------------------------------------------------------------------
export PATH="$HOME/.local/bin:/workspace/carrel/bin:/workspace/carrel/tools:/usr/local/bin:$PATH"
export MANPAGER="sh -c 'col -bx | bat -l man -p'"

export PI_NO_APPEARANCE_POLL=1
# bun
if [[ -d "$HOME/.bun" ]]; then
  export BUN_INSTALL="$HOME/.bun"
  export PATH="$BUN_INSTALL/bin:$PATH"
fi

# bun completions
[[ -s "$HOME/.bun/_bun" ]] && source "$HOME/.bun/_bun"

# nvm
if [[ -d "$HOME/.nvm" ]]; then
  export NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
fi

# ---------------------------------------------------------------------------
# Prompt customization (p10k)
# ---------------------------------------------------------------------------
[[ ! -f ~/.p10k.zsh ]] || source ~/.p10k.zsh

# Syntax-highlighting loaded after p10k prompt customization
# to ensure terminal state is fully settled before its hook activates.
source /usr/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh

# ---------------------------------------------------------------------------
# User extensions (not tracked by carrel)
# ---------------------------------------------------------------------------
[[ -f ~/.zshrc.local ]] && source ~/.zshrc.local
