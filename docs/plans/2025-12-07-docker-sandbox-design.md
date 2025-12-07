# Docker Sandbox Feature Design

## Overview

`claude-pm sandbox` launches Claude Code inside a Docker container with TTY passthrough. The sandbox isolates Claude from the host system while still allowing it to work on the current project.

## Goals

**Primary:** Security isolation - defense in depth against both malicious plugins and accidental harm from plugin bugs, including MCP server isolation.

**Secondary:** Environment consistency - reproducible Claude setups across machines.

## Design Decisions

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Scope | Entire Claude environment sandboxed | Simpler than sandboxing individual plugins; no complex IPC needed |
| Interaction | TTY passthrough | Matches current Claude CLI UX |
| Persistence | Ephemeral by default, profile-based persistence optional | Flexibility without complexity |
| Filesystem | Working directory mounted at `/workspace` | Users still need Claude to work on their projects |
| Secrets | Profile-defined with CLI override | Layered approach: sensible defaults with escape hatches |
| Image | Minimal base for v1 | YAGNI; profile-defined images can come later |
| Extensibility | Docker-specific now | VMs/jumpboxes not on near-term roadmap; refactor when needed |

## CLI UX

```bash
# Ephemeral session - starts fresh, nothing persists
claude-pm sandbox

# Profile-based session - state persists in profile's sandbox directory
claude-pm sandbox --profile untrusted

# Mount control
claude-pm sandbox --no-mount              # No filesystem access
claude-pm sandbox --mount ~/other:/data   # Additional mount

# Secret control (overrides profile)
claude-pm sandbox --secret EXTRA_KEY      # Add secret
claude-pm sandbox --no-secret GITHUB_TOKEN  # Exclude secret

# Utility
claude-pm sandbox --shell                 # Drop to bash instead of Claude CLI
claude-pm sandbox --clean --profile foo   # Reset a profile's sandbox state
```

## Profile Schema Extension

```json
{
  "name": "untrusted",
  "description": "Sandboxed environment for testing untrusted plugins",
  "mcpServers": [...],
  "plugins": [...],

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

**Field Semantics:**

- `sandbox.secrets`: Secret names to resolve and inject (uses existing 1Password/Keychain/env resolution)
- `sandbox.mounts`: Additional persistent mounts beyond the working directory
- `sandbox.env`: Static environment variables

Profiles without a `sandbox` field work normally on host; if used with `claude-pm sandbox --profile`, they get default sandbox behavior (no extra secrets, no extra mounts).

## Container Architecture

### Base Image

```dockerfile
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Claude CLI
RUN curl -fsSL https://claude.ai/install.sh | sh

# Install claude-pm
COPY claude-pm /usr/local/bin/claude-pm

WORKDIR /workspace
ENTRYPOINT ["claude"]
```

### Image Distribution

- Published to GitHub Container Registry: `ghcr.io/malston/claude-pm-sandbox:latest`
- Tagged by claude-pm version: `ghcr.io/malston/claude-pm-sandbox:v1.2.3`
- Built in CI alongside existing release workflow

### Runtime Container Setup

```bash
docker run -it --rm \
  -v "$(pwd):/workspace" \
  -v "$SANDBOX_STATE:/root/.claude" \
  -e "ANTHROPIC_API_KEY=..." \
  -e "OPENAI_API_KEY=..." \
  --network bridge \
  ghcr.io/malston/claude-pm-sandbox:latest
```

### Sandbox State Location

- **Ephemeral:** No volume mount for `~/.claude`; container uses tmpfs
- **Profile-based:** `~/.claude-pm/sandboxes/<profile-name>/` mounted to `/root/.claude`

## Implementation Structure

### New Package

```
internal/
├── sandbox/
│   ├── sandbox.go       # Core sandbox orchestration
│   ├── docker.go        # Docker-specific implementation
│   ├── secrets.go       # Secret resolution for sandbox injection
│   └── state.go         # Sandbox state directory management
├── commands/
│   └── sandbox.go       # New CLI command
```

### Key Components

**`sandbox.Manager`** - Orchestrates sandbox lifecycle:
- `Start(profile *Profile, opts Options) error` - Launch sandbox session
- `ListRunning() []Container` - Show active sandboxes
- `Clean(profile string) error` - Reset a profile's sandbox state

**`sandbox.DockerRunner`** - Docker-specific logic:
- Image pull/verification
- Container creation with correct mounts, env, network
- TTY attachment and signal forwarding
- Cleanup on exit

### Build Order

1. Dockerfile + CI workflow to publish image
2. `sandbox` package with Docker runner
3. `sandbox` command wired into CLI
4. Profile schema extension for sandbox fields
5. Secret injection integration

## Not in v1

- Profile-defined custom base images
- VM/jumpbox sandbox backends
- Background daemon mode
- Fine-grained per-plugin sandboxing

## Security Model

The sandbox provides isolation with intentional access:

**Isolated from:**
- Host home directory
- SSH keys (unless explicitly mounted)
- Other projects
- System files
- Host secrets (unless explicitly passed)

**Has access to:**
- Network (for MCP servers, git, APIs)
- Current working directory (mounted at `/workspace`)
- Secrets defined in profile or passed via CLI

For maximum isolation, use `--no-mount` and run from an empty directory.
