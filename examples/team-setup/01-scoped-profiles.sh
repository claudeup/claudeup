#!/usr/bin/env bash
# ABOUTME: Example showing how scopes work in Claude Code
# ABOUTME: Demonstrates user, project, and local scope differences

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║             Team Setup: Understanding Scopes                   ║
╚════════════════════════════════════════════════════════════════╝

Claude Code uses three configuration scopes. Understanding them
is key to effective team collaboration.

EOF
pause

section "1. The Three Scopes"

info "USER scope (~/.claude/settings.json)"
info "  • Your personal defaults"
info "  • Applies to all projects"
info "  • Not shared with team"
echo

info "PROJECT scope (.claude/settings.json)"
info "  • Shared team configuration"
info "  • Checked into git"
info "  • Everyone on the team gets these settings"
echo

info "LOCAL scope (.claude/settings.local.json)"
info "  • Your personal overrides for this project"
info "  • Git-ignored (not shared)"
info "  • Highest precedence"
pause

section "2. Scope Precedence"

info "Settings are merged with later scopes winning:"
echo
info "  user → project → local"
info "  (lowest)        (highest)"
echo
info "Example: If user scope enables plugin A"
info "         and local scope disables plugin A"
info "         → Plugin A is disabled"
pause

section "3. Apply to Different Scopes"

step "See which scopes have active profiles"
run_cmd "$EXAMPLE_CLAUDEUP_BIN profile list"

step "Apply a profile to a specific scope"
info "Commands:"
echo -e "${YELLOW}\$ claudeup profile apply myprofile --scope user    # Personal default${NC}"
echo -e "${YELLOW}\$ claudeup profile apply myprofile --scope project # Team setting${NC}"
echo -e "${YELLOW}\$ claudeup profile apply myprofile --scope local   # Local override${NC}"
pause

section "4. View Scope Contents"

step "See what's configured at each scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN scope list" || \
    info "Scope list would show files and their contents"
pause

section "Summary"

success "You understand Claude Code's scope system"
echo
info "Best practices:"
info "  • User scope: Your personal productivity tools"
info "  • Project scope: Team-required plugins and settings"
info "  • Local scope: Personal tweaks that don't affect team"
echo

prompt_cleanup
