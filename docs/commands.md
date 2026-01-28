---
title: Command Reference
---

# Command Reference

## Global Flags

| Flag           | Description                                                   |
| -------------- | ------------------------------------------------------------- |
| `--claude-dir` | Override Claude installation directory (default: `~/.claude`) |
| `-y, --yes`    | Skip interactive prompts, use defaults                        |

## Setup & Profiles

### setup

Initialize or configure Claude Code installation.

**Behavior depends on installation state:**

- **Existing installation detected**: Preserves your current settings and offers to save them as a profile
- **Fresh installation**: Applies the default profile (or specified `--profile`)

```bash
claudeup setup                    # Interactive setup
claudeup setup --profile frontend # Setup with specific profile (fresh installs only)
claudeup setup --yes              # Non-interactive
```

**For existing installations:**

When claudeup detects an existing Claude Code installation (plugins, MCP servers, settings), it preserves your configuration instead of overwriting it. You'll be prompted to:

- **Save as profile** - Save your current setup as a named profile for future use
- **Continue without saving** - Keep your existing configuration, proceed with setup
- **Abort** - Cancel the setup process

The `--profile` flag is ignored for existing installations since your current settings take precedence.

**For fresh installations:**

When no existing configuration is detected, setup applies a profile to get you started:

```bash
claudeup setup                    # Applies the "default" profile
claudeup setup --profile backend  # Applies the "backend" profile
```

If the specified profile doesn't exist, setup fails with a helpful error message listing available profiles.

### profile

Manage configuration profiles.

```bash
claudeup profile list                        # List available profiles
claudeup profile show <name>                 # Display profile contents
claudeup profile current                     # Show active profile (with scope)
claudeup profile status [name]               # Show differences from current Claude state
claudeup profile diff <name>                 # Compare customized built-in to original
claudeup profile save [name]                 # Save current setup as profile
claudeup profile create <name>               # Create profile with interactive wizard
claudeup profile clone <name>                # Clone an existing profile
claudeup profile apply <name>                # Apply a profile (user scope)
claudeup profile sync                        # Install plugins from .claudeup.json
claudeup profile suggest                     # Suggest profile for current project
claudeup profile delete <name>               # Delete a custom profile
claudeup profile restore <name>              # Restore a built-in profile
claudeup profile reset <name>                # Remove everything a profile installed
claudeup profile rename <old> <new>          # Rename a custom profile
claudeup profile clean <plugin>              # Remove orphaned plugin from config

# With description flag
claudeup profile save my-work --description "My work setup"
claudeup profile clone home --from work --description "Home setup"
```

#### Project-Level Profiles

Apply profiles at project scope for team sharing:

```bash
# Apply profile to current project (creates .mcp.json + .claudeup.json)
claudeup profile apply frontend --scope project

# Team members clone and sync plugins
claudeup profile sync              # Install plugins from .claudeup.json
claudeup profile sync --dry-run    # Preview without changes

# Apply profile locally only (not shared via git)
claudeup profile apply frontend --scope local
```

**Scope options:**

| Scope     | MCP Servers      | Plugins        | Shared?       |
| --------- | ---------------- | -------------- | ------------- |
| `user`    | `~/.claude.json` | user-scoped    | No            |
| `project` | `.mcp.json`      | project-scoped | Yes (via git) |
| `local`   | `~/.claude.json` | local-scoped   | No            |

**`profile apply` flags:**

| Flag               | Description                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------- |
| `--scope`          | Apply scope: user, project, or local (default: user, or project if .claudeup.json exists) |
| `--reset`          | Clear target scope before applying (replaces instead of adding)                           |
| `--setup`          | Force post-apply setup wizard to run                                                      |
| `--no-interactive` | Skip post-apply setup wizard (for CI/scripting)                                           |
| `-f, --force`      | Force reapply even with unsaved changes                                                   |
| `--reinstall`      | Force reinstall all plugins and marketplaces                                              |
| `--no-progress`    | Disable progress display (for CI/scripting)                                               |

**Reset mode:**

The `--reset` flag clears the target scope before applying the profile:

```bash
# Replace user scope with new profile (instead of merging)
claudeup profile apply backend-stack --reset

# Replace without prompts (for scripting)
claudeup profile apply backend-stack --reset -y
```

A backup is created automatically when using `--reset` (unless `-y` is used).
Use `claudeup scope restore user` to recover if needed.

**Files created by `--scope project`:**

- `.mcp.json` - MCP servers (Claude auto-loads this)
- `.claudeup.json` - Plugins manifest (team runs `profile sync`)

**Team workflow:**

1. One team member applies the profile with `--scope project`
2. Commit `.mcp.json` and `.claudeup.json` to version control
3. Other team members clone/pull and run `claudeup profile sync`
4. MCP servers load automatically; plugins are installed by sync

**Scope precedence:**

When determining the active profile, `profile current` checks in order:

1. **Project scope** - `.claudeup.json` in current directory
2. **Local scope** - Entry in `~/.claudeup/projects.json` for current directory
3. **User scope** - Global profile from `~/.claudeup/config.json`

The first match wins. This means project-level configuration always takes precedence over personal settings when you're in a project directory.

**Secrets and project scope:**

MCP servers often require secrets (API keys, tokens). When using `--scope project`:

- Secrets are **not** stored in `.mcp.json` - only secret references
- Each team member must have the referenced secrets available locally
- Common secret sources: environment variables, 1Password, system keychain
- The sync command does not handle secrets - configure them separately

Example `.mcp.json` with secret reference:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@anthropic/github-mcp"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
```

Team members set `GITHUB_TOKEN` in their environment before using Claude.

#### Profile Status vs Diff

Two commands for understanding profile differences:

**`profile status [name]`** - Shows how a profile differs from your current Claude configuration:

```bash
# Show drift for active profile
claudeup profile status

# Show drift for specific profile
claudeup profile status backend-stack
```

Use this to see:

- Which plugins are missing from your configuration
- Which plugins are extra (not in the profile)
- Differences at each scope (user, project, local)
- Actionable commands to fix drift

**`profile diff <name>`** - Compares a customized built-in profile to its original:

```bash
# See what you changed in the default profile
claudeup profile diff default

# See customizations to the frontend profile
claudeup profile diff frontend
```

Use this to see:

- What plugins you added to a built-in profile
- What plugins you removed from a built-in profile
- Description changes
- Only works with built-in profiles that have been customized

**When to use each:**

| Scenario                                     | Command                               |
| -------------------------------------------- | ------------------------------------- |
| "Does my Claude match my profile?"           | `profile status`                      |
| "What did I change from the original?"       | `profile diff`                        |
| "Why does `profile list` show (customized)?" | `profile diff`                        |
| "How do I fix drift?"                        | `profile status` (shows fix commands) |

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

## Scope Management

### scope

View and manage Claude Code settings across different scopes.

```bash
claudeup scope list                    # Show all scopes
claudeup scope list --scope user       # Show only user scope
claudeup scope list --scope project    # Show only project scope
claudeup scope clear user              # Clear user scope (type 'yes' to confirm)
claudeup scope clear user --backup     # Create backup before clearing
claudeup scope clear project --force   # Clear project scope without confirmation
claudeup scope clear local             # Clear local scope with confirmation
claudeup scope restore user            # Restore from backup
```

Claude Code uses three scope levels (in precedence order):

| Scope     | Location                        | Description                          |
| --------- | ------------------------------- | ------------------------------------ |
| `local`   | `./.claude/settings.local.json` | Machine-specific, highest precedence |
| `project` | `./.claude/settings.json`       | Project-level, shared via git        |
| `user`    | `~/.claude/settings.json`       | Global personal defaults             |

**`scope list` flags:**

| Flag      | Description                              |
| --------- | ---------------------------------------- |
| `--scope` | Filter to scope: user, project, or local |

**`scope clear` flags:**

| Flag       | Description                   |
| ---------- | ----------------------------- |
| `--force`  | Skip confirmation prompts     |
| `--backup` | Create backup before clearing |

**`scope restore` flags:**

| Flag      | Description               |
| --------- | ------------------------- |
| `--force` | Skip confirmation prompts |

**Notes:**

- User scope requires typing 'yes' to confirm (extra safety)
- Backups are stored in `~/.claudeup/backups/`
- Project scope cannot be restored (use `git checkout` instead)
- Each backup overwrites the previous one (only the most recent backup is kept)
- Local scope backups are project-specific (different projects have separate backups)

**Troubleshooting backup/restore:**

| Problem                            | Cause                                      | Solution                                                             |
| ---------------------------------- | ------------------------------------------ | -------------------------------------------------------------------- |
| "no backup found"                  | No backup exists for this scope            | Run `scope clear --backup` first to create one                       |
| "no backup found" for local scope  | Backup was made from a different directory | Run restore from the same project directory where backup was created |
| Restore contains old data          | Backups are overwritten on each save       | Only the most recent backup is available; older backups are lost     |
| "homeDir must be an absolute path" | Internal error                             | Report as bug - this shouldn't happen in normal use                  |
| "source is a symlink"              | Settings file is a symlink                 | Remove symlink and use a regular file                                |

## Status & Discovery

### status

Overview of your Claude Code installation.

```bash
claudeup status                        # Show status for all scopes
claudeup status --scope user           # Show only user scope
claudeup status --scope project        # Filter to project scope
```

Shows marketplaces, plugin counts, MCP servers, and any detected issues.

**Flags:**

| Flag      | Description                              |
| --------- | ---------------------------------------- |
| `--scope` | Filter to scope: user, project, or local |

### plugin

Manage plugins.

```bash
claudeup plugin list              # Full list with details
claudeup plugin list --summary    # Summary statistics only
claudeup plugin list --by-scope   # Group enabled plugins by scope
claudeup plugin enable <name>     # Enable a disabled plugin
claudeup plugin disable <name>    # Disable a plugin
claudeup plugin browse <marketplace>                  # List available plugins
claudeup plugin browse <marketplace> --format table  # Table format
claudeup plugin browse <marketplace> --show <name>   # Show plugin contents
claudeup plugin show <plugin>@<marketplace>          # Show plugin contents
claudeup plugin search <query>                        # Search installed plugins
claudeup plugin search <query> --all                  # Search all cached plugins
```

**`plugin list` flags:**

| Flag         | Description                    |
| ------------ | ------------------------------ |
| `--summary`  | Show only summary statistics   |
| `--enabled`  | Show only enabled plugins      |
| `--disabled` | Show only disabled plugins     |
| `--format`   | Output format (table)          |
| `--by-scope` | Group enabled plugins by scope |

**`plugin browse` flags:**

| Flag       | Description                        |
| ---------- | ---------------------------------- |
| `--format` | Output format (table)              |
| `--show`   | Show contents of a specific plugin |

**`plugin show`:**

Display the directory structure of a plugin in a marketplace. Shows agents, commands, skills, and other files.

```bash
# Direct access
claudeup plugin show observability-monitoring@claude-code-workflows

# While browsing
claudeup plugin browse claude-code-workflows --show observability-monitoring
```

**`plugin search`:**

Search across plugins to find those with specific capabilities. Searches plugin names, descriptions, keywords, and component names/descriptions.

```bash
# Search installed plugins
claudeup plugin search tdd

# Search all cached plugins (all synced marketplaces)
claudeup plugin search "skill" --all

# Filter by component type
claudeup plugin search commit --type commands

# Search specific marketplace
claudeup plugin search api --marketplace claude-code-workflows

# Group results by component type
claudeup plugin search frontend --by-component

# Use regex patterns
claudeup plugin search "front.?end|react" --regex --all

# Output formats
claudeup plugin search api --format table
claudeup plugin search api --format json
```

**`plugin search` flags:**

| Flag             | Description                                                  |
| ---------------- | ------------------------------------------------------------ |
| `--all`          | Search all cached plugins, not just installed                |
| `--type`         | Filter by component type: skills, commands, agents           |
| `--marketplace`  | Limit search to specific marketplace                         |
| `--by-component` | Group results by component type instead of plugin            |
| `--regex`        | Treat query as regular expression                            |
| `--format`       | Output format: json, table (default: styled text with trees) |

**Output formats:**

- **Default** - Styled text showing matching plugins with their components and directory trees
- **Table** - Tabular view with plugin, type, component, and description columns
- **JSON** - Machine-readable output for scripting

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

Update the claudeup CLI to the latest version.

```bash
claudeup update              # Update to latest version
claudeup update --check-only # Check for updates without applying
```

### upgrade

Update marketplaces and plugins.

```bash
claudeup upgrade              # Update all marketplaces and plugins
claudeup upgrade --check-only # Preview updates without applying
```

### outdated

Show available updates for the CLI, marketplaces, and plugins.

```bash
claudeup outdated  # List what has updates available
```

## Configuration

Configuration is stored in `~/.claudeup/`:

```text
~/.claudeup/
├── config.json       # Disabled plugins/servers, preferences
├── projects.json     # Local-scope project-to-profile mappings
├── profiles/         # Saved profiles
└── sandboxes/        # Persistent sandbox state
```

Project-level configuration files (created by `--scope project`):

```text
your-project/
├── .mcp.json         # Claude native MCP server config (auto-loaded)
└── .claudeup.json    # Plugins manifest (team runs `profile sync`)
```
