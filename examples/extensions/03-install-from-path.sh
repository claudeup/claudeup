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

step "Clone a repo, then install extensions per category"
info "The ext install command works with local paths, so clone first:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Step 1: Clone the repository"
    echo -e "${YELLOW}\$ git clone https://github.com/myteam/claude-extensions ~/team-ext${NC}"
    echo
    info "Step 2: Install each category from the cloned directory"
    echo -e "${YELLOW}\$ claudeup ext install agents ~/team-ext/agents${NC}"
    echo -e "${YELLOW}\$ claudeup ext install rules ~/team-ext/rules${NC}"
    echo -e "${YELLOW}\$ claudeup ext install hooks ~/team-ext/hooks${NC}"
    echo
    info "Files are copied to ~/.claudeup/ext/<category>/ and automatically enabled."
else
    echo -e "${YELLOW}\$ git clone https://github.com/example/team-extensions ~/team-ext${NC}"
    echo -e "${YELLOW}\$ claudeup ext install agents ~/team-ext/agents${NC}"
    echo -e "${YELLOW}\$ claudeup ext install rules ~/team-ext/rules${NC}"
    info "(Example - clone first, then install per category)"
fi
pause

section "2. Install from a Local Directory"

step "Copy extensions from a local path"
info "If you've already downloaded or created extensions locally:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    # Create a sample local extension structure in a temp directory
    DEMO_TEMP=$(mktemp -d "/tmp/claudeup-demo-ext-XXXXXXXXXX")
    DEMO_PATH="$DEMO_TEMP/my-extensions"
    mkdir -p "$DEMO_PATH/rules"
    cat > "$DEMO_PATH/rules/example-rule.md" <<'RULE'
# Example Rule

This is a sample coding rule for demonstration.
RULE
    
    step "Create a sample local extension"
    run_cmd ls -la "$DEMO_PATH/rules/"
    echo
    
    step "Install from local path (one category at a time)"
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext install rules "$DEMO_PATH/rules"
else
    echo -e "${YELLOW}\$ claudeup ext install rules ~/Downloads/my-extensions/rules${NC}"
    echo -e "${YELLOW}\$ claudeup ext install agents /path/to/team-shared/agents${NC}"
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
info "Since install takes one category at a time, just run it for the ones you want:"
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Install only specific categories (run once per category):"
    echo -e "${YELLOW}\$ claudeup ext install rules ~/team-ext/rules${NC}"
    echo -e "${YELLOW}\$ claudeup ext install agents ~/team-ext/agents${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext install rules <path>${NC}"
    echo -e "${YELLOW}\$ claudeup ext install agents <path>${NC}"
    info "(Example - install specific categories by passing them positionally)"
fi

echo
info "Skip categories you don't need — just don't run install for them."
pause

section "4. Team Workflow Example"

step "Typical team extension sharing workflow"

info "1. Team creates a shared git repository:"
info "   git clone https://github.com/myteam/claude-extensions"
echo

info "2. Each team member clones and installs per category:"
info "   git clone https://github.com/myteam/claude-extensions ~/team-ext"
info "   claudeup ext install rules ~/team-ext/rules"
info "   claudeup ext install agents ~/team-ext/agents"
echo

info "3. Extensions are enabled automatically on install."
info "   Disable what you don't need:"
info "   claudeup ext disable rules 'unwanted-*'    # Disable specific rules"
info "   claudeup ext disable agents code-reviewer   # Disable specific agent"
echo

info "4. When the team updates the repo:"
info "   cd ~/team-ext && git pull                      # Update local clone"
info "   claudeup ext install rules ~/team-ext/rules    # Reinstall categories"
pause

section "5. Verify Installation"

step "Check that extensions were installed"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" ext list --full

info "Newly installed extensions are automatically enabled."
info "Use 'ext list' without --full for a summary, or specify a category."
pause

section "6. Manage Installed Extensions"

step "Disable extensions you don't need"

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Extensions are enabled on install. Disable what you don't need:"
    echo -e "${YELLOW}\$ claudeup ext disable rules 'unwanted-*'${NC}"
    echo
    info "Re-enable later:"
    echo -e "${YELLOW}\$ claudeup ext enable rules team-standards${NC}"
else
    echo -e "${YELLOW}\$ claudeup ext disable rules 'unwanted-*'${NC}"
    info "Disable rules you don't need"
    echo
    echo -e "${YELLOW}\$ claudeup ext enable agents reviewer${NC}"
    info "Re-enable a specific agent"
    info "(Examples - no real extensions in temp mode)"
fi
pause

section "Summary"

success "You can install extensions from external sources"
echo
info "Key commands:"
info "  claudeup ext install <category> <path>      Install from local directory"
info "  claudeup ext view <category> <name>         View extension contents"
echo
info "Best practices:"
info "  • Use a team git repo for shared extensions"
info "  • Clone the repo locally, then install per category"
info "  • Extensions are auto-enabled on install; disable what you don't need"
info "  • Review extension contents with: claudeup ext view <category> <name>"
info "  • Keep team repos updated and reinstall periodically"
echo
info "Next: Combine with profiles to apply extensions at different scopes!"
echo

# Clean up demo temp directory if created in real mode
if [[ -n "${DEMO_TEMP:-}" && -d "$DEMO_TEMP" ]]; then
    rm -rf "$DEMO_TEMP"
fi

prompt_cleanup
