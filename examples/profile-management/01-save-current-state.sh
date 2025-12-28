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
