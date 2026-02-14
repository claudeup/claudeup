#!/usr/bin/env bash
# ABOUTME: Demonstrates scope-aware plugin filtering in outdated and upgrade
# ABOUTME: Tests projectPath-based filtering across user, project, local, and non-project contexts
set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------

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
    echo "Cleaning up: $TEST_DIR"
    rm -rf "$TEST_DIR"
  fi
}

trap cleanup EXIT

run_cmd() {
  local dir="$1"
  shift
  echo "\$ (cd $dir && claudeup $*)"
  pushd "$dir" > /dev/null
  "$CLAUDEUP" "$@" 2>&1 || true
  popd > /dev/null
  echo ""
}

# -----------------------------------------------------------------------------
# Build claudeup
# -----------------------------------------------------------------------------

section "Building claudeup"

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

# Two separate projects
PROJECT_A="$TEST_DIR/project-alpha"
PROJECT_B="$TEST_DIR/project-beta"

# A non-project directory (no .claude/)
NON_PROJECT="$TEST_DIR/random-dir"

echo "TEST_DIR=$TEST_DIR"
echo "CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR"
echo "PROJECT_A=$PROJECT_A"
echo "PROJECT_B=$PROJECT_B"
echo "NON_PROJECT=$NON_PROJECT"

mkdir -p "$CLAUDE_CONFIG_DIR/plugins"
mkdir -p "$CLAUDEUP_HOME"
mkdir -p "$PROJECT_A/.claude"
mkdir -p "$PROJECT_B/.claude"
mkdir -p "$NON_PROJECT"

# Resolve symlinks for projectPath (macOS /var -> /private/var)
PROJECT_A_RESOLVED="$(cd "$PROJECT_A" && pwd -P)"
PROJECT_B_RESOLVED="$(cd "$PROJECT_B" && pwd -P)"

# -----------------------------------------------------------------------------
# Create fake marketplace (git repo)
# -----------------------------------------------------------------------------

section "Creating fake marketplace"

MARKETPLACE_DIR="$TEST_DIR/marketplaces/test-marketplace"
mkdir -p "$MARKETPLACE_DIR"
pushd "$MARKETPLACE_DIR" > /dev/null
git init -q
git commit --allow-empty -m "initial" -q
MARKETPLACE_COMMIT=$(git rev-parse HEAD)
popd > /dev/null

echo "Marketplace: $MARKETPLACE_DIR"
echo "Commit: $MARKETPLACE_COMMIT"

# Register the marketplace
cat > "$CLAUDE_CONFIG_DIR/plugins/known_marketplaces.json" << EOF
{
  "test-marketplace": {
    "source": {"source": "directory"},
    "installLocation": "$MARKETPLACE_DIR",
    "lastUpdated": "2025-01-01T00:00:00Z"
  }
}
EOF
echo "Registered marketplace in known_marketplaces.json"

# -----------------------------------------------------------------------------
# Create plugin registry with multi-scope, multi-project plugins
# -----------------------------------------------------------------------------

section "Creating plugin registry"

cat > "$CLAUDE_CONFIG_DIR/plugins/installed_plugins.json" << EOF
{
  "version": 2,
  "plugins": {
    "global-tool@test-marketplace": [
      {
        "scope": "user",
        "version": "1.0.0",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z",
        "installPath": "$MARKETPLACE_DIR/plugins/global-tool",
        "gitCommitSha": "$MARKETPLACE_COMMIT"
      }
    ],
    "alpha-tool@test-marketplace": [
      {
        "scope": "project",
        "version": "1.0.0",
        "projectPath": "$PROJECT_A_RESOLVED",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z",
        "installPath": "$MARKETPLACE_DIR/plugins/alpha-tool",
        "gitCommitSha": "$MARKETPLACE_COMMIT"
      }
    ],
    "alpha-local@test-marketplace": [
      {
        "scope": "local",
        "version": "1.0.0",
        "projectPath": "$PROJECT_A_RESOLVED",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z",
        "installPath": "$MARKETPLACE_DIR/plugins/alpha-local",
        "gitCommitSha": "$MARKETPLACE_COMMIT"
      }
    ],
    "beta-tool@test-marketplace": [
      {
        "scope": "project",
        "version": "1.0.0",
        "projectPath": "$PROJECT_B_RESOLVED",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z",
        "installPath": "$MARKETPLACE_DIR/plugins/beta-tool",
        "gitCommitSha": "$MARKETPLACE_COMMIT"
      }
    ],
    "shared-tool@test-marketplace": [
      {
        "scope": "user",
        "version": "1.0.0",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z",
        "installPath": "$MARKETPLACE_DIR/plugins/shared-tool",
        "gitCommitSha": "$MARKETPLACE_COMMIT"
      },
      {
        "scope": "project",
        "version": "1.0.0",
        "projectPath": "$PROJECT_A_RESOLVED",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z",
        "installPath": "$MARKETPLACE_DIR/plugins/shared-tool",
        "gitCommitSha": "$MARKETPLACE_COMMIT"
      },
      {
        "scope": "local",
        "version": "1.0.0",
        "projectPath": "$PROJECT_B_RESOLVED",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z",
        "installPath": "$MARKETPLACE_DIR/plugins/shared-tool",
        "gitCommitSha": "$MARKETPLACE_COMMIT"
      }
    ]
  }
}
EOF

echo "Plugin registry:"
echo "  global-tool       -> user scope"
echo "  alpha-tool        -> project scope (project-alpha)"
echo "  alpha-local       -> local scope (project-alpha)"
echo "  beta-tool         -> project scope (project-beta)"
echo "  shared-tool       -> user + project(alpha) + local(beta)"
echo ""
echo "Total: 7 plugin instances across 5 plugins"

# -----------------------------------------------------------------------------
# Test 1: From non-project directory (no .claude/)
# -----------------------------------------------------------------------------

section "Test 1: Non-project directory -- only user-scope plugins"

echo "Without --all (only user scope):"
run_cmd "$NON_PROJECT" outdated

echo "---"
echo "With --all (all scopes):"
run_cmd "$NON_PROJECT" outdated --all

echo "---"
echo "Without --all (upgrade):"
run_cmd "$NON_PROJECT" upgrade

echo "---"
echo "With --all (upgrade):"
run_cmd "$NON_PROJECT" upgrade --all

# -----------------------------------------------------------------------------
# Test 2: From project-alpha directory
# -----------------------------------------------------------------------------

section "Test 2: Project Alpha -- user + alpha's project/local plugins"

echo "Without --all (user + alpha's project/local):"
run_cmd "$PROJECT_A" outdated

echo "---"
echo "With --all (all scopes, all projects):"
run_cmd "$PROJECT_A" outdated --all

echo "---"
echo "Without --all (upgrade):"
run_cmd "$PROJECT_A" upgrade

echo "---"
echo "With --all (upgrade):"
run_cmd "$PROJECT_A" upgrade --all

# -----------------------------------------------------------------------------
# Test 3: From project-beta directory
# -----------------------------------------------------------------------------

section "Test 3: Project Beta -- user + beta's project/local plugins"

echo "Without --all (user + beta's project/local):"
run_cmd "$PROJECT_B" outdated

echo "---"
echo "With --all (all scopes, all projects):"
run_cmd "$PROJECT_B" outdated --all

echo "---"
echo "Without --all (upgrade):"
run_cmd "$PROJECT_B" upgrade

echo "---"
echo "With --all (upgrade):"
run_cmd "$PROJECT_B" upgrade --all

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------

section "Expected plugin counts"

echo "Non-project directory:"
echo "  without --all -> Plugins (2)  [global-tool(user), shared-tool(user)]"
echo "  with --all    -> Plugins (7)  [all instances]"
echo ""
echo "Project Alpha:"
echo "  without --all -> Plugins (5)  [global-tool(user), alpha-tool(project),"
echo "                                 alpha-local(local), shared-tool(user),"
echo "                                 shared-tool(project/alpha)]"
echo "  with --all    -> Plugins (7)  [all instances]"
echo ""
echo "Project Beta:"
echo "  without --all -> Plugins (4)  [global-tool(user), beta-tool(project),"
echo "                                 shared-tool(user), shared-tool(local/beta)]"
echo "  with --all    -> Plugins (7)  [all instances]"

# -----------------------------------------------------------------------------
# Debug output
# -----------------------------------------------------------------------------

if [[ "${DEBUG:-}" == "true" ]]; then
  section "Debug: Environment for manual testing"
  echo "export CLAUDE_CONFIG_DIR=\"$CLAUDE_CONFIG_DIR\""
  echo "export CLAUDEUP_HOME=\"$CLAUDEUP_HOME\""
  echo "export PROJECT_A=\"$PROJECT_A\""
  echo "export PROJECT_B=\"$PROJECT_B\""
  echo "export NON_PROJECT=\"$NON_PROJECT\""
  echo ""
  echo "To clean up: rm -rf \"$TEST_DIR\""
fi
