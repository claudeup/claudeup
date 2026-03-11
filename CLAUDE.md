# claudeup

CLI tool for managing Claude Code profiles and configurations.

## Design Philosophy

**claudeup is a profile manager for bootstrapping, not an ongoing config management layer.**

- Profiles are most valuable at bootstrap time (applying curated settings to new projects)
- Users edit `settings.json` directly during daily work -- this is natural and low-friction
- Bootstrap and get out of the way -- Claude's native settings files handle ongoing management

**Core scope:** Profile management (save/apply/list/delete), diagnostics (doctor), onboarding (setup).

**Intentionally excluded:** Sandbox (devcontainers do this better), `.claudeup.json` auto-detection (profiles should be explicit).

## Architecture

### Two-Directory Model

claudeup operates across two directory trees:

**Claude's active directory** (`claudeDir`, defaults to `~/.claude/`):

- Owned by Claude Code -- claudeup only writes symlinks here
- `agents/`, `commands/`, `skills/`, `hooks/`, `rules/`, `output-styles/` contain symlinks to enabled items
- `settings.json`, `installed_plugins.json` are read/modified during profile operations

**claudeup's home directory** (`claudeupHome`, defaults to `~/.claudeup/`):

- Owned by claudeup -- all claudeup-managed data lives here
- `ext/<category>/` stores actual item files; `enabled.json` tracks enabled state
- `profiles/`, `last-applied.json`, `events/`, `config.json`

Symlinks are absolute: `~/.claude/agents/foo.md --> ~/.claudeup/ext/agents/foo.md`

In code, `ext.NewManager(claudeDir, claudeupHome)` takes both paths explicitly.
In tests, each path gets its own `t.TempDir()` to verify proper separation.

### Project Structure

- `cmd/claudeup/` - Entry point
- `internal/acceptance/` - Acceptance test helpers
- `internal/backup/` - Backup and restore for configuration files
- `internal/breadcrumb/` - Per-scope tracking of last-applied profile
- `internal/claude/` - Claude Code configuration file handling
- `internal/commands/` - Cobra command implementations
- `internal/config/` - Path resolution and global configuration
- `internal/events/` - Operation logging and event diffs
- `internal/ext/` - Extension management (symlinks, install, list, resolve)
- `internal/local/` - Local-scope operations
- `internal/mcp/` - MCP server discovery
- `internal/pluginsearch/` - Plugin search and filtering
- `internal/profile/` - Profile management (save, load, apply, snapshot)
- `internal/secrets/` - Secret resolution (env, 1Password, keychain)
- `internal/selfupdate/` - Binary self-update
- `internal/ui/` - Terminal output, styling, and prompts
- `test/acceptance/` - Acceptance tests (real binary, isolated temp dirs)
- `test/integration/` - Integration tests (internal packages, fake fixtures)
- `test/helpers/` - Shared test utilities

### Embedded Profiles

Built-in profiles are embedded from `internal/profile/profiles/*.json` using Go's embed directive.

## Build & Test

```bash
go build -o bin/claudeup ./cmd/claudeup
go test ./...
go test ./test/acceptance/... -v   # CLI behavior tests
go test ./test/integration/... -v  # Internal package tests
```

Tests use [Ginkgo](https://onsi.github.io/ginkgo/) BDD with [Gomega](https://onsi.github.io/gomega/) matchers.

```bash
go run github.com/onsi/ginkgo/v2/ginkgo -v ./test/...
```

### Ginkgo Gotchas

- Use `ginkgo -focus "pattern"` to filter tests. `go test -run` filters Go test functions, not Ginkgo spec names.
- On macOS, symlink resolution causes path mismatches (`/var` vs `/private/var`). Use `filepath.EvalSymlinks()` to normalize paths before comparison.

### Test Environment Isolation

Tests use `CLAUDE_CONFIG_DIR` to redirect Claude Code's user-scope directory (`~/.claude/`) to an isolated temp dir. This prevents tests from touching real configuration.

Key constraint: `CLAUDE_CONFIG_DIR` only redirects user-scope files. Project-scope (`.claude/settings.json`, `.mcp.json`) and local-scope (`.claude/settings.local.json`) files remain in the project directory and are unaffected.

## Non-Obvious Conventions

### Plans and Documentation

Design documents and implementation plans go in a separate repository: `https://github.com/claudeup/claudeup-superpowers.git`. Clone as `../claudeup-superpowers`. Save plans there, not in this repo.

Read latest Claude Code docs on [settings](https://code.claude.com/docs/en/settings).

### Feature Development

Feature development uses git worktrees in `.worktrees/` directory (in .gitignore).

### Profile Scope Awareness

claudeup respects Claude Code's scope layering (user -> project -> local, later overrides earlier).

- `profile list`: `*` = highest precedence active profile, `○` = overridden at lower scope
- `profile apply` writes breadcrumb entries for all scopes the profile touches

### Breadcrumb System

`profile diff` and `profile save` default to the last-applied profile (stored in `~/.claudeup/last-applied.json`):

```json
{
  "user": { "profile": "base-tools", "appliedAt": "2026-02-28T13:00:00Z" },
  "project": { "profile": "claudeup", "appliedAt": "2026-02-28T14:00:00Z" }
}
```

- Scope flags (`--user`, `--project`, `--local`, `--scope`) select a specific scope's breadcrumb
- Breadcrumb write errors are non-fatal warnings
- If breadcrumb file is unreadable, pass an explicit profile name to `save`/`diff`

### Claude CLI Format Compatibility

claudeup parses Claude CLI's internal JSON files (`installed_plugins.json`, `settings.json`). When Claude CLI updates break parsing:

1. Smoke tests in `test/integration/claude/format_compatibility_test.go` fail against your real `~/.claude/`
2. Examine actual file structure to understand changes
3. Update validation in `internal/claude/validation.go`
4. Extend `LoadPlugins()` to handle the new version
5. Update supported version range in error messages
