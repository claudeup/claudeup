# Untracked Scope Hints for Profile Commands

## Problem

When a user has `.claude/settings.json` (project scope) or `.claude/settings.local.json` (local scope) with `enabledPlugins` that weren't applied via a claudeup profile, `profile list` and `profile status` don't indicate these settings exist. The display is misleading -- it shows only the tracked user-scope profile while project/local settings silently override or augment it.

## Design

Add warning hints to `profile list` and `profile status` when settings exist at project or local scope without a tracked profile.

### Detection logic

For each of project and local scope:

1. Check if a profile is already tracked at that scope (via `getAllActiveProfiles`)
2. If not tracked, load settings for that scope (`LoadSettingsForScope`)
3. Count enabled plugins (`enabledPlugins` entries where value is `true`)
4. If count > 0, emit a hint

Extract this into a shared helper (`getUntrackedScopes`) in `scope_helpers.go` so both commands use identical logic.

### `profile list` output

After the profiles sections, before the footer arrows:

```
Your profiles

* base                 base marketplaces and plugins [user]

  ⚠ project: 7 plugins in .claude/settings.json (no profile tracked)
    → Save with: claudeup profile save <name> && claudeup profile apply <name> --project

→ Use 'claudeup profile show <name>' for details
→ Use 'claudeup profile apply <name>' to apply a profile
```

### `profile status` output

In the header area, after the `[active at X scope]` line:

```
Profile: base
  [active at user scope]
  ⚠ project: 7 plugins in .claude/settings.json (no profile tracked)
    → Save with: claudeup profile save <name> && claudeup profile apply <name> --project
```

### Conditions for showing hints

- Settings file exists at that scope
- At least one plugin is enabled in `enabledPlugins`
- No profile is tracked at that scope (not in `getAllActiveProfiles` results)
- Only shown when not filtering by scope (`--user`, `--project`, `--local` flags suppress hints)

### Suggested command

Each hint includes an actionable two-step workflow:

```
→ Save with: claudeup profile save <name> && claudeup profile apply <name> --project
```

This uses existing commands:

1. `profile save` snapshots all scopes into a profile
2. `profile apply --project` registers the profile as tracked at project scope

## Files to change

- `internal/commands/scope_helpers.go` -- add `getUntrackedScopes` helper
- `internal/commands/profile_cmd.go` -- add hints to `runProfileList` and `runProfileStatus`
- `test/acceptance/profile_list_test.go` -- test hint appears/doesn't appear
- `test/acceptance/profile_status_test.go` -- test hint appears/doesn't appear

## Out of scope

- Detecting non-plugin settings (MCP servers, permissions, hooks) at untracked scopes
- Automatic profile creation from untracked settings
- Drift detection between tracked profile and current settings
