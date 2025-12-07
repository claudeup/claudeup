# claude-pm

A CLI tool for managing Claude Code plugins, profiles, and sandboxed environments.

## Install

```bash
# From source
go install github.com/malston/claude-pm/cmd/claude-pm@latest

# Or download from releases
# https://github.com/malston/claude-pm/releases
```

## Get Started

```bash
# First-time setup - installs Claude CLI and applies a profile
claude-pm setup

# Or setup with a specific profile
claude-pm setup --profile frontend
```

That's it. You now have a working Claude Code installation with your chosen plugins and MCP servers.

## Key Features

### Profiles

Save and switch between different Claude configurations. Great for different projects or sharing setups across machines.

```bash
claude-pm profile list              # See available profiles
claude-pm profile create my-setup   # Save current config as a profile
claude-pm profile use backend       # Switch to a different profile
```

Profiles include plugins, MCP servers, marketplaces, and secrets. [Learn more →](docs/profiles.md)

### Sandbox

Run Claude Code in an isolated Docker container for security.

```bash
claude-pm sandbox                      # Ephemeral session
claude-pm sandbox --profile untrusted  # Persistent sandboxed environment
```

Protects your system from untrusted plugins while still letting Claude work on your projects. [Learn more →](docs/sandbox.md)

### Plugin & MCP Management

Fine-grained control over what's enabled:

```bash
claude-pm status                    # Overview of your installation
claude-pm disable plugin@marketplace   # Disable a plugin
claude-pm mcp disable plugin:server    # Disable just an MCP server
```

[Full command reference →](docs/commands.md)

### Diagnostics & Maintenance

```bash
claude-pm doctor   # Diagnose issues
claude-pm cleanup  # Fix plugin path problems
claude-pm update   # Check for updates
```

[Troubleshooting guide →](docs/troubleshooting.md)

## Documentation

- [Profiles](docs/profiles.md) - Configuration profiles and secret management
- [Sandbox](docs/sandbox.md) - Running Claude in isolated containers
- [Commands](docs/commands.md) - Full command reference
- [Troubleshooting](docs/troubleshooting.md) - Common issues and fixes

## Development

```bash
git clone https://github.com/malston/claude-pm.git
cd claude-pm
go build -o bin/claude-pm ./cmd/claude-pm
go test ./...
```

## License

MIT
