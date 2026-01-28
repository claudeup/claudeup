---
title: Team Workflows
---

# Team Workflows

Share Claude Code configurations with your team by storing profiles in your project repository.

## Overview

claudeup supports two locations for profile storage:

| Location                | Purpose           | Shared                     |
| ----------------------- | ----------------- | -------------------------- |
| `~/.claudeup/profiles/` | Personal profiles | No (local to your machine) |
| `.claudeup/profiles/`   | Project profiles  | Yes (committed to git)     |

When loading a profile, claudeup checks the project directory first, then falls back to user profiles.

## Quick Start

**Team lead creates a shared profile:**

```bash
cd your-project

# Configure Claude with the plugins your team needs
claude plugin install tdd-workflows@claude-code-workflows --scope project
claude plugin install backend-development@claude-code-workflows --scope project

# Save current state as a profile (captures all scopes)
claudeup profile save team-config

# Apply at project scope to create .claudeup.json for team sharing
claudeup profile apply team-config --scope project

# Commit to git
git add .claudeup.json .claudeup/profiles/
git commit -m "Add team Claude profile"
git push
```

**Team member syncs after clone:**

```bash
git clone <repo-url>
cd your-project
claudeup profile sync
```

## Project Structure

After saving a profile to project scope, your repo will have:

```text
your-project/
├── .claudeup/
│   └── profiles/
│       └── team-config.json    # Shared profile definition
├── .claudeup.json              # Optional: project configuration
├── .claude/
│   └── settings.json           # Claude Code settings
└── src/
```

**What to commit:**

- `.claudeup/profiles/` - Profile definitions (commit this)
- `.claudeup.json` - Project configuration (commit this)
- `.claude/settings.json` - Project Claude settings (commit this)
- `.claude/settings.local.json` - Personal overrides (add to .gitignore)

## Workflows

### Creating a Team Profile

As a team lead, capture your current Claude configuration:

```bash
# Save current state as a profile (captures all scopes)
claudeup profile save backend-go

# Apply at project scope to create .claudeup.json for team sharing
claudeup profile apply backend-go --scope project
```

The profile includes:

- Marketplaces (sources for finding plugins)
- Installed plugins
- MCP server configurations
- Profile metadata

### Syncing Team Configuration

When joining a project or after pulling changes:

```bash
claudeup profile sync
```

Sync will:

1. Read `.claudeup.json` for the profile name
2. Find the profile in `.claudeup/profiles/` (project) or `~/.claudeup/profiles/` (user)
3. If profile not found, bootstrap from current state (captures all current settings as the profile)
4. Install any missing marketplaces
5. Install any missing plugins

**Bootstrap behavior:** If you have `.claudeup.json` but the profile definition doesn't exist (common when upgrading from older versions), sync creates the profile by capturing your current settings. This ensures sync always works.

### Viewing Profile Sources

See where each profile comes from:

```bash
claudeup profile list
```

Output:

```text
Your profiles (3)

  base-tools        Personal toolkit [user]
* team-config       Team configuration [project]
  frontend-dev      Frontend setup [project]
```

- `[user]` = from `~/.claudeup/profiles/`
- `[project]` = from `.claudeup/profiles/`
- `*` = currently active profile

### Layering User and Project Profiles

Combine personal preferences with team requirements:

```bash
# User scope: Your personal tools (available in all projects)
claudeup profile apply my-tools --scope user

# Project scope: Team requirements (this project only)
claudeup profile apply team-config --scope project
```

Both profiles are active simultaneously. Claude Code merges them with project settings taking precedence on conflicts.

**Example setup:**

```text
~/.claudeup/profiles/
└── my-tools.json           # Your personal: superpowers, writing tools

your-project/.claudeup/profiles/
└── backend-go.json         # Team: Go-specific plugins
```

## Complete Example

### Alice Sets Up a Go Project

```bash
cd my-go-api

# Install plugins for the team
claude plugin marketplace add superpowers-marketplace
claude plugin install tdd-workflows@claude-code-workflows --scope project
claude plugin install backend-development@claude-code-workflows --scope project

# Save and apply as project profile
claudeup profile save backend-go
claudeup profile apply backend-go --scope project

# Commit to git
git add .claudeup.json .claudeup/profiles/
git commit -m "Add Claude Code team profile"
git push
```

### Bob Joins the Project

```bash
git clone git@github.com:team/my-go-api.git
cd my-go-api

# Sync Claude configuration
claudeup profile sync
# Output:
#   Syncing profile: backend-go (from project)
#   ✓ Installing tdd-workflows@claude-code-workflows
#   ✓ Installing backend-development@claude-code-workflows
#   Synced: 2 plugins installed

# Ready to work
claude
```

### Alice Adds a Plugin Later

```bash
# Add new plugin
claude plugin install debugging-toolkit@claude-code-workflows --scope project

# Update the profile (re-save to capture new plugins)
claudeup profile save backend-go

# Share with team
git add .claudeup/profiles/ && git commit -m "Add debugging toolkit" && git push
```

### Bob Gets the Update

```bash
git pull
claudeup profile sync
# ✓ Installing debugging-toolkit@claude-code-workflows
```

## Resolution Order

When loading a profile by name:

1. Check `.claudeup/profiles/<name>.json` (project)
2. If not found, check `~/.claudeup/profiles/<name>.json` (user)

This means:

- Project profiles override user profiles of the same name
- Teams can customize shared profiles without affecting each member's personal setup
- No external dependencies - everything lives in your git repo

## Best Practices

### What to Put Where

**User profiles (`~/.claudeup/profiles/`):**

- Personal productivity tools
- Writing and style plugins
- Tools you use across all projects

**Project profiles (`.claudeup/profiles/`):**

- Language/framework specific plugins
- Security scanning tools
- Required team plugins
- Project-specific MCP servers

### Git Configuration

Add to your project's `.gitignore`:

```gitignore
# Claude Code local settings (personal overrides)
.claude/settings.local.json
```

Keep tracked:

- `.claudeup/profiles/` - Shared profile definitions
- `.claudeup.json` - Project configuration
- `.claude/settings.json` - Project-level Claude settings

## Related Documentation

- [Profiles](profiles.md) - Profile creation and management
- [Commands](commands.md) - Full command reference
