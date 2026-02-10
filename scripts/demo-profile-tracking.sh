#!/usr/bin/env bash
# ABOUTME: Demo script showing how event tracking provides profile change visibility
# ABOUTME: Demonstrates the power of temporal awareness vs static comparison

set -euo pipefail

CLAUDEUP_BIN="${CLAUDEUP_BIN:-./bin/claudeup}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

section() {
    echo
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${BLUE}  $1${NC}"
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo
}

demo() {
    echo -e "${MAGENTA}▶${NC} ${BOLD}$1${NC}"
    echo -e "${YELLOW}$ $2${NC}"
    echo
}

wait_for_enter() {
    echo
    echo -e "${GREEN}Press ENTER to continue...${NC}"
    read -r
}

cat <<EOF
${BOLD}═══════════════════════════════════════════════════════════════${NC}
${BOLD}          Profile Change Tracking with Event Monitoring       ${NC}
${BOLD}═══════════════════════════════════════════════════════════════${NC}

This demo shows how claudeup's event tracking system provides
comprehensive visibility into profile changes over time.

${YELLOW}Key advantages over static comparison:${NC}
  ✓ See WHEN changes happened (temporal awareness)
  ✓ See WHO made changes (user vs claudeup operations)
  ✓ See complete change history, not just current state
  ✓ Diff any two points in time
  ✓ Audit trail for troubleshooting

EOF

wait_for_enter

# Demo 1: Show current active profile
section "1. Current Active Profile"

demo "Show which profile is currently active" \
     "$CLAUDEUP_BIN profile list | grep '^\*'"

$CLAUDEUP_BIN profile list | grep '^\*' || echo "No active profile"

wait_for_enter

# Demo 2: Timeline of recent profile operations
section "2. Recent Profile Activity Timeline"

demo "Show all settings changes in the last 24 hours" \
     "$CLAUDEUP_BIN events --operation \"settings update\" --since 24h"

$CLAUDEUP_BIN events --operation "settings update" --since 24h | head -100

cat <<EOF

${YELLOW}💡 Notice:${NC} You can see:
  - When profiles were saved/applied
  - Which files were affected
  - Size changes for each operation
  - Chronological timeline of changes

EOF

wait_for_enter

# Demo 3: Find profile apply events
section "3. Finding Profile Apply Operations"

demo "Show when settings were updated at user scope (changes to ~/.claude/)" \
     "$CLAUDEUP_BIN events --operation \"settings update\" --user --since 7d"

$CLAUDEUP_BIN events --operation "settings update" --user --since 7d | head -80

cat <<EOF

${YELLOW}💡 Key insight:${NC}
  - 'scope: user' = changes to ~/.claude/ (user-level config)
  - 'scope: project' = changes to .claude/ (project-level config)
  - 'scope: local' = changes to .claude/settings.local.json

EOF

wait_for_enter

# Demo 4: Interactive diff demo
section "4. See Exact Changes with Event Diff"

cat <<EOF
${YELLOW}The power of event-based diffing:${NC}

When claudeup modifies a configuration file, it captures snapshots
of the file before and after the change. This lets you see EXACTLY
what changed, including:
  - Which plugins were added or removed
  - Which settings changed
  - Plugin versions that changed

${GREEN}Let's diff some configuration files...${NC}

EOF

demo "Show most recent change to settings.json" \
     "$CLAUDEUP_BIN events diff --file ~/.claude/settings.json"

$CLAUDEUP_BIN events diff --file ~/.claude/settings.json 2>/dev/null || echo "No diff available for settings.json"

echo

demo "Show most recent change to plugins (with full nested diff)" \
     "$CLAUDEUP_BIN events diff --file ~/.claude/plugins/installed_plugins.json --full"

$CLAUDEUP_BIN events diff --file ~/.claude/plugins/installed_plugins.json --full 2>/dev/null | head -50 || echo "No diff available for installed_plugins.json"

wait_for_enter

# Demo 5: Comparison with static script
section "5. Event-Based vs Static Comparison"

cat <<EOF
${BOLD}Static Comparison (compare-profile.sh):${NC}
  ✓ Shows differences from saved profile
  ✓ Fast and simple
  ✗ No history - only shows current state
  ✗ Can't see what changed when
  ✗ Can't diff arbitrary points in time

${BOLD}Event-Based Tracking (claudeup events):${NC}
  ✓ Complete change history
  ✓ See when and why changes happened
  ✓ Diff any two snapshots
  ✓ Audit trail for compliance/debugging
  ✓ Works even after profile deleted
  ✗ Requires events to have been captured

${YELLOW}💡 Best practice:${NC} Use BOTH!
  - Static comparison for quick checks
  - Event tracking for deep investigation

EOF

wait_for_enter

# Demo 6: Practical workflows
section "6. Practical Workflows"

cat <<EOF
${BOLD}Common workflows with event tracking:${NC}

${GREEN}1. "What did I change yesterday?"${NC}
   $ $CLAUDEUP_BIN events --since 24h

${GREEN}2. "When did my settings last change?"${NC}
   $ $CLAUDEUP_BIN events --operation "settings update"

${GREEN}3. "Show me all plugin changes this week"${NC}
   $ $CLAUDEUP_BIN events --since 7d --operation "plugin update"

${GREEN}4. "What exactly changed when I applied a profile?"${NC}
   $ $CLAUDEUP_BIN events diff --file ~/.claude/settings.json --full

${GREEN}5. "What changed in my plugins?"${NC}
   $ $CLAUDEUP_BIN events diff --file ~/.claude/plugins/installed_plugins.json

${GREEN}6. "Troubleshoot: why is my setup different?"${NC}
   $ $CLAUDEUP_BIN events --since 30d
   $ $CLAUDEUP_BIN events diff --file ~/.claude/settings.json --full

EOF

wait_for_enter

# Summary
section "Summary"

cat <<EOF
${BOLD}Event-based profile tracking gives you:${NC}

📊 ${BOLD}Visibility${NC}     - See complete change history, not just current state
⏰ ${BOLD}Temporal${NC}       - Know when changes happened and in what order
🔍 ${BOLD}Detail${NC}         - Exact diffs of what changed in configuration files
🛡️  ${BOLD}Audit Trail${NC}   - Compliance and troubleshooting capabilities
🔄 ${BOLD}Reversibility${NC}  - Can see state at any point in time

${YELLOW}Next steps:${NC}
  • Try applying a profile: $CLAUDEUP_BIN profile apply <name>
  • Watch events capture it: $CLAUDEUP_BIN events --operation "settings update"
  • Diff the changes: $CLAUDEUP_BIN events diff --file ~/.claude/settings.json

${GREEN}The event system turns profile management from "set and forget"
into "track, audit, and understand" - crucial for team environments!${NC}

EOF

echo
