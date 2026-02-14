# Devcontainer Demo Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create `examples/team-setup/04-devcontainer-demo.sh` that uses claudeup-lab to spin up real Docker containers for three team members with profile stacking.

**Architecture:** Single bash script following the existing `common.sh` example pattern. Creates fixture profiles in a temp `CLAUDEUP_HOME`, initializes a temp git repo, starts three claudeup-lab containers, verifies profile state inside each, and cleans up.

**Tech Stack:** Bash, claudeup-lab CLI, Docker, common.sh shared library

---

### Task 1: Create the demo script

**Files:**

- Create: `examples/team-setup/04-devcontainer-demo.sh`

**Reference files (read first for patterns):**

- `examples/team-setup/02-isolated-workspace-demo.sh` -- structure, common.sh usage, fixture profiles
- `examples/lib/common.sh` -- available helpers

**Step 1: Write the script**

Create `examples/team-setup/04-devcontainer-demo.sh` with the following content:

```bash
#!/usr/bin/env bash
# ABOUTME: End-to-end demo using claudeup-lab to create devcontainer environments
# ABOUTME: Spins up real Docker containers for Alice, Bob, and Charlie with profile stacking

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"

# This script creates real Docker containers via claudeup-lab.
# The --real flag is not supported -- labs always use isolated temp profiles.
if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    error "This demo always uses isolated temp directories for profiles."
    error "The --real flag is not supported."
    exit 1
fi

# ---------------------------------------------------------------------------
# Prerequisite checks
# ---------------------------------------------------------------------------

if ! command -v claudeup-lab &>/dev/null; then
    error "claudeup-lab not found in PATH"
    error "Install: curl -fsSL https://raw.githubusercontent.com/claudeup/claudeup-lab/main/scripts/install.sh | bash"
    exit 1
fi
success "Found claudeup-lab: $(command -v claudeup-lab)"

resolve_claudeup_bin
check_claudeup_installed

step "Running claudeup-lab doctor to verify prerequisites"
if ! claudeup-lab doctor; then
    error "claudeup-lab doctor reported failures. Fix the issues above and retry."
    exit 1
fi
echo

# ---------------------------------------------------------------------------
# Temp directory and profile setup
# ---------------------------------------------------------------------------

EXAMPLE_TEMP_DIR=$(mktemp -d "/tmp/claudeup-example-XXXXXXXXXX")
export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/claudeup-home"
mkdir -p "$CLAUDEUP_HOME/profiles"

# Track lab names for cleanup
LAB_NAMES=()

cleanup_labs() {
    for lab_name in "${LAB_NAMES[@]}"; do
        claudeup-lab rm --lab "$lab_name" --force 2>/dev/null || true
    done
}

on_error() {
    local exit_code=$?
    echo ""
    error "Script failed with exit code $exit_code"
    cleanup_labs
    if [[ -n "$EXAMPLE_TEMP_DIR" && -d "$EXAMPLE_TEMP_DIR" ]]; then
        warn "Preserving temp directory for debugging: $EXAMPLE_TEMP_DIR"
    fi
    exit $exit_code
}

trap on_error ERR

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║        Team Setup: Devcontainer Demo                           ║
╚════════════════════════════════════════════════════════════════╝

Three engineers -- Alice, Bob, and Charlie -- each get their own
Docker container via claudeup-lab. The team profile provides a
shared base configuration, and personal profiles layer on top.

Unlike the isolated workspace demo (02), which simulates isolation
with environment variables, this demo creates real containers with
separate filesystems, networks, and process spaces.

EOF
pause

# ===================================================================
section "1. Create Fixture Profiles"
# ===================================================================

step "Create team and personal profiles in temp CLAUDEUP_HOME"
info "CLAUDEUP_HOME=$CLAUDEUP_HOME"
echo

# -- Team profile --
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

# -- Alice's personal profile --
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

# -- Bob's personal profile --
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
pause

# ===================================================================
section "2. Create a Temp Git Repository"
# ===================================================================

step "Initialize a minimal git repo for claudeup-lab"
info "claudeup-lab requires a git project to create worktrees."
echo

DEMO_PROJECT="$EXAMPLE_TEMP_DIR/demo-project"
mkdir -p "$DEMO_PROJECT"
git -C "$DEMO_PROJECT" init --quiet
git -C "$DEMO_PROJECT" commit --allow-empty -m "Initial commit" --quiet
success "Created git repo at $DEMO_PROJECT"
pause

# ===================================================================
section "3. Alice Starts Her Lab"
# ===================================================================

info "Alice uses the team profile as a base and layers her personal tools."
echo

step "Start Alice's lab"
run_cmd claudeup-lab start \
    --project "$DEMO_PROJECT" \
    --base-profile go-backend-team \
    --profile alice-tools \
    --name alice-lab
LAB_NAMES+=("alice-lab")
echo
success "Alice's lab is running"
pause

# ===================================================================
section "4. Bob Starts His Lab"
# ===================================================================

info "Bob uses the same team base profile with his own review tools."
echo

step "Start Bob's lab"
run_cmd claudeup-lab start \
    --project "$DEMO_PROJECT" \
    --base-profile go-backend-team \
    --profile bob-tools \
    --name bob-lab
LAB_NAMES+=("bob-lab")
echo
success "Bob's lab is running"
pause

# ===================================================================
section "5. Charlie Starts His Lab"
# ===================================================================

info "Charlie has no personal profile -- just the team configuration."
echo

step "Start Charlie's lab"
run_cmd claudeup-lab start \
    --project "$DEMO_PROJECT" \
    --profile go-backend-team \
    --name charlie-lab
LAB_NAMES+=("charlie-lab")
echo
success "Charlie's lab is running"
pause

# ===================================================================
section "6. Verify All Labs"
# ===================================================================

step "List all running labs"
run_cmd claudeup-lab list
echo

step "Check profile state inside each container"
for lab_name in alice-lab bob-lab charlie-lab; do
    info "$lab_name:"
    run_cmd claudeup-lab exec --lab "$lab_name" -- claudeup profile list
    echo
done
pause

# ===================================================================
section "7. Cleanup"
# ===================================================================

step "Remove all labs"
for lab_name in alice-lab bob-lab charlie-lab; do
    run_cmd claudeup-lab rm --lab "$lab_name" --force
done
success "All labs removed"
echo

# ===================================================================
section "Summary"
# ===================================================================

success "Devcontainer demo complete"
echo
info "Key takeaways:"
info ""
info "  claudeup-lab provides true container isolation"
info "    Each lab gets its own filesystem, network, and process space"
info ""
info "  --base-profile layers team config under personal tools"
info "    Alice and Bob share go-backend-team as a foundation"
info "    Their personal plugins are applied on top"
info ""
info "  Charlie uses just the team profile"
info "    No personal layer needed -- works with --profile alone"
info ""
info "  Labs are ephemeral -- destroy and recreate freely"
info "    claudeup-lab rm cleans up containers, volumes, and worktrees"
info ""
info "  Compare with 02-isolated-workspace-demo.sh"
info "    Isolated workspaces: lightweight, no Docker, env var isolation"
info "    Devcontainers: heavyweight, real containers, full isolation"
echo

prompt_cleanup
```

**Step 2: Make it executable**

Run: `chmod +x examples/team-setup/04-devcontainer-demo.sh`

**Step 3: Run shellcheck**

Run: `shellcheck examples/team-setup/04-devcontainer-demo.sh`
Expected: No errors or warnings (may need shellcheck disables for sourced common.sh variables)

**Step 4: Syntax check**

Run: `bash -n examples/team-setup/04-devcontainer-demo.sh`
Expected: No output (syntax valid)

**Step 5: Commit**

```bash
git add examples/team-setup/04-devcontainer-demo.sh
git commit -m "feat: add devcontainer demo using claudeup-lab

Three team members (Alice, Bob, Charlie) each get real Docker
containers with profile stacking via claudeup-lab."
```

---

### Task 2: Update README

**Files:**

- Modify: `examples/team-setup/README.md`

**Step 1: Add script to table**

Add a row to the Scripts table:

```
| `04-devcontainer-demo.sh`       | End-to-end demo using claudeup-lab to create real Docker containers for three team members with profile stacking via `--base-profile`          |
```

**Step 2: Update suggested order**

Append to the suggested order paragraph:

```
Finally, `04-devcontainer-demo.sh` shows the same team pattern using real
Docker containers instead of environment variable isolation.
```

**Step 3: Add prerequisites note to important details**

Add a bullet:

```
- `04-devcontainer-demo.sh` requires Docker and `claudeup-lab` installed.
  Run `claudeup-lab doctor` to check prerequisites. It creates real containers
  that take a few minutes to start.
```

**Step 4: Commit**

```bash
git add examples/team-setup/README.md
git commit -m "docs: add devcontainer demo to team-setup README"
```

---

### Task 3: Manual verification

**Requires:** Docker running, claudeup-lab installed

**Step 1: Run the demo interactively**

Run: `./examples/team-setup/04-devcontainer-demo.sh`

Verify:

- Prerequisites pass (doctor output clean)
- Three labs start without errors
- `claudeup-lab list` shows all three
- `claudeup profile list` inside each container shows expected profiles
- Cleanup removes all three labs
- No temp files left behind

**Step 2: Run non-interactively**

Run: `./examples/team-setup/04-devcontainer-demo.sh --non-interactive`

Verify: Same as above but without pause prompts.

**Step 3: Test error recovery**

Kill Docker mid-run and verify the ERR trap fires and attempts cleanup.

---

## Verification

```bash
# Syntax check (no Docker needed)
shellcheck examples/team-setup/04-devcontainer-demo.sh
bash -n examples/team-setup/04-devcontainer-demo.sh

# Full run (Docker required)
./examples/team-setup/04-devcontainer-demo.sh

# Non-interactive
./examples/team-setup/04-devcontainer-demo.sh --non-interactive
```
