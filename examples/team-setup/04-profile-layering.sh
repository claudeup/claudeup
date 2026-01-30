#!/usr/bin/env bash
# ABOUTME: Example showing how user and project profiles layer together
# ABOUTME: Demonstrates combining personal and team configurations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║          Team Setup: Profile Layering Strategy                 ║
╚════════════════════════════════════════════════════════════════╝

Combine personal preferences with team requirements by layering
profiles from different scopes.

EOF
pause

section "1. The Layering Concept"

info "Two levels of profiles work together:"
echo
info "  USER profiles (~/.claudeup/profiles/)"
info "    → Your personal tools and preferences"
info "    → Available in all projects"
info "    → Only you have these"
echo
info "  PROJECT profiles (.claudeup/profiles/)"
info "    → Team-shared configurations"
info "    → Specific to this repository"
info "    → Everyone on the team gets these"
pause

section "2. Example: Full-Stack Team"

step "User profile: Personal productivity tools"
cat <<'PROFILE1'
# ~/.claudeup/profiles/my-tools.json
{
  "name": "my-tools",
  "description": "My personal coding helpers",
  "plugins": [
    "superpowers@superpowers-marketplace",
    "elements-of-style@superpowers-marketplace"
  ]
}
PROFILE1
echo
info "These are YOUR tools that you use everywhere"
pause

step "Project profile: Team backend requirements"
cat <<'PROFILE2'
# .claudeup/profiles/backend-team.json
{
  "name": "backend-team",
  "description": "Required for backend development",
  "plugins": [
    "backend-development@claude-code-workflows",
    "security-scanning@claude-code-workflows"
  ]
}
PROFILE2
echo
info "These are plugins EVERYONE on the team needs"
pause

section "3. How to Set Up Layering"

step "Save your personal profile"
echo -e "${YELLOW}\$ claudeup profile save my-tools${NC}"
echo
info "This saves to ~/.claudeup/profiles/my-tools.json"
echo
info "Your personal profile follows you to any project"
pause

step "Team lead saves and applies project profile"
echo -e "${YELLOW}\$ claudeup profile save backend-team${NC}"
echo -e "${YELLOW}\$ claudeup profile apply backend-team --project${NC}"
echo
info "This writes settings to .claude/settings.json for team sharing"
echo
info "Then commit and share with the team"
pause

section "4. Applying Layered Profiles"

step "Apply user profile at user scope"
echo -e "${YELLOW}\$ claudeup profile apply my-tools --user${NC}"
echo
info "Your personal tools are now active globally"
pause

step "Apply project profile at project scope"
echo -e "${YELLOW}\$ claudeup profile apply backend-team --project${NC}"
echo
info "Team requirements are active for this project"
pause

step "View the combined configuration"
echo -e "${YELLOW}\$ claudeup profile list${NC}"
echo
info "Example output showing layered profiles:"
cat <<'EXAMPLE'
Your profiles (4)

○ my-tools          Personal tools [user]
* backend-team      Team config [project]
  frontend-dev      Frontend setup [project]
  data-science      DS tools [user]
EXAMPLE
echo
info "○ = active at lower scope (user)"
info "* = active at highest scope (project)"
echo
info "Both are active! User settings + project settings combine"
pause

section "5. Scope Precedence"

info "When the same setting exists in multiple scopes:"
echo
info "  user → project → local"
info "  (lowest)        (highest)"
echo
info "Later scopes override earlier ones"
echo
step "Example: Plugin enabled in user, disabled in project"
info "  User scope:    Plugin A = enabled"
info "  Project scope: Plugin A = disabled"
info "  Result:        Plugin A = disabled (project wins)"
pause

section "6. Best Practices"

info "USER profiles (personal):"
info "  • General productivity tools"
info "  • Writing style plugins"
info "  • Your preferred workflow helpers"
echo
info "PROJECT profiles (team):"
info "  • Language/framework specific plugins"
info "  • Security scanning tools"
info "  • Required MCP servers"
echo
info "LOCAL scope (personal overrides):"
info "  • Temporary experiments"
info "  • Debugging configurations"
info "  • Sensitive personal settings"
pause

section "7. Recommended Workflow"

step "Developer setup (one time)"
echo -e "${YELLOW}\$ claudeup profile save my-tools${NC}"
echo -e "${YELLOW}\$ claudeup profile apply my-tools --user${NC}"
echo
step "Each project (after clone)"
echo -e "${YELLOW}\$ claudeup profile apply${NC}"
echo
info "Now you have: personal tools + team requirements"
pause

section "Summary"

success "Layer profiles for personalized team collaboration"
echo
info "Strategy:"
info "  • User scope: Your personal toolkit (all projects)"
info "  • Project scope: Team requirements (this project)"
info "  • Local scope: Personal overrides (git-ignored)"
echo
info "Both user and project profiles can be active simultaneously"
info "They combine, with project settings winning on conflicts"
echo

prompt_cleanup
