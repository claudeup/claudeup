#!/usr/bin/env bash
# ABOUTME: Demo showing how user and project profiles layer together
# ABOUTME: Executes real commands to demonstrate combining personal and team configurations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║          Team Setup: Profile Layering in Action                ║
╚════════════════════════════════════════════════════════════════╝

Personal preferences + team requirements combine through scope
layering. This demo applies profiles at user and project scopes
to show how they merge.

EOF
pause

# ===================================================================
section "1. Create Fixture Profiles"
# ===================================================================

step "Create a personal tools profile (for user scope)"
cat > "$CLAUDEUP_HOME/profiles/personal-tools.json" <<'PROFILE'
{
  "name": "personal-tools",
  "description": "Personal productivity tools for all projects",
  "plugins": [
    "superpowers@superpowers-marketplace",
    "elements-of-style@superpowers-marketplace"
  ]
}
PROFILE
success "Created personal-tools.json"
echo

step "Create a team backend profile (for project scope)"
cat > "$CLAUDEUP_HOME/profiles/go-team.json" <<'PROFILE'
{
  "name": "go-team",
  "description": "Shared Go backend team configuration",
  "plugins": [
    "backend-development@claude-code-workflows",
    "tdd-workflows@claude-code-workflows"
  ]
}
PROFILE
success "Created go-team.json"
echo

step "Verify both profiles exist"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

# ===================================================================
section "2. Set Up a Project Directory"
# ===================================================================

step "Create a project with a .claude/ directory"
PROJECT_DIR="$EXAMPLE_TEMP_DIR/team-project"
mkdir -p "$PROJECT_DIR/.claude"
cd "$PROJECT_DIR" || exit 1
success "Working in $PROJECT_DIR"
echo

info "Project scope requires a .claude/ subdirectory in the project root."
pause

# ===================================================================
section "3. Apply Personal Tools at User Scope"
# ===================================================================

step "Apply personal-tools at user scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply personal-tools --user --yes
echo

step "Check profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

step "Inspect user-scope settings"
info "File: $CLAUDE_CONFIG_DIR/settings.json"
cat "$CLAUDE_CONFIG_DIR/settings.json" 2>/dev/null || info "(not found)"
echo

info "personal-tools is the only active profile, so it gets the * marker."
info "These settings live in ~/.claude/settings.json and apply to all projects."
pause

# ===================================================================
section "4. Apply Team Config at Project Scope"
# ===================================================================

step "Apply go-team at project scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply go-team --project --yes
echo

step "Check profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

step "Inspect project-scope settings"
info "File: $PROJECT_DIR/.claude/settings.json"
cat "$PROJECT_DIR/.claude/settings.json" 2>/dev/null || info "(not found)"
echo

info "Both profiles are now active simultaneously:"
info "  * go-team [project]          -- highest precedence"
info "  ○ personal-tools [user]      -- overridden on conflicts"
echo
info "Claude sees plugins from BOTH scopes. If the same setting"
info "appears at both scopes, project wins."
pause

# ===================================================================
section "5. Scope Precedence"
# ===================================================================

info "Claude Code merges settings from all scopes:"
echo
info "  user → project → local"
info "  (lowest)        (highest)"
echo
info "Both profiles contribute their plugins."
info "If the same plugin appears at both scopes,"
info "the higher scope's setting wins."
echo

step "Files created at each scope"
info "User:    $CLAUDE_CONFIG_DIR/settings.json"
info "Project: $PROJECT_DIR/.claude/settings.json"
echo

step "The project registry tracks applied profiles"
info "File: $CLAUDEUP_HOME/projects.json"
cat "$CLAUDEUP_HOME/projects.json" 2>/dev/null || info "(not found)"
pause

# ===================================================================
section "6. Best Practices"
# ===================================================================

info "USER scope (--user): tools you use on EVERY project"
info "  - General productivity tools"
info "  - Writing style plugins"
info "  - Workflow helpers that follow you everywhere"
echo
info "PROJECT scope (--project): team requirements for THIS project"
info "  - Language/framework-specific plugins"
info "  - Security scanning tools"
info "  - Required MCP servers"
echo
info "LOCAL scope (--local): personal overrides for this project"
info "  - Override team settings with your own preferences"
info "  - Temporary experiments or debugging config"
info "  - See 06-scope-apply-demo.sh for a full three-scope example"
echo
info "profile list shows * for highest precedence and ○ for overridden."
info "All scopes are active simultaneously -- they accumulate, not replace."
pause

# ===================================================================
section "Summary"
# ===================================================================

success "Layered personal tools (user) with team config (project)"
echo
info "What we demonstrated:"
info "  1. --user    writes to ~/.claude/settings.json (tools for all projects)"
info "  2. --project writes to .claude/settings.json (team config, git-tracked)"
info ""
info "  profile list shows * for highest-precedence and ○ for overridden"
info "  Both scopes are active simultaneously -- they accumulate, not replace"
info ""
info "New to a team? Apply your personal tools at user scope, then use"
info "local scope (--local) to override any team project settings you"
info "want to customize for yourself."
echo

prompt_cleanup
