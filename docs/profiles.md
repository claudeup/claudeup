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
claudeup profile save [name]       # Save current setup as a profile
claudeup profile create <name>     # Create a profile by copying an existing one
claudeup profile use <name>        # Apply a profile
claudeup profile reset <name>      # Remove everything a profile installed
claudeup profile delete <name>     # Delete a custom user profile
claudeup profile restore <name>    # Restore a built-in profile to original state
claudeup profile rename <old> <new> # Rename a custom profile
claudeup profile suggest           # Get profile suggestion based on project
```

## Profile Scopes

Profiles can be applied at different scopes, allowing you to layer configurations:

```bash
# User scope (default) - Your personal configuration
claudeup profile use my-defaults

# Project scope - Project-specific plugins (shared with team via git)
claudeup profile use backend-project --scope project

# Local scope - Machine-specific plugins (not shared)
claudeup profile use laptop-only --scope local
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
claudeup profile use base-tools

# Project scope: 3 project plugins
cd ~/my-project
claudeup profile use backend-stack --scope project

# Local scope: 2 machine-specific plugins
claudeup profile use docker-tools --scope local

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

| Scope | Location | Shared? | Use Case |
|-------|----------|---------|----------|
| **User** | `~/.claude/settings.json` | No | Personal default plugins used everywhere |
| **Project** | `.claude/settings.json` | Yes (via git) | Project-specific plugins, shared with team |
| **Local** | `~/.claude/settings.local.json` | No (gitignored) | Machine-specific plugins, personal overrides |

### Project Scope Files

When you apply a profile with `--scope project`, two files are created:

```bash
.claudeup.json    # Plugin manifest (lists enabled plugins)
.mcp.json         # MCP server configuration (Claude native format)
```

**Recommended git workflow:**

```bash
# After applying project profile
git add .claudeup.json .mcp.json
git commit -m "Add project-level Claude configuration"

# Team members sync with:
claudeup profile sync
```

### Project Sync

Team members who clone a repository with `.claudeup.json` can install the project's plugins:

```bash
git clone <repo>
cd <repo>
claudeup profile sync
```

This installs all plugins listed in `.claudeup.json`. MCP servers from `.mcp.json` are loaded automatically by Claude Code.

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

**Project Scope** (`--scope project`):

- Project requires specific plugins
- Team needs shared configuration
- Plugin selection is project-dependent
- Files committed to git

**Local Scope** (`--scope local`):

- Machine-specific tools (e.g., Docker-related plugins only on machines with Docker)
- Personal overrides (disable heavy plugins on laptop)
- Experiment without affecting team
- Files NOT committed to git (in `.gitignore`)

### Common Patterns

**Pattern 1: Base + Project**

```bash
# Once: Set up personal defaults
claudeup profile use my-base-tools

# Per-project: Add project plugins
cd ~/backend-api
claudeup profile use backend-stack --scope project
```

**Pattern 2: Base + Project + Local Override**

```bash
# User scope: Base tools
claudeup profile use base

# Project scope: Backend tools
cd ~/api-service
claudeup profile use backend --scope project

# Local scope: Disable heavy plugins on laptop
claudeup profile use lightweight --scope local
```

**Pattern 3: Team Collaboration**

```bash
# Maintainer: Set up project configuration
cd ~/team-project
claudeup profile use team-stack --scope project
git add .claudeup.json .mcp.json
git commit -m "Add Claude configuration"
git push

# Team member: Sync project plugins
git clone <repo>
cd team-project
claudeup profile sync  # Installs plugins from .claudeup.json
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
claudeup profile use hobson --setup

# Skip wizard (for CI/scripting)
claudeup profile use hobson --no-interactive
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

**Auto-generated descriptions:**

- Profiles automatically get meaningful descriptions based on their contents
- Example: "2 marketplaces, 5 plugins" or "1 marketplace, 10 plugins, 2 MCP servers"
- Empty profiles show "Empty profile"

**Description preservation:**

- Custom descriptions (set via `--description`) are preserved when re-saving
- Old generic "Snapshot of current Claude Code configuration" descriptions are automatically updated to auto-generated ones
- Use `--description` flag to override at any time

### Creating Profiles from Existing Ones

Use `profile create` to copy an existing profile with a new name:

```bash
# Interactive selection
claudeup profile create home-setup

# Copy from specific profile
claudeup profile create home-setup --from work

# Copy with custom description
claudeup profile create home-setup --from work --description "Personal development"

# Copy from active profile (with -y flag)
claudeup profile create backup -y
```

Like `profile save`, created profiles inherit the source's description (if custom) or get an auto-generated one (if the source had the old generic description).

## Profile Structure

Profiles are stored in `~/.claudeup/profiles/` as JSON files:

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

| Field | Description |
|-------|-------------|
| `script` | Path to a bash script (relative to profile). Takes precedence over `command`. |
| `command` | Direct bash command to run (used if `script` is not set). |
| `condition` | When to run: `"always"` (default) or `"first-run"` (only if no plugins from the profile's marketplaces are enabled). |

### Hook Flags

```bash
# Force the hook to run even if first-run detection would skip it
claudeup profile use myprofile --setup

# Skip the hook entirely (for CI/scripting)
claudeup profile use myprofile --no-interactive
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

| Profile Type | Delete | Restore |
|--------------|--------|---------|
| Custom profile | ✓ Permanently removes | ✗ Error (not built-in) |
| Customized built-in | ✗ Error (use restore) | ✓ Removes customizations |
| Unmodified built-in | ✗ Error (can't delete) | ✗ Error (nothing to restore) |

### Reset vs Delete vs Restore

These commands serve different purposes:

| Command | What it does |
|---------|--------------|
| `profile reset` | Uninstalls components (plugins, MCP servers, marketplaces) |
| `profile delete` | Permanently removes a custom profile file |
| `profile restore` | Removes customizations from a built-in profile |

**To fully restore a customized built-in profile:**

```bash
# 1. Remove installed components
claudeup profile reset frontend

# 2. Remove your customizations (restores original definition)
claudeup profile restore frontend

# 3. Reinstall from the original built-in
claudeup profile use frontend
```

**Note:** If you only want to restore the profile definition without changing what's installed, just run `profile restore`.

## Sandbox Integration

Profiles can include sandbox-specific settings. See [Sandbox documentation](sandbox.md) for details.
