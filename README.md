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

#### Maintenance & Diagnostics

Diagnose and fix common issues with your Claude installation:

```bash
# Check for updates
claude-pm update --check-only   # See what's available
claude-pm update                # Apply updates

# Run comprehensive diagnostics
claude-pm doctor

# Fix known plugin path issues
claude-pm fix-paths

# Clean up stale plugin entries
claude-pm cleanup --dry-run    # See what would be removed
claude-pm cleanup               # Actually remove stale entries
```

**Doctor Command:**
- Checks all marketplaces exist
- Identifies fixable path issues (missing subdirectories)
- Identifies truly missing plugins
- Provides specific fix recommendations

**Fix-Paths Command:**
- Automatically corrects known path issues
- Handles marketplace subdirectory problems
- No reinstall needed

**Cleanup Command:**
- Removes plugin entries where directories don't exist
- `--dry-run` flag to preview changes
- `--reinstall` flag to show reinstall commands

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

There's a **known bug in the Claude CLI** that affects local plugins:

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

# Automatically fix the paths
claude-pm fix-paths
```

### Why the --summary Flag Exists

The `--summary` flag was added to help you:
1. **Identify affected plugins** - Local plugins are the ones with path bugs
2. **Understand your installation** - See how Claude CLI installed your plugins
3. **Track the fix** - Verify that `fix-paths` corrected the issues
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

### Phase 4: Project-Level Config & Polish

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
│   ├── config/             # Configuration management (future)
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
