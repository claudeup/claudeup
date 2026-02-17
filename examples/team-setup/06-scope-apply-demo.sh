#!/usr/bin/env bash
# ABOUTME: End-to-end demo applying profiles at all three scopes (user, project, local)
# ABOUTME: Shows how settings accumulate across scopes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║        Scope Apply Demo: All Three Scopes in Action            ║
╚════════════════════════════════════════════════════════════════╝

This demo applies profiles at each scope (user, project, local)
and shows how settings accumulate across scopes.

EOF
pause

# ===================================================================
section "1. Create Fixture Profiles"
# ===================================================================

step "Create a user-scope profile (personal defaults)"
cat > "$CLAUDEUP_HOME/profiles/base-tools.json" <<'PROFILE'
{
  "name": "base-tools",
  "description": "Personal defaults for all projects",
  "plugins": [
    "superpowers@superpowers-marketplace",
    "elements-of-style@superpowers-marketplace"
  ]
}
PROFILE
success "Created base-tools.json"
echo

step "Create a project-scope profile (team configuration)"
cat > "$CLAUDEUP_HOME/profiles/team-backend.json" <<'PROFILE'
{
  "name": "team-backend",
  "description": "Shared Go backend team settings",
  "plugins": [
    "backend-development@claude-code-workflows",
    "tdd-workflows@claude-code-workflows"
  ]
}
PROFILE
success "Created team-backend.json"
echo

step "Create a local-scope profile (personal project overrides)"
cat > "$CLAUDEUP_HOME/profiles/my-overrides.json" <<'PROFILE'
{
  "name": "my-overrides",
  "description": "Personal overrides for this project only",
  "plugins": [
    "pr-review-toolkit@claude-plugins-official"
  ]
}
PROFILE
success "Created my-overrides.json"
echo

step "Verify all three profiles exist"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

# ===================================================================
section "2. Set Up a Project Directory"
# ===================================================================

step "Create a fake project with a .claude/ directory"
PROJECT_DIR="$EXAMPLE_TEMP_DIR/my-project"
mkdir -p "$PROJECT_DIR/.claude"
cd "$PROJECT_DIR" || exit 1
success "Working in $PROJECT_DIR"
echo

info "Project and local scopes require being inside a project directory"
info "with a .claude/ subdirectory."
pause

# ===================================================================
section "3. Apply at User Scope (lowest precedence)"
# ===================================================================

step "Apply base-tools at user scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply base-tools --user --yes
echo

step "Check profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

step "Inspect user-scope settings"
info "File: $CLAUDE_CONFIG_DIR/settings.json"
cat "$CLAUDE_CONFIG_DIR/settings.json" 2>/dev/null || info "(not found)"
echo

info "base-tools is applied at user scope -- these settings apply to all projects."
pause

# ===================================================================
section "4. Apply at Project Scope (overrides user)"
# ===================================================================

step "Apply team-backend at project scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply team-backend --project --yes
echo

step "Check profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

step "Inspect project-scope settings"
info "File: $PROJECT_DIR/.claude/settings.json"
cat "$PROJECT_DIR/.claude/settings.json" 2>/dev/null || info "(not found)"
echo

info "Both profiles are now applied. Project scope takes precedence over user."
info "  team-backend [project] -- higher precedence"
info "  base-tools [user]      -- lower precedence"
pause

# ===================================================================
section "5. Apply at Local Scope (highest precedence)"
# ===================================================================

step "Apply my-overrides at local scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply my-overrides --local --yes
echo

step "Check profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

step "Inspect local-scope settings"
info "File: $PROJECT_DIR/.claude/settings.local.json"
cat "$PROJECT_DIR/.claude/settings.local.json" 2>/dev/null || info "(not found)"
echo

info "Local scope overlays on top of both user and project settings."
info "All three scopes are now applied simultaneously:"
info "  my-overrides [local]   -- highest precedence"
info "  team-backend [project] -- middle precedence"
info "  base-tools [user]      -- lowest precedence"
info ""
info "Claude sees all plugins from every scope. If the same setting"
info "appears at multiple scopes, the highest scope wins."
pause

# ===================================================================
section "6. Scope Precedence Summary"
# ===================================================================

info "Claude Code merges settings from all scopes:"
echo
info "  user → project → local"
info "  (lowest)        (highest)"
echo
info "All three profiles contribute their plugins."
info "If the same plugin appears at multiple scopes,"
info "the highest scope's setting wins."
echo

step "Files created at each scope"
info "User:    $CLAUDE_CONFIG_DIR/settings.json"
info "Project: $PROJECT_DIR/.claude/settings.json"
info "Local:   $PROJECT_DIR/.claude/settings.local.json"
echo

step "Use profile status to see the effective configuration"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile status || info "(no profile status available)"
pause

# ===================================================================
section "Summary"
# ===================================================================

success "Applied profiles at all three scopes"
echo
info "What we demonstrated:"
info "  1. --user    writes to ~/.claude/settings.json (personal defaults)"
info "  2. --project writes to .claude/settings.json (team, git-tracked)"
info "  3. --local   writes to .claude/settings.local.json (personal, git-ignored)"
info ""
info "  Use 'profile status' to see the effective configuration across all scopes"
info "  All scopes are applied simultaneously -- they accumulate, not replace"
echo

prompt_cleanup
