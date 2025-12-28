#!/usr/bin/env bash
# ABOUTME: Example script demonstrating claudeup installation verification
# ABOUTME: Shows version, status, and doctor commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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
run_cmd "$EXAMPLE_CLAUDEUP_BIN" --version
pause

section "2. View Installation Status"

step "Get an overview of your Claude Code installation"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" status

info "The status command shows:"
info "  • Installed plugins and their state"
info "  • Active marketplaces"
info "  • Current profile (if any)"
pause

section "3. Run Diagnostics"

step "Check for common issues with claudeup doctor"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" doctor

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
