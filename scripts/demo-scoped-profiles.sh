#!/bin/bash
# ABOUTME: Demo script for scoped profiles functionality
# ABOUTME: Creates and applies profiles at different scopes to test the feature

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
print_section() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_step() {
    echo -e "${GREEN}▸ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

pause() {
    echo ""
    read -p "Press ENTER to continue..."
}

# Build the binary
print_section "Building claudeup"
print_step "Building fresh binary..."
go build -o bin/claudeup ./cmd/claudeup
print_info "Binary built successfully"

# Create test directory
TEST_DIR=$(mktemp -d -t claudeup-demo-XXXXXX)
print_info "Test directory: $TEST_DIR"
cd "$TEST_DIR"

print_section "Setup: Creating Sample Profiles"

# Create profile directories
mkdir -p .claudeup/profiles

print_step "Creating 'base-tools' profile (user scope profile)"
cat > .claudeup/profiles/base-tools.json <<'EOF'
{
  "name": "base-tools",
  "plugins": [
    "claude-mem@thedotmack",
    "superpowers@superpowers-marketplace",
    "code-review-ai@claude-code-workflows"
  ],
  "marketplaces": [
    {
      "source": "github",
      "repo": "thedotmack/claude-mem"
    },
    {
      "source": "github",
      "repo": "superpowers-marketplace/superpowers"
    },
    {
      "source": "github",
      "repo": "claude-code-workflows/workflows"
    }
  ]
}
EOF
print_info "Created: base-tools.json (3 plugins)"

print_step "Creating 'backend-stack' profile (project scope profile)"
cat > .claudeup/profiles/backend-stack.json <<'EOF'
{
  "name": "backend-stack",
  "plugins": [
    "gopls-lsp@claude-plugins-official",
    "backend-development@claude-code-workflows",
    "tdd-workflows@claude-code-workflows",
    "debugging-toolkit@claude-code-workflows"
  ],
  "marketplaces": [
    {
      "source": "github",
      "repo": "claude-plugins-official/plugins"
    },
    {
      "source": "github",
      "repo": "claude-code-workflows/workflows"
    }
  ]
}
EOF
print_info "Created: backend-stack.json (4 plugins)"

print_step "Creating 'docker-tools' profile (local scope profile)"
cat > .claudeup/profiles/docker-tools.json <<'EOF'
{
  "name": "docker-tools",
  "plugins": [
    "systems-programming@claude-code-workflows",
    "shell-scripting@claude-code-workflows"
  ],
  "marketplaces": [
    {
      "source": "github",
      "repo": "claude-code-workflows/workflows"
    }
  ]
}
EOF
print_info "Created: docker-tools.json (2 plugins)"

print_info "All profiles created in .claudeup/profiles/"
ls -la .claudeup/profiles/

pause

# Show initial status
print_section "Initial Status (No Profiles Applied)"
print_step "Running: claudeup status"
$OLDPWD/bin/claudeup status || true

pause

# Apply base-tools at user scope
print_section "Phase 1: Apply 'base-tools' at User Scope"
print_step "Running: claudeup profile use base-tools --scope user"
print_info "This sets personal defaults that apply everywhere"

# Note: This will fail in a test environment without Claude installed
# Show what would happen
cat <<'EOF'

Expected behavior:
- Profile applied to ~/.claude/settings.json
- Plugins enabled at user scope
- Available in all projects

Command would be:
  claudeup profile use base-tools --scope user

EOF

pause

# Apply backend-stack at project scope
print_section "Phase 2: Apply 'backend-stack' at Project Scope"
print_step "Running: claudeup profile use backend-stack --scope project"
print_info "This sets project-specific plugins (team-shared, committed to git)"

cat <<'EOF'

Expected behavior:
- Profile applied to ./.claude/settings.json
- Plugins enabled at project scope
- Settings accumulate: user + project plugins all active
- Project settings committed to git for team

Command would be:
  claudeup profile use backend-stack --scope project

EOF

pause

# Apply docker-tools at local scope
print_section "Phase 3: Apply 'docker-tools' at Local Scope"
print_step "Running: claudeup profile use docker-tools --scope local"
print_info "This sets machine-specific plugins (gitignored)"

cat <<'EOF'

Expected behavior:
- Profile applied to ./.claude-local/settings.json
- Plugins enabled at local scope
- Settings accumulate: user + project + local all active
- Local settings NOT committed (in .gitignore)

Command would be:
  claudeup profile use docker-tools --scope local

EOF

pause

# Show accumulated state
print_section "Accumulated State"
print_info "After applying all three profiles, all plugins are active:"

cat <<'EOF'

Settings Precedence (local > project > user):
┌─────────────────────────────────────┐
│ Local Scope (docker-tools)          │  ← Highest precedence
│  - systems-programming              │
│  - shell-scripting                  │
├─────────────────────────────────────┤
│ Project Scope (backend-stack)       │
│  - gopls-lsp                        │
│  - backend-development              │
│  - tdd-workflows                    │
│  - debugging-toolkit                │
├─────────────────────────────────────┤
│ User Scope (base-tools)             │  ← Base layer
│  - claude-mem                       │
│  - superpowers                      │
│  - code-review-ai                   │
└─────────────────────────────────────┘

Total Active: 9 plugins (3 + 4 + 2)

EOF

pause

# Demonstrate drift detection
print_section "Drift Detection Demo"
print_info "Simulating manual plugin changes to demonstrate drift detection..."

cat <<'EOF'

Scenario: User manually enables an extra plugin at project scope

Steps:
1. Manually edit ./.claude/settings.json
2. Add "extra-plugin@marketplace" to enabledPlugins
3. Run: claudeup status
4. Run: claudeup status --scope project

Expected output:
  Active profile 'backend-stack' has unsaved changes:
    • project scope: 1 plugin added

  Run 'claudeup profile save --scope project' to persist changes.

EOF

pause

# Show scope-specific status checks
print_section "Scope-Specific Status Checks"

print_step "Check user scope drift:"
print_info "claudeup status --scope user"
echo "  Shows only drift at user scope"

echo ""
print_step "Check project scope drift:"
print_info "claudeup status --scope project"
echo "  Shows only drift at project scope"

echo ""
print_step "Check local scope drift:"
print_info "claudeup status --scope local"
echo "  Shows only drift at local scope"

echo ""
print_step "Check all scopes:"
print_info "claudeup status"
echo "  Shows drift at all active scopes"

pause

# Show enhanced plugin list
print_section "Enhanced Plugin List (Scope Information)"
print_info "Running: claudeup plugin list"

cat <<'EOF'

Expected output format:

✓ gopls-lsp@claude-plugins-official (v1.0.0)
  Version: 1.0.0
  Status: enabled
  Enabled at: project              ← Shows which scopes
  Active source: project           ← Which installation is used
  Path: ./.claude/plugins/gopls-lsp
  Type: local

✓ claude-mem@thedotmack (v7.4.5)
  Version: 7.4.5
  Status: enabled
  Enabled at: user                 ← Only at user scope
  Active source: user
  Also installed at: project       ← Other installations
  Path: ~/.claude/plugins/cache/claude-mem
  Type: cached

✓ systems-programming@claude-code-workflows
  Version: 1.2.0
  Status: enabled
  Enabled at: local, project       ← Multiple scopes
  Active source: local             ← Local has highest precedence
  Also installed at: project, user
  Path: ./.claude-local/plugins/systems-programming
  Type: local

EOF

pause

# Real-world workflow example
print_section "Real-World Workflow Example"

cat <<'EOF'

Team Workflow with Scoped Profiles:

1. Initial Setup (Team Lead):
   ┌─────────────────────────────────────────────┐
   │ cd ~/my-project                             │
   │ claudeup profile use backend-stack          │
   │   --scope project                           │
   │ git add .claudeup/ .claude/                 │
   │ git commit -m "Add backend profile"         │
   │ git push                                    │
   └─────────────────────────────────────────────┘

2. Team Member Clones Repo:
   ┌─────────────────────────────────────────────┐
   │ git clone <repo>                            │
   │ cd <repo>                                   │
   │ claudeup status                             │
   │   → Shows project profile is active         │
   │ # Project plugins automatically available   │
   └─────────────────────────────────────────────┘

3. Personal Customization:
   ┌─────────────────────────────────────────────┐
   │ # Add personal tools (not shared)           │
   │ claudeup profile use my-tools --scope user  │
   │                                             │
   │ # Add machine-specific tools                │
   │ claudeup profile use docker --scope local   │
   │                                             │
   │ # Result: project + user + local all active │
   └─────────────────────────────────────────────┘

4. Making Project Changes:
   ┌─────────────────────────────────────────────┐
   │ # Manually add plugin at project scope      │
   │ claude plugin install new-linter            │
   │                                             │
   │ # Check what changed                        │
   │ claudeup status --scope project             │
   │   → Shows 1 plugin added at project scope   │
   │                                             │
   │ # Save changes to profile                   │
   │ claudeup profile save backend-stack         │
   │   --scope project                           │
   │                                             │
   │ # Commit and share with team                │
   │ git add .claudeup/ .claude/                 │
   │ git commit -m "Add new-linter to profile"   │
   │ git push                                    │
   └─────────────────────────────────────────────┘

Git Configuration:
  .gitignore:
    .claude-local/        # Machine-specific (don't commit)

  Committed files:
    .claudeup/profiles/   # Team-shared profiles
    .claude/              # Project scope settings

EOF

pause

# Summary
print_section "Summary: Key Benefits"

cat <<'EOF'

✓ Team Collaboration
  - Project profiles shared via git
  - Team members get same plugin setup
  - Consistent development environment

✓ Personal Flexibility
  - User scope for personal defaults
  - Local scope for machine-specific tools
  - No conflicts with team settings

✓ Clear Separation
  - User: Personal preferences (not shared)
  - Project: Team requirements (shared via git)
  - Local: Machine-specific (gitignored)

✓ Drift Detection
  - Per-scope drift reporting
  - Know exactly what changed where
  - Easy to sync changes back to profiles

✓ Precedence System
  - Local overrides project overrides user
  - Settings accumulate (all active together)
  - Predictable, clear behavior

EOF

# Cleanup
print_section "Cleanup"
print_step "Test directory: $TEST_DIR"
print_info "Leaving test directory for inspection"
print_info "Delete with: rm -rf $TEST_DIR"

echo -e "\n${GREEN}Demo complete!${NC}\n"
