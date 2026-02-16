# Extension Management

Manage Claude Code extensions - custom agents, rules, commands, skills, hooks, and output-styles stored as files in `~/.claudeup/ext/`.

## Who is this for?

Users who want to:

- Understand how extensions differ from marketplace plugins
- Install team-shared extensions from git repositories
- Enable and disable extensions selectively
- Create reusable extension libraries for team collaboration

## Scripts

| Script                      | What it does                                                                                          |
| --------------------------- | ----------------------------------------------------------------------------------------------------- |
| `01-list-extensions.sh`     | Lists all installed extensions across all categories and shows their enabled/disabled status          |
| `02-enable-disable.sh`      | Demonstrates enabling and disabling extensions individually, in bulk, or using wildcard patterns      |
| `03-install-from-path.sh`   | Shows how to install extensions from local directories (clone git repos first, then install per category) |

## What you'll learn

- Extensions are files (not marketplace plugins) stored in `~/.claudeup/ext/`
- Each extension has a category: agents, commands, skills, hooks, rules, or output-styles
- `ext list` shows all extensions and their enabled/disabled state
- `ext enable` and `ext disable` control which extensions are active
- Wildcard patterns allow bulk enable/disable operations (e.g., `rules/*`)
- `ext install <category> <path>` copies extensions from local paths into storage
- Enabled extensions are symlinked into `~/.claude/<category>/`
- Extensions can be shared across teams via git repositories

## Extension vs Plugin

- **Plugins** come from marketplaces (repositories of pre-built packages)
- **Extensions** are individual files you manage directly
- Profiles can reference both plugins and extensions
- Extensions are great for team-specific customizations

## Recommended team workflow

1. Create a team git repository for shared extensions
2. Organize by category (agents/, rules/, etc.)
3. Team members clone the repo: `git clone https://github.com/myteam/extensions ~/team-extensions`
4. Install per category: `claudeup ext install agents ~/team-extensions/agents`
5. Selectively disable what they don't need: `claudeup ext disable agents 'unwanted-*'`
6. Update regularly: `cd ~/team-extensions && git pull && claudeup ext install agents ./agents`

## Important details

- The `ext install` command copies files from external sources into `~/.claudeup/ext/`
- Installed extensions are enabled automatically after install; use `ext disable` and `ext enable` to control which ones are active
- `ext sync` rebuilds symlinks from the enabled state (useful after git operations)
- Quote wildcard patterns to prevent shell expansion: `'rules/*'` not `rules/*`

## Next steps

- [Profile Management](../profile-management/) -- bundle extension selections into profiles
- [Team Setup](../team-setup/) -- apply extensions at project scope for team sharing
