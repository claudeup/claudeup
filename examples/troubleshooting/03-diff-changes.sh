#!/usr/bin/env bash
# ABOUTME: Example showing how to see detailed file changes
# ABOUTME: Demonstrates events diff command

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
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

section "1. Find a Change to Inspect"

step "List recent events to find an interesting change"
run_cmd "$EXAMPLE_CLAUDEUP_BIN events --limit 5"
pause

section "2. View the Diff"

step "See what changed in a specific file"
info "The diff command shows before/after comparison"
echo

info "Command syntax:"
echo -e "${YELLOW}\$ claudeup events diff --file ~/.claude/settings.json${NC}"
echo

info "This shows the most recent change to that file"
pause

section "3. Full Diff Mode"

step "Get detailed nested changes"
info "Use --full for complete recursive diff of nested objects"
echo

info "Example output:"
cat <<'EXAMPLE'
~ plugins:
  ~ superpowers@superpowers-marketplace:
    ~ scope: "project" → "user"
    ~ installedAt: "2025-12-26T05:14:20Z" → "2025-12-28T10:30:00Z"
  + newplugin@marketplace:
    + scope: "user" (added)
EXAMPLE
echo

info "Symbols: + added, - removed, ~ modified"
pause

section "4. Practical Use Case"

info "Common debugging workflow:"
echo
info "  1. Something broke after a change"
info "  2. Run: claudeup events --since 1h"
info "  3. Find the relevant file change"
info "  4. Run: claudeup events diff --file <path> --full"
info "  5. See exactly what changed"
info "  6. Decide: revert or fix forward"
pause

section "Summary"

success "You can see exactly what changed and when"
echo
info "Key commands:"
info "  claudeup events diff --file <path>        Basic diff"
info "  claudeup events diff --file <path> --full Detailed diff"
echo

prompt_cleanup
