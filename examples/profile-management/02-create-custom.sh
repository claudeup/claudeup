#!/usr/bin/env bash
# ABOUTME: Example showing the interactive profile creation wizard
# ABOUTME: Demonstrates profile create command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Profile Management: Create Custom Profile              ║
╚════════════════════════════════════════════════════════════════╝

Create a new profile from scratch using the interactive wizard.
You can select which plugins and settings to include.

EOF
pause

section "1. Understanding Profile Creation"

info "The profile create command launches an interactive wizard that lets you:"
info "  • Name your profile"
info "  • Select plugins from installed marketplaces"
info "  • Configure MCP servers"
info "  • Set custom settings"
echo
info "In non-interactive mode, this example shows the command syntax."
pause

section "2. Create Command"

step "Create a new profile interactively"
echo
if [[ "$EXAMPLE_INTERACTIVE" == "true" && "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Running the interactive wizard..."
    echo
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile create
else
    info "To create a profile interactively, run:"
    echo -e "${YELLOW}\$ claudeup profile create${NC}"
    echo
    info "The wizard will guide you through:"
    info "  1. Naming the profile"
    info "  2. Selecting plugins"
    info "  3. Configuring options"
fi
pause

section "3. Alternative: Clone and Modify"

info "Another approach is to clone an existing profile:"
echo -e "${YELLOW}\$ claudeup profile clone default my-custom-profile${NC}"
echo
info "This copies all settings from the source profile,"
info "which you can then modify."
pause

section "Summary"

success "You know how to create custom profiles"
echo
info "Key commands:"
info "  claudeup profile create              Interactive wizard"
info "  claudeup profile clone <src> <dst>   Copy existing profile"
info "  claudeup profile save <name>         Save current state"
echo

prompt_cleanup
