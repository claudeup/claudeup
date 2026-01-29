# claudeup

CLI tool for managing Claude Code profiles and configurations.

## Design Philosophy

**claudeup is a profile manager for bootstrapping, not an ongoing config management layer.**

- Profiles are most valuable at bootstrap time (applying curated settings to new projects)
- Users will edit `settings.json` directly during daily work - this is natural and low-friction
- Forcing users to keep profiles in sync with settings adds friction that leads to abandonment
- Bootstrap and get out of the way - Claude's native settings files handle ongoing management

**Core scope:**

- Profile management (save/apply/list/delete)
- Diagnostics (doctor)
- Onboarding (setup)

**Intentionally excluded:**

- Sandbox (devcontainers do this better)
- `.claudeup.json` auto-detection (profiles should be explicit)
- Drift detection / `(modified)` markers (noise, not signal)

## Project Structure

- `cmd/claudeup/` - Main entry point
- `internal/commands/` - Cobra command implementations
- `internal/profile/` - Profile management (save, load, apply, snapshot)
- `internal/claude/` - Claude Code configuration file handling
- `internal/secrets/` - Secret resolution (env, 1Password, keychain)
- `test/acceptance/` - Acceptance tests (CLI behavior, real binary execution)
- `test/integration/` - Integration tests (internal packages with fake fixtures)
- `test/helpers/` - Shared test utilities

## Plans and Documentation

**Design documents and implementation plans go in a separate repository:**

```sh
https://github.com/claudeup/claudeup-superpowers.git
```

Clone locally as `../claudeup-superpowers` or similar. When brainstorming features or creating implementation plans, save them there - not in this repository.

**Claude Code Documentation (Settings):**

- Read latest version of published Claude Code docs on [settings](https://code.claude.com/docs/en/settings)

## Development

```bash
# Run all tests
go test ./...

# Build
go build -o bin/claudeup ./cmd/claudeup

# Run acceptance tests (CLI behavior)
go test ./test/acceptance/... -v

# Run integration tests
go test ./test/integration/... -v
```

## Testing

Tests use [Ginkgo](https://onsi.github.io/ginkgo/) BDD framework with [Gomega](https://onsi.github.io/gomega/) matchers.

**Test types:**

- **Acceptance tests** (`test/acceptance/`) - Execute the real `claudeup` binary in isolated temp directories. Test CLI behavior end-to-end.
- **Integration tests** (`test/integration/`) - Test internal packages with fake Claude installations. No binary execution.
- **Unit tests** (`internal/*/`) - Standard Go tests for individual functions.

**Writing tests:**

```go
var _ = Describe("feature", func() {
    var env *helpers.TestEnv

    BeforeEach(func() {
        env = helpers.NewTestEnv(binaryPath)
    })

    It("does something", func() {
        result := env.Run("command", "args")
        Expect(result.ExitCode).To(Equal(0))
    })
})
```

**Running with Ginkgo CLI (optional, nicer output):**

```bash
go run github.com/onsi/ginkgo/v2/ginkgo -v ./test/...
```

## Testing Environment Isolation

This project uses the `CLAUDE_CONFIG_DIR` environment variable to create isolated Claude Code environments for testing. This prevents tests from interfering with your real `~/.claude` configuration.

### How It Works

When running tests or development builds of claudeup, we set:

```bash
export CLAUDE_CONFIG_DIR=/path/to/test/claude-config
```

This redirects Claude Code's **user-scope** configuration directory only, including:

- `.claude.json` (main config)
- `.credentials.json` (auth tokens)
- `projects/` (project-specific settings)
- `settings.json` (user settings)
- `agents/` (user-level agents)
- `todos/`, `statsig/`, `shell-snapshots/`

### Important Caveats

1. **Only user-scope files are redirected** — `CLAUDE_CONFIG_DIR` affects `~/.claude/` but NOT project-scope or local-scope files. Claude Code's scope system determines what is and isn't redirected.

2. **Project-scope files remain in the project directory** — These are NOT affected by `CLAUDE_CONFIG_DIR`:
   - `.claude/settings.json` (project settings)
   - `.claude/agents/` (project agents)
   - `.mcp.json` (project MCP servers)
   - `CLAUDE.md` or `.claude/CLAUDE.md` (project memory)

3. **Local-scope files remain in the project directory** — These are also NOT redirected:
   - `.claude/settings.local.json` (local settings overrides)
   - `CLAUDE.local.md` (local memory)

4. **Managed settings are NOT redirected** — System-level managed settings at `/Library/Application Support/ClaudeCode/` (macOS) or `/etc/claude-code/` (Linux) are unaffected.

5. **IDE integration may not respect it** — The `/ide` command may look for lock files in `~/.claude/ide/` by default.

6. **Always verify the correct directory is being used** — When debugging, check that files are being read/written to `$CLAUDE_CONFIG_DIR` and not `~/.claude`.

### Testing Commands

Before running tests, ensure `CLAUDE_CONFIG_DIR` is set and points to an isolated directory:

```bash
# Verify the variable is set
echo $CLAUDE_CONFIG_DIR

# Check what config Claude Code is actually using
ls -la $CLAUDE_CONFIG_DIR
```

When writing tests that interact with Claude Code configuration, always use `$CLAUDE_CONFIG_DIR` rather than hardcoding `~/.claude`.

## Worktrees

Feature development uses git worktrees in `.worktrees/` directory (already in .gitignore).

## Embedded Profiles

Built-in profiles are embedded from `internal/profile/profiles/*.json` using Go's embed directive.

## Profile Scope Awareness

claudeup respects Claude Code's scope layering system (user → project → local).

**How Claude Code works:**

- Settings files exist at three scopes: user (`~/.claude/settings.json`), project (`.claude/settings.json`), local (`.claude/settings.local.json`)
- Claude Code **accumulates** settings from all scopes: user → project → local
- Later scopes override earlier ones (local > project > user)
- The effective configuration is the combination of all three scopes

**How claudeup handles this:**

- `profile list` shows which profile is active at each scope
- `*` marker shows the highest precedence active profile (what Claude actually uses)
- `○` marker shows profiles active at lower precedence scopes (overridden)

**Example output:**

```text
Your profiles (5)

○ base-tools           Base tools [user]
* claudeup             My claudeup setup [local]
```

This shows:

- `base-tools` is active at user scope but overridden by `claudeup` at local scope
- Claude Code is actually using `claudeup` (highest precedence)

## Event Tracking & Privacy

claudeup tracks file operations in `~/.claudeup/events/operations.log` for audit trails and troubleshooting.

### Content Capture Behavior

**What is captured:**

- JSON files under 1MB have their full content stored in event snapshots
- This enables the `claudeup events diff` command to show detailed changes
- Files tracked include: settings.json, installed_plugins.json, profiles, mcp configs

**Privacy considerations:**

- ⚠️ **Event logs may contain sensitive data** if configuration files include API keys, tokens, or credentials
- Event logs are stored with 0600 permissions (owner-only access)
- Logs are stored locally at `~/.claudeup/events/operations.log`

**Recommendations:**

- Do not store secrets in Claude configuration files (use environment variables or secret managers instead)
- Review event logs before sharing for debugging: `cat ~/.claudeup/events/operations.log`
- Event log retention can be configured in `~/.claudeup/config.json` (future feature)

**Disabling event tracking:**
Set `monitoring.enabled: false` in `~/.claudeup/config.json` to disable all event tracking.

### Viewing Event Diffs

The `claudeup events diff` command shows what changed in file operations:

```bash
# Show most recent change (truncated for readability)
claudeup events diff --file ~/.claude/plugins/installed_plugins.json

# Show full details with deep diff (recommended for debugging)
claudeup events diff --file ~/.claude/plugins/installed_plugins.json --full
```

**Default mode** (truncated):

- Nested objects shown as `{...}` to prevent terminal overflow
- Good for quick overview of changes

**Full mode** (`--full` flag):

- Recursively diffs nested objects showing only changed fields
- Color-coded symbols: 🟢 `+` added, 🔴 `-` removed, 🔵 `~` modified
- Bold key names with gray `(added)`/`(removed)` labels
- Ideal for understanding complex configuration changes

**Example output:**

```text
~ plugins:
  ~ conductor@claude-conductor:
    ~ scope: "project" → "user"
    ~ installedAt: "2025-12-26T05:14:20.184Z" → "2025-12-26T19:11:07.257Z"
  ~ backend-api-security@claude-code-workflows:
    - projectPath: "/Users/markalston/workspace/claudeup" (removed)
```

## Claude CLI Format Compatibility

claudeup parses Claude CLI's internal JSON files (`installed_plugins.json`, `settings.json`). To protect against format changes:

### Runtime Protection

- **Schema validation** - All JSON parsing includes validation that fails loudly on unknown formats
- **Structured errors** - Clear error messages guide users to update or report issues
- **Path detection** - Distinguishes between "Claude not installed" and "file paths changed"

### Development Protection

- **Smoke tests** - `test/integration/claude/format_compatibility_test.go` tests against your real `~/.claude/` directory
- **Pre-commit hook** - Optional hook runs smoke tests when `internal/claude/` changes
- **Early detection** - Catch format changes during development, not from user reports

### When Claude CLI Format Changes

1. **Smoke tests fail** - You'll see failures when your local Claude updates
2. **Investigate changes** - Examine actual file structure: `cat ~/.claude/plugins/installed_plugins.json | jq .`
3. **Update validation** - Add new version support in `internal/claude/validation.go`
4. **Update migration** - Extend `LoadPlugins()` to handle new version
5. **Update error messages** - Change supported version range in validation

See `plans/2025-12-17-claude-format-resilience-design.md` for full architecture details.
