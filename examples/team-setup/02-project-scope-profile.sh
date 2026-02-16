#!/usr/bin/env bash
# ABOUTME: Example showing how to apply a profile at project scope
# ABOUTME: Demonstrates project-scoped configuration in a single directory

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║        Team Setup: Project-Scoped Profile Application          ║
╚════════════════════════════════════════════════════════════════╝

Apply a profile at project scope to share team configuration through
git, without affecting your personal user-scope settings.

This example shows the workflow for setting up team configuration
in a single project directory.

EOF
pause

section "1. Create a Team Profile"

step "Create a profile for your team's shared configuration"

TEAM_PROFILE_PATH="$CLAUDEUP_HOME/profiles/backend-team.json"

cat > "$TEAM_PROFILE_PATH" <<'PROFILE'
{
  "name": "backend-team",
  "description": "Shared backend team configuration",
  "perScope": {
    "project": {
      "plugins": [
        "backend-development@claude-code-workflows",
        "tdd-workflows@claude-code-workflows"
      ],
      "extensions": {
        "rules": ["api-design.md", "error-handling.md"],
        "agents": ["reviewer.md"]
      }
    }
  }
}
PROFILE

success "Created backend-team.json profile"
echo
info "This profile specifies project-scoped plugins and extensions."
info "The perScope.project section controls what goes in .claude/"
pause

section "2. Create Team Extensions"

step "Create the extension files referenced in the profile"

mkdir -p "$CLAUDEUP_HOME/ext/rules"
mkdir -p "$CLAUDEUP_HOME/ext/agents"

cat > "$CLAUDEUP_HOME/ext/rules/api-design.md" <<'RULE'
# API Design Guidelines

- Use RESTful conventions for HTTP endpoints
- Return appropriate status codes (200, 201, 400, 404, 500)
- Include proper error messages in response bodies
- Version APIs with URL path prefix (e.g., /v1/...)
- Document all endpoints with OpenAPI/Swagger
RULE

cat > "$CLAUDEUP_HOME/ext/rules/error-handling.md" <<'RULE'
# Error Handling Standards

- Always check and handle errors explicitly
- Log errors with context (request ID, user ID, timestamp)
- Return user-friendly error messages (no stack traces in production)
- Use structured error types (not just strings)
- Implement retry logic for transient failures
RULE

cat > "$CLAUDEUP_HOME/ext/agents/reviewer.md" <<'AGENT'
# Code Reviewer

You are a meticulous code reviewer for backend services.

Focus areas:
- Error handling and edge cases
- API contract consistency
- Security considerations (input validation, auth)
- Performance implications (N+1 queries, unnecessary loops)
- Test coverage

Be constructive and suggest improvements, not just problems.
AGENT

success "Created team extension files"
info "  • rules/api-design.md"
info "  • rules/error-handling.md"
info "  • agents/reviewer.md"
pause

section "3. Create a Project Directory"

step "Set up a project workspace"

PROJECT_DIR="$EXAMPLE_TEMP_DIR/my-backend-project"
mkdir -p "$PROJECT_DIR"
cd "$PROJECT_DIR" || exit 1

success "Created project directory: $PROJECT_DIR"
echo
info "This simulates a git repository for a team project."
info "In real usage, this would be your actual git working directory."
pause

section "4. Apply Profile at Project Scope"

step "Apply the team profile to this project"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply backend-team --project --yes

echo
info "This created .claude/ directory with team configuration."
pause

section "5. Examine Project Configuration"

step "Explore what was created in .claude/"

run_cmd ls -la .claude/ || info "(No .claude directory created)"
echo

step "Check project settings"
if [[ -f .claude/settings.json ]]; then
    info "Project settings.json:"
    run_cmd cat .claude/settings.json
else
    info "(No settings.json - plugins listed there when installed)"
fi
echo

step "Check project rules"
if [[ -d .claude/rules ]]; then
    info "Project rules (copied from extensions into .claude/):"
    run_cmd ls -la .claude/rules/
    echo
    info "Content of api-design.md:"
    run_cmd cat .claude/rules/api-design.md
else
    info "(No rules directory)"
fi
echo

step "Check project agents"
if [[ -d .claude/agents ]]; then
    info "Project agents:"
    run_cmd ls -la .claude/agents/
    echo
    info "Content of reviewer.md:"
    run_cmd cat .claude/agents/reviewer.md
else
    info "(No agents directory)"
fi
pause

section "6. Understanding Project Scope Files"

info "The .claude/ directory contains:"
echo
info "  settings.json        - Team plugin configuration"
info "  rules/               - Copies of team rules"
info "  agents/              - Copies of team agents"
info "  settings.local.json  - Personal overrides (git-ignored)"
echo
info "Files in .claude/ (except .local.json) should be committed to git."
info "This way, everyone on the team gets the same configuration."
pause

section "7. Commit to Git (Simulated)"

step "In a real project, you would commit .claude/ and .claudeup/ to git"

info "Typical workflow:"
echo -e "${YELLOW}\$ git add .claude/ .claudeup/${NC}"
echo -e "${YELLOW}\$ git commit -m 'Add team Claude Code configuration'${NC}"
echo -e "${YELLOW}\$ git push${NC}"
echo
info ".claude/          - Applied settings, rules, and agents"
info ".claudeup/profiles/ - Profile definition for team sharing"
echo
info "Team members clone the repo and get both directories automatically."
info "No need for them to re-apply the profile!"
pause

section "8. Personal Overrides with Local Scope"

step "Team members can add personal overrides without affecting git"

info "Example: Disable a team plugin locally"
echo -e "${YELLOW}\$ claudeup plugin disable backend-development --local${NC}"
echo
info "This creates .claude/settings.local.json (git-ignored)"
info "Your personal preference doesn't affect the team."
echo

info "Or apply a personal profile to local scope:"
echo -e "${YELLOW}\$ claudeup profile apply my-personal-tools --local${NC}"
pause

section "9. Verify Profile Application"

step "Check which profiles are active"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list

echo
info "Look for 'backend-team' with scope=project"
pause

section "10. Compare Scopes"

step "Understand where settings are stored"

info "USER scope (~/.claude/settings.json):"
info "  • Your personal defaults across all projects"
info "  • Not in git"
echo

info "PROJECT scope (.claude/settings.json):"
info "  • Team configuration for this project"
info "  • Committed to git"
echo

info "LOCAL scope (.claude/settings.local.json):"
info "  • Your personal overrides for this project"
info "  • Git-ignored (not shared)"
echo

info "Precedence: user → project → local (highest)"
pause

section "Summary"

success "You've applied a team profile at project scope"
echo
info "Key workflow:"
info "  1. Create a profile with perScope.project section"
info "  2. Create extension files in CLAUDEUP_HOME/ext/"
info "  3. Apply profile with --project flag"
info "  4. Commit .claude/ to git (except .local.json)"
info "  5. Team members clone and get configuration automatically"
echo
info "Benefits:"
info "  ✓ Team configuration travels through git"
info "  ✓ No manual setup for new team members"
info "  ✓ Personal preferences via local scope don't conflict"
info "  ✓ Easy to update: change profile, re-apply, commit"
echo
info "Next step:"
info "  • See 03-isolated-workspace-demo.sh for multi-member simulation"
echo

prompt_cleanup
