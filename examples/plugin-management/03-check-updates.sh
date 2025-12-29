#!/usr/bin/env bash
# ABOUTME: Example showing how to check for and apply plugin updates
# ABOUTME: Demonstrates update command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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
run_cmd "$EXAMPLE_CLAUDEUP_BIN" update --check || \
    info "Update check would show available updates"
pause

section "2. Apply Updates"

step "Update all plugins and marketplaces"
info "This fetches latest versions from all marketplaces"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" update
else
    info "Command: claudeup update"
    info "(Skipped in temp mode - no real plugins to update)"
fi
pause

section "3. Verify After Update"

step "Check status after updating"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" status
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
