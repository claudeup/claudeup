# Team Setup

Share Claude Code configurations across a team using scopes and profile layering.

## Who is this for?

Team leads and developers who want to:

- Establish shared project configurations that every team member gets
- Layer personal preferences on top of team requirements
- Understand how Claude Code's scope system enables team collaboration

## Scripts

| Script                          | What it does                                                                                                                                          |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `01-scoped-profiles.sh`         | Explains the three configuration scopes (user, project, local), their precedence, and how to apply profiles to each scope                             |
| `02-isolated-workspace-demo.sh` | End-to-end demo simulating three team members (Alice, Bob, Charlie) with isolated environments, project-scoped extensions, and git-based sharing      |
| `03-profile-layering.sh`        | Demonstrates combining personal (user scope) and team (project scope) profiles, shows precedence rules, and walks through a recommended team workflow |
| `04-devcontainer-demo.sh`       | End-to-end demo using claudeup-lab to create real Docker containers for three team members with profile stacking via `--base-profile`                 |
| `05-scope-apply-demo.sh`        | Applies profiles at all three scopes (user, project, local) and shows how `profile list` markers change as scopes accumulate                          |

## Suggested order

Read `01-scoped-profiles.sh` first to understand scopes, then `02-isolated-workspace-demo.sh`
to see a realistic team workflow in action, then `03-profile-layering.sh` for profile
layering details. Finally, `04-devcontainer-demo.sh` shows the same team pattern using real
Docker containers instead of environment variable isolation.

## What you'll learn

- **User scope** (`~/.claude/settings.json`) holds personal defaults that apply everywhere
- **Project scope** (`.claude/settings.json`) holds team settings checked into git
- **Local scope** (`.claude/settings.local.json`) holds personal overrides, git-ignored
- Later scopes override earlier ones: local > project > user
- User and project profiles can be active simultaneously -- they combine
- `profile list` shows which profiles are active at each scope with `*` (active) and `○` (overridden) markers

## Recommended team workflow

The `02-isolated-workspace-demo.sh` script demonstrates this pattern:

1. **Team lead** creates and applies a project-scope profile (plugins, rules, agents)
2. **Team lead** commits `.claude/` to git so settings and extensions travel with the repo
3. **After cloning**, teammates get the team config automatically -- no re-application needed
4. **Each developer** applies their own user-scope profile for personal tools

## Important details

- `01-scoped-profiles.sh` and `03-profile-layering.sh` are mostly informational --
  they explain concepts with example output rather than making changes.
- `02-isolated-workspace-demo.sh` runs real `claudeup` commands against isolated
  temp directories. It demonstrates project-scoped extensions (rules and agents
  copied into `.claude/`) and how project configuration travels through git.
  The `--real` flag is not supported because each simulated team member needs
  their own isolated config directory.
- `04-devcontainer-demo.sh` requires Docker and `claudeup-lab` installed.
  Run `claudeup-lab doctor` to check prerequisites. It creates real containers
  that take a few minutes to start. The `--real` flag is not supported because
  containers provide their own isolation.

## Next steps

- [Profile Management](../profile-management/) -- create the profiles to use in team setups
- [Troubleshooting](../troubleshooting/) -- diagnose scope-related issues
