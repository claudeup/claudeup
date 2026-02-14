#!/usr/bin/env bash
# ABOUTME: End-to-end demo using claudeup-lab to spin up real Docker containers
# ABOUTME: Creates three team members (Alice, Bob, Charlie) with profile stacking

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"

# This script uses claudeup-lab (Docker containers) for isolation,
# so the --real flag does not apply.
if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    error "This demo uses claudeup-lab for container-based isolation."
    error "The --real flag is not supported."
    exit 1
fi

resolve_claudeup_bin
check_claudeup_installed

# ---------------------------------------------------------------------------
# Prerequisites: claudeup-lab must be available
# ---------------------------------------------------------------------------
if ! command -v claudeup-lab &>/dev/null; then
    error "claudeup-lab not found in PATH"
    error "Please install claudeup-lab first:"
    error "  go install github.com/claudeup/claudeup-lab/cmd/claudeup-lab@latest"
    exit 1
fi
success "Found claudeup-lab: $(command -v claudeup-lab)"

step "Running claudeup-lab doctor to check prerequisites"
if ! claudeup-lab doctor; then
    error "claudeup-lab doctor reported failures. Fix the issues above and retry."
    exit 1
fi
echo

# ---------------------------------------------------------------------------
# Temp directory and CLAUDEUP_HOME
# ---------------------------------------------------------------------------
EXAMPLE_TEMP_DIR=$(mktemp -d "/tmp/claudeup-example-XXXXXXXXXX")
export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/claudeup-home"
mkdir -p "$CLAUDEUP_HOME/profiles"

# ---------------------------------------------------------------------------
# Lab tracking and error handler
# ---------------------------------------------------------------------------
LAB_NAMES=()

on_error() {
    local exit_code=$?
    echo ""
    error "Script failed with exit code $exit_code"

    if [[ ${#LAB_NAMES[@]} -gt 0 ]]; then
        warn "Cleaning up labs..."
        for lab in "${LAB_NAMES[@]}"; do
            claudeup-lab rm --lab "$lab" --force 2>/dev/null || true
        done
    fi

    if [[ -n "$EXAMPLE_TEMP_DIR" && -d "$EXAMPLE_TEMP_DIR" ]]; then
        warn "Preserving temp directory for debugging: $EXAMPLE_TEMP_DIR"
        warn "Contents:"
        ls -la "$EXAMPLE_TEMP_DIR" 2>/dev/null || true
    fi

    exit "$exit_code"
}
trap on_error ERR

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║       Team Setup: Devcontainer Demo (claudeup-lab)             ║
╚════════════════════════════════════════════════════════════════╝

Three engineers -- Alice, Bob, and Charlie -- collaborate on a Go
backend project. Each gets a real Docker container with their own
Claude Code environment, managed by claudeup-lab.

  Alice   = team profile + personal tools (superpowers)
  Bob     = team profile + personal tools (style + PR review)
  Charlie = team profile only (no personal tools)

EOF
pause

# ===================================================================
section "1. Create Fixture Profiles"
# ===================================================================

# claudeup-lab applies all profiles at user scope inside the container,
# so both team and personal profiles target "user" here (unlike demo 02
# which uses project scope for team config).
step "Create team profile: go-backend-team"
cat > "$CLAUDEUP_HOME/profiles/go-backend-team.json" <<'PROFILE'
{
  "name": "go-backend-team",
  "description": "Shared Go backend team configuration",
  "perScope": {
    "user": {
      "plugins": [
        "backend-development@claude-code-workflows",
        "tdd-workflows@claude-code-workflows"
      ]
    }
  }
}
PROFILE
success "Created go-backend-team.json (team profile)"

step "Create Alice's personal profile: alice-tools"
cat > "$CLAUDEUP_HOME/profiles/alice-tools.json" <<'PROFILE'
{
  "name": "alice-tools",
  "description": "Alice's personal productivity tools",
  "perScope": {
    "user": {
      "plugins": [
        "superpowers@superpowers-marketplace"
      ]
    }
  }
}
PROFILE
success "Created alice-tools.json (Alice's personal profile)"

step "Create Bob's personal profile: bob-tools"
cat > "$CLAUDEUP_HOME/profiles/bob-tools.json" <<'PROFILE'
{
  "name": "bob-tools",
  "description": "Bob's code review and documentation tools",
  "perScope": {
    "user": {
      "plugins": [
        "elements-of-style@superpowers-marketplace",
        "pr-review-toolkit@claude-plugins-official"
      ]
    }
  }
}
PROFILE
success "Created bob-tools.json (Bob's personal profile)"
echo

info "Profiles stored in: $CLAUDEUP_HOME/profiles/"
pause

# ===================================================================
section "2. Create a Temp Git Repository"
# ===================================================================

TEMP_REPO="$EXAMPLE_TEMP_DIR/sample-project"
mkdir -p "$TEMP_REPO"
git -C "$TEMP_REPO" init --quiet
git -C "$TEMP_REPO" commit --allow-empty --message "initial" --quiet
success "Created temp git repo: $TEMP_REPO"
pause

# ===================================================================
section "3. Start Labs for Each Team Member"
# ===================================================================

step "Start Alice's lab (team + personal profile)"
run_cmd claudeup-lab start \
    --project "$TEMP_REPO" \
    --base-profile go-backend-team \
    --profile alice-tools \
    --name alice-lab
LAB_NAMES+=("alice-lab")
echo

step "Start Bob's lab (team + personal profile)"
run_cmd claudeup-lab start \
    --project "$TEMP_REPO" \
    --base-profile go-backend-team \
    --profile bob-tools \
    --name bob-lab
LAB_NAMES+=("bob-lab")
echo

step "Start Charlie's lab (team profile only)"
run_cmd claudeup-lab start \
    --project "$TEMP_REPO" \
    --profile go-backend-team \
    --name charlie-lab
LAB_NAMES+=("charlie-lab")
pause

# ===================================================================
section "4. Verify Lab Status"
# ===================================================================

step "List all running labs"
run_cmd claudeup-lab list
echo

step "Verify Alice's profile configuration"
run_cmd claudeup-lab exec --lab alice-lab -- claudeup profile list
echo

step "Verify Bob's profile configuration"
run_cmd claudeup-lab exec --lab bob-lab -- claudeup profile list
echo

step "Verify Charlie's profile configuration"
run_cmd claudeup-lab exec --lab charlie-lab -- claudeup profile list
pause

# ===================================================================
section "5. Cleanup Labs"
# ===================================================================

for lab in "${LAB_NAMES[@]}"; do
    step "Removing $lab"
    run_cmd claudeup-lab rm --lab "$lab" --force
    echo
done
LAB_NAMES=()
success "All labs removed"
pause

# ===================================================================
section "Summary"
# ===================================================================

success "Devcontainer demo complete"
echo
info "Key takeaways:"
info ""
info "  claudeup-lab provides real Docker containers for each team member"
info "    No env var tricks -- each person gets a full isolated environment"
info ""
info "  --base-profile layers a shared team profile under personal tools"
info "    Alice and Bob get team config + their own plugins"
info "    Charlie gets only the team config"
info ""
info "  Profile stacking uses user scope inside the container"
info "    claudeup-lab handles profile application at container startup"
info ""
info "  Labs are disposable -- spin up, test, tear down"
info "    No risk to your host Claude Code configuration"
echo

prompt_cleanup
