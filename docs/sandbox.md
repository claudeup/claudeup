# Sandbox

Run Claude Code in an isolated Docker container for security. Protects your system from malicious or buggy plugins while still letting Claude work on your projects.

## Quick Start

```bash
# Ephemeral session - nothing persists after exit
claudeup sandbox

# Persistent session - state saved between sessions
claudeup sandbox --profile untrusted
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
claudeup sandbox                           # Ephemeral session
claudeup sandbox --profile <name>          # Persistent with profile

# Mount control
claudeup sandbox --no-mount                # No filesystem access
claudeup sandbox --mount ~/data:/data      # Additional mount

# Secret control
claudeup sandbox --secret EXTRA_KEY        # Add secret for this session
claudeup sandbox --no-secret GITHUB_TOKEN  # Exclude a secret

# Utilities
claudeup sandbox --shell                   # Drop to bash instead of Claude
claudeup sandbox --clean --profile foo     # Reset sandbox state

# Advanced options
claudeup sandbox --ephemeral               # Force ephemeral mode (no persistence)
claudeup sandbox --image my-image:latest   # Use custom Docker image
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--profile` | Profile for persistent state |
| `--copy-auth` | Copy authentication from ~/.claude.json |
| `--mount` | Additional mounts (host:container[:ro]) |
| `--no-mount` | Don't mount working directory |
| `--secret` | Additional secrets to inject |
| `--no-secret` | Secrets to exclude |
| `--shell` | Drop to bash instead of Claude CLI |
| `--clean` | Reset sandbox state for profile |
| `--ephemeral` | Force ephemeral mode even with --profile |
| `--image` | Override sandbox Docker image |

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
claudeup sandbox
```

- Container state is discarded on exit
- Plugins must be reinstalled each session
- Maximum isolation

### Profile Mode

```bash
claudeup sandbox --profile untrusted
```

- State saved to `~/.claudeup/sandboxes/<profile>/`
- Plugins and configuration persist between sessions
- Each profile has its own isolated state

### Resetting State

```bash
claudeup sandbox --clean --profile untrusted
```

Removes all persistent state for a profile's sandbox, returning it to a fresh state.

## Authentication

By default, sandboxes require interactive authentication on first run (API key entry and workspace trust confirmation). You can bypass this by copying your existing authentication:

```bash
# Copy authentication for this session only
claudeup sandbox --profile untrusted --copy-auth

# Or enable it globally in ~/.claudeup/config.json
{
  "sandbox": {
    "copyAuth": true
  }
}
```

**How it works:**

- Copies `~/.claude.json` from your local machine to the sandbox state directory
- Includes OAuth credentials and workspace trust decisions
- Only works with profile-based sandboxes (not ephemeral)
- Subsequent runs of the same profile use the copied authentication

**Security considerations:**

- Your authentication credentials are copied into the sandbox state directory
- Anyone with access to `~/.claudeup/sandboxes/<profile>/` can see these credentials
- The credentials work across machines/containers
- Only enable `copyAuth` if you trust the sandbox environment

**File permissions:**

- State directory: `0700` (owner-only access)
- Auth file: `0600` (owner-only read/write)

### Build the local docker image and use it for a shell

```bash
# 1. Build the local image:
./scripts/build-sandbox-image.sh
```

```bash
# 1. Launch a shell into the local image
claudeup sandbox --image ghcr.io/claudeup/claudeup-sandbox:local --mount ~/.claude:/claude --shell
```

## Requirements

- Docker installed and running
- First run will pull the sandbox image from `ghcr.io/claudeup/claudeup-sandbox`

## Security Model

The sandbox provides defense in depth:

1. **Filesystem isolation** - Only explicitly mounted paths are accessible
2. **Process isolation** - Container processes can't affect host
3. **Secret scoping** - Only configured secrets are available
4. **Ephemeral option** - No persistent state to be compromised

For maximum security when testing truly untrusted plugins:

```bash
cd $(mktemp -d)
claudeup sandbox --no-mount
```

This runs with no filesystem access at all.
