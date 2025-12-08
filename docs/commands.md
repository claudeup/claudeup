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
claudeup setup                    # Interactive setup with default profile
claudeup setup --profile frontend # Setup with specific profile
claudeup setup --yes              # Non-interactive
```

### profile

Manage configuration profiles.

```bash
claudeup profile list             # List available profiles
claudeup profile show <name>      # Display profile contents
claudeup profile create <name>    # Save current setup as profile
claudeup profile use <name>       # Apply a profile
claudeup profile suggest          # Suggest profile for current project
```

## Sandbox

### sandbox

Run Claude Code in an isolated Docker container.

```bash
claudeup sandbox                       # Ephemeral session
claudeup sandbox --profile <name>      # Persistent session
claudeup sandbox --shell               # Drop to bash
claudeup sandbox --mount <host:container>  # Additional mount
claudeup sandbox --no-mount            # No working directory mount
claudeup sandbox --secret <name>       # Add secret
claudeup sandbox --no-secret <name>    # Exclude secret
claudeup sandbox --clean --profile <name>  # Reset sandbox state
```

## Status & Discovery

### status

Overview of your Claude Code installation.

```bash
claudeup status
```

Shows marketplaces, plugin counts, MCP servers, and any detected issues.

### plugins

List installed plugins.

```bash
claudeup plugins           # Full list with details
claudeup plugins --summary # Summary statistics only
```

### marketplace

Manage marketplace repositories.

```bash
claudeup marketplace list          # List installed marketplaces
```

### mcp

Manage MCP servers.

```bash
claudeup mcp list                              # List all MCP servers
claudeup mcp disable <plugin>:<server>         # Disable specific server
claudeup mcp enable <plugin>:<server>          # Re-enable server
```

## Enable/Disable

### enable

Re-enable a disabled plugin.

```bash
claudeup enable <plugin>@<marketplace>
```

### disable

Disable a plugin without uninstalling.

```bash
claudeup disable <plugin>@<marketplace>
```

Disabled plugins are stored in `~/.claudeup/config.json` and can be re-enabled.

## Maintenance

### doctor

Diagnose common issues with your installation.

```bash
claudeup doctor
```

Checks for missing marketplaces, broken plugin paths, and other problems.

### cleanup

Fix plugin issues.

```bash
claudeup cleanup              # Fix paths and remove broken entries
claudeup cleanup --dry-run    # Preview changes
claudeup cleanup --fix-only   # Only fix paths
claudeup cleanup --remove-only # Only remove broken entries
claudeup cleanup --reinstall  # Show reinstall commands
```

### update

Check for and apply updates.

```bash
claudeup update              # Apply updates
claudeup update --check-only # Preview without applying
```

## Configuration

Configuration is stored in `~/.claudeup/`:

```
~/.claudeup/
├── config.json       # Disabled plugins/servers, preferences
├── profiles/         # Saved profiles
└── sandboxes/        # Persistent sandbox state
```
