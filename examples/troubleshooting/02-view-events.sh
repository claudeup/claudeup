#!/usr/bin/env bash
# ABOUTME: Example showing how to view file operation history
# ABOUTME: Demonstrates events and events diff commands with real data

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║           Troubleshooting: View Events                         ║
╚════════════════════════════════════════════════════════════════╝

See what changes claudeup has made to your configuration over time.
Essential for understanding "what happened?"

EOF
pause

# ===================================================================
section "1. Generate Some Events"
# ===================================================================

step "Create a fixture profile to apply"
cat > "$CLAUDEUP_HOME/profiles/demo-tools.json" <<'PROFILE'
{
  "name": "demo-tools",
  "description": "Demo profile for event tracking",
  "plugins": [
    "superpowers@superpowers-marketplace",
    "elements-of-style@superpowers-marketplace"
  ]
}
PROFILE
success "Created demo-tools.json"
echo

step "Apply the profile to generate events"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply demo-tools --user --yes
pause

# ===================================================================
section "2. View Recent Events"
# ===================================================================

step "See the most recent file operations"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" events --limit 10
echo

info "Events show:"
info "  • When changes happened"
info "  • What files were modified"
info "  • What operation caused the change"
pause

# ===================================================================
section "3. Filter Events"
# ===================================================================

step "Filter events by file path"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" events --file "$CLAUDE_CONFIG_DIR/settings.json"
echo

info "Other filter options:"
info "  --operation <name>   Filter by operation type"
info "  --since <duration>   Filter by time (e.g., 24h, 7d)"
info "  --user / --project / --local   Filter by scope"
pause

# ===================================================================
section "4. View a Diff"
# ===================================================================

step "See exactly what changed in settings.json"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" events diff --file "$CLAUDE_CONFIG_DIR/settings.json"
echo

info "Diffs show before/after snapshots of configuration changes."
info "Use --full for complete nested object comparison."
pause

# ===================================================================
section "Summary"
# ===================================================================

success "You can track all configuration changes"
echo
info "Key commands:"
info "  claudeup events                Show recent events"
info "  claudeup events --since 24h    Filter by time"
info "  claudeup events diff --file    Show what changed"
echo

prompt_cleanup
