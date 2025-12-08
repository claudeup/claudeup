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

## Built-in Profiles

claudeup ships with built-in profiles that are ready to use without any setup:

### default

Minimal base configuration with essential marketplaces.

```bash
claudeup setup --profile default
```

**Marketplaces:**
- `anthropics/claude-code` - Official Anthropic plugins

**Use when:** Starting fresh or want a clean slate.

---

### frontend

Lean frontend development profile for Next.js, Tailwind CSS, and shadcn/ui projects.

```bash
claudeup setup --profile frontend
```

**Marketplaces:**
- `anthropics/claude-code` - Official Anthropic plugins
- `obra/superpowers-marketplace` - Productivity skills and workflows
- `malston/claude-code-templates` - Next.js/Vercel tooling

**Plugins:**
- `frontend-design@claude-code-plugins` - Distinctive UI/UX implementation
- `nextjs-vercel-pro@claude-code-templates` - Next.js scaffolding, components, Vercel deployment
- `superpowers@superpowers-marketplace` - TDD, debugging, collaboration patterns
- `episodic-memory@superpowers-marketplace` - Memory across sessions
- `commit-commands@claude-code-plugins` - Git workflow automation

**Auto-detects:** `next.config.*`, `tailwind.config.*`, `components.json`

**Use when:** Building Next.js apps with Tailwind and shadcn.

---

### frontend-full

Complete frontend development profile with E2E testing and performance tools.

```bash
claudeup setup --profile frontend-full
```

**Marketplaces:** Same as `frontend`

**Plugins:** Everything in `frontend`, plus:
- `testing-suite@claude-code-templates` - Playwright E2E testing (adds Playwright MCP)
- `performance-optimizer@claude-code-templates` - Bundle analysis, profiling
- `superpowers-chrome@superpowers-marketplace` - Chrome DevTools Protocol access
- `code-review@claude-code-plugins` - PR review automation

**Auto-detects:** Everything in `frontend`, plus `playwright.config.*`

**Use when:** Need comprehensive testing and performance tooling. Note: heavier token usage due to Playwright MCP.

---

Built-in profiles appear with `[built-in]` in the profile list:

```
$ claudeup profile list
Available profiles:

  default              Base Claude Code setup with essential marketplaces [built-in]
  frontend             Frontend development: Next.js, Tailwind, shadcn, Vercel [built-in]
  frontend-full        Complete frontend development with E2E testing... [built-in]
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
    "files": ["go.mod", "go.sum"],
    "contains": {"go.mod": "github.com/"}
  }
}
```

Detection uses OR-based matching within each category:

- `files`: Profile matches if **any** of these files exist
- `contains`: Profile matches if **any** file contains its pattern

Both categories must have at least one match if both are specified.

**Example:** The `frontend` profile matches if it finds `next.config.js` OR `tailwind.config.ts` OR `components.json` (any one is enough).

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
