#!/usr/bin/env bash
# ABOUTME: Example showing how to clone and customize a profile
# ABOUTME: Demonstrates profile clone and modification workflow

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Profile Management: Clone and Modify                   ║
╚════════════════════════════════════════════════════════════════╝

Start with an existing profile and customize it to your needs.
This is often easier than building a profile from scratch.

EOF
pause

section "1. Choose a Base Profile"

step "List profiles to find a good starting point"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

section "2. Clone the Profile"

step "Create a copy with a new name"
info "This copies all plugins, MCP servers, and settings"
echo

if $EXAMPLE_CLAUDEUP_BIN profile list 2>/dev/null | grep -q "base-tools"; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile clone base-tools my-customized
else
    info "Command: claudeup profile clone <source> <new-name>"
    info "Example: claudeup profile clone base-tools my-customized"
fi
pause

section "3. Modify the Clone"

info "Now you can modify your cloned profile by:"
echo
info "  1. Apply it: claudeup profile apply my-customized"
info "  2. Make changes (install/remove plugins, change settings)"
info "  3. Save changes: claudeup profile save my-customized"
echo
info "Or directly edit the profile file:"
info "  ~/.claudeup/profiles/my-customized.json"
pause

section "4. Verify Your Changes"

step "View the modified profile"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile show my-customized || \
    info "In a real installation, this shows the profile contents"
pause

section "Summary"

success "Clone-and-modify is a fast way to create custom profiles"
echo
info "Workflow:"
info "  1. claudeup profile clone <base> <new>"
info "  2. claudeup profile apply <new>"
info "  3. Make your changes"
info "  4. claudeup profile save <new>"
echo

prompt_cleanup
