#!/usr/bin/env bash
# ABOUTME: Demo script for claudeup local management feature
# ABOUTME: Creates isolated environment, installs GSD, and configures with claudeup

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"
}

print_step() {
    echo -e "${GREEN}▶${NC} $1"
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Create isolated demo environment
DEMO_DIR=$(mktemp -d)
export CLAUDE_CONFIG_DIR="$DEMO_DIR/.claude"
export CLAUDEUP_HOME="$DEMO_DIR/.claudeup"

# Build claudeup - handle running from main repo or worktree
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

# Check if we're already in a worktree (has cmd/claudeup but no .worktrees)
if [[ -d "$REPO_DIR/cmd/claudeup" && ! -d "$REPO_DIR/.worktrees" ]]; then
    # Running from within worktree
    BUILD_DIR="$REPO_DIR"
elif [[ -d "$REPO_DIR/.worktrees/local-management" ]]; then
    # Running from main repo
    BUILD_DIR="$REPO_DIR/.worktrees/local-management"
else
    # Fallback to current repo dir
    BUILD_DIR="$REPO_DIR"
fi
CLAUDEUP_BIN="$BUILD_DIR/bin/claudeup"

cleanup() {
    print_header "Cleanup"
    print_info "Demo directory was: $DEMO_DIR"
    read -p "Remove demo directory? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$DEMO_DIR"
        print_success "Cleaned up demo directory"
    else
        print_info "Demo directory preserved at: $DEMO_DIR"
        print_info "  CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR"
        print_info "  CLAUDEUP_HOME=$CLAUDEUP_HOME"
    fi
}

trap cleanup EXIT

print_header "claudeup Local Management Demo"

print_info "Demo directory: $DEMO_DIR"
print_info "CLAUDE_CONFIG_DIR: $CLAUDE_CONFIG_DIR"
print_info "CLAUDEUP_HOME: $CLAUDEUP_HOME"
echo

# Step 1: Build claudeup if needed
print_header "Step 1: Build claudeup"

if [[ ! -f "$CLAUDEUP_BIN" ]]; then
    print_step "Building claudeup from local-management branch..."
    (cd "$BUILD_DIR" && go build -o bin/claudeup ./cmd/claudeup)
    print_success "Built claudeup"
else
    print_info "Using existing claudeup binary"
fi

# Step 2: Create initial directory structure
print_header "Step 2: Create Directory Structure"

print_step "Creating $CLAUDE_CONFIG_DIR..."
mkdir -p "$CLAUDE_CONFIG_DIR"
print_success "Created CLAUDE_CONFIG_DIR"

print_step "Creating $CLAUDEUP_HOME/profiles..."
mkdir -p "$CLAUDEUP_HOME/profiles"
print_success "Created CLAUDEUP_HOME"

# Step 3: Install GSD
print_header "Step 3: Install GSD"

print_step "Running npx get-shit-done-cc..."
echo
npx get-shit-done-cc --claude --global --config-dir "$CLAUDE_CONFIG_DIR" --force-statusline
echo
print_success "GSD installed"

# Step 4: Import GSD files to .library using claudeup
print_header "Step 4: Import GSD to .library"

print_info "GSD installs directly to active directories."
print_info "Using 'claudeup local import-all' to move to .library and create symlinks..."
echo

print_step "Importing all GSD items..."
"$CLAUDEUP_BIN" --claude-dir "$CLAUDE_CONFIG_DIR" local import-all "gsd-*" gsd
echo

# Count imported items
GSD_AGENTS=$(ls "$CLAUDE_CONFIG_DIR/.library/agents/" 2>/dev/null | grep -c "gsd-" || echo "0")
GSD_COMMANDS=$(ls "$CLAUDE_CONFIG_DIR/.library/commands/gsd/" 2>/dev/null | wc -l | tr -d ' ')
GSD_HOOKS=$(ls "$CLAUDE_CONFIG_DIR/.library/hooks/" 2>/dev/null | grep -c "gsd-" || echo "0")

print_success "Imported $GSD_AGENTS GSD agents, $GSD_COMMANDS GSD commands, $GSD_HOOKS GSD hooks"

# Step 5: Verify symlinks were created
print_header "Step 5: Verify Symlinks"

print_step "Checking agent symlinks..."
AGENT_SYMLINKS=$(ls -la "$CLAUDE_CONFIG_DIR/agents/" 2>/dev/null | grep "gsd-" | grep -c "^l" || echo "0")
if [[ "$AGENT_SYMLINKS" -gt 0 ]]; then
    print_success "Found $AGENT_SYMLINKS agent symlinks"
    ls -la "$CLAUDE_CONFIG_DIR/agents/" | grep "gsd-" | head -3
else
    print_error "No agent symlinks created!"
    exit 1
fi
echo

print_step "Checking command symlinks..."
if [[ -L "$CLAUDE_CONFIG_DIR/commands/gsd" ]]; then
    print_success "GSD commands directory symlinked"
    ls -la "$CLAUDE_CONFIG_DIR/commands/" | grep gsd
else
    print_error "GSD commands not symlinked!"
    exit 1
fi
echo

print_step "Checking hook symlinks..."
HOOK_SYMLINKS=$(ls -la "$CLAUDE_CONFIG_DIR/hooks/" 2>/dev/null | grep "gsd-" | grep -c "^l" || echo "0")
if [[ "$HOOK_SYMLINKS" -gt 0 ]]; then
    print_success "Found $HOOK_SYMLINKS hook symlinks"
    ls -la "$CLAUDE_CONFIG_DIR/hooks/" | grep "gsd-"
else
    print_error "No hook symlinks created!"
    exit 1
fi
echo

# Step 6: Check enabled.json
print_header "Step 6: Check enabled.json"

print_step "Contents of enabled.json:"
echo
cat "$CLAUDE_CONFIG_DIR/enabled.json"
echo

# Step 7: List only enabled items
print_header "Step 7: List Enabled Items"

print_step "Running: claudeup local list --enabled"
echo
"$CLAUDEUP_BIN" --claude-dir "$CLAUDE_CONFIG_DIR" local list --enabled
echo

# Step 8: Save as a profile
print_header "Step 8: Save as Profile"

print_step "Saving current state as 'gsd-demo' profile..."
"$CLAUDEUP_BIN" --claude-dir "$CLAUDE_CONFIG_DIR" profile save gsd-demo --description "GSD demo profile with local items"
print_success "Profile saved"
echo

print_step "Profile contents:"
cat "$CLAUDEUP_HOME/profiles/gsd-demo.json" | head -50
echo "..."
echo

# Step 9: Demonstrate 'install' command with external source
print_header "Step 9: Install External Agent"

print_info "Now demonstrating 'claudeup local install' (different from import)..."
print_info "- import: moves files from active dirs to .library"
print_info "- install: copies files from external sources to .library"
echo

# Create a custom agent in a temporary external location
EXTERNAL_AGENTS="$DEMO_DIR/my-custom-agents"
mkdir -p "$EXTERNAL_AGENTS"

print_step "Creating custom agent in external directory..."
cat > "$EXTERNAL_AGENTS/demo-agent.md" << 'EOF'
# Demo Agent

This is a custom agent created to demonstrate the `claudeup local install` command.

## Purpose
Show how to install agents from external sources (git repos, downloads, etc.)

## Usage
This agent was installed using:
```bash
claudeup local install agents /path/to/my-custom-agents/demo-agent.md
```
EOF
print_success "Created demo-agent.md in $EXTERNAL_AGENTS"
echo

print_step "Installing custom agent from external source..."
"$CLAUDEUP_BIN" --claude-dir "$CLAUDE_CONFIG_DIR" local install agents "$EXTERNAL_AGENTS/demo-agent.md"
echo

print_step "Verifying installation..."
# Check that source file still exists (install copies, doesn't move)
if [[ -f "$EXTERNAL_AGENTS/demo-agent.md" ]]; then
    print_success "Source file still exists (install copies, doesn't move)"
else
    print_error "Source file was removed! (unexpected)"
    exit 1
fi

# Check that file was copied to .library
if [[ -f "$CLAUDE_CONFIG_DIR/.library/agents/demo-agent.md" ]]; then
    print_success "File copied to .library/agents/"
else
    print_error "File not found in .library!"
    exit 1
fi

# Check that symlink was created
if [[ -L "$CLAUDE_CONFIG_DIR/agents/demo-agent.md" ]]; then
    print_success "Symlink created in agents/"
    ls -la "$CLAUDE_CONFIG_DIR/agents/demo-agent.md"
else
    print_error "Symlink not created!"
    exit 1
fi
echo

# Step 10: Demonstrate installing a directory (agent group)
print_header "Step 10: Install Agent Group from Directory"

print_step "Creating agent group in external directory..."
AGENT_GROUP="$EXTERNAL_AGENTS/my-agents"
mkdir -p "$AGENT_GROUP"

cat > "$AGENT_GROUP/planner.md" << 'EOF'
# Planner Agent
Part of the custom agent group.
EOF

cat > "$AGENT_GROUP/executor.md" << 'EOF'
# Executor Agent
Part of the custom agent group.
EOF

print_success "Created agent group with 2 agents"
echo

print_step "Installing agent group directory..."
"$CLAUDEUP_BIN" --claude-dir "$CLAUDE_CONFIG_DIR" local install agents "$AGENT_GROUP"
echo

print_step "Verifying agent group installation..."
if [[ -d "$CLAUDE_CONFIG_DIR/.library/agents/my-agents" ]]; then
    print_success "Agent group directory copied to .library/"
    ls -la "$CLAUDE_CONFIG_DIR/.library/agents/my-agents/"
else
    print_error "Agent group not found in .library!"
    exit 1
fi

# Check that symlink was created for agent group
if [[ -L "$CLAUDE_CONFIG_DIR/agents/my-agents" ]]; then
    print_success "Symlink created for agent group"
    ls -la "$CLAUDE_CONFIG_DIR/agents/my-agents"
else
    print_error "Agent group symlink not created!"
    exit 1
fi
echo

# Step 11: Show final state with all items
print_header "Step 11: Final State"

print_step "All enabled local items:"
echo
"$CLAUDEUP_BIN" --claude-dir "$CLAUDE_CONFIG_DIR" local list --enabled
echo

print_step "Updated enabled.json:"
echo
cat "$CLAUDE_CONFIG_DIR/enabled.json"
echo

# Summary
print_header "Demo Complete!"

CUSTOM_AGENTS=0
if [ -d "$CLAUDE_CONFIG_DIR/.library/agents" ]; then
    CUSTOM_AGENTS=$(find "$CLAUDE_CONFIG_DIR/.library/agents" -name '*.md' \( -name 'demo-agent*' -o -path '*/my-agents/*' \) | wc -l | tr -d ' ')
fi

echo "Summary:"
echo "  - GSD agents enabled: $GSD_AGENTS"
echo "  - GSD commands enabled: $GSD_COMMANDS files"
echo "  - GSD hooks enabled: $GSD_HOOKS"
echo "  - Custom agents installed: $CUSTOM_AGENTS (demo-agent + 2 in my-agents group)"
echo "  - Profile saved: gsd-demo"
echo
echo "Commands demonstrated:"
echo "  ✓ claudeup local import-all  - Move files from active dirs to .library"
echo "  ✓ claudeup local install     - Copy external files to .library"
echo "  ✓ claudeup local list        - View enabled/disabled items"
echo "  ✓ claudeup profile save      - Save configuration as profile"
echo
echo "Key differences:"
echo "  • import: MOVES files (from active dirs), removes source"
echo "  • install: COPIES files (from anywhere), keeps source"
echo
echo "The demo environment uses:"
echo "  CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR"
echo "  CLAUDEUP_HOME=$CLAUDEUP_HOME"
echo
