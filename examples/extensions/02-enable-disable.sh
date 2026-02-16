#!/usr/bin/env bash
# ABOUTME: Example showing how to enable and disable extensions
# ABOUTME: Demonstrates ext enable/disable with patterns and wildcard support

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║       Extension Management: Enable & Disable Extensions        ║
╚════════════════════════════════════════════════════════════════╝

Control which extensions are active in your Claude Code environment.
Supports individual items, multiple items, and wildcard patterns.

EOF
pause

section "1. Disable a Single Extension"

step "Turn off an extension without removing it"
info "When you want to keep an extension installed but temporarily inactive:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext list
    echo
    info "To disable an extension:"
    echo -e "${YELLOW}\$ claudeup ext disable rules/my-coding-standards${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext disable rules/golang-style${NC}"
    info "(Example - no real extensions in temp mode)"
fi

echo
info "The extension stays in ~/.claudeup/ext/ but the symlink is removed."
info "It won't appear in Claude Code until re-enabled."
pause

section "2. Enable a Disabled Extension"

step "Reactivate an extension"
info "Bring back a disabled extension:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "To enable an extension:"
    echo -e "${YELLOW}\$ claudeup ext enable rules/my-coding-standards${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext enable rules/golang-style${NC}"
    info "(Example - no real extensions in temp mode)"
fi

echo
info "This creates the symlink from ~/.claude/ to ~/.claudeup/ext/"
info "The extension immediately becomes available in Claude Code."
pause

section "3. Enable/Disable Multiple Extensions"

step "Manage several extensions at once"
info "You can specify multiple extensions in one command:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Examples:"
    echo -e "${YELLOW}\$ claudeup ext enable rules/python-style rules/testing-standards${NC}"
    echo -e "${YELLOW}\$ claudeup ext disable agents/reviewer agents/writer${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext enable rules/python rules/go rules/typescript${NC}"
    echo -e "${YELLOW}\$ claudeup ext disable agents/creative agents/formal${NC}"
    info "(Examples - no real extensions in temp mode)"
fi

echo
info "Separate each extension with a space."
pause

section "4. Using Wildcard Patterns"

step "Enable or disable extensions by pattern"
info "Use wildcards to match multiple extensions:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Examples:"
    echo -e "${YELLOW}\$ claudeup ext enable 'rules/*'           ${NC}# Enable all rules"
    echo -e "${YELLOW}\$ claudeup ext disable 'agents/test-*'    ${NC}# Disable test agents"
    echo -e "${YELLOW}\$ claudeup ext enable '*-style'           ${NC}# Enable all *-style items"
else
    echo -e "${YELLOW}\$ claudeup ext enable 'rules/*'${NC}"
    info "Enables all rule extensions at once"
    echo
    echo -e "${YELLOW}\$ claudeup ext disable 'agents/draft-*'${NC}"
    info "Disables all agents starting with 'draft-'"
    echo
    echo -e "${YELLOW}\$ claudeup ext enable 'commands/dev-*'${NC}"
    info "Enables all development commands"
    info "(Examples - no real extensions in temp mode)"
fi

echo
warn "Remember to quote patterns to prevent shell expansion!"
pause

section "5. Verify Changes"

step "Check the updated status"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext list

info "Look for the ✓ (enabled) and ✗ (disabled) indicators."
pause

section "6. Sync Enabled State"

step "Rebuild symlinks from enabled.json"
info "If symlinks get out of sync, you can rebuild them:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext sync
else
    echo -e "${YELLOW}\$ claudeup ext sync${NC}"
    info "(Skipped in temp mode)"
fi

echo
info "This reads ~/.claudeup/enabled.json and recreates all symlinks."
info "Useful after manual file changes or git operations."
pause

section "Summary"

success "You can control which extensions are active"
echo
info "Key commands:"
info "  claudeup ext enable <category>/<name>     Enable an extension"
info "  claudeup ext disable <category>/<name>    Disable an extension"
info "  claudeup ext enable 'pattern/*'           Enable by wildcard"
info "  claudeup ext disable 'pattern/*'          Disable by wildcard"
info "  claudeup ext sync                         Rebuild symlinks"
echo
info "Best practices:"
info "  • Use disable/enable for temporary changes (keeps files)"
info "  • Use patterns for bulk operations"
info "  • Quote wildcard patterns to avoid shell expansion"
echo

prompt_cleanup
