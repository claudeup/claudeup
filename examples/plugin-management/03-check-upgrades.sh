#!/usr/bin/env bash
# ABOUTME: Example showing how to check for and apply plugin upgrades
# ABOUTME: Demonstrates upgrade command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Plugin Management: Check Upgrades                    ║
╚════════════════════════════════════════════════════════════════╝

Keep your plugins and marketplaces up to date.

EOF
pause

section "1. Check for Upgrades"

step "See if any upgrades are available"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" outdated || \
    info "Outdated check would show available upgrades"
pause

section "2. Apply Upgrades"

step "Upgrade all plugins and marketplaces"
info "This fetches latest versions from all marketplaces"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" upgrade
else
    info "Command: claudeup upgrade"
    info "(Skipped in temp mode - no real plugins to upgrade)"
fi
pause

section "3. Verify After Upgrade"

step "Check status after upgrading"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" status
pause

section "Summary"

success "You can keep plugins up to date"
echo
info "Key commands:"
info "  claudeup outdated         See available upgrades"
info "  claudeup upgrade          Apply all upgrades"
echo
info "Tip: Run 'claudeup upgrade' regularly to get new features and fixes"
echo

prompt_cleanup
