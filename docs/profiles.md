# Profiles

Profiles are saved configurations of plugins, MCP servers, and marketplaces. Use them to:

- Save your current setup for later
- Switch between different configurations (e.g., frontend vs backend work)
- Share configurations between machines
- Quickly set up new installations

## Commands

```bash
claudeup profile list              # List available profiles
claudeup profile show <name>       # Show profile contents
claudeup profile create <name>     # Save current setup as a profile
claudeup profile use <name>        # Apply a profile (replaces current config)
claudeup profile suggest           # Get profile suggestion based on project
```

## Profile Structure

Profiles are stored in `~/.claudeup/profiles/` as JSON files:

```json
{
  "name": "frontend",
  "description": "Frontend development with React tooling",
  "plugins": [
    "superpowers@superpowers-marketplace",
    "frontend-design@claude-code-plugins"
  ],
  "mcpServers": [
    {
      "name": "context7",
      "command": "npx",
      "args": ["-y", "@context7/mcp"],
      "scope": "user"
    }
  ],
  "marketplaces": [
    {"source": "github", "repo": "anthropics/claude-code-plugins"}
  ],
  "detect": {
    "files": ["package.json", "tsconfig.json"],
    "contains": {"package.json": "react"}
  }
}
```

## Secret Management

MCP servers often need API keys. Profiles support multiple secret backends that are tried in order:

```json
{
  "mcpServers": [
    {
      "name": "my-api",
      "command": "npx",
      "args": ["-y", "my-mcp-server"],
      "secrets": {
        "API_KEY": {
          "description": "API key for the service",
          "sources": [
            {"type": "env", "key": "MY_API_KEY"},
            {"type": "1password", "ref": "op://Private/My API/credential"},
            {"type": "keychain", "service": "my-api", "account": "default"}
          ]
        }
      }
    }
  ]
}
```

### Secret Backends

| Backend | Platform | Requirement |
|---------|----------|-------------|
| `env` | All | Environment variable set |
| `1password` | All | `op` CLI installed and signed in |
| `keychain` | macOS | Keychain item exists |

Resolution tries each source in order. First success wins.

## Project Detection

The `detect` field enables automatic profile suggestion based on project files:

```json
{
  "detect": {
    "files": ["go.mod"],
    "contains": {"go.mod": "github.com/"}
  }
}
```

- `files`: Profile matches if these files exist
- `contains`: Profile matches if files contain these strings

Run `claudeup profile suggest` in a project directory to get a recommendation.

## Setup Integration

The `claudeup setup` command uses profiles:

```bash
# Setup with default profile
claudeup setup

# Setup with specific profile
claudeup setup --profile backend
```

If an existing Claude installation is detected, setup offers to save it as a profile before applying the new one.

## Sandbox Integration

Profiles can include sandbox-specific settings. See [Sandbox documentation](sandbox.md) for details.
