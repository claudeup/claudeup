# Scope-Aware Plugin Upgrade Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace user-biased PluginRegistry API with scope-explicit methods and make upgrade/outdated commands scope-aware.

**Architecture:** Remove `GetPlugin`/`GetAllPlugins`/`PluginExists`/`IsPluginInstalled` from PluginRegistry. Add scope-explicit replacements (`GetPluginAtScope`, `GetPluginsAtScopes`, `PluginExistsAtScope`, `PluginExistsAtAnyScope`, `GetPluginInstances`). Migrate all callers. Add `--all` flag to upgrade/outdated commands.

**Tech Stack:** Go 1.25, ginkgo/gomega (upgrade_test.go), stdlib testing (plugins_test.go)

**Design doc:** `docs/plans/2026-02-13-scope-aware-upgrade-design.md`

---

### Task 1: Add ScopedPlugin Type and Scope-Explicit Methods to PluginRegistry

**Files:**
- Modify: `internal/claude/plugins.go:139-227` (replace methods)
- Test: `internal/claude/plugins_test.go`

**Step 1: Write failing tests for the new methods**

Add to `internal/claude/plugins_test.go`. These tests replace `TestPluginExists`, `TestDisablePlugin` (the GetPlugin assertion), `TestEnablePlugin` (the GetPlugin assertion), and `TestPluginRegistryJSONMarshaling` (the GetPlugin assertion). Do NOT delete those tests yet -- that happens after the methods are replaced.

```go
func TestGetPluginAtScope(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {
				{Scope: "user", Version: "1.0.0"},
				{Scope: "project", Version: "2.0.0"},
			},
		},
	}

	// Get user-scoped instance
	plugin, exists := registry.GetPluginAtScope("test-plugin", "user")
	if !exists {
		t.Error("should find user-scoped instance")
	}
	if plugin.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", plugin.Version)
	}

	// Get project-scoped instance
	plugin, exists = registry.GetPluginAtScope("test-plugin", "project")
	if !exists {
		t.Error("should find project-scoped instance")
	}
	if plugin.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", plugin.Version)
	}

	// Non-existent scope returns false
	_, exists = registry.GetPluginAtScope("test-plugin", "local")
	if exists {
		t.Error("should not find local-scoped instance")
	}

	// Non-existent plugin returns false
	_, exists = registry.GetPluginAtScope("missing", "user")
	if exists {
		t.Error("should not find non-existent plugin")
	}
}

func TestGetPluginInstances(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"multi-scope": {
				{Scope: "user", Version: "1.0.0"},
				{Scope: "project", Version: "2.0.0"},
			},
			"single-scope": {
				{Scope: "user", Version: "1.0.0"},
			},
		},
	}

	instances := registry.GetPluginInstances("multi-scope")
	if len(instances) != 2 {
		t.Errorf("expected 2 instances, got %d", len(instances))
	}

	instances = registry.GetPluginInstances("single-scope")
	if len(instances) != 1 {
		t.Errorf("expected 1 instance, got %d", len(instances))
	}

	instances = registry.GetPluginInstances("missing")
	if len(instances) != 0 {
		t.Errorf("expected 0 instances, got %d", len(instances))
	}
}

func TestGetPluginsAtScopes(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"plugin-a": {
				{Scope: "user", Version: "1.0.0"},
				{Scope: "project", Version: "2.0.0"},
			},
			"plugin-b": {
				{Scope: "local", Version: "3.0.0"},
			},
		},
	}

	// User scope only -- should get plugin-a user instance
	result := registry.GetPluginsAtScopes([]string{"user"})
	if len(result) != 1 {
		t.Errorf("expected 1 result for user scope, got %d", len(result))
	}
	if result[0].Name != "plugin-a" || result[0].Version != "1.0.0" {
		t.Errorf("unexpected result: %+v", result[0])
	}

	// User + project -- should get plugin-a at both scopes
	result = registry.GetPluginsAtScopes([]string{"user", "project"})
	if len(result) != 2 {
		t.Errorf("expected 2 results for user+project, got %d", len(result))
	}

	// All scopes -- should get all 3 instances
	result = registry.GetPluginsAtScopes([]string{"user", "project", "local"})
	if len(result) != 3 {
		t.Errorf("expected 3 results for all scopes, got %d", len(result))
	}

	// Empty scopes -- should return nothing
	result = registry.GetPluginsAtScopes([]string{})
	if len(result) != 0 {
		t.Errorf("expected 0 results for empty scopes, got %d", len(result))
	}
}

func TestPluginExistsAtScope(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {
				{Scope: "user", Version: "1.0.0"},
			},
		},
	}

	if !registry.PluginExistsAtScope("test-plugin", "user") {
		t.Error("should exist at user scope")
	}
	if registry.PluginExistsAtScope("test-plugin", "project") {
		t.Error("should not exist at project scope")
	}
	if registry.PluginExistsAtScope("missing", "user") {
		t.Error("should not find non-existent plugin")
	}
}

func TestPluginExistsAtAnyScope(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {
				{Scope: "project", Version: "1.0.0"},
			},
		},
	}

	if !registry.PluginExistsAtAnyScope("test-plugin") {
		t.Error("should exist at some scope")
	}
	if registry.PluginExistsAtAnyScope("missing") {
		t.Error("should not find non-existent plugin")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/claude/ -run "TestGetPluginAtScope|TestGetPluginInstances|TestGetPluginsAtScopes|TestPluginExistsAtScope|TestPluginExistsAtAnyScope" -v`
Expected: FAIL -- methods don't exist yet

**Step 3: Implement the new methods and ScopedPlugin type**

In `internal/claude/plugins.go`, add the `ScopedPlugin` type after `PluginMetadata` (around line 30), and replace lines 139-218 (the old methods) with:

```go
// ScopedPlugin pairs a plugin name with its scope-specific metadata.
// Used by GetPluginsAtScopes to flatten the registry into a single slice.
type ScopedPlugin struct {
	Name string
	PluginMetadata
}

// GetPluginAtScope retrieves a plugin's metadata for a specific scope.
// Returns (metadata, true) if found, (zero, false) if not.
func (r *PluginRegistry) GetPluginAtScope(pluginName, scope string) (PluginMetadata, bool) {
	instances, exists := r.Plugins[pluginName]
	if !exists {
		return PluginMetadata{}, false
	}
	for _, inst := range instances {
		if inst.Scope == scope {
			return inst, true
		}
	}
	return PluginMetadata{}, false
}

// GetPluginInstances returns all scope instances for a plugin.
// Returns nil if the plugin is not in the registry.
func (r *PluginRegistry) GetPluginInstances(pluginName string) []PluginMetadata {
	return r.Plugins[pluginName]
}

// GetPluginsAtScopes returns all plugin instances installed at the given scopes.
// Each instance is paired with its plugin name.
func (r *PluginRegistry) GetPluginsAtScopes(scopes []string) []ScopedPlugin {
	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}

	var result []ScopedPlugin
	for name, instances := range r.Plugins {
		for _, inst := range instances {
			if scopeSet[inst.Scope] {
				result = append(result, ScopedPlugin{Name: name, PluginMetadata: inst})
			}
		}
	}
	return result
}

// PluginExistsAtScope checks if a plugin is installed at a specific scope
func (r *PluginRegistry) PluginExistsAtScope(pluginName, scope string) bool {
	_, exists := r.GetPluginAtScope(pluginName, scope)
	return exists
}

// PluginExistsAtAnyScope checks if a plugin is installed at any scope
func (r *PluginRegistry) PluginExistsAtAnyScope(pluginName string) bool {
	instances, exists := r.Plugins[pluginName]
	return exists && len(instances) > 0
}
```

Also remove the old methods: `GetPlugin` (lines 139-153), `GetAllPlugins` (lines 182-192), `PluginExists` (lines 209-213), `IsPluginInstalled` (lines 215-218).

Keep `SetPlugin`, `DisablePlugin`, `EnablePlugin`, `RemovePlugin` unchanged.

**Step 4: Run the new tests to verify they pass**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/claude/ -run "TestGetPluginAtScope|TestGetPluginInstances|TestGetPluginsAtScopes|TestPluginExistsAtScope|TestPluginExistsAtAnyScope" -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/claude/plugins.go internal/claude/plugins_test.go
git commit -m "Replace user-biased PluginRegistry API with scope-explicit methods"
```

Note: The build will be broken at this point because callers still reference removed methods. That's expected -- they're fixed in subsequent tasks.

---

### Task 2: Update Existing Tests to Use Scope-Explicit Methods

**Files:**
- Modify: `internal/claude/plugins_test.go`

**Step 1: Update the existing tests that reference removed methods**

Replace `GetPlugin` calls in existing tests with `GetPluginAtScope`:

In `TestDisablePlugin` (line 80):
```go
// Old: if _, exists := registry.GetPlugin("test-plugin"); exists {
// New:
if _, exists := registry.GetPluginAtScope("test-plugin", "user"); exists {
```

In `TestEnablePlugin` (line 106):
```go
// Old: plugin, exists := registry.GetPlugin("test-plugin")
// New:
plugin, exists := registry.GetPluginAtScope("test-plugin", "user")
```

In `TestLoadAndSavePlugins` (line 190):
```go
// Old: plugin, exists := loaded.GetPlugin("test-plugin@test-marketplace")
// New:
plugin, exists := loaded.GetPluginAtScope("test-plugin@test-marketplace", "user")
```

In `TestPluginRegistryJSONMarshaling` (line 295):
```go
// Old: plugin, exists := loaded.GetPlugin("test-plugin")
// New:
plugin, exists := loaded.GetPluginAtScope("test-plugin", "user")
```

Replace `PluginExists` calls:

In `TestPluginExists` (lines 131-137):
```go
// Old: if !registry.PluginExists("existing-plugin") {
// New:
if !registry.PluginExistsAtAnyScope("existing-plugin") {
    t.Error("PluginExistsAtAnyScope should return true for existing plugin")
}
// Old: if registry.PluginExists("non-existent") {
// New:
if registry.PluginExistsAtAnyScope("non-existent") {
    t.Error("PluginExistsAtAnyScope should return false for non-existent plugin")
}
```

Replace `DisablePlugin` test assertion (line 75-76) -- keep as-is, `DisablePlugin` is unchanged.

Replace `EnablePlugin` test assertion (line 103) -- keep as-is, `EnablePlugin` is unchanged.

**Step 2: Run all plugins tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/claude/ -v`
Expected: PASS

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/claude/plugins_test.go
git commit -m "Update plugin tests to use scope-explicit methods"
```

---

### Task 3: Export IsProjectContext from plugin_analysis.go

**Files:**
- Modify: `internal/claude/plugin_analysis.go:79`

**Step 1: Rename `isProjectContext` to `IsProjectContext`**

In `internal/claude/plugin_analysis.go`, change:

```go
// Old (line 79):
func isProjectContext(claudeDir, projectDir string) bool {
// New:
func IsProjectContext(claudeDir, projectDir string) bool {
```

Also update the call site within the same file (line 44):

```go
// Old:
if isProjectContext(claudeDir, projectDir) {
// New:
if IsProjectContext(claudeDir, projectDir) {
```

**Step 2: Verify the existing tests still pass**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/claude/ -v`
Expected: PASS

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/claude/plugin_analysis.go
git commit -m "Export IsProjectContext for use by commands package"
```

---

### Task 4: Migrate plugin_analysis.go Caller

**Files:**
- Modify: `internal/claude/plugin_analysis.go:147`

**Step 1: Replace PluginExists with PluginExistsAtAnyScope**

In `internal/claude/plugin_analysis.go` line 147:

```go
// Old:
if enabled && !ctx.registry.PluginExists(pluginName) {
// New:
if enabled && !ctx.registry.PluginExistsAtAnyScope(pluginName) {
```

**Step 2: Run analysis tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/claude/ -v`
Expected: PASS

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/claude/plugin_analysis.go
git commit -m "Migrate plugin_analysis orphan detection to PluginExistsAtAnyScope"
```

---

### Task 5: Migrate profile/apply_concurrent.go Caller

**Files:**
- Modify: `internal/profile/apply_concurrent.go:63`

**Step 1: Replace PluginExists with PluginExistsAtScope**

The `ConcurrentApplyOptions` already has a `Scope` field (line 17). Use it:

```go
// Old (line 63):
if opts.Reinstall || currentPlugins == nil || !currentPlugins.PluginExists(plugin) {
// New:
if opts.Reinstall || currentPlugins == nil || !currentPlugins.PluginExistsAtScope(plugin, opts.Scope) {
```

**Step 2: Run profile tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/profile/ -v`
Expected: PASS

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/profile/apply_concurrent.go
git commit -m "Migrate profile apply to PluginExistsAtScope"
```

---

### Task 6: Migrate mcp/discovery.go Caller

**Files:**
- Modify: `internal/mcp/discovery.go:43`

**Step 1: Replace GetAllPlugins with GetPluginsAtScopes**

```go
// Old (line 43):
for name, plugin := range pluginRegistry.GetAllPlugins() {
// New:
for _, sp := range pluginRegistry.GetPluginsAtScopes(claude.ValidScopes) {
    name := sp.Name
    plugin := sp.PluginMetadata
```

The rest of the loop body remains unchanged since `name` and `plugin` variables have the same types.

**Step 2: Run MCP tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/mcp/ -v`
Expected: PASS

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/mcp/discovery.go
git commit -m "Migrate MCP discovery to GetPluginsAtScopes across all scopes"
```

---

### Task 7: Migrate plugin.go and plugin_search.go Callers

**Files:**
- Modify: `internal/commands/plugin.go:296,358,396`
- Modify: `internal/commands/plugin_search.go:101`

**Step 1: Replace PluginExists with PluginExistsAtAnyScope**

In `internal/commands/plugin.go`:

Line 296:
```go
// Old:
if installed != nil && installed.PluginExists(fullName) {
// New:
if installed != nil && installed.PluginExistsAtAnyScope(fullName) {
```

Line 358:
```go
// Old:
if installed != nil && installed.PluginExists(fullName) {
// New:
if installed != nil && installed.PluginExistsAtAnyScope(fullName) {
```

Line 396:
```go
// Old:
Installed:   installed != nil && installed.PluginExists(fullName),
// New:
Installed:   installed != nil && installed.PluginExistsAtAnyScope(fullName),
```

In `internal/commands/plugin_search.go` line 101:
```go
// Old:
if installed.PluginExists(fullName) {
// New:
if installed.PluginExistsAtAnyScope(fullName) {
```

**Step 2: Verify build compiles**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go build ./...`
Expected: May still fail due to remaining unmigrated callers, but these files should have no errors.

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/commands/plugin.go internal/commands/plugin_search.go
git commit -m "Migrate plugin browse/search to PluginExistsAtAnyScope"
```

---

### Task 8: Migrate doctor.go Callers

**Files:**
- Modify: `internal/commands/doctor.go:114,242`

**Step 1: Replace GetAllPlugins with scope-explicit calls**

Line 114 -- detecting plugins enabled but not installed:
```go
// Old:
if _, installed := plugins.GetAllPlugins()[name]; !installed {
// New:
if !plugins.PluginExistsAtAnyScope(name) {
```

Line 242 -- `analyzePathIssues` function iterating all plugins:
```go
// Old:
for name, plugin := range plugins.GetAllPlugins() {
// New:
for _, sp := range plugins.GetPluginsAtScopes(claude.ValidScopes) {
    name := sp.Name
    plugin := sp.PluginMetadata
```

Note: You'll need to add `"github.com/claudeup/claudeup/v5/internal/claude"` to the imports if not already there. Check the existing imports first.

**Step 2: Run doctor tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/commands/ -run TestDoctor -v`
Expected: PASS (or no test matches -- doctor may not have dedicated tests)

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/commands/doctor.go
git commit -m "Migrate doctor to scope-explicit plugin methods"
```

---

### Task 9: Migrate status.go Callers

**Files:**
- Modify: `internal/commands/status.go:137,165,178`

**Step 1: Replace GetAllPlugins with GetPluginsAtScopes**

The status command already has scope detection logic (lines 121-128). It builds a `scopes` variable. Use it.

Line 137 -- building pluginScopes map:
```go
// Old:
for name := range plugins.GetAllPlugins() {
// New:
for _, sp := range plugins.GetPluginsAtScopes(scopes) {
    name := sp.Name
```

Line 165 -- checking installed plugins for issues:
```go
// Old:
for name, plugin := range plugins.GetAllPlugins() {
// New:
for _, sp := range plugins.GetPluginsAtScopes(scopes) {
    name := sp.Name
    plugin := sp.PluginMetadata
```

Line 178 -- finding plugins enabled but not installed:
```go
// Old:
if _, installed := plugins.GetAllPlugins()[name]; !installed {
// New:
if !plugins.PluginExistsAtAnyScope(name) {
```

**Step 2: Verify build compiles**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go build ./internal/commands/`
Expected: May still fail due to remaining unmigrated callers

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/commands/status.go
git commit -m "Migrate status to scope-aware plugin iteration"
```

---

### Task 10: Migrate cleanup.go Callers

**Files:**
- Modify: `internal/commands/cleanup.go:92,191,193`

**Step 1: Replace scope-blind calls**

Line 92 -- detecting missing plugins:
```go
// Old:
if _, installed := plugins.GetAllPlugins()[name]; !installed {
// New:
if !plugins.PluginExistsAtAnyScope(name) {
```

Lines 191-193 -- fixing path issues. The `PathIssue` struct needs a `Scope` field so we know which scope's plugin to update. First, find the `PathIssue` definition and add `Scope`:

```go
// Add to PathIssue struct:
Scope        string
```

Then update `analyzePathIssues` (which is in `doctor.go`, already migrated in Task 8) to populate the scope field. Since Task 8 changed it to iterate `ScopedPlugin`, add:

```go
// In analyzePathIssues, after the ScopedPlugin loop variable setup:
// Add to each PathIssue creation:
Scope: sp.Scope,
```

Then update cleanup.go lines 191-193:
```go
// Old:
if plugin, exists := plugins.GetPlugin(issue.PluginName); exists {
    plugin.InstallPath = issue.ExpectedPath
    plugins.SetPlugin(issue.PluginName, plugin)
// New:
if plugin, exists := plugins.GetPluginAtScope(issue.PluginName, issue.Scope); exists {
    plugin.InstallPath = issue.ExpectedPath
    plugins.SetPlugin(issue.PluginName, plugin)
```

**Step 2: Run cleanup tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/commands/ -run TestCleanup -v`
Expected: PASS (or no test matches)

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/commands/cleanup.go internal/commands/doctor.go
git commit -m "Migrate cleanup to scope-explicit plugin lookups"
```

---

### Task 11: Migrate profile_cmd.go Caller

**Files:**
- Modify: `internal/commands/profile_cmd.go:1244-1246`

**Step 1: Replace GetAllPlugins in cleanupStalePlugins**

```go
// Old (lines 1244-1246):
for name, plugin := range plugins.GetAllPlugins() {
    if !plugin.PathExists() {
        if plugins.DisablePlugin(name) {
// New:
for _, sp := range plugins.GetPluginsAtScopes(claude.ValidScopes) {
    if !sp.PathExists() {
        if plugins.DisablePlugin(sp.Name) {
```

Note: `DisablePlugin` removes all scope instances for a plugin name, which is correct for stale cleanup.

**Step 2: Verify build compiles**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go build ./internal/commands/`
Expected: Should compile now -- all callers migrated

**Step 3: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/commands/profile_cmd.go
git commit -m "Migrate profile cleanup to scope-aware plugin iteration"
```

---

### Task 12: Full Build Verification

**Files:** None (verification only)

**Step 1: Verify the full project builds**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go build ./...`
Expected: PASS -- no compilation errors

**Step 2: Run all tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./... -v`
Expected: PASS

**Step 3: Commit (if any fixes needed)**

If any tests required fixes, commit those fixes here.

---

### Task 13: Add --all Flag and Scope-Aware Upgrade

**Files:**
- Modify: `internal/commands/upgrade.go`
- Test: `internal/commands/upgrade_test.go`

**Step 1: Write failing test for --all flag parsing**

Add to `internal/commands/upgrade_test.go`:

```go
var _ = Describe("availableScopes", func() {
	It("returns all scopes when allFlag is true", func() {
		scopes := availableScopes(true)
		Expect(scopes).To(Equal(claude.ValidScopes))
	})
})
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/commands/ -run "TestUpgradeCommands/availableScopes" -v`
Expected: FAIL -- `availableScopes` doesn't exist

**Step 3: Implement the scope-aware upgrade**

In `internal/commands/upgrade.go`:

Add the `--all` flag and `availableScopes` helper:

```go
var upgradeAll bool

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&upgradeAll, "all", false, "Upgrade plugins across all scopes, not just the current context")
}

func availableScopes(allFlag bool) []string {
	if allFlag {
		return claude.ValidScopes
	}
	scopes := []string{"user"}
	if claude.IsProjectContext(claudeDir, projectDir) {
		scopes = append(scopes, "project", "local")
	}
	return scopes
}
```

Add `Scope` field to `PluginUpdate`:

```go
type PluginUpdate struct {
	Name          string
	Scope         string
	HasUpdate     bool
	CurrentCommit string
	LatestCommit  string
}
```

Refactor `runUpgrade` to pass scopes:

```go
// In runUpgrade, after loading plugins (line 129):
scopes := availableScopes(upgradeAll)

// Change line 171:
// Old:
fmt.Println(ui.RenderSection("Checking Plugins", len(plugins.GetAllPlugins())))
// New:
scopedPlugins := plugins.GetPluginsAtScopes(scopes)
fmt.Println(ui.RenderSection("Checking Plugins", len(scopedPlugins)))

// Change line 172:
// Old:
pluginUpdates := checkPluginUpdates(plugins, marketplaces)
// New:
pluginUpdates := checkPluginUpdates(plugins, marketplaces, scopes)
```

Update output format to show scope (in the `pluginUpdates` loop):

```go
// Old format:
fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), update.Name, ui.Warning("Update available"))
// New format:
fmt.Printf("  %s %s (%s): %s\n", ui.Warning(ui.SymbolWarning), update.Name, update.Scope, ui.Warning("Update available"))
```

Apply same scope label to the "Update available (skipped)" and "Up to date" lines.

Refactor `checkPluginUpdates` to accept scopes:

```go
func checkPluginUpdates(plugins *claude.PluginRegistry, marketplaces claude.MarketplaceRegistry, scopes []string) []PluginUpdate {
	var updates []PluginUpdate

	for _, sp := range plugins.GetPluginsAtScopes(scopes) {
		name := sp.Name
		plugin := sp.PluginMetadata

		if !plugin.PathExists() {
			continue
		}

		// Find the marketplace this plugin belongs to
		var marketplacePath string
		for _, marketplace := range marketplaces {
			if strings.Contains(plugin.InstallPath, marketplace.InstallLocation) {
				marketplacePath = marketplace.InstallLocation
				break
			}
		}

		if marketplacePath == "" {
			continue
		}

		gitDir := filepath.Join(marketplacePath, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			continue
		}

		currentCmd := exec.Command("git", "-C", marketplacePath, "rev-parse", "HEAD")
		currentOutput, err := currentCmd.Output()
		if err != nil {
			continue
		}
		currentCommit := strings.TrimSpace(string(currentOutput))

		if plugin.GitCommitSha != currentCommit {
			updates = append(updates, PluginUpdate{
				Name:          name,
				Scope:         plugin.Scope,
				HasUpdate:     true,
				CurrentCommit: truncateHash(plugin.GitCommitSha),
				LatestCommit:  truncateHash(currentCommit),
			})
		}
	}

	return updates
}
```

Refactor `updatePlugin` to accept scope:

```go
func updatePlugin(name string, scope string, plugins *claude.PluginRegistry) error {
	plugin, exists := plugins.GetPluginAtScope(name, scope)
	if !exists {
		return fmt.Errorf("plugin not found at scope %s", scope)
	}
	// ... rest of the function remains the same until the SetPlugin call ...
```

Update the `updatePlugin` call site in `runUpgrade`:

```go
// Old:
for _, name := range outdatedPlugins {
    if err := updatePlugin(name, plugins); err != nil {
// New: outdatedPlugins should now carry scope info
```

To carry scope info, change `outdatedPlugins` from `[]string` to `[]PluginUpdate` (or a struct with name+scope). Simplest: just iterate the `pluginUpdates` directly:

```go
// Replace the outdatedPlugins collection and iteration.
// Instead of building a []string, just collect the filtered PluginUpdate entries:
var outdatedUpdates []PluginUpdate
// ... (populate from pluginUpdates, applying target filters) ...

// Then:
if len(outdatedUpdates) > 0 {
    fmt.Println()
    fmt.Println(ui.RenderSection("Updating Plugins", len(outdatedUpdates)))
    for _, update := range outdatedUpdates {
        if err := updatePlugin(update.Name, update.Scope, plugins); err != nil {
            ui.PrintError(fmt.Sprintf("%s (%s): %v", update.Name, update.Scope, err))
        } else {
            ui.PrintSuccess(fmt.Sprintf("%s (%s): Updated", update.Name, update.Scope))
        }
    }
    // Save updated plugin registry
    if err := claude.SavePlugins(claudeDir, plugins); err != nil {
        return fmt.Errorf("failed to save plugins: %w", err)
    }
}
```

**Step 4: Run upgrade tests**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./internal/commands/ -run TestUpgradeCommands -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./... -v`
Expected: PASS

**Step 6: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/commands/upgrade.go internal/commands/upgrade_test.go
git commit -m "Add --all flag and scope-aware plugin upgrade"
```

---

### Task 14: Add --all Flag and Scope-Aware Outdated

**Files:**
- Modify: `internal/commands/outdated.go`

**Step 1: Add --all flag**

```go
var outdatedAll bool

func init() {
	rootCmd.AddCommand(outdatedCmd)
	outdatedCmd.Flags().BoolVar(&outdatedAll, "all", false, "Check plugins across all scopes, not just the current context")
}
```

**Step 2: Replace GetAllPlugins calls**

Lines 81-82:
```go
// Old:
fmt.Println(ui.RenderSection("Plugins", len(plugins.GetAllPlugins())))
if len(plugins.GetAllPlugins()) == 0 {
// New:
scopes := availableScopes(outdatedAll)
scopedPlugins := plugins.GetPluginsAtScopes(scopes)
fmt.Println(ui.RenderSection("Plugins", len(scopedPlugins)))
if len(scopedPlugins) == 0 {
```

Update the `checkPluginUpdates` call (line 85) to pass scopes:
```go
// Old:
pluginUpdates := checkPluginUpdates(plugins, marketplaces)
// New:
pluginUpdates := checkPluginUpdates(plugins, marketplaces, scopes)
```

Update the output format to show scope labels:
```go
// Old:
fmt.Printf("  %s %s %s %s %s\n", ui.Warning(ui.SymbolWarning), update.Name, update.CurrentCommit, ui.SymbolArrow, ui.Success(update.LatestCommit))
// New:
fmt.Printf("  %s %s (%s) %s %s %s\n", ui.Warning(ui.SymbolWarning), update.Name, update.Scope, update.CurrentCommit, ui.SymbolArrow, ui.Success(update.LatestCommit))
```

**Step 3: Run full test suite**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./... -v`
Expected: PASS

**Step 4: Commit**

```bash
cd /Users/markalston/workspace/claudeup/claudeup && git add internal/commands/outdated.go
git commit -m "Add --all flag and scope-aware outdated checking"
```

---

### Task 15: Final Verification and Cleanup

**Files:** All modified files

**Step 1: Full build**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go build ./...`
Expected: PASS

**Step 2: Full test suite**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && go test ./... -v`
Expected: PASS

**Step 3: Verify no references to removed methods remain**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && grep -rn "\.GetPlugin\b\|\.GetAllPlugins\b\|\.PluginExists\b\|\.IsPluginInstalled\b" internal/ --include="*.go" | grep -v "_test.go"`
Expected: No output (no remaining references in production code)

Also check test files:
Run: `cd /Users/markalston/workspace/claudeup/claudeup && grep -rn "\.GetPlugin\b\|\.GetAllPlugins\b\|\.PluginExists\b\|\.IsPluginInstalled\b" internal/ --include="*.go"`
Expected: No output (no remaining references anywhere)

**Step 4: Commit any fixes**

If any straggler references found, fix and commit.

**Step 5: Verify git log**

Run: `cd /Users/markalston/workspace/claudeup/claudeup && git log --oneline scope-aware-upgrade`
Expected: Design doc commit + ~13 implementation commits
