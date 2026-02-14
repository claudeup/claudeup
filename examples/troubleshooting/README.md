# Troubleshooting

Diagnose and fix issues with your Claude Code configuration using claudeup's
diagnostic and audit tools.

## Who is this for?

Users who need to:

- Figure out why something broke after a configuration change
- Find and fix corrupted or orphaned plugin entries
- Trace what claudeup changed and when
- Understand the before/after state of configuration files

## Scripts

| Script               | What it does                                                                                                            |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `01-run-doctor.sh`   | Runs `doctor` to diagnose issues (missing files, invalid config, path problems) and shows `cleanup` for automatic fixes |
| `02-view-events.sh`  | Lists recent file operations with `events`, demonstrates filtering by file, operation type, and time range              |
| `03-diff-changes.sh` | Uses `events diff` to show before/after comparisons of configuration files, with `--full` mode for nested object diffs  |

## Suggested order

Run them in order when troubleshooting:

1. **Run doctor** -- is there a known issue?
2. **View events** -- what changed recently?
3. **Diff changes** -- what exactly was different?

## What you'll learn

- `doctor` checks for missing plugin files, invalid configuration, orphaned entries, and path mismatches
- `cleanup` can automatically fix many issues that `doctor` finds (use `--dry-run` to preview)
- `events` shows a timestamped log of every file operation claudeup has performed
- `events --file <path>` filters to changes affecting a specific file
- `events --since 24h` filters by time window
- `events diff --file <path>` shows the most recent before/after snapshot for a file
- `events diff --file <path> --full` recursively diffs nested objects with color-coded symbols (`+` added, `-` removed, `~` modified)

## Debugging workflow

The scripts teach this pattern:

1. Something broke after a change
2. `claudeup events --since 1h` -- find recent operations
3. `claudeup events diff --file <path> --full` -- see what changed
4. Decide: revert with `profile apply` or fix forward

## Important details

- `cleanup` only runs in `--real` mode. In temp mode the script shows the command
  and `--dry-run` flag.
- Event logs are stored at `~/.claudeup/events/operations.log`
- Event logs may contain sensitive data if configuration files include API keys or
  tokens (see the project CLAUDE.md for privacy details)

## Next steps

- [Getting Started](../getting-started/) -- re-verify installation after fixing issues
- [Profile Management](../profile-management/) -- restore a known-good profile
