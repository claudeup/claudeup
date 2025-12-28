#!/usr/bin/env bash
# ABOUTME: Example showing how to diagnose issues with claudeup doctor
# ABOUTME: Demonstrates doctor and cleanup commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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
run_cmd "$EXAMPLE_CLAUDEUP_BIN" doctor

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
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" cleanup --dry-run
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
