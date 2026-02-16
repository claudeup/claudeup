#!/usr/bin/env bash
# ABOUTME: Example showing how to browse and inspect plugins
# ABOUTME: Demonstrates claudeup's read-only plugin management features

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║        Plugin Management: Browse and Inspect Plugins           ║
╚════════════════════════════════════════════════════════════════╝

Learn how to discover available plugins, inspect their contents before
installing, and manage installed plugins with claudeup and the claude CLI.

EOF
pause

section "1. List Currently Installed Plugins"

step "View all installed plugins and their status"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" plugin list || \
    info "Shows plugins with name, version, status, enabled scope, and source"

echo
info "The list shows which plugins are installed and where they're enabled."
info "Use --format detail for more information about each plugin."
pause

section "2. Browse Available Plugins in a Marketplace"

step "Discover plugins before installing them"
info "First, make sure you have a marketplace installed:"
echo -e "${YELLOW}\$ claudeup marketplace list${NC}"
echo

step "Browse plugins available in a marketplace"
info "Example command:"
echo -e "${YELLOW}\$ claudeup plugin browse claude-code-workflows${NC}"
echo
info "This shows all plugins available in the marketplace with:"
info "  • Plugin name and description"
info "  • Version number"
info "  • Installation status (if already installed)"
echo
info "You can also browse using the repo format:"
echo -e "${YELLOW}\$ claudeup plugin browse wshobson/agents${NC}"

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    echo
    step "Try browsing a marketplace now"
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" marketplace list
    echo
    info "Pick a marketplace from above and browse its plugins:"
    echo -e "${YELLOW}\$ claudeup plugin browse <marketplace-name>${NC}"
fi

pause

section "3. Inspect Plugin Contents Before Installing"

step "View what's inside a plugin"
info "Use the 'plugin show' command to inspect a plugin's structure:"
echo
echo -e "${YELLOW}\$ claudeup plugin show <plugin>@<marketplace>${NC}"
echo
info "This displays the plugin's directory tree - all agents, skills, and files."
info "Example:"
echo -e "${YELLOW}\$ claudeup plugin show observability-monitoring@claude-code-workflows${NC}"
echo

step "View specific files within a plugin"
info "You can also inspect individual files:"
echo
echo -e "${YELLOW}\$ claudeup plugin show my-plugin@marketplace agents/architect${NC}"
echo -e "${YELLOW}\$ claudeup plugin show my-plugin@marketplace skills/database${NC}"
echo
info "This helps you understand what the plugin does before installing it."
info "Markdown files are rendered; use --raw for unformatted output."

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    echo
    step "Try inspecting a plugin"
    info "Format: claudeup plugin show <plugin>@<marketplace>"
fi

pause

section "4. Install and Uninstall Plugins with the Claude CLI"

step "Installing plugins requires the claude CLI"
info "claudeup provides read-only plugin management (browse, inspect, list)."
info "To actually install or uninstall plugins, use the 'claude' CLI:"
echo
info "Install a plugin:"
echo -e "${YELLOW}\$ claude plugin install <plugin>@<marketplace>${NC}"
echo
info "Uninstall a plugin:"
echo -e "${YELLOW}\$ claude plugin uninstall <plugin>${NC}"
echo
info "The 'claude' CLI is the main Claude Code CLI that manages installations."
info "claudeup helps you discover and inspect plugins before installing them."
pause

section "5. Check Plugin Status After Installation"

step "Verify plugins were installed correctly"
info "After installing with 'claude plugin install', verify with claudeup:"
echo
echo -e "${YELLOW}\$ claudeup plugin list${NC}"
echo
info "The list will show the newly installed plugin with its enabled scope."
echo
info "Use additional flags for filtered views:"
info "  --enabled          Show only enabled plugins"
info "  --disabled         Show only disabled plugins"
info "  --format detail    Verbose per-plugin information"
info "  --by-scope         Group enabled plugins by scope"
pause

section "Summary"

success "You know how to discover and inspect plugins"
echo
info "claudeup read-only commands:"
info "  claudeup plugin list                            View installed plugins"
info "  claudeup plugin browse <marketplace>            Browse available plugins"
info "  claudeup plugin show <plugin>@<marketplace>     Inspect plugin contents"
echo
info "claude CLI commands for installation:"
info "  claude plugin install <plugin>@<marketplace>    Install a plugin"
info "  claude plugin uninstall <plugin>                Remove a plugin"
echo
info "Workflow:"
info "  1. Browse available plugins with 'claudeup plugin browse'"
info "  2. Inspect interesting plugins with 'claudeup plugin show'"
info "  3. Install plugins you want with 'claude plugin install'"
info "  4. Verify installation with 'claudeup plugin list'"
echo

prompt_cleanup
