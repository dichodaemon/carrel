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

# Exec the main command
exec "$@"
