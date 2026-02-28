#!/usr/bin/env bash
# ABOUTME: Demo script showing last-applied breadcrumb feature
# ABOUTME: Demonstrates how profile diff and save default to the last-applied profile

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
${BOLD}          Last-Applied Breadcrumb for Profile Defaults        ${NC}
${BOLD}═══════════════════════════════════════════════════════════════${NC}

When you apply a profile, claudeup records a "breadcrumb" --
a note of which profile was applied at which scope and when.

This lets ${BOLD}profile diff${NC} and ${BOLD}profile save${NC} work without arguments:
  - No need to remember which profile you applied
  - Scope-aware: tracks user, project, and local separately
  - Highest-precedence scope wins (local > project > user)

EOF

wait_for_enter

# Demo 1: Apply a profile and see the breadcrumb
section "1. Apply a Profile (Breadcrumb Written Automatically)"

demo "Apply a profile at user scope" \
     "$CLAUDEUP_BIN profile apply <name> --user --yes"

echo -e "${YELLOW}After applying, claudeup writes:${NC}"
echo -e "  ~/.claudeup/last-applied.json"
echo
echo -e "${YELLOW}Contents look like:${NC}"
cat <<'EXAMPLE'
  {
    "user": {
      "profile": "my-setup",
      "appliedAt": "2026-02-28T06:00:00Z"
    }
  }
EXAMPLE

if [ -f "${CLAUDEUP_HOME:-$HOME/.claudeup}/last-applied.json" ]; then
    echo
    echo -e "${GREEN}Your actual breadcrumb file:${NC}"
    cat "${CLAUDEUP_HOME:-$HOME/.claudeup}/last-applied.json"
fi

wait_for_enter

# Demo 2: Diff without arguments
section "2. Profile Diff -- No Arguments Needed"

demo "Diff defaults to the last-applied profile" \
     "$CLAUDEUP_BIN profile diff"

echo -e "${YELLOW}Without arguments, claudeup:${NC}"
echo "  1. Reads the breadcrumb file"
echo "  2. Picks the highest-precedence scope (local > project > user)"
echo "  3. Uses that profile name for the diff"
echo
echo -e "${YELLOW}You can also target a specific scope:${NC}"
echo -e "  $ $CLAUDEUP_BIN profile diff --user     ${BLUE}# use user-scope breadcrumb${NC}"
echo -e "  $ $CLAUDEUP_BIN profile diff --project   ${BLUE}# use project-scope breadcrumb${NC}"
echo -e "  $ $CLAUDEUP_BIN profile diff --local     ${BLUE}# use local-scope breadcrumb${NC}"

wait_for_enter

# Demo 3: Save without arguments
section "3. Profile Save -- No Arguments Needed"

demo "Save defaults to the last-applied profile name" \
     "$CLAUDEUP_BIN profile save"

echo -e "${YELLOW}Without arguments, claudeup:${NC}"
echo "  1. Reads the breadcrumb file"
echo "  2. Uses the last-applied profile name"
echo '  3. Prints: Saving to "my-setup" (applied Feb 28, user scope)'
echo "  4. Overwrites the existing profile with current live state"
echo
echo -e "${YELLOW}Explicit name still works:${NC}"
echo -e "  $ $CLAUDEUP_BIN profile save my-new-name   ${BLUE}# ignores breadcrumb${NC}"

wait_for_enter

# Demo 4: Multi-scope breadcrumbs
section "4. Multi-Scope Breadcrumbs"

echo -e "${YELLOW}Each scope gets its own breadcrumb entry:${NC}"
echo
cat <<'EXAMPLE'
  {
    "user": {
      "profile": "base-tools",
      "appliedAt": "2026-02-27T10:00:00Z"
    },
    "project": {
      "profile": "team-setup",
      "appliedAt": "2026-02-28T08:00:00Z"
    }
  }
EXAMPLE
echo
echo -e "${YELLOW}Precedence for no-args diff/save:${NC}"
echo "  local > project > user"
echo
echo "  In this example, 'profile diff' defaults to 'team-setup'"
echo "  because project scope has higher precedence than user."

wait_for_enter

# Demo 5: Delete and rename maintain breadcrumbs
section "5. Delete and Rename Keep Breadcrumbs Clean"

echo -e "${BOLD}Profile delete:${NC}"
echo "  Removes breadcrumb entries for the deleted profile."
echo "  Other scopes' entries are preserved."
echo
echo -e "${BOLD}Profile rename:${NC}"
echo "  Updates breadcrumb entries from old name to new name."
echo "  Breadcrumb stays valid after rename."
echo
echo -e "${YELLOW}Example:${NC}"
echo -e "  $ $CLAUDEUP_BIN profile rename my-setup better-name"
echo "  Breadcrumb 'my-setup' becomes 'better-name' automatically."

wait_for_enter

# Summary
section "Summary"

cat <<EOF
${BOLD}The breadcrumb system gives you:${NC}

  ${BOLD}Convenience${NC}    -- diff and save work without remembering profile names
  ${BOLD}Scope-aware${NC}    -- tracks each scope independently
  ${BOLD}Precedence${NC}     -- highest scope wins for default resolution
  ${BOLD}Consistency${NC}    -- delete and rename keep breadcrumbs in sync
  ${BOLD}Non-blocking${NC}   -- breadcrumb errors never fail an apply operation

${YELLOW}Quick reference:${NC}
  $ $CLAUDEUP_BIN profile apply my-setup --yes   # writes breadcrumb
  $ $CLAUDEUP_BIN profile diff                    # uses breadcrumb
  $ $CLAUDEUP_BIN profile save                    # uses breadcrumb
  $ $CLAUDEUP_BIN profile diff --user             # specific scope
  $ $CLAUDEUP_BIN profile diff my-setup           # explicit name

EOF

echo
