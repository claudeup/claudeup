#!/bin/bash
set -euo pipefail
# ABOUTME: Entrypoint for claudeup sandbox container
# ABOUTME: Syncs plugins before launching Claude or shell

# Sync plugins if .claudeup.json exists in workspace
if [ -f /workspace/.claudeup.json ]; then
    echo "Syncing plugins..."
    if ! claudeup profile sync; then
        echo "Warning: Plugin sync failed" >&2
    fi
fi

# Execute the requested command
# Use parameter expansion to handle case where no arguments provided
if [ "${1:-}" = "bash" ] || [ "${1:-}" = "shell" ]; then
    exec bash
else
    exec claude "$@"
fi
