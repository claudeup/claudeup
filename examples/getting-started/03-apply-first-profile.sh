#!/usr/bin/env bash
# ABOUTME: Example script showing how to apply a profile
# ABOUTME: Demonstrates profile apply command and its effects

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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
