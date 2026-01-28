#!/usr/bin/env bash
# ABOUTME: Demonstrates profile sync functionality for team collaboration
# ABOUTME: Tests sync with existing profile vs new profile (team member clone scenario)
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

verify_plugin_exists() {
  local settings_file="$1"
  local plugin="$2"
  if jq -e ".enabledPlugins[\"$plugin\"]" "$settings_file" > /dev/null 2>&1; then
    return 0
  else
    return 1
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

# =============================================================================
# TEST -1: Sync fails when profile doesn't exist anywhere
# =============================================================================

section "Test -1: Sync bootstraps profile from current state when missing"

echo "Scenario: .claudeup.json exists but profile definition is missing."
echo "Sync should create the profile by snapshotting the current state,"
echo "enabling teams to use sync even with older project setups."
echo ""

# First, set up some plugins in the project scope
cat > "$PROJECT_DIR/.claude/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "existing-plugin@marketplace": true
  }
}
SETTINGS
echo "Created project settings with existing-plugin"

# Create .claudeup.json pointing to a profile that doesn't exist
cat > "$PROJECT_DIR/.claudeup.json" << 'CONFIG'
{
  "version": "1",
  "profile": "bootstrapped-profile"
}
CONFIG
echo "Created .claudeup.json pointing to 'bootstrapped-profile'"

# Verify profile does NOT exist in either location
if [[ -f "$CLAUDEUP_HOME/profiles/bootstrapped-profile.json" ]]; then
  echo "ERROR: Profile should not exist in user profiles"
  exit 1
fi
if [[ -f "$PROJECT_DIR/.claudeup/profiles/bootstrapped-profile.json" ]]; then
  echo "ERROR: Profile should not exist in project profiles"
  exit 1
fi
echo "✓ Confirmed: Profile does not exist in user or project profiles"

# Run sync - should SUCCEED by bootstrapping from current state
echo ""
echo "Running: claudeup profile sync -y (expecting bootstrap)"
pushd "$PROJECT_DIR" > /dev/null
if ! $CLAUDEUP profile sync -y 2>&1; then
  echo ""
  echo "✗ ERROR: Sync should have succeeded by bootstrapping profile"
  popd > /dev/null
  exit 1
fi
popd > /dev/null

echo ""
echo "✓ Sync succeeded by bootstrapping profile from current state"

# Verify profile was created in user profiles directory
if [[ ! -f "$CLAUDEUP_HOME/profiles/bootstrapped-profile.json" ]]; then
  echo "✗ ERROR: Profile should have been created in user profiles"
  exit 1
fi
echo "✓ Profile was created: $CLAUDEUP_HOME/profiles/bootstrapped-profile.json"

# Verify the bootstrapped profile captured the existing plugin
if ! jq -e '.perScope.project.plugins | index("existing-plugin@marketplace")' "$CLAUDEUP_HOME/profiles/bootstrapped-profile.json" > /dev/null 2>&1; then
  echo "✗ ERROR: Bootstrapped profile should contain existing-plugin"
  cat "$CLAUDEUP_HOME/profiles/bootstrapped-profile.json"
  exit 1
fi
echo "✓ Bootstrapped profile captured existing plugins"

# Clean up for next test
rm "$PROJECT_DIR/.claudeup.json"
rm "$CLAUDEUP_HOME/profiles/bootstrapped-profile.json"

echo ""
echo "Test -1 PASSED: Sync bootstraps profile from current state when missing"

# =============================================================================
# TEST 0: profile apply --scope project creates .claudeup/profiles/
# =============================================================================

section "Test 0: profile apply --scope project creates project profile"

echo "Scenario: Team lead applies profile at project scope to share with team."
echo "This should create .claudeup/profiles/<name>.json for team members."
echo ""

# First, create the profile in user profiles directory (as team lead would have)
cat > "$CLAUDEUP_HOME/profiles/team-backend.json" << 'PROFILE'
{
  "name": "team-backend",
  "description": "Team backend development profile",
  "perScope": {
    "user": {
      "plugins": ["superpowers@superpowers-marketplace"]
    },
    "project": {
      "plugins": ["backend-development@claude-code-workflows"]
    }
  }
}
PROFILE
echo "Created profile in user profiles directory: $CLAUDEUP_HOME/profiles/team-backend.json"

# Verify project profile directory does NOT exist yet
if [[ -d "$PROJECT_DIR/.claudeup/profiles" ]]; then
  echo "ERROR: Project profiles directory should not exist yet"
  exit 1
fi
echo "✓ Confirmed: Project profiles directory does not exist yet"

# Run profile apply --scope project
echo ""
echo "Running: claudeup profile apply team-backend --scope project -y"
pushd "$PROJECT_DIR" > /dev/null
$CLAUDEUP profile apply team-backend --scope project -y
popd > /dev/null

# Verify .claudeup.json was created
if [[ ! -f "$PROJECT_DIR/.claudeup.json" ]]; then
  echo ""
  echo "✗ ERROR: .claudeup.json should have been created"
  exit 1
fi
echo ""
echo "✓ .claudeup.json was created"

# Verify profile was saved to project profiles directory
if [[ ! -f "$PROJECT_DIR/.claudeup/profiles/team-backend.json" ]]; then
  echo "✗ ERROR: Profile should have been saved to .claudeup/profiles/"
  exit 1
fi
echo "✓ Profile was saved to .claudeup/profiles/team-backend.json"

# Verify the content matches
SAVED_NAME=$(jq -r '.name' "$PROJECT_DIR/.claudeup/profiles/team-backend.json")
if [[ "$SAVED_NAME" != "team-backend" ]]; then
  echo "✗ ERROR: Saved profile has wrong name: $SAVED_NAME"
  exit 1
fi
echo "✓ Saved profile content is correct"

echo ""
echo "Test 0 PASSED: profile apply --scope project creates .claudeup/profiles/"

# Now remove the user profile to simulate team member scenario
rm "$CLAUDEUP_HOME/profiles/team-backend.json"
echo ""
echo "Removed user profile to simulate team member scenario"

# =============================================================================
# TEST 1: Sync WITHOUT existing profile (team member clone scenario)
# =============================================================================

section "Test 1: Sync WITHOUT existing local profile"

echo "Scenario: A team member clones a repo with .claudeup.json but doesn't"
echo "have the profile installed locally yet."
echo ""

# Project profile already exists from Test 0 (as if checked into git)
PROJECT_PROFILES_DIR="$PROJECT_DIR/.claudeup/profiles"
echo "Using project profile from Test 0: $PROJECT_PROFILES_DIR/team-backend.json"

cat > "$PROJECT_PROFILES_DIR/team-backend.json" << 'PROFILE'
{
  "name": "team-backend",
  "description": "Team backend development profile",
  "perScope": {
    "user": {
      "plugins": ["superpowers@superpowers-marketplace"]
    },
    "project": {
      "plugins": ["backend-development@claude-code-workflows"]
    }
  }
}
PROFILE
echo "Created team profile in project directory: $PROJECT_PROFILES_DIR/team-backend.json"

# Create .claudeup.json (as if checked into git)
cat > "$PROJECT_DIR/.claudeup.json" << 'CONFIG'
{
  "version": "1",
  "profile": "team-backend",
  "profileSource": "custom"
}
CONFIG
echo "Created .claudeup.json pointing to 'team-backend' profile"

# Verify profile does NOT exist in user profiles directory
if [[ -f "$CLAUDEUP_HOME/profiles/team-backend.json" ]]; then
  echo "ERROR: Profile should not exist in user profiles directory yet"
  exit 1
fi
echo "✓ Confirmed: Profile does not exist in user profiles directory"

# Show initial state
echo ""
echo "Initial state:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope"
check_plugins "$PROJECT_DIR/.claude/settings.json" "Project scope"
echo ""
echo "User profiles directory:"
ls -la "$CLAUDEUP_HOME/profiles/" 2>/dev/null || echo "  (empty)"

# Run sync
echo ""
echo "Running: claudeup profile sync"
pushd "$PROJECT_DIR" > /dev/null
$CLAUDEUP profile sync -y
popd > /dev/null

# Verify results
echo ""
echo "After sync:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope"
check_plugins "$PROJECT_DIR/.claude/settings.json" "Project scope"

# Verify profile was CREATED in user profiles directory
if [[ ! -f "$CLAUDEUP_HOME/profiles/team-backend.json" ]]; then
  echo ""
  echo "✗ ERROR: Profile should have been created in user profiles directory"
  exit 1
fi
echo ""
echo "✓ Profile created in user profiles directory"
echo "  $CLAUDEUP_HOME/profiles/team-backend.json"

# Verify plugins were installed
if ! verify_plugin_exists "$CLAUDE_CONFIG_DIR/settings.json" "superpowers@superpowers-marketplace"; then
  echo "✗ ERROR: User scope plugin not installed"
  exit 1
fi
echo "✓ User scope plugin installed: superpowers@superpowers-marketplace"

if ! verify_plugin_exists "$PROJECT_DIR/.claude/settings.json" "backend-development@claude-code-workflows"; then
  echo "✗ ERROR: Project scope plugin not installed"
  exit 1
fi
echo "✓ Project scope plugin installed: backend-development@claude-code-workflows"

echo ""
echo "Test 1 PASSED: Sync creates local profile copy and installs plugins"

# =============================================================================
# TEST 2: Sync WITH existing profile (profile already exists locally)
# =============================================================================

section "Test 2: Sync WITH existing local profile"

echo "Scenario: User already has the profile, runs sync to update/reinstall."
echo ""

# Simulate drift by removing a plugin from project scope
echo "Simulating drift: removing backend-development plugin from project scope..."
cat > "$PROJECT_DIR/.claude/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {}
}
SETTINGS

echo ""
echo "State before sync (with drift):"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope"
check_plugins "$PROJECT_DIR/.claude/settings.json" "Project scope (drift: missing plugin)"

# Verify profile EXISTS in user profiles directory
if [[ ! -f "$CLAUDEUP_HOME/profiles/team-backend.json" ]]; then
  echo "ERROR: Profile should exist from Test 1"
  exit 1
fi
echo ""
echo "✓ Profile exists in user profiles directory (from Test 1)"

# Run sync again
echo ""
echo "Running: claudeup profile sync"
pushd "$PROJECT_DIR" > /dev/null
$CLAUDEUP profile sync -y
popd > /dev/null

# Verify drift was fixed
echo ""
echo "After sync:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope"
check_plugins "$PROJECT_DIR/.claude/settings.json" "Project scope (should be restored)"

if ! verify_plugin_exists "$PROJECT_DIR/.claude/settings.json" "backend-development@claude-code-workflows"; then
  echo ""
  echo "✗ ERROR: Project scope plugin not restored after sync"
  exit 1
fi
echo ""
echo "✓ Project scope plugin restored: backend-development@claude-code-workflows"

echo ""
echo "Test 2 PASSED: Sync restores missing plugins when profile exists"

# =============================================================================
# TEST 3: Sync with --replace flag (declarative user scope)
# =============================================================================

section "Test 3: Sync with --replace flag"

echo "Scenario: User has extra plugins at user scope, wants declarative sync."
echo ""

# Add extra plugins to user scope
cat > "$CLAUDE_CONFIG_DIR/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "superpowers@superpowers-marketplace": true,
    "extra-plugin-1@marketplace": true,
    "extra-plugin-2@marketplace": true
  }
}
SETTINGS
echo "Added extra plugins to user scope (not in profile)"

echo ""
echo "State before sync with --replace:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope (has 3 plugins, profile has 1)"

# Run sync with --replace
echo ""
echo "Running: claudeup profile sync --replace -y"
pushd "$PROJECT_DIR" > /dev/null
$CLAUDEUP profile sync --replace -y
popd > /dev/null

echo ""
echo "After sync --replace:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope (should have only 1 plugin)"

# Verify extra plugins were removed
USER_PLUGIN_COUNT=$(jq '.enabledPlugins | length' "$CLAUDE_CONFIG_DIR/settings.json")
if [[ "$USER_PLUGIN_COUNT" -ne 1 ]]; then
  echo ""
  echo "✗ ERROR: Expected 1 plugin in user scope after --replace, got $USER_PLUGIN_COUNT"
  exit 1
fi
echo ""
echo "✓ Extra plugins removed with --replace (user scope now has 1 plugin)"

if verify_plugin_exists "$CLAUDE_CONFIG_DIR/settings.json" "extra-plugin-1@marketplace"; then
  echo "✗ ERROR: extra-plugin-1 should have been removed"
  exit 1
fi
echo "✓ extra-plugin-1 was removed"

if verify_plugin_exists "$CLAUDE_CONFIG_DIR/settings.json" "extra-plugin-2@marketplace"; then
  echo "✗ ERROR: extra-plugin-2 should have been removed"
  exit 1
fi
echo "✓ extra-plugin-2 was removed"

echo ""
echo "Test 3 PASSED: Sync --replace removes extra plugins at user scope"

# =============================================================================
# TEST 4: Sync without --replace (additive user scope - default)
# =============================================================================

section "Test 4: Sync without --replace (additive default)"

echo "Scenario: User has extra plugins at user scope, wants to keep them."
echo ""

# Add extra plugins to user scope again
cat > "$CLAUDE_CONFIG_DIR/settings.json" << 'SETTINGS'
{
  "enabledPlugins": {
    "extra-plugin-a@marketplace": true,
    "extra-plugin-b@marketplace": true
  }
}
SETTINGS
echo "Added extra plugins to user scope (not in profile)"

echo ""
echo "State before additive sync:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope (has 2 extra plugins)"

# Run sync WITHOUT --replace (additive)
echo ""
echo "Running: claudeup profile sync -y (no --replace, additive mode)"
pushd "$PROJECT_DIR" > /dev/null
$CLAUDEUP profile sync -y
popd > /dev/null

echo ""
echo "After additive sync:"
check_plugins "$CLAUDE_CONFIG_DIR/settings.json" "User scope (should have 3 plugins)"

# Verify extra plugins were preserved AND profile plugin was added
USER_PLUGIN_COUNT=$(jq '.enabledPlugins | length' "$CLAUDE_CONFIG_DIR/settings.json")
if [[ "$USER_PLUGIN_COUNT" -ne 3 ]]; then
  echo ""
  echo "✗ ERROR: Expected 3 plugins in user scope (2 extra + 1 from profile), got $USER_PLUGIN_COUNT"
  exit 1
fi
echo ""
echo "✓ Additive mode preserved extra plugins (3 total)"

if ! verify_plugin_exists "$CLAUDE_CONFIG_DIR/settings.json" "extra-plugin-a@marketplace"; then
  echo "✗ ERROR: extra-plugin-a should have been preserved"
  exit 1
fi
echo "✓ extra-plugin-a was preserved"

if ! verify_plugin_exists "$CLAUDE_CONFIG_DIR/settings.json" "extra-plugin-b@marketplace"; then
  echo "✗ ERROR: extra-plugin-b should have been preserved"
  exit 1
fi
echo "✓ extra-plugin-b was preserved"

if ! verify_plugin_exists "$CLAUDE_CONFIG_DIR/settings.json" "superpowers@superpowers-marketplace"; then
  echo "✗ ERROR: Profile plugin should have been added"
  exit 1
fi
echo "✓ Profile plugin was added: superpowers@superpowers-marketplace"

echo ""
echo "Test 4 PASSED: Sync without --replace preserves existing user plugins"

# =============================================================================
# TEST 5: Sync is idempotent
# =============================================================================

section "Test 5: Sync is idempotent"

echo "Scenario: Running sync multiple times should produce same result."
echo ""

# Capture state after first sync
PLUGINS_BEFORE=$(jq -c '.enabledPlugins' "$CLAUDE_CONFIG_DIR/settings.json")
PROJECT_PLUGINS_BEFORE=$(jq -c '.enabledPlugins' "$PROJECT_DIR/.claude/settings.json")

echo "Running sync twice more..."
pushd "$PROJECT_DIR" > /dev/null
$CLAUDEUP profile sync -y
$CLAUDEUP profile sync -y
popd > /dev/null

# Capture state after additional syncs
PLUGINS_AFTER=$(jq -c '.enabledPlugins' "$CLAUDE_CONFIG_DIR/settings.json")
PROJECT_PLUGINS_AFTER=$(jq -c '.enabledPlugins' "$PROJECT_DIR/.claude/settings.json")

if [[ "$PLUGINS_BEFORE" != "$PLUGINS_AFTER" ]]; then
  echo "✗ ERROR: User scope plugins changed after idempotent syncs"
  echo "  Before: $PLUGINS_BEFORE"
  echo "  After:  $PLUGINS_AFTER"
  exit 1
fi
echo "✓ User scope unchanged after multiple syncs"

if [[ "$PROJECT_PLUGINS_BEFORE" != "$PROJECT_PLUGINS_AFTER" ]]; then
  echo "✗ ERROR: Project scope plugins changed after idempotent syncs"
  echo "  Before: $PROJECT_PLUGINS_BEFORE"
  echo "  After:  $PROJECT_PLUGINS_AFTER"
  exit 1
fi
echo "✓ Project scope unchanged after multiple syncs"

echo ""
echo "Test 5 PASSED: Sync is idempotent"

# =============================================================================
# Summary
# =============================================================================

section "All tests passed!"

echo "Profile sync feature works correctly:"
echo "  ✓ Test -1: Sync bootstraps profile from current state when missing"
echo "  ✓ Test 0: profile apply --scope project creates .claudeup/profiles/"
echo "  ✓ Test 1: Sync creates local profile copy when profile doesn't exist"
echo "  ✓ Test 2: Sync restores missing plugins when profile exists"
echo "  ✓ Test 3: Sync --replace removes extra plugins at user scope"
echo "  ✓ Test 4: Sync without --replace preserves existing user plugins"
echo "  ✓ Test 5: Sync is idempotent (multiple runs produce same result)"
echo ""
echo "Key behaviors:"
echo "  • Sync reads .claudeup.json to find profile name"
echo "  • Sync loads profile from project dir first, then user profiles"
echo "  • Sync creates/updates local profile copy in user profiles dir"
echo "  • User scope: additive by default, declarative with --replace"
echo "  • Project scope: always declarative (replaces settings)"

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
