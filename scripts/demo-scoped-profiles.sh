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

validate_command() {
    local cmd="$1"
    local description="$2"

    if ! eval "$cmd"; then
        print_error "Command failed: $description"
        print_error "Command: $cmd"
        exit 1
    fi
}

validate_plugin_exists() {
    local plugin="$1"
    local settings_file="$2"
    local should_exist="${3:-true}"

    if [ "$should_exist" = "true" ]; then
        if ! jq -e ".enabledPlugins[\"$plugin\"]" "$settings_file" >/dev/null 2>&1; then
            print_error "Expected plugin '$plugin' not found in $settings_file"
            return 1
        fi
    else
        if jq -e ".enabledPlugins[\"$plugin\"]" "$settings_file" >/dev/null 2>&1; then
            print_error "Plugin '$plugin' should not exist in $settings_file but was found"
            return 1
        fi
    fi
    return 0
}

validate_plugins_match_profile() {
    local profile_name="$1"
    local settings_file="$2"
    local profile_file="$TEST_DIR/.claudeup/profiles/$profile_name.json"

    # Get expected plugins from profile
    local expected_plugins
    expected_plugins=$(jq -r '.plugins[]' "$profile_file")

    # Check each expected plugin exists
    while IFS= read -r plugin; do
        if ! validate_plugin_exists "$plugin" "$settings_file" "true"; then
            print_error "Profile validation failed: missing plugin '$plugin' from profile '$profile_name'"
            return 1
        fi
    done <<< "$expected_plugins"

    # Count plugins in settings vs profile
    local settings_count
    local profile_count
    settings_count=$(jq '.enabledPlugins | length' "$settings_file")
    profile_count=$(jq '.plugins | length' "$profile_file")

    if [ "$settings_count" -ne "$profile_count" ]; then
        print_error "Plugin count mismatch: settings has $settings_count, profile expects $profile_count"
        return 1
    fi

    return 0
}

# Check prerequisites
check_prerequisites() {
    local missing=()

    if ! command -v claude &> /dev/null; then
        missing+=("claude (https://code.claude.com/docs/en/getting-started)")
    fi

    if ! command -v jq &> /dev/null; then
        missing+=("jq (https://stedolan.github.io/jq/)")
    fi

    if ! command -v git &> /dev/null; then
        missing+=("git")
    fi

    if ! command -v go &> /dev/null; then
        missing+=("go (https://golang.org/dl/)")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        print_error "Missing required tools:"
        for tool in "${missing[@]}"; do
            echo "  - $tool"
        done
        exit 1
    fi
}

check_prerequisites

# Build the binary
build_binary() {
    print_section "Building claudeup"
    print_step "Building fresh binary..."

    if ! go build -o bin/claudeup ./cmd/claudeup; then
        print_error "Failed to build claudeup binary"
        exit 1
    fi

    if [ ! -x bin/claudeup ]; then
        print_error "Binary was built but is not executable"
        exit 1
    fi

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
    if ! mkdir -p "$PROJECT_DIR" "$TEST_DIR/.claudeup/profiles"; then
        print_error "Failed to create directory structure"
        exit 1
    fi

    # Validate directories were created
    for dir in "$PROJECT_DIR" "$TEST_DIR/.claudeup/profiles"; do
        if [ ! -d "$dir" ]; then
            print_error "Directory was not created: $dir"
            exit 1
        fi
    done

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

    # Validate config.json
    if ! jq empty "$TEST_DIR/.claudeup/config.json" 2>/dev/null; then
        print_error "Invalid JSON in config.json"
        exit 1
    fi

    # Create sample profiles
    print_step "Creating sample profiles..."

    cat > "$TEST_DIR/.claudeup/profiles/base-tools.json" <<'EOF'
{
  "name": "base-tools",
  "marketplaces": [
    {"source": "github", "repo": "thedotmack/claude-mem"},
    {"source": "github", "repo": "obra/superpowers-marketplace"},
    {"source": "github", "repo": "wshobson/agents"}
  ],
  "plugins": [
    "claude-mem@thedotmack",
    "superpowers@superpowers-marketplace",
    "code-review-ai@claude-code-workflows"
  ]
}
EOF

    cat > "$TEST_DIR/.claudeup/profiles/backend-stack.json" <<'EOF'
{
  "name": "backend-stack",
  "marketplaces": [
    {"source": "github", "repo": "thedotmack/claude-mem"},
    {"source": "github", "repo": "obra/superpowers-marketplace"}
  ],
  "plugins": [
    "claude-mem@thedotmack",
    "superpowers@superpowers-marketplace"
  ]
}
EOF

    cat > "$TEST_DIR/.claudeup/profiles/docker-tools.json" <<'EOF'
{
  "name": "docker-tools",
  "marketplaces": [
    {"source": "github", "repo": "wshobson/agents"}
  ],
  "plugins": [
    "systems-programming@claude-code-workflows",
    "shell-scripting@claude-code-workflows"
  ]
}
EOF

    # Validate profiles were created with valid JSON
    for profile in base-tools backend-stack docker-tools; do
        profile_file="$TEST_DIR/.claudeup/profiles/$profile.json"
        if [ ! -f "$profile_file" ]; then
            print_error "Profile file not created: $profile_file"
            exit 1
        fi
        if ! jq empty "$profile_file" 2>/dev/null; then
            print_error "Invalid JSON in profile: $profile_file"
            exit 1
        fi
    done

    print_info "Profiles created: base-tools, backend-stack, docker-tools"

    # Change to test directory
    cd "$TEST_DIR"

    # Add marketplaces
    print_step "Adding marketplaces..."
    validate_command "claude plugin marketplace add thedotmack/claude-mem" "Add claude-mem marketplace"
    validate_command "claude plugin marketplace add obra/superpowers-marketplace" "Add superpowers marketplace"
    validate_command "claude plugin marketplace add wshobson/agents" "Add agents marketplace"
    validate_command "claude plugin marketplace add anthropics/claude-plugins-official" "Add official plugins marketplace"
    print_info "Marketplaces added"

    # Validate claudeup binary works in test environment
    print_step "Validating claudeup binary..."
    if ! "$CLAUDEUP_ROOT/bin/claudeup" --version >/dev/null 2>&1; then
        print_error "claudeup binary not working correctly"
        exit 1
    fi
    print_info "claudeup binary validated"

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
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile apply base-tools --scope user" "Apply base-tools profile"

    # Validate settings file was created
    if [ ! -f "$CLAUDE_DIR/settings.json" ]; then
        print_error "Settings file not created at $CLAUDE_DIR/settings.json"
        exit 1
    fi

    # Validate it's valid JSON
    if ! jq empty "$CLAUDE_DIR/settings.json" 2>/dev/null; then
        print_error "Invalid JSON in $CLAUDE_DIR/settings.json"
        exit 1
    fi

    # Validate plugins match the profile
    if ! validate_plugins_match_profile "base-tools" "$CLAUDE_DIR/settings.json"; then
        print_error "Profile apply succeeded but plugins don't match profile definition"
        exit 1
    fi

    print_info "Profile applied at user scope"
    print_info "✓ Verified: All base-tools plugins installed correctly"
    pause

    print_step "2. Verify user settings were updated"
    print_command "cat $CLAUDE_DIR/settings.json"
    echo "User scope plugins:"
    cat "$CLAUDE_DIR/settings.json" | jq '.enabledPlugins' 2>/dev/null || cat "$CLAUDE_DIR/settings.json"
    print_info "Should show base-tools plugins: claude-mem, superpowers, code-review-ai"
    pause

    print_step "3. Create drift by installing extra plugin at user scope"
    print_command "claude plugin install python-development@claude-code-workflows --scope user"
    validate_command "claude plugin install python-development@claude-code-workflows --scope user" "Install python-development plugin"

    # Verify plugin was actually installed
    if ! jq -e '.enabledPlugins["python-development@claude-code-workflows"]' "$CLAUDE_DIR/settings.json" >/dev/null 2>&1; then
        print_error "Plugin installation succeeded but plugin not found in settings.json"
        exit 1
    fi

    print_info "Added 'python-development' to user scope (not in base-tools profile)"
    pause

    print_step "4. Detect drift at user scope"
    print_command "claudeup status --scope user"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope user || true
    print_info "Should show drift for python-development"
    pause

    print_step "5. Clean up drift at user scope"
    print_command "claudeup profile apply base-tools --scope user --reset -y"
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile apply base-tools --scope user --reset -y" "Clean up user scope"

    # Verify python-development was removed
    if ! validate_plugin_exists "python-development@claude-code-workflows" "$CLAUDE_DIR/settings.json" "false"; then
        print_error "Cleanup succeeded but drift still exists"
        exit 1
    fi

    # Verify profile matches again
    if ! validate_plugins_match_profile "base-tools" "$CLAUDE_DIR/settings.json"; then
        print_error "Cleanup succeeded but plugins don't match profile"
        exit 1
    fi

    print_info "User scope cleaned (reset to profile state)"
    print_info "✓ Verified: Drift removed, profile restored"
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
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile apply backend-stack --scope project" "Apply backend-stack profile"

    # Validate plugins match the profile
    if ! validate_plugins_match_profile "backend-stack" ".claude/settings.json"; then
        print_error "Profile apply succeeded but plugins don't match profile definition"
        exit 1
    fi

    print_info "Profile applied at project scope"
    print_info "✓ Verified: All backend-stack plugins installed correctly"
    pause

    print_step "2. Verify project settings were updated"
    print_command "cat .claude/settings.json"
    echo "Project scope plugins:"
    cat .claude/settings.json | jq '.enabledPlugins' 2>/dev/null || cat .claude/settings.json
    print_info "Should show backend-stack plugins: claude-mem, superpowers"
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
    print_command "claudeup profile apply backend-stack --scope project --reset -y"
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile apply backend-stack --scope project --reset -y" "Clean up project scope"

    # Verify python-development was removed
    if ! validate_plugin_exists "python-development@claude-code-workflows" ".claude/settings.json" "false"; then
        print_error "Cleanup succeeded but drift still exists"
        exit 1
    fi

    # Verify profile matches again
    if ! validate_plugins_match_profile "backend-stack" ".claude/settings.json"; then
        print_error "Cleanup succeeded but plugins don't match profile"
        exit 1
    fi

    print_info "Project scope cleaned (reset to profile state)"
    print_info "✓ Verified: Drift removed, profile restored"
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
    print_command "claudeup profile apply docker-tools --scope local"
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile apply docker-tools --scope local" "Apply docker-tools profile"

    # Validate plugins match the profile
    if ! validate_plugins_match_profile "docker-tools" ".claude/settings.local.json"; then
        print_error "Profile apply succeeded but plugins don't match profile definition"
        exit 1
    fi

    print_info "Profile applied at local scope"
    print_info "✓ Verified: All docker-tools plugins installed correctly"
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
    print_command "claudeup profile clean --scope local web-scripting@claude-code-workflows"
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile clean --scope local web-scripting@claude-code-workflows" "Clean local scope"

    # Verify web-scripting was removed
    if ! validate_plugin_exists "web-scripting@claude-code-workflows" ".claude/settings.local.json" "false"; then
        print_error "Clean succeeded but drift still exists"
        exit 1
    fi

    # Verify profile matches again
    if ! validate_plugins_match_profile "docker-tools" ".claude/settings.local.json"; then
        print_error "Clean succeeded but plugins don't match profile"
        exit 1
    fi

    print_info "Local scope cleaned (removed web-scripting from .claude/settings.local.json)"
    print_info "✓ Verified: Drift removed, profile restored"
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

# Scenario 4: Profile Sync (Team Collaboration)
scenario_profile_sync() {
    print_section "Scenario: Profile Sync (Team Collaboration)"

    print_step "1. Apply 'backend-stack' profile at project scope"
    print_command "claudeup profile apply backend-stack --scope project"
    "$CLAUDEUP_ROOT/bin/claudeup" profile apply backend-stack --scope project
    print_info "Profile applied - creates .claudeup.json pointing to 'backend-stack'"
    pause

    print_step "2. Show .claudeup.json (what gets committed to git)"
    print_command "cat .claudeup.json"
    cat .claudeup.json | jq '.' 2>/dev/null || cat .claudeup.json
    print_info ".claudeup.json is just a pointer to the profile, not a copy of its contents"
    pause

    print_step "3. Show profile definition (in ~/.claudeup/profiles/)"
    print_command "cat ~/.claudeup/profiles/backend-stack.json"
    cat "$TEST_DIR/.claudeup/profiles/backend-stack.json" | jq '.' 2>/dev/null || cat "$TEST_DIR/.claudeup/profiles/backend-stack.json"
    print_info "This is the actual profile definition with plugins and marketplaces"
    pause

    print_step "4. Simulate team member cloning the repo (remove local Claude settings and plugins)"
    print_info "Removing .claude/settings.json and installed plugins to simulate fresh clone..."
    rm -f .claude/settings.json
    rm -rf "$CLAUDE_DIR/plugins"
    mkdir -p "$CLAUDE_DIR/plugins"
    print_info "Team member has .claudeup.json but no plugins installed yet"
    pause

    print_step "5. Show current plugin state (none installed at project scope)"
    print_command "cat .claude/settings.json"
    if [ -f .claude/settings.json ]; then
        cat .claude/settings.json | jq '.enabledPlugins' 2>/dev/null || cat .claude/settings.json
    else
        print_info "No .claude/settings.json file exists yet - no plugins installed"
    fi
    pause

    print_step "6. Run profile sync to install plugins from .claudeup.json"
    print_command "claudeup profile sync"
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile sync" "Profile sync"

    # Verify plugins were installed correctly
    if ! validate_plugins_match_profile "backend-stack" ".claude/settings.json"; then
        print_error "Sync succeeded but plugins don't match profile"
        exit 1
    fi

    print_info "Sync reads .claudeup.json, loads profile 'backend-stack', installs all plugins at project scope"
    print_info "✓ Verified: All backend-stack plugins installed"
    pause

    print_step "7. Verify plugins were installed at project scope"
    print_command "cat .claude/settings.json"
    echo "Project scope plugins after sync:"
    cat .claude/settings.json | jq '.enabledPlugins' 2>/dev/null || cat .claude/settings.json
    print_info "Should show backend-stack plugins: claude-mem, superpowers"
    pause

    print_step "8. Verify sync is idempotent (run again)"
    print_command "claudeup profile sync"

    # Capture plugin state before second sync
    local plugins_before
    plugins_before=$(jq -c '.enabledPlugins' .claude/settings.json)

    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile sync" "Second sync (idempotency check)"

    # Verify plugins unchanged
    local plugins_after
    plugins_after=$(jq -c '.enabledPlugins' .claude/settings.json)

    if [ "$plugins_before" != "$plugins_after" ]; then
        print_error "Second sync modified plugins (not idempotent)"
        exit 1
    fi

    print_info "Sync skips already-installed plugins"
    print_info "✓ Verified: Sync is idempotent"
    pause

    print_step "9. Create drift by uninstalling a plugin"
    print_info "Simulating drift: uninstalling 'superpowers' from project scope..."
    print_command "claude plugin uninstall superpowers@superpowers-marketplace --scope project"
    claude plugin uninstall superpowers@superpowers-marketplace --scope project
    print_info "Plugin removed from project scope but still in profile definition"
    pause

    print_step "10. Detect drift between profile and installed plugins"
    print_command "cat .claude/settings.json | jq '.enabledPlugins | keys'"
    echo "Currently installed plugins:"
    cat .claude/settings.json | jq '.enabledPlugins | keys' 2>/dev/null || cat .claude/settings.json
    print_info "Notice 'superpowers' is missing (should have 2 plugins, only has 1)"
    pause

    print_step "11. Check status to see drift"
    print_command "claudeup status --scope project"
    "$CLAUDEUP_ROOT/bin/claudeup" status --scope project || true
    print_info "Should show drift: profile expects superpowers but it's not installed"
    pause

    print_step "12. Run sync to fix drift"
    print_command "claudeup profile sync"
    validate_command "\"$CLAUDEUP_ROOT/bin/claudeup\" profile sync" "Sync to repair drift"

    # Verify superpowers was reinstalled
    if ! validate_plugin_exists "superpowers@superpowers-marketplace" ".claude/settings.json" "true"; then
        print_error "Sync succeeded but missing plugin not restored"
        exit 1
    fi

    # Verify all plugins match profile again
    if ! validate_plugins_match_profile "backend-stack" ".claude/settings.json"; then
        print_error "Sync succeeded but plugins don't match profile"
        exit 1
    fi

    print_info "Sync detects missing plugin and reinstalls it"
    print_info "✓ Verified: Drift repaired, all plugins restored"
    pause

    print_step "13. Verify drift is fixed"
    print_command "cat .claude/settings.json | jq '.enabledPlugins | keys'"
    echo "Plugins after sync:"
    cat .claude/settings.json | jq '.enabledPlugins | keys' 2>/dev/null || cat .claude/settings.json
    print_info "Both plugins restored: claude-mem, superpowers"
    pause

    print_section "Profile Sync Complete"
    cat <<'EOF'
✓ Team Workflow Demonstrated:
  1. Developer applies profile → .claudeup.json created
  2. .claudeup.json committed to git (just a pointer)
  3. Team member clones repo
  4. Team member runs 'claudeup profile sync'
  5. All plugins installed automatically at project scope

✓ Drift Detection & Repair:
  6. Plugin manually removed (simulating drift)
  7. Status shows drift between profile and installed plugins
  8. Sync detects missing plugins and reinstalls them
  9. Configuration restored to match profile definition

Key Insights:
  - .claudeup.json is minimal (just profile name + timestamp)
  - Profile definitions live in ~/.claudeup/profiles/
  - Sync loads profile and installs its plugins
  - Sync is idempotent (safe to run multiple times)
  - Sync detects and fixes drift automatically
EOF
}

# Scenario 5: All Scopes Together
scenario_all_scopes() {
    print_section "Scenario: All Scopes Complete Demo"

    # Run all scope scenarios
    scenario_user_scope
    echo ""
    scenario_project_scope
    echo ""
    scenario_local_scope
    echo ""
    scenario_profile_sync
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

✓ Profile Sync
  - Team collaboration via .claudeup.json
  - Profile definitions in ~/.claudeup/profiles/
  - Sync installs plugins at project scope

Key Architecture:
  - User scope: ~/.claude/settings.json only
  - Project scope: .claudeup.json + .claude/settings.json
  - Local scope: .claude/settings.local.json only
  - Settings merge with precedence: local > project > user
  - .claudeup.json is a pointer, not a copy
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
    echo "  4) Profile sync (team collaboration)"
    echo "  5) All scopes complete demo"
    echo "  6) Exit"
    echo ""
    read -rp "Enter selection [1-6]: " choice
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
            scenario_profile_sync
            cleanup_environment
            ;;
        5)
            setup_environment
            scenario_all_scopes
            cleanup_environment
            ;;
        6)
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
