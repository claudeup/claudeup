# CLAUDEUP_HOME Environment Variable Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make claudeup respect the `CLAUDEUP_HOME` environment variable for all operations, eliminating the need to override `HOME` in scripts.

**Architecture:** Create a centralized `MustClaudeupHome()` helper that checks `CLAUDEUP_HOME` env var first, falls back to `~/.claudeup`. Update all call sites across config, backup, events, and commands packages. Update backup package API to accept `claudeupHome` instead of `homeDir`.

**Tech Stack:** Go, standard library only

**Design:** See `docs/plans/2026-01-01-claudeup-home-env-var-design.md`

---

## Task 1: Create paths.go with MustClaudeupHome

**Files:**
- Create: `internal/config/paths.go`
- Create: `internal/config/paths_test.go`

**Step 1: Write the failing test**

Create `internal/config/paths_test.go`:

```go
// ABOUTME: Tests for centralized path resolution functions
// ABOUTME: Verifies CLAUDEUP_HOME environment variable is respected

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMustClaudeupHome(t *testing.T) {
	t.Run("uses CLAUDEUP_HOME when set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "/custom/path")
		got := MustClaudeupHome()
		if got != "/custom/path" {
			t.Errorf("got %q, want /custom/path", got)
		}
	})

	t.Run("falls back to ~/.claudeup when not set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "")
		got := MustClaudeupHome()
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".claudeup")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestMustClaudeupHome -v`
Expected: FAIL with "undefined: MustClaudeupHome"

**Step 3: Write minimal implementation**

Create `internal/config/paths.go`:

```go
// ABOUTME: Centralized path resolution for claudeup directories
// ABOUTME: Respects CLAUDEUP_HOME environment variable for isolation

package config

import (
	"os"
	"path/filepath"
)

// MustClaudeupHome returns the claudeup home directory.
// Checks CLAUDEUP_HOME env var first, falls back to ~/.claudeup.
// Panics if home directory cannot be determined.
func MustClaudeupHome() string {
	if home := os.Getenv("CLAUDEUP_HOME"); home != "" {
		return home
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return filepath.Join(homeDir, ".claudeup")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestMustClaudeupHome -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/paths.go internal/config/paths_test.go
git commit -m "feat: add MustClaudeupHome helper for CLAUDEUP_HOME env var

Resolves #75 - creates centralized path resolution that checks
CLAUDEUP_HOME environment variable before falling back to ~/.claudeup"
```

---

## Task 2: Update global.go to use MustClaudeupHome

**Files:**
- Modify: `internal/config/global.go:81-84`

**Step 1: No new test needed**

Existing tests cover config loading/saving behavior.

**Step 2: Update configPath function**

In `internal/config/global.go`, replace the `configPath()` function (lines 80-84):

Before:
```go
// configPath returns the path to the global config file
func configPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".claudeup", "config.json")
}
```

After:
```go
// configPath returns the path to the global config file
func configPath() string {
	return filepath.Join(MustClaudeupHome(), "config.json")
}
```

**Step 3: Run tests to verify nothing broke**

Run: `go test ./internal/config -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/config/global.go
git commit -m "refactor: use MustClaudeupHome in config.configPath"
```

---

## Task 3: Update projects.go to use MustClaudeupHome

**Files:**
- Modify: `internal/config/projects.go:96-99`

**Step 1: Update projectsPath function**

In `internal/config/projects.go`, replace the `projectsPath()` function (lines 96-99):

Before:
```go
func projectsPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".claudeup", ProjectsFile)
}
```

After:
```go
func projectsPath() string {
	return filepath.Join(MustClaudeupHome(), ProjectsFile)
}
```

**Step 2: Run tests to verify nothing broke**

Run: `go test ./internal/config -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/config/projects.go
git commit -m "refactor: use MustClaudeupHome in config.projectsPath"
```

---

## Task 4: Update backup.go to use claudeupHome parameter

**Files:**
- Modify: `internal/backup/backup.go`
- Modify: `internal/backup/backup_test.go`

The backup package takes `homeDir` and constructs `.claudeup/backups` internally. We need to:
1. Rename parameter `homeDir` → `claudeupHome`
2. Change path from `filepath.Join(homeDir, ".claudeup", "backups")` → `filepath.Join(claudeupHome, "backups")`
3. Update validation function and error messages

**Step 1: Update backup.go**

Replace `validateHomeDir` with `validateClaudeupHome`:

```go
// validateClaudeupHome ensures claudeupHome is an absolute path that exists
func validateClaudeupHome(claudeupHome string) error {
	if !filepath.IsAbs(claudeupHome) {
		return fmt.Errorf("claudeupHome must be an absolute path: %s", claudeupHome)
	}
	// Note: We don't require the directory to exist - it will be created on first use
	return nil
}
```

Update `EnsureBackupDir`:

Before:
```go
func EnsureBackupDir(homeDir string) (string, error) {
	if err := validateHomeDir(homeDir); err != nil {
		return "", err
	}
	backupDir := filepath.Join(homeDir, ".claudeup", "backups")
```

After:
```go
func EnsureBackupDir(claudeupHome string) (string, error) {
	if err := validateClaudeupHome(claudeupHome); err != nil {
		return "", err
	}
	backupDir := filepath.Join(claudeupHome, "backups")
```

Update all other functions similarly:
- `SaveScopeBackup(claudeupHome, scope, settingsPath)`
- `SaveLocalScopeBackup(claudeupHome, projectDir, settingsPath)`
- `RestoreScopeBackup(claudeupHome, scope, settingsPath)`
- `RestoreLocalScopeBackup(claudeupHome, projectDir, settingsPath)`
- `GetBackupInfo(claudeupHome, scope)`
- `GetLocalBackupInfo(claudeupHome, projectDir)`

Each function:
1. Rename first parameter from `homeDir` to `claudeupHome`
2. Change `validateHomeDir` call to `validateClaudeupHome`
3. Change `filepath.Join(homeDir, ".claudeup", "backups")` to `filepath.Join(claudeupHome, "backups")`

**Step 2: Update backup_test.go**

Update test setup to pass `claudeupHome` (the `.claudeup` directory) instead of `homeDir`:

Before (example):
```go
backupDir := filepath.Join(tempDir, ".claudeup", "backups")
```

After:
```go
claudeupHome := filepath.Join(tempDir, ".claudeup")
backupDir := filepath.Join(claudeupHome, "backups")
```

And update function calls:
```go
// Before
backup.EnsureBackupDir(tempDir)
// After
backup.EnsureBackupDir(claudeupHome)
```

**Step 3: Run tests to verify**

Run: `go test ./internal/backup -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/backup/backup.go internal/backup/backup_test.go
git commit -m "refactor: backup package uses claudeupHome instead of homeDir

API change: all backup functions now take claudeupHome (the .claudeup
directory) instead of homeDir. This enables CLAUDEUP_HOME env var support."
```

---

## Task 5: Update events/global.go to use config.MustClaudeupHome

**Files:**
- Modify: `internal/events/global.go:34-36`

**Step 1: Add import for config package**

Add to imports:
```go
"github.com/claudeup/claudeup/internal/config"
```

**Step 2: Update initializeGlobalTracker**

Before:
```go
func initializeGlobalTracker() *Tracker {
	homeDir, _ := os.UserHomeDir()
	eventsDir := filepath.Join(homeDir, ".claudeup", "events")
```

After:
```go
func initializeGlobalTracker() *Tracker {
	eventsDir := filepath.Join(config.MustClaudeupHome(), "events")
```

Also remove unused `"os"` import if no longer needed.

**Step 3: Run tests**

Run: `go test ./internal/events -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/events/global.go
git commit -m "refactor: use config.MustClaudeupHome in events package"
```

---

## Task 6: Update commands/events.go

**Files:**
- Modify: `internal/commands/events.go:53-57`

**Step 1: Update eventsListCmd run function**

Before:
```go
homeDir, err := os.UserHomeDir()
if err != nil {
	return fmt.Errorf("failed to get home directory: %w", err)
}
eventsDir := filepath.Join(homeDir, ".claudeup", "events")
```

After:
```go
eventsDir := filepath.Join(config.MustClaudeupHome(), "events")
```

Remove unused error handling for UserHomeDir.

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/events.go
git commit -m "refactor: use config.MustClaudeupHome in events command"
```

---

## Task 7: Update commands/events_diff.go

**Files:**
- Modify: `internal/commands/events_diff.go:55-59`

**Step 1: Update eventsDiffCmd run function**

Same pattern as Task 6.

Before:
```go
homeDir, err := os.UserHomeDir()
if err != nil {
	return fmt.Errorf("failed to get home directory: %w", err)
}
eventsDir := filepath.Join(homeDir, ".claudeup", "events")
```

After:
```go
eventsDir := filepath.Join(config.MustClaudeupHome(), "events")
```

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/events_diff.go
git commit -m "refactor: use config.MustClaudeupHome in events diff command"
```

---

## Task 8: Update commands/events_audit.go

**Files:**
- Modify: `internal/commands/events_audit.go:58-62`

**Step 1: Update eventsAuditCmd run function**

Same pattern as Tasks 6-7.

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/events_audit.go
git commit -m "refactor: use config.MustClaudeupHome in events audit command"
```

---

## Task 9: Update commands/status.go

**Files:**
- Modify: `internal/commands/status.go:87-88`
- Modify: `internal/commands/status.go:309-310`

**Step 1: Update both occurrences**

There are two places using `os.UserHomeDir()` for `.claudeup/profiles`:

Line ~87-88:
```go
// Before
homeDir, _ := os.UserHomeDir()
profilesDir := filepath.Join(homeDir, ".claudeup", "profiles")

// After
profilesDir := filepath.Join(config.MustClaudeupHome(), "profiles")
```

Line ~309-310:
```go
// Before
homeDir, _ := os.UserHomeDir()
profilesDir := filepath.Join(homeDir, ".claudeup", "profiles")

// After
profilesDir := filepath.Join(config.MustClaudeupHome(), "profiles")
```

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/status.go
git commit -m "refactor: use config.MustClaudeupHome in status command"
```

---

## Task 10: Update commands/plugin.go

**Files:**
- Modify: `internal/commands/plugin.go:207-218`

**Step 1: Update profilesDir construction**

Before:
```go
homeDir, err := os.UserHomeDir()
if err != nil {
	return fmt.Errorf("failed to get home directory: %w", err)
}
// ...
profilesDir := filepath.Join(homeDir, ".claudeup", "profiles")
```

After:
```go
profilesDir := filepath.Join(config.MustClaudeupHome(), "profiles")
```

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/plugin.go
git commit -m "refactor: use config.MustClaudeupHome in plugin command"
```

---

## Task 11: Update commands/setup.go

**Files:**
- Modify: `internal/commands/setup.go:322-324`

**Step 1: Update getProfilesDir function**

Before:
```go
func getProfilesDir() string {
	return filepath.Join(profile.MustHomeDir(), ".claudeup", "profiles")
}
```

After:
```go
func getProfilesDir() string {
	return filepath.Join(config.MustClaudeupHome(), "profiles")
}
```

Add import for config package if not present.

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/setup.go
git commit -m "refactor: use config.MustClaudeupHome in setup command"
```

---

## Task 12: Update commands/sandbox.go

**Files:**
- Modify: `internal/commands/sandbox.go:70`

**Step 1: Update runSandbox function**

Before:
```go
claudePMDir := filepath.Join(profile.MustHomeDir(), ".claudeup")
```

After:
```go
claudePMDir := config.MustClaudeupHome()
```

Add import for config package.

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/sandbox.go
git commit -m "refactor: use config.MustClaudeupHome in sandbox command"
```

---

## Task 13: Update commands/scope.go backup calls

**Files:**
- Modify: `internal/commands/scope.go:327-336`
- Modify: `internal/commands/scope.go:415-431`
- Modify: `internal/commands/scope.go:466-468`

**Step 1: Update all backup function calls**

For each occurrence, change from `os.UserHomeDir()` to `config.MustClaudeupHome()`:

Line ~327:
```go
// Before
homeDir, err := os.UserHomeDir()
if err != nil {
	return fmt.Errorf("failed to get home directory: %w", err)
}
// ...
backupPath, err = backup.SaveLocalScopeBackup(homeDir, projectDir, settingsPath)
// ...
backupPath, err = backup.SaveScopeBackup(homeDir, scope, settingsPath)

// After
claudeupHome := config.MustClaudeupHome()
// ...
backupPath, err = backup.SaveLocalScopeBackup(claudeupHome, projectDir, settingsPath)
// ...
backupPath, err = backup.SaveScopeBackup(claudeupHome, scope, settingsPath)
```

Similar updates for lines ~415-431 and ~466-468.

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/scope.go
git commit -m "refactor: use config.MustClaudeupHome in scope command backup calls"
```

---

## Task 14: Update commands/profile_cmd.go backup calls

**Files:**
- Modify: `internal/commands/profile_cmd.go:702-713`

**Step 1: Update backup function calls**

Before:
```go
homeDir, err := os.UserHomeDir()
if err != nil {
	return fmt.Errorf("failed to get home directory: %w", err)
}
// ...
backupPath, err = backup.SaveLocalScopeBackup(homeDir, cwd, settingsPath)
// ...
backupPath, err = backup.SaveScopeBackup(homeDir, scopeStr, settingsPath)
```

After:
```go
claudeupHome := config.MustClaudeupHome()
// ...
backupPath, err = backup.SaveLocalScopeBackup(claudeupHome, cwd, settingsPath)
// ...
backupPath, err = backup.SaveScopeBackup(claudeupHome, scopeStr, settingsPath)
```

**Step 2: Run tests**

Run: `go test ./internal/commands -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "refactor: use config.MustClaudeupHome in profile command backup calls"
```

---

## Task 15: Remove HOME override from examples/lib/common.sh

**Files:**
- Modify: `examples/lib/common.sh:182-203`

**Step 1: Update setup_temp_claude_dir function**

Remove the HOME override and update comments:

Before:
```bash
    # Save real HOME for any operations that need it
    EXAMPLE_REAL_HOME="$HOME"
    export EXAMPLE_REAL_HOME

    # Override HOME so claudeup's config.json is isolated
    # This is necessary because claudeup reads ~/.claudeup/config.json
    # which contains the active profile setting
    export HOME="$EXAMPLE_TEMP_DIR"
    export CLAUDE_CONFIG_DIR="$EXAMPLE_TEMP_DIR/.claude"
    export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/.claudeup"
```

After:
```bash
    # Set isolated directories for Claude and claudeup
    export CLAUDE_CONFIG_DIR="$EXAMPLE_TEMP_DIR/.claude"
    export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/.claudeup"
```

Also update the info output at the end of the function:
```bash
    success "Created isolated environment: $EXAMPLE_TEMP_DIR"
    info "CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR"
    info "CLAUDEUP_HOME=$CLAUDEUP_HOME"
```

**Step 2: Test a script works**

Run: `./examples/getting-started/01-check-installation.sh --non-interactive`
Expected: Script runs successfully with isolated environment

**Step 3: Commit**

```bash
git add examples/lib/common.sh
git commit -m "chore: remove HOME override from example scripts

Now that claudeup respects CLAUDEUP_HOME, we no longer need to
override HOME to achieve isolation in example scripts."
```

---

## Task 16: Remove HOME override from bob-syncs-profile.sh

**Files:**
- Modify: `~/workspace/claudeup-test-repos/scripts/bob-syncs-profile.sh:72-84`

**Step 1: Update setup_temp_env function**

Before:
```bash
setup_temp_env() {
    TEMP_DIR=$(mktemp -d "/tmp/bob-sync-test-XXXXXXXXXX")

    # Set up isolated Claude environment for Bob
    export HOME="$TEMP_DIR"
    export CLAUDE_CONFIG_DIR="$TEMP_DIR/.claude"
    export CLAUDEUP_HOME="$TEMP_DIR/.claudeup"
```

After:
```bash
setup_temp_env() {
    TEMP_DIR=$(mktemp -d "/tmp/bob-sync-test-XXXXXXXXXX")

    # Set up isolated Claude environment for Bob
    export CLAUDE_CONFIG_DIR="$TEMP_DIR/.claude"
    export CLAUDEUP_HOME="$TEMP_DIR/.claudeup"
```

**Step 2: Run the integration test**

Run: `NON_INTERACTIVE=true ~/workspace/claudeup-test-repos/scripts/bob-syncs-profile.sh`
Expected: All tests pass

**Step 3: Commit**

```bash
cd ~/workspace/claudeup-test-repos
git add scripts/bob-syncs-profile.sh
git commit -m "chore: remove HOME override - claudeup now respects CLAUDEUP_HOME"
```

---

## Task 17: Run full test suite

**Step 1: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 2: Build and verify**

Run: `go build -o bin/claudeup ./cmd/claudeup`

**Step 3: Run integration script**

Run: `NON_INTERACTIVE=true ~/workspace/claudeup-test-repos/scripts/bob-syncs-profile.sh`
Expected: All tests pass, confirming end-to-end behavior

---

## Task 18: Final commit and cleanup

**Step 1: Verify all changes are committed**

Run: `git status`
Expected: Clean working directory

**Step 2: Squash or organize commits if needed**

Review commit history and ensure it tells a clear story.

---

## Summary of Changes

| Package | File | Change |
|---------|------|--------|
| config | paths.go | New - MustClaudeupHome() |
| config | global.go | Use MustClaudeupHome() |
| config | projects.go | Use MustClaudeupHome() |
| backup | backup.go | Rename homeDir→claudeupHome, update paths |
| events | global.go | Use config.MustClaudeupHome() |
| commands | events.go | Use config.MustClaudeupHome() |
| commands | events_diff.go | Use config.MustClaudeupHome() |
| commands | events_audit.go | Use config.MustClaudeupHome() |
| commands | status.go | Use config.MustClaudeupHome() |
| commands | plugin.go | Use config.MustClaudeupHome() |
| commands | setup.go | Use config.MustClaudeupHome() |
| commands | sandbox.go | Use config.MustClaudeupHome() |
| commands | scope.go | Use config.MustClaudeupHome() for backup calls |
| commands | profile_cmd.go | Use config.MustClaudeupHome() for backup calls |
| scripts | common.sh | Remove HOME override |
| scripts | bob-syncs-profile.sh | Remove HOME override |
