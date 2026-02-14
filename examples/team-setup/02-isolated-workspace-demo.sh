#!/usr/bin/env bash
# ABOUTME: End-to-end demo simulating three team members with isolated environments
# ABOUTME: Creates fixture profiles and runs real commands per member using CLAUDE_CONFIG_DIR

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"

# This script manages multiple CLAUDE_CONFIG_DIR values (one per team member),
# so it cannot use setup_environment which sets a single directory.
if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    error "This demo always uses isolated temp directories."
    error "The --real flag is not supported."
    exit 1
fi

resolve_claudeup_bin
check_claudeup_installed
check_claude_config_dir_override

EXAMPLE_TEMP_DIR=$(mktemp -d "/tmp/claudeup-example-XXXXXXXXXX")
trap_preserve_on_error

PROJECT_DIR="$EXAMPLE_TEMP_DIR/project"

# ---------------------------------------------------------------------------
# Helper: switch active team member
# ---------------------------------------------------------------------------
switch_member() {
    local name="$1"
    export CLAUDE_CONFIG_DIR="$EXAMPLE_TEMP_DIR/$name/claude-config"
    export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/$name/claudeup-home"
    info "Switched to $name (CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR)"
}

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║        Team Setup: Isolated Workspace Demo                     ║
╚════════════════════════════════════════════════════════════════╝

Three engineers -- Alice, Bob, and Charlie -- collaborate on a Go
backend project. Each has an isolated CLAUDE_CONFIG_DIR so their
personal plugins never collide, while the shared project directory
gives everyone the same team configuration.

EOF
pause

# ===================================================================
section "1. Setting Up the Team"
# ===================================================================

step "Create the shared project directory"
mkdir -p "$PROJECT_DIR"
success "Created $PROJECT_DIR"
echo

step "Create isolated directories for each team member"

for member in alice bob charlie; do
    mkdir -p "$EXAMPLE_TEMP_DIR/$member/claude-config/plugins"
    mkdir -p "$EXAMPLE_TEMP_DIR/$member/claudeup-home/profiles"
done
success "Created alice/, bob/, charlie/ under $EXAMPLE_TEMP_DIR"
echo

info "Directory layout:"
info "  $EXAMPLE_TEMP_DIR/"
info "  ├── project/              # Shared project (git repo)"
info "  ├── alice/"
info "  │   ├── claude-config/    # CLAUDE_CONFIG_DIR"
info "  │   └── claudeup-home/    # CLAUDEUP_HOME"
info "  ├── bob/"
info "  │   ├── claude-config/"
info "  │   └── claudeup-home/"
info "  └── charlie/"
info "      ├── claude-config/"
info "      └── claudeup-home/"
echo

step "Create fixture profiles"

# -- Team profile (shared by everyone) --
cat > "$EXAMPLE_TEMP_DIR/alice/claudeup-home/profiles/go-backend-team.json" <<'PROFILE'
{
  "name": "go-backend-team",
  "description": "Shared Go backend team configuration",
  "perScope": {
    "project": {
      "plugins": [
        "backend-development@claude-code-workflows",
        "tdd-workflows@claude-code-workflows"
      ],
      "localItems": {
        "rules": ["golang.md"],
        "agents": ["reviewer.md"]
      }
    }
  }
}
PROFILE
success "Created go-backend-team.json (team profile)"

# -- Alice's personal profile --
cat > "$EXAMPLE_TEMP_DIR/alice/claudeup-home/profiles/alice-tools.json" <<'PROFILE'
{
  "name": "alice-tools",
  "description": "Alice's personal productivity tools",
  "perScope": {
    "user": {
      "plugins": [
        "superpowers@superpowers-marketplace"
      ],
      "localItems": {
        "rules": ["coding-standards.md"]
      }
    }
  }
}
PROFILE
success "Created alice-tools.json (Alice's personal profile)"

# -- Bob's personal profile --
cat > "$EXAMPLE_TEMP_DIR/bob/claudeup-home/profiles/bob-tools.json" <<'PROFILE'
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

step "Create fixture local items for the team profile"

mkdir -p "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/rules"
mkdir -p "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/agents"

cat > "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/rules/golang.md" <<'RULE'
# Go Coding Rules

- Follow Effective Go guidelines
- Use gofmt for formatting
- Handle all errors explicitly
- Prefer table-driven tests
RULE
success "Created local/rules/golang.md"

cat > "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/rules/coding-standards.md" <<'RULE'
# Coding Standards

- Write clear, descriptive variable names
- Keep functions under 40 lines
- Document all exported symbols
RULE
success "Created local/rules/coding-standards.md"

cat > "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/agents/reviewer.md" <<'AGENT'
# Code Reviewer

You are a thorough code reviewer focused on Go best practices.
Check for error handling, naming conventions, and test coverage.
AGENT
success "Created local/agents/reviewer.md"
pause

# ===================================================================
section "2. Alice Sets Up the Team Project"
# ===================================================================

switch_member alice
cd "$PROJECT_DIR" || exit 1
echo

step "Apply the team profile at project scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply go-backend-team --project --yes
echo

step "Check project-scoped files"
info "Rules directory:"
ls -la "$PROJECT_DIR/.claude/rules/" 2>/dev/null || info "  (no rules directory yet)"
echo
info "Agents directory:"
ls -la "$PROJECT_DIR/.claude/agents/" 2>/dev/null || info "  (no agents directory yet)"
echo

step "Apply Alice's personal profile at user scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply alice-tools --yes
echo

step "View Alice's profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

# ===================================================================
section "3. Bob Joins the Team"
# ===================================================================

switch_member bob
cd "$PROJECT_DIR" || exit 1
echo

step "Copy team profile from Alice (simulating team sharing)"
cp "$EXAMPLE_TEMP_DIR/alice/claudeup-home/profiles/go-backend-team.json" \
   "$EXAMPLE_TEMP_DIR/bob/claudeup-home/profiles/"
success "Copied go-backend-team.json to Bob's profiles"
echo

# Bob also needs the local items that the team profile references
step "Copy team local items to Bob (simulating shared tooling)"
mkdir -p "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/rules"
mkdir -p "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/agents"
cp "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/rules/golang.md" \
   "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/rules/"
cp "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/agents/reviewer.md" \
   "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/agents/"
success "Copied local items to Bob"
echo

step "Apply team profile for Bob"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply go-backend-team --project --yes
echo

step "Apply Bob's personal profile"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply bob-tools --yes
echo

step "View Bob's profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

step "Verify project rules are still present (shared across members)"
info "Contents of project golang rule:"
cat "$PROJECT_DIR/.claude/rules/golang.md" 2>/dev/null || info "  (not found)"
pause

# ===================================================================
section "4. Charlie Joins with Minimal Setup"
# ===================================================================

switch_member charlie
cd "$PROJECT_DIR" || exit 1
echo

step "Charlie has no personal profile -- just list what's available"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list || info "No profiles configured yet"
echo

info "Charlie can still work in the project. The project-scoped files"
info "that Alice and Bob applied (in .claude/) are regular files on disk."
info "Claude Code reads them regardless of who created them."
echo

step "Verify Charlie sees the project configuration"
if [[ -f "$PROJECT_DIR/.claude/settings.json" ]]; then
    info "Project settings.json exists:"
    head -20 "$PROJECT_DIR/.claude/settings.json"
else
    info "No project settings.json (team plugins would appear here)"
fi
pause

# ===================================================================
section "5. Compare Team Members"
# ===================================================================

info "All three members share the same project directory, but each has"
info "different personal (user-scope) settings."
echo

step "Project-scoped settings (shared by everyone)"
if [[ -f "$PROJECT_DIR/.claude/settings.json" ]]; then
    info "Contents of $PROJECT_DIR/.claude/settings.json:"
    head -20 "$PROJECT_DIR/.claude/settings.json"
else
    info "(no project settings.json)"
fi
echo

step "Alice's user-scoped settings"
if [[ -f "$EXAMPLE_TEMP_DIR/alice/claude-config/settings.json" ]]; then
    info "Contents of alice/claude-config/settings.json:"
    head -20 "$EXAMPLE_TEMP_DIR/alice/claude-config/settings.json"
else
    info "(no user settings for Alice)"
fi
echo

step "Bob's user-scoped settings"
if [[ -f "$EXAMPLE_TEMP_DIR/bob/claude-config/settings.json" ]]; then
    info "Contents of bob/claude-config/settings.json:"
    head -20 "$EXAMPLE_TEMP_DIR/bob/claude-config/settings.json"
else
    info "(no user settings for Bob)"
fi
echo

step "Charlie's user-scoped settings"
if [[ -f "$EXAMPLE_TEMP_DIR/charlie/claude-config/settings.json" ]]; then
    info "Contents of charlie/claude-config/settings.json:"
    head -20 "$EXAMPLE_TEMP_DIR/charlie/claude-config/settings.json"
else
    info "(no user settings for Charlie -- minimal setup)"
fi
pause

# ===================================================================
section "Summary"
# ===================================================================

success "Isolated workspace demo complete"
echo
info "Key takeaways:"
info ""
info "  CLAUDE_CONFIG_DIR isolates each member's personal config"
info "    Alice, Bob, and Charlie each have their own settings.json"
info ""
info "  CLAUDEUP_HOME isolates each member's profiles and local items"
info "    Profiles and rules are stored per-member, not globally"
info ""
info "  --project scope writes to the shared project/.claude/ directory"
info "    These files are regular files, not symlinks -- git-committable"
info ""
info "  User-scope profiles are personal and independent"
info "    Alice has superpowers, Bob has style and review tools,"
info "    Charlie has nothing -- all without conflict"
info ""
info "  Project-scoped items are visible to everyone"
info "    Rules and agents in .claude/ work for any team member"
echo

prompt_cleanup
