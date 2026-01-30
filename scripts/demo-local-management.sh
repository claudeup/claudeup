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

# Build claudeup from the local-management branch
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
WORKTREE_DIR="$REPO_DIR/.worktrees/local-management"
CLAUDEUP_BIN="$WORKTREE_DIR/bin/claudeup"

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
    (cd "$WORKTREE_DIR" && go build -o bin/claudeup ./cmd/claudeup)
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

# Summary
print_header "Demo Complete!"

echo "Summary:"
echo "  - GSD agents enabled: $GSD_AGENTS"
echo "  - GSD commands enabled: $GSD_COMMANDS files"
echo "  - GSD hooks enabled: $GSD_HOOKS"
echo "  - Profile saved: gsd-demo"
echo
echo "The demo environment uses:"
echo "  CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR"
echo "  CLAUDEUP_HOME=$CLAUDEUP_HOME"
echo
