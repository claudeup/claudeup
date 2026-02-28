# Last-Applied Breadcrumb

Records which profile was last applied at each scope so `profile diff` and
`profile save` can default to it.

## Problem

After applying a profile and tweaking settings across projects, there is no way
to know which profile was last applied. `profile diff` requires a name argument,
so the user must guess. Saving changes means inventing a new profile name --
leading to profile sprawl -- because the original name is unknown.

## Approach

Write a lightweight breadcrumb at apply time. Not "active profile" state (that
was removed in PR #208 for good reason) -- just a historical record of what was
applied and when.

## Breadcrumb File

**Location:** `~/.claudeup/last-applied.json` (path derived from `claudeupHome`)

**Format:**

```json
{
  "user": {
    "profile": "base-tools",
    "appliedAt": "2026-02-27T21:15:00Z"
  },
  "project": {
    "profile": "my-project",
    "appliedAt": "2026-02-28T10:30:00Z"
  }
}
```

- Keys are scope names: `user`, `project`, `local`
- Each entry records the profile name and timestamp
- Entries are independent -- applying at project scope does not touch the user
  entry
- Written atomically (write-tmp + rename)
- If the file does not exist, commands behave as today (explicit name required)

## Write Behavior

The breadcrumb is written at one point: the end of a successful `profile apply`.

| Scenario                     | Behavior                             |
| ---------------------------- | ------------------------------------ |
| Single-scope apply           | Write one entry for the target scope |
| Multi-scope apply (PerScope) | Write entries for each scope touched |
| Stack apply (includes)       | Record the top-level profile name    |
| Dry run                      | No breadcrumb written                |
| Failed apply                 | No breadcrumb written                |
| `--replace` flag             | Breadcrumb written normally          |

What the breadcrumb does NOT do:

- Does not clear other scopes when applying
- Does not validate that the profile still exists
- Is not written by `profile save` (save creates profiles, it does not apply
  them)

## Read Behavior

### `profile diff` (no args)

1. Load `last-applied.json`
2. Find highest-precedence scope with a breadcrumb (local > project > user)
3. Load that profile by name
4. Diff against live state at that scope
5. Print header: `Comparing against "base-tools" (applied Feb 27, user scope)`
6. If no breadcrumbs exist: print usage guidance

`profile diff <name>` works exactly as today. Explicit name always wins.

### `profile save` (no args)

1. Load `last-applied.json`
2. Find highest-precedence breadcrumb
3. Use that profile name as the save target
4. Print confirmation: `Saving to "base-tools" (last applied Feb 27)`
5. If no breadcrumbs exist: name is required (same error as today)

`profile save <name>` works exactly as today.

### `--scope` flag interaction

When `--scope` is passed, use the breadcrumb for that specific scope instead of
highest-precedence. If no breadcrumb exists at that scope, error with guidance.

## Cleanup

### `profile delete`

When deleting a profile, scan breadcrumbs and remove any entries referencing the
deleted profile. This avoids stale references.

### Deleted profiles (without cleanup)

If a breadcrumb references a profile that no longer exists (e.g., deleted
outside claudeup), `profile diff` prints:
`Profile "my-setup" no longer exists. Run: claudeup profile diff <name>`

### Staleness

No expiry. The timestamp in output gives the user context to judge relevance.

## Edge Cases

**Multiple worktrees** -- The breadcrumb file is global (`~/.claudeup/`).
Project-scope breadcrumbs could collide across worktrees. Acceptable for v1
because the breadcrumb is a convenience hint, not a source of truth.

**Profile rename** -- The rename command updates breadcrumb entries.

## Testing

The breadcrumb file path comes from `claudeupHome`, so tests get isolation
automatically via `t.TempDir()`. Test cases:

- Apply writes breadcrumb at correct scope
- Multi-scope apply writes multiple entries
- Stack apply records top-level name
- Dry run does not write breadcrumb
- `profile diff` with no args uses highest-precedence breadcrumb
- `profile diff` with `--scope` uses that scope's breadcrumb
- `profile diff <name>` ignores breadcrumb
- `profile save` with no args uses breadcrumb name
- `profile delete` removes breadcrumb entry
- Missing breadcrumb file falls back to requiring explicit name
- Stale breadcrumb (deleted profile) produces clear error

## Backward Compatibility

Fully backward compatible. Both `profile diff` and `profile save` continue
accepting explicit name arguments. The breadcrumb only provides defaults when no
argument is given.
