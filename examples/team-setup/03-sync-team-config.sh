#!/usr/bin/env bash
# ABOUTME: Example showing the complete team sync workflow
# ABOUTME: Demonstrates profile sync with project-local profiles

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║             Team Setup: Sync Team Configuration                ║
╚════════════════════════════════════════════════════════════════╝

Keep your Claude setup in sync with your team's project requirements.
The sync command discovers profiles from both project and user locations.

EOF
pause

section "1. What Sync Discovers"

step "The sync command checks multiple locations"
echo
info "Project profiles (checked first):"
info "  .claudeup/profiles/           Project-local profiles"
info "  .claudeup.json                Plugin requirements"
echo
info "User profiles (fallback):"
info "  ~/.claudeup/profiles/         Personal profiles"
echo
info "Project profiles take precedence over user profiles"
pause

section "2. Team Lead Workflow"

step "Creating a shared profile for the team"
echo
info "The team lead configures Claude as desired, then saves:"
echo
echo -e "${YELLOW}\$ claudeup profile save team-config --scope project${NC}"
echo
info "This creates .claudeup/profiles/team-config.json"
echo
step "Commit the profile to version control"
echo -e "${YELLOW}\$ git add .claudeup/profiles/${NC}"
echo -e "${YELLOW}\$ git commit -m \"Add shared Claude profile\"${NC}"
echo -e "${YELLOW}\$ git push${NC}"
pause

section "3. Team Member Workflow"

step "After cloning or pulling changes"
echo
info "Run sync to discover and apply project profiles:"
echo
echo -e "${YELLOW}\$ claudeup profile sync${NC}"
echo
info "Sync will:"
info "  • Find profiles in .claudeup/profiles/"
info "  • Install required plugins"
info "  • Report what was installed"
pause

section "4. See Where Profiles Come From"

step "The profile list command shows source locations"
echo
echo -e "${YELLOW}\$ claudeup profile list${NC}"
echo
info "Example output:"
cat <<'EXAMPLE'
Your profiles (3)

  base-tools        Personal toolkit [user]
* team-config       Team configuration [project]
  frontend-dev      Frontend setup [project]
EXAMPLE
echo
info "[user] = from ~/.claudeup/profiles/"
info "[project] = from .claudeup/profiles/"
pause

section "5. Try It: Check Project Configuration"

step "Look for configuration in the current project"
if [[ -d ".claudeup/profiles" ]]; then
    info "Found project profiles directory:"
    run_cmd ls -la .claudeup/profiles/ || true
elif [[ -f ".claudeup.json" ]]; then
    info "Found .claudeup.json:"
    cat .claudeup.json
else
    info "No project configuration found in current directory"
    info "This is expected in the isolated demo environment"
fi
pause

section "6. Run Sync"

step "Install plugins from project requirements"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile sync || \
    info "Sync found no project configuration (expected in demo)"
pause

section "7. Complete Onboarding Workflow"

info "When joining a project with shared Claude configuration:"
echo
info "  1. Clone the repository"
echo -e "     ${YELLOW}git clone <repo-url>${NC}"
echo
info "  2. Enter the project"
echo -e "     ${YELLOW}cd <project>${NC}"
echo
info "  3. Sync Claude configuration"
echo -e "     ${YELLOW}claudeup profile sync${NC}"
echo
info "  4. (Optional) Apply the team profile"
echo -e "     ${YELLOW}claudeup profile apply team-config${NC}"
echo
info "  5. (Optional) Add personal overrides"
echo -e "     ${YELLOW}claudeup profile apply my-tools --scope local${NC}"
pause

section "8. Staying in Sync"

info "After pulling changes that add/modify profiles:"
echo
echo -e "${YELLOW}\$ git pull${NC}"
echo -e "${YELLOW}\$ claudeup profile sync${NC}"
echo
info "This ensures you have the latest team plugins installed"
pause

section "Summary"

success "You can sync with team configuration automatically"
echo
info "Key concepts:"
info "  • Project profiles live in .claudeup/profiles/"
info "  • Sync discovers profiles from project first, then user"
info "  • profile list shows [project] or [user] source"
echo
info "Key commands:"
info "  claudeup profile sync              Install project requirements"
info "  claudeup profile list              See profile sources"
info "  claudeup profile apply <name>      Apply a specific profile"
echo

prompt_cleanup
