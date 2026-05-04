#!/usr/bin/env bash
set -e

# Link user config overrides from workspace mount
if [ -d /workspace/config/local/zsh ]; then
    ln -sf /workspace/config/local/zsh/zshrc.local /home/dev/.zshrc.local 2>/dev/null || true
fi

# Run carrel OS setup
carrel os-setup

# Bootstrap registry if needed
carrel bootstrap 2>/dev/null || true

# Exec the main command
exec "$@"
