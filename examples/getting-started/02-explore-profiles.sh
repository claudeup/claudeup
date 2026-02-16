#!/usr/bin/env bash
# ABOUTME: Example script showing how to explore available profiles
# ABOUTME: Demonstrates profile list and show commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list

info "Profile markers:"
info "  • * = currently active (highest precedence)"
info "  • ○ = active but overridden by higher scope"
pause

section "2. View Profile Contents"

step "Examine what a profile contains"
info "Let's look at a built-in profile to understand the structure"
echo

# Try to show a common profile, fall back gracefully
if $EXAMPLE_CLAUDEUP_BIN profile show default &>/dev/null; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile show default
else
    info "No built-in profiles available in this environment"
    info "In a real installation, you'd see plugins, MCP servers, and settings"
fi
pause

section "3. Understanding Scopes"

info "Claude Code has three configuration scopes:"
echo
info "  user     Your personal defaults, apply everywhere"
info "  project  Shared team settings, checked into git"
info "  local    Your local overrides, git-ignored"
echo
info "Later scopes override earlier ones: local > project > user"
info "See the team-setup examples for a detailed walkthrough of scope layering."
pause

section "Summary"

success "You now understand profiles and scopes"
echo
info "Next steps:"
info "  • Run 03-apply-first-profile.sh to apply a profile"
info "  • Run profile-management/01-save-current-state.sh to save your setup"
echo

prompt_cleanup
