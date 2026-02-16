#!/usr/bin/env bash
# ABOUTME: Example showing how to view installed extensions
# ABOUTME: Demonstrates ext list command to show enabled/disabled status

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║          Extension Management: List Extensions                 ║
╚════════════════════════════════════════════════════════════════╝

View all installed Claude Code extensions and their enabled status.

Extensions are files (not marketplace plugins) that extend Claude with
custom agents, commands, skills, hooks, rules, and output-styles.

EOF
pause

section "1. List All Extensions"

step "View extension summary across all categories"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext list

info "Without arguments, 'ext list' shows summary counts by category."
info "Use 'ext list --full' to see individual items, or specify a category."
echo
info "Extensions are organized by category:"
info "  • agents          - Custom agent personalities"
info "  • commands        - Custom slash commands"
info "  • skills          - Reusable code skills"
info "  • hooks           - Event-driven automation"
info "  • rules           - Context and coding rules"
info "  • output-styles   - Custom output formatting"
pause

section "2. Understanding Extension Status"

info "When listing individual extensions, each shows its enabled status:"
echo
info "  ✓ enabled   - Active and available in Claude Code"
info "  · disabled  - Installed but not currently active"
echo
info "Extensions are stored in ~/.claudeup/ext/<category>/"
info "Enabled extensions are symlinked to ~/.claude/<category>/"
pause

section "3. Filter by Category"

step "View extensions in a specific category"

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "List just the rules:"
    run_cmd ls -la "$HOME/.claudeup/ext/rules/" 2>/dev/null || \
        info "(No rules installed yet)"
    echo
    info "List just the agents:"
    run_cmd ls -la "$HOME/.claudeup/ext/agents/" 2>/dev/null || \
        info "(No agents installed yet)"
else
    info "Example: ls ~/.claudeup/ext/rules/"
    info "Example: ls ~/.claudeup/ext/agents/"
    info "(Skipped in temp mode)"
fi
pause

section "4. View Extension Details"

step "Examine the contents of an extension"
info "Use 'claudeup ext view <category> <item>' to see what's in an extension:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Example command:"
    echo -e "${YELLOW}\$ claudeup ext view rules my-coding-standards.md${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext view rules golang-style.md${NC}"
    info "(Example - no real extensions in temp mode)"
fi

echo
info "This shows the file contents without needing to navigate directories."
pause

section "Summary"

success "You can view all your extensions and their status"
echo
info "Key commands:"
info "  claudeup ext list                List all extensions"
info "  claudeup ext view <category> <name>  View extension contents"
echo
info "Next steps:"
info "  • Learn to enable/disable extensions (02-enable-disable.sh)"
info "  • Learn to install new extensions (03-install-from-path.sh)"
echo

prompt_cleanup
