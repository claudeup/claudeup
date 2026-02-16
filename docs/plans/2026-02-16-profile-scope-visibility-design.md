# Profile Scope Visibility and Save/Apply UX

**Issue:** [#205](https://github.com/claudeup/claudeup/issues/205)
**Date:** 2026-02-16

## Problem

When a user has plugins at multiple scopes (user + project), claudeup provides
no single view of the effective configuration. `profile show current` only
displays the tracked profile at the highest-precedence scope. User-scope plugins
are invisible unless the user manually inspects `~/.claude/settings.json`.

The flow for adopting existing untracked settings into a profile requires two
commands (`save && apply`), produces misleading output ("Installed 7 plugins"
when nothing changed), and prompts for confirmation on an idempotent operation.

## Design

### 1. `profile status` becomes the live effective config view

`profile status` reads all three `settings.json` files directly and displays the
effective configuration grouped by scope. Each scope is annotated with its
tracked profile name, or "(untracked)" if no profile is tracked.

**Output format:**

```
$ claudeup profile status
Effective configuration for /Users/mark/code/claudeup

  User scope (profile: base)
    Plugins:
      - claude-hud@claude-hud
      - episodic-memory@superpowers-marketplace
      - feature-dev@claude-plugins-official
      - gopls-lsp@claude-plugins-official
      - hookify@claude-plugins-official
      - pr-review-toolkit@claude-plugins-official
    Disabled:
      - safety-hooks@cctools-plugins
      - vsphere-architect@vsphere-architect

  Project scope (profile: projects/claudeup)
    Plugins:
      - backend-development@claude-code-workflows
      - claude-mem@thedotmack
      - code-documentation@claude-code-workflows
      - elements-of-style@superpowers-marketplace
      - superpowers@superpowers-marketplace
      - tdd-workflows@claude-code-workflows
      - unit-testing@claude-code-workflows

  Marketplaces:
    - obra/superpowers-marketplace
    - anthropics/claude-plugins-official
```

When a scope has no tracked profile:

```
  Project scope (untracked)
    Plugins:
      - backend-development@claude-code-workflows
      ...
    -> Save with: claudeup profile save <name> --project --apply
```

When a scope has no plugins, it is omitted entirely.

**Implementation:** Read each scope's `settings.json` via `LoadSettingsForScope()`.
Look up tracking info from `projects.json` and `config.json`. Render grouped
output. This replaces the current `runProfileStatus()` which resolves a profile
name and shows its saved contents.

**`profile show current` becomes an alias for `profile status`.** Users who have
learned the old command get the new behavior without breakage. `profile show <name>`
(non-"current") stays unchanged -- it loads and displays a saved profile definition.

### 2. `profile list` detects untracked user-scope plugins

`getUntrackedScopes()` currently only checks project and local scopes. User scope
is never flagged as untracked.

**Change:** Add user scope to the scopes checked. If `~/.claude/settings.json`
has enabled plugins but no user-scope profile is tracked in `config.json`, show
the same hint pattern used for project/local:

```
  user: 12 plugins in ~/.claude/settings.json (no profile tracked)
    -> Save with: claudeup profile save <name> --apply
```

No `--user` flag needed in the hint since user scope is the default for
save/apply.

**Footer hints update:**

```
-> Use 'claudeup profile status' to see effective configuration
-> Use 'claudeup profile show <name>' for profile details
-> Use 'claudeup profile apply <name>' to apply a profile
```

### 3. `profile save --apply` flag

Collapse the two-step ceremony (`save && apply`) into a single command.

```sh
claudeup profile save projects/claudeup --project --apply
```

**Behavior:**

1. Save the snapshot as usual (respecting `--project`/`--local`/`--user` scope)
2. Run the same tracking logic that `apply` does (update `projects.json` or
   `config.json`)
3. Skip the "Proceed? [Y/n]" prompt since the state was just snapshotted from
   existing settings -- there is nothing to confirm
4. Output: `Saved and applied profile "projects/claudeup" (project scope)`

**Interaction with `--yes`:** The existing `--yes` flag skips the overwrite
confirmation when saving over an existing profile. `--apply` is orthogonal --
it controls whether to also track the profile. They compose:
`--yes --apply` does a fully non-interactive save-and-track.

**Untracked scope hints update to recommend the single command:**

```
  project: 7 plugins in .claude/settings.json (no profile tracked)
    -> Save with: claudeup profile save <name> --project --apply
```

### 4. Smarter apply messaging

#### 4a. Idempotent apply uses "tracking" instead of "installed"

When the plugins in the profile already match the settings file (the
save-then-apply case), reflect reality:

```
# When state already matches:
Tracking 7 existing plugins as profile "projects/claudeup"

# When state actually changes:
Installed 3 new plugins, disabled 1 plugin
```

The apply logic already diffs the profile against settings. This changes
only the output verb.

#### 4b. Better auto-generated descriptions

When `profile save` auto-generates a description, include plugin counts
per scope:

```
# Current:
"1 marketplace"

# New (single scope):
"7 project plugins, 1 marketplace"

# New (multi-scope):
"5 user plugins, 7 project plugins, 2 marketplaces"
```

## Summary of changes

| Area                   | Change                                                            |
| ---------------------- | ----------------------------------------------------------------- |
| `profile status`       | Live effective config across all scopes with tracking annotations |
| `profile show current` | Alias for `profile status`                                        |
| `profile show <name>`  | Unchanged -- shows saved profile definition                       |
| `profile list`         | Detect untracked user-scope plugins                               |
| `profile list` footer  | Updated hints pointing to `status`                                |
| `profile save`         | Add `--apply` flag for single-command adopt                       |
| `profile apply` output | "Tracking N existing plugins" when idempotent                     |
| Description generation | Include plugin counts per scope                                   |

## Files likely affected

- `internal/commands/profile_cmd.go` -- `runProfileStatus()`, `runProfileShow()`,
  `runProfileSave()`, `runProfileApply()`, footer hints
- `internal/commands/scope_helpers.go` -- `getUntrackedScopes()` add user scope
- `internal/profile/snapshot.go` -- description generation
- `test/acceptance/profile_status_test.go` -- new/updated tests
- `test/acceptance/profile_show_test.go` -- `show current` alias behavior
- `test/acceptance/profile_save_test.go` -- `--apply` flag tests
- `test/acceptance/profile_list_test.go` -- user-scope untracked hints
