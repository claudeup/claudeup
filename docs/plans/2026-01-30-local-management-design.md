# Design: claudeup Local Management

## Overview

claudeup gains a `local` subcommand for managing local Claude Code extensions - agents, commands, skills, hooks, rules, and output-styles that live in `~/.claude/.library/`.

This replaces the need for the separate `claude-config` Python script, consolidating all Claude Code configuration management into claudeup.

## CLI Commands

```bash
# List all local items and their status
claudeup local list [category]
claudeup local list --enabled
claudeup local list --disabled

# Enable/disable items
claudeup local enable <category> <items...>
claudeup local disable <category> <items...>

# View item contents
claudeup local view <category> <item>

# Import items from active directories to .library
claudeup local import <category> <items...>
claudeup local import-all [patterns...]

# Sync symlinks from enabled.json (repair command)
claudeup local sync
```

**Categories:** `agents`, `commands`, `skills`, `hooks`, `rules`, `output-styles`

**Wildcards:**

- `gsd-*` - matches items starting with `gsd-`
- `gsd/*` - matches all items in a subdirectory
- `*` - matches everything in a category

**Examples:**

```bash
claudeup local enable agents gsd-*
claudeup local enable commands gsd/*
claudeup local disable hooks gsd-check-update
claudeup local list agents --enabled
```

## Profile Integration

Profiles gain two new sections: `local` for enabling local items, and `settingsHooks` for registering hooks in settings.json.

### Profile Schema Addition

```json
{
  "name": "gsd",
  "description": "Get Shit Done workflow system",
  "marketplaces": [...],
  "plugins": [...],

  "local": {
    "agents": ["gsd-*"],
    "commands": ["gsd/*"],
    "hooks": ["gsd-check-update.js", "gsd-statusline.js"]
  },

  "settingsHooks": {
    "SessionStart": [
      {
        "type": "command",
        "command": "node \"$HOME/.claude/hooks/gsd-check-update.js\""
      }
    ]
  }
}
```

### Apply Behavior

When `claudeup profile apply gsd` runs:

1. Install marketplaces and plugins (existing behavior)
2. Enable local items via symlinks (new)
3. Merge `settingsHooks` into settings.json, deduplicating by command string (new)

### Save Behavior

When `claudeup profile save my-setup` runs:

1. Capture marketplaces, plugins, MCP servers (existing)
2. Capture enabled local items from `enabled.json` (new)
3. Capture relevant hooks from settings.json (new - only captures hooks that reference files in `~/.claude/hooks/`)

### Reset Behavior

`claudeup profile reset gsd` removes plugins and marketplaces but does NOT disable local items. Local items are sticky - user must explicitly run `claudeup local disable agents gsd-*` if desired.

### Settings Hooks Merge Behavior

- Hooks are merged, not replaced
- If profile defines `SessionStart` hooks, they're added alongside existing `SessionStart` hooks
- Deduplication by command string prevents running the same hook twice
- statusLine is NOT managed by profiles (user preference)

## Internal Architecture

### Package Structure

```
internal/
  local/
    local.go        # Core types and interface
    enabled.go      # enabled.json read/write
    symlinks.go     # Symlink creation/removal logic
    categories.go   # Category definitions and paths
    resolve.go      # Item name resolution (handles missing extensions)
    wildcard.go     # Pattern matching (gsd-*, gsd/*)
```

### Key Types

```go
// Category represents a type of local item
type Category string

const (
    Agents       Category = "agents"
    Commands     Category = "commands"
    Skills       Category = "skills"
    Hooks        Category = "hooks"
    Rules        Category = "rules"
    OutputStyles Category = "output-styles"
)

// Manager handles local item operations
type Manager struct {
    claudeDir  string  // ~/.claude
    libraryDir string  // ~/.claude/.library
    configFile string  // ~/.claude/enabled.json
}

func (m *Manager) List(category Category) ([]Item, error)
func (m *Manager) Enable(category Category, patterns []string) error
func (m *Manager) Disable(category Category, patterns []string) error
func (m *Manager) Sync() error
func (m *Manager) View(category Category, name string) (string, error)
```

### Settings Hooks

Settings hook management extends `internal/claude/settings.go`:

```go
func (s *Settings) MergeHooks(hooks map[string][]HookEntry) error
func (s *Settings) RemoveHooks(commands []string) error
```

## Agent Group Handling

Agents can be organized in groups (subdirectories).

### Directory Structure

```
.library/agents/
  gsd-codebase-mapper.md    # flat (no group)
  gsd-debugger.md
  business-product/         # group directory
    analyst.md
    strategist.md
```

### Resolution Rules

1. **Flat agents** (`gsd-planner`) - resolve to `gsd-planner.md`
2. **Grouped agents** (`business-product/analyst`) - resolve to `business-product/analyst.md`
3. **Shorthand** (`analyst`) - searches all groups, returns first match
4. **Group wildcard** (`business-product/*`) - matches all agents in group

### Symlink Structure

```
~/.claude/agents/
  gsd-planner.md -> ../.library/agents/gsd-planner.md
  business-product/
    analyst.md -> ../../.library/agents/business-product/analyst.md
```

### enabled.json Format

```json
{
  "agents": {
    "gsd-planner.md": true,
    "gsd-executor.md": true,
    "business-product/analyst.md": true
  }
}
```

## Storage

- **enabled.json** - `~/.claude/enabled.json` (unchanged from claude-config)
- **.library/** - `~/.claude/.library/` (unchanged)
- **Symlinks** - `~/.claude/{category}/` (unchanged)

claudeup reads/writes the same files as claude-config for backwards compatibility.

## Migration Path

1. **Phase 1:** Ship `claudeup local` commands. Both tools work with same data.
2. **Phase 2:** Update documentation to recommend claudeup.
3. **Phase 3:** Optionally deprecate claude-config.

No breaking changes - enabled.json format, .library/ structure, and symlinks all stay the same.

## Design Decisions

| Decision            | Choice                              | Rationale                                     |
| ------------------- | ----------------------------------- | --------------------------------------------- |
| CLI name            | `local`                             | Distinguishes from marketplace plugins        |
| Go package location | `internal/local/`                   | No external dependencies                      |
| Settings hooks      | First-class `settingsHooks` section | Simpler than generic JSON patches             |
| Hook merge behavior | Merge + deduplicate                 | Least surprising, matches plugin accumulation |
| statusLine          | Not managed by profiles             | User preference                               |
| Storage             | Keep `enabled.json`                 | Backwards compatible                          |
| Reset behavior      | Local items are sticky              | User owns local files                         |
