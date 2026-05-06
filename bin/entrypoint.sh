#!/usr/bin/env bash
set -e

# Link OMP persistent state to workspace-mounted carrel data
if [ -d /workspace/.carrel/local/omp ]; then
    ln -sfn /workspace/.carrel/local/omp /home/dev/.omp
fi

# Bootstrap registry if needed (must run before os-setup)
carrel bootstrap 2>/dev/null || true

# Run carrel OS setup (compose + deploy OS config)
carrel os-setup || true

# omp source checkout override (development)
OMP_SOURCE="/workspace/oh-my-pi/packages/coding-agent/src/cli.ts"
if [[ -f "$OMP_SOURCE" ]]; then
    mkdir -p "$HOME/.local/bin"
    ln -sf "$OMP_SOURCE" "$HOME/.local/bin/omp"
else
    rm -f "$HOME/.local/bin/omp"
fi

# Exec the main command
exec "$@"
