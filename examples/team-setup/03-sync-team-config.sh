#!/usr/bin/env bash
# ABOUTME: Example showing how to sync team configuration
# ABOUTME: Demonstrates profile sync command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║             Team Setup: Sync Team Configuration                ║
╚════════════════════════════════════════════════════════════════╝

Keep your Claude setup in sync with your team's project requirements.

EOF
pause

section "1. Check for Project Configuration"

step "Look for .claudeup.json in the current project"
if [[ -f ".claudeup.json" ]]; then
    info "Found .claudeup.json:"
    cat .claudeup.json
else
    info "No .claudeup.json found in current directory"
    info "This file defines project plugin requirements"
fi
pause

section "2. Sync Configuration"

step "Install plugins defined in .claudeup.json"
info "The sync command ensures you have all required plugins"
echo

run_cmd "$EXAMPLE_CLAUDEUP_BIN profile sync" || \
    info "Sync would install any missing plugins"
pause

section "3. Onboarding Workflow"

info "When joining a project with Claude configuration:"
echo
info "  1. Clone the repository"
info "     git clone <repo-url>"
echo
info "  2. Sync Claude configuration"
info "     cd <project>"
info "     claudeup profile sync"
echo
info "  3. (Optional) Add personal overrides"
info "     claudeup profile apply my-tools --scope local"
pause

section "4. Keeping in Sync"

info "After pulling changes that modify .claudeup.json:"
echo
echo -e "${YELLOW}\$ git pull${NC}"
echo -e "${YELLOW}\$ claudeup profile sync${NC}"
echo
info "This installs any new plugins the team has added"
pause

section "Summary"

success "You can stay in sync with team configuration"
echo
info "Key commands:"
info "  claudeup profile sync   Install project requirements"
echo
info "Workflow:"
info "  1. Team adds plugin to .claudeup.json"
info "  2. Team commits and pushes"
info "  3. You pull and run 'claudeup profile sync'"
echo

prompt_cleanup
