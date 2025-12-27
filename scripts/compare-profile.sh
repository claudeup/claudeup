#!/usr/bin/env bash
# ABOUTME: Compare currently installed Claude plugins against a saved claudeup profile
# ABOUTME: Shows differences between saved profile and effective configuration (all scopes combined)
#
# NOTE: As of v0.x, claudeup uses combined scope comparison (user + project + local).
# This script provides a simplified view comparing just the enabled plugins.
# For the full picture including scope layering, use: claudeup profile list

set -euo pipefail

# Configuration
CLAUDEUP_BIN="${CLAUDEUP_BIN:-./bin/claudeup}"
CLAUDE_PLUGINS_FILE="${HOME}/.claude/plugins/installed_plugins.json"

# Auto-detect active profile if not specified
if [[ -z "${1:-}" ]]; then
    echo "🔍 Auto-detecting active profile..."
    PROFILE_NAME=$("$CLAUDEUP_BIN" profile list 2>/dev/null | grep '^\*' | awk '{print $2}')
    if [[ -z "$PROFILE_NAME" ]]; then
        echo "Error: No active profile found" >&2
        echo "Hint: Run 'claudeup profile list' or specify a profile name" >&2
        exit 1
    fi
    echo "📌 Active profile: $PROFILE_NAME"
    echo
else
    PROFILE_NAME="$1"
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

usage() {
    cat <<EOF
Usage: $(basename "$0") [profile-name]

Compare currently installed Claude plugins against a saved claudeup profile.

Arguments:
  profile-name    Name of the profile to compare against (optional)
                  If not specified, uses the currently active profile

Examples:
  $(basename "$0")              # Compare against active profile
  $(basename "$0") base-tools   # Compare against specific profile
  $(basename "$0") my-work-setup

Environment:
  CLAUDEUP_BIN    Path to claudeup binary (default: ./bin/claudeup)
EOF
    exit 1
}

# Show help if requested
if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
fi

# Check if Claude is installed
if [[ ! -f "$CLAUDE_PLUGINS_FILE" ]]; then
    echo "Error: Claude plugins file not found at: $CLAUDE_PLUGINS_FILE" >&2
    echo "Is Claude Code installed?" >&2
    exit 1
fi

# Check if claudeup binary exists
if [[ ! -x "$CLAUDEUP_BIN" ]]; then
    echo "Error: claudeup binary not found or not executable: $CLAUDEUP_BIN" >&2
    exit 1
fi

# Create temp files
CURRENT_PLUGINS=$(mktemp)
SAVED_PLUGINS=$(mktemp)
trap 'rm -f "$CURRENT_PLUGINS" "$SAVED_PLUGINS"' EXIT

# Extract currently enabled plugins from all scopes (user + project + local)
# This mimics how Claude Code accumulates settings across scopes
echo "📋 Reading effective plugin configuration (user + project + local)..."

# Start with user-scope enabled plugins
jq -r '.enabledPlugins | to_entries[] | select(.value == true) | .key' \
    "${HOME}/.claude/settings.json" 2>/dev/null | sort > "$CURRENT_PLUGINS" || touch "$CURRENT_PLUGINS"

# Add project-scope enabled plugins (if in a project directory with .claude/settings.json)
if [[ -f "./.claude/settings.json" ]]; then
    jq -r '.enabledPlugins | to_entries[] | select(.value == true) | .key' \
        "./.claude/settings.json" 2>/dev/null >> "$CURRENT_PLUGINS" || true
fi

# Add local-scope enabled plugins (if in a project directory with .claude/settings.local.json)
if [[ -f "./.claude/settings.local.json" ]]; then
    jq -r '.enabledPlugins | to_entries[] | select(.value == true) | .key' \
        "./.claude/settings.local.json" 2>/dev/null >> "$CURRENT_PLUGINS" || true
fi

# Sort and deduplicate
sort -u "$CURRENT_PLUGINS" -o "$CURRENT_PLUGINS"
CURRENT_COUNT=$(wc -l < "$CURRENT_PLUGINS" | xargs)

# Extract plugins from saved profile
echo "📋 Reading saved profile: $PROFILE_NAME..."
if ! "$CLAUDEUP_BIN" profile show "$PROFILE_NAME" > /dev/null 2>&1; then
    echo "Error: Profile '$PROFILE_NAME' not found" >&2
    exit 1
fi

"$CLAUDEUP_BIN" profile show "$PROFILE_NAME" 2>/dev/null | \
    grep -A 200 "Plugins:" | \
    grep "  - " | \
    sed 's/  - //' | \
    sort > "$SAVED_PLUGINS"
SAVED_COUNT=$(wc -l < "$SAVED_PLUGINS" | xargs)

# Calculate differences
ADDED=$(comm -23 "$CURRENT_PLUGINS" "$SAVED_PLUGINS")
REMOVED=$(comm -13 "$CURRENT_PLUGINS" "$SAVED_PLUGINS")

# Count non-empty lines
if [[ -n "$ADDED" ]]; then
    ADDED_COUNT=$(echo "$ADDED" | wc -l | xargs)
else
    ADDED_COUNT=0
fi

if [[ -n "$REMOVED" ]]; then
    REMOVED_COUNT=$(echo "$REMOVED" | wc -l | xargs)
else
    REMOVED_COUNT=0
fi

NET_CHANGE=$((CURRENT_COUNT - SAVED_COUNT))

# Display results
echo
echo -e "${BOLD}=== Changes from $PROFILE_NAME ===${NC}"
echo

if [[ $ADDED_COUNT -gt 0 ]]; then
    echo -e "${GREEN}${BOLD}+ Added on top of $PROFILE_NAME: $ADDED_COUNT plugins${NC}"
    echo "$ADDED" | sed 's/^/  → /'
    echo
fi

if [[ $REMOVED_COUNT -gt 0 ]]; then
    echo -e "${RED}${BOLD}- Removed from $PROFILE_NAME: $REMOVED_COUNT plugins${NC}"
    echo "$REMOVED" | sed 's/^/  → /'
    echo
fi

if [[ $ADDED_COUNT -eq 0 && $REMOVED_COUNT -eq 0 ]]; then
    echo -e "${GREEN}✓ No changes - current installation matches saved profile${NC}"
    echo
fi

# Summary
echo -e "${BLUE}${BOLD}📊 SUMMARY:${NC}"
echo "  Saved profile:        $SAVED_COUNT plugins"
echo "  Currently installed:  $CURRENT_COUNT plugins"
echo -n "  Net change:           "
if [[ $NET_CHANGE -gt 0 ]]; then
    echo -e "${GREEN}+$NET_CHANGE plugins${NC}"
elif [[ $NET_CHANGE -lt 0 ]]; then
    echo -e "${RED}$NET_CHANGE plugins${NC}"
else
    echo -e "${YELLOW}0 plugins${NC}"
fi

# Next steps
if [[ $ADDED_COUNT -gt 0 || $REMOVED_COUNT -gt 0 ]]; then
    echo
    echo -e "${YELLOW}${BOLD}💡 NEXT STEPS:${NC}"
    echo "  Update profile:   $CLAUDEUP_BIN profile save $PROFILE_NAME"
    echo "  Revert to saved:  $CLAUDEUP_BIN profile apply $PROFILE_NAME"
fi
