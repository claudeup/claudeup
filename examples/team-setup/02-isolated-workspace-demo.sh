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

# ---------------------------------------------------------------------------
# Helper: switch active team member and cd to their project directory
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
backend project. Each has their own machine (simulated with
isolated directories) with a separate CLAUDE_CONFIG_DIR. Team
configuration travels through the project repository (git), while
personal plugins remain independent.

EOF
pause

# ===================================================================
section "1. Setting Up the Team"
# ===================================================================

step "Create isolated directories for each team member"

for member in alice bob charlie; do
    mkdir -p "$EXAMPLE_TEMP_DIR/$member/project"
    mkdir -p "$EXAMPLE_TEMP_DIR/$member/claude-config/plugins"
    mkdir -p "$EXAMPLE_TEMP_DIR/$member/claudeup-home/profiles"
done
success "Created alice/, bob/, charlie/ under $EXAMPLE_TEMP_DIR"
echo

info "Directory layout (simulating separate machines):"
info "  $EXAMPLE_TEMP_DIR/"
info "  ├── alice/"
info "  │   ├── project/          # Alice's working copy"
info "  │   ├── claude-config/    # CLAUDE_CONFIG_DIR"
info "  │   └── claudeup-home/    # CLAUDEUP_HOME"
info "  ├── bob/"
info "  │   ├── project/          # Bob's clone"
info "  │   ├── claude-config/"
info "  │   └── claudeup-home/"
info "  └── charlie/"
info "      ├── project/          # Charlie's clone"
info "      ├── claude-config/"
info "      └── claudeup-home/"
echo

step "Create fixture profiles"

# -- Team profile (shared by everyone via onboarding docs / wiki) --
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
      "extensions": {
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
      "extensions": {
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

step "Create fixture extensions for the team profile"

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
cd "$EXAMPLE_TEMP_DIR/alice/project" || exit 1
echo

step "Apply the team profile at project scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply go-backend-team --project --yes
echo

step "Check project-scoped files created by the team profile"
info "Rules directory:"
ls -la "$EXAMPLE_TEMP_DIR/alice/project/.claude/rules/" 2>/dev/null || info "  (no rules directory yet)"
echo
info "Agents directory:"
ls -la "$EXAMPLE_TEMP_DIR/alice/project/.claude/agents/" 2>/dev/null || info "  (no agents directory yet)"
echo

step "Apply Alice's personal profile at user scope"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply alice-tools --yes
echo

step "View Alice's profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
echo

info "Alice's project is ready. In a real workflow, she would commit"
info "the .claude/ directory to git so teammates get the team config."
pause

# ===================================================================
section "3. Bob Clones the Project"
# ===================================================================

info "Bob clones the project repository. The project-scoped settings"
info "and extensions (.claude/) travel with git -- no re-application needed."
echo

step "Simulate git clone (copy Alice's project to Bob's machine)"
cp -R "$EXAMPLE_TEMP_DIR/alice/project/." "$EXAMPLE_TEMP_DIR/bob/project/"
success "Copied project to Bob's directory (simulating git clone)"
echo

step "Verify Bob's clone has the team configuration"
info "Project settings.json:"
if [[ -f "$EXAMPLE_TEMP_DIR/bob/project/.claude/settings.json" ]]; then
    cat "$EXAMPLE_TEMP_DIR/bob/project/.claude/settings.json"
else
    info "  (no project settings.json)"
fi
echo
info "Project rules:"
ls "$EXAMPLE_TEMP_DIR/bob/project/.claude/rules/" 2>/dev/null || info "  (no rules)"
info "Project agents:"
ls "$EXAMPLE_TEMP_DIR/bob/project/.claude/agents/" 2>/dev/null || info "  (no agents)"
echo

switch_member bob
cd "$EXAMPLE_TEMP_DIR/bob/project" || exit 1
echo

# The profile and extensions below let Bob see the team profile in
# 'claudeup profile list' and apply it to other projects. They're not
# needed for this project -- .claude/ from the clone already has everything.
step "Copy team profile to Bob's profiles (from onboarding wiki)"
cp "$EXAMPLE_TEMP_DIR/alice/claudeup-home/profiles/go-backend-team.json" \
   "$EXAMPLE_TEMP_DIR/bob/claudeup-home/profiles/"
success "Copied go-backend-team.json to Bob's profiles"
echo

step "Copy team extensions to Bob (from shared team tooling)"
mkdir -p "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/rules"
mkdir -p "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/agents"
cp "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/rules/golang.md" \
   "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/rules/"
cp "$EXAMPLE_TEMP_DIR/alice/claudeup-home/local/agents/reviewer.md" \
   "$EXAMPLE_TEMP_DIR/bob/claudeup-home/local/agents/"
success "Copied extensions to Bob"
echo

info "Bob does NOT need to re-apply the team profile at project scope."
info "The .claude/ directory from Alice's commit already has everything."
echo

step "Apply Bob's personal profile (user scope only)"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply bob-tools --yes
echo

step "View Bob's profile list"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

# ===================================================================
section "4. Charlie Clones with Minimal Setup"
# ===================================================================

info "Charlie clones the project but has no personal profile."
info "He can start working immediately with the team configuration."
echo

step "Simulate git clone (copy Alice's project to Charlie's machine)"
cp -R "$EXAMPLE_TEMP_DIR/alice/project/." "$EXAMPLE_TEMP_DIR/charlie/project/"
success "Copied project to Charlie's directory (simulating git clone)"
echo

switch_member charlie
cd "$EXAMPLE_TEMP_DIR/charlie/project" || exit 1
echo

step "Charlie has no personal profile -- just list what's available"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list || info "No profiles configured yet"
echo

step "Verify Charlie sees the project configuration from git"
if [[ -f "$EXAMPLE_TEMP_DIR/charlie/project/.claude/settings.json" ]]; then
    info "Project settings.json exists:"
    cat "$EXAMPLE_TEMP_DIR/charlie/project/.claude/settings.json"
else
    info "No project settings.json (team plugins would appear here)"
fi
echo

info "Project rules from git:"
cat "$EXAMPLE_TEMP_DIR/charlie/project/.claude/rules/golang.md" 2>/dev/null || info "  (not found)"
pause

# ===================================================================
section "5. Compare Team Members"
# ===================================================================

info "All three members have their own project clone with identical"
info "team configuration, but different personal (user-scope) settings."
echo

step "Project-scoped settings (identical across all clones)"
for member in alice bob charlie; do
    project_settings="$EXAMPLE_TEMP_DIR/$member/project/.claude/settings.json"
    if [[ -f "$project_settings" ]]; then
        info "$member's project settings.json:"
        cat "$project_settings"
    else
        info "$member: (no project settings.json)"
    fi
    echo
done

step "Verify project settings match across all clones"
if diff -q "$EXAMPLE_TEMP_DIR/alice/project/.claude/settings.json" \
        "$EXAMPLE_TEMP_DIR/bob/project/.claude/settings.json" &>/dev/null && \
   diff -q "$EXAMPLE_TEMP_DIR/alice/project/.claude/settings.json" \
        "$EXAMPLE_TEMP_DIR/charlie/project/.claude/settings.json" &>/dev/null; then
    success "All project settings identical (as expected from git)"
else
    warn "Project settings differ (unexpected -- git should sync these)"
fi
echo

step "User-scoped settings (personal to each member)"
for member in alice bob charlie; do
    user_settings="$EXAMPLE_TEMP_DIR/$member/claude-config/settings.json"
    if [[ -f "$user_settings" ]]; then
        info "$member's user settings.json:"
        cat "$user_settings"
    else
        info "$member: (no user settings -- minimal setup)"
    fi
    echo
done
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
info "  CLAUDEUP_HOME isolates each member's profiles and extensions"
info "    Profiles and rules are stored per-member, not globally"
info ""
info "  Project settings travel through git, not re-application"
info "    Alice applies once, commits .claude/ -- teammates clone it"
info ""
info "  --project scope writes to the project's .claude/ directory"
info "    These files are regular files, not symlinks -- git-committable"
info ""
info "  User-scope profiles are personal and independent"
info "    Alice has superpowers, Bob has style and review tools,"
info "    Charlie has nothing -- all without conflict"
echo

prompt_cleanup
