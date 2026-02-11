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
- [File Operations](file-operations.html) - Reference catalog of all files read and written by claudeup
- [Team Workflows](team-workflows.html) - Share configurations across teams
- [Troubleshooting](troubleshooting.html) - Common issues and solutions

## Features

**Profiles** - Save your plugin and MCP server configurations, switch between them instantly. Compose reusable profiles into stacks for multi-language or multi-tool setups. Profiles are for bootstrapping -- apply once, then manage settings directly.

**Local Extensions** - Manage local agents, commands, skills, hooks, rules, and output-styles from `~/.claudeup/local`. Import, install, enable, and disable with wildcard support.

**Event Tracking** - Audit trail of all file operations. View recent changes, diff configurations, and generate reports.

**File Operations** - Reference catalog of all files claudeup reads, writes, and modifies, organized by ownership.

**Team Sharing** - Share profile definitions via `.claudeup/profiles/` in your project repository.
