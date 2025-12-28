#!/usr/bin/env bash
# ABOUTME: Example showing how to use .claudeup.json for project configuration
# ABOUTME: Demonstrates project-level config file usage

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║             Team Setup: Project Configuration                  ║
╚════════════════════════════════════════════════════════════════╝

Use .claudeup.json to define project-specific Claude configuration
that travels with your repository.

EOF
pause

section "1. The .claudeup.json File"

info "Place .claudeup.json in your project root to define:"
info "  • Required plugins for the project"
info "  • Recommended profiles"
info "  • MCP server configurations"
echo

step "Example .claudeup.json structure"
cat <<'EXAMPLE'
{
  "plugins": [
    "superpowers@superpowers-marketplace",
    "backend-development@claude-code-workflows"
  ]
}
EXAMPLE
pause

section "2. Create a Project Config"

step "Generate .claudeup.json from current state"
info "This captures your project's plugin requirements"
echo

# In real mode, actually create it
if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    info "Creating .claudeup.json in current directory..."
    # Note: claudeup doesn't have a direct command for this yet
    info "Command: claudeup profile save --output .claudeup.json"
else
    info "In your project, run:"
    echo -e "${YELLOW}\$ claudeup profile save --output .claudeup.json${NC}"
fi
pause

section "3. Sync Team Configuration"

step "Apply project configuration"
info "When a teammate clones the repo, they can sync:"
echo
echo -e "${YELLOW}\$ claudeup profile sync${NC}"
echo

info "This installs all plugins defined in .claudeup.json"
pause

section "4. Git Integration"

info "Recommended .gitignore entries:"
cat <<'GITIGNORE'
# Claude Code local settings (personal overrides)
.claude/settings.local.json

# Keep these tracked for team sharing:
# .claude/settings.json
# .claudeup.json
GITIGNORE
pause

section "Summary"

success "You can share Claude configuration via git"
echo
info "Key files:"
info "  .claudeup.json              Project plugin requirements"
info "  .claude/settings.json       Project Claude settings"
info "  .claude/settings.local.json Personal overrides (git-ignored)"
echo

prompt_cleanup
