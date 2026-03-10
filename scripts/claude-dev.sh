#!/bin/bash
# Start a tmux session with Claude Code and a Go test watcher
# for claudeup development.
#
# Layout:
#   +----------------------------+----------------+
#   |                            | Go test watcher|
#   |   Claude Code              | (entr)         |
#   |                            |                |
#   +----------------------------+----------------+

set -euo pipefail

SESSION="claudeup"
DIR="$HOME/code/claudeup"

if tmux has-session -t "$SESSION" 2>/dev/null; then
    echo "Session '$SESSION' already exists. Attaching..."
    exec tmux attach -t "$SESSION"
fi

# -P -F '#{pane_id}' returns the ID of each pane as it's created,
# so send-keys targets the right pane regardless of base-index config.
CLAUDE=$(tmux new-session -d -s "$SESSION" -c "$DIR" -P -F '#{pane_id}')
tmux send-keys -t "$CLAUDE" "claude --dangerously-skip-permissions -w" Enter

# Go test watcher (right side, 35% width)
TESTS=$(tmux split-window -h -t "$CLAUDE" -c "$DIR" -p 35 -P -F '#{pane_id}')
tmux send-keys -t "$TESTS" \
    "find cmd internal test -name '*.go' | entr -c go test ./internal/..." Enter

# Focus Claude pane
tmux select-pane -t "$CLAUDE"

tmux attach -t "$SESSION"
