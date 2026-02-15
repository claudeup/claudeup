#!/usr/bin/env bash
# ABOUTME: Example showing how to see detailed file changes
# ABOUTME: Demonstrates events diff command with before/after data

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Troubleshooting: Diff Changes                        ║
╚════════════════════════════════════════════════════════════════╝

See exactly what changed in a file operation.
Essential for understanding why something broke.

EOF
pause

# ===================================================================
section "1. Create Two Profiles"
# ===================================================================

step "Create the first profile"
cat > "$CLAUDEUP_HOME/profiles/starter-kit.json" <<'PROFILE'
{
  "name": "starter-kit",
  "description": "Basic starter plugins",
  "plugins": [
    "superpowers@superpowers-marketplace"
  ]
}
PROFILE
success "Created starter-kit.json"
echo

step "Create a second profile with different plugins"
cat > "$CLAUDEUP_HOME/profiles/full-kit.json" <<'PROFILE'
{
  "name": "full-kit",
  "description": "Extended plugin set",
  "plugins": [
    "superpowers@superpowers-marketplace",
    "elements-of-style@superpowers-marketplace",
    "tdd-workflows@claude-code-workflows"
  ]
}
PROFILE
success "Created full-kit.json"
pause

# ===================================================================
section "2. Apply Profiles Sequentially"
# ===================================================================

step "Apply starter-kit first (creates the baseline)"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply starter-kit --user --yes
echo

step "Apply full-kit to overwrite (creates the change)"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply full-kit --user --yes
pause

# ===================================================================
section "3. View the Diff"
# ===================================================================

step "See what changed in settings.json (default mode)"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" events diff --file "$CLAUDE_CONFIG_DIR/settings.json"
echo

info "Default mode truncates nested objects as {...} for readability."
pause

# ===================================================================
section "4. Full Diff Mode"
# ===================================================================

step "See detailed nested changes with --full"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" events diff --file "$CLAUDE_CONFIG_DIR/settings.json" --full
echo

info "Symbols: + added, - removed, ~ modified"
pause

# ===================================================================
section "5. Practical Use Case"
# ===================================================================

info "Common debugging workflow:"
echo
info "  1. Something broke after a change"
info "  2. Run: claudeup events --since 1h"
info "  3. Find the relevant file change"
info "  4. Run: claudeup events diff --file <path> --full"
info "  5. See exactly what changed"
info "  6. Decide: revert or fix forward"
pause

# ===================================================================
section "Summary"
# ===================================================================

success "You can see exactly what changed and when"
echo
info "Key commands:"
info "  claudeup events diff --file <path>        Basic diff"
info "  claudeup events diff --file <path> --full  Detailed diff"
echo

prompt_cleanup
