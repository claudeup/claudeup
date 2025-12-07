# Troubleshooting

## Quick Diagnosis

```bash
claudeup doctor
```

This checks for common issues and recommends fixes.

## Plugin Path Bug

There's a known bug in Claude CLI ([#11278](https://github.com/anthropics/claude-code/issues/11278), [#12457](https://github.com/anthropics/claude-code/issues/12457)) that causes broken plugin paths.

### Symptoms

- Plugins show as installed but don't work
- `claudeup status` shows "stale paths"
- Plugin commands, skills, and MCP servers are unavailable

### Cause

Claude CLI sets `isLocal: true` for marketplace plugins but creates paths without the `/plugins/` subdirectory:

```
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
claudeup plugins --summary
```

## Common Issues

### "Stale paths detected"

```bash
claudeup cleanup
```

### MCP server not working after changes

MCP server changes require restarting Claude Code to take effect.

### Plugin disabled but still appears

Check if it's in the disabled list:

```bash
cat ~/.claudeup/config.json | grep disabledPlugins
```

Re-enable with:

```bash
claudeup enable <plugin>@<marketplace>
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
