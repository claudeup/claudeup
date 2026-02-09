#!/usr/bin/env bash
# ABOUTME: Example showing composable profile stacks with includes
# ABOUTME: Demonstrates building blocks, stacks, and resolution

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
source "$SCRIPT_DIR/../lib/common.sh"
parse_common_args "$@"
setup_environment

cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║       Profile Management: Composable Stacks                    ║
╚════════════════════════════════════════════════════════════════╝

Organize profiles as small, focused building blocks and compose
them into stacks. A single "apply" installs everything.

EOF
pause

section "1. Create Building Block Profiles"

info "Building blocks are regular profiles focused on a single concern."
info "Organize them in subdirectories by category:"
info ""
info "  profiles/"
info "    languages/go.json         -- gopls plugin, detect go.mod"
info "    platforms/backend.json    -- backend API plugins"
info "    workflow/testing.json     -- TDD and unit testing plugins"
info "    tools/memory.json         -- episodic memory, claude-mem"
echo

step "Create a language profile"

mkdir -p "$CLAUDEUP_HOME/profiles/languages"
cat > "$CLAUDEUP_HOME/profiles/languages/go.json" <<'PROFILE'
{
  "name": "go",
  "description": "Go language development with gopls LSP",
  "marketplaces": [
    { "source": "github", "repo": "anthropics/claude-plugins-official" }
  ],
  "perScope": {
    "project": {
      "plugins": ["gopls-lsp@claude-plugins-official"]
    }
  },
  "detect": {
    "files": ["go.mod", "go.sum"]
  }
}
PROFILE
success "Created languages/go.json"

step "Create a workflow profile"

mkdir -p "$CLAUDEUP_HOME/profiles/workflow"
cat > "$CLAUDEUP_HOME/profiles/workflow/testing.json" <<'PROFILE'
{
  "name": "testing",
  "description": "Testing, TDD, and performance testing",
  "marketplaces": [
    { "source": "github", "repo": "anthropics/claude-plugins-official" }
  ],
  "perScope": {
    "project": {
      "plugins": ["tdd-workflows@claude-plugins-official"]
    }
  }
}
PROFILE
success "Created workflow/testing.json"

step "Create a tools profile"

mkdir -p "$CLAUDEUP_HOME/profiles/tools"
cat > "$CLAUDEUP_HOME/profiles/tools/memory.json" <<'PROFILE'
{
  "name": "memory",
  "description": "Memory and context persistence",
  "marketplaces": [
    { "source": "github", "repo": "anthropics/claude-plugins-official" }
  ],
  "perScope": {
    "user": {
      "plugins": ["episodic-memory@claude-plugins-official"]
    }
  }
}
PROFILE
success "Created tools/memory.json"
pause

section "2. Create a Stack Profile"

info "A stack profile composes building blocks via 'includes'."
info "Stacks are pure -- they contain only name, description, and includes."
echo

step "Create a stack that composes the building blocks"

mkdir -p "$CLAUDEUP_HOME/profiles/stacks"
cat > "$CLAUDEUP_HOME/profiles/stacks/go-dev.json" <<'PROFILE'
{
  "name": "go-dev",
  "description": "Go development: memory + Go language + testing",
  "includes": ["memory", "go", "testing"]
}
PROFILE
success "Created stacks/go-dev.json"
echo

info "The stack is just 4 lines of JSON, but it resolves to all"
info "marketplaces, plugins, and settings from the included profiles."
pause

section "3. View the Stack"

step "List profiles -- stacks are marked with [stack]"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile list
pause

step "Show the stack -- see the expanded include tree and resolved summary"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile show go-dev
pause

section "4. Nested Stacks"

info "Stacks can include other stacks. For example, an 'essentials' stack"
info "that bundles core tools, then a 'fullstack-go' stack that includes it."
echo

cat > "$CLAUDEUP_HOME/profiles/stacks/essentials.json" <<'PROFILE'
{
  "name": "essentials",
  "description": "Core tools: memory and testing",
  "includes": ["memory", "testing"]
}
PROFILE
success "Created stacks/essentials.json"

cat > "$CLAUDEUP_HOME/profiles/stacks/fullstack-go.json" <<'PROFILE'
{
  "name": "fullstack-go",
  "description": "Go fullstack: essentials + Go language",
  "includes": ["essentials", "go"]
}
PROFILE
success "Created stacks/fullstack-go.json"
echo

step "Show the nested stack -- essentials is expanded one level"
run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile show fullstack-go
pause

section "5. Apply a Stack"

step "Apply the stack -- resolves all includes and installs at correct scopes"
info "Stack profiles always use ApplyAllScopes (no --scope flag needed)."
echo

if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
    run_cmd "$EXAMPLE_CLAUDEUP_BIN" profile apply go-dev
else
    info "Command: claudeup profile apply go-dev"
    info ""
    info "This would resolve the include tree and install:"
    info "  User scope:    episodic-memory@claude-plugins-official"
    info "  Project scope: gopls-lsp@claude-plugins-official"
    info "                 tdd-workflows@claude-plugins-official"
fi
pause

section "Summary"

success "You know how to build composable profile stacks"
echo
info "Key concepts:"
info "  Building blocks  Small, focused profiles in category subdirectories"
info "  Stacks           Pure composition via 'includes' -- no own config"
info "  Nesting          Stacks can include other stacks"
info "  Scope merging    Plugins from different scopes merge correctly"
info "  Deduplication    Shared marketplaces and plugins appear only once"
echo
info "Key commands:"
info "  claudeup profile show <stack>    See expanded includes and resolved summary"
info "  claudeup profile apply <stack>   Resolve and apply all included profiles"
echo

prompt_cleanup
