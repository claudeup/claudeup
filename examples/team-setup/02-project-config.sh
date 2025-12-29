#!/usr/bin/env bash
# ABOUTME: Example showing how to share profiles via project-local storage
# ABOUTME: Demonstrates profile save --scope project and .claudeup/profiles/

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Team Setup: Project-Local Profile Sharing              ║
╚════════════════════════════════════════════════════════════════╝

Share Claude Code profiles with your team by storing them directly
in your project repository.

EOF
pause

section "1. Two Ways to Share Configuration"

info "claudeup offers two complementary approaches:"
echo
info "  A. Project-local profiles (.claudeup/profiles/)"
info "     • Store profile definitions in your repo"
info "     • Team members get profiles when they clone"
info "     • Profiles are versioned with your code"
echo
info "  B. Project config file (.claudeup.json)"
info "     • Define required plugins for the project"
info "     • Install plugins on sync"
echo
info "This example focuses on project-local profiles (A)"
pause

section "2. Save a Profile to Project Scope"

step "Use --scope project to save profile in the repo"
echo
info "When you run:"
echo -e "${YELLOW}\$ claudeup profile save team-config --scope project${NC}"
echo
info "claudeup creates:"
info "  .claudeup/profiles/team-config.json"
echo
info "This file contains:"
info "  • Snapshot of current Claude plugins"
info "  • Settings configuration"
info "  • Description and metadata"
pause

section "3. Context-Aware Default"

step "Smart scoping when .claudeup.json exists"
echo
info "claudeup detects project context:"
echo
info "  If .claudeup.json exists in current directory:"
info "    → profile save defaults to project scope"
echo
info "  Otherwise:"
info "    → profile save defaults to user scope (~/.claudeup/profiles/)"
echo
info "You can always override with --scope user or --scope project"
pause

section "4. Directory Structure"

step "Where project profiles are stored"
cat <<'STRUCTURE'
your-project/
├── .claudeup/
│   └── profiles/
│       ├── team-config.json     # Shared team profile
│       └── frontend-dev.json    # Role-specific profile
├── .claudeup.json               # Project plugin requirements
├── .gitignore
└── src/
STRUCTURE
echo
info "The .claudeup/profiles/ directory should be committed to git"
pause

section "5. Git Integration"

info "Recommended .gitignore entries:"
cat <<'GITIGNORE'
# Claude Code local settings (personal overrides)
.claude/settings.local.json

# Keep these tracked for team sharing:
# .claude/settings.json
# .claudeup.json
# .claudeup/profiles/
GITIGNORE
echo
info "Note: .claudeup/profiles/ should NOT be ignored"
pause

section "6. Full Workflow Example"

step "Team lead creates shared profile"
echo -e "${YELLOW}\$ cd your-project${NC}"
echo -e "${YELLOW}\$ claudeup profile save team-config --scope project${NC}"
echo -e "${YELLOW}\$ git add .claudeup/profiles/${NC}"
echo -e "${YELLOW}\$ git commit -m \"Add team Claude profile\"${NC}"
echo -e "${YELLOW}\$ git push${NC}"
echo
step "Team member applies after clone"
echo -e "${YELLOW}\$ git clone <repo-url>${NC}"
echo -e "${YELLOW}\$ cd your-project${NC}"
echo -e "${YELLOW}\$ claudeup profile sync${NC}"
echo
info "The sync command finds profiles in .claudeup/profiles/"
pause

section "Summary"

success "You can share profiles via your project repository"
echo
info "Key commands:"
info "  claudeup profile save <name> --scope project"
info "  claudeup profile sync"
echo
info "Key files:"
info "  .claudeup/profiles/     Project-local profile storage"
info "  .claudeup.json          Project plugin requirements"
echo

prompt_cleanup
