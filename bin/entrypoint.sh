#!/usr/bin/env bash
set -e

# Link OMP persistent state to workspace-mounted carrel data
if [ -d /workspace/.carrel/local/omp ]; then
    ln -sfn /workspace/.carrel/local/omp /home/dev/.omp
fi

# Run carrel OS setup
carrel os-setup

# Bootstrap registry if needed
carrel bootstrap 2>/dev/null || true

# Exec the main command
exec "$@"
