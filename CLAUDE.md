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
