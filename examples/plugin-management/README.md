# Plugin Management

View and update the Claude Code plugins managed through claudeup.

## Who is this for?

Users who already have claudeup set up and want to:

- See which plugins are installed and their current state
- Check for available plugin updates
- Understand how plugins relate to marketplaces

## Scripts

| Script                 | What it does                                                                                        |
| ---------------------- | --------------------------------------------------------------------------------------------------- |
| `01-list-plugins.sh`   | Lists all installed plugins with their enabled/disabled state and shows plugin details via `status` |
| `02-manage-plugins.sh` | Browse and inspect plugins with claudeup before installing via the claude CLI                       |
| `03-check-upgrades.sh` | Runs `outdated` to find available updates, then `upgrade` to apply them                             |

## What you'll learn

- `plugin list` shows all plugins and whether they're enabled or disabled
- `status` groups plugins by their source marketplace
- `outdated` checks all marketplaces for available updates
- `upgrade` fetches and applies updates from all marketplaces
- Plugins are installed from marketplaces (plugin repositories), not individually

## Important details

- The `upgrade` command only runs in `--real` mode since there are no plugins in the temp environment
- In temp mode, the script shows the commands you would run but skips actual execution

## Next steps

- [Profile Management](../profile-management/) -- bundle plugin selections into reusable profiles
- [Troubleshooting](../troubleshooting/) -- diagnose plugin issues with `doctor`
