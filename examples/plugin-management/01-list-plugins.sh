#!/usr/bin/env bash
# ABOUTME: Example showing how to view installed plugins
# ABOUTME: Demonstrates plugin list and status commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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
