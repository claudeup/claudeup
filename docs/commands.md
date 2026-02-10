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

**Automatic plugin installation:**

After handling the existing installation, setup automatically installs any enabled plugins discovered in your configuration. This eliminates the need for a separate `claudeup profile apply` step.

```bash
claudeup setup -y  # Non-interactive: saves profile and installs plugins automatically
```

Plugin installation is non-blocking -- if individual plugins fail to install, setup continues and shows a summary:

```text
Installing 5 plugins...
  ✓ 4 plugins installed
  ⚠ 1 plugins failed
    • missing-plugin@unknown-marketplace: marketplace not found
Setup complete!
```

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
claudeup profile show <name>                 # Display profile contents (with scope labels)
claudeup profile current                     # Show active profile (with scope)
claudeup profile status [name]               # Show differences from current Claude state
claudeup profile diff <name>                 # Compare customized built-in to original
claudeup profile save [name]                 # Save current setup as profile (all scopes)
claudeup profile create <name>               # Create profile with interactive wizard
claudeup profile clone <name>                # Clone an existing profile
claudeup profile apply <name>                # Apply a profile (user scope)
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

**`profile list` flags:**

| Flag        | Description              |
| ----------- | ------------------------ |
| `--scope`   | Filter to specific scope |
| `--user`    | Show only user scope     |
| `--project` | Show only project scope  |
| `--local`   | Show only local scope    |

#### Project-Level Profiles

Apply profiles at project scope for team sharing:

```bash
# Apply profile to current project (creates .claude/settings.json)
claudeup profile apply frontend --project

# Apply profile locally only (not shared via git)
claudeup profile apply frontend --local
```

**Scope options:**

| Scope     | MCP Servers      | Plugins        | Shared?       |
| --------- | ---------------- | -------------- | ------------- |
| `user`    | `~/.claude.json` | user-scoped    | No            |
| `project` | `.mcp.json`      | project-scoped | Yes (via git) |
| `local`   | `~/.claude.json` | local-scoped   | No            |

**`profile apply` flags:**

| Flag               | Description                                                     |
| ------------------ | --------------------------------------------------------------- |
| `--user`           | Apply to user scope (~/.claude/) - default                      |
| `--project`        | Apply to project scope (.claude/settings.json)                  |
| `--local`          | Apply to local scope (.claude/settings.local.json)              |
| `--replace`        | Clear target scope before applying (replaces instead of adding) |
| `--setup`          | Force post-apply setup wizard to run                            |
| `--no-interactive` | Skip post-apply setup wizard (for CI/scripting)                 |
| `-f, --force`      | Force reapply even with unsaved changes                         |
| `--reinstall`      | Force reinstall all plugins and marketplaces                    |
| `--no-progress`    | Disable progress display (for CI/scripting)                     |
| `--dry-run`        | Show what would be changed without making modifications         |

**Replace mode:**

The `--replace` flag clears the target scope before applying the profile:

```bash
# Replace user scope with new profile (instead of merging)
claudeup profile apply backend-stack --replace

# Replace without prompts (for scripting)
claudeup profile apply backend-stack --replace -y
```

A backup is created automatically when using `--replace` (unless `-y` is used).
Backups are stored in `~/.claudeup/backups/`.

**Files created by `--project`:**

- `.claude/settings.json` - Project settings (plugins, MCP servers)
- `.mcp.json` - MCP servers (Claude auto-loads this)

**Team workflow:**

1. One team member applies the profile with `--project`
2. Commit `.claude/settings.json` and `.mcp.json` to version control
3. Other team members clone/pull and run `claudeup profile apply <name> --project`
4. MCP servers load automatically; plugins are configured by apply

**Scope precedence:**

When determining the active profile, `profile current` checks in order:

1. **Local scope** - Entry in `~/.claudeup/projects.json` for current directory
2. **User scope** - Global profile from `~/.claudeup/config.json`

Local scope takes precedence over user scope when you're in a project directory.

**Secrets and project scope:**

MCP servers often require secrets (API keys, tokens). When using `--project`:

- Secrets are **not** stored in `.mcp.json` - only secret references
- Each team member must have the referenced secrets available locally
- Common secret sources: environment variables, 1Password, system keychain
- Profile apply does not handle secrets - configure them separately

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

**`profile status [name]`** - Shows profile contents and activation status:

```bash
# Show status for active profile
claudeup profile status

# Show status for specific profile
claudeup profile status backend-stack
```

Use this to see:

- Which profile is active and at what scope
- Plugins, MCP servers, and marketplaces in the profile

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

| Scenario                                     | Command          |
| -------------------------------------------- | ---------------- |
| "What's in this profile?"                    | `profile status` |
| "What did I change from the original?"       | `profile diff`   |
| "Why does `profile list` show (customized)?" | `profile diff`   |

## Status & Discovery

### status

Overview of your Claude Code installation.

```bash
claudeup status           # Show status for all scopes
claudeup status --user    # Show only user scope
claudeup status --project # Filter to project scope
```

Shows marketplaces, plugin counts, MCP servers, and any detected issues.

**Flags:**

| Flag        | Description                              |
| ----------- | ---------------------------------------- |
| `--scope`   | Filter to scope: user, project, or local |
| `--user`    | Show only user scope                     |
| `--project` | Show only project scope                  |
| `--local`   | Show only local scope                    |

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

### mcp

Manage MCP servers.

```bash
claudeup mcp list                              # List all MCP servers
claudeup mcp disable <plugin>:<server>         # Disable specific server
claudeup mcp enable <plugin>:<server>          # Re-enable server
```

## Local Extensions

### local

Manage local Claude Code extensions (agents, commands, skills, hooks, rules, output-styles) from `~/.claudeup/local`.

```bash
claudeup local list                          # List all items and enabled status
claudeup local list agents                   # List items in a category
claudeup local list --enabled                # Show only enabled items
claudeup local list hooks --disabled         # Show only disabled hooks
claudeup local enable <category> <items...>  # Enable items (supports wildcards)
claudeup local disable <category> <items...> # Disable items (supports wildcards)
claudeup local view <category> <item>        # View item contents
claudeup local sync                          # Recreate symlinks from enabled.json
claudeup local import <category> <items...>  # Move items from active dir to local storage
claudeup local import-all [patterns...]      # Import items from all categories
claudeup local install <category> <path>     # Install items from an external path
```

**Categories:** `agents`, `commands`, `skills`, `hooks`, `rules`, `output-styles`

**`local list` flags:**

| Flag             | Description              |
| ---------------- | ------------------------ |
| `-e, --enabled`  | Show only enabled items  |
| `-d, --disabled` | Show only disabled items |

**Wildcard support (enable, disable, import):**

| Pattern | Description                       |
| ------- | --------------------------------- |
| `gsd-*` | Items starting with "gsd-"        |
| `gsd/*` | All items in the "gsd/" directory |
| `*`     | All items in the category         |

**Import commands:**

`import` moves items from active directories (`~/.claude/<category>/`) to local storage (`~/.claudeup/local/<category>/`) and creates symlinks back. Use when tools install directly to active directories.

`import-all` scans all categories at once. Without patterns, imports everything. With patterns, only matching items.

`install` copies items from an external path (git repos, downloads) to local storage and enables them.

## Event Tracking

### events

View file operation history.

```bash
claudeup events                              # Show recent events (last 20)
claudeup events --limit 50                   # Show last 50 events
claudeup events --file ~/.claude/settings.json
claudeup events --operation "profile apply"
claudeup events --user                       # User scope only
claudeup events --since 24h
```

**Flags:**

| Flag          | Description                                    |
| ------------- | ---------------------------------------------- |
| `--file`      | Filter by file path                            |
| `--operation` | Filter by operation name                       |
| `--scope`     | Filter by scope (user/project/local)           |
| `--user`      | Filter to user scope                           |
| `--project`   | Filter to project scope                        |
| `--local`     | Filter to local scope                          |
| `--since`     | Show events since duration (e.g., `24h`, `7d`) |
| `--limit`     | Maximum number of events to show (default: 20) |

### events diff

Show detailed changes for a file operation.

```bash
claudeup events diff --file ~/.claude/settings.json
claudeup events diff --file ~/.claude/plugins/installed_plugins.json --full
```

**Flags:**

| Flag     | Description                                     |
| -------- | ----------------------------------------------- |
| `--file` | File path to show diff for (required)           |
| `--full` | Show complete nested objects without truncation |

### events audit

Generate comprehensive audit trail with summary statistics.

```bash
claudeup events audit                        # Last 7 days, all scopes
claudeup events audit --user                 # User scope only
claudeup events audit --since 30d            # Last 30 days
claudeup events audit --since 2025-01-01     # Since specific date
claudeup events audit --format markdown > report.md
```

**Flags:**

| Flag          | Description                                                        |
| ------------- | ------------------------------------------------------------------ |
| `--scope`     | Filter by scope (user/project/local)                               |
| `--user`      | Filter to user scope                                               |
| `--project`   | Filter to project scope                                            |
| `--local`     | Filter to local scope                                              |
| `--operation` | Filter by operation name                                           |
| `--since`     | Duration (e.g., `7d`, `30d`) or date (`YYYY-MM-DD`); default: `7d` |
| `--format`    | Output format: `text` or `markdown` (default: `text`)              |

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
├── enabled.json      # Tracks which local items are enabled per category
├── projects.json     # Local-scope project-to-profile mappings
├── events/           # Operation event logs
├── local/            # Local storage for extensions
│   ├── agents/
│   ├── commands/
│   ├── hooks/
│   ├── output-styles/
│   ├── rules/
│   └── skills/
└── profiles/         # Saved profiles
```

Project-level configuration files (created by `--project`):

```text
your-project/
├── .claude/settings.json  # Project-scoped settings (plugins, etc.)
└── .mcp.json              # Claude native MCP server config (auto-loaded)
```
