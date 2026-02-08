---
title: Profiles
---

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
claudeup profile current           # Show currently active profile
claudeup profile status [name]     # Show differences from current Claude state
claudeup profile diff <name>       # Compare customized built-in to original
claudeup profile save [name]       # Save current setup as a profile
claudeup profile create <name>     # Create a new profile with interactive wizard
claudeup profile clone <name>      # Clone an existing profile
claudeup profile apply <name>      # Apply a profile
claudeup profile apply             # Re-apply active profile
claudeup profile reset <name>      # Remove everything a profile installed
claudeup profile clean <plugin>    # Remove orphaned plugin from config
claudeup profile delete <name>     # Delete a custom user profile
claudeup profile restore <name>    # Restore a built-in profile to original state
claudeup profile rename <old> <new> # Rename a custom profile
claudeup profile suggest           # Get profile suggestion based on project
```

## Viewing Profiles

Use `profile show` to inspect a profile's contents:

```bash
claudeup profile show my-work
```

```text
Profile: my-work
Description: 3 marketplaces, 5 plugins, 1 MCP server

Plugins:
  - feature-dev@claude-plugins-official [user]
  - superpowers@superpowers-marketplace [user]
  - backend-dev@claude-code-workflows [project]

MCP Servers:
  - context7 (npx) [user]

Marketplaces:
  - anthropics/claude-plugins-official
  - obra/superpowers-marketplace

Local Items:
  Agents:
    - test-runner/test-runner.md
  Commands:
    - commit.md
  Skills:
    - golang
```

Multi-scope profiles show scope labels (`[user]`, `[project]`, `[local]`) next to each plugin and MCP server. Local items are grouped by category.

## Profile Scopes

Profiles can be applied at different scopes, allowing you to layer configurations:

```bash
# User scope (default) - Your personal configuration
claudeup profile apply my-defaults

# Project scope - Project-specific plugins (shared with team via git)
claudeup profile apply backend-project --project

# Local scope - Machine-specific plugins (not shared)
claudeup profile apply laptop-only --local
```

### How Scopes Work

Claude Code uses a **layered settings model** where all scopes are active simultaneously:

```text
┌─────────────────────────────────────┐
│ Local Scope (machine-specific)      │  ← Highest precedence
├─────────────────────────────────────┤
│ Project Scope (shared with team)    │
├─────────────────────────────────────┤
│ User Scope (personal defaults)      │  ← Base layer
└─────────────────────────────────────┘
```

**Key behaviors:**

1. **Settings accumulate** - Plugins from all scopes are enabled simultaneously
2. **Scopes add, not replace** - Applying a local profile ADDS to your user and project plugins
3. **Precedence for conflicts** - When the same plugin is configured in multiple scopes, more specific scope wins

### Accumulation Example

```bash
# User scope: 5 base plugins
claudeup profile apply base-tools

# Project scope: 3 project plugins
cd ~/my-project
claudeup profile apply backend-stack --project

# Local scope: 2 machine-specific plugins
claudeup profile apply docker-tools --local

# Result: All 10 plugins (5 + 3 + 2) are active
```

### Override Example

To disable a plugin from a lower scope, explicitly set it to `false`:

```json
// User scope (~/.claude/settings.json)
{
  "enabledPlugins": {
    "heavy-plugin@marketplace": true
  }
}

// Local scope (.claude/settings.local.json)
{
  "enabledPlugins": {
    "heavy-plugin@marketplace": false  // Disables it on this machine
  }
}
```

### Scope Storage

| Scope       | Location                        | Shared?         | Use Case                                     |
| ----------- | ------------------------------- | --------------- | -------------------------------------------- |
| **User**    | `~/.claude/settings.json`       | No              | Personal default plugins used everywhere     |
| **Project** | `.claude/settings.json`         | Yes (via git)   | Project-specific plugins, shared with team   |
| **Local**   | `./.claude/settings.local.json` | No (gitignored) | Machine-specific plugins, personal overrides |

### Project Scope Files

When you apply a profile with `--project`, these files are created:

```text
.claude/settings.json   # Project settings (plugins)
.mcp.json              # MCP server configuration (Claude native format)
```

**Recommended git workflow:**

```bash
# After applying project profile
git add .claude/settings.json .mcp.json
git commit -m "Add project-level Claude configuration"

# Team members apply after clone:
claudeup profile apply <name> --project
```

MCP servers from `.mcp.json` are loaded automatically by Claude Code.

### Local Scope Registry

Local scope profiles are tracked in `~/.claudeup/projects.json`:

```json
{
  "projects": {
    "/Users/you/projects/my-app": {
      "profile": "laptop-dev-tools",
      "lastUsed": "2025-01-15T10:30:00Z"
    }
  }
}
```

This allows claudeup to remember which local profile applies to each project directory.

### Viewing Active Configuration

To see which profile is active at each scope:

```bash
# Show current profile (checks local → project → user)
claudeup profile current

# Example output for project with local profile:
# Current profile: dev-tools (local scope)
#   Marketplaces: 2
#   Plugins: 8
```

### When to Use Each Scope

**User Scope** (default):

- Your personal base configuration
- Plugins you want everywhere
- One-time setup on new machines

**Project Scope** (`--project`):

- Project requires specific plugins
- Team needs shared configuration
- Plugin selection is project-dependent
- Files committed to git

**Local Scope** (`--local`):

- Machine-specific tools (e.g., Docker-related plugins only on machines with Docker)
- Personal overrides (disable heavy plugins on laptop)
- Experiment without affecting team
- Files NOT committed to git (in `.gitignore`)

### Common Patterns

**Pattern 1: Base + Project**

```bash
# Once: Set up personal defaults
claudeup profile apply my-base-tools

# Per-project: Add project plugins
cd ~/backend-api
claudeup profile apply backend-stack --project
```

**Pattern 2: Base + Project + Local Override**

```bash
# User scope: Base tools
claudeup profile apply base

# Project scope: Backend tools
cd ~/api-service
claudeup profile apply backend --project

# Local scope: Disable heavy plugins on laptop
claudeup profile apply lightweight --local
```

**Pattern 3: Team Collaboration**

```bash
# Maintainer: Set up project configuration
cd ~/team-project
claudeup profile apply team-stack --project
git add .claude/settings.json .mcp.json
git commit -m "Add Claude configuration"
git push

# Team member: Apply same profile after clone
git clone <repo>
cd team-project
claudeup profile apply team-stack --project
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
- `davila7/claude-code-templates` - Next.js/Vercel tooling

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

### hobson

Full access to the [wshobson/agents](https://github.com/wshobson/agents) plugin marketplace with an interactive category-based setup wizard.

```bash
claudeup setup --profile hobson
```

**Marketplaces:**

- `wshobson/agents` - Comprehensive plugin collection with 65+ plugins

**Plugins:** Selected during interactive setup wizard

**Categories available:**

- Core Development - workflows, debugging, docs, refactoring
- Quality & Testing - code review, testing, cleanup
- AI & Machine Learning - LLM dev, agents, MLOps
- Infrastructure & DevOps - K8s, cloud, CI/CD, monitoring
- Security & Compliance - scanning, compliance, API security
- Data & Databases - ETL, schema design, migrations
- Languages - Python, JS/TS, Go, Rust, etc.
- Business & Specialty - SEO, analytics, blockchain, gaming

**Setup wizard:** On first use, an interactive wizard guides you through selecting which categories to enable. Use `--setup` to re-run the wizard, or `--no-interactive` to skip it.

```bash
# Re-run setup wizard
claudeup profile apply hobson --setup

# Skip wizard (for CI/scripting)
claudeup profile apply hobson --no-interactive
```

**Use when:** Want access to a large plugin marketplace with guided setup.

---

Built-in and user profiles are grouped separately in the list:

```bash
$ claudeup profile list
Built-in profiles:

  default              Base Claude Code setup with essential marketplaces
  frontend             Frontend development: Next.js, Tailwind, shadcn, Vercel
  frontend-full        Complete frontend development with E2E testing...
  hobson               Full access to wshobson/agents with setup wizard

Your profiles:

* my-setup             Custom configuration for my workflow
  backend              Backend development profile
```

## Creating and Managing Profiles

### Saving Your Current Setup

Use `profile save` to capture your current Claude Code configuration as a profile:

```bash
# Save with auto-generated description
claudeup profile save my-work
# Description: "2 marketplaces, 5 plugins, 3 MCP servers"

# Save with custom description
claudeup profile save my-work --description "TAS development setup"

# Update existing profile (preserves custom description)
claudeup profile save my-work
```

**What gets captured:**

- Plugins and MCP servers from all scopes (user, project, local)
- Marketplaces referenced by at least one enabled plugin (unreferenced marketplaces are excluded)
- Local items (agents, commands, skills, hooks, rules, output styles) that exist in the active directory

**When re-saving an existing profile:**

- Local items are preserved from the existing profile (not re-scanned from the system)
- Marketplaces are always re-filtered based on current plugins
- Custom descriptions are preserved unless overridden with `--description`

**Auto-generated descriptions:**

- Profiles automatically get meaningful descriptions based on their contents
- Example: "2 marketplaces, 5 plugins" or "1 marketplace, 10 plugins, 2 MCP servers"
- Empty profiles show "Empty profile"

### Creating Profiles from Existing Ones

Use `profile clone` to copy an existing profile with a new name:

```bash
# Clone from specific profile
claudeup profile clone home-setup --from work

# Clone with custom description
claudeup profile clone home-setup --from work --description "Personal development"

# Clone from active profile (with -y flag)
claudeup profile clone backup -y
```

Like `profile save`, cloned profiles inherit the source's description (if custom) or get an auto-generated one (if the source had the old generic description).

### Creating New Profiles with Wizard

Use `profile create` for an interactive wizard that guides you through creating a new profile:

```bash
claudeup profile create my-new-profile
```

The wizard prompts you to select marketplaces, plugins, and configure MCP servers step by step.

### Creating Profiles Non-Interactively

For automation or scripting, use flags to create profiles without the wizard:

```bash
# Create profile with flags
claudeup profile create my-profile \
  --description "My development setup" \
  --marketplace "anthropics/claude-code-plugins" \
  --marketplace "obra/superpowers-marketplace" \
  --plugin "plugin-dev@claude-code-plugins"
```

You can also create profiles from JSON files or stdin:

```bash
# From file
claudeup profile create my-profile --from-file spec.json

# From stdin (useful for piping)
echo '{"description": "Piped profile", "marketplaces": ["owner/repo"]}' | \
  claudeup profile create my-profile --from-stdin

# Override description from file
claudeup profile create my-profile --from-file spec.json --description "Custom description"
```

The JSON format supports both shorthand and full marketplace syntax:

```json
{
  "description": "Example profile",
  "marketplaces": ["anthropics/claude-code-plugins"],
  "plugins": ["plugin-dev@claude-code-plugins"],
  "mcpServers": [],
  "detect": {}
}
```

## Profile Structure

Profiles are stored in `~/.claudeup/profiles/` as JSON files.

### Multi-Scope Format (v3+)

Profiles capture settings from all scopes (user, project, local) using the `perScope` structure:

```json
{
  "name": "team-backend",
  "description": "Backend development profile",
  "marketplaces": [
    { "source": "github", "repo": "anthropics/claude-code-plugins" }
  ],
  "perScope": {
    "user": {
      "plugins": ["superpowers@superpowers-marketplace"],
      "mcpServers": []
    },
    "project": {
      "plugins": ["backend-development@claude-code-workflows"],
      "mcpServers": []
    }
  },
  "localItems": {
    "agents": ["test-runner/test-runner.md"],
    "commands": ["commit.md"],
    "skills": ["golang"]
  }
}
```

When you run `profile save`, all three scopes are captured automatically. When you run `profile apply`, settings are restored to the correct scope.

**Marketplace filtering:** Only marketplaces referenced by at least one enabled plugin are included. Marketplaces installed by other tools (e.g., mpm) that have no corresponding plugins in the profile are excluded.

**Local items:** Enabled local items (agents, commands, skills, hooks, rules, output styles) are captured from the active directory. When re-saving an existing profile, local items are preserved from the original to prevent accumulation of items enabled by other tools.

### Legacy Format (backward compatible)

Older profiles with flat `plugins` arrays are still supported and treated as user-scope:

```json
{
  "name": "frontend",
  "description": "1 marketplace, 2 plugins, 1 MCP server",
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
    { "source": "github", "repo": "anthropics/claude-code-plugins" }
  ],
  "detect": {
    "files": ["package.json", "tsconfig.json"],
    "contains": { "package.json": "react" }
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
            { "type": "env", "key": "MY_API_KEY" },
            { "type": "1password", "ref": "op://Private/My API/credential" },
            { "type": "keychain", "service": "my-api", "account": "default" }
          ]
        }
      }
    }
  ]
}
```

### Secret Backends

| Backend     | Platform | Requirement                      |
| ----------- | -------- | -------------------------------- |
| `env`       | All      | Environment variable set         |
| `1password` | All      | `op` CLI installed and signed in |
| `keychain`  | macOS    | Keychain item exists             |

Resolution tries each source in order. First success wins.

## Project Detection

The `detect` field enables automatic profile suggestion based on project files:

```json
{
  "detect": {
    "files": ["go.mod", "go.sum"],
    "contains": { "go.mod": "github.com/" }
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

## Post-Apply Hooks

Profiles can include hooks that run after the profile is applied. This enables interactive setup wizards, custom configuration, and automation.

```json
{
  "postApply": {
    "script": "setup.sh",
    "condition": "first-run"
  }
}
```

### Hook Fields

| Field       | Description                                                                                                          |
| ----------- | -------------------------------------------------------------------------------------------------------------------- |
| `script`    | Path to a bash script (relative to profile). Takes precedence over `command`.                                        |
| `command`   | Direct bash command to run (used if `script` is not set).                                                            |
| `condition` | When to run: `"always"` (default) or `"first-run"` (only if no plugins from the profile's marketplaces are enabled). |

### Hook Flags

```bash
# Force the hook to run even if first-run detection would skip it
claudeup profile apply myprofile --setup

# Skip the hook entirely (for CI/scripting)
claudeup profile apply myprofile --no-interactive
```

### Security Considerations

**Hooks execute arbitrary shell commands.** Only use profiles from trusted sources.

- **Built-in profiles** (like `hobson`, `frontend`, `default`) are safe - they're embedded in the claudeup binary and reviewed by maintainers.
- **User-created profiles** with hooks should only be shared with or used by people who trust the source.
- **Downloaded profiles** from unknown sources could contain malicious hooks. Review the profile JSON before applying.

When applying a profile with hooks, claudeup does not prompt for confirmation. If you're unsure about a profile's contents, use `claudeup profile show <name>` to inspect it first.

## Resetting Profiles

Use `profile reset` to remove everything a profile installed:

```bash
# Remove all plugins, MCP servers, and marketplaces from a profile
claudeup profile reset hobson
```

This removes:

- All plugins installed from the profile's marketplaces
- All MCP servers defined in the profile
- All marketplaces added by the profile

**Use cases:**

- Testing a profile's setup wizard from scratch
- Cleaning up before switching to a different profile
- Removing a profile's effects without applying a new one

The reset command shows what will be removed and prompts for confirmation:

```text
Reset profile: hobson

  Will remove:
    - Plugin: debugging-toolkit@wshobson-agents
    - Plugin: code-review-ai@wshobson-agents
    - Marketplace: wshobson/agents

Proceed? [y]:
```

## Deleting, Renaming, and Restoring Profiles

### Deleting Custom Profiles

Use `profile delete` to permanently remove custom profiles you've created:

```bash
claudeup profile delete my-workflow
```

This command only works on custom profiles. Attempting to delete a built-in profile returns an error with guidance to use `restore` instead.

### Renaming Custom Profiles

Use `profile rename` to rename custom profiles:

```bash
claudeup profile rename old-name new-name
```

This only works on custom profiles you've created. Built-in profiles cannot be renamed.

If the profile being renamed is currently active, the active profile config is updated to point to the new name.

### Restoring Built-in Profiles

Use `profile restore` to remove your customizations from a built-in profile:

```bash
claudeup profile restore frontend
```

This removes your customization file, immediately revealing the original built-in version. The profile list shows customized built-ins with "(customized)" - use `restore` to revert them.

### Understanding Built-in vs Custom Profiles

**Built-in profiles** (like `default`, `frontend`, `hobson`) are embedded in the claudeup binary. They always exist and cannot be deleted.

When you modify a built-in profile (e.g., by saving over it), a custom file is created in `~/.claudeup/profiles/` that shadows the built-in.

| Profile Type        | Delete                 | Restore                      |
| ------------------- | ---------------------- | ---------------------------- |
| Custom profile      | ✓ Permanently removes  | ✗ Error (not built-in)       |
| Customized built-in | ✗ Error (use restore)  | ✓ Removes customizations     |
| Unmodified built-in | ✗ Error (can't delete) | ✗ Error (nothing to restore) |

### Reset vs Delete vs Restore

These commands serve different purposes:

| Command           | What it does                                               |
| ----------------- | ---------------------------------------------------------- |
| `profile reset`   | Uninstalls components (plugins, MCP servers, marketplaces) |
| `profile delete`  | Permanently removes a custom profile file                  |
| `profile restore` | Removes customizations from a built-in profile             |

**To fully restore a customized built-in profile:**

```bash
# 1. Remove installed components
claudeup profile reset frontend

# 2. Remove your customizations (restores original definition)
claudeup profile restore frontend

# 3. Reinstall from the original built-in
claudeup profile apply frontend
```

**Note:** If you only want to restore the profile definition without changing what's installed, just run `profile restore`.
