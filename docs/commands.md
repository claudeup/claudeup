---
title: Command Reference
---

# Command Reference

## Global Flags

| Flag              | Description                                                                      |
| ----------------- | -------------------------------------------------------------------------------- |
| `--claude-dir`    | Override Claude installation directory (default: `~/.claude`)                    |
| `--claudeup-home` | Override claudeup home directory; must be absolute path (default: `~/.claudeup`) |
| `-y, --yes`       | Skip interactive prompts, use defaults                                           |

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
claudeup profile status                      # Show effective configuration across all scopes
claudeup profile diff <name>                 # Compare customized built-in to original
claudeup profile save <name>                 # Save current setup as profile (all scopes)
claudeup profile create <name>               # Create profile with interactive wizard
claudeup profile clone <name>                # Clone an existing profile
claudeup profile apply <name>                # Apply a profile (user scope)
claudeup profile suggest                     # Suggest profile based on project files
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

**`profile status`** - Shows the live effective configuration by reading settings files directly:

```bash
claudeup profile status
```

Use this to see:

- Plugins grouped by scope (user, project, local)
- Enabled/disabled status
- Marketplace summary

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

#### Profile Suggest

Suggests a profile based on files in the current directory. Profiles with `detect` rules are matched against the project:

```bash
# In a Go project directory
claudeup profile suggest
# => Suggested profile: go-backend
#    Shared Go backend team configuration
#    Apply this profile? [Y/n]
```

Detection rules in a profile match by file existence and/or file content:

```json
{
  "name": "go-backend",
  "detect": {
    "files": ["go.mod"],
    "contains": { "go.mod": "module" }
  }
}
```

If multiple profiles match, the first is suggested. If none match, available profiles are listed.

## Status & Discovery

### status

Overview of your Claude Code installation.

```bash
claudeup status           # Show status for all scopes
claudeup status --user    # Show only user scope
claudeup status --project # Filter to project scope
```

Shows marketplaces, plugin counts, MCP servers, and any detected issues.

Without a scope flag, `status` shows all scopes. With a scope flag, only that scope's items are shown:

```bash
claudeup status           # All scopes combined (user + project + local)
claudeup status --user    # Only user-scope plugins, MCP servers, marketplaces
claudeup status --project # Only project-scope plugins for current directory
claudeup status --local   # Only local-scope plugins for current directory
```

This is useful when you want to understand what a specific scope contributes. For example, `--project` shows only what the team shares via git, while `--user` shows your personal configuration.

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
claudeup plugin list                                  # Table view (default)
claudeup plugin list --format detail                  # Verbose per-plugin details
claudeup plugin list --summary                        # Summary statistics only
claudeup plugin list --by-scope                       # Group enabled plugins by scope
claudeup plugin list --enabled                        # Show only enabled plugins
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
| `--format`   | Output format (table, detail)  |
| `--by-scope` | Group enabled plugins by scope |

**`plugin browse` flags:**

| Flag       | Description                        |
| ---------- | ---------------------------------- |
| `--format` | Output format (table)              |
| `--show`   | Show contents of a specific plugin |

**`plugin show`:**

Display the directory structure or file contents of a plugin. Without a file argument, shows the directory tree. With a file argument, displays the file contents (Markdown is rendered for the terminal).

```bash
# Show plugin directory tree
claudeup plugin show observability-monitoring@claude-code-workflows

# Show a specific file within a plugin
claudeup plugin show my-plugin@acme-marketplace agents/test

# Raw output (useful for piping to glow or bat)
claudeup plugin show my-plugin@acme-marketplace agents/test --raw

# While browsing
claudeup plugin browse claude-code-workflows --show observability-monitoring
```

| Flag    | Description                          |
| ------- | ------------------------------------ |
| `--raw` | Output raw content without rendering |

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

| Flag             | Description                                                                |
| ---------------- | -------------------------------------------------------------------------- |
| `--all`          | Search all cached plugins, not just installed                              |
| `--type`         | Filter by component type: skills, commands, agents                         |
| `--marketplace`  | Limit search to specific marketplace                                       |
| `--by-component` | Group results by component type instead of plugin                          |
| `--content`      | Search SKILL.md body content (not yet implemented; falls back to metadata) |
| `--regex`        | Treat query as regular expression                                          |
| `--format`       | Output format: json, table (default: styled text with trees)               |

**Output formats:**

- **Default** - Styled text showing matching plugins with their components and directory trees
- **Table** - Tabular view with plugin, type, component, and description columns
- **JSON** - Machine-readable output for scripting

## Extensions

### extensions

Manage Claude Code extensions (agents, commands, skills, hooks, rules, output-styles) from `~/.claudeup/ext`.

Alias: `ext`

```bash
claudeup extensions list                          # List all items and enabled status
claudeup extensions list agents                   # List items in a category
claudeup extensions list --enabled                # Show only enabled items
claudeup extensions list hooks --disabled         # Show only disabled hooks
claudeup extensions enable <category> <items...>  # Enable items (supports wildcards)
claudeup extensions disable <category> <items...> # Disable items (supports wildcards)
claudeup extensions view <category> <item>        # View item contents
claudeup extensions sync                          # Recreate symlinks from enabled.json
claudeup extensions import <category> <items...>  # Move items from active dir to storage
claudeup extensions import-all [patterns...]      # Import items from all categories
claudeup extensions install <category> <path>     # Install items from an external path
claudeup extensions uninstall <category> <items...> # Remove items from storage
```

**Categories:** `agents`, `commands`, `skills`, `hooks`, `rules`, `output-styles`

**`extensions list` flags:**

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

`import` moves items from active directories (`~/.claude/<category>/`) to storage (`~/.claudeup/ext/<category>/`) and creates symlinks back. Use when tools install directly to active directories.

`import-all` scans all categories at once. Without patterns, imports everything. With patterns, only matching items.

`install` copies items from an external path (git repos, downloads) to storage and enables them.

`uninstall` removes items from storage entirely -- disables the item, removes its symlink from `~/.claude/<category>/`, deletes the file from `~/.claudeup/ext/<category>/`, and removes the config entry. Supports the same wildcards as enable/disable.

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
claudeup upgrade              # Update marketplaces and plugins for current context
claudeup upgrade --all        # Update across all scopes and projects
claudeup upgrade --check-only # Preview updates without applying
```

By default, `upgrade` is scope-aware: it only processes user-scope plugins and plugins scoped to the current project directory. Use `--all` to upgrade plugins across all scopes and projects.

### outdated

Show available updates for the CLI, marketplaces, and plugins.

```bash
claudeup outdated        # Check plugins for current context
claudeup outdated --all  # Check across all scopes and projects
```

By default, `outdated` is scope-aware: it only checks user-scope plugins and plugins scoped to the current project directory. Use `--all` to check all plugins regardless of scope or project.

## Configuration

Configuration is stored in `~/.claudeup/`:

```text
~/.claudeup/
├── config.json       # Preferences
├── enabled.json      # Tracks which extensions are enabled per category
├── events/           # Operation event logs
├── ext/              # Storage for extensions
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
