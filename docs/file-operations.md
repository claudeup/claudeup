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
  - `plugin enable/disable` commands

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
  - `profile apply --scope project` - declarative replace of enabledPlugins
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
  - `profile apply --scope local` - declarative replace of enabledPlugins
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
  - `profile apply --scope project` - writes all MCP servers from profile

---

### `~/.claudeup/sandboxes/{profile}/.claude.json`
**Owner:** Sandbox (copy of user auth)
**Format:** JSON
**Purpose:** Authentication credentials copied to sandbox for --copy-auth

**Read by:**
- Source: user's `~/.claude.json`
- Destination: read by Claude CLI inside sandbox

**Written by:**
- `internal/sandbox/sandbox.go:CopyAuthFile()`
- Triggered by: `sandbox --copy-auth` flag or `sandbox.copyAuth` config setting

---

## Files Owned by claudeup

These files are created and managed exclusively by `claudeup`.

### `~/.claudeup/config.json`
**Owner:** claudeup
**Format:** JSON
**Purpose:** Global configuration (disabled MCP servers, preferences, active profile)

**Read by:**
- `internal/config/global.go:Load()`
- Used by: all commands that need global config

**Written by:**
- `internal/config/global.go:Save()`
- Triggered by:
  - First run (creates with defaults)
  - Any command that modifies global config

---

### `~/.claudeup/projects.json`
**Owner:** claudeup
**Format:** JSON
**Purpose:** Maps project directories to profiles for local scope

**Read by:**
- `internal/config/projects.go:LoadProjectsRegistry()`
- Used by: status, scope operations

**Written by:**
- `internal/config/projects.go:SaveProjectsRegistry()`
- Triggered by:
  - `profile apply --scope local` - records which profile is active
  - `scope restore` - updates when restoring local scope

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
- Used by: scope restore operations

**Written by:**
- `internal/backup/backup.go:SaveScopeBackup()`
- Triggered by:
  - `profile apply` - backs up current settings before replacing
  - `scope save` - explicitly saves current scope

---

### `~/.claudeup/backups/local-scope-{hash}.json`
**Owner:** claudeup
**Format:** JSON (copy of settings.local.json)
**Purpose:** Backup of local scope settings (project-specific)

**Read by:**
- `internal/backup/backup.go:RestoreLocalScopeBackup()`
- Used by: scope restore operations

**Written by:**
- `internal/backup/backup.go:SaveLocalScopeBackup()`
- Triggered by:
  - `profile apply --scope local` - backs up before replacing
  - `scope save --scope local` - explicitly saves

---

### `./.claudeup.json`
**Owner:** claudeup
**Format:** JSON
**Purpose:** Project config tracking which profile is applied (team-shareable)

**Read by:**
- `internal/profile/project_config.go:LoadProjectConfig()`
- Used by: profile detect, status

**Written by:**
- `internal/profile/project_config.go:SaveProjectConfig()`
- Triggered by:
  - `profile apply --scope project` - records profile metadata

---

### `~/.claudeup/sandboxes/{profile}/`
**Owner:** claudeup
**Format:** Directory
**Purpose:** Persistent state for named sandbox profiles

**Read by:**
- Docker mounts this directory into container
- Used by: sandbox sessions with `--profile` flag

**Written by:**
- `internal/sandbox/sandbox.go:StateDir()` - creates directory
- Triggered by: first sandbox run with a named profile

**Removed by:**
- `internal/sandbox/sandbox.go:CleanState()`
- Triggered by: explicit cleanup operations

---

## Operation-to-File Matrix

| Operation | Files Modified | Event Type |
|-----------|---------------|------------|
| `profile apply` (user) | `~/.claude/settings.json` (enabledPlugins declaratively replaced) | WRITE |
| `profile apply` (user) | `~/.claudeup/backups/user-scope.json` (backup before apply) | WRITE |
| `profile apply` (project) | `./.claude/settings.json` (enabledPlugins replaced) | WRITE |
| `profile apply` (project) | `./.mcp.json` (MCP servers written) | WRITE |
| `profile apply` (project) | `./.claudeup.json` (profile metadata) | WRITE |
| `profile apply` (local) | `./.claude/settings.local.json` (enabledPlugins replaced) | WRITE |
| `profile apply` (local) | `~/.claudeup/projects.json` (project mapping updated) | WRITE |
| `profile apply` (local) | `~/.claudeup/backups/local-scope-{hash}.json` (backup) | WRITE |
| `profile save` | `~/.claudeup/profiles/{name}.json` | WRITE |
| `plugin install/uninstall` | Via claude CLI - may update registry | INDIRECT |
| `marketplace add/remove` | Via claude CLI - updates `known_marketplaces.json` | INDIRECT |
| `mcp add/remove` | Via claude CLI - updates `~/.claude.json` | INDIRECT |
| `sandbox --copy-auth` | `~/.claudeup/sandboxes/{profile}/.claude.json` | WRITE |
| `scope restore` | Restores from `~/.claudeup/backups/` | READ+WRITE |
| `setup` | `~/.claudeup/config.json` (initial config) | WRITE |

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
