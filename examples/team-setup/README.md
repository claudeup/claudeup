# Team Setup

Share Claude Code configurations across a team using scopes and profile layering.

## Who is this for?

Team leads and developers who want to:

- Establish shared project configurations that every team member gets
- Layer personal preferences on top of team requirements
- Understand how Claude Code's scope system enables team collaboration

## Scripts

| Script                   | What it does                                                                                                                                          |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `01-scoped-profiles.sh`  | Explains the three configuration scopes (user, project, local), their precedence, and how to apply profiles to each scope                             |
| `04-profile-layering.sh` | Demonstrates combining personal (user scope) and team (project scope) profiles, shows precedence rules, and walks through a recommended team workflow |

> **Note:** Script numbering has gaps (no `02-*` or `03-*`). This is a known issue
> and does not affect functionality.

## Suggested order

Read `01-scoped-profiles.sh` first to understand scopes, then `04-profile-layering.sh`
for the practical team workflow.

## What you'll learn

- **User scope** (`~/.claude/settings.json`) holds personal defaults that apply everywhere
- **Project scope** (`.claude/settings.json`) holds team settings checked into git
- **Local scope** (`.claude/settings.local.json`) holds personal overrides, git-ignored
- Later scopes override earlier ones: local > project > user
- User and project profiles can be active simultaneously -- they combine
- `profile list` shows which profiles are active at each scope with `*` (active) and `○` (overridden) markers

## Recommended team workflow

The `04-profile-layering.sh` script demonstrates this pattern:

1. **Each developer** saves personal tools as a user-scope profile
2. **Team lead** creates and applies a project-scope profile, commits `.claude/settings.json`
3. **After cloning**, each developer runs `claudeup profile apply` to get team config
4. **Personal overrides** go in local scope (git-ignored)

## Important details

- These scripts are mostly informational -- they explain concepts with example
  output rather than making changes. They work well in temp mode for learning.
- `04-profile-layering.sh` shows example JSON and simulated output to illustrate
  the layering concept without requiring real plugins.

## Next steps

- [Profile Management](../profile-management/) -- create the profiles to use in team setups
- [Troubleshooting](../troubleshooting/) -- diagnose scope-related issues
