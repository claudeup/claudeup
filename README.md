# claudeup

A CLI tool for managing Claude Code profiles and configurations.

**[Documentation](https://claudeup.github.io/claudeup/)**

## Install

```bash
# One-liner install (macOS/Linux)
curl -fsSL https://claudeup.github.io/install.sh | bash

# Or from source
go install github.com/claudeup/claudeup/v5/cmd/claudeup@latest
```

## Get Started

```bash
# Setup preserves existing Claude Code configuration
claudeup setup

# For fresh installations, apply a specific profile
claudeup setup --profile frontend
```

**For existing Claude Code users**: Setup detects your current installation and preserves it. You'll be offered the option to save your existing configuration as a profile. Enabled plugins are automatically installed -- no separate `profile apply` needed.

**For new users**: Setup applies the default profile (or your specified `--profile`) to get you started with plugins and MCP servers.

## Key Features

### Profiles

Save and switch between different Claude configurations. Great for different projects or sharing setups across machines.

```bash
claudeup profile list              # See available profiles
claudeup profile save my-setup     # Save current config as a profile
claudeup profile apply backend     # Switch to a different profile
```

Profiles include plugins, MCP servers, marketplaces, and secrets. [Learn more →](docs/profiles.md)

### Team Configuration

Share Claude configurations with your team via git:

```bash
# Team lead: Save profile and share via project settings
claudeup profile save team-config
claudeup profile apply team-config --project
git add .claude/settings.json && git commit -m "Add team profile"

# Team member: Apply after clone/pull
claudeup profile apply team-config --project
```

Profiles capture settings from all scopes (user, project, local). Use `profile apply --project` to write settings to `.claude/settings.json` for team sharing. [Learn more →](docs/team-workflows.md)

### Plugin Discovery

Find plugins by capability:

```bash
claudeup status                       # Overview of your installation
claudeup plugin search tdd            # Find plugins by capability
claudeup plugin search api --all      # Search all cached plugins
claudeup plugin browse my-marketplace # Browse marketplace plugins
```

[Full command reference →](docs/commands.md)

### Diagnostics & Maintenance

```bash
claudeup doctor     # Diagnose issues
claudeup cleanup    # Fix plugin path problems
claudeup outdated   # Check for updates
claudeup update     # Update claudeup CLI
claudeup upgrade    # Update plugins and marketplaces
```

[Troubleshooting guide →](docs/troubleshooting.md)

## Commands

### Update Commands

| Command             | Description                                               |
| ------------------- | --------------------------------------------------------- |
| `claudeup update`   | Update the claudeup CLI to the latest version             |
| `claudeup upgrade`  | Update marketplaces and plugins                           |
| `claudeup outdated` | Show available updates for CLI, marketplaces, and plugins |

For the complete command reference, see [Full command reference →](docs/commands.md)

## Environment Variables

| Variable            | Description                                    | Default       |
| ------------------- | ---------------------------------------------- | ------------- |
| `CLAUDEUP_HOME`     | Override claudeup's configuration directory    | `~/.claudeup` |
| `CLAUDE_CONFIG_DIR` | Override Claude Code's configuration directory | `~/.claude`   |

### Isolated Testing Environment

Use environment variables to run claudeup in isolation (useful for testing or CI):

```bash
export CLAUDEUP_HOME="/tmp/test-env/.claudeup"
export CLAUDE_CONFIG_DIR="/tmp/test-env/.claude"
claudeup profile apply base-tools
```

## Documentation

- [Profiles](docs/profiles.md) - Configuration profiles and secret management
- [Team Workflows](docs/team-workflows.md) - Sharing configurations via git
- [Commands](docs/commands.md) - Full command reference
- [Troubleshooting](docs/troubleshooting.md) - Common issues and fixes
- [Development](DEVELOPMENT.md) - Building, testing, and releasing

## License

MIT
