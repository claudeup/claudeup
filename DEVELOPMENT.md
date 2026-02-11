# Development

## Prerequisites

- Go 1.25.1+ (see `go.mod` for exact version)
- Git

## Install

```bash
# One-liner install (macOS/Linux)
curl -fsSL https://claudeup.github.io/install.sh | bash

# Or from source
go install github.com/claudeup/claudeup/v5/cmd/claudeup@latest

# Or update an existing installation
claudeup update
```

## Build from Source

```bash
git clone https://github.com/claudeup/claudeup.git
cd claudeup
go build -o bin/claudeup ./cmd/claudeup
```

## Tests

Tests use [Ginkgo](https://onsi.github.io/ginkgo/) BDD framework with [Gomega](https://onsi.github.io/gomega/) matchers.

```bash
# Run all tests
go test ./...

# Unit tests only
go test ./internal/...

# Integration tests (internal packages with fake fixtures)
go test ./test/integration/... -v

# Acceptance tests (execute the real binary in isolated temp dirs)
go test ./test/acceptance/... -v
```

Optional: use the Ginkgo CLI for better output:

```bash
go run github.com/onsi/ginkgo/v2/ginkgo -v ./test/...
```

### Test types

- **Unit tests** (`internal/*/`) -- Standard Go tests for individual functions.
- **Integration tests** (`test/integration/`) -- Test internal packages with fake Claude installations. No binary execution.
- **Acceptance tests** (`test/acceptance/`) -- Execute the real `claudeup` binary in isolated temp directories. Test CLI behavior end-to-end.

### Test isolation

Tests use `CLAUDE_CONFIG_DIR` and `CLAUDEUP_HOME` environment variables to avoid touching your real `~/.claude` configuration. Each test gets its own temp directory.

## Project Structure

```
cmd/claudeup/          Main entry point
internal/
  commands/            Cobra command implementations
  profile/             Profile management (save, load, apply, snapshot)
  claude/              Claude Code configuration file handling
  secrets/             Secret resolution (env, 1Password, keychain)
test/
  acceptance/          Acceptance tests (real binary execution)
  integration/         Integration tests (internal packages)
  helpers/             Shared test utilities
examples/              Example scripts with shared library
docs/                  User-facing documentation
```

## Releasing

### Patch and minor releases

1. Merge your changes to `main`
2. Tag the commit: `git tag v5.1.0`
3. Push the tag: `git push origin v5.1.0`
4. The `release.yml` workflow builds binaries and creates a GitHub release automatically

### Major releases (v5 -> v6, etc.)

Major releases require updating the Go module path across the entire codebase. This is automated:

1. Create a GitHub milestone titled `X.0.0` (e.g., `6.0.0` or `v6.0.0`)
2. Close the milestone -- this triggers the `major-release.yml` workflow
3. The workflow migrates all `/vN` references (`go.mod`, import paths, docs, error messages)
4. The workflow validates the build (`go mod tidy`, `go build`, `go vet`, all tests)
5. The workflow creates a PR for review
6. Merge the PR
7. Tag and push: `git tag v6.0.0 && git push origin v6.0.0`
8. The `release.yml` workflow handles the rest

## Environment Variables

| Variable            | Description                                    | Default       |
| ------------------- | ---------------------------------------------- | ------------- |
| `CLAUDEUP_HOME`     | Override claudeup's configuration directory    | `~/.claudeup` |
| `CLAUDE_CONFIG_DIR` | Override Claude Code's configuration directory | `~/.claude`   |
