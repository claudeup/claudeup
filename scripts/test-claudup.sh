#!/usr/bin/env bash
# Test script for canonical key ordering fix
set -e

# curl -fsSL https://claudeup.github.io/install.sh | bash
go build -o bin/claudeup ./cmd/claudeup
cp bin/claudeup ~/.local/bin/claudeup

# Create isolated test environment
TEST_DIR=$(mktemp -d)
export CLAUDE_CONFIG_DIR="$TEST_DIR/.claude"
export CLAUDEUP_HOME="$TEST_DIR/.claudeup"
PROJECT_DIR="$TEST_DIR/project"

echo "TEST_DIR=$TEST_DIR"
echo "CLAUDEUP_HOME=$CLAUDEUP_HOME"
echo "PROJECT_DIR=$PROJECT_DIR"
echo ""

# Create settings.json (simulates existing Claude Code user config)
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

# Make a project directory
mkdir -p "$PROJECT_DIR"
cd "$PROJECT_DIR"

# Add marketplace via Claude CLI (creates proper known_marketplaces.json)
echo "Adding marketplace via Claude CLI..."
claude plugin marketplace add anthropics/claude-plugins-official

# setup -y now preserves existing settings automatically (doesn't apply default)
# and saves them as "my-setup" profile by default
claudeup setup -y

# Create a test profile with a plugin to trigger file write
cat > "$TEST_DIR/test.json" << 'PROFILE'
{
  "name": "test",
  "description": "Test profile for canonical ordering verification",
  "marketplaces": [
    {
      "source": "github",
      "repo": "anthropics/skills"
    }
  ],
  "plugins": ["document-skills@anthropic-agent-skills"]
}
PROFILE

claudeup profile create test --description "Test profile for canonical ordering verification" --from-file "$TEST_DIR/test.json"

# Verify claudeup sees the profile
echo "=== PROFILE LIST ==="
claudeup profile list

echo ""
echo "=== BEFORE (alphabetical order) ==="
if ! jq < "$CLAUDE_CONFIG_DIR/settings.json"; then
  echo "Error: Invalid JSON in $CLAUDE_CONFIG_DIR/settings.json"
  exit 1
fi

echo ""

# Apply profile at project scope
claudeup profile apply test -y --scope project
if ! jq < "$PROJECT_DIR/.claude/settings.json"; then
  echo "Error: Invalid JSON in $PROJECT_DIR/.claude/settings.json"
  exit 1
fi

claudeup profile show test
claudeup plugin list

# Apply profile at user scope (now works because profile has marketplace)
claudeup profile apply my-setup -y --scope user

# echo ""
# echo "=== AFTER (canonical order) ==="
# jq < "$TEST_DIR/settings.json" || echo "invalid json: $TEST_DIR/settings.json"
# claudeup profile show test


# Cleanup
# rm -rf "$TEST_DIR"

echo ""
echo "=== Test environment variables ==="
echo "export CLAUDE_CONFIG_DIR=\"$TEST_DIR/.claude\""
echo "export CLAUDEUP_HOME=\"$TEST_DIR/.claudeup\""
echo "export PROJECT_DIR=\"$TEST_DIR/project\""

cd "$CLAUDE_CONFIG_DIR"
# Profile name changed from "saved" to "my-setup" in new setup behavior
claudeup profile show my-setup
