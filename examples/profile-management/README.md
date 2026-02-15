# Profile Management

Create, save, switch, and compose Claude Code configuration profiles.

Profiles are the core concept in claudeup -- they bundle plugins, MCP servers,
and settings into named configurations that can be saved, shared, and reapplied.

## Who is this for?

Users who want to:

- Capture their current Claude Code setup so they can restore it later
- Create purpose-built configurations for different projects or workflows
- Build composable profile stacks from small, focused building blocks
- Switch between configurations without manually reconfiguring each time

## Scripts

| Script                     | What it does                                                                                                                         |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `01-save-current-state.sh` | Snapshots your current plugins, MCP servers, and settings into a named profile with `profile save`                                   |
| `02-create-custom.sh`      | Creates a profile by writing JSON directly, inspects it with `profile show`, applies it, and captures state with `profile save`      |
| `03-switch-profiles.sh`    | Previews differences with `profile status`, applies a different profile, and verifies the switch                                     |
| `04-clone-and-modify.sh`   | Clones an existing profile with `profile clone`, then walks through the modify-and-save workflow                                     |
| `05-composable-stacks.sh`  | Creates building-block profiles in category subdirectories, composes them into stacks via `includes`, and demonstrates nested stacks |

## Suggested order

Start with `01-save-current-state.sh` to understand profiles, then explore based on
your needs:

- **Quick start:** 01 then 03 (save your setup, learn to switch)
- **Custom profiles:** 01 then 02 or 04 (save, then create or clone)
- **Advanced composition:** 01 then 05 (understand stacks and includes)

## What you'll learn

- `profile save <name>` captures your current configuration as a reusable profile
- `profile create` launches an interactive wizard for building profiles from scratch
- `profile clone <new> --from <existing>` copies a profile as a starting point
- `profile apply <name> --user|--project|--local` applies at a specific scope
- `profile status <name>` previews what would change before applying
- Profiles can include other profiles via `includes` for composable stacks
- Building-block profiles live in subdirectories (e.g., `profiles/languages/go.json`)
- Stack profiles are pure composition -- they only contain name, description, and includes

## Important details

- `02-create-custom.sh` writes profile JSON directly to demonstrate the file format.
  For guided creation, use the interactive wizard: `claudeup profile create`.
- `05-composable-stacks.sh` creates real profile files in the temp environment to
  demonstrate stacks. This is the most comprehensive example in the set.
- Profiles are stored as JSON in `~/.claudeup/profiles/`

## Next steps

- [Team Setup](../team-setup/) -- layer personal and team profiles across scopes
- [Troubleshooting](../troubleshooting/) -- diagnose profile issues
