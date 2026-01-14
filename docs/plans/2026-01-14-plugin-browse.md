# Plugin Browse Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `claudeup plugin browse <marketplace>` command to list available plugins from a marketplace.

**Architecture:** New subcommand under `plugin` that reads `.claude-plugin/marketplace.json` from installed marketplaces. Marketplace lookup supports name, repo, or URL identification. Output shows plugin name, description, version, and installation status.

**Tech Stack:** Go, Cobra CLI, Ginkgo/Gomega tests

---

## Task 1: Add Test Helper for Marketplace Index

**Files:**
- Modify: `test/helpers/testenv.go`

**Step 1: Write the helper method**

Add to `test/helpers/testenv.go` after `CreateKnownMarketplaces`:

```go
// CreateMarketplaceIndex creates a fake .claude-plugin/marketplace.json for a marketplace
func (e *TestEnv) CreateMarketplaceIndex(installLocation string, name string, plugins []map[string]string) {
	indexDir := filepath.Join(installLocation, ".claude-plugin")
	Expect(os.MkdirAll(indexDir, 0755)).To(Succeed())

	index := map[string]interface{}{
		"name":    name,
		"plugins": plugins,
	}
	jsonData, err := json.MarshalIndent(index, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(indexDir, "marketplace.json"), jsonData, 0644)).To(Succeed())
}
```

**Step 2: Verify it compiles**

Run: `go build ./test/helpers/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add test/helpers/testenv.go
git commit -m "test: add CreateMarketplaceIndex helper"
```

---

## Task 2: Write Failing Acceptance Tests

**Files:**
- Create: `test/acceptance/plugin_browse_test.go`

**Step 1: Write the test file**

```go
// ABOUTME: Acceptance tests for plugin browse command
// ABOUTME: Tests browsing available plugins from marketplaces
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin browse", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no arguments", func() {
		It("shows error requiring marketplace argument", func() {
			result := env.Run("plugin", "browse")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("requires"))
		})
	})

	Describe("with unknown marketplace", func() {
		It("shows error with helpful message", func() {
			result := env.Run("plugin", "browse", "unknown-marketplace")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
			Expect(result.Stderr).To(ContainSubstring("claude marketplace add"))
		})
	})

	Describe("with installed marketplace", func() {
		var marketplacePath string

		BeforeEach(func() {
			// Create marketplace directory
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			// Register marketplace in known_marketplaces.json
			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			// Create marketplace index
			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "acme-marketplace", []map[string]string{
				{"name": "plugin-a", "description": "First plugin", "version": "1.0.0"},
				{"name": "plugin-b", "description": "Second plugin", "version": "2.0.0"},
			})
		})

		It("lists available plugins by marketplace name", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("acme-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("plugin-a"))
			Expect(result.Stdout).To(ContainSubstring("plugin-b"))
		})

		It("lists available plugins by repo", func() {
			result := env.Run("plugin", "browse", "acme-corp/plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("plugin-a"))
		})

		It("shows plugin descriptions", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("First plugin"))
			Expect(result.Stdout).To(ContainSubstring("Second plugin"))
		})

		It("shows plugin versions", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1.0.0"))
			Expect(result.Stdout).To(ContainSubstring("2.0.0"))
		})

		It("shows plugin count", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("2"))
		})
	})

	Describe("with installed plugins", func() {
		var marketplacePath string
		var pluginPath string

		BeforeEach(func() {
			// Create marketplace
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "acme-marketplace", []map[string]string{
				{"name": "plugin-a", "description": "First plugin", "version": "1.0.0"},
				{"name": "plugin-b", "description": "Second plugin", "version": "2.0.0"},
			})

			// Install one plugin
			pluginPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "plugin-a")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"plugin-a@acme-marketplace": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pluginPath,
						"scope":       "user",
					},
				},
			})
		})

		It("shows installed status for installed plugins", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("installed"))
		})
	})

	Describe("table format", func() {
		var marketplacePath string

		BeforeEach(func() {
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "acme-marketplace", []map[string]string{
				{"name": "plugin-a", "description": "First plugin", "version": "1.0.0"},
			})
		})

		It("shows table headers", func() {
			result := env.Run("plugin", "browse", "--format", "table", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("PLUGIN"))
			Expect(result.Stdout).To(ContainSubstring("DESCRIPTION"))
			Expect(result.Stdout).To(ContainSubstring("VERSION"))
		})
	})

	Describe("empty marketplace", func() {
		var marketplacePath string

		BeforeEach(func() {
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "empty-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"empty-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "empty-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "empty-marketplace", []map[string]string{})
		})

		It("shows no plugins message", func() {
			result := env.Run("plugin", "browse", "empty-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No plugins"))
		})
	})
})
```

**Step 2: Run tests to verify they fail**

Run: `go test ./test/acceptance/... -v -run "plugin browse" 2>&1 | head -50`
Expected: FAIL - unknown command "browse"

**Step 3: Commit failing tests**

```bash
git add test/acceptance/plugin_browse_test.go
git commit -m "test: add failing acceptance tests for plugin browse"
```

---

## Task 3: Add MarketplaceIndex Types

**Files:**
- Modify: `internal/claude/marketplaces.go`

**Step 1: Add types after MarketplaceSource struct**

```go
// MarketplaceIndex represents the .claude-plugin/marketplace.json file
type MarketplaceIndex struct {
	Name    string                  `json:"name"`
	Plugins []MarketplacePluginInfo `json:"plugins"`
}

// MarketplacePluginInfo represents a plugin entry in the marketplace index
type MarketplacePluginInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/claude/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/claude/marketplaces.go
git commit -m "feat: add MarketplaceIndex types"
```

---

## Task 4: Add FindMarketplace Function

**Files:**
- Modify: `internal/claude/marketplaces.go`

**Step 1: Add FindMarketplace function**

```go
// FindMarketplace finds a marketplace by name, repo, or URL
// Returns the marketplace metadata, its key in the registry, and any error
func FindMarketplace(claudeDir string, identifier string) (*MarketplaceMetadata, string, error) {
	registry, err := LoadMarketplaces(claudeDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load marketplaces: %w", err)
	}

	// First, check by key (marketplace name in registry)
	if meta, exists := registry[identifier]; exists {
		return &meta, identifier, nil
	}

	// Check by repo or URL
	for name, meta := range registry {
		if meta.Source.Repo == identifier || meta.Source.URL == identifier {
			return &meta, name, nil
		}
	}

	// Check by marketplace name from index files
	for name, meta := range registry {
		index, err := LoadMarketplaceIndex(meta.InstallLocation)
		if err != nil {
			continue
		}
		if index.Name == identifier {
			return &meta, name, nil
		}
	}

	return nil, "", fmt.Errorf("marketplace %q not found", identifier)
}
```

**Step 2: Add import for fmt if not present**

Check top of file and add `"fmt"` to imports if missing.

**Step 3: Verify it compiles**

Run: `go build ./internal/claude/...`
Expected: Build fails - LoadMarketplaceIndex not defined yet (expected)

**Step 4: Commit partial progress**

```bash
git add internal/claude/marketplaces.go
git commit -m "feat: add FindMarketplace function (partial)"
```

---

## Task 5: Add LoadMarketplaceIndex Function

**Files:**
- Modify: `internal/claude/marketplaces.go`

**Step 1: Add LoadMarketplaceIndex function**

```go
// LoadMarketplaceIndex reads the .claude-plugin/marketplace.json from a marketplace
func LoadMarketplaceIndex(installLocation string) (*MarketplaceIndex, error) {
	indexPath := filepath.Join(installLocation, ".claude-plugin", "marketplace.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read marketplace index: %w", err)
	}

	var index MarketplaceIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse marketplace index: %w", err)
	}

	return &index, nil
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/claude/...`
Expected: Build succeeds

**Step 3: Run unit tests**

Run: `go test ./internal/claude/... -v`
Expected: All existing tests pass

**Step 4: Commit**

```bash
git add internal/claude/marketplaces.go
git commit -m "feat: add LoadMarketplaceIndex function"
```

---

## Task 6: Add Browse Command Structure

**Files:**
- Modify: `internal/commands/plugin.go`

**Step 1: Add browse flag variable**

Add after existing flag variables (around line 23):

```go
var pluginBrowseFormat string
```

**Step 2: Add browseCmd definition**

Add after `pluginEnableCmd` definition:

```go
var pluginBrowseCmd = &cobra.Command{
	Use:   "browse <marketplace>",
	Short: "Browse available plugins in a marketplace",
	Long: `Display plugins available in a marketplace before installing.

Accepts marketplace name, repo (user/repo), or URL as identifier.`,
	Example: `  claudeup plugin browse claude-code-workflows
  claudeup plugin browse wshobson/agents
  claudeup plugin browse --format table my-marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginBrowse,
}
```

**Step 3: Register command in init()**

Add to init() function:

```go
pluginCmd.AddCommand(pluginBrowseCmd)
pluginBrowseCmd.Flags().StringVar(&pluginBrowseFormat, "format", "", "Output format (table)")
```

**Step 4: Add stub runPluginBrowse function**

```go
func runPluginBrowse(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("not implemented")
}
```

**Step 5: Verify it compiles**

Run: `go build ./...`
Expected: Build succeeds

**Step 6: Run acceptance tests**

Run: `go test ./test/acceptance/... -v -run "plugin browse" 2>&1 | head -30`
Expected: Tests run but fail with "not implemented"

**Step 7: Commit**

```bash
git add internal/commands/plugin.go
git commit -m "feat: add plugin browse command structure"
```

---

## Task 7: Implement Browse Command - Marketplace Lookup

**Files:**
- Modify: `internal/commands/plugin.go`

**Step 1: Implement marketplace lookup in runPluginBrowse**

Replace the stub with:

```go
func runPluginBrowse(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Find the marketplace
	meta, marketplaceName, err := claude.FindMarketplace(claudeDir, identifier)
	if err != nil {
		// Build helpful error message
		registry, loadErr := claude.LoadMarketplaces(claudeDir)
		if loadErr != nil {
			return fmt.Errorf("marketplace %q not found\n\nTo add this marketplace, use Claude Code CLI:\n  claude marketplace add <repo-or-url>", identifier)
		}

		var installed []string
		for name, m := range registry {
			if m.Source.Repo != "" {
				installed = append(installed, fmt.Sprintf("  %s (%s)", name, m.Source.Repo))
			} else {
				installed = append(installed, fmt.Sprintf("  %s", name))
			}
		}
		sort.Strings(installed)

		msg := fmt.Sprintf("Error: marketplace %q not found\n\nTo add this marketplace, use Claude Code CLI:\n  claude marketplace add <repo-or-url>", identifier)
		if len(installed) > 0 {
			msg += "\n\nInstalled marketplaces:\n" + strings.Join(installed, "\n")
		}
		return fmt.Errorf(msg)
	}

	// Load the marketplace index
	index, err := claude.LoadMarketplaceIndex(meta.InstallLocation)
	if err != nil {
		return fmt.Errorf("Error: marketplace %q has no plugin index\n\nThe marketplace at %s\nis missing .claude-plugin/marketplace.json", marketplaceName, meta.InstallLocation)
	}

	_ = index // Will use in next task
	return nil
}
```

**Step 2: Add strings import if needed**

Add `"strings"` to imports.

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/commands/plugin.go
git commit -m "feat: implement marketplace lookup for browse command"
```

---

## Task 8: Implement Browse Command - Output Display

**Files:**
- Modify: `internal/commands/plugin.go`

**Step 1: Complete runPluginBrowse with output display**

Replace the end of runPluginBrowse (after loading index) with:

```go
	// Load the marketplace index
	index, err := claude.LoadMarketplaceIndex(meta.InstallLocation)
	if err != nil {
		return fmt.Errorf("Error: marketplace %q has no plugin index\n\nThe marketplace at %s\nis missing .claude-plugin/marketplace.json", marketplaceName, meta.InstallLocation)
	}

	// Handle empty marketplace
	if len(index.Plugins) == 0 {
		fmt.Printf("No plugins available in %s\n", index.Name)
		return nil
	}

	// Load installed plugins to check status
	plugins, _ := claude.LoadPlugins(claudeDir)

	// Sort plugins alphabetically
	sortedPlugins := make([]claude.MarketplacePluginInfo, len(index.Plugins))
	copy(sortedPlugins, index.Plugins)
	sort.Slice(sortedPlugins, func(i, j int) bool {
		return sortedPlugins[i].Name < sortedPlugins[j].Name
	})

	// Display based on format
	if pluginBrowseFormat == "table" {
		printBrowseTable(sortedPlugins, index.Name, marketplaceName, plugins)
	} else {
		printBrowseDefault(sortedPlugins, index.Name, marketplaceName, plugins)
	}

	return nil
```

**Step 2: Add helper functions for output**

```go
func printBrowseDefault(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	fmt.Printf("Available in %s (%d plugins)\n\n", indexName, len(plugins))

	for _, p := range plugins {
		// Check if installed
		fullName := p.Name + "@" + marketplaceName
		status := ""
		if installed != nil && installed.PluginExists(fullName) {
			status = "  [installed]"
		}

		// Truncate description if needed
		desc := p.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}

		fmt.Printf("  %-30s %-42s %s%s\n", p.Name, desc, p.Version, status)
	}
}

func printBrowseTable(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	fmt.Printf("PLUGIN                         DESCRIPTION                                VERSION    STATUS\n")

	for _, p := range plugins {
		fullName := p.Name + "@" + marketplaceName
		status := ""
		if installed != nil && installed.PluginExists(fullName) {
			status = "installed"
		}

		desc := p.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}

		fmt.Printf("%-30s %-42s %-10s %s\n", p.Name, desc, p.Version, status)
	}
}
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Run acceptance tests**

Run: `go test ./test/acceptance/... -v -run "plugin browse"`
Expected: Most tests pass

**Step 5: Commit**

```bash
git add internal/commands/plugin.go
git commit -m "feat: implement browse command output display"
```

---

## Task 9: Run All Tests and Fix Issues

**Files:**
- May need to adjust test expectations or implementation

**Step 1: Run all acceptance tests**

Run: `go test ./test/acceptance/... -v -run "plugin browse"`
Expected: All tests pass

**Step 2: Run full test suite**

Run: `go test ./...`
Expected: All tests pass

**Step 3: Fix any failing tests**

Adjust implementation or test expectations as needed.

**Step 4: Commit fixes if any**

```bash
git add -A
git commit -m "fix: address test failures in plugin browse"
```

---

## Task 10: Final Verification and Squash

**Step 1: Run full test suite**

Run: `go test ./...`
Expected: All tests pass

**Step 2: Build and test manually**

```bash
go build -o bin/claudeup ./cmd/claudeup
./bin/claudeup plugin browse --help
```
Expected: Help text displays correctly

**Step 3: Squash commits for clean PR**

```bash
git rebase -i main
# Squash all commits into one with message:
# feat: add plugin browse command (#89)
#
# Add `claudeup plugin browse <marketplace>` to list available plugins
# from a marketplace before installing.
#
# - Accepts marketplace name, repo, or URL as identifier
# - Shows plugin name, description, version, and installation status
# - Supports --format table for compact output
# - Error messages guide users to `claude marketplace add`
```

**Step 4: Push branch**

```bash
git push -u origin feature/plugin-browse-89
```
