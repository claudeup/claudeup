# Simplify claudeup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove sandbox, .claudeup.json auto-detection, and drift detection to focus claudeup on its core value: profile management for bootstrapping.

**Architecture:** This is a removal/simplification. We'll delete code in dependency order: sandbox first (uses .claudeup.json), then .claudeup.json handling, then drift detection. Each removal is self-contained with its own tests cleanup.

**Tech Stack:** Go, Cobra CLI, Ginkgo/Gomega testing

**Issue:** https://github.com/claudeup/claudeup/issues/116

---

## Phase 1: Remove Sandbox

Sandbox is isolated and has the fewest dependencies on other code being removed.

### Task 1.1: Remove Sandbox Command

**Files:**

- Delete: `internal/commands/sandbox.go`
- Modify: `internal/commands/root.go` (remove sandbox command registration)

**Step 1: Find and remove sandbox command registration**

```bash
grep -n "sandbox" internal/commands/root.go
```

**Step 2: Edit root.go to remove sandbox**

Remove the line that adds the sandbox command to root (likely `rootCmd.AddCommand(sandboxCmd)`).

**Step 3: Delete sandbox command file**

```bash
rm internal/commands/sandbox.go
```

**Step 4: Verify build**

```bash
go build ./cmd/claudeup
```

Expected: Build succeeds (sandbox package not yet deleted but command removed)

**Step 5: Commit**

```bash
git add internal/commands/sandbox.go internal/commands/root.go
git commit -m "chore: remove sandbox command"
```

---

### Task 1.2: Remove Sandbox Package

**Files:**

- Delete: `internal/sandbox/` (entire directory)

**Step 1: Check for imports of sandbox package**

```bash
grep -r "claudeup/internal/sandbox" --include="*.go" .
```

Expected: Only the deleted sandbox.go command should have imported it

**Step 2: Delete sandbox package**

```bash
rm -rf internal/sandbox/
```

**Step 3: Verify build**

```bash
go build ./cmd/claudeup
```

Expected: Build succeeds

**Step 4: Run tests to see what breaks**

```bash
go test ./... 2>&1 | head -50
```

Expected: Some tests may fail (sandbox acceptance tests)

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove sandbox package"
```

---

### Task 1.3: Remove Sandbox Tests

**Files:**

- Delete: `test/acceptance/sandbox_autodetect_test.go`
- Delete: `test/acceptance/sandbox_copy_auth_test.go`
- Delete: `test/acceptance/sandbox_credentials_test.go`

**Step 1: Delete sandbox acceptance tests**

```bash
rm test/acceptance/sandbox_autodetect_test.go
rm test/acceptance/sandbox_copy_auth_test.go
rm test/acceptance/sandbox_credentials_test.go
```

**Step 2: Run acceptance tests**

```bash
go test ./test/acceptance/... -v 2>&1 | tail -20
```

Expected: Tests pass (sandbox tests gone)

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove sandbox acceptance tests"
```

---

### Task 1.4: Remove Sandbox from Config

**Files:**

- Modify: `internal/config/global.go` (remove Sandbox struct field)

**Step 1: Read the config file**

```bash
cat internal/config/global.go
```

**Step 2: Remove Sandbox struct and field**

Remove the `Sandbox` struct definition and any references to it in the `GlobalConfig` struct.

**Step 3: Check for usages**

```bash
grep -r "\.Sandbox" --include="*.go" .
```

Expected: No remaining usages

**Step 4: Verify build and tests**

```bash
go build ./cmd/claudeup && go test ./...
```

**Step 5: Commit**

```bash
git add internal/config/global.go
git commit -m "chore: remove Sandbox from global config"
```

---

### Task 1.5: Remove Sandbox from Profile Struct

**Files:**

- Modify: `internal/profile/profile.go` (remove Sandbox field if present)

**Step 1: Check if Profile has Sandbox field**

```bash
grep -n "Sandbox" internal/profile/profile.go
```

**Step 2: If present, remove Sandbox field from Profile struct**

Remove the field and any related types.

**Step 3: Check for usages**

```bash
grep -r "profile\.Sandbox\|\.Sandbox\." --include="*.go" internal/
```

**Step 4: Verify build and tests**

```bash
go build ./cmd/claudeup && go test ./...
```

**Step 5: Commit**

```bash
git add internal/profile/profile.go
git commit -m "chore: remove Sandbox field from Profile struct"
```

---

### Task 1.6: Remove Docker Files and Scripts

**Files:**

- Delete: `docker/Dockerfile`
- Delete: `docker/entrypoint.sh`
- Delete: `scripts/build-sandbox-image.sh`

**Step 1: Delete docker directory**

```bash
rm -rf docker/
```

**Step 2: Delete build script**

```bash
rm scripts/build-sandbox-image.sh
```

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove docker files and sandbox build script"
```

---

### Task 1.7: Remove Sandbox CI/CD

**Files:**

- Delete or modify: `.github/workflows/docker.yml`

**Step 1: Check if docker.yml only handles sandbox**

```bash
cat .github/workflows/docker.yml
```

**Step 2: If only sandbox, delete it**

```bash
rm .github/workflows/docker.yml
```

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove sandbox docker workflow"
```

---

### Task 1.8: Remove Sandbox Documentation

**Files:**

- Delete: `docs/sandbox.md`
- Delete: `docs/_site/sandbox.html` (if exists)

**Step 1: Delete sandbox docs**

```bash
rm -f docs/sandbox.md docs/_site/sandbox.html
```

**Step 2: Update docs index if needed**

Check `docs/index.md` or similar for sandbox references.

**Step 3: Commit**

```bash
git add -A
git commit -m "docs: remove sandbox documentation"
```

---

## Phase 2: Remove .claudeup.json Auto-Detection

Now that sandbox is gone (which was the main consumer of .claudeup.json auto-detection), we can remove this feature.

### Task 2.1: Remove Project Config Package

**Files:**

- Delete: `internal/profile/project_config.go`
- Delete: `test/integration/profile/project_config_test.go`

**Step 1: Check what imports project_config functions**

```bash
grep -r "ProjectConfig\|DetectProfileFromProject\|LoadProjectConfig" --include="*.go" .
```

Note all files that need updating.

**Step 2: Delete project_config.go**

```bash
rm internal/profile/project_config.go
```

**Step 3: Delete project_config tests**

```bash
rm test/integration/profile/project_config_test.go
```

**Step 4: Build to see errors**

```bash
go build ./cmd/claudeup 2>&1
```

Expected: Build fails - shows what needs updating

**Step 5: Commit deletion (we'll fix imports in next tasks)**

```bash
git add -A
git commit -m "chore: remove project_config.go (will fix imports next)"
```

---

### Task 2.2: Update Scope Helpers

**Files:**

- Modify: `internal/commands/scope_helpers.go`

**Step 1: Read scope_helpers.go**

```bash
cat internal/commands/scope_helpers.go
```

**Step 2: Remove .claudeup.json checks from getActiveProfile()**

The function checks .claudeup.json first - remove that check. Simplify to only check user scope settings.

**Step 3: Verify build**

```bash
go build ./cmd/claudeup
```

**Step 4: Commit**

```bash
git add internal/commands/scope_helpers.go
git commit -m "chore: remove .claudeup.json from scope helpers"
```

---

### Task 2.3: Update Profile Apply Command

**Files:**

- Modify: `internal/commands/profile_cmd.go`

**Step 1: Find profile apply function**

```bash
grep -n "runProfileApply\|ProfileApply" internal/commands/profile_cmd.go
```

**Step 2: Remove project config saving logic**

Remove calls to `SaveProjectConfig()` or similar.

**Step 3: Verify build**

```bash
go build ./cmd/claudeup
```

**Step 4: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "chore: remove project config from profile apply"
```

---

### Task 2.4: Update Profile Sync Command

**Files:**

- Modify: `internal/commands/profile_cmd.go`

**Step 1: Find profile sync function**

```bash
grep -n "runProfileSync" internal/commands/profile_cmd.go
```

**Step 2: Remove .claudeup.json auto-detection from sync**

Sync should require explicit `--profile` flag or work with current profile.

**Step 3: Verify build**

```bash
go build ./cmd/claudeup
```

**Step 4: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "chore: remove .claudeup.json auto-detection from sync"
```

---

### Task 2.5: Remove Test Helper for .claudeup.json

**Files:**

- Modify: `test/helpers/testenv.go`

**Step 1: Find CreateClaudeupJSON helper**

```bash
grep -n "CreateClaudeupJSON" test/helpers/testenv.go
```

**Step 2: Remove the helper function**

**Step 3: Find tests using this helper**

```bash
grep -r "CreateClaudeupJSON" --include="*.go" test/
```

**Step 4: Update or remove those tests (handled in Task 2.6)**

**Step 5: Commit**

```bash
git add test/helpers/testenv.go
git commit -m "chore: remove CreateClaudeupJSON test helper"
```

---

### Task 2.6: Update Acceptance Tests for .claudeup.json Removal

**Files:**

- Modify: Various acceptance tests that create/use .claudeup.json

**Step 1: Find all tests referencing .claudeup.json**

```bash
grep -r "claudeup.json\|ClaudeupJSON" --include="*.go" test/acceptance/
```

**Step 2: Update each test**

For each test:

- If testing .claudeup.json behavior specifically, delete the test
- If testing profile behavior with .claudeup.json as setup, update to use explicit profile apply

**Step 3: Run acceptance tests**

```bash
go test ./test/acceptance/... -v 2>&1 | tail -30
```

**Step 4: Fix any remaining failures**

**Step 5: Commit**

```bash
git add -A
git commit -m "test: update acceptance tests for .claudeup.json removal"
```

---

## Phase 3: Remove Drift Detection

### Task 3.1: Remove Profile Diff Package

**Files:**

- Delete: `internal/profile/diff.go`

**Step 1: Check what imports diff functions**

```bash
grep -r "ProfileDiff\|CompareWith\|IsProfileModified\|HasChanges" --include="*.go" internal/commands/
```

Note all files that need updating.

**Step 2: Delete diff.go**

```bash
rm internal/profile/diff.go
```

**Step 3: Build to see errors**

```bash
go build ./cmd/claudeup 2>&1
```

Expected: Build fails - shows what needs updating

**Step 4: Commit deletion**

```bash
git add internal/profile/diff.go
git commit -m "chore: remove profile diff.go (will fix imports next)"
```

---

### Task 3.2: Update Status Command

**Files:**

- Modify: `internal/commands/status.go`

**Step 1: Find drift detection in status**

```bash
grep -n "Modified\|Drift\|diff\|CompareWith" internal/commands/status.go
```

**Step 2: Remove drift detection and warning display**

Remove the code that:

- Calls `IsProfileModifiedAtScope()`
- Shows "System differs from profile" warning
- Displays sync suggestions

Keep the basic status display (installed plugins, active profile, etc.)

**Step 3: Verify build**

```bash
go build ./cmd/claudeup
```

**Step 4: Commit**

```bash
git add internal/commands/status.go
git commit -m "chore: remove drift detection from status command"
```

---

### Task 3.3: Update Profile Status Command

**Files:**

- Modify: `internal/commands/profile_cmd.go`

**Step 1: Find profile status function**

```bash
grep -n "runProfileStatus\|profile status" internal/commands/profile_cmd.go
```

**Step 2: Remove drift detection display**

Remove:

- "Active profile's effective configuration differs" indicator
- Scope-by-scope drift summary
- Orphaned config entries display

Keep: Basic profile info display

**Step 3: Verify build**

```bash
go build ./cmd/claudeup
```

**Step 4: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "chore: remove drift detection from profile status"
```

---

### Task 3.4: Update Profile List Command

**Files:**

- Modify: `internal/commands/profile_cmd.go`

**Step 1: Find profile list display logic**

```bash
grep -n "modified\|Modified" internal/commands/profile_cmd.go
```

**Step 2: Remove (modified) indicator logic**

Remove the code that compares profiles and adds "(modified)" to the display.

**Step 3: Verify build**

```bash
go build ./cmd/claudeup
```

**Step 4: Test profile list**

```bash
./bin/claudeup profile list
```

Expected: No "(modified)" indicators

**Step 5: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "chore: remove (modified) indicator from profile list"
```

---

### Task 3.5: Clean Up Snapshot Functions

**Files:**

- Modify: `internal/profile/snapshot.go`

**Step 1: Check what snapshot functions are used**

```bash
grep -r "Snapshot\|SnapshotWithScope\|SnapshotCombined" --include="*.go" internal/commands/
```

**Step 2: Remove unused snapshot functions**

If `SnapshotWithScope()` or `SnapshotCombined()` were only used for drift detection, remove them. Keep `Snapshot()` if still needed for profile save.

**Step 3: Verify build and tests**

```bash
go build ./cmd/claudeup && go test ./...
```

**Step 4: Commit**

```bash
git add internal/profile/snapshot.go
git commit -m "chore: remove unused snapshot scope functions"
```

---

### Task 3.6: Remove Drift Detection Tests

**Files:**

- Delete: `test/acceptance/profile_modification_test.go`
- Delete: `test/acceptance/profile_diff_builtin_test.go`
- Delete: `test/acceptance/status_drift_test.go`
- Delete: `test/acceptance/status_scope_test.go`
- Delete: `test/acceptance/profile_status_test.go`
- Delete: `test/acceptance/profile_status_multiscope_test.go`

**Step 1: Delete drift-related acceptance tests**

```bash
rm test/acceptance/profile_modification_test.go
rm test/acceptance/profile_diff_builtin_test.go
rm test/acceptance/status_drift_test.go
rm test/acceptance/status_scope_test.go
rm test/acceptance/profile_status_test.go
rm test/acceptance/profile_status_multiscope_test.go
```

**Step 2: Run remaining acceptance tests**

```bash
go test ./test/acceptance/... -v 2>&1 | tail -30
```

**Step 3: Commit**

```bash
git add -A
git commit -m "test: remove drift detection acceptance tests"
```

---

## Phase 4: Documentation Cleanup

### Task 4.1: Update Project CLAUDE.md

**Files:**

- Modify: `CLAUDE.md`

**Step 1: Remove these sections from CLAUDE.md:**

- "Sandbox" section entirely
- "Sandbox Profile Auto-Detection" subsection
- References to .claudeup.json
- "(modified)" indicator documentation in "Profile Scope Awareness"

**Step 2: Update intro line**

Change:

```
CLI tool for managing Claude Code configurations, profiles, and sandboxed environments.
```

To:

```
CLI tool for managing Claude Code profiles and configurations.
```

**Step 3: Keep "Profile Scope Awareness" but simplify**

Remove drift detection references, keep scope layering explanation.

**Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md for simplification"
```

---

### Task 4.2: Update README.md

**Files:**

- Modify: `README.md`

**Step 1: Remove sandbox from feature list**

**Step 2: Update description**

**Step 3: Remove any sandbox examples or references**

**Step 4: Commit**

```bash
git add README.md
git commit -m "docs: update README for simplification"
```

---

### Task 4.3: Update Profiles Documentation

**Files:**

- Modify: `docs/profiles.md`

**Step 1: Remove .claudeup.json documentation**

**Step 2: Remove drift detection documentation**

**Step 3: Keep core profile save/apply/list/delete docs**

**Step 4: Commit**

```bash
git add docs/profiles.md
git commit -m "docs: simplify profiles documentation"
```

---

### Task 4.4: Update Other Documentation

**Files:**

- Modify: `docs/team-workflows.md`
- Modify: `docs/troubleshooting.md`
- Modify: `docs/file-operations.md`
- Delete: `docs/sandbox.md` (if not done in Phase 1)

**Step 1: Remove sandbox and drift references from each file**

**Step 2: Simplify team workflows to focus on profile sharing without .claudeup.json**

**Step 3: Commit**

```bash
git add docs/
git commit -m "docs: remove sandbox and drift references from documentation"
```

---

### Task 4.5: Remove Example Scripts

**Files:**

- Review and update: `examples/` directory

**Step 1: Find examples referencing sandbox or .claudeup.json**

```bash
grep -r "sandbox\|claudeup.json" examples/
```

**Step 2: Delete or update affected examples**

**Step 3: Commit**

```bash
git add examples/
git commit -m "docs: update examples for simplification"
```

---

## Phase 5: Final Verification

### Task 5.1: Full Test Suite

**Step 1: Run all tests**

```bash
go test ./... -v 2>&1 | tee test-output.txt
```

**Step 2: Check for failures**

```bash
grep -E "FAIL|Error" test-output.txt
```

Expected: No failures

**Step 3: Fix any remaining issues**

---

### Task 5.2: Build and Manual Test

**Step 1: Build fresh binary**

```bash
go build -o bin/claudeup ./cmd/claudeup
```

**Step 2: Test core commands**

```bash
./bin/claudeup profile list
./bin/claudeup profile save test-profile
./bin/claudeup profile apply test-profile
./bin/claudeup profile delete test-profile
./bin/claudeup doctor
./bin/claudeup setup --help
```

**Step 3: Verify sandbox command is gone**

```bash
./bin/claudeup sandbox
```

Expected: "unknown command 'sandbox'"

**Step 4: Verify no .claudeup.json references in help**

```bash
./bin/claudeup --help
./bin/claudeup profile --help
```

---

### Task 5.3: Clean Up Unused Code

**Step 1: Run deadcode detection**

```bash
go install golang.org/x/tools/cmd/deadcode@latest
deadcode ./...
```

**Step 2: Remove any unused functions found**

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove dead code"
```

---

### Task 5.4: Final Commit and PR

**Step 1: Squash or organize commits if needed**

**Step 2: Push branch**

```bash
git push -u origin simplify-claudeup
```

**Step 3: Create PR referencing issue #116**

```bash
gh pr create --title "Simplify claudeup: Remove sandbox and drift detection" --body "Closes #116

## Summary
- Remove sandbox command and Docker support
- Remove .claudeup.json auto-detection
- Remove drift detection / (modified) markers
- Update documentation

## Design Philosophy
claudeup is now focused on profile management for bootstrapping, not ongoing config management."
```

---

## Rollback Plan

If issues are discovered after merging:

1. **Revert the PR** - `git revert <merge-commit>`
2. **Re-add sandbox** - Can be restored from git history
3. **Create issues** for specific problems found

The removal is clean (deleting files, not modifying shared code extensively), so rollback is straightforward.
