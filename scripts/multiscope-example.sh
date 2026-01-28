#!/usr/bin/env bash
# ABOUTME: Demonstrates multi-scope profile capture and restoration
# ABOUTME: Tests additive user-scope behavior and --replace override
set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------

# Set USE_LOCAL_BUILD=true to test local changes instead of released version
USE_LOCAL_BUILD="${USE_LOCAL_BUILD:-true}"
CLEANUP_ON_EXIT="${CLEANUP_ON_EXIT:-false}"

# -----------------------------------------------------------------------------
# Helper functions
# -----------------------------------------------------------------------------

section() {
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  $1"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
}

cleanup() {
  if [[ "$CLEANUP_ON_EXIT" == "true" && -n "${TEST_DIR:-}" ]]; then
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Cleanup"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Removing test directory: $TEST_DIR"
    rm -rf "$TEST_DIR"
    echo "Done."
  fi
}

trap cleanup EXIT

check_plugins() {
  local settings_file="$1"
  local description="$2"
  echo "$description:"
  if [[ -f "$settings_file" ]]; then
    jq -r '.enabledPlugins // {} | keys[]' "$settings_file" 2>/dev/null | sed 's/^/  - /' || echo "  (none)"
  else
    echo "  (file does not exist)"
  fi
}

# -----------------------------------------------------------------------------
# Build claudeup
# -----------------------------------------------------------------------------

section "Building claudeup"

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if [[ "$USE_LOCAL_BUILD" == "true" ]]; then
  echo "Building from local source..."
  pushd "$PROJECT_ROOT" > /dev/null
  go build -o bin/claudeup ./cmd/claudeup
  popd > /dev/null
  CLAUDEUP="$PROJECT_ROOT/bin/claudeup"
else
  echo "Using installed claudeup..."
  CLAUDEUP="claudeup"
fi

# -----------------------------------------------------------------------------
# Create isolated test environment
# -----------------------------------------------------------------------------

section "Setting up test environment"

TEST_DIR=$(mktemp -d)
export CLAUDE_CONFIG_DIR="$TEST_DIR/.claude"
export CLAUDEUP_HOME="$TEST_DIR/.claudeup"
PROJECT_DIR="$TEST_DIR/project"

echo "TEST_DIR=$TEST_DIR"
echo "CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR"
echo "CLAUDEUP_HOME=$CLAUDEUP_HOME"
echo "PROJECT_DIR=$PROJECT_DIR"

mkdir -p "$CLAUDE_CONFIG_DIR"
mkdir -p "$CLAUDEUP_HOME/profiles"
mkdir -p "$PROJECT_DIR/.claude"

# -----------------------------------------------------------------------------
# Create multi-scope configuration
# -----------------------------------------------------------------------------

section "Creating multi-scope configuration"

# User-scope settings (global plugins)
cat > "$CLAUDE_CONFIG_DIR/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "superpowers@claude-plugins-official": true,
    "code-review@claude-plugins-official": true,
    "pr-review-toolkit@claude-plugins-official": true
  }
}
SETTINGS
echo "Created user-scope settings with 3 plugins"

# Project-scope settings (project-specific plugins)
cat > "$PROJECT_DIR/.claude/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "backend-development@claude-code-workflows": true,
    "tdd-workflows@claude-code-workflows": true
  }
}
SETTINGS
echo "Created project-scope settings with 2 plugins"

# Local-scope settings (developer-specific overrides)
cat > "$PROJECT_DIR/.claude/settings.local.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "hookify@claude-plugins-official": true
  }
}
SETTINGS
echo "Created local-scope settings with 1 plugin"

# -----------------------------------------------------------------------------
# Show initial state
# -----------------------------------------------------------------------------

section "Initial state (before profile save)"

check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope"
check_plugins "$PROJECT_DIR/.claude/settings.json" "Project scope"
check_plugins "$PROJECT_DIR/.claude/settings.local.json" "Local scope"

# -----------------------------------------------------------------------------
# Save multi-scope profile
# -----------------------------------------------------------------------------

section "Saving multi-scope profile"

pushd "$PROJECT_DIR" > /dev/null
$CLAUDEUP profile save my-multiscope -y
popd > /dev/null

echo ""
echo "Saved profile contents:"
cat "$CLAUDEUP_HOME/profiles/my-multiscope.json" | jq .

# Verify perScope structure
if jq -e '.perScope' "$CLAUDEUP_HOME/profiles/my-multiscope.json" > /dev/null; then
  echo ""
  echo "✓ Profile has perScope structure (multi-scope format)"
else
  echo ""
  echo "✗ ERROR: Profile missing perScope structure!"
  exit 1
fi

# -----------------------------------------------------------------------------
# Test 1: Additive user-scope behavior (default)
# -----------------------------------------------------------------------------

section "Test 1: Additive user-scope behavior (default)"

# Add some extra plugins to user scope that aren't in the profile
cat > "$CLAUDE_CONFIG_DIR/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "extra-plugin-1@marketplace": true,
    "extra-plugin-2@marketplace": true
  }
}
SETTINGS
echo "Set up user scope with 2 extra plugins (not in profile)"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope before apply"

# Clear project/local to test restoration
rm -f "$PROJECT_DIR/.claude/settings.json"
rm -f "$PROJECT_DIR/.claude/settings.local.json"
echo "Cleared project and local scope settings"

# Apply profile (additive by default)
pushd "$PROJECT_DIR" > /dev/null
echo ""
echo "Running: claudeup profile apply my-multiscope -y"
$CLAUDEUP profile apply my-multiscope -y
popd > /dev/null

echo ""
echo "After additive apply:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope (should have BOTH extra and profile plugins)"
check_plugins "$PROJECT_DIR/.claude/settings.json" "Project scope (should have profile plugins)"
check_plugins "$PROJECT_DIR/.claude/settings.local.json" "Local scope (should have profile plugins)"

# Verify additive behavior
USER_PLUGIN_COUNT=$(jq '.enabledPlugins | length' "$CLAUDE_CONFIG_DIR/settings.json")
if [[ "$USER_PLUGIN_COUNT" -eq 5 ]]; then
  echo ""
  echo "✓ Additive behavior works: user scope has 5 plugins (2 extra + 3 from profile)"
else
  echo ""
  echo "✗ ERROR: Expected 5 plugins in user scope, got $USER_PLUGIN_COUNT"
  exit 1
fi

# -----------------------------------------------------------------------------
# Test 2: Replace user-scope behavior (--replace)
# -----------------------------------------------------------------------------

section "Test 2: Replace user-scope behavior (--replace)"

# Reset to have extra plugins again
cat > "$CLAUDE_CONFIG_DIR/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "extra-plugin-1@marketplace": true,
    "extra-plugin-2@marketplace": true,
    "another-extra@marketplace": true
  }
}
SETTINGS
echo "Set up user scope with 3 extra plugins (not in profile)"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope before --replace apply"

# Apply profile with --replace (declarative)
pushd "$PROJECT_DIR" > /dev/null
echo ""
echo "Running: claudeup profile apply my-multiscope --replace -y"
$CLAUDEUP profile apply my-multiscope --replace -y
popd > /dev/null

echo ""
echo "After --replace apply:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope (should have ONLY profile plugins)"

# Verify replace behavior
USER_PLUGIN_COUNT=$(jq '.enabledPlugins | length' "$CLAUDE_CONFIG_DIR/settings.json")
if [[ "$USER_PLUGIN_COUNT" -eq 3 ]]; then
  echo ""
  echo "✓ Replace behavior works: user scope has 3 plugins (only profile plugins)"
else
  echo ""
  echo "✗ ERROR: Expected 3 plugins in user scope, got $USER_PLUGIN_COUNT"
  exit 1
fi

# Verify extra plugins were removed
if jq -e '.enabledPlugins["extra-plugin-1@marketplace"]' "$CLAUDE_CONFIG_DIR/settings.json" > /dev/null 2>&1; then
  echo "✗ ERROR: extra-plugin-1 should have been removed with --replace"
  exit 1
else
  echo "✓ Extra plugins were removed as expected"
fi

# -----------------------------------------------------------------------------
# Test 3: Project/local scope is always declarative
# -----------------------------------------------------------------------------

section "Test 3: Project/local scope is always declarative"

# Add extra plugins to project scope
cat > "$PROJECT_DIR/.claude/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "project-extra@marketplace": true,
    "backend-development@claude-code-workflows": true
  }
}
SETTINGS
echo "Set up project scope with 1 extra + 1 profile plugin"

# Apply profile (even without --replace, project scope should be replaced)
pushd "$PROJECT_DIR" > /dev/null
echo ""
echo "Running: claudeup profile apply my-multiscope -y (no --replace)"
$CLAUDEUP profile apply my-multiscope -y
popd > /dev/null

echo ""
check_plugins "$PROJECT_DIR/.claude/settings.json" "Project scope after apply"

# Verify project scope was replaced (not additive)
if jq -e '.enabledPlugins["project-extra@marketplace"]' "$PROJECT_DIR/.claude/settings.json" > /dev/null 2>&1; then
  echo ""
  echo "✗ ERROR: project-extra should have been removed (project scope is declarative)"
  exit 1
else
  echo ""
  echo "✓ Project scope is declarative: extra plugin was removed"
fi

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------

section "All tests passed!"

echo "Multi-scope profile feature works correctly:"
echo "  ✓ Profile save captures all scopes (user, project, local)"
echo "  ✓ Profile apply uses additive behavior for user scope by default"
echo "  ✓ Profile apply --replace uses declarative behavior for user scope"
echo "  ✓ Project and local scopes always use declarative behavior"

# -----------------------------------------------------------------------------
# Debug output (optional)
# -----------------------------------------------------------------------------

if [[ "${DEBUG:-}" == "true" ]]; then
  section "Debug: Environment variables for manual testing"
  echo "export CLAUDE_CONFIG_DIR=\"$CLAUDE_CONFIG_DIR\""
  echo "export CLAUDEUP_HOME=\"$CLAUDEUP_HOME\""
  echo "export PROJECT_DIR=\"$PROJECT_DIR\""
  echo ""
  echo "To clean up: rm -rf \"$TEST_DIR\""
fi
