# Untracked Scope Hints Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show warning hints in `profile list` and `profile status` when project or local scope has settings with enabled plugins but no tracked profile.

**Architecture:** Add a shared `UntrackedScopeInfo` type and `getUntrackedScopes()` helper in `scope_helpers.go`. Both `runProfileList` and `runProfileStatus` call it and render hints. A `CreateProjectScopeSettings` test helper enables writing `.claude/settings.json` inside a project directory.

**Tech Stack:** Go, Ginkgo/Gomega acceptance tests, existing `claude.LoadSettingsForScope()`

---

### Task 1: Add test helper for project-scope settings

**Files:**

- Modify: `test/helpers/testenv.go:273` (after `CreateSettings`)

**Step 1: Add `CreateProjectScopeSettings` to TestEnv**

```go
// CreateProjectScopeSettings writes a .claude/settings.json in the given project directory
func (e *TestEnv) CreateProjectScopeSettings(projectDir string, enabledPlugins map[string]bool) {
	claudeDir := filepath.Join(projectDir, ".claude")
	Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
	settings := map[string]interface{}{
		"enabledPlugins": enabledPlugins,
	}
	jsonData, err := json.MarshalIndent(settings, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(claudeDir, "settings.json"), jsonData, 0644)).To(Succeed())
}

// CreateLocalScopeSettings writes a .claude/settings.local.json in the given project directory
func (e *TestEnv) CreateLocalScopeSettings(projectDir string, enabledPlugins map[string]bool) {
	claudeDir := filepath.Join(projectDir, ".claude")
	Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
	settings := map[string]interface{}{
		"enabledPlugins": enabledPlugins,
	}
	jsonData, err := json.MarshalIndent(settings, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), jsonData, 0644)).To(Succeed())
}
```

**Step 2: Verify it compiles**

Run: `go build ./test/helpers/...`
Expected: No errors (helpers is not a test package, but verify no syntax issues with `go vet ./test/...`)

**Step 3: Commit**

```bash
git add test/helpers/testenv.go
git commit -m "test: add helpers for project and local scope settings"
```

---

### Task 2: Write failing tests for `getUntrackedScopes` helper

**Files:**

- Create: `internal/commands/scope_helpers_test.go`

The `getUntrackedScopes` function will be in the `commands` package, so we test it directly with a unit test. We need to set up `CLAUDE_CONFIG_DIR` and a project directory with `.claude/settings.json`.

**Step 1: Write the test file**

```go
package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetUntrackedScopes(t *testing.T) {
	t.Run("returns empty when no project settings exist", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()
		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 0 {
			t.Errorf("expected 0 untracked scopes, got %d", len(result))
		}
	})

	t.Run("detects project scope with enabled plugins", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		// Create .claude/settings.json in project dir
		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-a@marketplace":true,"plugin-b@marketplace":true}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 untracked scope, got %d", len(result))
		}
		if result[0].Scope != "project" {
			t.Errorf("expected scope 'project', got %q", result[0].Scope)
		}
		if result[0].PluginCount != 2 {
			t.Errorf("expected 2 plugins, got %d", result[0].PluginCount)
		}
		if result[0].SettingsFile != ".claude/settings.json" {
			t.Errorf("expected settings file '.claude/settings.json', got %q", result[0].SettingsFile)
		}
	})

	t.Run("detects local scope with enabled plugins", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-x@marketplace":true}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.local.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 untracked scope, got %d", len(result))
		}
		if result[0].Scope != "local" {
			t.Errorf("expected scope 'local', got %q", result[0].Scope)
		}
		if result[0].SettingsFile != ".claude/settings.local.json" {
			t.Errorf("expected settings file '.claude/settings.local.json', got %q", result[0].SettingsFile)
		}
	})

	t.Run("skips scope when profile is tracked there", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		// Create project-scope settings
		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-a@marketplace":true}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		// Pass tracked profiles that include project scope
		tracked := []ActiveProfileInfo{{Name: "team-profile", Scope: "project"}}
		result := getUntrackedScopes(cwd, claudeDir, tracked)
		if len(result) != 0 {
			t.Errorf("expected 0 untracked scopes when project is tracked, got %d", len(result))
		}
	})

	t.Run("skips scope when plugins are all disabled", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-a@marketplace":false}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 0 {
			t.Errorf("expected 0 untracked scopes when all plugins disabled, got %d", len(result))
		}
	})

	t.Run("detects both project and local simultaneously", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		projectSettings := `{"enabledPlugins":{"plugin-a@marketplace":true}}`
		localSettings := `{"enabledPlugins":{"plugin-b@marketplace":true}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(projectSettings), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.local.json"), []byte(localSettings), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 untracked scopes, got %d", len(result))
		}
		// Should be in order: project, local
		if result[0].Scope != "project" {
			t.Errorf("expected first scope 'project', got %q", result[0].Scope)
		}
		if result[1].Scope != "local" {
			t.Errorf("expected second scope 'local', got %q", result[1].Scope)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/commands/ -run TestGetUntrackedScopes -v`
Expected: FAIL with compilation error (function and type don't exist yet)

**Step 3: Commit**

```bash
git add internal/commands/scope_helpers_test.go
git commit -m "test: add failing tests for getUntrackedScopes"
```

---

### Task 3: Implement `getUntrackedScopes` helper

**Files:**

- Modify: `internal/commands/scope_helpers.go:80` (after `getAllActiveProfiles`)

**Step 1: Add the type and function**

Add after `getAllActiveProfiles` (after line 80):

```go
// UntrackedScopeInfo describes a scope that has settings but no tracked profile
type UntrackedScopeInfo struct {
	Scope        string // "project" or "local"
	PluginCount  int
	SettingsFile string // relative path like ".claude/settings.json"
}

// getUntrackedScopes checks project and local scopes for settings files with
// enabled plugins that have no corresponding tracked profile.
func getUntrackedScopes(cwd, claudeDir string, trackedProfiles []ActiveProfileInfo) []UntrackedScopeInfo {
	// Build set of tracked scopes
	trackedScopes := make(map[string]bool)
	for _, p := range trackedProfiles {
		trackedScopes[p.Scope] = true
	}

	var untracked []UntrackedScopeInfo
	for _, scope := range []string{"project", "local"} {
		if trackedScopes[scope] {
			continue
		}

		settings, err := claude.LoadSettingsForScope(scope, claudeDir, cwd)
		if err != nil {
			continue
		}

		count := 0
		for _, enabled := range settings.EnabledPlugins {
			if enabled {
				count++
			}
		}
		if count == 0 {
			continue
		}

		settingsFile := ".claude/settings.json"
		if scope == "local" {
			settingsFile = ".claude/settings.local.json"
		}

		untracked = append(untracked, UntrackedScopeInfo{
			Scope:        scope,
			PluginCount:  count,
			SettingsFile: settingsFile,
		})
	}

	return untracked
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/commands/ -run TestGetUntrackedScopes -v`
Expected: All 6 tests PASS

**Step 3: Commit**

```bash
git add internal/commands/scope_helpers.go
git commit -m "feat: add getUntrackedScopes helper for detecting untracked settings"
```

---

### Task 4: Write failing acceptance tests for `profile list` hints

**Files:**

- Modify: `test/acceptance/profile_list_test.go` (add new Describe block before the closing of `profile list` Describe)

**Step 1: Add test cases**

Add a new `Describe("untracked scope hints", ...)` block inside the existing `profile list` Describe, after the "scope flag without active profile" block (before the closing `})` of the `profile list` Describe at line 354):

```go
	Describe("untracked scope hints", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("hint-test")
			env.SetActiveProfile("default")
		})

		Context("with untracked project-scope settings", func() {
			BeforeEach(func() {
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
					"plugin-c@marketplace": true,
				})
			})

			It("shows warning about untracked project scope", func() {
				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("project"))
				Expect(result.Stdout).To(ContainSubstring("3 plugins"))
				Expect(result.Stdout).To(ContainSubstring("no profile tracked"))
			})

			It("shows suggested save command", func() {
				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("profile save"))
				Expect(result.Stdout).To(ContainSubstring("profile apply"))
				Expect(result.Stdout).To(ContainSubstring("--project"))
			})
		})

		Context("with untracked local-scope settings", func() {
			BeforeEach(func() {
				env.CreateLocalScopeSettings(projectDir, map[string]bool{
					"plugin-x@marketplace": true,
				})
			})

			It("shows warning about untracked local scope", func() {
				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("local"))
				Expect(result.Stdout).To(ContainSubstring("1 plugin"))
				Expect(result.Stdout).To(ContainSubstring("no profile tracked"))
			})
		})

		Context("when profile is tracked at project scope", func() {
			BeforeEach(func() {
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"plugin-a@marketplace": true,
				})
				env.CreateProfile(&profile.Profile{
					Name:        "team-profile",
					Description: "Team config",
				})
				env.RegisterProjectScope(projectDir, "team-profile")
			})

			It("does not show untracked hint for project scope", func() {
				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("no profile tracked"))
			})
		})

		Context("when filtering by scope", func() {
			BeforeEach(func() {
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"plugin-a@marketplace": true,
				})
			})

			It("does not show untracked hints when --user is specified", func() {
				result := env.RunInDir(projectDir, "profile", "list", "--user")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("no profile tracked"))
			})
		})

		Context("with no project-scope settings", func() {
			It("does not show any untracked hints", func() {
				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("no profile tracked"))
			})
		})
	})
```

**Note:** This test uses `env.RegisterProjectScope()` which doesn't exist yet. We'll add it in Task 1 alongside the settings helpers. Update Task 1 to also add:

```go
// RegisterProjectScope registers a profile at project scope for a directory
func (e *TestEnv) RegisterProjectScope(projectDir, profileName string) {
	path := filepath.Join(e.ClaudeupDir, "projects.json")

	normalizedDir := projectDir
	if resolved, err := filepath.EvalSymlinks(projectDir); err == nil {
		normalizedDir = resolved
	}

	// Read existing registry or create new
	var registry map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &registry)
	}
	if registry == nil {
		registry = map[string]interface{}{
			"version":  "1",
			"projects": map[string]interface{}{},
		}
	}

	projects := registry["projects"].(map[string]interface{})
	entry, ok := projects[normalizedDir].(map[string]interface{})
	if !ok {
		entry = map[string]interface{}{}
	}
	entry["projectProfile"] = profileName
	entry["projectAppliedAt"] = "2025-01-01T00:00:00Z"
	projects[normalizedDir] = entry

	data, err := json.MarshalIndent(registry, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(path, data, 0644)).To(Succeed())
}
```

**Step 2: Run tests to verify they fail**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "untracked scope hints" ./test/acceptance/`
Expected: FAIL (output doesn't contain hint text yet)

**Step 3: Commit**

```bash
git add test/acceptance/profile_list_test.go test/helpers/testenv.go
git commit -m "test: add failing acceptance tests for profile list untracked hints"
```

---

### Task 5: Implement hints in `runProfileList`

**Files:**

- Modify: `internal/commands/profile_cmd.go:859` (in `runProfileList`, before the footer arrows)

**Step 1: Add hint rendering**

Insert before the "reserved name warning" block (line 859), after the custom profiles section:

```go
	// Show untracked scope hints (only when not filtering by scope)
	if profileListScope == "" {
		untrackedScopes := getUntrackedScopes(cwd, claudeDir, allActiveProfiles)
		for _, us := range untrackedScopes {
			pluginWord := "plugins"
			if us.PluginCount == 1 {
				pluginWord = "plugin"
			}
			fmt.Printf("  %s %d %s in %s (no profile tracked)\n",
				ui.Warning(us.Scope+":"),
				us.PluginCount, pluginWord, us.SettingsFile)
			fmt.Printf("    %s Save with: claudeup profile save <name> && claudeup profile apply <name> --%s\n",
				ui.Muted(ui.SymbolArrow), us.Scope)
		}
		if len(untrackedScopes) > 0 {
			fmt.Println()
		}
	}
```

**Step 2: Run acceptance tests to verify they pass**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "untracked scope hints" ./test/acceptance/`
Expected: All tests PASS

**Step 3: Run full test suite**

Run: `go test ./...`
Expected: All tests PASS (no regressions)

**Step 4: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "feat: show untracked scope hints in profile list"
```

---

### Task 6: Write failing acceptance tests for `profile status` hints

**Files:**

- Create: `test/acceptance/profile_status_test.go`

**Step 1: Write the test file**

```go
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile status", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("untracked scope hints", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("status-hint-test")
			env.CreateProfile(&profile.Profile{
				Name:        "my-profile",
				Description: "Test profile",
				Plugins:     []string{"some-plugin@marketplace"},
			})
			env.SetActiveProfile("my-profile")
		})

		Context("with untracked project-scope settings", func() {
			BeforeEach(func() {
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
				})
			})

			It("shows warning about untracked project scope", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("project"))
				Expect(result.Stdout).To(ContainSubstring("2 plugins"))
				Expect(result.Stdout).To(ContainSubstring("no profile tracked"))
			})

			It("shows suggested save command", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("profile save"))
				Expect(result.Stdout).To(ContainSubstring("--project"))
			})
		})

		Context("with no untracked settings", func() {
			It("does not show untracked hints", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("no profile tracked"))
			})
		})
	})
})
```

**Step 2: Run tests to verify they fail**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "profile status.*untracked" ./test/acceptance/`
Expected: FAIL (output doesn't contain hint text yet)

**Step 3: Commit**

```bash
git add test/acceptance/profile_status_test.go
git commit -m "test: add failing acceptance tests for profile status untracked hints"
```

---

### Task 7: Implement hints in `runProfileStatus`

**Files:**

- Modify: `internal/commands/profile_cmd.go:1781` (in `runProfileStatus`, after the scope line)

**Step 1: Add hint rendering**

Insert after the active scope display block (after line 1781, the `fmt.Println()` after the scope indicator), before the "Show profile contents" comment:

```go
	// Show untracked scope hints
	untrackedScopes := getUntrackedScopes(cwd, claudeDir, allActiveProfiles)
	for _, us := range untrackedScopes {
		pluginWord := "plugins"
		if us.PluginCount == 1 {
			pluginWord = "plugin"
		}
		fmt.Printf("  %s %d %s in %s (no profile tracked)\n",
			ui.Warning(us.Scope+":"),
			us.PluginCount, pluginWord, us.SettingsFile)
		fmt.Printf("    %s Save with: claudeup profile save <name> && claudeup profile apply <name> --%s\n",
			ui.Muted(ui.SymbolArrow), us.Scope)
	}
	if len(untrackedScopes) > 0 {
		fmt.Println()
	}
```

**Step 2: Run acceptance tests to verify they pass**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "profile status.*untracked" ./test/acceptance/`
Expected: All tests PASS

**Step 3: Run full test suite**

Run: `go test ./...`
Expected: All tests PASS (no regressions)

**Step 4: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "feat: show untracked scope hints in profile status"
```

---

### Task 8: Manual verification and final commit

**Step 1: Build and test against the real claudeup repo**

Run from the claudeup project directory (which has `.claude/settings.json` with project-scope plugins):

```bash
go run ./cmd/claudeup profile list
go run ./cmd/claudeup profile status
```

Expected: Both commands show the untracked project scope hint with plugin count and save suggestion.

**Step 2: Test edge cases manually**

- Run from a directory without `.claude/`: no hints shown
- Run `profile list --user`: no hints shown (filtered)
- Run `profile list --project` from a project with untracked settings: shows "no profile is active at project scope" (existing behavior, not the new hint)

**Step 3: Run full test suite one final time**

Run: `go test ./...`
Expected: All tests PASS

**Step 4: Verify no unintended changes**

Run: `git diff --stat`
Expected: Only the planned files changed
