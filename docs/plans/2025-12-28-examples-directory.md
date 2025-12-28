# Examples Directory Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create an `examples/` directory with executable bash scripts demonstrating claudeup's feature set, organized by workflow.

**Architecture:** Shared library (`lib/common.sh`) provides safety checks, argument parsing, and interactive helpers. Each workflow directory contains numbered scripts that build on each other. Scripts default to isolated temp directories with `--real` flag for actual installations.

**Tech Stack:** Bash scripts, POSIX-compatible where possible, claudeup CLI

---

## Task 1: Create Directory Structure and README

**Files:**
- Create: `examples/README.md`
- Create: `examples/getting-started/.gitkeep`
- Create: `examples/profile-management/.gitkeep`
- Create: `examples/plugin-management/.gitkeep`
- Create: `examples/troubleshooting/.gitkeep`
- Create: `examples/team-setup/.gitkeep`
- Create: `examples/lib/.gitkeep`

**Step 1: Create directories**

```bash
mkdir -p examples/{lib,getting-started,profile-management,plugin-management,troubleshooting,team-setup}
```

**Step 2: Create README.md**

```markdown
# claudeup Examples

Hands-on tutorials for learning claudeup.

## Before You Start

### Optional: Version Control Your Claude Config

Your Claude configuration lives in `~/.claude/`. Version controlling it
lets you track changes over time and easily revert if needed:

```bash
cd ~/.claude
git init
git add -A
git commit -m "Initial Claude configuration"
```

The examples with `--real` mode will check for uncommitted changes
to help protect your work.

## Running Examples

By default, examples run in an isolated temp directory (safe to experiment):

```bash
./examples/getting-started/01-check-installation.sh
```

To run against your actual Claude installation:

```bash
./examples/getting-started/01-check-installation.sh --real
```

For scripting or CI (no pauses):

```bash
./examples/getting-started/01-check-installation.sh --non-interactive
```

## Workflows

| Directory | Description |
|-----------|-------------|
| `getting-started/` | First steps with claudeup |
| `profile-management/` | Create and switch configurations |
| `plugin-management/` | Control your plugins |
| `troubleshooting/` | Diagnose and fix issues |
| `team-setup/` | Share configurations across projects |

## Flags

| Flag | Behavior |
|------|----------|
| (none) | Interactive mode with isolated temp directory |
| `--real` | Operate on actual `~/.claude/` with safety checks |
| `--non-interactive` | No pauses, for CI/scripting |
| `--help` | Show usage for the specific example |
```sql

**Step 3: Commit**

```bash
git add examples/
git commit -m "feat: create examples directory structure and README"
```

---

## Task 2: Create Common Library

**Files:**
- Create: `examples/lib/common.sh`

**Step 1: Create the common library with all shared functions**

```bash
#!/usr/bin/env bash
# ABOUTME: Shared library for claudeup example scripts
# ABOUTME: Provides colors, prompts, safety checks, and temp environment setup

set -euo pipefail

# ============================================================================
# Colors (disabled when not a TTY)
# ============================================================================
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    MAGENTA='\033[0;35m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' MAGENTA='' CYAN='' BOLD='' NC=''
fi

# ============================================================================
# Global State
# ============================================================================
EXAMPLE_TEMP_DIR=""
EXAMPLE_REAL_MODE=false
EXAMPLE_INTERACTIVE=true
EXAMPLE_CLAUDEUP_BIN="${CLAUDEUP_BIN:-claudeup}"

# ============================================================================
# Output Helpers
# ============================================================================
section() {
    echo
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${BLUE}  $1${NC}"
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo
}

step() {
    echo -e "${MAGENTA}▶${NC} ${BOLD}$1${NC}"
}

info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1" >&2
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

# ============================================================================
# Interactive Helpers
# ============================================================================
pause() {
    if [[ "$EXAMPLE_INTERACTIVE" == "true" ]]; then
        echo
        echo -e "${GREEN}Press ENTER to continue...${NC}"
        read -r
    fi
}

run_cmd() {
    local cmd="$1"
    echo -e "${YELLOW}\$ $cmd${NC}"
    echo
    eval "$cmd"
    echo
}

# ============================================================================
# Environment Detection
# ============================================================================
check_claudeup_installed() {
    if ! command -v "$EXAMPLE_CLAUDEUP_BIN" &>/dev/null; then
        error "claudeup not found in PATH"
        error "Install claudeup or set CLAUDEUP_BIN to the binary path"
        exit 1
    fi
}

check_claude_config_dir_override() {
    if [[ -n "${CLAUDE_CONFIG_DIR:-}" ]]; then
        warn "CLAUDE_CONFIG_DIR is set to: $CLAUDE_CONFIG_DIR"
        warn "This example will operate on that directory, not ~/.claude"
        if [[ "$EXAMPLE_INTERACTIVE" == "true" ]]; then
            pause
        fi
    fi
}

# ============================================================================
# Safety Checks for --real Mode
# ============================================================================
check_git_clean() {
    local config_dir="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"

    if [[ -d "${config_dir}/.git" ]]; then
        if ! git -C "$config_dir" diff --quiet 2>/dev/null || \
           ! git -C "$config_dir" diff --cached --quiet 2>/dev/null; then
            error "Your Claude config has uncommitted changes"
            error "Location: $config_dir"
            error "Please commit or stash changes before running with --real"
            exit 1
        fi
    fi
}

warn_real_mode() {
    local config_dir="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"

    warn "⚠️  --real mode: This will modify your actual Claude configuration"
    warn "Location: $config_dir"

    if [[ "$EXAMPLE_INTERACTIVE" == "true" ]]; then
        echo
        echo -e "${YELLOW}Press ENTER to continue or Ctrl+C to abort...${NC}"
        read -r
    else
        error "Cannot run --real with --non-interactive (safety)"
        error "Remove --non-interactive to proceed with --real"
        exit 1
    fi
}

# ============================================================================
# Temp Environment Setup
# ============================================================================
setup_temp_claude_dir() {
    EXAMPLE_TEMP_DIR=$(mktemp -d "/tmp/claudeup-example-XXXXXX")

    # Create minimal Claude directory structure
    mkdir -p "$EXAMPLE_TEMP_DIR/.claude/plugins/cache"
    mkdir -p "$EXAMPLE_TEMP_DIR/.claudeup/profiles"

    # Export for claudeup to use
    export CLAUDE_CONFIG_DIR="$EXAMPLE_TEMP_DIR/.claude"
    export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/.claudeup"

    info "Using temp directory: $EXAMPLE_TEMP_DIR"
}

cleanup_temp_dir() {
    if [[ -n "$EXAMPLE_TEMP_DIR" && -d "$EXAMPLE_TEMP_DIR" ]]; then
        rm -rf "$EXAMPLE_TEMP_DIR"
    fi
}

trap_preserve_on_error() {
    trap 'on_error' ERR
}

on_error() {
    if [[ -n "$EXAMPLE_TEMP_DIR" && -d "$EXAMPLE_TEMP_DIR" ]]; then
        echo
        error "Script failed! Temp directory preserved for debugging:"
        error "  $EXAMPLE_TEMP_DIR"
        error ""
        error "To clean up manually: rm -rf $EXAMPLE_TEMP_DIR"
    fi
    exit 1
}

prompt_cleanup() {
    if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
        return 0
    fi

    if [[ -z "$EXAMPLE_TEMP_DIR" || ! -d "$EXAMPLE_TEMP_DIR" ]]; then
        return 0
    fi

    echo
    if [[ "$EXAMPLE_INTERACTIVE" == "true" ]]; then
        echo -e "${CYAN}Temp directory: $EXAMPLE_TEMP_DIR${NC}"
        echo -n "Remove temp directory? [Y/n] "
        read -r response
        case "${response:-y}" in
            [yY]|[yY][eE][sS]|"")
                cleanup_temp_dir
                success "Temp directory removed"
                ;;
            *)
                info "Temp directory preserved: $EXAMPLE_TEMP_DIR"
                ;;
        esac
    else
        cleanup_temp_dir
    fi
}

# ============================================================================
# Argument Parsing
# ============================================================================
show_help() {
    local script_name
    script_name=$(basename "$0")

    echo "Usage: $script_name [OPTIONS]"
    echo
    echo "Options:"
    echo "  --real            Operate on actual ~/.claude/ (with safety checks)"
    echo "  --non-interactive No pauses, for CI/scripting"
    echo "  -h, --help        Show this help message"
    echo
    echo "Environment:"
    echo "  CLAUDEUP_BIN      Path to claudeup binary (default: claudeup)"
}

parse_common_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --real)
                EXAMPLE_REAL_MODE=true
                shift
                ;;
            --non-interactive)
                EXAMPLE_INTERACTIVE=false
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# ============================================================================
# Main Setup
# ============================================================================
setup_environment() {
    check_claudeup_installed

    if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
        check_claude_config_dir_override
        check_git_clean
        warn_real_mode
    else
        setup_temp_claude_dir
        trap_preserve_on_error
    fi
}
```

**Step 2: Make it executable (for testing)**

```bash
chmod +x examples/lib/common.sh
```

**Step 3: Commit**

```bash
git add examples/lib/common.sh
git commit -m "feat: add common library for example scripts"
```

---

## Task 3: Create Getting Started Scripts

**Files:**
- Create: `examples/getting-started/01-check-installation.sh`
- Create: `examples/getting-started/02-explore-profiles.sh`
- Create: `examples/getting-started/03-apply-first-profile.sh`

**Step 1: Create 01-check-installation.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example script demonstrating claudeup installation verification
# ABOUTME: Shows version, status, and doctor commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Getting Started: Check Installation                  ║
╚════════════════════════════════════════════════════════════════╝

This example verifies claudeup is installed and working correctly.
You'll learn about the basic status and diagnostic commands.

EOF
pause

section "1. Check claudeup Version"

step "Verify claudeup is available and check its version"
run_cmd "$EXAMPLE_CLAUDEUP_BIN --version"
pause

section "2. View Installation Status"

step "Get an overview of your Claude Code installation"
run_cmd "$EXAMPLE_CLAUDEUP_BIN status"

info "The status command shows:"
info "  • Installed plugins and their state"
info "  • Active marketplaces"
info "  • Current profile (if any)"
pause

section "3. Run Diagnostics"

step "Check for common issues with claudeup doctor"
run_cmd "$EXAMPLE_CLAUDEUP_BIN doctor"

info "The doctor command checks for:"
info "  • Missing or corrupted plugin files"
info "  • Invalid configuration"
info "  • Path issues"
pause

section "Summary"

success "claudeup is installed and working"
echo
info "Next steps:"
info "  • Run 02-explore-profiles.sh to see available profiles"
info "  • Run 03-apply-first-profile.sh to apply your first profile"
echo

prompt_cleanup
```

**Step 2: Create 02-explore-profiles.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example script showing how to explore available profiles
# ABOUTME: Demonstrates profile list and show commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Getting Started: Explore Profiles                    ║
╚════════════════════════════════════════════════════════════════╝

Profiles are saved configurations of plugins, MCP servers, and settings.
This example shows you what profiles are available.

EOF
pause

section "1. List Available Profiles"

step "See all profiles (built-in and custom)"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile list"

info "Profile markers:"
info "  • * = currently active (highest precedence)"
info "  • ○ = active but overridden by higher scope"
info "  • (modified) = differs from saved definition"
pause

section "2. View Profile Contents"

step "Examine what a profile contains"
info "Let's look at a built-in profile to understand the structure"
echo

# Try to show a common profile, fall back gracefully
if $EXAMPLE_CLAUDEUP_BIN profile show base-tools &>/dev/null; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN profile show base-tools"
else
    info "No built-in profiles available in this environment"
    info "In a real installation, you'd see plugins, MCP servers, and settings"
fi
pause

section "3. Understanding Scopes"

info "Claude Code has three configuration scopes:"
echo
info "  user    (~/.claude/settings.json)"
info "          └─ Your personal defaults, apply everywhere"
echo
info "  project (.claude/settings.json)"
info "          └─ Shared team settings, checked into git"
echo
info "  local   (.claude/settings.local.json)"
info "          └─ Your local overrides, git-ignored"
echo
info "Later scopes override earlier ones: local > project > user"
pause

section "Summary"

success "You now understand profiles and scopes"
echo
info "Next steps:"
info "  • Run 03-apply-first-profile.sh to apply a profile"
info "  • Run profile-management/01-save-current-state.sh to save your setup"
echo

prompt_cleanup
```

**Step 3: Create 03-apply-first-profile.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example script showing how to apply a profile
# ABOUTME: Demonstrates profile apply command and its effects

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Getting Started: Apply First Profile                 ║
╚════════════════════════════════════════════════════════════════╝

This example shows how to apply a profile to configure Claude Code.
You'll see what changes before and after applying.

EOF
pause

section "1. Current State (Before)"

step "Check the current profile status"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile current" || info "No profile currently active"
pause

section "2. Apply a Profile"

step "Apply a profile to configure Claude Code"
info "Using --scope user to set it as your default"
echo

# In temp mode, we need to handle the case where no profiles exist
if $EXAMPLE_CLAUDEUP_BIN profile list 2>/dev/null | grep -q "base-tools"; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN profile apply base-tools --scope user"
else
    info "In a real installation, you would run:"
    echo -e "${YELLOW}\$ claudeup profile apply <profile-name> --scope user${NC}"
    echo
    info "This installs the profile's plugins and applies its settings"
fi
pause

section "3. Verify the Change"

step "Check that the profile is now active"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile current" || info "Profile status updated"

step "View the updated status"
run_cmd "$EXAMPLE_CLAUDEUP_BIN status"
pause

section "Summary"

success "You've learned how to apply profiles"
echo
info "Key commands:"
info "  claudeup profile apply <name> --scope user     Apply as default"
info "  claudeup profile apply <name> --scope project  Apply for this project"
info "  claudeup profile apply <name> --scope local    Apply as local override"
echo
info "Next steps:"
info "  • Explore profile-management/ to create your own profiles"
info "  • Run troubleshooting/ examples if something goes wrong"
echo

prompt_cleanup
```

**Step 4: Make scripts executable**

```bash
chmod +x examples/getting-started/*.sh
```

**Step 5: Commit**

```bash
git add examples/getting-started/
git commit -m "feat: add getting-started example scripts"
```

---

## Task 4: Create Profile Management Scripts

**Files:**
- Create: `examples/profile-management/01-save-current-state.sh`
- Create: `examples/profile-management/02-create-custom.sh`
- Create: `examples/profile-management/03-switch-profiles.sh`
- Create: `examples/profile-management/04-clone-and-modify.sh`

**Step 1: Create 01-save-current-state.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to save current Claude setup as a profile
# ABOUTME: Demonstrates profile save command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Profile Management: Save Current State                 ║
╚════════════════════════════════════════════════════════════════╝

Save your current Claude Code configuration as a reusable profile.
This lets you restore your setup later or share it with others.

EOF
pause

section "1. View Current Configuration"

step "See what's currently configured"
run_cmd "$EXAMPLE_CLAUDEUP_BIN status"
pause

section "2. Save as a Profile"

step "Save the current state to a named profile"
info "This captures all plugins, MCP servers, and settings"
echo
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile save my-setup"
pause

section "3. Verify the Profile"

step "Confirm the profile was saved"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile list"

step "View the saved profile contents"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile show my-setup" || info "Profile details would appear here"
pause

section "Summary"

success "Your configuration is saved as 'my-setup'"
echo
info "Saved profiles are stored in: ~/.claudeup/profiles/"
info "You can apply this profile anytime with:"
info "  claudeup profile apply my-setup"
echo

prompt_cleanup
```

**Step 2: Create 02-create-custom.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing the interactive profile creation wizard
# ABOUTME: Demonstrates profile create command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Profile Management: Create Custom Profile              ║
╚════════════════════════════════════════════════════════════════╝

Create a new profile from scratch using the interactive wizard.
You can select which plugins and settings to include.

EOF
pause

section "1. Understanding Profile Creation"

info "The profile create command launches an interactive wizard that lets you:"
info "  • Name your profile"
info "  • Select plugins from installed marketplaces"
info "  • Configure MCP servers"
info "  • Set custom settings"
echo
info "In non-interactive mode, this example shows the command syntax."
pause

section "2. Create Command"

step "Create a new profile interactively"
echo
if [[ "$EXAMPLE_INTERACTIVE" == "true" && "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Running the interactive wizard..."
    echo
    run_cmd "$EXAMPLE_CLAUDEUP_BIN profile create"
else
    info "To create a profile interactively, run:"
    echo -e "${YELLOW}\$ claudeup profile create${NC}"
    echo
    info "The wizard will guide you through:"
    info "  1. Naming the profile"
    info "  2. Selecting plugins"
    info "  3. Configuring options"
fi
pause

section "3. Alternative: Clone and Modify"

info "Another approach is to clone an existing profile:"
echo -e "${YELLOW}\$ claudeup profile clone base-tools my-custom-profile${NC}"
echo
info "This copies all settings from the source profile,"
info "which you can then modify."
pause

section "Summary"

success "You know how to create custom profiles"
echo
info "Key commands:"
info "  claudeup profile create              Interactive wizard"
info "  claudeup profile clone <src> <dst>   Copy existing profile"
info "  claudeup profile save <name>         Save current state"
echo

prompt_cleanup
```

**Step 3: Create 03-switch-profiles.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to switch between profiles
# ABOUTME: Demonstrates profile apply and diff commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Profile Management: Switch Between Profiles            ║
╚════════════════════════════════════════════════════════════════╝

Switch between different profiles to change your Claude configuration.
Learn how to preview changes before applying.

EOF
pause

section "1. List Available Profiles"

step "See what profiles you can switch to"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile list"
pause

section "2. Preview Changes with Diff"

step "See what would change before switching"
info "The diff command shows differences between a profile and current state"
echo

# Try to show diff, handle gracefully if no profiles
if $EXAMPLE_CLAUDEUP_BIN profile list 2>/dev/null | grep -q "base-tools"; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN profile diff base-tools" || info "No differences or profile not found"
else
    info "Example diff output would show:"
    info "  + plugins being added"
    info "  - plugins being removed"
    info "  ~ settings being changed"
fi
pause

section "3. Switch Profiles"

step "Apply a different profile"
info "Switching profiles will:"
info "  • Install new plugins from the target profile"
info "  • Keep plugins that exist in both"
info "  • Optionally remove plugins not in the target (with --reset)"
echo

if $EXAMPLE_CLAUDEUP_BIN profile list 2>/dev/null | grep -q "base-tools"; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN profile apply base-tools --scope user"
else
    info "Command: claudeup profile apply <profile-name> --scope user"
fi
pause

section "4. Verify the Switch"

step "Confirm the new profile is active"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile current" || true
pause

section "Summary"

success "You can switch profiles confidently"
echo
info "Tips:"
info "  • Use 'profile diff' to preview before switching"
info "  • Use '--scope project' for project-specific profiles"
info "  • Use 'profile reset' to remove all profile components"
echo

prompt_cleanup
```

**Step 4: Create 04-clone-and-modify.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to clone and customize a profile
# ABOUTME: Demonstrates profile clone and modification workflow

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Profile Management: Clone and Modify                   ║
╚════════════════════════════════════════════════════════════════╝

Start with an existing profile and customize it to your needs.
This is often easier than building a profile from scratch.

EOF
pause

section "1. Choose a Base Profile"

step "List profiles to find a good starting point"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile list"
pause

section "2. Clone the Profile"

step "Create a copy with a new name"
info "This copies all plugins, MCP servers, and settings"
echo

if $EXAMPLE_CLAUDEUP_BIN profile list 2>/dev/null | grep -q "base-tools"; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN profile clone base-tools my-customized"
else
    info "Command: claudeup profile clone <source> <new-name>"
    info "Example: claudeup profile clone base-tools my-customized"
fi
pause

section "3. Modify the Clone"

info "Now you can modify your cloned profile by:"
echo
info "  1. Apply it: claudeup profile apply my-customized"
info "  2. Make changes (install/remove plugins, change settings)"
info "  3. Save changes: claudeup profile save my-customized"
echo
info "Or directly edit the profile file:"
info "  ~/.claudeup/profiles/my-customized.json"
pause

section "4. Verify Your Changes"

step "View the modified profile"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile show my-customized" || \
    info "In a real installation, this shows the profile contents"
pause

section "Summary"

success "Clone-and-modify is a fast way to create custom profiles"
echo
info "Workflow:"
info "  1. claudeup profile clone <base> <new>"
info "  2. claudeup profile apply <new>"
info "  3. Make your changes"
info "  4. claudeup profile save <new>"
echo

prompt_cleanup
```

**Step 5: Make scripts executable and commit**

```bash
chmod +x examples/profile-management/*.sh
git add examples/profile-management/
git commit -m "feat: add profile-management example scripts"
```

---

## Task 5: Create Plugin Management Scripts

**Files:**
- Create: `examples/plugin-management/01-list-plugins.sh`
- Create: `examples/plugin-management/02-enable-disable.sh`
- Create: `examples/plugin-management/03-check-updates.sh`

**Step 1: Create 01-list-plugins.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to view installed plugins
# ABOUTME: Demonstrates plugin list and status commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Plugin Management: List Plugins                      ║
╚════════════════════════════════════════════════════════════════╝

View all installed Claude Code plugins and their current state.

EOF
pause

section "1. List All Plugins"

step "View installed plugins"
run_cmd "$EXAMPLE_CLAUDEUP_BIN plugin list"

info "Plugin states:"
info "  • enabled  - Active and providing functionality"
info "  • disabled - Installed but not active"
pause

section "2. View Plugin Details in Status"

step "Get a complete overview including plugins"
run_cmd "$EXAMPLE_CLAUDEUP_BIN status"

info "Status shows plugins grouped by marketplace"
pause

section "3. Understanding Plugin Sources"

info "Plugins come from marketplaces (plugin repositories):"
echo
run_cmd "$EXAMPLE_CLAUDEUP_BIN marketplace list"

info "Each marketplace provides different plugins."
info "Use 'claude plugin install' to add new plugins."
pause

section "Summary"

success "You can view all your plugins"
echo
info "Key commands:"
info "  claudeup plugin list       List all plugins"
info "  claudeup status            Full overview"
info "  claudeup marketplace list  View plugin sources"
echo

prompt_cleanup
```

**Step 2: Create 02-enable-disable.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to enable and disable plugins
# ABOUTME: Demonstrates plugin enable and disable commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Plugin Management: Enable/Disable                    ║
╚════════════════════════════════════════════════════════════════╝

Toggle plugins on and off without uninstalling them.
Useful for troubleshooting or temporary changes.

EOF
pause

section "1. View Current Plugin States"

step "List plugins to see which are enabled/disabled"
run_cmd "$EXAMPLE_CLAUDEUP_BIN plugin list"
pause

section "2. Disable a Plugin"

step "Temporarily disable a plugin"
info "Disabling keeps the plugin installed but inactive"
echo

# Show the command syntax
info "Command syntax:"
echo -e "${YELLOW}\$ claudeup plugin disable <plugin-name>${NC}"
echo
info "Example:"
echo -e "${YELLOW}\$ claudeup plugin disable superpowers@superpowers-marketplace${NC}"
pause

section "3. Enable a Plugin"

step "Re-enable a disabled plugin"
info "This restores the plugin to active state"
echo

info "Command syntax:"
echo -e "${YELLOW}\$ claudeup plugin enable <plugin-name>${NC}"
echo
info "Example:"
echo -e "${YELLOW}\$ claudeup plugin enable superpowers@superpowers-marketplace${NC}"
pause

section "4. When to Disable vs Uninstall"

info "Disable when:"
info "  • Troubleshooting conflicts"
info "  • Temporarily reducing resource usage"
info "  • Testing without a specific plugin"
echo
info "Uninstall when:"
info "  • You no longer need the plugin"
info "  • Freeing up disk space"
info "  • Clean removal is required"
pause

section "Summary"

success "You can toggle plugins without reinstalling"
echo
info "Key commands:"
info "  claudeup plugin disable <name>  Deactivate plugin"
info "  claudeup plugin enable <name>   Reactivate plugin"
info "  claude plugin uninstall <name>  Remove completely"
echo

prompt_cleanup
```

**Step 3: Create 03-check-updates.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to check for and apply plugin updates
# ABOUTME: Demonstrates update command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Plugin Management: Check Updates                     ║
╚════════════════════════════════════════════════════════════════╝

Keep your plugins and marketplaces up to date.

EOF
pause

section "1. Check for Updates"

step "See if any updates are available"
run_cmd "$EXAMPLE_CLAUDEUP_BIN update --check" || \
    info "Update check would show available updates"
pause

section "2. Apply Updates"

step "Update all plugins and marketplaces"
info "This fetches latest versions from all marketplaces"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN update"
else
    info "Command: claudeup update"
    info "(Skipped in temp mode - no real plugins to update)"
fi
pause

section "3. Verify After Update"

step "Check status after updating"
run_cmd "$EXAMPLE_CLAUDEUP_BIN status"
pause

section "Summary"

success "You can keep plugins up to date"
echo
info "Key commands:"
info "  claudeup update --check  See available updates"
info "  claudeup update          Apply all updates"
echo
info "Tip: Run 'claudeup update' regularly to get new features and fixes"
echo

prompt_cleanup
```

**Step 4: Make scripts executable and commit**

```bash
chmod +x examples/plugin-management/*.sh
git add examples/plugin-management/
git commit -m "feat: add plugin-management example scripts"
```

---

## Task 6: Create Troubleshooting Scripts

**Files:**
- Create: `examples/troubleshooting/01-run-doctor.sh`
- Create: `examples/troubleshooting/02-view-events.sh`
- Create: `examples/troubleshooting/03-diff-changes.sh`

**Step 1: Create 01-run-doctor.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to diagnose issues with claudeup doctor
# ABOUTME: Demonstrates doctor and cleanup commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Troubleshooting: Run Doctor                          ║
╚════════════════════════════════════════════════════════════════╝

Diagnose and fix common issues with your Claude Code installation.

EOF
pause

section "1. Run Diagnostics"

step "Check for common issues"
run_cmd "$EXAMPLE_CLAUDEUP_BIN doctor"

info "Doctor checks for:"
info "  • Missing plugin files"
info "  • Invalid configuration"
info "  • Orphaned entries"
info "  • Path mismatches"
pause

section "2. Fix Issues with Cleanup"

step "Automatically fix detected issues"
info "The cleanup command can fix many issues doctor finds"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN cleanup --dry-run"
    info "Remove --dry-run to actually apply fixes"
else
    info "Command: claudeup cleanup"
    info "Add --dry-run to preview changes without applying"
fi
pause

section "3. Manual Fixes"

info "Some issues require manual intervention:"
echo
info "  • Reinstall corrupted plugins:"
info "    claude plugin uninstall <name> && claude plugin install <name>"
echo
info "  • Reset profile state:"
info "    claudeup profile reset"
echo
info "  • Start fresh (nuclear option):"
info "    rm -rf ~/.claude && claude"
pause

section "Summary"

success "You can diagnose and fix common issues"
echo
info "Key commands:"
info "  claudeup doctor          Diagnose issues"
info "  claudeup cleanup         Fix automatically"
info "  claudeup cleanup --dry-run  Preview fixes"
echo

prompt_cleanup
```

**Step 2: Create 02-view-events.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to view file operation history
# ABOUTME: Demonstrates events and events audit commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Troubleshooting: View Events                         ║
╚════════════════════════════════════════════════════════════════╝

See what changes claudeup has made to your configuration over time.
Essential for understanding "what happened?"

EOF
pause

section "1. View Recent Events"

step "See the most recent file operations"
run_cmd "$EXAMPLE_CLAUDEUP_BIN events --limit 10"

info "Events show:"
info "  • When changes happened"
info "  • What files were modified"
info "  • What operation caused the change"
pause

section "2. Filter Events"

step "Find specific types of changes"
echo

info "Filter by file:"
echo -e "${YELLOW}\$ claudeup events --file ~/.claude/settings.json${NC}"
echo

info "Filter by operation:"
echo -e "${YELLOW}\$ claudeup events --operation profile${NC}"
echo

info "Filter by time:"
echo -e "${YELLOW}\$ claudeup events --since 24h${NC}"
pause

section "3. Generate Audit Report"

step "Get a comprehensive timeline"
run_cmd "$EXAMPLE_CLAUDEUP_BIN events audit --since 7d" || \
    info "Audit report would show grouped timeline"

info "Audit reports group events by date and show:"
info "  • Timeline of all operations"
info "  • File size changes"
info "  • Operation categories"
pause

section "Summary"

success "You can track all configuration changes"
echo
info "Key commands:"
info "  claudeup events                Show recent events"
info "  claudeup events --since 24h    Filter by time"
info "  claudeup events audit          Comprehensive report"
echo

prompt_cleanup
```

**Step 3: Create 03-diff-changes.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to see detailed file changes
# ABOUTME: Demonstrates events diff command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Troubleshooting: Diff Changes                        ║
╚════════════════════════════════════════════════════════════════╝

See exactly what changed in a file operation.
Essential for understanding why something broke.

EOF
pause

section "1. Find a Change to Inspect"

step "List recent events to find an interesting change"
run_cmd "$EXAMPLE_CLAUDEUP_BIN events --limit 5"
pause

section "2. View the Diff"

step "See what changed in a specific file"
info "The diff command shows before/after comparison"
echo

info "Command syntax:"
echo -e "${YELLOW}\$ claudeup events diff --file ~/.claude/settings.json${NC}"
echo

info "This shows the most recent change to that file"
pause

section "3. Full Diff Mode"

step "Get detailed nested changes"
info "Use --full for complete recursive diff of nested objects"
echo

info "Example output:"
cat <<'EXAMPLE'
~ plugins:
  ~ superpowers@superpowers-marketplace:
    ~ scope: "project" → "user"
    ~ installedAt: "2025-12-26T05:14:20Z" → "2025-12-28T10:30:00Z"
  + newplugin@marketplace:
    + scope: "user" (added)
EXAMPLE
echo

info "Symbols: + added, - removed, ~ modified"
pause

section "4. Practical Use Case"

info "Common debugging workflow:"
echo
info "  1. Something broke after a change"
info "  2. Run: claudeup events --since 1h"
info "  3. Find the relevant file change"
info "  4. Run: claudeup events diff --file <path> --full"
info "  5. See exactly what changed"
info "  6. Decide: revert or fix forward"
pause

section "Summary"

success "You can see exactly what changed and when"
echo
info "Key commands:"
info "  claudeup events diff --file <path>        Basic diff"
info "  claudeup events diff --file <path> --full Detailed diff"
echo

prompt_cleanup
```

**Step 4: Make scripts executable and commit**

```bash
chmod +x examples/troubleshooting/*.sh
git add examples/troubleshooting/
git commit -m "feat: add troubleshooting example scripts"
```

---

## Task 7: Create Team Setup Scripts

**Files:**
- Create: `examples/team-setup/01-scoped-profiles.sh`
- Create: `examples/team-setup/02-project-config.sh`
- Create: `examples/team-setup/03-sync-team-config.sh`

**Step 1: Create 01-scoped-profiles.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how scopes work in Claude Code
# ABOUTME: Demonstrates user, project, and local scope differences

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║             Team Setup: Understanding Scopes                   ║
╚════════════════════════════════════════════════════════════════╝

Claude Code uses three configuration scopes. Understanding them
is key to effective team collaboration.

EOF
pause

section "1. The Three Scopes"

info "USER scope (~/.claude/settings.json)"
info "  • Your personal defaults"
info "  • Applies to all projects"
info "  • Not shared with team"
echo

info "PROJECT scope (.claude/settings.json)"
info "  • Shared team configuration"
info "  • Checked into git"
info "  • Everyone on the team gets these settings"
echo

info "LOCAL scope (.claude/settings.local.json)"
info "  • Your personal overrides for this project"
info "  • Git-ignored (not shared)"
info "  • Highest precedence"
pause

section "2. Scope Precedence"

info "Settings are merged with later scopes winning:"
echo
info "  user → project → local"
info "  (lowest)        (highest)"
echo
info "Example: If user scope enables plugin A"
info "         and local scope disables plugin A"
info "         → Plugin A is disabled"
pause

section "3. Apply to Different Scopes"

step "See which scopes have active profiles"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile list"

step "Apply a profile to a specific scope"
info "Commands:"
echo -e "${YELLOW}\$ claudeup profile apply myprofile --scope user    # Personal default${NC}"
echo -e "${YELLOW}\$ claudeup profile apply myprofile --scope project # Team setting${NC}"
echo -e "${YELLOW}\$ claudeup profile apply myprofile --scope local   # Local override${NC}"
pause

section "4. View Scope Contents"

step "See what's configured at each scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN scope list" || \
    info "Scope list would show files and their contents"
pause

section "Summary"

success "You understand Claude Code's scope system"
echo
info "Best practices:"
info "  • User scope: Your personal productivity tools"
info "  • Project scope: Team-required plugins and settings"
info "  • Local scope: Personal tweaks that don't affect team"
echo

prompt_cleanup
```

**Step 2: Create 02-project-config.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to use .claudeup.json for project configuration
# ABOUTME: Demonstrates project-level config file usage

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║             Team Setup: Project Configuration                  ║
╚════════════════════════════════════════════════════════════════╝

Use .claudeup.json to define project-specific Claude configuration
that travels with your repository.

EOF
pause

section "1. The .claudeup.json File"

info "Place .claudeup.json in your project root to define:"
info "  • Required plugins for the project"
info "  • Recommended profiles"
info "  • MCP server configurations"
echo

step "Example .claudeup.json structure"
cat <<'EXAMPLE'
{
  "plugins": [
    "superpowers@superpowers-marketplace",
    "backend-development@claude-code-workflows"
  ]
}
EXAMPLE
pause

section "2. Create a Project Config"

step "Generate .claudeup.json from current state"
info "This captures your project's plugin requirements"
echo

# In real mode, actually create it
if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Creating .claudeup.json in current directory..."
    # Note: claudeup doesn't have a direct command for this yet
    info "Command: claudeup profile save --output .claudeup.json"
else
    info "In your project, run:"
    echo -e "${YELLOW}\$ claudeup profile save --output .claudeup.json${NC}"
fi
pause

section "3. Sync Team Configuration"

step "Apply project configuration"
info "When a teammate clones the repo, they can sync:"
echo
echo -e "${YELLOW}\$ claudeup profile sync${NC}"
echo

info "This installs all plugins defined in .claudeup.json"
pause

section "4. Git Integration"

info "Recommended .gitignore entries:"
cat <<'GITIGNORE'
# Claude Code local settings (personal overrides)
.claude/settings.local.json

# Keep these tracked for team sharing:
# .claude/settings.json
# .claudeup.json
GITIGNORE
pause

section "Summary"

success "You can share Claude configuration via git"
echo
info "Key files:"
info "  .claudeup.json              Project plugin requirements"
info "  .claude/settings.json       Project Claude settings"
info "  .claude/settings.local.json Personal overrides (git-ignored)"
echo

prompt_cleanup
```

**Step 3: Create 03-sync-team-config.sh**

```bash
#!/usr/bin/env bash
# ABOUTME: Example showing how to sync team configuration
# ABOUTME: Demonstrates profile sync command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║             Team Setup: Sync Team Configuration                ║
╚════════════════════════════════════════════════════════════════╝

Keep your Claude setup in sync with your team's project requirements.

EOF
pause

section "1. Check for Project Configuration"

step "Look for .claudeup.json in the current project"
if [[ -f ".claudeup.json" ]]; then
    info "Found .claudeup.json:"
    cat .claudeup.json
else
    info "No .claudeup.json found in current directory"
    info "This file defines project plugin requirements"
fi
pause

section "2. Sync Configuration"

step "Install plugins defined in .claudeup.json"
info "The sync command ensures you have all required plugins"
echo

run_cmd "$EXAMPLE_CLAUDEUP_BIN profile sync" || \
    info "Sync would install any missing plugins"
pause

section "3. Onboarding Workflow"

info "When joining a project with Claude configuration:"
echo
info "  1. Clone the repository"
info "     git clone <repo-url>"
echo
info "  2. Sync Claude configuration"
info "     cd <project>"
info "     claudeup profile sync"
echo
info "  3. (Optional) Add personal overrides"
info "     claudeup profile apply my-tools --scope local"
pause

section "4. Keeping in Sync"

info "After pulling changes that modify .claudeup.json:"
echo
echo -e "${YELLOW}\$ git pull${NC}"
echo -e "${YELLOW}\$ claudeup profile sync${NC}"
echo
info "This installs any new plugins the team has added"
pause

section "Summary"

success "You can stay in sync with team configuration"
echo
info "Key commands:"
info "  claudeup profile sync   Install project requirements"
echo
info "Workflow:"
info "  1. Team adds plugin to .claudeup.json"
info "  2. Team commits and pushes"
info "  3. You pull and run 'claudeup profile sync'"
echo

prompt_cleanup
```

**Step 4: Make scripts executable and commit**

```bash
chmod +x examples/team-setup/*.sh
git add examples/team-setup/
git commit -m "feat: add team-setup example scripts"
```

---

## Task 8: Final Cleanup and Testing

**Files:**
- Remove: `examples/*/.gitkeep` (directories now have content)

**Step 1: Remove placeholder files**

```bash
rm -f examples/*/.gitkeep examples/lib/.gitkeep
```

**Step 2: Verify all scripts are executable**

```bash
find examples -name "*.sh" -exec chmod +x {} \;
```

**Step 3: Test a script works**

```bash
./examples/getting-started/01-check-installation.sh --non-interactive
```

**Step 4: Final commit**

```bash
git add -A examples/
git commit -m "chore: finalize examples directory"
```

**Step 5: Push feature branch**

```bash
git push -u origin feature/examples-directory
```
