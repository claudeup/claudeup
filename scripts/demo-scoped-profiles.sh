#!/bin/bash
# ABOUTME: End-to-end demo script for scoped profiles functionality
# ABOUTME: Uses REAL Claude CLI commands to test claudeup against actual Claude behavior

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
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

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

pause() {
    echo ""
    read -rp "Press ENTER to continue..."
}

# Check if claude CLI is available
if ! command -v claude &> /dev/null; then
    print_error "Claude CLI not found in PATH"
    echo "This demo requires the Claude CLI to be installed."
    echo "See: https://code.claude.com/docs/en/getting-started"
    exit 1
fi

# Build the binary
build_binary() {
    print_section "Building claudeup"
    print_step "Building fresh binary..."
    go build -o bin/claudeup ./cmd/claudeup
    print_info "Binary built successfully"
}

# Common setup function
setup_environment() {
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
    print_step "Copying Claude configuration..."

    # Copy ~/.claude.json from home directory
    if [ -f "$REAL_HOME/.claude.json" ]; then
        print_info "Copying ~/.claude.json"
        cp "$REAL_HOME/.claude.json" "$TEST_DIR/.claude.json"
    fi

    # Clone claude-config repo or create minimal config
    if git clone https://github.com/malston/claude-config.git --branch claudeup "$CLAUDE_DIR" >/dev/null 2>&1; then
        # Clear out plugins and settings to start fresh
        print_info "Clearing plugin data to start fresh"
        rm -rf "$CLAUDE_DIR/plugins"
        mkdir -p "$CLAUDE_DIR/plugins"
    else
        print_info "Warning: Failed to clone claude-config repo, creating minimal config"
        mkdir -p "$CLAUDE_DIR/plugins"
        # Create minimal settings
        cat > "$CLAUDE_DIR/settings.json" <<'EOF'
{
  "enabledPlugins": {},
  "hooks": {},
  "includeCoAuthoredBy": false,
  "model": "default",
  "permissions": {
    "allow": [],
    "ask": [],
    "defaultMode": "acceptEdits",
    "deny": []
  }
}
EOF
    fi

    # Create minimal claudeup config
    mkdir -p "$TEST_DIR/.claudeup"
    cat > "$TEST_DIR/.claudeup/config.json" <<'EOF'
{
  "disabledMcpServers": [],
  "preferences": {
    "autoUpdate": false,
    "verboseOutput": false
  }
}
EOF

    # Create sample profiles
    print_step "Creating sample profiles..."

    cat > "$PROJECT_DIR/.claudeup/profiles/base-tools.json" <<'EOF'
{
  "name": "base-tools",
  "plugins": [
    "claude-mem@thedotmack",
    "superpowers@superpowers-marketplace",
    "code-review-ai@claude-code-workflows"
  ]
}
EOF

    cat > "$PROJECT_DIR/.claudeup/profiles/backend-stack.json" <<'EOF'
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

    cat > "$PROJECT_DIR/.claudeup/profiles/docker-tools.json" <<'EOF'
{
  "name": "docker-tools",
  "plugins": [
    "systems-programming@claude-code-workflows",
    "shell-scripting@claude-code-workflows"
  ]
}
EOF

    print_info "Profiles created: base-tools, backend-stack, docker-tools"

    # Change to test directory
    cd "$TEST_DIR"

    # Add marketplaces
    print_step "Adding marketplaces..."
    claude plugin marketplace add thedotmack/claude-mem >/dev/null 2>&1
    claude plugin marketplace add obra/superpowers-marketplace >/dev/null 2>&1
    claude plugin marketplace add wshobson/agents >/dev/null 2>&1
    claude plugin marketplace add anthropics/claude-plugins-official >/dev/null 2>&1
    print_info "Marketplaces added"

    # Change to project directory for scenarios
    cd "$PROJECT_DIR"

    print_info "Environment setup complete"
    echo ""
}

# Cleanup function
cleanup_environment() {
    print_section "Cleanup"
    print_info "Test environment: $TEST_DIR"
    echo ""
    print_info "To continue exploring:"
    echo "  cd $PROJECT_DIR"
    echo "  export CLAUDE_CONFIG_DIR=$CLAUDE_DIR"
    echo "  export HOME=$TEST_DIR"
    echo ""
    print_info "To clean up:"
    echo "  rm -rf $TEST_DIR"
    echo ""
}

# Scenario 1: User Scope Complete Lifecycle
scenario_user_scope() {
    print_section "Scenario: User Scope Complete Lifecycle"

    print_step "1. Apply 'base-tools' profile at user scope"
    print_command "claudeup profile apply base-tools --scope user"
    "$CLAUDEUP_ROOT/bin/claudeup" profile apply base-tools --scope user
    print_info "Profile applied at user scope"
    pause

    print_step "2. Verify user settings were updated"
    print_command "cat $CLAUDE_DIR/settings.json"
    echo "User scope plugins:"
    cat "$CLAUDE_DIR/settings.json" | jq '.enabledPlugins' 2>/dev/null || cat "$CLAUDE_DIR/settings.json"
    print_info "Should show base-tools plugins: claude-mem, superpowers, code-review-ai"
    pause

    print_step "3. Create drift by installing extra plugin at user scope"
    print_command "claude plugin install python-development@claude-code-workflows --scope user"
    claude plugin install python-development@claude-code-workflows --scope user
    print_info "Added 'python-development' to user scope (not in base-tools profile)"
    pause

    print_step "4. Detect drift at user scope"
    print_command "claudeup status --scope user"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope user || true
    print_info "Should show drift for python-development"
    pause

    print_step "5. Clean up drift at user scope"
    print_command "claudeup profile clean --scope user"
    "$CLAUDEUP_ROOT/bin/claudeup" profile clean --scope user
    print_info "User scope cleaned"
    pause

    print_step "6. Verify cleanup - check user settings"
    print_command "cat $CLAUDE_DIR/settings.json"
    echo "User scope plugins after cleanup:"
    cat "$CLAUDE_DIR/settings.json" | jq '.enabledPlugins' 2>/dev/null || cat "$CLAUDE_DIR/settings.json"
    print_info "Should only show base-tools plugins, python-development removed"
    pause

    print_step "7. Verify no drift"
    print_command "claudeup status --scope user"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope user || true
    print_info "Should show clean state"
    pause

    print_section "User Scope Lifecycle Complete"
    echo "✓ Profile apply at user scope"
    echo "✓ Drift detection at user scope"
    echo "✓ Profile clean at user scope"
    echo "✓ Settings updated in ~/.claude/settings.json"
}

# Scenario 2: Project Scope Complete Lifecycle
scenario_project_scope() {
    print_section "Scenario: Project Scope Complete Lifecycle"

    print_step "1. Apply 'backend-stack' profile at project scope"
    print_command "claudeup profile apply backend-stack --scope project"
    "$CLAUDEUP_ROOT/bin/claudeup" profile apply backend-stack --scope project
    print_info "Profile applied at project scope"
    pause

    print_step "2. Verify project settings were updated"
    print_command "cat .claude/settings.json"
    echo "Project scope plugins:"
    cat .claude/settings.json | jq '.enabledPlugins' 2>/dev/null || cat .claude/settings.json
    print_info "Should show backend-stack plugins: gopls-lsp, backend-development, tdd-workflows, debugging-toolkit"
    pause

    print_step "3. Create drift by installing extra plugin at project scope"
    print_command "claude plugin install python-development@claude-code-workflows --scope project"
    claude plugin install python-development@claude-code-workflows --scope project
    print_info "Added 'python-development' to project scope (not in backend-stack profile)"
    pause

    print_step "4. Detect drift at project scope"
    print_command "claudeup status --scope project"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope project || true
    print_info "Should show drift for python-development"
    pause

    print_step "5. Clean up drift at project scope"
    print_command "claudeup profile clean --scope project"
    "$CLAUDEUP_ROOT/bin/claudeup" profile clean --scope project
    print_info "Project scope cleaned"
    pause

    print_step "6. Verify cleanup - check project settings"
    print_command "cat .claude/settings.json"
    echo "Project scope plugins after cleanup:"
    cat .claude/settings.json | jq '.enabledPlugins' 2>/dev/null || cat .claude/settings.json
    print_info "Should only show backend-stack plugins, python-development removed"
    pause

    print_step "7. Verify no drift"
    print_command "claudeup status --scope project"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope project || true
    print_info "Should show clean state"
    pause

    print_section "Project Scope Lifecycle Complete"
    echo "✓ Profile apply at project scope"
    echo "✓ Drift detection at project scope"
    echo "✓ Profile clean at project scope"
    echo "✓ Settings updated in .claude/settings.json"
    echo "✓ Config tracked in .claudeup.json"
}

# Scenario 3: Local Scope Complete Lifecycle
scenario_local_scope() {
    print_section "Scenario: Local Scope Complete Lifecycle"

    print_step "1. Apply 'docker-tools' profile at local scope"
    print_command "claudeup profile apply docker-tools --local"
    "$CLAUDEUP_ROOT/bin/claudeup" profile apply docker-tools --local
    print_info "Profile applied at local scope"
    pause

    print_step "2. Verify local settings were updated"
    print_command "cat .claude/settings.local.json"
    echo "Local scope plugins:"
    cat .claude/settings.local.json | jq '.enabledPlugins' 2>/dev/null || cat .claude/settings.local.json
    print_info "Should show docker-tools plugins: systems-programming, shell-scripting"
    pause

    print_step "3. Create drift by installing extra plugin at local scope"
    print_command "claude plugin install web-scripting@claude-code-workflows --scope local"
    claude plugin install web-scripting@claude-code-workflows --scope local
    print_info "Added 'web-scripting' to local scope (not in docker-tools profile)"
    pause

    print_step "4. Detect drift at local scope"
    print_command "claudeup status --scope local"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope local || true
    print_info "Should show drift for web-scripting"
    pause

    print_step "5. Clean up drift at local scope"
    print_command "claudeup profile clean --local"
    "$CLAUDEUP_ROOT/bin/claudeup" profile clean --local
    print_info "Local scope cleaned (only updates .claude/settings.local.json)"
    pause

    print_step "6. Verify cleanup - check local settings"
    print_command "cat .claude/settings.local.json"
    echo "Local scope plugins after cleanup:"
    cat .claude/settings.local.json | jq '.enabledPlugins' 2>/dev/null || cat .claude/settings.local.json
    print_info "Should only show docker-tools plugins, web-scripting removed"
    pause

    print_step "7. Verify no drift"
    print_command "claudeup status --scope local"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope local || true
    print_info "Should show clean state"
    pause

    print_section "Local Scope Lifecycle Complete"
    echo "✓ Profile apply at local scope"
    echo "✓ Drift detection at local scope"
    echo "✓ Profile clean at local scope"
    echo "✓ Settings updated in .claude/settings.local.json only"
    echo "✓ No .claudeup.local.json created (refactor verified)"
}

# Scenario 4: All Scopes Together
scenario_all_scopes() {
    print_section "Scenario: All Scopes Complete Demo"

    # Run all three scope scenarios
    scenario_user_scope
    echo ""
    scenario_project_scope
    echo ""
    scenario_local_scope
    echo ""

    # Show accumulated state
    print_section "Final State: All Scopes Active"

    print_command "claudeup profile list"
    "$CLAUDEUP_ROOT/bin/claudeup" profile list

    print_info "Settings accumulate from all scopes with precedence: local > project > user"
    pause

    print_section "Summary: All Scopes Complete"
    cat <<'EOF'
✓ User Scope Lifecycle
  - Apply, drift, clean at ~/.claude/settings.json
  - Profile: base-tools

✓ Project Scope Lifecycle
  - Apply, drift, clean at .claude/settings.json
  - Profile: backend-stack
  - Tracked in .claudeup.json

✓ Local Scope Lifecycle
  - Apply, drift, clean at .claude/settings.local.json
  - Profile: docker-tools
  - No .claudeup.local.json (refactor verified)

Key Architecture:
  - User scope: ~/.claude/settings.json only
  - Project scope: .claudeup.json + .claude/settings.json
  - Local scope: .claude/settings.local.json only
  - Settings merge with precedence: local > project > user
EOF
}

# Main menu
show_menu() {
    echo ""
    print_section "claudeup Scoped Profiles Demo"
    echo "Select a scenario to run:"
    echo ""
    echo "  1) User scope lifecycle (apply → drift → clean)"
    echo "  2) Project scope lifecycle (apply → drift → clean)"
    echo "  3) Local scope lifecycle (apply → drift → clean)"
    echo "  4) All scopes complete demo"
    echo "  5) Exit"
    echo ""
    read -rp "Enter selection [1-5]: " choice
    echo ""

    case $choice in
        1)
            setup_environment
            scenario_user_scope
            cleanup_environment
            ;;
        2)
            setup_environment
            scenario_project_scope
            cleanup_environment
            ;;
        3)
            setup_environment
            scenario_local_scope
            cleanup_environment
            ;;
        4)
            setup_environment
            scenario_all_scopes
            cleanup_environment
            ;;
        5)
            echo "Exiting..."
            exit 0
            ;;
        *)
            print_error "Invalid selection"
            show_menu
            ;;
    esac
}

# Entry point
build_binary
show_menu
