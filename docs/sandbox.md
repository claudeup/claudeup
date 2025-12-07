# Sandbox

Run Claude Code in an isolated Docker container for security. Protects your system from malicious or buggy plugins while still letting Claude work on your projects.

## Quick Start

```bash
# Ephemeral session - nothing persists after exit
claude-pm sandbox

# Persistent session - state saved between sessions
claude-pm sandbox --profile untrusted
```

## How It Works

The sandbox runs the entire Claude Code environment (CLI, plugins, MCP servers) inside a Docker container:

- Your current directory is mounted at `/workspace`
- Network access is enabled (for MCP servers, git, APIs)
- Secrets are injected from your profile configuration
- Interactive terminal is attached for normal Claude usage

### What's Isolated

The sandbox protects:
- Your home directory and dotfiles
- SSH keys and credentials (unless explicitly mounted)
- Other projects and files
- System files and configurations

### What's Accessible

The sandbox has access to:
- Network (for API calls, git operations)
- Your current working directory (mounted at `/workspace`)
- Secrets you explicitly configure in the profile

## Commands

```bash
# Basic usage
claude-pm sandbox                           # Ephemeral session
claude-pm sandbox --profile <name>          # Persistent with profile

# Mount control
claude-pm sandbox --no-mount                # No filesystem access
claude-pm sandbox --mount ~/data:/data      # Additional mount

# Secret control
claude-pm sandbox --secret EXTRA_KEY        # Add secret for this session
claude-pm sandbox --no-secret GITHUB_TOKEN  # Exclude a secret

# Utilities
claude-pm sandbox --shell                   # Drop to bash instead of Claude
claude-pm sandbox --clean --profile foo     # Reset sandbox state
```

## Profile Configuration

Add sandbox settings to your profile:

```json
{
  "name": "untrusted",
  "description": "Sandbox for testing untrusted plugins",
  "plugins": ["experimental-plugin@some-marketplace"],

  "sandbox": {
    "secrets": [
      "ANTHROPIC_API_KEY",
      "OPENAI_API_KEY"
    ],
    "mounts": [
      {"host": "~/.ssh/known_hosts", "container": "/root/.ssh/known_hosts", "readonly": true}
    ],
    "env": {
      "NODE_ENV": "development"
    }
  }
}
```

### Sandbox Fields

| Field | Description |
|-------|-------------|
| `secrets` | Secret names to resolve and inject (uses your configured secret backends) |
| `mounts` | Additional host paths to mount into the container |
| `env` | Static environment variables to set |

## Persistence

### Ephemeral Mode (default)

```bash
claude-pm sandbox
```

- Container state is discarded on exit
- Plugins must be reinstalled each session
- Maximum isolation

### Profile Mode

```bash
claude-pm sandbox --profile untrusted
```

- State saved to `~/.claude-pm/sandboxes/<profile>/`
- Plugins and configuration persist between sessions
- Each profile has its own isolated state

### Resetting State

```bash
claude-pm sandbox --clean --profile untrusted
```

Removes all persistent state for a profile's sandbox, returning it to a fresh state.

## Requirements

- Docker installed and running
- First run will pull the sandbox image from `ghcr.io/malston/claude-pm-sandbox`

## Security Model

The sandbox provides defense in depth:

1. **Filesystem isolation** - Only explicitly mounted paths are accessible
2. **Process isolation** - Container processes can't affect host
3. **Secret scoping** - Only configured secrets are available
4. **Ephemeral option** - No persistent state to be compromised

For maximum security when testing truly untrusted plugins:

```bash
cd $(mktemp -d)
claude-pm sandbox --no-mount
```

This runs with no filesystem access at all.
