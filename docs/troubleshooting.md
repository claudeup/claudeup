# Troubleshooting

## Quick Diagnosis

```bash
claudeup doctor
```

This checks for common issues and recommends fixes.

## Investigating Configuration Changes

When something stops working or behaves unexpectedly, you can use event tracking to understand what changed.

### View Recent Changes

See what operations claudeup performed recently:

```bash
claudeup events audit
```

This shows a chronological log of all file operations (profile applies, plugin installs, etc.) with timestamps.

### Compare File Versions

When a configuration file has been modified, see exactly what changed:

```bash
# Quick overview (truncated for readability)
claudeup events diff --file ~/.claude/settings.json

# Full detailed diff (recommended for debugging)
claudeup events diff --file ~/.claude/settings.json --full
```

**Common files to check:**

- `~/.claude/settings.json` - User-level Claude settings
- `./.claude/settings.json` - Project-level Claude settings
- `./.claude/settings.local.json` - Local (machine-specific) settings
- `~/.claude/plugins/installed_plugins.json` - Plugin registry
- `~/.claude.json` - User MCP server configurations
- `./.mcp.json` - Project MCP server configurations
- `~/.claudeup/profiles/{name}.json` - Profile definitions
- `~/.claudeup/config.json` - claudeup global configuration

> 📖 **See also:** [File Operations Reference](file-operations.md) for a complete catalog of all files tracked by claudeup, including what operations modify each file.

**Understanding diff output:**

- **Default mode**: Nested objects shown as `{...}` to prevent terminal overflow. Good for quick overview.
- **Full mode** (`--full` flag):
  - Recursively diffs nested objects showing only changed fields
  - Color-coded symbols: 🟢 `+` added, 🔴 `-` removed, 🔵 `~` modified
  - Bold key names with gray `(added)`/`(removed)` labels
  - Shows the actual values that changed

**Example output:**

```text
~ plugins:
  ~ conductor@claude-conductor:
    ~ scope: "project" → "user"
    ~ installedAt: "2025-12-26T05:14:20.184Z" → "2025-12-26T19:11:07.257Z"
  ~ backend-api-security@claude-code-workflows:
    - projectPath: "/Users/markalston/workspace/claudeup" (removed)
```

### Common Scenarios

**Profile was marked as modified:**

```bash
# See what changed compared to saved profile
claudeup profile list

# If a profile shows (modified), check what changed:
claudeup events diff --file ~/.claude/settings.json --full
```

**Plugin stopped working:**

```bash
# Check recent plugin operations
claudeup events audit | grep plugin

# See if plugin configuration changed
claudeup events diff --file ~/.claude/plugins/installed_plugins.json --full
```

**MCP server configuration issues:**

```bash
# Check what changed in user-level MCP configs
claudeup events diff --file ~/.claude.json --full

# Check what changed in project-level MCP configs
claudeup events diff --file ./.mcp.json --full
```

**Something changed but you don't know when:**

```bash
# Review full audit trail
claudeup events audit

# Find specific file operations
claudeup events audit | grep "settings.json"
```

### Privacy Note

⚠️ Event logs may contain sensitive data if configuration files include API keys or tokens. Logs are stored locally at `~/.claudeup/events/operations.log` with owner-only permissions (0600).

To disable event tracking, set `monitoring.enabled: false` in `~/.claudeup/config.json`.

## Plugin Path Bug

There's a known bug in Claude CLI ([#11278](https://github.com/anthropics/claude-code/issues/11278), [#12457](https://github.com/anthropics/claude-code/issues/12457)) that causes broken plugin paths.

### Symptoms

- Plugins show as installed but don't work
- `claudeup status` shows "stale paths"
- Plugin commands, skills, and MCP servers are unavailable

### Cause

Claude CLI sets `isLocal: true` for marketplace plugins but creates paths without the `/plugins/` subdirectory:

```text
Wrong: ~/.claude/plugins/marketplaces/claude-code-plugins/hookify
Right: ~/.claude/plugins/marketplaces/claude-code-plugins/plugins/hookify
```

### Fix

```bash
claudeup cleanup
```

This automatically corrects the paths. Use `--dry-run` to preview changes first.

## Plugin Types

Understanding plugin types helps with troubleshooting:

### Cached Plugins (`isLocal: false`)

- Copied to `~/.claude/plugins/cache/`
- Independent of marketplace directory
- More stable, less prone to path issues

### Local Plugins (`isLocal: true`)

- Reference marketplace directory directly
- Path: `~/.claude/plugins/marketplaces/<marketplace>/plugins/<plugin>`
- Affected by the path bug above

Check your plugin types:

```bash
claudeup plugin list --summary
```

## Common Issues

### "Stale paths detected"

```bash
claudeup cleanup
```

### MCP server not working after changes

MCP server changes require restarting Claude Code to take effect.

### Plugin disabled but still appears

Re-enable with:

```bash
claudeup plugin enable <plugin-name>
```

### Marketplace missing

If a marketplace was deleted but plugins still reference it:

```bash
claudeup doctor        # Diagnose
claudeup cleanup       # Remove broken references
```

### Secrets not resolving

Check your secret configuration in the profile. Resolution tries sources in order:

1. Environment variable
2. 1Password (`op` CLI must be installed and signed in)
3. macOS Keychain

Test 1Password:

```bash
op read "op://Private/My Secret/credential"
```

### Sandbox won't start

Check Docker is running:

```bash
docker info
```

Pull the image manually:

```bash
docker pull ghcr.io/claudeup/claudeup-sandbox:latest
```

## Getting Help

If `claudeup doctor` and `claudeup cleanup` don't resolve your issue:

1. Check existing issues: https://github.com/claudeup/claudeup/issues
2. Open a new issue with output from `claudeup doctor`
