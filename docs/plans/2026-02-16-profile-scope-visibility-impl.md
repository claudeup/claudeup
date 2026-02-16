# Profile Scope Visibility Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `profile status` show the live effective configuration across all scopes, add `--apply` flag to `profile save`, and improve messaging throughout.

**Architecture:** Four independent changes to `profile_cmd.go` and `scope_helpers.go`, plus a description generation fix in `profile.go`. Each change has its own test file updates. Tasks are ordered by dependency: scope_helpers first (used by status), then status, then save, then messaging.

**Tech Stack:** Go, Cobra CLI, Ginkgo/Gomega testing

---

### Task 1: Add user scope to untracked detection

**Files:**

- Modify: `internal/commands/scope_helpers.go:91-131` (`getUntrackedScopes`)
- Modify: `internal/commands/scope_helpers.go:82-87` (`UntrackedScopeInfo` comment)
- Modify: `internal/commands/scope_helpers.go:134-148` (`renderUntrackedScopeHints`)
- Test: `test/acceptance/profile_status_test.go`
- Test: `test/acceptance/profile_list_test.go`

**Step 1: Write the failing test**

Add to `test/acceptance/profile_status_test.go`, inside the existing `Describe("untracked scope hints")`:

```go
Context("with untracked user-scope settings", func() {
    BeforeEach(func() {
        env.CreateSettings(map[string]bool{
            "plugin-x@marketplace": true,
            "plugin-y@marketplace": true,
            "plugin-z@marketplace": true,
        })
        // No active profile set at user scope
        // (clear the one set in parent BeforeEach)
        env.ClearActiveProfile()
    })

    It("shows warning about untracked user scope", func() {
        result := env.RunInDir(projectDir, "profile", "status")
        // status will error because no active profile
        // but we should test via profile list instead
    })
})
```

Actually, `profile status` without an active profile errors. The untracked hints are shown on `profile list`. Write the test there instead:

Add to `test/acceptance/profile_list_test.go` (find the existing untracked hints section or create one):

```go
Describe("untracked scope hints", func() {
    Context("with untracked user-scope settings", func() {
        BeforeEach(func() {
            env.CreateSettings(map[string]bool{
                "plugin-x@marketplace": true,
                "plugin-y@marketplace": true,
                "plugin-z@marketplace": true,
            })
            // Don't set any active profile at user scope
        })

        It("shows warning about untracked user scope", func() {
            result := env.Run("profile", "list")

            Expect(result.ExitCode).To(Equal(0))
            Expect(result.Stdout).To(ContainSubstring("user:"))
            Expect(result.Stdout).To(ContainSubstring("3 plugins"))
            Expect(result.Stdout).To(ContainSubstring("no profile tracked"))
        })

        It("suggests save with --apply flag", func() {
            result := env.Run("profile", "list")

            Expect(result.ExitCode).To(Equal(0))
            Expect(result.Stdout).To(ContainSubstring("profile save <name> --apply"))
        })
    })
})
```

**Step 2: Run test to verify it fails**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "untracked user-scope" ./test/acceptance/...`
Expected: FAIL -- user scope not checked, hint doesn't appear

**Step 3: Write minimal implementation**

In `internal/commands/scope_helpers.go`, modify `getUntrackedScopes`:

```go
// Update UntrackedScopeInfo comment (line 84):
Scope        string // "user", "project", or "local"

// Update the scopes loop (line 98):
for _, scope := range []string{"user", "project", "local"} {
```

For user scope, the settings file path is different. Update the file path logic (lines 118-121):

```go
var settingsFile string
switch scope {
case "user":
    settingsFile = config.ClaudeDirDisplay() + "/settings.json"
case "local":
    settingsFile = ".claude/settings.local.json"
default:
    settingsFile = ".claude/settings.json"
}
```

In `renderUntrackedScopeHints`, update the hint for user scope (no scope flag needed since user is default). Modify lines 143-144:

```go
if us.Scope == "user" {
    fmt.Printf("    %s Save with: claudeup profile save <name> --apply\n",
        ui.Muted(ui.SymbolArrow))
} else {
    fmt.Printf("    %s Save with: claudeup profile save <name> --%s --apply\n",
        ui.Muted(ui.SymbolArrow), us.Scope)
}
```

**Step 4: Run test to verify it passes**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "untracked user-scope" ./test/acceptance/...`
Expected: PASS

**Step 5: Update existing tests**

The existing untracked hint tests in `profile_status_test.go` check for `"profile save <name> --project"` and `"profile apply <name> --project"`. Update them to match the new single-command hint format:

```go
// Old:
Expect(result.Stdout).To(ContainSubstring("profile save <name> --project"))
Expect(result.Stdout).To(ContainSubstring("profile apply <name> --project"))

// New:
Expect(result.Stdout).To(ContainSubstring("profile save <name> --project --apply"))
```

**Step 6: Run full test suite**

Run: `go test ./...`
Expected: All pass

**Step 7: Commit**

```bash
git add internal/commands/scope_helpers.go test/acceptance/profile_list_test.go test/acceptance/profile_status_test.go
git commit -m "feat: detect untracked user-scope plugins in profile list"
```

---

### Task 2: Rewrite `profile status` as live effective config view

**Files:**

- Modify: `internal/commands/profile_cmd.go:204-223` (profileStatusCmd definition)
- Modify: `internal/commands/profile_cmd.go:1758-1837` (`runProfileStatus`)
- Test: `test/acceptance/profile_status_test.go` (rewrite)

**Step 1: Write failing tests for new status behavior**

Rewrite `test/acceptance/profile_status_test.go` to test the new live-view behavior:

```go
var _ = Describe("profile status", func() {
    var env *helpers.TestEnv

    BeforeEach(func() {
        env = helpers.NewTestEnv(binaryPath)
    })

    Describe("live effective configuration", func() {
        Context("with user-scope plugins only", func() {
            BeforeEach(func() {
                env.CreateSettings(map[string]bool{
                    "plugin-a@marketplace": true,
                    "plugin-b@marketplace": true,
                    "disabled-plugin@marketplace": false,
                })
            })

            It("shows user-scope plugins", func() {
                result := env.Run("profile", "status")

                Expect(result.ExitCode).To(Equal(0))
                Expect(result.Stdout).To(ContainSubstring("User scope"))
                Expect(result.Stdout).To(ContainSubstring("plugin-a@marketplace"))
                Expect(result.Stdout).To(ContainSubstring("plugin-b@marketplace"))
            })

            It("shows disabled plugins", func() {
                result := env.Run("profile", "status")

                Expect(result.ExitCode).To(Equal(0))
                Expect(result.Stdout).To(ContainSubstring("Disabled"))
                Expect(result.Stdout).To(ContainSubstring("disabled-plugin@marketplace"))
            })

            Context("with tracked user profile", func() {
                BeforeEach(func() {
                    env.CreateProfile(&profile.Profile{
                        Name:    "my-profile",
                        Plugins: []string{"plugin-a@marketplace"},
                    })
                    env.SetActiveProfile("my-profile")
                })

                It("shows profile name annotation", func() {
                    result := env.Run("profile", "status")

                    Expect(result.ExitCode).To(Equal(0))
                    Expect(result.Stdout).To(ContainSubstring("profile: my-profile"))
                })
            })

            Context("without tracked profile", func() {
                It("shows untracked annotation", func() {
                    result := env.Run("profile", "status")

                    Expect(result.ExitCode).To(Equal(0))
                    Expect(result.Stdout).To(ContainSubstring("untracked"))
                })
            })
        })

        Context("with multi-scope plugins", func() {
            var projectDir string

            BeforeEach(func() {
                projectDir = env.ProjectDir("multi-scope-test")

                // User scope
                env.CreateSettings(map[string]bool{
                    "user-plugin@marketplace": true,
                })
                env.SetActiveProfile("user-prof")
                env.CreateProfile(&profile.Profile{
                    Name:    "user-prof",
                    Plugins: []string{"user-plugin@marketplace"},
                })

                // Project scope
                env.CreateProjectScopeSettings(projectDir, map[string]bool{
                    "proj-plugin@marketplace": true,
                })
            })

            It("shows plugins from both scopes", func() {
                result := env.RunInDir(projectDir, "profile", "status")

                Expect(result.ExitCode).To(Equal(0))
                Expect(result.Stdout).To(ContainSubstring("User scope"))
                Expect(result.Stdout).To(ContainSubstring("user-plugin@marketplace"))
                Expect(result.Stdout).To(ContainSubstring("Project scope"))
                Expect(result.Stdout).To(ContainSubstring("proj-plugin@marketplace"))
            })

            It("shows untracked for project scope without profile", func() {
                result := env.RunInDir(projectDir, "profile", "status")

                Expect(result.ExitCode).To(Equal(0))
                Expect(result.Stdout).To(MatchRegexp(`Project scope.*untracked`))
            })
        })

        Context("with no plugins at any scope", func() {
            It("shows a message about empty configuration", func() {
                result := env.Run("profile", "status")

                Expect(result.ExitCode).To(Equal(0))
                Expect(result.Stdout).To(ContainSubstring("No plugins"))
            })
        })
    })
})
```

**Step 2: Run tests to verify they fail**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "live effective" ./test/acceptance/...`
Expected: FAIL -- old status doesn't read settings files directly

**Step 3: Implement new `runProfileStatus`**

Replace `runProfileStatus` in `profile_cmd.go` (lines 1758-1837). The new implementation reads settings files directly instead of loading a saved profile:

```go
func runProfileStatus(cmd *cobra.Command, args []string) error {
    cwd, _ := os.Getwd()

    // Get tracked profiles for annotation
    allActive := getAllActiveProfiles(cwd)
    trackedByScope := make(map[string]string) // scope -> profile name
    for _, ap := range allActive {
        trackedByScope[ap.Scope] = ap.Name
    }

    // Header
    fmt.Printf("Effective configuration for %s\n\n", ui.Bold(cwd))

    anyPlugins := false
    var allPluginNames []string

    for _, scope := range []string{"user", "project", "local"} {
        // Skip project/local if not in project directory
        if scope != "user" {
            if _, err := os.Stat(filepath.Join(cwd, ".claude")); os.IsNotExist(err) {
                continue
            }
        }

        settings, err := claude.LoadSettingsForScope(scope, claudeDir, cwd)
        if err != nil {
            continue
        }

        var enabled, disabled []string
        for name, isEnabled := range settings.EnabledPlugins {
            if isEnabled {
                enabled = append(enabled, name)
            } else {
                disabled = append(disabled, name)
            }
        }
        sort.Strings(enabled)
        sort.Strings(disabled)

        if len(enabled) == 0 && len(disabled) == 0 {
            continue
        }

        anyPlugins = true
        allPluginNames = append(allPluginNames, enabled...)

        // Scope header with tracking annotation
        scopeLabel := formatScopeName(scope)
        if profileName, ok := trackedByScope[scope]; ok {
            fmt.Printf("  %s (profile: %s)\n", ui.Bold(scopeLabel+" scope"), profileName)
        } else {
            fmt.Printf("  %s (%s)\n", ui.Bold(scopeLabel+" scope"), ui.Warning("untracked"))
        }

        // Enabled plugins
        if len(enabled) > 0 {
            fmt.Println("    Plugins:")
            for _, name := range enabled {
                fmt.Printf("      - %s\n", name)
            }
        }

        // Disabled plugins
        if len(disabled) > 0 {
            fmt.Println("    Disabled:")
            for _, name := range disabled {
                fmt.Printf("      - %s\n", name)
            }
        }

        // Hint for untracked scopes
        if _, ok := trackedByScope[scope]; !ok && len(enabled) > 0 {
            if scope == "user" {
                fmt.Printf("    %s Save with: claudeup profile save <name> --apply\n",
                    ui.Muted(ui.SymbolArrow))
            } else {
                fmt.Printf("    %s Save with: claudeup profile save <name> --%s --apply\n",
                    ui.Muted(ui.SymbolArrow), scope)
            }
        }

        fmt.Println()
    }

    if !anyPlugins {
        fmt.Printf("  %s\n\n", ui.Muted("No plugins configured at any scope."))
    }

    // Marketplaces section
    marketplaces, err := profile.ReadUsedMarketplacesPublic(claudeDir, allPluginNames)
    if err == nil && len(marketplaces) > 0 {
        fmt.Println("  Marketplaces:")
        for _, m := range marketplaces {
            fmt.Printf("    - %s\n", m.DisplayName())
        }
        fmt.Println()
    }

    return nil
}
```

Note: `readUsedMarketplaces` is unexported in `profile/snapshot.go`. We need to either export it or use an alternative. Check if there's already a public way to get marketplaces. If not, add a thin exported wrapper:

In `internal/profile/snapshot.go`, add:

```go
// UsedMarketplaces returns marketplaces referenced by the given plugins.
func UsedMarketplaces(claudeDir string, plugins []string) ([]Marketplace, error) {
    return readUsedMarketplaces(claudeDir, plugins)
}
```

Also update the `profileStatusCmd` definition (lines 204-223):

```go
var profileStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show effective configuration across all scopes",
    Long: `Display the live effective configuration by reading settings files directly.

Shows plugins from all active scopes (user, project, local) with:
  - Scope grouping and tracking annotations
  - Which profile is tracked at each scope (if any)
  - Hints for untracked scopes with enabled plugins
  - Marketplace summary`,
    Example: `  # Show what Claude is actually running
  claudeup profile status`,
    Args: cobra.NoArgs,
    RunE: runProfileStatus,
}
```

Note: Changed from `MaximumNArgs(1)` to `NoArgs` since status no longer takes a profile name argument.

**Step 4: Run tests to verify they pass**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "live effective" ./test/acceptance/...`
Expected: PASS

**Step 5: Run full test suite, fix any broken tests**

Run: `go test ./...`
Expected: Some existing tests in `profile_status_test.go` and `profile_current_keyword_test.go` may need updating. Fix them.

**Step 6: Commit**

```bash
git add internal/commands/profile_cmd.go internal/profile/snapshot.go test/acceptance/profile_status_test.go
git commit -m "feat: profile status shows live effective config across all scopes"
```

---

### Task 3: Make `profile show current` an alias for `profile status`

**Files:**

- Modify: `internal/commands/profile_cmd.go:1414-1443` (the "current" handling in `runProfileShow`)
- Test: `test/acceptance/profile_current_keyword_test.go`

**Step 1: Write the failing test**

Update `test/acceptance/profile_current_keyword_test.go`. The `profile show current` tests should now expect live-view output matching `profile status`:

```go
Describe("profile show current", func() {
    Context("with plugins at user scope", func() {
        BeforeEach(func() {
            env.CreateSettings(map[string]bool{
                "plugin-a@marketplace": true,
            })
        })

        It("shows live effective configuration (same as status)", func() {
            result := env.Run("profile", "show", "current")

            Expect(result.ExitCode).To(Equal(0))
            Expect(result.Stdout).To(ContainSubstring("Effective configuration"))
            Expect(result.Stdout).To(ContainSubstring("plugin-a@marketplace"))
        })
    })

    Context("with no plugins at any scope", func() {
        It("succeeds with empty message instead of erroring", func() {
            result := env.Run("profile", "show", "current")

            Expect(result.ExitCode).To(Equal(0))
            Expect(result.Stdout).To(ContainSubstring("No plugins"))
        })
    })
})
```

**Step 2: Run tests to verify they fail**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "profile show current" ./test/acceptance/...`
Expected: FAIL -- old behavior loads a profile, new test expects live view

**Step 3: Implement the alias**

In `runProfileShow` (line 1420), replace the `"current"` handling block:

```go
if name == "current" {
    return runProfileStatus(cmd, nil)
}
```

This replaces the entire block from lines 1420-1443.

**Step 4: Run tests to verify they pass**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "profile show current" ./test/acceptance/...`
Expected: PASS

**Step 5: Run full suite**

Run: `go test ./...`
Expected: All pass

**Step 6: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_current_keyword_test.go
git commit -m "feat: profile show current delegates to profile status"
```

---

### Task 4: Add `--apply` flag to `profile save`

**Files:**

- Modify: `internal/commands/profile_cmd.go:24` (add flag variable)
- Modify: `internal/commands/profile_cmd.go:642-646` (register flag)
- Modify: `internal/commands/profile_cmd.go:1277-1412` (`runProfileSave`)
- Test: `test/acceptance/profile_save_test.go`

**Step 1: Write the failing test**

Add to `test/acceptance/profile_save_test.go`:

```go
Describe("--apply flag", func() {
    var projectDir string

    BeforeEach(func() {
        projectDir = env.ProjectDir("save-apply-test")
        env.CreateProjectScopeSettings(projectDir, map[string]bool{
            "plugin-a@marketplace": true,
            "plugin-b@marketplace": true,
        })
    })

    It("saves and tracks the profile in one command", func() {
        result := env.RunInDir(projectDir, "profile", "save", "my-proj", "--project", "--apply")

        Expect(result.ExitCode).To(Equal(0))
        Expect(result.Stdout).To(ContainSubstring("Saved and applied"))
        Expect(result.Stdout).To(ContainSubstring("my-proj"))
    })

    It("shows profile as active in list after save --apply", func() {
        result := env.RunInDir(projectDir, "profile", "save", "my-proj", "--project", "--apply")
        Expect(result.ExitCode).To(Equal(0))

        listResult := env.RunInDir(projectDir, "profile", "list")
        Expect(listResult.ExitCode).To(Equal(0))
        Expect(listResult.Stdout).To(ContainSubstring("* my-proj"))
    })

    It("composes with --yes flag for fully non-interactive operation", func() {
        // Create profile first so overwrite prompt would trigger
        env.RunInDir(projectDir, "profile", "save", "my-proj", "--project")

        result := env.RunInDir(projectDir, "profile", "save", "my-proj", "--project", "--apply", "-y")

        Expect(result.ExitCode).To(Equal(0))
        Expect(result.Stdout).To(ContainSubstring("Saved and applied"))
    })
})
```

**Step 2: Run tests to verify they fail**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "apply flag" ./test/acceptance/...`
Expected: FAIL -- `--apply` flag doesn't exist yet

**Step 3: Implement the flag**

Add flag variable near line 24:

```go
profileSaveApply bool
```

Register the flag near line 646:

```go
profileSaveCmd.Flags().BoolVar(&profileSaveApply, "apply", false, "Also track this profile at the saved scope (save + apply in one step)")
```

In `runProfileSave`, after the save succeeds (after line 1383), add tracking logic when `--apply` is set:

```go
if profileSaveApply {
    // Track the profile at the appropriate scope
    if resolvedScope == "project" || resolvedScope == "local" {
        if registry, regErr := config.LoadProjectsRegistry(); regErr == nil {
            if resolvedScope == "project" {
                registry.SetProjectScope(cwd, name)
            } else {
                registry.SetProject(cwd, name)
            }
            _ = config.SaveProjectsRegistry(registry)
        }
    }
    // User scope tracking is already done above (cfg.Preferences.ActiveProfile = name)
    ui.PrintSuccess(fmt.Sprintf("Saved and applied profile %q (%s)", name, scopeLabel))
} else {
    ui.PrintSuccess(fmt.Sprintf("Saved profile %q (%s)", name, scopeLabel))
}
```

This replaces the existing `ui.PrintSuccess` line at 1383.

**Step 4: Run tests to verify they pass**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "apply flag" ./test/acceptance/...`
Expected: PASS

**Step 5: Run full suite**

Run: `go test ./...`
Expected: All pass

**Step 6: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_save_test.go
git commit -m "feat: add --apply flag to profile save for single-command adopt"
```

---

### Task 5: Smarter apply messaging (idempotent detection)

**Files:**

- Modify: `internal/commands/setup.go:546-558` (`showApplyResults`)
- Test: `test/acceptance/profile_save_test.go` or relevant apply test

**Step 1: Write the failing test**

Add a test that applies a profile when plugins are already present:

```go
Describe("idempotent apply messaging", func() {
    BeforeEach(func() {
        env.CreateSettings(map[string]bool{
            "plugin-a@marketplace": true,
        })
        env.CreateProfile(&profile.Profile{
            Name:    "existing-state",
            Plugins: []string{"plugin-a@marketplace"},
            Marketplaces: []profile.Marketplace{
                {Source: "github", Repo: "test/marketplace"},
            },
        })
        env.CreateInstalledPlugins(map[string]interface{}{
            "plugin-a@marketplace": map[string]interface{}{
                "version": "1.0.0",
            },
        })
        env.CreateKnownMarketplaces(map[string]interface{}{
            "marketplace": map[string]interface{}{
                "source": map[string]interface{}{
                    "source": "github",
                    "repo":   "test/marketplace",
                },
            },
        })
    })

    It("says tracking instead of installed when state matches", func() {
        result := env.Run("profile", "apply", "existing-state", "-y")

        Expect(result.ExitCode).To(Equal(0))
        Expect(result.Stdout).To(ContainSubstring("already installed"))
        Expect(result.Stdout).NotTo(ContainSubstring("Installed 1 plugin"))
    })
})
```

Note: The existing `showApplyResults` already has `PluginsAlreadyPresent` tracking. When all plugins are already present and nothing was installed/removed, the output shows "N plugins were already installed" but not "Installed N plugins". Verify this is the actual behavior -- if so, this test may already pass. If the message needs refinement, adjust.

**Step 2: Run tests to verify behavior**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v -focus "idempotent apply" ./test/acceptance/...`
Expected: Check actual behavior. If it already says "already installed" and nothing else, the test passes. If "Installed" also appears, we need the fix.

**Step 3: Adjust messaging if needed**

The `ApplyResult` already tracks `PluginsAlreadyPresent` separately from `PluginsInstalled`. The `showApplyResults` function in `setup.go:546` already handles both cases. If the behavior is correct, this task just needs the test to document the expected behavior. If not, adjust the messaging in `showApplyResults`.

**Step 4: Run full suite**

Run: `go test ./...`
Expected: All pass

**Step 5: Commit**

```bash
git add internal/commands/setup.go test/acceptance/...
git commit -m "test: verify idempotent apply messaging"
```

---

### Task 6: Better auto-generated descriptions (plugin counts per scope)

**Files:**

- Modify: `internal/profile/profile.go:1022-1071` (`GenerateDescription`)
- Test: `internal/profile/profile_test.go` (existing `TestProfile_GenerateDescription`)

**Step 1: Write the failing test**

Add test cases to the existing `TestProfile_GenerateDescription` in `internal/profile/profile_test.go`:

```go
{
    name: "multi-scope with plugins and marketplaces",
    profile: &Profile{
        PerScope: &PerScopeSettings{
            User: &ScopeSettings{
                Plugins: []string{"a@m", "b@m", "c@m", "d@m", "e@m"},
            },
            Project: &ScopeSettings{
                Plugins: []string{"f@m", "g@m", "h@m"},
            },
        },
        Marketplaces: []Marketplace{
            {Source: "github", Repo: "test/marketplace"},
        },
    },
    expected: "5 user plugins, 3 project plugins, 1 marketplace",
},
{
    name: "multi-scope single plugin per scope",
    profile: &Profile{
        PerScope: &PerScopeSettings{
            User: &ScopeSettings{
                Plugins: []string{"a@m"},
            },
            Project: &ScopeSettings{
                Plugins: []string{"b@m"},
            },
        },
        Marketplaces: []Marketplace{
            {Source: "github", Repo: "test/m1"},
            {Source: "github", Repo: "test/m2"},
        },
    },
    expected: "1 user plugin, 1 project plugin, 2 marketplaces",
},
{
    name: "single scope project-only",
    profile: &Profile{
        PerScope: &PerScopeSettings{
            Project: &ScopeSettings{
                Plugins: []string{"a@m", "b@m", "c@m"},
            },
        },
        Marketplaces: []Marketplace{
            {Source: "github", Repo: "test/marketplace"},
        },
    },
    expected: "3 project plugins, 1 marketplace",
},
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/profile/ -run TestProfile_GenerateDescription -v`
Expected: FAIL -- current code uses root-level `p.Plugins` count, not per-scope

**Step 3: Implement the fix**

Update `GenerateDescription` in `profile.go` (line 1022). Add multi-scope handling before the legacy path:

```go
func (p *Profile) GenerateDescription() string {
    if p.IsStack() {
        n := len(p.Includes)
        if n == 1 {
            return "stack: 1 include"
        }
        return fmt.Sprintf("stack: %d includes", n)
    }

    var parts []string

    // Multi-scope: count plugins per scope
    if p.IsMultiScope() {
        for _, s := range []struct {
            label    string
            settings *ScopeSettings
        }{
            {"user", p.PerScope.User},
            {"project", p.PerScope.Project},
            {"local", p.PerScope.Local},
        } {
            if s.settings == nil || len(s.settings.Plugins) == 0 {
                continue
            }
            n := len(s.settings.Plugins)
            word := "plugins"
            if n == 1 {
                word = "plugin"
            }
            parts = append(parts, fmt.Sprintf("%d %s %s", n, s.label, word))
        }
    } else {
        // Legacy single-scope
        pluginCount := len(p.Plugins)
        if pluginCount > 0 {
            if pluginCount == 1 {
                parts = append(parts, "1 plugin")
            } else {
                parts = append(parts, fmt.Sprintf("%d plugins", pluginCount))
            }
        }
    }

    // Marketplaces (always root-level)
    marketplaceCount := len(p.Marketplaces)
    if marketplaceCount > 0 {
        if marketplaceCount == 1 {
            parts = append(parts, "1 marketplace")
        } else {
            parts = append(parts, fmt.Sprintf("%d marketplaces", marketplaceCount))
        }
    }

    // MCP servers (legacy only, multi-scope has them per-scope)
    if !p.IsMultiScope() {
        mcpCount := len(p.MCPServers)
        if mcpCount > 0 {
            if mcpCount == 1 {
                parts = append(parts, "1 MCP server")
            } else {
                parts = append(parts, fmt.Sprintf("%d MCP servers", mcpCount))
            }
        }
    }

    if len(parts) == 0 {
        return "Empty profile"
    }

    return strings.Join(parts, ", ")
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/profile/ -run TestProfile_GenerateDescription -v`
Expected: PASS

**Step 5: Run full suite to check for regressions**

Run: `go test ./...`
Expected: All pass. Some acceptance tests may check for the old description format (e.g., "1 marketplace"). Update any broken assertions.

**Step 6: Commit**

```bash
git add internal/profile/profile.go internal/profile/profile_test.go
git commit -m "feat: include plugin counts per scope in auto-generated descriptions"
```

---

### Task 7: Update `profile list` footer hints

**Files:**

- Modify: `internal/commands/profile_cmd.go` (footer section in `runProfileList`)
- Test: `test/acceptance/profile_list_test.go`

**Step 1: Find and update the footer hints**

Search for the footer in `runProfileList` that shows "Use 'claudeup profile show <name>'" hints. Update to:

```go
fmt.Printf("%s Use '%s' to see effective configuration\n",
    ui.Muted(ui.SymbolArrow), "claudeup profile status")
fmt.Printf("%s Use '%s' for profile details\n",
    ui.Muted(ui.SymbolArrow), "claudeup profile show <name>")
fmt.Printf("%s Use '%s' to apply a profile\n",
    ui.Muted(ui.SymbolArrow), "claudeup profile apply <name>")
```

**Step 2: Write test for new footer**

```go
It("shows updated footer hints with status command", func() {
    result := env.Run("profile", "list")

    Expect(result.ExitCode).To(Equal(0))
    Expect(result.Stdout).To(ContainSubstring("claudeup profile status"))
    Expect(result.Stdout).To(ContainSubstring("claudeup profile show <name>"))
})
```

**Step 3: Run tests**

Run: `go test ./...`
Expected: All pass (update any tests that checked old footer text)

**Step 4: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_list_test.go
git commit -m "feat: update profile list footer hints to reference status command"
```

---

### Task 8: Final integration test and cleanup

**Step 1: Run the full acceptance and integration suite**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v ./test/acceptance/... ./test/integration/...`
Expected: All pass

**Step 2: Run the complete test suite**

Run: `go test ./...`
Expected: All pass

**Step 3: Check for any remaining references to old patterns**

Search for stale references:

```bash
grep -rn "profile save <name> --project && claudeup profile apply" internal/
grep -rn "profile show.*current.*active profile" internal/
```

Fix any remaining references.

**Step 4: Commit any cleanup**

```bash
git add -A
git commit -m "chore: clean up stale references to old profile UX patterns"
```
