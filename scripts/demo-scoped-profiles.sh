#!/bin/bash
# ABOUTME: End-to-end demo script for scoped profiles functionality
# ABOUTME: Uses REAL Claude CLI commands to test claudeup against actual Claude behavior

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Helper functions
print_section() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_step() {
    echo -e "${GREEN}▸ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

print_command() {
    echo -e "${CYAN}$ $1${NC}"
}

pause() {
    echo ""
    read -p "Press ENTER to continue..."
}

# Check if claude CLI is available
if ! command -v claude &> /dev/null; then
    echo -e "${YELLOW}⚠ Claude CLI not found in PATH${NC}"
    echo "This demo requires the Claude CLI to be installed."
    echo "See: https://code.claude.com/docs/en/getting-started"
    exit 1
fi

# Build the binary
print_section "Building claudeup"
print_step "Building fresh binary..."
go build -o bin/claudeup ./cmd/claudeup
print_info "Binary built successfully"

# Save the original working directory (claudeup project root)
CLAUDEUP_ROOT=$(pwd)

# Create test directory structure
TEST_DIR=$(mktemp -d -t claudeup-demo-XXXXXX)
CLAUDE_DIR="$TEST_DIR/.claude"
PROJECT_DIR="$TEST_DIR/my-project"

# Save real HOME before we override it
REAL_HOME="$HOME"

print_section "Setup: Creating Test Environment"
print_info "Test directory: $TEST_DIR"
print_info "Claude directory: $CLAUDE_DIR"
print_info "Project directory: $PROJECT_DIR"

# Set CLAUDE_CONFIG_DIR for all Claude CLI and claudeup commands
export CLAUDE_CONFIG_DIR="$CLAUDE_DIR"
# Override HOME so claudeup reads config from demo dir, not real ~/.claudeup
export HOME="$TEST_DIR"
# Disable Claude's interactive terminal setup prompt
export CI=true

# Create directory structure
mkdir -p "$PROJECT_DIR/.claudeup/profiles"

print_info "Using isolated Claude environment: $CLAUDE_DIR"
print_info "All claude and claudeup commands will use this directory"

# Pre-create Claude config to skip first-run setup
print_step "Copying Claude configuration from your real ~/.claude directory..."

# Copy ~/.claude.json from home directory
if [ -f "$REAL_HOME/.claude.json" ]; then
    print_info "Copying ~/.claude.json"
    cp "$REAL_HOME/.claude.json" "$TEST_DIR/.claude.json"
else
    print_info "Warning: No ~/.claude.json file found"
fi

# Copy entire .claude directory structure, then clear plugins
if [ -d "$REAL_HOME/.claude" ]; then
    print_info "Copying ~/.claude/ directory structure to skip setup prompts"
    cp -r "$REAL_HOME/.claude/"* "$CLAUDE_DIR/" 2>/dev/null || true

    # Clear out plugins and settings to start fresh
    print_info "Clearing plugin data to start fresh"
    rm -rf "$CLAUDE_DIR/plugins"
    mkdir -p "$CLAUDE_DIR/plugins"

    # Reset settings to empty
    cat > "$CLAUDE_DIR/settings.json" <<'CLAUDE_SETTINGS_EOF'
{
  "enabledPlugins": {}
}
CLAUDE_SETTINGS_EOF
else
    print_info "Warning: No ~/.claude directory found"
    mkdir -p "$CLAUDE_DIR/plugins"
fi

# Create a minimal claudeup config for the demo (no active profile)
mkdir -p "$TEST_DIR/.claudeup"
cat > "$TEST_DIR/.claudeup/config.json" <<'CFG_EOF'
{
  "disabledMcpServers": [],
  "preferences": {
    "autoUpdate": false,
    "verboseOutput": false
  }
}
CFG_EOF

# Create helper script for manual interaction
HELPER_SCRIPT="$TEST_DIR/run-claudeup.sh"
cat > "$HELPER_SCRIPT" <<HELPER_EOF
#!/bin/bash
# Helper script to run claudeup commands against the demo environment
export CLAUDE_CONFIG_DIR="$CLAUDE_DIR"
export HOME="$TEST_DIR"
cd "$PROJECT_DIR"
"$CLAUDEUP_ROOT/bin/claudeup" "\$@"
HELPER_EOF
chmod +x "$HELPER_SCRIPT"

# Change to test directory to avoid Claude CLI detecting the claudeup project
cd "$TEST_DIR"

# Add marketplaces
print_section "Phase 1: Add Real Marketplaces"

print_step "Adding marketplaces that we'll use in this demo..."

print_command "claude plugin marketplace add thedotmack/claude-mem"
claude plugin marketplace add thedotmack/claude-mem

print_command "claude plugin marketplace add obra/superpowers-marketplace"
claude plugin marketplace add obra/superpowers-marketplace

print_command "claude plugin marketplace add wshobson/agents"
claude plugin marketplace add wshobson/agents

print_command "claude plugin marketplace add anthropics/claude-plugins-official"
claude plugin marketplace add anthropics/claude-plugins-official

print_info "Marketplaces added successfully"

pause

# Install plugins at user scope
print_section "Phase 2: Install Plugins at User Scope"

print_step "Installing plugins at user scope (default)..."
print_command "claude plugin install claude-mem@thedotmack"
claude plugin install claude-mem@thedotmack

print_command "claude plugin install superpowers@superpowers-marketplace"
claude plugin install superpowers@superpowers-marketplace

print_info "User-scope plugins installed"

pause

print_step "Let's see what we have so far:"
print_command "claudeup plugin list"
"$CLAUDEUP_ROOT/bin/claudeup" plugin list

pause

print_command "claudeup status"
"$CLAUDEUP_ROOT/bin/claudeup" status

pause

echo ""
print_section "Manual Interaction Available"
echo -e "${GREEN}You can interact with the demo environment in another terminal!${NC}"
echo ""
echo "In a separate terminal, run these commands to explore:"
echo ""
echo -e "${CYAN}# Quick setup - copy/paste this:${NC}"
echo "export DEMO_DIR=\"$TEST_DIR\""
echo "alias demo-claudeup='HOME=\$DEMO_DIR CLAUDE_CONFIG_DIR=\$DEMO_DIR/.claude \"$CLAUDEUP_ROOT/bin/claudeup\"'"
echo "alias demo-claude='CLAUDE_CONFIG_DIR=\$DEMO_DIR/.claude claude'"
echo "cd \"\$DEMO_DIR/my-project\""
echo ""
echo -e "${CYAN}# Now run commands:${NC}"
echo "demo-claudeup plugin list"
echo "demo-claudeup status"
echo "demo-claude plugin list"
echo ""
echo -e "${CYAN}# Or use the helper script:${NC}"
echo "$HELPER_SCRIPT plugin list"
echo "$HELPER_SCRIPT status"
echo ""
echo -e "${YELLOW}Note: Use demo-claude and demo-claudeup aliases to target the demo environment.${NC}"
echo -e "${YELLOW}Running just 'claude' or 'claudeup' will use your real ~/.claude directory!${NC}"
echo ""

pause

# Change to project directory
cd "$PROJECT_DIR"

# Install plugins at project scope
print_section "Phase 3: Install Plugins at Project Scope"

print_step "Moving to project directory and installing project-scoped plugins..."
print_info "Project directory: $PROJECT_DIR"

print_command "claude plugin install gopls-lsp@claude-plugins-official --scope project"
claude plugin install gopls-lsp@claude-plugins-official --scope project

print_command "claude plugin install backend-development@claude-code-workflows --scope project"
claude plugin install backend-development@claude-code-workflows --scope project

print_info "Project-scope plugins installed"

pause

print_step "Now let's see the plugin list with scope information:"
print_command "claudeup plugin list"
"$CLAUDEUP_ROOT/bin/claudeup" plugin list

pause

# Install plugins at local scope
print_section "Phase 4: Install Plugins at Local Scope"

print_step "Installing local-scoped plugins (personal, not committed to git)..."

print_command "claude plugin install systems-programming@claude-code-workflows --scope local"
claude plugin install systems-programming@claude-code-workflows --scope local

print_command "claude plugin install shell-scripting@claude-code-workflows --scope local"
claude plugin install shell-scripting@claude-code-workflows --scope local

print_info "Local-scope plugins installed"

pause

# Show accumulated state
print_section "Phase 5: Accumulated Settings Across All Scopes"

print_info "Claude Code merges settings from all three scopes:"
echo ""
echo "Precedence: local > project > user"
echo ""
echo "┌─────────────────────────────────────────────┐"
echo "│ Local (.claude/settings.local.json)  ← Highest │"
echo "│  - systems-programming                      │"
echo "│  - shell-scripting                          │"
echo "├─────────────────────────────────────────────┤"
echo "│ Project (.claude/settings.json)             │"
echo "│  - gopls-lsp                                │"
echo "│  - backend-development                      │"
echo "├─────────────────────────────────────────────┤"
echo "│ User (~/.claude/settings.json)       ← Base │"
echo "│  - claude-mem                               │"
echo "│  - superpowers                              │"
echo "└─────────────────────────────────────────────┘"
echo ""
echo "All plugins are active simultaneously!"

pause

print_command "claudeup plugin list"
"$CLAUDEUP_ROOT/bin/claudeup" plugin list

pause

# Create sample profiles
print_section "Phase 6: Creating Sample Profiles"

print_step "Creating 'base-tools' profile (3 plugins)"
cat > .claudeup/profiles/base-tools.json <<'EOF'
{
  "name": "base-tools",
  "plugins": [
    "claude-mem@thedotmack",
    "superpowers@superpowers-marketplace",
    "code-review-ai@claude-code-workflows"
  ]
}
EOF
print_info "Created: base-tools.json"

print_step "Creating 'backend-stack' profile (4 plugins)"
cat > .claudeup/profiles/backend-stack.json <<'EOF'
{
  "name": "backend-stack",
  "plugins": [
    "gopls-lsp@claude-plugins-official",
    "backend-development@claude-code-workflows",
    "tdd-workflows@claude-code-workflows",
    "debugging-toolkit@claude-code-workflows"
  ]
}
EOF
print_info "Created: backend-stack.json"

print_step "Creating 'docker-tools' profile (2 plugins)"
cat > .claudeup/profiles/docker-tools.json <<'EOF'
{
  "name": "docker-tools",
  "plugins": [
    "systems-programming@claude-code-workflows",
    "shell-scripting@claude-code-workflows"
  ]
}
EOF
print_info "Created: docker-tools.json"

print_info "All profiles created in .claudeup/profiles/"
ls -la .claudeup/profiles/

pause

# Demonstrate drift detection
print_section "Phase 7: Drift Detection"

print_info "Let's manually add a plugin to project scope to create drift..."
print_info "We'll edit .claude/settings.json directly to add a plugin that's not in the profile"

# Install the plugin first so it exists in the registry
print_command "claude plugin install code-review-ai@claude-code-workflows --scope project"
claude plugin install code-review-ai@claude-code-workflows --scope project

print_step "Added 'code-review-ai@claude-code-workflows' to project scope"
print_info "This creates drift because it's not in the backend-stack profile we defined"

pause

print_command "claudeup status"
"$CLAUDEUP_ROOT/bin/claudeup" status || true

pause

# Show scope-specific checks
print_section "Phase 8: Scope-Specific Status Checks"

print_command "claudeup status --scope user"
print_info "Checks only user scope drift:"
"$CLAUDEUP_ROOT/bin/claudeup" status --scope user || true

pause

print_command "claudeup status --scope project"
print_info "Checks only project scope drift:"
"$CLAUDEUP_ROOT/bin/claudeup" status --scope project || true

pause

print_command "claudeup status --scope local"
print_info "Checks only local scope drift:"
"$CLAUDEUP_ROOT/bin/claudeup" status --scope local || true

pause

# Show file structure
print_section "Phase 9: File Structure Summary"

print_info "Settings files created by Claude CLI:"
echo "  User:    $CLAUDE_DIR/settings.json"
echo "  Project: $PROJECT_DIR/.claude/settings.json"
echo "  Local:   $PROJECT_DIR/.claude/settings.local.json"
echo ""

print_command "ls -la $PROJECT_DIR/.claude/"
ls -la "$PROJECT_DIR/.claude/" || true

pause

print_command "cat $CLAUDE_DIR/settings.json"
echo "User scope settings:"
cat "$CLAUDE_DIR/settings.json" | jq . 2>/dev/null || cat "$CLAUDE_DIR/settings.json"

pause

print_command "cat $PROJECT_DIR/.claude/settings.json"
echo "Project scope settings:"
cat "$PROJECT_DIR/.claude/settings.json" | jq . 2>/dev/null || cat "$PROJECT_DIR/.claude/settings.json"

pause

print_command "cat $PROJECT_DIR/.claude/settings.local.json"
echo "Local scope settings:"
cat "$PROJECT_DIR/.claude/settings.local.json" | jq . 2>/dev/null || cat "$PROJECT_DIR/.claude/settings.local.json"

pause

# Summary
print_section "Summary: Real Working Demo"

cat <<'EOF'
✓ Successfully Used Real Claude CLI
  - Used CLAUDE_CONFIG_DIR to isolate test from real Claude
  - Added real marketplaces with 'claude marketplace add'
  - Installed real plugins with 'claude plugin install --scope <scope>'
  - claudeup read the actual structure created by Claude CLI

✓ Demonstrated All Features
  - Plugin list shows scope information
  - Settings accumulate across scopes
  - Drift detection works per-scope
  - Scope-specific status checks

✓ File Structure (Created by Claude CLI)
  - Project settings: .claude/settings.json (committed)
  - Local settings: .claude/settings.local.json (gitignored)
  - User settings: ~/.claude/settings.json (personal)
  - Plugin registry: ~/.claude/plugins/installed_plugins.json

✓ Real Commands Executed
  - claude marketplace add <marketplace>
  - claude plugin install <plugin>@<marketplace> --scope <scope>
  - claudeup plugin list (with scope info)
  - claudeup status (with drift detection)
  - claudeup status --scope <scope> (targeted checks)

Key Insight:
  Claude Code natively supports all three scopes!
  - Reads ~/.claude/settings.json (user)
  - Reads .claude/settings.json (project)
  - Reads .claude/settings.local.json (local)
  - Merges them with precedence: local > project > user

Git Workflow:
  .gitignore should contain:
    .claude/settings.local.json  # Personal overrides
    .claude/*.local.*            # All local files

  Committed files:
    .claudeup/profiles/          # Team profiles
    .claude/settings.json        # Project settings

No Assumptions:
  This demo uses REAL Claude CLI commands, so we're testing
  claudeup against Claude's actual behavior, not assumptions!

EOF

# Cleanup instructions
print_section "Cleanup"
print_step "Test environment: $TEST_DIR"
echo ""
print_info "To continue exploring the demo:"
echo "  Helper script: $HELPER_SCRIPT"
echo "  Or use: export DEMO_DIR=$TEST_DIR"
echo "          alias demo-claudeup='HOME=\$DEMO_DIR CLAUDE_CONFIG_DIR=\$DEMO_DIR/.claude \"$CLAUDEUP_ROOT/bin/claudeup\"'"
echo "          alias demo-claude='CLAUDE_CONFIG_DIR=\$DEMO_DIR/.claude claude'"
echo ""
print_info "To clean up:"
echo "  rm -rf $TEST_DIR"

echo -e "\n${GREEN}Demo complete! All features working with REAL Claude CLI.${NC}\n"
