# Command Reference

## Global Flags

| Flag | Description |
|------|-------------|
| `--claude-dir` | Override Claude installation directory (default: `~/.claude`) |
| `-y, --yes` | Skip interactive prompts, use defaults |

## Setup & Profiles

### setup

First-time setup or reset of Claude Code installation.

```bash
claude-pm setup                    # Interactive setup with default profile
claude-pm setup --profile frontend # Setup with specific profile
claude-pm setup --yes              # Non-interactive
```

### profile

Manage configuration profiles.

```bash
claude-pm profile list             # List available profiles
claude-pm profile show <name>      # Display profile contents
claude-pm profile create <name>    # Save current setup as profile
claude-pm profile use <name>       # Apply a profile
claude-pm profile suggest          # Suggest profile for current project
```

## Sandbox

### sandbox

Run Claude Code in an isolated Docker container.

```bash
claude-pm sandbox                       # Ephemeral session
claude-pm sandbox --profile <name>      # Persistent session
claude-pm sandbox --shell               # Drop to bash
claude-pm sandbox --mount <host:container>  # Additional mount
claude-pm sandbox --no-mount            # No working directory mount
claude-pm sandbox --secret <name>       # Add secret
claude-pm sandbox --no-secret <name>    # Exclude secret
claude-pm sandbox --clean --profile <name>  # Reset sandbox state
```

## Status & Discovery

### status

Overview of your Claude Code installation.

```bash
claude-pm status
```

Shows marketplaces, plugin counts, MCP servers, and any detected issues.

### plugins

List installed plugins.

```bash
claude-pm plugins           # Full list with details
claude-pm plugins --summary # Summary statistics only
```

### marketplaces

List installed marketplace repositories.

```bash
claude-pm marketplaces
```

### mcp

Manage MCP servers.

```bash
claude-pm mcp list                              # List all MCP servers
claude-pm mcp disable <plugin>:<server>         # Disable specific server
claude-pm mcp enable <plugin>:<server>          # Re-enable server
```

## Enable/Disable

### enable

Re-enable a disabled plugin.

```bash
claude-pm enable <plugin>@<marketplace>
```

### disable

Disable a plugin without uninstalling.

```bash
claude-pm disable <plugin>@<marketplace>
```

Disabled plugins are stored in `~/.claude-pm/config.json` and can be re-enabled.

## Maintenance

### doctor

Diagnose common issues with your installation.

```bash
claude-pm doctor
```

Checks for missing marketplaces, broken plugin paths, and other problems.

### cleanup

Fix plugin issues.

```bash
claude-pm cleanup              # Fix paths and remove broken entries
claude-pm cleanup --dry-run    # Preview changes
claude-pm cleanup --fix-only   # Only fix paths
claude-pm cleanup --remove-only # Only remove broken entries
claude-pm cleanup --reinstall  # Show reinstall commands
```

### update

Check for and apply updates.

```bash
claude-pm update              # Apply updates
claude-pm update --check-only # Preview without applying
```

## Configuration

Configuration is stored in `~/.claude-pm/`:

```
~/.claude-pm/
├── config.json       # Disabled plugins/servers, preferences
├── profiles/         # Saved profiles
└── sandboxes/        # Persistent sandbox state
```
