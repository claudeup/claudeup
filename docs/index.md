---
title: Documentation
---

# claudeup

Manage Claude Code plugins, profiles, and sandboxes.

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
- [Sandbox](sandbox.html) - Run Claude Code in isolated Docker environments
- [Team Workflows](team-workflows.html) - Share configurations across teams
- [Troubleshooting](troubleshooting.html) - Common issues and solutions

## Features

**Profiles** - Save your plugin and MCP server configurations, switch between them instantly.

**File Operations** - Manage CLAUDE.md files and settings.json with merge, diff, and sync commands.

**Sandbox Mode** - Run Claude Code in isolated Docker containers for security and experimentation.

**Team Sharing** - Export and import configurations via `.claudeup.json` files.
