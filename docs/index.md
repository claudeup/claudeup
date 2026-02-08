---
title: Documentation
---

# claudeup

Manage Claude Code plugins and profiles.

## Quick Start

```bash
# Install claudeup
curl -fsSL https://claudeup.github.io/install.sh | bash

# First-time setup
claudeup setup

# List available profiles
claudeup profile list

# Apply a profile
claudeup profile apply <name>
```

## Documentation

- [Command Reference](commands.html) - Complete list of all commands and flags
- [Profiles](profiles.html) - Save and switch between configurations
- [File Operations](file-operations.html) - Manage CLAUDE.md and settings.json
- [Team Workflows](team-workflows.html) - Share configurations across teams
- [Troubleshooting](troubleshooting.html) - Common issues and solutions

## Features

**Profiles** - Save your plugin and MCP server configurations, switch between them instantly. Profiles are for bootstrapping - apply once, then manage settings directly.

**Local Extensions** - Manage local agents, commands, skills, hooks, rules, and output-styles from `~/.claude/.library`. Import, install, enable, and disable with wildcard support.

**Event Tracking** - Audit trail of all file operations. View recent changes, diff configurations, and generate reports.

**File Operations** - Manage CLAUDE.md files and settings.json with merge, diff, and sync commands.

**Team Sharing** - Share profile definitions via `.claudeup/profiles/` in your project repository.
