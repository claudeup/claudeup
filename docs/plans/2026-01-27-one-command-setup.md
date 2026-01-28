# One-Command Setup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce claudeup onboarding from 2 commands to 1 by having setup install plugins automatically.

**Architecture:** After setup saves/applies a profile, call a new helper function that installs any plugins defined in that profile. This reuses the existing `profile.Apply` infrastructure for plugin installation. The function shows progress, handles failures gracefully (continue on error), and displays a summary.

**Tech Stack:** Go, Cobra CLI, existing profile/claude packages

---

## Task 1: Add installPluginsFromProfile helper function

**Files:**

- Modify: `internal/commands/setup.go` (add new function after line 476)

**Step 1: Write the failing test**

Add to `test/acceptance/setup_test.go`:

```go
It("installs plugins when saving profile for existing installation", func() {
    // Create an existing installation with enabled plugins but not installed
    env.CreateClaudeSettingsWithPlugins(map[string]bool{
        "test-plugin@test-marketplace": true,
    })

    // Create the marketplace and plugin so installation can succeed
    env.CreateMarketplace("test-marketplace", "github.com/test/marketplace")
    env.CreateMarketplacePlugin("test-marketplace", "test-plugin", "1.0.0")

    // Run setup with -y to auto-confirm, accept defaults
    result := env.Run("setup", "-y")

    Expect(result.ExitCode).To(Equal(0))
    Expect(result.Stdout).To(ContainSubstring("Installing plugins"))
    Expect(result.Stdout).To(ContainSubstring("1 plugins installed"))
})
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run "installs plugins when saving profile" ./test/acceptance/...`
Expected: FAIL - "Installing plugins" not found in output

**Step 3: Add helper function to setup.go**

Add after the `buildSecretChain` function (around line 476):

```go
// installPluginsFromProfile installs plugins defined in a profile.
// Shows progress spinner, continues on individual failures, displays summary.
// Returns nil even if some plugins fail (warnings only).
func installPluginsFromProfile(p *profile.Profile, claudeDir, claudeJSONPath string) error {
	if len(p.Plugins) == 0 {
		return nil
	}

	// Prompt unless -y
	if !config.YesFlag {
		fmt.Printf("Install %d plugins from profile? [Y/n]: ", len(p.Plugins))
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil
		}
		choice := strings.TrimSpace(strings.ToLower(input))
		if choice != "" && choice != "y" && choice != "yes" {
			ui.PrintMuted("Skipping plugin installation.")
			return nil
		}
	}

	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Installing %d plugins...", len(p.Plugins)))

	chain := buildSecretChain()
	result, err := profile.Apply(p, claudeDir, claudeJSONPath, chain)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Plugin installation failed: %v", err))
		return nil // Don't fail setup for plugin errors
	}

	// Show summary
	installed := len(result.PluginsInstalled)
	alreadyPresent := len(result.PluginsAlreadyPresent)
	failed := len(result.Errors)

	if installed > 0 {
		fmt.Printf("  %s %d plugins installed\n", ui.Success(ui.SymbolSuccess), installed)
		for _, plugin := range result.PluginsInstalled {
			fmt.Printf("    %s %s\n", ui.Muted(ui.SymbolBullet), plugin)
		}
	}
	if alreadyPresent > 0 {
		fmt.Printf("  %s %d plugins already installed\n", ui.Muted(ui.SymbolSuccess), alreadyPresent)
	}
	if failed > 0 {
		fmt.Printf("  %s %d plugins failed\n", ui.Warning(ui.SymbolWarning), failed)
		for _, e := range result.Errors {
			fmt.Printf("    %s %v\n", ui.Error(ui.SymbolBullet), e)
		}
	}

	return nil
}
```

**Step 4: Run test to verify it still fails**

Run: `go test -v -run "installs plugins when saving profile" ./test/acceptance/...`
Expected: Still FAIL - function exists but not called yet

**Step 5: Commit**

```bash
git add internal/commands/setup.go
git commit -m "$(cat <<'EOF'
feat(setup): add installPluginsFromProfile helper

Adds helper function that installs plugins from a profile with:
- Optional prompt (skipped with -y flag)
- Progress indication
- Continues on individual failures
- Summary of installed/failed plugins
EOF
)"
```

---

## Task 2: Call installPluginsFromProfile for existing installations

**Files:**

- Modify: `internal/commands/setup.go:110-150` (handleExistingInstallationPreserve)

**Step 1: The failing test from Task 1 should guide us**

The test expects "Installing plugins" in output after setup with existing installation.

**Step 2: Modify handleExistingInstallationPreserve**

Update the function to call installPluginsFromProfile after saving the profile. Replace the function:

```go
// handleExistingInstallationPreserve saves the existing config as a profile but keeps
// the user's current settings intact (doesn't overwrite with default)
func handleExistingInstallationPreserve(existing *profile.Profile, profilesDir string, claudeDir string, claudeJSONPath string) error {
	ui.PrintInfo("Existing Claude Code installation detected:")
	fmt.Printf("  %s %d MCP servers, %d marketplaces, %d plugins\n",
		ui.Muted(ui.SymbolArrow), len(existing.MCPServers), len(existing.Marketplaces), len(existing.Plugins))
	fmt.Println()

	fmt.Println("Your current settings will be preserved.")
	fmt.Println("claudeup can save them as a profile for easy backup/restore.")
	fmt.Println()
	fmt.Println(ui.Bold("Options:"))
	fmt.Println("  [s] Save current setup as a profile (recommended)")
	fmt.Println("  [c] Continue without saving")
	fmt.Println("  [a] Abort")
	fmt.Println()

	choice := promptChoice("Choice", "s")

	var savedProfile *profile.Profile

	switch strings.ToLower(choice) {
	case "s":
		name := promptProfileName("Profile name", "my-setup")
		existing.Name = name
		existing.Description = "Saved from existing installation"
		if err := profile.Save(profilesDir, existing); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}
		ui.PrintSuccess(fmt.Sprintf("Saved as '%s'", name))
		fmt.Println()
		fmt.Println(ui.Muted("Your current settings are unchanged."))
		fmt.Println(ui.Muted(fmt.Sprintf("To restore later: claudeup profile apply %s", name)))
		savedProfile = existing
	case "c":
		fmt.Println("  Continuing without saving...")
		// Still use existing profile for plugin installation
		savedProfile = existing
	case "a":
		return fmt.Errorf("setup aborted by user")
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}

	// Install plugins from the profile
	if savedProfile != nil && len(savedProfile.Plugins) > 0 {
		fmt.Println()
		if err := installPluginsFromProfile(savedProfile, claudeDir, claudeJSONPath); err != nil {
			ui.PrintWarning(fmt.Sprintf("Plugin installation issue: %v", err))
		}
	}

	return nil
}
```

**Step 3: Update the call site in runSetup**

Find the call to `handleExistingInstallationPreserve` (around line 80) and update it to pass the additional parameters:

```go
if hasExisting {
    // User has existing Claude Code setup - preserve it
    claudeJSONPath := filepath.Join(claudeDir, ".claude.json")
    if err := handleExistingInstallationPreserve(existing, profilesDir, claudeDir, claudeJSONPath); err != nil {
        return err
    }
}
```

Wait - claudeJSONPath is already defined earlier. Remove the duplicate definition and just pass it:

```go
if hasExisting {
    // User has existing Claude Code setup - preserve it
    if err := handleExistingInstallationPreserve(existing, profilesDir, claudeDir, claudeJSONPath); err != nil {
        return err
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run "installs plugins when saving profile" ./test/acceptance/...`
Expected: PASS

**Step 5: Run all tests to check for regressions**

Run: `go test ./test/acceptance/... -v -run setup`
Expected: All setup tests pass

**Step 6: Commit**

```bash
git add internal/commands/setup.go
git commit -m "$(cat <<'EOF'
feat(setup): install plugins for existing installations

Setup now automatically installs plugins after saving the profile.
This reduces onboarding from 2 commands to 1.
EOF
)"
```

---

## Task 3: Add test helper for creating settings with enabled plugins

**Files:**

- Modify: `test/helpers/test_env.go`

**Step 1: Check if helper already exists**

Run: `grep -n "CreateClaudeSettingsWithPlugins" test/helpers/test_env.go`
Expected: No match (helper doesn't exist)

**Step 2: Add the helper function**

Find the `CreateClaudeSettings` function and add a new variant after it:

```go
// CreateClaudeSettingsWithPlugins creates settings.json with enabled plugins
func (e *TestEnv) CreateClaudeSettingsWithPlugins(enabledPlugins map[string]bool) {
	settings := map[string]interface{}{
		"enabledPlugins": enabledPlugins,
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		panic(err)
	}
	settingsPath := filepath.Join(e.ClaudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		panic(err)
	}
}
```

**Step 3: Add helpers for creating marketplace and plugin**

```go
// CreateMarketplace creates a fake marketplace directory with git repo
func (e *TestEnv) CreateMarketplace(name, repo string) {
	marketplaceDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", name)
	if err := os.MkdirAll(marketplaceDir, 0755); err != nil {
		panic(err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = marketplaceDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	// Create initial commit
	readmePath := filepath.Join(marketplaceDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# "+name), 0644); err != nil {
		panic(err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = marketplaceDir
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = marketplaceDir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	_ = cmd.Run()

	// Create known_marketplaces.json
	registry := map[string]interface{}{
		name: map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   repo,
			},
			"installLocation": marketplaceDir,
			"lastUpdated":     "2024-01-01T00:00:00Z",
		},
	}
	data, _ := json.MarshalIndent(registry, "", "  ")
	registryPath := filepath.Join(e.ClaudeDir, "plugins", "known_marketplaces.json")
	_ = os.WriteFile(registryPath, data, 0644)
}

// CreateMarketplacePlugin creates a fake plugin in a marketplace
func (e *TestEnv) CreateMarketplacePlugin(marketplace, pluginName, version string) {
	pluginDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", marketplace, "plugins", pluginName)
	claudePluginDir := filepath.Join(pluginDir, ".claude-plugin")
	if err := os.MkdirAll(claudePluginDir, 0755); err != nil {
		panic(err)
	}

	pluginJSON := map[string]interface{}{
		"name":    pluginName,
		"version": version,
	}
	data, _ := json.MarshalIndent(pluginJSON, "", "  ")
	if err := os.WriteFile(filepath.Join(claudePluginDir, "plugin.json"), data, 0644); err != nil {
		panic(err)
	}
}
```

**Step 4: Run the test to verify helpers work**

Run: `go test -v -run "installs plugins when saving profile" ./test/acceptance/...`
Expected: PASS (or closer to passing)

**Step 5: Commit**

```bash
git add test/helpers/test_env.go
git commit -m "$(cat <<'EOF'
test: add helpers for creating settings with plugins

Adds CreateClaudeSettingsWithPlugins, CreateMarketplace, and
CreateMarketplacePlugin helpers for testing plugin installation flows.
EOF
)"
```

---

## Task 4: Add test for plugin installation with failures

**Files:**

- Modify: `test/acceptance/setup_test.go`

**Step 1: Write the test**

```go
It("continues setup when plugin installation fails", func() {
    // Create an existing installation with a plugin that can't be installed
    // (marketplace doesn't exist)
    env.CreateClaudeSettingsWithPlugins(map[string]bool{
        "missing-plugin@nonexistent-marketplace": true,
    })

    // Run setup with -y
    result := env.Run("setup", "-y")

    // Setup should complete successfully (exit 0)
    Expect(result.ExitCode).To(Equal(0))
    // Should show failure info
    Expect(result.Stdout).To(ContainSubstring("plugins failed"))
    // Should still complete
    Expect(result.Stdout).To(ContainSubstring("Setup complete"))
})
```

**Step 2: Run test**

Run: `go test -v -run "continues setup when plugin installation fails" ./test/acceptance/...`
Expected: PASS (the implementation should already handle this)

**Step 3: Commit**

```bash
git add test/acceptance/setup_test.go
git commit -m "$(cat <<'EOF'
test(setup): verify plugin failures don't block setup

Adds test confirming that setup completes successfully even when
individual plugin installations fail.
EOF
)"
```

---

## Task 5: Update test-claudup.sh script

**Files:**

- Modify: `scripts/test-claudup.sh`

**Step 1: Find and remove the separate profile apply step**

The script currently has:

```bash
section "Applying user profile (my-setup)"

claudeup profile apply my-setup -y --scope user
claudeup profile show my-setup
```

This is no longer needed because setup now installs plugins.

**Step 2: Update the script**

Remove the "Applying user profile" section entirely. The flow becomes:

1. Install claudeup
2. Set up test environment
3. Create simulated user configuration
4. Add marketplace
5. Run `claudeup setup -y` (now installs plugins too!)
6. Create and apply project profile
7. Show final state

Replace lines 159-166 (the "Applying user profile" section) with just a show command:

```bash
# -----------------------------------------------------------------------------
# Verify setup result
# -----------------------------------------------------------------------------

section "Verifying setup"

claudeup profile show my-setup
```

**Step 3: Test the script manually**

Run: `USE_LOCAL_BUILD=true ./scripts/test-claudup.sh`
Expected: Script completes successfully, shows plugins installed during setup

**Step 4: Commit**

```bash
git add scripts/test-claudup.sh
git commit -m "$(cat <<'EOF'
docs(scripts): simplify test-claudup.sh after one-command setup

Remove separate 'profile apply' step since setup now installs plugins
automatically. The onboarding flow is now simpler.
EOF
)"
```

---

## Task 6: Run full test suite and verify

**Step 1: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 2: Run acceptance tests specifically**

Run: `go test -v ./test/acceptance/... -run setup`
Expected: All setup tests pass

**Step 3: Build and test manually**

```bash
go build -o bin/claudeup ./cmd/claudeup
export CLAUDE_CONFIG_DIR=$(mktemp -d)
export CLAUDEUP_HOME=$(mktemp -d)
./bin/claudeup setup -y
```

Expected: Setup completes, shows plugin installation (if any plugins in profile)

**Step 4: Final commit with any fixes**

If any issues found, fix and commit with appropriate message.

---

## Task 7: Create PR

**Step 1: Push branch**

```bash
git push -u origin feat/one-command-setup
```

**Step 2: Create PR**

```bash
gh pr create --title "feat(setup): install plugins automatically (one-command setup)" --body "$(cat <<'EOF'
## Summary
- Setup now installs plugins automatically after creating/saving the profile
- Reduces onboarding from 2 commands (`setup` + `profile apply`) to 1 command (`setup`)
- Plugin failures don't block setup - continues and reports errors

## Test plan
- [x] New acceptance test: plugins installed for existing installation
- [x] New acceptance test: failures don't block setup
- [x] Existing setup tests still pass
- [x] Manual test with scripts/test-claudup.sh
EOF
)"
```

---

## Summary

| Task | Description                            | Files           |
| ---- | -------------------------------------- | --------------- |
| 1    | Add installPluginsFromProfile helper   | setup.go        |
| 2    | Call helper for existing installations | setup.go        |
| 3    | Add test helpers                       | test_env.go     |
| 4    | Add failure handling test              | setup_test.go   |
| 5    | Update example script                  | test-claudup.sh |
| 6    | Full test suite verification           | -               |
| 7    | Create PR                              | -               |
