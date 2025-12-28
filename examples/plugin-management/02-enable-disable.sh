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
