#!/usr/bin/env bash
# ABOUTME: Example showing how to install extensions from external sources
# ABOUTME: Demonstrates ext install from git repos and local paths

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║      Extension Management: Install from External Paths         ║
╚════════════════════════════════════════════════════════════════╝

Install extensions from git repositories, downloads, or local paths.
Great for sharing team extensions or using community-created content.

EOF
pause

section "1. Install from a Git Repository"

step "Clone and install extensions from a git repo"
info "Install directly from a GitHub (or other git) repository:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Example command:"
    echo -e "${YELLOW}\$ claudeup ext install https://github.com/myteam/claude-extensions${NC}"
    echo
    info "This will:"
    info "  1. Clone the repository to a temp location"
    info "  2. Find all valid extension categories (agents/, rules/, etc.)"
    info "  3. Copy them to ~/.claudeup/ext/"
    info "  4. Prompt you to enable them"
else
    echo -e "${YELLOW}\$ claudeup ext install https://github.com/example/team-extensions${NC}"
    info "(Example - no real git clone in temp mode)"
fi
pause

section "2. Install from a Local Directory"

step "Copy extensions from a local path"
info "If you've already downloaded or created extensions locally:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    # Create a sample local extension structure
    DEMO_PATH="$EXAMPLE_TEMP_DIR/my-extensions"
    mkdir -p "$DEMO_PATH/rules"
    cat > "$DEMO_PATH/rules/example-rule.md" <<'RULE'
# Example Rule

This is a sample coding rule for demonstration.
RULE
    
    step "Create a sample local extension"
    run_cmd ls -la "$DEMO_PATH/rules/"
    echo
    
    step "Install from local path"
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext install "$DEMO_PATH"
else
    echo -e "${YELLOW}\$ claudeup ext install ~/Downloads/my-extensions${NC}"
    echo -e "${YELLOW}\$ claudeup ext install /path/to/team-shared/extensions${NC}"
    info "(Examples - no real paths in temp mode)"
fi

echo
info "The directory should contain category folders:"
info "  my-extensions/"
info "  ├── rules/"
info "  │   └── coding-standards.md"
info "  ├── agents/"
info "  │   └── reviewer.md"
info "  └── commands/"
info "      └── deploy.sh"
pause

section "3. Install Specific Categories"

step "Choose which categories to install"
info "You can limit what gets installed:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Install only specific categories:"
    echo -e "${YELLOW}\$ claudeup ext install ~/team-ext --category rules${NC}"
    echo -e "${YELLOW}\$ claudeup ext install https://github.com/team/ext --category agents,rules${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext install <path> --category rules,agents${NC}"
    info "(Example - category filtering during install)"
fi

echo
info "This ignores other categories in the source."
pause

section "4. Team Workflow Example"

step "Typical team extension sharing workflow"

info "1. Team creates a shared git repository:"
info "   git clone https://github.com/myteam/claude-extensions"
echo

info "2. Each team member installs from the repo:"
info "   claudeup ext install https://github.com/myteam/claude-extensions"
echo

info "3. Team members enable the extensions they need:"
info "   claudeup ext enable 'rules/*'           # Enable all team rules"
info "   claudeup ext enable agents/code-reviewer  # Enable specific agent"
echo

info "4. When the team updates the repo:"
info "   git pull                                 # Update local clone"
info "   claudeup ext install ./claude-extensions # Reinstall"
pause

section "5. Verify Installation"

step "Check that extensions were installed"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext list

info "New extensions appear in the list (initially disabled)."
pause

section "6. Enable Newly Installed Extensions"

step "Activate the extensions you want to use"

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Enable all newly installed rules:"
    echo -e "${YELLOW}\$ claudeup ext enable 'rules/*'${NC}"
    echo
    info "Or enable selectively:"
    echo -e "${YELLOW}\$ claudeup ext enable rules/team-standards${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext enable 'rules/*'${NC}"
    info "Enable all rules at once"
    echo
    echo -e "${YELLOW}\$ claudeup ext enable agents/reviewer${NC}"
    info "Enable a specific agent"
    info "(Examples - no real extensions in temp mode)"
fi
pause

section "Summary"

success "You can install extensions from external sources"
echo
info "Key commands:"
info "  claudeup ext install <git-url>              Install from git repo"
info "  claudeup ext install <local-path>           Install from local directory"
info "  claudeup ext install <path> --category <c>  Install specific categories"
echo
info "Best practices:"
info "  • Use a team git repo for shared extensions"
info "  • Install first, then selectively enable what you need"
info "  • Review extension contents before enabling (claudeup ext view)"
info "  • Keep team repos updated and reinstall periodically"
echo
info "Next: Combine with profiles to apply extensions at different scopes!"
echo

prompt_cleanup
