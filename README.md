# claudeup

A CLI tool for managing Claude Code plugins, profiles, and sandboxed environments.

## Install

```bash
# One-liner install (macOS/Linux)
curl -fsSL https://claudeup.github.io/install.sh | bash

# Or from source
go install github.com/claudeup/claudeup/cmd/claudeup@latest
```

## Get Started

```bash
# First-time setup - installs Claude CLI and applies a profile
claudeup setup

# Or setup with a specific profile
claudeup setup --profile frontend
```

That's it. You now have a working Claude Code installation with your chosen plugins and MCP servers.

## Key Features

### Profiles

Save and switch between different Claude configurations. Great for different projects or sharing setups across machines.

```bash
claudeup profile list              # See available profiles
claudeup profile save my-setup     # Save current config as a profile
claudeup profile apply backend     # Switch to a different profile
```

Profiles include plugins, MCP servers, marketplaces, and secrets. [Learn more →](docs/profiles.md)

### Sandbox

Run Claude Code in an isolated Docker container for security.

```bash
claudeup sandbox                      # Ephemeral session
claudeup sandbox --profile untrusted  # Persistent sandboxed environment
```

Protects your system from untrusted plugins while still letting Claude work on your projects. [Learn more →](docs/sandbox.md)

### Plugin & MCP Management

Fine-grained control over what's enabled:

```bash
claudeup status                       # Overview of your installation
claudeup plugin disable name          # Disable a plugin
claudeup mcp disable plugin:server    # Disable just an MCP server
```

[Full command reference →](docs/commands.md)

### Diagnostics & Maintenance

```bash
claudeup doctor   # Diagnose issues
claudeup cleanup  # Fix plugin path problems
claudeup update   # Check for updates
```

[Troubleshooting guide →](docs/troubleshooting.md)

## Documentation

- [Profiles](docs/profiles.md) - Configuration profiles and secret management
- [Sandbox](docs/sandbox.md) - Running Claude in isolated containers
- [Commands](docs/commands.md) - Full command reference
- [Troubleshooting](docs/troubleshooting.md) - Common issues and fixes

## Development

```bash
git clone https://github.com/claudeup/claudeup.git
cd claudeup
go build -o bin/claudeup ./cmd/claudeup
cp bin/claudeup ~/.local/bin/claudeup
go test ./...
alias clup=claudeup
clup profile current
```

## License

MIT
