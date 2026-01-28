#!/usr/bin/env bash
# ABOUTME: Demonstrates claudeup onboarding workflow for existing Claude Code users
# ABOUTME: Creates isolated test environment, simulates user config, tests profile management
set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------

# Set USE_LOCAL_BUILD=true to test local changes instead of released version
USE_LOCAL_BUILD="${USE_LOCAL_BUILD:-false}"
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

# -----------------------------------------------------------------------------
# Install claudeup
# -----------------------------------------------------------------------------

section "Installing claudeup"

if [[ "$USE_LOCAL_BUILD" == "true" ]]; then
  echo "Building from local source..."
  go build -o bin/claudeup ./cmd/claudeup
  cp bin/claudeup ~/.local/bin/claudeup
else
  echo "Installing from release..."
  curl -fsSL https://claudeup.github.io/install.sh | bash
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

# -----------------------------------------------------------------------------
# Simulate existing Claude Code user configuration
# -----------------------------------------------------------------------------

section "Creating simulated user configuration"

mkdir -p "$CLAUDE_CONFIG_DIR"

cat > "$CLAUDE_CONFIG_DIR/settings.json" << 'SETTINGS'
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "apiKeyHelper": "/bin/generate_temp_api_key.sh",
  "awsCredentialExport": "/bin/generate_aws_grant.sh",
  "awsAuthRefresh": "aws sso login --profile myprofile",

  "forceLoginMethod": "claudeai",
  "forceLoginOrgUUID": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",

  "sandbox": {
    "enabled": true,
    "excludedCommands": ["docker", "podman"],
    "network": {
      "allowLocalBinding": true,
      "allowUnixSockets": ["/var/run/docker.sock"],
      "httpProxyPort": 8080,
      "socksProxyPort": 1080
    }
  },
  "permissions": {
    "allow": [
      "Bash(git add:*)",
      "Bash(npm run build)",
      "Bash(npm run:*)",
      "Edit(/src/**/*.ts)"
    ],
    "ask": ["Bash(gh pr create:*)", "Bash(git commit:*)"],
    "deny": ["Read(*.env)", "Read(./secrets/**)", "Bash(rm:*)", "Bash(curl:*)"],
    "defaultMode": "default",
    "additionalDirectories": []
  },
  "hooks": {
    "PreToolUse": [],
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "prettier --write \"$CLAUDE_FILE_PATH\""
          }
        ]
      }
    ]
  },
  "enabledPlugins": {
    "claude-code-setup@claude-plugins-official": true,
    "code-review@claude-plugins-official": true,
    "pr-review-toolkit@claude-plugins-official": true,
    "security-guidance@claude-plugins-official": true,
    "superpowers@claude-plugins-official": true
  },
  "includeCoAuthoredBy": false
}
SETTINGS

echo "Created settings.json with 5 enabled plugins"

# -----------------------------------------------------------------------------
# Add marketplace (required for plugin installation)
# -----------------------------------------------------------------------------

section "Adding official plugin marketplace"

claude plugin marketplace add anthropics/claude-plugins-official

# -----------------------------------------------------------------------------
# Run claudeup setup (preserves existing config)
# -----------------------------------------------------------------------------

section "Running claudeup setup"

mkdir -p "$PROJECT_DIR"
pushd "$PROJECT_DIR" > /dev/null

claudeup setup -y

# -----------------------------------------------------------------------------
# Verify setup result
# -----------------------------------------------------------------------------

section "Verifying setup"

claudeup profile show my-setup

# -----------------------------------------------------------------------------
# Create and apply project-specific profile
# -----------------------------------------------------------------------------

section "Creating project profile"

cat > "$TEST_DIR/my-project.json" << 'PROFILE'
{
  "name": "my-project",
  "marketplaces": [
    {
      "source": "github",
      "repo": "anthropics/skills"
    }
  ],
  "plugins": ["document-skills@anthropic-agent-skills"]
}
PROFILE

claudeup profile create my-project --description "My project description" --from-file "$TEST_DIR/my-project.json"

section "Applying project profile"

claudeup profile apply my-project -y --scope project

# Verify settings.json is valid
if ! jq < "$PROJECT_DIR/.claude/settings.json" > /dev/null; then
  echo "Error: Invalid JSON in $PROJECT_DIR/.claude/settings.json"
  exit 1
fi

# -----------------------------------------------------------------------------
# Show final state
# -----------------------------------------------------------------------------

section "Final state"

claudeup profile list

echo ""
claudeup profile show my-project

echo ""
claudeup plugin list --enabled --format table

popd > /dev/null

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
