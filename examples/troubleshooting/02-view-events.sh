#!/usr/bin/env bash
# ABOUTME: Example showing how to view file operation history
# ABOUTME: Demonstrates events and events diff commands

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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

section "1. View Recent Events"

step "See the most recent file operations"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" events --limit 10

info "Events show:"
info "  • When changes happened"
info "  • What files were modified"
info "  • What operation caused the change"
pause

section "2. Filter Events"

step "Find specific types of changes"
echo

info "Filter by file:"
echo -e "${YELLOW}\$ claudeup events --file ~/.claude/settings.json${NC}"
echo

info "Filter by operation:"
echo -e "${YELLOW}\$ claudeup events --operation profile${NC}"
echo

info "Filter by time:"
echo -e "${YELLOW}\$ claudeup events --since 24h${NC}"
pause

section "3. View Detailed Diffs"

step "See exactly what changed in a file"
echo -e "${YELLOW}\$ claudeup events diff --file ~/.claude/settings.json${NC}"
echo
info "Diffs show before/after snapshots of configuration changes."
info "Use --full for complete nested object comparison."
pause

section "Summary"

success "You can track all configuration changes"
echo
info "Key commands:"
info "  claudeup events                Show recent events"
info "  claudeup events --since 24h    Filter by time"
info "  claudeup events diff --file    Show what changed"
echo

prompt_cleanup
