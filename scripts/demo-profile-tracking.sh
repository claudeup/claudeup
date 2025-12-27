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

demo "Show all profile operations in the last 24 hours" \
     "$CLAUDEUP_BIN events audit --operation profile --since 24h"

$CLAUDEUP_BIN events audit --operation profile --since 24h | head -100

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

demo "Show when profiles were applied (changes to Claude config)" \
     "$CLAUDEUP_BIN events audit --operation profile --scope local --since 7d"

$CLAUDEUP_BIN events audit --operation profile --scope local --since 7d | head -80

cat <<EOF

${YELLOW}💡 Key insight:${NC}
  - 'scope: local' = changes to ~/.claudeup/profiles/
  - 'scope: global' = changes to ~/.claude/ (what we want!)

EOF

wait_for_enter

# Demo 4: Interactive diff demo
section "4. See Exact Changes with Event Diff"

cat <<EOF
${YELLOW}The power of event-based diffing:${NC}

When you apply a profile, claudeup captures a snapshot of:
  - ~/.claude/plugins/installed_plugins.json (before)
  - ~/.claude/plugins/installed_plugins.json (after)

This lets you see EXACTLY what changed, including:
  - Which plugins were added
  - Which plugins were removed
  - Which settings changed
  - Plugin versions that changed

${GREEN}Let's find a recent profile apply event and diff it...${NC}

EOF

# Get the most recent profile apply event ID
RECENT_EVENT=$($CLAUDEUP_BIN events audit --operation profile --scope local --since 30d 2>/dev/null |
               grep -E '^\[' |
               head -1 |
               grep -oE 'Event [0-9]+' |
               awk '{print $2}' || echo "")

if [[ -n "$RECENT_EVENT" ]]; then
    demo "Show detailed diff of event #$RECENT_EVENT" \
         "$CLAUDEUP_BIN events diff $RECENT_EVENT"

    echo -e "${YELLOW}(Showing first 50 lines of diff)${NC}"
    echo
    $CLAUDEUP_BIN events diff "$RECENT_EVENT" 2>/dev/null | head -50 || echo "No diff available"
else
    cat <<EOF
${YELLOW}No recent profile apply events found in the last 30 days.${NC}

To see this in action:
  1. Apply a profile:     $CLAUDEUP_BIN profile apply <name>
  2. Check events:        $CLAUDEUP_BIN events audit --operation profile
  3. Diff the change:     $CLAUDEUP_BIN events diff <event-id>

EOF
fi

wait_for_enter

# Demo 5: Comparison with static script
section "5. Event-Based vs Static Comparison"

cat <<EOF
${BOLD}Static Comparison (compare-profile.sh):${NC}
  ✓ Shows current drift from saved profile
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
   $ $CLAUDEUP_BIN events audit --since 24h

${GREEN}2. "When did I last apply the frontend profile?"${NC}
   $ $CLAUDEUP_BIN events audit --operation profile | grep frontend

${GREEN}3. "Show me all plugin changes this week"${NC}
   $ $CLAUDEUP_BIN events audit --since 7d --operation profile

${GREEN}4. "What exactly changed when I applied base-tools?"${NC}
   $ $CLAUDEUP_BIN events audit --operation profile | grep base-tools
   $ $CLAUDEUP_BIN events diff <event-id>

${GREEN}5. "Generate weekly report of profile changes"${NC}
   $ $CLAUDEUP_BIN events audit --since 7d --format markdown > weekly-changes.md

${GREEN}6. "Troubleshoot: why is my setup different?"${NC}
   $ $CLAUDEUP_BIN events audit --since 30d
   $ $CLAUDEUP_BIN events diff <suspect-event-id>

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
  • Watch events capture it: $CLAUDEUP_BIN events audit --operation profile
  • Diff the changes: $CLAUDEUP_BIN events diff <event-id>

${GREEN}The event system turns profile management from "set and forget"
into "track, audit, and understand" - crucial for team environments!${NC}

EOF

echo
