#!/usr/bin/env bash
# ABOUTME: Example showing how to enable, disable, and install plugins
# ABOUTME: Demonstrates plugin enable, disable, and install commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Plugin Management: Manage Plugins                    ║
╚════════════════════════════════════════════════════════════════╝

Learn how to enable, disable, and install plugins to customize your
Claude Code environment.

EOF
pause

section "1. Install a New Plugin"

step "Search for available plugins in a marketplace"
info "First, you need a marketplace installed. Check with 'claudeup status'"
echo

step "Install a plugin from a marketplace"
info "Example command:"
echo -e "${YELLOW}\$ claudeup plugin install superpowers@superpowers-marketplace${NC}"
echo
info "Format: claudeup plugin install <plugin-name>@<marketplace-name>"
info ""
info "If this is your first plugin from a marketplace, claudeup will:"
info "  1. Clone the marketplace repository"
info "  2. Install the requested plugin"
info "  3. Enable it automatically"
pause

section "2. Disable a Plugin"

step "Disable a plugin without removing it"
info "When you want to keep a plugin installed but temporarily turn it off:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    # In real mode, try to disable a common plugin if it exists
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" plugin list
    echo
    info "To disable a plugin:"
    echo -e "${YELLOW}\$ claudeup plugin disable <plugin-name>${NC}"
else
    echo -e "${YELLOW}\$ claudeup plugin disable backend-development${NC}"
    info "(Example - no real plugins in temp mode)"
fi

echo
info "Disabled plugins remain installed but won't load in Claude Code."
info "Their files stay on disk for quick re-enabling."
pause

section "3. Enable a Previously Disabled Plugin"

step "Re-enable a disabled plugin"
info "Bring back a disabled plugin without reinstalling:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "To enable a plugin:"
    echo -e "${YELLOW}\$ claudeup plugin enable <plugin-name>${NC}"
else
    echo -e "${YELLOW}\$ claudeup plugin enable backend-development${NC}"
    info "(Example - no real plugins in temp mode)"
fi

echo
info "This is instant - just updates the enabled state."
pause

section "4. View Current Plugin State"

step "Check which plugins are enabled vs disabled"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" plugin list || \
    info "Plugin list shows enabled/disabled state for each plugin"

echo
info "Look for the status column to see which plugins are active."
pause

section "5. Uninstall a Plugin"

step "Completely remove a plugin"
info "When you no longer need a plugin at all:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "To uninstall a plugin:"
    echo -e "${YELLOW}\$ claudeup plugin uninstall <plugin-name>${NC}"
else
    echo -e "${YELLOW}\$ claudeup plugin uninstall old-plugin${NC}"
    info "(Example - no real plugins in temp mode)"
fi

echo
info "This removes the plugin files from disk."
info "You'll need to reinstall if you want it back later."
pause

section "Summary"

success "You know how to manage your plugins"
echo
info "Key commands:"
info "  claudeup plugin install <name>@<marketplace>    Install new plugin"
info "  claudeup plugin disable <name>                  Disable temporarily"
info "  claudeup plugin enable <name>                   Re-enable plugin"
info "  claudeup plugin uninstall <name>                Remove completely"
info "  claudeup plugin list                            View all plugins"
echo
info "Best practices:"
info "  • Use disable/enable for temporary changes"
info "  • Use uninstall only when you're sure you won't need it"
info "  • Keep plugins updated with 'claudeup upgrade'"
echo

prompt_cleanup
