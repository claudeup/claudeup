# Getting Started

First steps with claudeup. Start here if you've just installed claudeup and want
to verify everything works before diving into profiles and plugins.

## Who is this for?

Anyone who just installed claudeup and wants to:

- Confirm the installation is working
- Understand what claudeup can see about their Claude Code setup
- Learn the basic commands before creating or applying profiles

## Scripts

| Script                      | What it does                                                                                                                   |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `01-check-installation.sh`  | Runs `--version`, `status`, and `doctor` to verify claudeup is installed and can see your Claude Code configuration            |
| `02-explore-profiles.sh`    | Lists available profiles, shows profile contents, and explains Claude Code's three configuration scopes (user, project, local) |
| `03-apply-first-profile.sh` | Walks through applying a profile -- checks the current state, applies a profile at user scope, and verifies the change         |

## Suggested order

Run them in numbered order. Each script builds on concepts from the previous one:

1. **Check installation** -- make sure claudeup works
2. **Explore profiles** -- understand what's available
3. **Apply first profile** -- make your first configuration change

## What you'll learn

- The `status` command gives a high-level overview of plugins, marketplaces, and profiles
- The `doctor` command diagnoses configuration issues
- Profiles bundle plugins, MCP servers, and settings into reusable configurations
- Claude Code has three scopes (user, project, local) with later scopes overriding earlier ones
- The `--user`, `--project`, and `--local` flags control where a profile is applied

## Next steps

After completing these scripts:

- [Profile Management](../profile-management/) -- create, save, and switch profiles
- [Plugin Management](../plugin-management/) -- manage individual plugins
