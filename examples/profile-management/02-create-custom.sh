#!/usr/bin/env bash
# ABOUTME: Example showing how to create custom profiles from scratch
# ABOUTME: Demonstrates writing profile JSON, showing, applying, and saving profiles

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Profile Management: Create Custom Profile              ║
╚════════════════════════════════════════════════════════════════╝

Create a custom profile by writing JSON directly, inspect it,
apply it, and then capture the current state as a new profile.

EOF
pause

# ===================================================================
section "1. Create a Custom Profile"
# ===================================================================

step "Write a profile JSON file to the profiles directory"
info "Profiles live in \$CLAUDEUP_HOME/profiles/ as JSON files."
info "The interactive wizard (profile create) builds these for you,"
info "but you can also write them directly."
echo

cat > "$CLAUDEUP_HOME/profiles/go-backend.json" <<'PROFILE'
{
  "name": "go-backend",
  "description": "Go backend development with TDD workflows",
  "plugins": [
    "backend-development@claude-code-workflows",
    "tdd-workflows@claude-code-workflows"
  ]
}
PROFILE
success "Created go-backend.json"
pause

# ===================================================================
section "2. Inspect the Profile"
# ===================================================================

step "Show the profile contents"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile show go-backend
echo

step "Confirm it appears in the profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

# ===================================================================
section "3. Apply the Profile"
# ===================================================================

step "Apply go-backend at user scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply go-backend --user --yes
echo

step "Verify it is now active"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

# ===================================================================
section "4. Save Current State as a New Profile"
# ===================================================================

step "Capture the current configuration as a new profile"
info "profile save snapshots whatever is currently configured"
info "and writes it to a new (or existing) profile file."
echo
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile save my-snapshot --yes
echo

step "Verify the saved profile exists"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

step "Show the saved profile contents"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile show my-snapshot
pause

# ===================================================================
section "Summary"
# ===================================================================

success "Created, applied, and saved custom profiles"
echo
info "What we demonstrated:"
info "  1. Write a profile JSON to \$CLAUDEUP_HOME/profiles/"
info "  2. Inspect it with: claudeup profile show <name>"
info "  3. Apply it with:   claudeup profile apply <name> --user"
info "  4. Capture state:   claudeup profile save <name>"
echo
info "For guided creation, use the interactive wizard:"
info "  claudeup profile create"
echo

prompt_cleanup
