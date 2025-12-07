# claude-pm

A comprehensive CLI tool for managing Claude Code plugins, marketplaces, and MCP servers.

## Overview

`claude-pm` provides visibility into and control over your Claude Code installation, including:

- Installed plugins and their state (enable/disable individual plugins)
- Marketplace repositories
- MCP server configuration (granular control over individual MCP servers)
- Plugin updates and maintenance

This is a community-built tool that provides a unified interface for managing Claude Code plugins and MCP servers.

## Installation

### From Source

```bash
go install github.com/malston/claude-pm/cmd/claude-pm@latest
```

### From GitHub Releases

Download the appropriate binary for your platform from the [releases page](https://github.com/malston/claude-pm/releases).

### Building Locally

```bash
git clone https://github.com/malston/claude-pm.git
cd claude-pm
go build -o bin/claude-pm ./cmd/claude-pm
```

## Usage

### Quick Start

```bash
# Show overview of your Claude installation
claude-pm status

# List all installed plugins
claude-pm plugins

# Enable/disable plugins
claude-pm disable compound-engineering@every-marketplace
claude-pm enable compound-engineering@every-marketplace

# List installed marketplaces
claude-pm marketplaces

# List MCP servers by plugin
claude-pm mcp list

# Enable/disable specific MCP servers
claude-pm mcp disable superpowers-chrome@superpowers-marketplace:chrome
claude-pm mcp enable superpowers-chrome@superpowers-marketplace:chrome
```

### Commands

#### Status

Show an overview of your Claude Code installation:

```bash
claude-pm status
```

Example output:

```
╔════════════════════════════════════════╗
║           claude-pm Status             ║
╚════════════════════════════════════════╝

Marketplaces (7)
  ✓ superpowers-marketplace
  ✓ claude-code-plugins
  ✓ every-marketplace
  ...

Plugins (27 total)
  ✓ 8 enabled

MCP Servers
  → Run 'claude-pm mcp list' for details

Issues Detected
  ⚠ 19 plugins have stale paths
  → Run 'claude-pm doctor' for details
```

#### Plugins

List all installed plugins with detailed information:

```bash
claude-pm plugins

# Show only summary statistics
claude-pm plugins --summary
```

Shows version, status, installation path, and type (local/cached) for each plugin.

**Plugin Types:**

- **Cached:** Plugin copied to `~/.claude/plugins/cache/`
- **Local:** Plugin referenced from marketplace directory

#### Marketplaces

List all installed marketplace repositories:

```bash
claude-pm marketplaces
```

Shows source, repository, location, and last update time for each marketplace.

#### MCP Servers

List MCP servers grouped by the plugin that provides them:

```bash
claude-pm mcp list
```

Shows command, arguments, and environment variables for each MCP server.

#### Enable/Disable Plugins

Enable or disable individual plugins:

```bash
# Disable a plugin
claude-pm disable hookify@claude-code-plugins

# Re-enable a plugin
claude-pm enable hookify@claude-code-plugins
```

When you disable a plugin:

- It's removed from `installed_plugins.json`
- Its metadata is saved to `~/.claude-pm/config.json`
- All commands, agents, skills, and MCP servers become unavailable

When you re-enable it:

- The metadata is restored from config
- The plugin becomes available again immediately
- No need to reinstall

#### Enable/Disable MCP Servers

Control individual MCP servers without disabling the entire plugin:

```bash
# Disable a specific MCP server
claude-pm mcp disable compound-engineering@every-marketplace:playwright

# Re-enable it
claude-pm mcp enable compound-engineering@every-marketplace:playwright
```

This is useful for:

- Reducing MCP context usage
- Disabling heavy servers (like Playwright) when not needed
- Keeping plugin features (commands, skills) while removing MCP tools

**Note:** MCP server changes require restarting Claude Code to take effect.

#### Setup

First-time setup or reset of your Claude Code installation:

```bash
# Interactive setup with default profile
claude-pm setup

# Setup with a specific profile
claude-pm setup --profile frontend

# Non-interactive setup (for scripts)
claude-pm setup --yes
```

The setup command:
1. Installs Claude CLI if missing (with your permission)
2. Creates the profiles directory with a default profile
3. Detects existing installations and offers to save them as a profile
4. Applies your chosen profile

#### Profiles

Profiles are saved configurations of plugins, MCP servers, and marketplaces. Use them to:
- Save your current setup for later
- Switch between different configurations (e.g., frontend vs backend)
- Share configurations between machines

```bash
# List available profiles
claude-pm profile list

# Show a profile's contents
claude-pm profile show default

# Create a profile from your current setup
claude-pm profile create my-config

# Apply a profile (replaces current config)
claude-pm profile use frontend

# Get a profile suggestion based on your project
claude-pm profile suggest
```

**Profile Storage:**

Profiles are stored in `~/.claude-pm/profiles/`:

```json
{
  "name": "frontend",
  "description": "Frontend development with React tooling",
  "plugins": ["superpowers@superpowers-marketplace"],
  "mcpServers": [{"name": "context7", "command": "npx", "args": ["-y", "@context7/mcp"]}],
  "marketplaces": [{"source": "github", "repo": "anthropics/claude-code-plugins"}]
}
```

**Secret Resolution:**

MCP servers often need API keys. Profiles support multiple secret backends:

```json
"secrets": {
  "API_KEY": {
    "sources": [
      {"type": "env", "key": "API_KEY"},
      {"type": "1password", "ref": "op://Private/API Key/credential"},
      {"type": "keychain", "service": "my-api", "account": "default"}
    ]
  }
}
```

Resolution tries each source in order. First success wins.

| Backend | Platform | Requirement |
|---------|----------|-------------|
| `env` | All | Environment variable set |
| `1password` | All | `op` CLI installed |
| `keychain` | macOS | Keychain item exists |

#### Maintenance & Diagnostics

Diagnose and fix common issues with your Claude installation:

```bash
# Check for updates
claude-pm update --check-only   # See what's available
claude-pm update                # Apply updates

# Run comprehensive diagnostics
claude-pm doctor

# Fix and clean up plugin issues (default: fix + remove)
claude-pm cleanup               # Fix paths and remove broken entries
claude-pm cleanup --dry-run     # Preview changes
claude-pm cleanup --fix-only    # Only fix paths, don't remove
claude-pm cleanup --remove-only # Only remove broken entries
```

**Doctor Command:**

- Checks all marketplaces exist
- Identifies fixable path issues (missing subdirectories)
- Identifies truly missing plugins
- Recommends using `cleanup` to resolve issues

**Cleanup Command:**

By default, fixes and removes plugin issues in one operation:

- **Fixes** plugins with correctable path issues (missing subdirectories)
- **Removes** plugin entries that are truly broken (no valid path exists)
- `--fix-only` flag to only fix paths without removing entries
- `--remove-only` flag to only remove broken entries without fixing
- `--dry-run` flag to preview changes without applying them
- `--reinstall` flag to show reinstall commands for removed plugins

**Update Command:**

- Checks marketplaces for git updates
- Checks plugins for newer marketplace commits
- `--check-only` flag to preview without applying
- Updates marketplaces via git pull
- Updates plugin commit references

## Understanding Plugin Types

Claude Code installs plugins in two different ways, which affects how they're stored and whether they're prone to path issues.

### Plugin Storage Types

**Cached Plugins (`isLocal: false`)**

- Plugin is **copied** to `~/.claude/plugins/cache/plugin-name`
- Creates an independent copy separate from the marketplace
- More stable - less prone to path issues
- Example: Most superpowers-marketplace plugins

**Local Plugins (`isLocal: true`)**

- Plugin **references** the marketplace directory directly
- Path: `~/.claude/plugins/marketplaces/marketplace-name/plugins/plugin-name`
- No separate copy - points directly to marketplace
- Example: Most claude-code-plugins plugins

### The Plugin Path Bug

There's a **known bug in the Claude CLI** that affects local plugins (see [Issue #11278](https://github.com/anthropics/claude-code/issues/11278) and [Issue #12457](https://github.com/anthropics/claude-code/issues/12457)):

**The Problem:**

1. Claude CLI sets `isLocal: true` for marketplace plugins
2. BUT creates paths **without** the `/plugins/` subdirectory
3. Results in broken paths like:
   - Wrong: `~/.claude/plugins/marketplaces/claude-code-plugins/hookify`
   - Right: `~/.claude/plugins/marketplaces/claude-code-plugins/plugins/hookify`

**The Impact:**

- Plugins appear in `installed_plugins.json` but paths don't exist
- Shows as "stale" in `claude-pm status`
- Plugin commands, skills, and MCP servers are unavailable

**The Fix:**

```bash
# Diagnose the issue
claude-pm doctor

# See which plugins are affected
claude-pm plugins --summary

# Fix the paths (and optionally remove any truly broken entries)
claude-pm cleanup
```

### Why the --summary Flag Exists

The `--summary` flag was added to help you:

1. **Identify affected plugins** - Local plugins are the ones with path bugs
2. **Understand your installation** - See how Claude CLI installed your plugins
3. **Track the fix** - Verify that `cleanup` corrected the issues
4. **Future troubleshooting** - Quickly check plugin type distribution

**Example:**

```bash
$ claude-pm plugins --summary

By Type:
  Cached: 7 (copied to ~/.claude/plugins/cache/)
  Local:  20 (referenced from marketplace)
```

If you have many "local" plugins, they were likely affected by the path bug. Run `claude-pm doctor` to check.

## Configuration

### Global Config File

`claude-pm` stores its configuration in `~/.claude-pm/config.json`:

```json
{
  "disabledPlugins": {
    "hookify@claude-code-plugins": {
      "version": "1.0.0",
      "installPath": "/Users/you/.claude/plugins/...",
      ...
    }
  },
  "disabledMcpServers": [
    "compound-engineering@every-marketplace:playwright"
  ],
  "claudeDir": "/Users/you/.claude",
  "preferences": {
    "autoUpdate": false,
    "verboseOutput": false
  }
}
```

The config file is created automatically on first use.

### Global Flags

- `--claude-dir` - Override the Claude installation directory (default: `~/.claude`)

Example:

```bash
claude-pm --claude-dir /custom/path status
```

## Roadmap

### Phase 1: Core Status & Discovery ✅ (Complete)

- ✅ `claude-pm status` - Overview of installation
- ✅ `claude-pm plugins` - List all plugins
- ✅ `claude-pm marketplaces` - List marketplaces
- ✅ `claude-pm mcp list` - List MCP servers

### Phase 2: Enable/Disable Control ✅ (Complete)

- ✅ `claude-pm enable <plugin>` - Enable a plugin
- ✅ `claude-pm disable <plugin>` - Disable a plugin
- ✅ `claude-pm mcp disable <plugin>:<server>` - Disable specific MCP server
- ✅ `claude-pm mcp enable <plugin>:<server>` - Re-enable MCP server
- ✅ Global config file for tracking disabled plugins and MCP servers

### Phase 3: Maintenance & Diagnostics ✅ (Complete)

- ✅ `claude-pm doctor` - Diagnose common issues
- ✅ `claude-pm fix-paths` - Fix plugin path issues automatically
- ✅ `claude-pm cleanup` - Clean stale plugin entries
- ✅ `claude-pm update` - Check and apply marketplace and plugin updates

### Phase 4: Profiles & Setup ✅ (Complete)

- ✅ `claude-pm setup` - First-time setup with profile selection
- ✅ `claude-pm profile list` - List available profiles
- ✅ `claude-pm profile show` - Display profile contents
- ✅ `claude-pm profile create` - Snapshot current state as profile
- ✅ `claude-pm profile use` - Apply a profile (replace strategy)
- ✅ `claude-pm profile suggest` - Project-based profile detection
- ✅ Secret resolution (env, 1Password, macOS Keychain)
- ✅ Embedded default profile

### Phase 5: Project-Level Config & Polish

- `claude-pm init` - Initialize project config
- `.claude-pm.json` - Project-specific configuration
- Config merging (global + project)
- `claude-pm maintain` - Interactive TUI mode
- `claude-pm config show` - Show effective merged config

## Architecture

The project is organized as follows:

```
claude-pm/
├── cmd/claude-pm/          # Main entry point
├── internal/
│   ├── claude/             # Claude data structures (plugins, marketplaces)
│   ├── mcp/                # MCP server discovery and management
│   ├── commands/           # CLI commands
│   ├── config/             # Global configuration management
│   ├── profile/            # Profile loading, saving, and application
│   ├── secrets/            # Secret resolution (env, 1Password, Keychain)
│   └── ui/                 # UI components (future)
```

## Development

### Prerequisites

- Go 1.21 or later

### Building

```bash
go build -o bin/claude-pm ./cmd/claude-pm
```

### Testing

```bash
# Test against your actual Claude installation
./bin/claude-pm status
./bin/claude-pm plugins
./bin/claude-pm mcp list
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Acknowledgments

This tool was created to improve upon the existing bash scripts in the Claude Code ecosystem. It aims to provide a more robust, cross-platform solution for managing Claude Code installations.
