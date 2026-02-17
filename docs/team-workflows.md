---
title: Team Workflows
---

# Team Workflows

Share Claude Code configurations with your team by applying profiles at project scope.

## Overview

claudeup profiles capture your Claude Code setup (plugins, MCP servers, settings) and apply them to projects. When applied at project scope, the resulting configuration files can be committed to git for team sharing.

Profiles themselves are stored in your user directory at `~/.claudeup/profiles/` and are not committed to the project repository. Only the resulting configuration files are shared.

> **Note:** If your project already has a `.claudeup/profiles/` directory from an earlier version of claudeup, those profiles are still recognized but this path is no longer written to.

## Quick Start

**Team lead creates a shared configuration:**

```bash
cd your-project

# Configure Claude with the plugins your team needs
claude plugin install tdd-workflows@claude-code-workflows --project
claude plugin install backend-development@claude-code-workflows --project

# Save current state as a profile (captures all scopes)
claudeup profile save team-config

# Apply at project scope (creates committable config files)
claudeup profile apply team-config --project

# Commit configuration to git
git add .claude/settings.json .mcp.json
git commit -m "Add team Claude configuration"
git push
```

**Team member gets configuration after clone:**

```bash
git clone <repo-url>
cd your-project
# Settings are already in .claude/settings.json from git
# Claude Code picks them up automatically
```

If a team member also has the profile (e.g., shared via dotfiles), they can re-apply it to reinstall plugins:

```bash
claudeup profile apply team-config --project
```

## Project Structure

After applying a profile at project scope, your repo will have:

```text
your-project/
├── .claude/
│   └── settings.json           # Claude Code settings (plugins)
├── .mcp.json                   # MCP server configuration
└── src/
```

**What to commit:**

- `.claude/settings.json` - Project Claude settings (commit this)
- `.mcp.json` - MCP server configuration (commit this)
- `.claude/settings.local.json` - Personal overrides (add to .gitignore)

## Workflows

### Creating a Team Profile

As a team lead, capture your current Claude configuration:

```bash
# Save current state as a profile (captures all scopes)
claudeup profile save backend-go

# Apply at project scope
claudeup profile apply backend-go --project
```

The profile includes:

- Marketplaces (sources for finding plugins)
- Installed plugins
- MCP server configurations
- Profile metadata

### Applying Team Configuration

When joining a project, apply the team profile:

```bash
claudeup profile apply backend-go --project
```

This installs any missing marketplaces and plugins defined in the profile.

**Philosophy:** Profiles are for bootstrapping -- apply once, then manage settings directly. After initial setup, team members can customize their local scope without affecting others.

### Viewing Available Profiles

See which profiles are available:

```bash
claudeup profile list
```

To see what's actually running across all scopes:

```bash
claudeup profile status
```

### Layering User and Project Profiles

Combine personal preferences with team requirements:

```bash
# User scope: Your personal tools (available in all projects)
claudeup profile apply my-tools --user

# Project scope: Team requirements (this project only)
claudeup profile apply team-config --project
```

Both profiles are active simultaneously. Claude Code merges them with project settings taking precedence on conflicts.

## Complete Example

### Alice Sets Up a Go Project

```bash
cd my-go-api

# Install plugins for the team
claude plugin marketplace add superpowers-marketplace
claude plugin install tdd-workflows@claude-code-workflows --project
claude plugin install backend-development@claude-code-workflows --project

# Save and apply as project profile
claudeup profile save backend-go
claudeup profile apply backend-go --project

# Commit configuration to git
git add .claude/settings.json .mcp.json
git commit -m "Add Claude Code team configuration"
git push
```

### Bob Joins the Project

```bash
git clone git@github.com:team/my-go-api.git
cd my-go-api

# Apply the team profile
claudeup profile apply backend-go --project
# Output:
#   Applying profile: backend-go
#   ✓ Installing tdd-workflows@claude-code-workflows
#   ✓ Installing backend-development@claude-code-workflows
#   Applied: 2 plugins installed

# Ready to work
claude
```

### Alice Adds a Plugin Later

```bash
# Add new plugin directly (profiles are for bootstrapping)
claude plugin install debugging-toolkit@claude-code-workflows --project

# Optionally update the profile for future team members
claudeup profile save backend-go

# Commit changes
git add .claude/settings.json
git commit -m "Add debugging toolkit"
git push
```

### Bob Gets the Update

```bash
git pull
# Settings updated automatically via git
# Plugin is now in .claude/settings.json
```

**Note:** After initial bootstrap, team members get plugin changes through git. The profile is primarily for onboarding new team members.

## Best Practices

### What to Put Where

**User profiles (`~/.claudeup/profiles/`):**

- Personal productivity tools
- Writing and style plugins
- Tools you use across all projects

**Project configuration (committed to git):**

- Language/framework specific plugins in `.claude/settings.json`
- Project-specific MCP servers in `.mcp.json`
- Shared via `profile apply --project`

### Git Configuration

Add to your project's `.gitignore`:

```gitignore
# Claude Code local settings (personal overrides)
.claude/settings.local.json
```

Keep tracked:

- `.claude/settings.json` - Project-level Claude settings
- `.mcp.json` - MCP server configuration

## Related Documentation

- [Profiles](profiles.md) - Profile creation and management
- [Commands](commands.md) - Full command reference
