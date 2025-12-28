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
