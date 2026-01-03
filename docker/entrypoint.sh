#!/bin/bash
# ABOUTME: Entrypoint for claudeup sandbox container
# ABOUTME: Syncs plugins before launching Claude or shell

# Sync plugins if .claudeup.json exists in workspace
if [ -f /workspace/.claudeup.json ]; then
    echo "Syncing plugins..."
    claudeup profile sync 2>&1 || echo "Warning: Plugin sync failed"
fi

# Execute the requested command
if [ "$1" = "bash" ] || [ "$1" = "shell" ]; then
    exec bash
else
    exec claude "$@"
fi
