# claudeup

CLI tool for managing Claude Code configurations, profiles, and sandboxed environments.

## Project Structure

- `cmd/claudeup/` - Main entry point
- `internal/commands/` - Cobra command implementations
- `internal/profile/` - Profile management (save, load, apply, snapshot)
- `internal/claude/` - Claude Code configuration file handling
- `internal/sandbox/` - Docker-based sandboxed execution
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

## Worktrees

Feature development uses git worktrees in `.worktrees/` directory (already in .gitignore).

## Embedded Profiles

Built-in profiles are embedded from `internal/profile/profiles/*.json` using Go's embed directive.

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
