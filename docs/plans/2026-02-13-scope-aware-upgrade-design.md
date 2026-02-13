# Scope-Aware Plugin Upgrade

## Problem

The `claudeup upgrade` and `claudeup outdated` commands only operate on user-scoped plugin installations. The V2 plugin registry (`installed_plugins.json`) supports multiple scope entries per plugin (user, project, local), but `GetPlugin()`, `GetAllPlugins()`, and the upgrade/outdated logic silently prefer the user scope. Project-scoped and local-scoped installations are never checked or upgraded.

## Decision: Approach C -- Replace User-Biased API

Remove the user-biased `GetPlugin`/`GetAllPlugins` methods entirely. All callers must pass explicit scopes. This is safe because the `internal/claude` package has no external consumers.

## New PluginRegistry API

### Remove

- `GetPlugin(name string) (PluginMetadata, bool)` -- silently prefers user scope
- `GetAllPlugins() map[string]PluginMetadata` -- same bias
- `PluginExists(name string) bool` -- wraps GetPlugin
- `IsPluginInstalled(name string) bool` -- alias for PluginExists

### Add

- `GetPluginAtScope(name, scope string) (PluginMetadata, bool)` -- explicit scope lookup
- `GetPluginInstances(name string) []PluginMetadata` -- all scope instances for a plugin
- `GetPluginsAtScopes(scopes []string) []ScopedPlugin` -- all plugin instances across given scopes
- `PluginExistsAtScope(name, scope string) bool` -- explicit scope check
- `PluginExistsAtAnyScope(name string) bool` -- for callers that don't care about scope
- `RemovePluginAtScope(name, scope string) bool` -- removes a single scope instance, cleans up entry if last

### Keep Unchanged

- `SetPlugin(name, metadata)` -- already scope-aware via `metadata.Scope`
- `DisablePlugin(name) bool` -- removes all scope instances
- `EnablePlugin(name, metadata)` -- delegates to SetPlugin
- `RemovePlugin(name) bool` -- removes all scope instances

## New Types

```go
// ScopedPlugin pairs a plugin name with its scope-specific metadata
type ScopedPlugin struct {
    Name string
    PluginMetadata
}
```

## Upgrade & Outdated Command Changes

### Context-aware scope detection

Default behavior: user scope always, project/local only when in a project directory. Reuses `IsProjectContext` (exported from `plugin_analysis.go`).

New `--all` flag on both commands forces all three scopes regardless of directory context.

Helper function:

```go
func availableScopes(allFlag bool) []string {
    if allFlag {
        return claude.ValidScopes
    }
    scopes := []string{"user"}
    if claude.IsProjectContext(claudeDir, projectDir) {
        scopes = append(scopes, "project", "local")
    }
    return scopes
}
```

### checkPluginUpdates refactored

- New signature: `checkPluginUpdates(plugins *PluginRegistry, marketplaces MarketplaceRegistry, scopes []string) []PluginUpdate`
- Uses `GetPluginsAtScopes(scopes)` instead of `GetAllPlugins()`
- `PluginUpdate` gains a `Scope string` field

### updatePlugin refactored

- New signature: `updatePlugin(name string, scope string, plugins *PluginRegistry) error`
- Uses `GetPluginAtScope(name, scope)` instead of `GetPlugin(name)`

### Output format

Always shows scope label per line:

```
Checking Plugins (3)
  ⚠ hookify@plugins (user): Update available
  ⚠ hookify@plugins (project): Update available
  ✓ tdd@superpowers (user): Up to date
```

## Caller Migration

| File | Lines | Current Call | Replacement | Rationale |
|------|-------|-------------|-------------|-----------|
| `upgrade.go` | 171, 339, 395, 463 | `GetAllPlugins()`, `GetPlugin()`, `SetPlugin()` | `GetPluginsAtScopes(scopes)`, `GetPluginAtScope(name, scope)`, `SetPlugin()` (unchanged) | Primary target of this change |
| `outdated.go` | 81-82 | `GetAllPlugins()` | `GetPluginsAtScopes(scopes)` | Same scope-awareness as upgrade |
| `doctor.go` | 114, 242 | `GetAllPlugins()` | `GetPluginsAtScopes(ValidScopes)`, `PluginExistsAtAnyScope()` | Doctor checks all scopes; path issue output includes scope label |
| `status.go` | 137, 165, 178 | `GetAllPlugins()` | `GetPluginsAtScopes(scopes)` + dedup by name | Context-aware like upgrade; dedup prevents over-counting enabled plugins |
| `cleanup.go` | 92, 191, 193 | `GetAllPlugins()`, `GetPlugin()`, `SetPlugin()` | `GetPluginsAtScopes(ValidScopes)`, `GetPluginAtScope(name, scope)`, `SetPlugin()` | Cleanup checks everything; issue struct gains Scope field |
| `profile_cmd.go` | 1244-1246 | `GetAllPlugins()`, `DisablePlugin()` | `GetPluginsAtScopes(ValidScopes)`, `RemovePluginAtScope(name, scope)` | Stale cleanup removes only the broken instance, preserving valid installations at other scopes |
| `plugin.go` | 296, 358, 396 | `PluginExists()` | `PluginExistsAtAnyScope()` | Installed badge, scope doesn't matter |
| `plugin_search.go` | 101 | `PluginExists()` | `PluginExistsAtAnyScope()` | Same as above |
| `mcp/discovery.go` | 43 | `GetAllPlugins()` | `GetPluginsAtScopes(ValidScopes)` + dedup by name | Discover across all scopes; dedup prevents duplicate MCP server registration |
| `plugin_analysis.go` | 147 | `PluginExists()` | `PluginExistsAtAnyScope()` | Orphan detection |
| `profile/apply_concurrent.go` | 63 | `PluginExists()` | `PluginExistsAtScope(plugin, scope)` | Profile apply operates at a specific scope |
| `plugins_test.go` | multiple | `GetPlugin()`, `PluginExists()`, etc. | New method names with explicit scopes | Test updates |

## Multi-Scope Dedup

`GetPluginsAtScopes` returns one entry per scope instance. When a plugin is installed at multiple scopes, callers that iterate by plugin name (not by instance) must deduplicate to avoid over-counting or duplicate registrations. Affected callers use a `seen` map to skip duplicate plugin names:

- `status.go` -- prevents `enabledCount` over-counting and duplicate `stalePlugins` entries
- `mcp/discovery.go` -- prevents duplicate `PluginMCPServers` entries

## What's Not Changing

- `installed_plugins.json` V2 format -- already supports scopes
- Marketplace handling -- not scope-dependent
- `DisablePlugin`/`RemovePlugin`/`EnablePlugin` -- already correct
- `SetPlugin` -- already uses `metadata.Scope`
