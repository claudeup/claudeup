---
title: File Operations
---

# File Operations Reference

This document catalogs all files that `claudeup` reads, writes, or modifies, organized by ownership and the operations that trigger changes.

## Files Owned by Claude CLI

These files are part of Claude Code's native configuration. `claudeup` reads and modifies them to manage Claude state.

### `~/.claude/plugins/installed_plugins.json`

**Owner:** Claude CLI
**Format:** JSON (V1 or V2)
**Purpose:** Plugin registry with metadata (version, install path, scope)

**Read by:**

- `internal/claude/plugins.go:LoadPlugins()`
- Used by: profile snapshot, status, plugin commands

**Written by:**

- `internal/claude/plugins.go:SavePlugins()`
- Triggered by: profile apply (when removing plugins from user scope)

---

### `~/.claude/plugins/known_marketplaces.json`

**Owner:** Claude CLI
**Format:** JSON
**Purpose:** Marketplace registry with source metadata

**Read by:**

- `internal/claude/marketplaces.go:LoadMarketplaces()`
- Used by: profile snapshot, status, apply operations

**Written by:**

- `internal/claude/marketplaces.go:SaveMarketplaces()`
- Triggered by: marketplace add/remove operations (via claude CLI)

---

### `~/.claude/settings.json` (User Scope)

**Owner:** Claude CLI
**Format:** JSON
**Purpose:** User-level settings including enabled plugins

**Read by:**

- `internal/claude/settings.go:LoadSettings()`
- `internal/claude/settings.go:LoadSettingsForScope("user")`
- Used by: profile snapshot, status, apply operations

**Written by:**

- `internal/claude/settings.go:SaveSettings()`
- `internal/claude/settings.go:SaveSettingsForScope("user")`
- Triggered by:
  - `profile apply` (user scope) - declarative replace of enabledPlugins

---

### `./.claude/settings.json` (Project Scope)

**Owner:** Claude CLI
**Format:** JSON
**Purpose:** Project-level settings including enabled plugins

**Read by:**

- `internal/claude/settings.go:LoadSettingsForScope("project")`
- Used by: profile snapshot (project scope), status

**Written by:**

- `internal/claude/settings.go:SaveSettingsForScope("project")`
- Triggered by:
  - `profile apply --project` - declarative replace of enabledPlugins
  - Project scope plugin operations

---

### `./.claude/settings.local.json` (Local Scope)

**Owner:** Claude CLI
**Format:** JSON
**Purpose:** Machine-specific project settings (gitignored)

**Read by:**

- `internal/claude/settings.go:LoadSettingsForScope("local")`
- Used by: merged settings load, status

**Written by:**

- `internal/claude/settings.go:SaveSettingsForScope("local")`
- Triggered by:
  - `profile apply --local` - declarative replace of enabledPlugins
  - Local scope plugin operations

---

### `~/.claude.json` (User MCP Servers)

**Owner:** Claude CLI
**Format:** JSON
**Purpose:** MCP server configurations at user scope

**Read by:**

- `internal/profile/snapshot.go:readMCPServersForScope("user")`
- Used by: profile snapshot, MCP discovery

**Written by:**

- Modified indirectly via `claude mcp add/remove` commands
- Triggered by: profile apply (user scope MCP operations)

---

### `./.mcp.json` (Project MCP Servers)

**Owner:** Claude CLI
**Format:** JSON (Claude native format)
**Purpose:** Project-scoped MCP servers that Claude auto-loads

**Read by:**

- `internal/profile/mcp_json.go:LoadMCPJSON()`
- `internal/profile/snapshot.go:readMCPServersForScope("project")`
- Used by: profile snapshot (project scope), status

**Written by:**

- `internal/profile/mcp_json.go:WriteMCPJSON()`
- Triggered by:
  - `profile apply --project` - writes all MCP servers from profile

---

## Files Owned by claudeup

These files are created and managed exclusively by `claudeup`.

### `~/.claudeup/config.json`

**Owner:** claudeup
**Format:** JSON
**Purpose:** Global configuration (preferences)

**Read by:**

- `internal/config/global.go:Load()`
- Used by: all commands that need global config

**Written by:**

- `internal/config/global.go:Save()`
- Triggered by:
  - First run (creates with defaults)
  - Any command that modifies global config

---

### `~/.claudeup/profiles/{name}.json`

**Owner:** claudeup
**Format:** JSON
**Purpose:** Profile definitions (plugins, MCP servers, marketplaces, etc.)

**Read by:**

- `internal/profile/profile.go:Load()`
- Used by: profile apply, list, status

**Written by:**

- `internal/profile/profile.go:Save()`
- Triggered by:
  - `profile save` - creates new profile from current state
  - `profile create` - creates new profile interactively
  - Manual editing by users

---

### `~/.claudeup/backups/{scope}-scope.json`

**Owner:** claudeup
**Format:** JSON (copy of settings.json)
**Purpose:** Backup of scope settings before applying profile

**Read by:**

- `internal/backup/backup.go:RestoreScopeBackup()`

**Written by:**

- `internal/backup/backup.go:SaveScopeBackup()`
- Triggered by:
  - `profile apply --replace` - backs up current settings before replacing

---

### `~/.claudeup/backups/local-scope-{hash}.json`

**Owner:** claudeup
**Format:** JSON (copy of settings.local.json)
**Purpose:** Backup of local scope settings (project-specific)

**Read by:**

- `internal/backup/backup.go:RestoreLocalScopeBackup()`

**Written by:**

- `internal/backup/backup.go:SaveLocalScopeBackup()`
- Triggered by:
  - `profile apply --local` - backs up before replacing

---

### `~/.claudeup/enabled.json`

**Owner:** claudeup
**Format:** JSON
**Purpose:** Tracks which extensions are enabled per category

**Read by:**

- `internal/ext/config.go:LoadConfig()`
- Used by: all `extensions` subcommands, symlink reconciliation

**Written by:**

- `internal/ext/config.go:SaveConfig()`
- Triggered by:
  - `extensions enable` - marks items as enabled
  - `extensions disable` - marks items as disabled
  - `extensions install` - enables newly installed items
  - `extensions import` - enables imported items

---

### `~/.claudeup/ext/<category>/`

**Owner:** claudeup
**Format:** Directory tree organized by category (agents, commands, hooks, output-styles, rules, skills)
**Purpose:** Stores extension files; symlinks in `~/.claude/<category>/` point here

**Read by:**

- `internal/ext/list.go:ListItems()`
- `internal/ext/view.go:ViewItem()`
- `internal/ext/symlinks.go:ReconcileSymlinks()`
- Used by: all `extensions` subcommands

**Written by:**

- `internal/ext/install.go:Install()`
- `internal/ext/symlinks.go:Import()`
- Triggered by:
  - `extensions install` - copies items from external paths
  - `extensions import` - moves items from active directory to storage

---

## Operation-to-File Matrix

| Operation                  | Files Modified                                                    | Event Type |
| -------------------------- | ----------------------------------------------------------------- | ---------- |
| `profile apply` (user)     | `~/.claude/settings.json` (enabledPlugins declaratively replaced) | WRITE      |
| `profile apply` (user)     | `~/.claudeup/backups/user-scope.json` (backup before apply)       | WRITE      |
| `profile apply` (project)  | `./.claude/settings.json` (enabledPlugins replaced)               | WRITE      |
| `profile apply` (project)  | `./.mcp.json` (MCP servers written)                               | WRITE      |
| `profile apply` (local)    | `./.claude/settings.local.json` (enabledPlugins replaced)         | WRITE      |
| `profile apply` (local)    | `~/.claudeup/backups/local-scope-{hash}.json` (backup)            | WRITE      |
| `profile save`             | `~/.claudeup/profiles/{name}.json`                                | WRITE      |
| `plugin install/uninstall` | Via claude CLI - may update registry                              | INDIRECT   |
| `marketplace add/remove`   | Via claude CLI - updates `known_marketplaces.json`                | INDIRECT   |
| `mcp add/remove`           | Via claude CLI - updates `~/.claude.json`                         | INDIRECT   |
| `setup`                    | `~/.claudeup/config.json` (initial config)                        | WRITE      |
| `extensions enable`        | `~/.claudeup/enabled.json`, `~/.claude/<category>/<item>` symlink | WRITE      |
| `extensions disable`       | `~/.claudeup/enabled.json`, removes `~/.claude/<category>/<item>` | WRITE      |
| `extensions install`       | `~/.claudeup/ext/<category>/`, `~/.claudeup/enabled.json`         | WRITE      |
| `extensions import`        | `~/.claudeup/ext/<category>/`, `~/.claudeup/enabled.json`         | WRITE      |

---

## File Change Events to Monitor

For each file modification, the following events should be trackable:

1. **Timestamp** - When the change occurred
2. **Operation** - Which claudeup command caused it
3. **Scope** - user/project/local (if applicable)
4. **Changes** - What was added/removed/modified
5. **Backup** - Whether a backup was created
6. **Errors** - Any failures during the operation

This enables:

- Troubleshooting why settings changed
- Understanding profile application effects
- Debugging scope conflicts
- Audit trail for team projects
