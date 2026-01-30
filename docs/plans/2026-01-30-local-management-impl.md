# Local Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `claudeup local` subcommand for managing local Claude Code extensions (agents, commands, skills, hooks, rules, output-styles), plus profile integration for `local` items and `settingsHooks`.

**Architecture:** New `internal/local` package handles symlink-based item management using `~/.claude/enabled.json`. CLI commands in `internal/commands/local.go`. Profile struct extended with `Local` and `SettingsHooks` fields.

**Tech Stack:** Go, Cobra CLI, filepath for symlinks, JSON for config

---

## Task 1: Create `internal/local` Package - Core Types

**Files:**
- Create: `internal/local/local.go`
- Test: `internal/local/local_test.go`

**Step 1: Write the failing test**

```go
// internal/local/local_test.go
// ABOUTME: Tests for local package core types and category validation
// ABOUTME: Verifies category constants and validation logic
package local

import "testing"

func TestCategoryValidation(t *testing.T) {
	tests := []struct {
		name     string
		category string
		wantErr  bool
	}{
		{"valid agents", "agents", false},
		{"valid commands", "commands", false},
		{"valid skills", "skills", false},
		{"valid hooks", "hooks", false},
		{"valid rules", "rules", false},
		{"valid output-styles", "output-styles", false},
		{"invalid category", "invalid", true},
		{"empty category", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCategory(tt.category)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCategory(%q) error = %v, wantErr %v", tt.category, err, tt.wantErr)
			}
		})
	}
}

func TestAllCategories(t *testing.T) {
	categories := AllCategories()
	if len(categories) != 6 {
		t.Errorf("AllCategories() returned %d categories, want 6", len(categories))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v`
Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

```go
// internal/local/local.go
// ABOUTME: Core types for managing local Claude Code extensions
// ABOUTME: Defines categories and provides validation
package local

import (
	"fmt"
	"sort"
)

// Category constants for local item types
const (
	CategoryAgents      = "agents"
	CategoryCommands    = "commands"
	CategorySkills      = "skills"
	CategoryHooks       = "hooks"
	CategoryRules       = "rules"
	CategoryOutputStyles = "output-styles"
)

var validCategories = map[string]bool{
	CategoryAgents:       true,
	CategoryCommands:     true,
	CategorySkills:       true,
	CategoryHooks:        true,
	CategoryRules:        true,
	CategoryOutputStyles: true,
}

// ValidateCategory checks if a category name is valid
func ValidateCategory(category string) error {
	if category == "" {
		return fmt.Errorf("category cannot be empty")
	}
	if !validCategories[category] {
		return fmt.Errorf("invalid category %q, valid categories: %v", category, AllCategories())
	}
	return nil
}

// AllCategories returns all valid category names sorted alphabetically
func AllCategories() []string {
	cats := make([]string, 0, len(validCategories))
	for cat := range validCategories {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/local/local.go internal/local/local_test.go
git commit -m "$(cat <<'EOF'
feat(local): add core types and category validation

Introduces internal/local package with category constants for agents,
commands, skills, hooks, rules, and output-styles. Includes validation
function and AllCategories helper.
EOF
)"
```

---

## Task 2: Create Manager Struct and Config Loading

**Files:**
- Modify: `internal/local/local.go`
- Create: `internal/local/config.go`
- Test: `internal/local/config_test.go`

**Step 1: Write the failing test**

```go
// internal/local/config_test.go
// ABOUTME: Tests for enabled.json config loading and saving
// ABOUTME: Verifies round-trip serialization and default behavior
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should return empty config, not error
	if config == nil {
		t.Fatal("LoadConfig() returned nil config")
	}
	if len(config) != 0 {
		t.Errorf("LoadConfig() returned non-empty config: %v", config)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create test config
	config := Config{
		"agents": {
			"gsd-planner.md": true,
			"gsd-executor.md": false,
		},
		"commands": {
			"gsd/new-project.md": true,
		},
	}

	// Save
	if err := manager.SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, "enabled.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("enabled.json was not created")
	}

	// Load back
	loaded, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify contents
	if !loaded["agents"]["gsd-planner.md"] {
		t.Error("agents/gsd-planner.md should be true")
	}
	if loaded["agents"]["gsd-executor.md"] {
		t.Error("agents/gsd-executor.md should be false")
	}
	if !loaded["commands"]["gsd/new-project.md"] {
		t.Error("commands/gsd/new-project.md should be true")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestLoadConfig`
Expected: FAIL - NewManager undefined

**Step 3: Write minimal implementation**

```go
// internal/local/config.go
// ABOUTME: Manages enabled.json configuration file for local items
// ABOUTME: Provides load/save operations for tracking enabled state
package local

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config maps category -> item -> enabled status
type Config map[string]map[string]bool

// Manager handles local item operations
type Manager struct {
	claudeDir  string
	libraryDir string
	configFile string
}

// NewManager creates a new Manager for the given Claude directory
func NewManager(claudeDir string) *Manager {
	return &Manager{
		claudeDir:  claudeDir,
		libraryDir: filepath.Join(claudeDir, ".library"),
		configFile: filepath.Join(claudeDir, "enabled.json"),
	}
}

// LoadConfig reads the enabled.json config file
func (m *Manager) LoadConfig() (Config, error) {
	data, err := os.ReadFile(m.configFile)
	if os.IsNotExist(err) {
		return make(Config), nil
	}
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Ensure all categories have initialized maps
	if config == nil {
		config = make(Config)
	}
	return config, nil
}

// SaveConfig writes the enabled.json config file
func (m *Manager) SaveConfig(config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(m.configFile, data, 0644)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/local/config.go internal/local/config_test.go
git commit -m "$(cat <<'EOF'
feat(local): add Manager and config loading

Introduces Manager struct for handling local item operations and
Config type for tracking enabled state. Loads/saves enabled.json
with JSON marshaling.
EOF
)"
```

---

## Task 3: List Library Items

**Files:**
- Create: `internal/local/list.go`
- Test: `internal/local/list_test.go`

**Step 1: Write the failing test**

```go
// internal/local/list_test.go
// ABOUTME: Tests for listing library items
// ABOUTME: Verifies directory scanning and item discovery
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListItems(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create some agent files
	os.WriteFile(filepath.Join(agentsDir, "planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(agentsDir, ".hidden.md"), []byte("# Hidden"), 0644) // Should be excluded

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	if len(items) != 2 {
		t.Errorf("ListItems() returned %d items, want 2", len(items))
	}

	// Should be sorted
	if items[0] != "executor.md" || items[1] != "planner.md" {
		t.Errorf("ListItems() = %v, want [executor.md, planner.md]", items)
	}
}

func TestListItemsWithGroups(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure with groups (for agents)
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	groupDir := filepath.Join(agentsDir, "business-product")
	os.MkdirAll(groupDir, 0755)

	// Flat agents
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# GSD Planner"), 0644)

	// Grouped agents
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)
	os.WriteFile(filepath.Join(groupDir, "strategist.md"), []byte("# Strategist"), 0644)

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	// Should include both flat and grouped items
	expected := []string{
		"business-product/analyst.md",
		"business-product/strategist.md",
		"gsd-planner.md",
	}

	if len(items) != len(expected) {
		t.Errorf("ListItems() returned %d items, want %d: %v", len(items), len(expected), items)
	}

	for i, want := range expected {
		if i < len(items) && items[i] != want {
			t.Errorf("ListItems()[%d] = %q, want %q", i, items[i], want)
		}
	}
}

func TestListItemsEmptyCategory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("ListItems() returned %d items for empty category, want 0", len(items))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestListItems`
Expected: FAIL - ListItems undefined

**Step 3: Write minimal implementation**

```go
// internal/local/list.go
// ABOUTME: Functions for listing items in the library
// ABOUTME: Handles both flat items and grouped items (like agents)
package local

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListItems returns all items in the library for a category.
// For agents, returns 'group/agent.md' format for grouped items.
// Excludes hidden files (starting with .) and CLAUDE.md.
func (m *Manager) ListItems(category string) ([]string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, err
	}

	libPath := filepath.Join(m.libraryDir, category)
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	var items []string

	if category == CategoryAgents {
		// Agents can have groups (subdirectories)
		items = m.listAgentItems(libPath)
	} else {
		// Other categories are flat
		items = m.listFlatItems(libPath)
	}

	sort.Strings(items)
	return items, nil
}

func (m *Manager) listFlatItems(dir string) []string {
	var items []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return items
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "CLAUDE.md" {
			continue
		}
		items = append(items, name)
	}

	return items
}

func (m *Manager) listAgentItems(dir string) []string {
	var items []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return items
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "CLAUDE.md" {
			continue
		}

		if entry.IsDir() {
			// This is a group directory - list agents inside
			groupPath := filepath.Join(dir, name)
			groupEntries, err := os.ReadDir(groupPath)
			if err != nil {
				continue
			}
			for _, groupEntry := range groupEntries {
				agentName := groupEntry.Name()
				if strings.HasPrefix(agentName, ".") || agentName == "CLAUDE.md" {
					continue
				}
				if strings.HasSuffix(agentName, ".md") {
					items = append(items, name+"/"+agentName)
				}
			}
		} else if strings.HasSuffix(name, ".md") {
			// Flat agent file
			items = append(items, name)
		}
	}

	return items
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestListItems`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/local/list.go internal/local/list_test.go
git commit -m "$(cat <<'EOF'
feat(local): add ListItems for discovering library items

Lists items from .library directory with support for agent groups
(subdirectories). Excludes hidden files and CLAUDE.md. Returns sorted
list with group/item.md format for grouped agents.
EOF
)"
```

---

## Task 4: Item Name Resolution

**Files:**
- Create: `internal/local/resolve.go`
- Test: `internal/local/resolve_test.go`

**Step 1: Write the failing test**

```go
// internal/local/resolve_test.go
// ABOUTME: Tests for resolving item names with extension inference
// ABOUTME: Verifies partial name matching and agent group resolution
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveItemName(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	hooksDir := filepath.Join(libraryDir, "hooks")
	os.MkdirAll(hooksDir, 0755)

	os.WriteFile(filepath.Join(hooksDir, "format-on-save.sh"), []byte("#!/bin/bash"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gsd-check-update.js"), []byte("// js"), 0644)

	tests := []struct {
		name     string
		category string
		item     string
		want     string
		wantErr  bool
	}{
		{"exact match", "hooks", "format-on-save.sh", "format-on-save.sh", false},
		{"without extension .sh", "hooks", "format-on-save", "format-on-save.sh", false},
		{"without extension .js", "hooks", "gsd-check-update", "gsd-check-update.js", false},
		{"nonexistent", "hooks", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.ResolveItemName(tt.category, tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveItemName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveItemName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveAgentName(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure with groups
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	groupDir := filepath.Join(agentsDir, "business-product")
	os.MkdirAll(groupDir, 0755)

	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)

	tests := []struct {
		name    string
		item    string
		want    string
		wantErr bool
	}{
		{"flat agent with ext", "gsd-planner.md", "gsd-planner.md", false},
		{"flat agent without ext", "gsd-planner", "gsd-planner.md", false},
		{"grouped agent full path", "business-product/analyst", "business-product/analyst.md", false},
		{"grouped agent with ext", "business-product/analyst.md", "business-product/analyst.md", false},
		{"agent name search", "analyst", "business-product/analyst.md", false},
		{"nonexistent", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.ResolveItemName("agents", tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveItemName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveItemName() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestResolve`
Expected: FAIL - ResolveItemName undefined

**Step 3: Write minimal implementation**

```go
// internal/local/resolve.go
// ABOUTME: Resolves item names with extension inference
// ABOUTME: Handles partial matches and agent group resolution
package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveItemName resolves an item name, allowing partial matches without extension.
// Returns the full item name if found, error otherwise.
// For agents, returns 'group/agent.md' format.
func (m *Manager) ResolveItemName(category, item string) (string, error) {
	if err := ValidateCategory(category); err != nil {
		return "", err
	}

	if category == CategoryAgents {
		return m.resolveAgentName(item)
	}

	return m.resolveFlatItem(category, item)
}

func (m *Manager) resolveFlatItem(category, item string) (string, error) {
	libPath := filepath.Join(m.libraryDir, category, item)

	// Try exact match first
	if _, err := os.Stat(libPath); err == nil {
		return item, nil
	}

	// Try common extensions
	extensions := []string{".md", ".py", ".sh", ".js"}
	for _, ext := range extensions {
		fullName := item + ext
		fullPath := filepath.Join(m.libraryDir, category, fullName)
		if _, err := os.Stat(fullPath); err == nil {
			return fullName, nil
		}
	}

	return "", fmt.Errorf("item not found: %s/%s", category, item)
}

func (m *Manager) resolveAgentName(item string) (string, error) {
	agentsDir := filepath.Join(m.libraryDir, CategoryAgents)

	// Handle 'group/agent' format
	if strings.Contains(item, "/") {
		parts := strings.SplitN(item, "/", 2)
		group, agent := parts[0], parts[1]

		if !strings.HasSuffix(agent, ".md") {
			agent = agent + ".md"
		}

		fullPath := filepath.Join(agentsDir, group, agent)
		if _, err := os.Stat(fullPath); err == nil {
			return group + "/" + agent, nil
		}
		return "", fmt.Errorf("agent not found: agents/%s/%s", group, agent)
	}

	// Try as flat agent first
	agentName := item
	if !strings.HasSuffix(agentName, ".md") {
		agentName = agentName + ".md"
	}

	flatPath := filepath.Join(agentsDir, agentName)
	if _, err := os.Stat(flatPath); err == nil {
		return agentName, nil
	}

	// Search all groups for the agent
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return "", fmt.Errorf("agent not found: agents/%s", item)
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		groupPath := filepath.Join(agentsDir, entry.Name(), agentName)
		if _, err := os.Stat(groupPath); err == nil {
			return entry.Name() + "/" + agentName, nil
		}
	}

	return "", fmt.Errorf("agent not found: agents/%s", item)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestResolve`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/local/resolve.go internal/local/resolve_test.go
git commit -m "$(cat <<'EOF'
feat(local): add item name resolution with extension inference

ResolveItemName handles partial names (without extension) and searches
agent groups when given just an agent name. Supports common extensions
(.md, .py, .sh, .js).
EOF
)"
```

---

## Task 5: Wildcard Pattern Matching

**Files:**
- Create: `internal/local/wildcard.go`
- Test: `internal/local/wildcard_test.go`

**Step 1: Write the failing test**

```go
// internal/local/wildcard_test.go
// ABOUTME: Tests for wildcard pattern matching
// ABOUTME: Verifies prefix (gsd-*) and directory (gsd/*) wildcards
package local

import "testing"

func TestMatchWildcard(t *testing.T) {
	items := []string{
		"gsd-planner.md",
		"gsd-executor.md",
		"gsd-verifier.md",
		"other-agent.md",
		"business-product/analyst.md",
		"business-product/strategist.md",
		"gsd/new-project.md",
		"gsd/execute-phase.md",
	}

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			"prefix wildcard",
			"gsd-*",
			[]string{"gsd-executor.md", "gsd-planner.md", "gsd-verifier.md"},
		},
		{
			"directory wildcard",
			"business-product/*",
			[]string{"business-product/analyst.md", "business-product/strategist.md"},
		},
		{
			"directory wildcard for commands",
			"gsd/*",
			[]string{"gsd/execute-phase.md", "gsd/new-project.md"},
		},
		{
			"global wildcard",
			"*",
			items,
		},
		{
			"exact match",
			"gsd-planner.md",
			[]string{"gsd-planner.md"},
		},
		{
			"no match",
			"nonexistent-*",
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchWildcard(tt.pattern, items)
			if len(got) != len(tt.want) {
				t.Errorf("MatchWildcard(%q) returned %d items, want %d: %v", tt.pattern, len(got), len(tt.want), got)
				return
			}
			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("MatchWildcard(%q)[%d] = %q, want %q", tt.pattern, i, got[i], want)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestMatchWildcard`
Expected: FAIL - MatchWildcard undefined

**Step 3: Write minimal implementation**

```go
// internal/local/wildcard.go
// ABOUTME: Wildcard pattern matching for item selection
// ABOUTME: Supports prefix (gsd-*), directory (gsd/*), and global (*) wildcards
package local

import (
	"sort"
	"strings"
)

// MatchWildcard returns items matching the given pattern.
// Patterns:
//   - "gsd-*" matches items starting with "gsd-"
//   - "gsd/*" matches items in the "gsd/" directory
//   - "*" matches all items
//   - exact string matches that specific item
func MatchWildcard(pattern string, items []string) []string {
	var matched []string

	// Global wildcard
	if pattern == "*" {
		result := make([]string, len(items))
		copy(result, items)
		sort.Strings(result)
		return result
	}

	// Directory wildcard: "group/*"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "*")
		for _, item := range items {
			if strings.HasPrefix(item, prefix) {
				matched = append(matched, item)
			}
		}
		sort.Strings(matched)
		return matched
	}

	// Prefix wildcard: "prefix*"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		for _, item := range items {
			// For prefix matching, only match the base name (not path)
			baseName := item
			if idx := strings.LastIndex(item, "/"); idx >= 0 {
				baseName = item[idx+1:]
			}
			if strings.HasPrefix(baseName, prefix) {
				matched = append(matched, item)
			}
		}
		sort.Strings(matched)
		return matched
	}

	// Exact match
	for _, item := range items {
		if item == pattern {
			return []string{item}
		}
	}

	return []string{}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestMatchWildcard`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/local/wildcard.go internal/local/wildcard_test.go
git commit -m "$(cat <<'EOF'
feat(local): add wildcard pattern matching

MatchWildcard supports prefix wildcards (gsd-*), directory wildcards
(gsd/*), global wildcard (*), and exact matches. Returns sorted results.
EOF
)"
```

---

## Task 6: Enable/Disable with Symlinks

**Files:**
- Create: `internal/local/symlinks.go`
- Test: `internal/local/symlinks_test.go`

**Step 1: Write the failing test**

```go
// internal/local/symlinks_test.go
// ABOUTME: Tests for symlink-based enable/disable operations
// ABOUTME: Verifies symlink creation, removal, and sync behavior
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnableDisable(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	hooksDir := filepath.Join(libraryDir, "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "format-on-save.sh"), []byte("#!/bin/bash"), 0644)

	// Enable
	enabled, notFound, err := manager.Enable("hooks", []string{"format-on-save"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 1 || enabled[0] != "format-on-save.sh" {
		t.Errorf("Enable() enabled = %v, want [format-on-save.sh]", enabled)
	}
	if len(notFound) != 0 {
		t.Errorf("Enable() notFound = %v, want []", notFound)
	}

	// Verify symlink exists
	targetDir := filepath.Join(tmpDir, "hooks")
	symlinkPath := filepath.Join(targetDir, "format-on-save.sh")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created")
	}

	// Verify config was updated
	config, _ := manager.LoadConfig()
	if !config["hooks"]["format-on-save.sh"] {
		t.Error("Config was not updated")
	}

	// Disable
	disabled, notFound, err := manager.Disable("hooks", []string{"format-on-save"})
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if len(disabled) != 1 {
		t.Errorf("Disable() disabled = %v, want [format-on-save.sh]", disabled)
	}

	// Verify symlink removed
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error("Symlink was not removed")
	}

	// Verify config was updated
	config, _ = manager.LoadConfig()
	if config["hooks"]["format-on-save.sh"] {
		t.Error("Config still shows enabled")
	}
}

func TestEnableAgentWithGroup(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure with groups
	libraryDir := filepath.Join(tmpDir, ".library")
	groupDir := filepath.Join(libraryDir, "agents", "business-product")
	os.MkdirAll(groupDir, 0755)
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)

	// Enable grouped agent
	enabled, _, err := manager.Enable("agents", []string{"business-product/analyst"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 1 || enabled[0] != "business-product/analyst.md" {
		t.Errorf("Enable() enabled = %v, want [business-product/analyst.md]", enabled)
	}

	// Verify symlink exists in correct location
	symlinkPath := filepath.Join(tmpDir, "agents", "business-product", "analyst.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created in group directory")
	}

	// Verify it's a symlink pointing to the right place
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	expectedTarget := filepath.Join("..", "..", ".library", "agents", "business-product", "analyst.md")
	if target != expectedTarget {
		t.Errorf("Symlink target = %q, want %q", target, expectedTarget)
	}
}

func TestEnableWildcard(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "other-agent.md"), []byte("# Other"), 0644)

	// Enable with wildcard
	enabled, _, err := manager.Enable("agents", []string{"gsd-*"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("Enable() enabled %d items, want 2: %v", len(enabled), enabled)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run "TestEnable|TestDisable"`
Expected: FAIL - Enable/Disable undefined

**Step 3: Write minimal implementation**

```go
// internal/local/symlinks.go
// ABOUTME: Symlink-based enable/disable for local items
// ABOUTME: Creates relative symlinks from target dirs to .library
package local

import (
	"os"
	"path/filepath"
	"strings"
)

// Enable enables items matching the given patterns.
// Returns (enabled items, not found patterns, error).
func (m *Manager) Enable(category string, patterns []string) ([]string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, err
	}

	config, err := m.LoadConfig()
	if err != nil {
		return nil, nil, err
	}

	// Initialize category map if needed
	if config[category] == nil {
		config[category] = make(map[string]bool)
	}

	allItems, err := m.ListItems(category)
	if err != nil {
		return nil, nil, err
	}

	var enabled []string
	var notFound []string

	for _, pattern := range patterns {
		matched := MatchWildcard(pattern, allItems)
		if len(matched) == 0 {
			// Try to resolve as a single item
			resolved, err := m.ResolveItemName(category, pattern)
			if err != nil {
				notFound = append(notFound, pattern)
				continue
			}
			matched = []string{resolved}
		}

		for _, item := range matched {
			config[category][item] = true
			enabled = append(enabled, item)
		}
	}

	if len(enabled) > 0 {
		if err := m.SaveConfig(config); err != nil {
			return nil, nil, err
		}
		if err := m.syncCategory(category, config); err != nil {
			return nil, nil, err
		}
	}

	return enabled, notFound, nil
}

// Disable disables items matching the given patterns.
// Returns (disabled items, not found patterns, error).
func (m *Manager) Disable(category string, patterns []string) ([]string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, err
	}

	config, err := m.LoadConfig()
	if err != nil {
		return nil, nil, err
	}

	if config[category] == nil {
		config[category] = make(map[string]bool)
	}

	allItems, err := m.ListItems(category)
	if err != nil {
		return nil, nil, err
	}

	var disabled []string
	var notFound []string

	for _, pattern := range patterns {
		matched := MatchWildcard(pattern, allItems)
		if len(matched) == 0 {
			resolved, err := m.ResolveItemName(category, pattern)
			if err != nil {
				notFound = append(notFound, pattern)
				continue
			}
			matched = []string{resolved}
		}

		for _, item := range matched {
			config[category][item] = false
			disabled = append(disabled, item)
		}
	}

	if len(disabled) > 0 {
		if err := m.SaveConfig(config); err != nil {
			return nil, nil, err
		}
		if err := m.syncCategory(category, config); err != nil {
			return nil, nil, err
		}
	}

	return disabled, notFound, nil
}

// syncCategory creates/removes symlinks based on config state
func (m *Manager) syncCategory(category string, config Config) error {
	targetDir := filepath.Join(m.claudeDir, category)

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	catConfig := config[category]
	if catConfig == nil {
		catConfig = make(map[string]bool)
	}

	if category == CategoryAgents {
		return m.syncAgents(targetDir, catConfig)
	}

	return m.syncFlatCategory(category, targetDir, catConfig)
}

func (m *Manager) syncFlatCategory(category string, targetDir string, catConfig map[string]bool) error {
	// Remove existing symlinks
	entries, _ := os.ReadDir(targetDir)
	for _, entry := range entries {
		path := filepath.Join(targetDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(path)
		}
	}

	// Create symlinks for enabled items
	for item, enabled := range catConfig {
		if !enabled {
			continue
		}

		target := filepath.Join(targetDir, item)
		// Relative path: ../.library/{category}/{item}
		relSource := filepath.Join("..", ".library", category, item)
		if err := os.Symlink(relSource, target); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) syncAgents(targetDir string, catConfig map[string]bool) error {
	// Remove existing symlinks and empty group directories
	entries, _ := os.ReadDir(targetDir)
	for _, entry := range entries {
		path := filepath.Join(targetDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(path)
		} else if entry.IsDir() {
			// Remove symlinks inside group directories
			groupEntries, _ := os.ReadDir(path)
			for _, ge := range groupEntries {
				gePath := filepath.Join(path, ge.Name())
				geInfo, _ := os.Lstat(gePath)
				if geInfo != nil && geInfo.Mode()&os.ModeSymlink != 0 {
					os.Remove(gePath)
				}
			}
			// Remove group dir if empty
			remaining, _ := os.ReadDir(path)
			if len(remaining) == 0 {
				os.Remove(path)
			}
		}
	}

	// Create symlinks for enabled agents
	for item, enabled := range catConfig {
		if !enabled {
			continue
		}

		if strings.Contains(item, "/") {
			// Grouped agent: group/agent.md
			parts := strings.SplitN(item, "/", 2)
			group, agent := parts[0], parts[1]

			groupTargetDir := filepath.Join(targetDir, group)
			if err := os.MkdirAll(groupTargetDir, 0755); err != nil {
				return err
			}

			target := filepath.Join(groupTargetDir, agent)
			// Relative path: ../../.library/agents/{group}/{agent}
			relSource := filepath.Join("..", "..", ".library", "agents", group, agent)
			if err := os.Symlink(relSource, target); err != nil {
				return err
			}
		} else {
			// Flat agent
			target := filepath.Join(targetDir, item)
			relSource := filepath.Join("..", ".library", "agents", item)
			if err := os.Symlink(relSource, target); err != nil {
				return err
			}
		}
	}

	return nil
}

// Sync synchronizes all categories from config to symlinks
func (m *Manager) Sync() error {
	config, err := m.LoadConfig()
	if err != nil {
		return err
	}

	for _, category := range AllCategories() {
		if err := m.syncCategory(category, config); err != nil {
			return err
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/local/symlinks.go internal/local/symlinks_test.go
git commit -m "$(cat <<'EOF'
feat(local): add Enable/Disable with symlink management

Enable and Disable operations update enabled.json and sync symlinks.
Handles both flat categories and agents with group subdirectories.
Uses relative symlinks for portability.
EOF
)"
```

---

## Task 7: View Item Contents

**Files:**
- Create: `internal/local/view.go`
- Test: `internal/local/view_test.go`

**Step 1: Write the failing test**

```go
// internal/local/view_test.go
// ABOUTME: Tests for viewing item contents
// ABOUTME: Verifies content retrieval for different item types
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestView(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	hooksDir := filepath.Join(libraryDir, "hooks")
	skillsDir := filepath.Join(libraryDir, "skills", "bash")
	os.MkdirAll(hooksDir, 0755)
	os.MkdirAll(skillsDir, 0755)

	os.WriteFile(filepath.Join(hooksDir, "format-on-save.sh"), []byte("#!/bin/bash\necho hello"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# Bash Skill\nDescription here"), 0644)

	// View hook
	content, err := manager.View("hooks", "format-on-save")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "#!/bin/bash\necho hello" {
		t.Errorf("View() content = %q, want script content", content)
	}

	// View skill (directory with SKILL.md)
	content, err = manager.View("skills", "bash")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "# Bash Skill\nDescription here" {
		t.Errorf("View() content = %q, want skill content", content)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestView`
Expected: FAIL - View undefined

**Step 3: Write minimal implementation**

```go
// internal/local/view.go
// ABOUTME: View item contents from the library
// ABOUTME: Handles files and skill directories
package local

import (
	"fmt"
	"os"
	"path/filepath"
)

// View returns the contents of an item.
// For skills (which are directories), returns SKILL.md content.
// For agents, returns the .md file content.
// For other categories, returns the file content.
func (m *Manager) View(category, item string) (string, error) {
	if err := ValidateCategory(category); err != nil {
		return "", err
	}

	if category == CategorySkills {
		// Skills are directories with SKILL.md inside
		skillDir := filepath.Join(m.libraryDir, category, item)
		if info, err := os.Stat(skillDir); err == nil && info.IsDir() {
			skillFile := filepath.Join(skillDir, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				return "", fmt.Errorf("skill not found: %s", item)
			}
			return string(data), nil
		}
		return "", fmt.Errorf("skill not found: %s", item)
	}

	// Resolve item name (handles missing extensions)
	resolved, err := m.ResolveItemName(category, item)
	if err != nil {
		return "", err
	}

	// For agents, handle group paths
	var filePath string
	if category == CategoryAgents {
		filePath = filepath.Join(m.libraryDir, category, resolved)
	} else {
		filePath = filepath.Join(m.libraryDir, category, resolved)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("item not found: %s/%s", category, item)
	}

	return string(data), nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/local/... -v -run TestView`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/local/view.go internal/local/view_test.go
git commit -m "$(cat <<'EOF'
feat(local): add View for reading item contents

View returns file contents for hooks/commands/rules/output-styles,
SKILL.md for skills directories, and agent definitions.
EOF
)"
```

---

## Task 8: CLI Commands - local list

**Files:**
- Create: `internal/commands/local.go`

**Step 1: Write the failing test**

For CLI commands, we'll do integration testing. First, let's create the command structure.

**Step 2: Write implementation**

```go
// internal/commands/local.go
// ABOUTME: CLI commands for managing local Claude Code extensions
// ABOUTME: Provides list, enable, disable, view, and sync subcommands
package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v4/internal/local"
	"github.com/claudeup/claudeup/v4/internal/ui"
	"github.com/spf13/cobra"
)

var (
	localFilterEnabled  bool
	localFilterDisabled bool
)

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Manage local extensions (agents, commands, skills, hooks, rules, output-styles)",
	Long: `Manage local Claude Code extensions from ~/.claude/.library.

These are local files (not marketplace plugins) that extend Claude Code
with custom agents, commands, skills, hooks, rules, and output-styles.`,
}

var localListCmd = &cobra.Command{
	Use:   "list [category]",
	Short: "List local items and their enabled status",
	Long: `List all local items in the library and their enabled status.

Optionally filter by category. Use --enabled or --disabled to filter by status.`,
	Example: `  claudeup local list
  claudeup local list agents
  claudeup local list --enabled
  claudeup local list hooks --disabled`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLocalList,
}

var localEnableCmd = &cobra.Command{
	Use:   "enable <category> <items...>",
	Short: "Enable local items",
	Long: `Enable one or more local items by creating symlinks.

Supports wildcards:
  - gsd-* matches items starting with "gsd-"
  - gsd/* matches all items in the "gsd/" directory
  - * matches all items in the category`,
	Example: `  claudeup local enable agents gsd-*
  claudeup local enable commands gsd/*
  claudeup local enable hooks format-on-save`,
	Args: cobra.MinimumNArgs(2),
	RunE: runLocalEnable,
}

var localDisableCmd = &cobra.Command{
	Use:   "disable <category> <items...>",
	Short: "Disable local items",
	Long: `Disable one or more local items by removing symlinks.

Supports the same wildcards as enable.`,
	Example: `  claudeup local disable agents gsd-*
  claudeup local disable hooks gsd-check-update`,
	Args: cobra.MinimumNArgs(2),
	RunE: runLocalDisable,
}

var localViewCmd = &cobra.Command{
	Use:   "view <category> <item>",
	Short: "View contents of a local item",
	Long:  `Display the contents of a local item from the library.`,
	Example: `  claudeup local view agents gsd-planner
  claudeup local view hooks format-on-save
  claudeup local view skills bash`,
	Args: cobra.ExactArgs(2),
	RunE: runLocalView,
}

var localSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync symlinks from enabled.json",
	Long:  `Recreate all symlinks based on the enabled.json configuration.`,
	Args:  cobra.NoArgs,
	RunE:  runLocalSync,
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.AddCommand(localListCmd)
	localCmd.AddCommand(localEnableCmd)
	localCmd.AddCommand(localDisableCmd)
	localCmd.AddCommand(localViewCmd)
	localCmd.AddCommand(localSyncCmd)

	localListCmd.Flags().BoolVarP(&localFilterEnabled, "enabled", "e", false, "Show only enabled items")
	localListCmd.Flags().BoolVarP(&localFilterDisabled, "disabled", "d", false, "Show only disabled items")
}

func runLocalList(cmd *cobra.Command, args []string) error {
	if localFilterEnabled && localFilterDisabled {
		return fmt.Errorf("--enabled and --disabled are mutually exclusive")
	}

	manager := local.NewManager(claudeDir)
	config, err := manager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var categories []string
	if len(args) > 0 {
		if err := local.ValidateCategory(args[0]); err != nil {
			return err
		}
		categories = []string{args[0]}
	} else {
		categories = local.AllCategories()
	}

	for _, category := range categories {
		items, err := manager.ListItems(category)
		if err != nil {
			continue
		}

		catConfig := config[category]
		if catConfig == nil {
			catConfig = make(map[string]bool)
		}

		// Filter items by status
		type itemStatus struct {
			name    string
			enabled bool
		}
		var filtered []itemStatus

		for _, item := range items {
			enabled := catConfig[item]
			if localFilterEnabled && !enabled {
				continue
			}
			if localFilterDisabled && enabled {
				continue
			}
			filtered = append(filtered, itemStatus{item, enabled})
		}

		if len(filtered) == 0 {
			if len(args) > 0 {
				// User requested specific category
				fmt.Printf("\n%s/: (empty)\n", category)
			}
			continue
		}

		fmt.Printf("\n%s/:\n", category)

		if category == local.CategoryAgents {
			// Group agents by their group directory
			printGroupedAgents(filtered)
		} else {
			for _, item := range filtered {
				status := ui.Error("✗")
				if item.enabled {
					status = ui.Success("✓")
				}
				fmt.Printf("  %s %s\n", status, item.name)
			}
		}
	}

	return nil
}

func printGroupedAgents(items []struct{ name string; enabled bool }) {
	// Group by directory
	groups := make(map[string][]struct{ name string; enabled bool })
	var flatItems []struct{ name string; enabled bool }

	for _, item := range items {
		if strings.Contains(item.name, "/") {
			parts := strings.SplitN(item.name, "/", 2)
			group := parts[0]
			groups[group] = append(groups[group], struct{ name string; enabled bool }{
				name:    parts[1],
				enabled: item.enabled,
			})
		} else {
			flatItems = append(flatItems, item)
		}
	}

	// Print flat items first
	for _, item := range flatItems {
		status := ui.Error("✗")
		if item.enabled {
			status = ui.Success("✓")
		}
		fmt.Printf("  %s %s\n", status, item.name)
	}

	// Print grouped items
	groupNames := make([]string, 0, len(groups))
	for g := range groups {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	for _, group := range groupNames {
		fmt.Printf("  %s/\n", group)
		for _, item := range groups[group] {
			status := ui.Error("✗")
			if item.enabled {
				status = ui.Success("✓")
			}
			fmt.Printf("    %s %s\n", status, strings.TrimSuffix(item.name, ".md"))
		}
	}
}

func runLocalEnable(cmd *cobra.Command, args []string) error {
	category := args[0]
	patterns := args[1:]

	manager := local.NewManager(claudeDir)
	enabled, notFound, err := manager.Enable(category, patterns)
	if err != nil {
		return err
	}

	for _, item := range enabled {
		ui.PrintSuccess(fmt.Sprintf("Enabled: %s/%s", category, item))
	}

	for _, pattern := range notFound {
		ui.PrintWarning(fmt.Sprintf("Not found: %s/%s", category, pattern))
	}

	if len(notFound) > 0 && len(enabled) == 0 {
		return fmt.Errorf("no items found matching patterns")
	}

	return nil
}

func runLocalDisable(cmd *cobra.Command, args []string) error {
	category := args[0]
	patterns := args[1:]

	manager := local.NewManager(claudeDir)
	disabled, notFound, err := manager.Disable(category, patterns)
	if err != nil {
		return err
	}

	for _, item := range disabled {
		ui.PrintSuccess(fmt.Sprintf("Disabled: %s/%s", category, item))
	}

	for _, pattern := range notFound {
		ui.PrintWarning(fmt.Sprintf("Not found: %s/%s", category, pattern))
	}

	if len(notFound) > 0 && len(disabled) == 0 {
		return fmt.Errorf("no items found matching patterns")
	}

	return nil
}

func runLocalView(cmd *cobra.Command, args []string) error {
	category := args[0]
	item := args[1]

	manager := local.NewManager(claudeDir)
	content, err := manager.View(category, item)
	if err != nil {
		return err
	}

	fmt.Println(content)
	return nil
}

func runLocalSync(cmd *cobra.Command, args []string) error {
	manager := local.NewManager(claudeDir)

	fmt.Println("Syncing local items from enabled.json...")
	if err := manager.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	ui.PrintSuccess("Sync complete")
	return nil
}
```

**Step 3: Fix compilation and run basic test**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go build ./... && ./bin/claudeup local --help`
Expected: Help output for local command

**Step 4: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/commands/local.go
git commit -m "$(cat <<'EOF'
feat(local): add CLI commands for local item management

Adds claudeup local subcommand with:
- list: Show items with enabled status and filtering
- enable: Enable items with wildcard support
- disable: Disable items with wildcard support
- view: Display item contents
- sync: Recreate symlinks from config
EOF
)"
```

---

## Task 9: Extend Profile Struct with Local and SettingsHooks

**Files:**
- Modify: `internal/profile/profile.go`
- Test: `internal/profile/profile_test.go`

**Step 1: Write the failing test**

Add to existing test file:

```go
// Add to internal/profile/profile_test.go

func TestProfileWithLocal(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	p := &Profile{
		Name:        "gsd-profile",
		Description: "Get Shit Done workflow",
		Local: &LocalSettings{
			Agents:   []string{"gsd-*"},
			Commands: []string{"gsd/*"},
			Hooks:    []string{"gsd-check-update.js"},
		},
		SettingsHooks: map[string][]HookEntry{
			"SessionStart": {
				{Type: "command", Command: "node ~/.claude/hooks/gsd-check-update.js"},
			},
		},
	}

	// Save
	err := Save(profilesDir, p)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := Load(profilesDir, "gsd-profile")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify Local
	if loaded.Local == nil {
		t.Fatal("Local is nil")
	}
	if len(loaded.Local.Agents) != 1 || loaded.Local.Agents[0] != "gsd-*" {
		t.Errorf("Local.Agents = %v, want [gsd-*]", loaded.Local.Agents)
	}

	// Verify SettingsHooks
	if loaded.SettingsHooks == nil {
		t.Fatal("SettingsHooks is nil")
	}
	if len(loaded.SettingsHooks["SessionStart"]) != 1 {
		t.Errorf("SettingsHooks[SessionStart] = %v, want 1 entry", loaded.SettingsHooks["SessionStart"])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/profile/... -v -run TestProfileWithLocal`
Expected: FAIL - LocalSettings and HookEntry undefined

**Step 3: Write minimal implementation**

Add to `internal/profile/profile.go`:

```go
// Add these types and fields to profile.go

// LocalSettings contains local item patterns to enable
type LocalSettings struct {
	Agents       []string `json:"agents,omitempty"`
	Commands     []string `json:"commands,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	Hooks        []string `json:"hooks,omitempty"`
	Rules        []string `json:"rules,omitempty"`
	OutputStyles []string `json:"output-styles,omitempty"`
}

// HookEntry represents a single hook configuration
type HookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// Add these fields to the Profile struct:
// Local contains patterns for local items to enable
// Local *LocalSettings `json:"local,omitempty"`

// SettingsHooks contains hooks to merge into settings.json
// SettingsHooks map[string][]HookEntry `json:"settingsHooks,omitempty"`
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/profile/... -v -run TestProfileWithLocal`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/profile/profile.go internal/profile/profile_test.go
git commit -m "$(cat <<'EOF'
feat(profile): add Local and SettingsHooks fields

Profile struct now supports:
- Local: patterns for enabling local items (agents, commands, etc.)
- SettingsHooks: hooks to merge into settings.json by event type
EOF
)"
```

---

## Task 10: Settings Hooks Merge in claude package

**Files:**
- Modify: `internal/claude/settings.go`
- Test: `internal/claude/settings_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/claude/settings_test.go

func TestMergeHooks(t *testing.T) {
	// Create settings with existing hooks
	settings := &Settings{
		raw: map[string]interface{}{
			"hooks": map[string]interface{}{
				"PostToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "Edit|Write",
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "command",
								"command": "~/.claude/hooks/format-on-save.sh",
							},
						},
					},
				},
			},
		},
	}

	// Merge new hooks
	newHooks := map[string][]map[string]interface{}{
		"SessionStart": {
			{"type": "command", "command": "node ~/.claude/hooks/gsd-check-update.js"},
		},
	}

	err := settings.MergeHooks(newHooks)
	if err != nil {
		t.Fatalf("MergeHooks() error = %v", err)
	}

	// Verify PostToolUse still exists
	hooks := settings.raw["hooks"].(map[string]interface{})
	if hooks["PostToolUse"] == nil {
		t.Error("PostToolUse hooks were removed")
	}

	// Verify SessionStart was added
	if hooks["SessionStart"] == nil {
		t.Error("SessionStart hooks were not added")
	}
}

func TestMergeHooksDeduplicate(t *testing.T) {
	settings := &Settings{
		raw: map[string]interface{}{
			"hooks": map[string]interface{}{
				"SessionStart": []interface{}{
					map[string]interface{}{
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "command",
								"command": "node ~/.claude/hooks/existing.js",
							},
						},
					},
				},
			},
		},
	}

	// Try to add a duplicate
	newHooks := map[string][]map[string]interface{}{
		"SessionStart": {
			{"type": "command", "command": "node ~/.claude/hooks/existing.js"},
			{"type": "command", "command": "node ~/.claude/hooks/new.js"},
		},
	}

	err := settings.MergeHooks(newHooks)
	if err != nil {
		t.Fatalf("MergeHooks() error = %v", err)
	}

	// Count hooks - should have 2 (existing + new, no duplicate)
	hooks := settings.raw["hooks"].(map[string]interface{})
	sessionStart := hooks["SessionStart"].([]interface{})

	// The first entry should have 2 hooks (deduplicated)
	totalHooks := 0
	for _, entry := range sessionStart {
		entryMap := entry.(map[string]interface{})
		hooksList := entryMap["hooks"].([]interface{})
		totalHooks += len(hooksList)
	}

	if totalHooks != 2 {
		t.Errorf("Expected 2 hooks after dedup, got %d", totalHooks)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/claude/... -v -run TestMergeHooks`
Expected: FAIL - MergeHooks undefined

**Step 3: Write minimal implementation**

Add to `internal/claude/settings.go`:

```go
// MergeHooks merges new hooks into settings, deduplicating by command string.
// Hooks are added to the settings without removing existing ones.
func (s *Settings) MergeHooks(newHooks map[string][]map[string]interface{}) error {
	if s.raw == nil {
		s.raw = make(map[string]interface{})
	}

	// Get or create hooks map
	var hooks map[string]interface{}
	if existing, ok := s.raw["hooks"].(map[string]interface{}); ok {
		hooks = existing
	} else {
		hooks = make(map[string]interface{})
		s.raw["hooks"] = hooks
	}

	// For each event type in newHooks
	for eventType, entries := range newHooks {
		// Get existing hooks for this event type
		var existingEntries []interface{}
		if existing, ok := hooks[eventType].([]interface{}); ok {
			existingEntries = existing
		}

		// Collect all existing commands for deduplication
		existingCommands := make(map[string]bool)
		for _, entry := range existingEntries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if hooksList, ok := entryMap["hooks"].([]interface{}); ok {
					for _, hook := range hooksList {
						if hookMap, ok := hook.(map[string]interface{}); ok {
							if cmd, ok := hookMap["command"].(string); ok {
								existingCommands[cmd] = true
							}
						}
					}
				}
			}
		}

		// Filter new hooks to only include non-duplicates
		var newHooksList []interface{}
		for _, entry := range entries {
			cmd := entry["command"].(string)
			if !existingCommands[cmd] {
				newHooksList = append(newHooksList, entry)
				existingCommands[cmd] = true
			}
		}

		if len(newHooksList) > 0 {
			// Add as a new entry with no matcher (applies to all)
			newEntry := map[string]interface{}{
				"hooks": newHooksList,
			}
			existingEntries = append(existingEntries, newEntry)
			hooks[eventType] = existingEntries
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/claude/... -v -run TestMergeHooks`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/claude/settings.go internal/claude/settings_test.go
git commit -m "$(cat <<'EOF'
feat(claude): add MergeHooks for settings.json hook management

MergeHooks adds hooks to settings without removing existing ones.
Deduplicates by command string to prevent running the same hook twice.
EOF
)"
```

---

## Task 11: Profile Apply - Local Items and SettingsHooks

**Files:**
- Modify: `internal/profile/apply.go`
- Test: `internal/profile/apply_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/profile/apply_test.go

func TestApplyWithLocal(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Create library structure
	libraryDir := filepath.Join(claudeDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)

	// Create profile
	p := &Profile{
		Name: "test-local",
		Local: &LocalSettings{
			Agents: []string{"gsd-*"},
		},
	}

	// Apply
	err := Apply(claudeDir, "", p, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify symlinks created
	symlinkPath := filepath.Join(claudeDir, "agents", "gsd-planner.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created for gsd-planner.md")
	}
}

func TestApplyWithSettingsHooks(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Create existing settings.json
	existingSettings := `{"enabledPlugins": {}}`
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(existingSettings), 0644)

	// Create profile with SettingsHooks
	p := &Profile{
		Name: "test-hooks",
		SettingsHooks: map[string][]HookEntry{
			"SessionStart": {
				{Type: "command", Command: "node ~/.claude/hooks/test.js"},
			},
		},
	}

	// Apply
	err := Apply(claudeDir, "", p, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify settings.json was updated
	data, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if !strings.Contains(string(data), "SessionStart") {
		t.Error("SessionStart hook was not added to settings.json")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/profile/... -v -run "TestApplyWithLocal|TestApplyWithSettingsHooks"`
Expected: FAIL - Apply doesn't handle Local or SettingsHooks

**Step 3: Write minimal implementation**

Modify the Apply function in `internal/profile/apply.go` to handle Local and SettingsHooks:

```go
// Add to Apply function in internal/profile/apply.go

// After existing plugin/MCP application logic, add:

// Apply local items
if p.Local != nil {
	localMgr := local.NewManager(claudeDir)

	// Enable agents
	if len(p.Local.Agents) > 0 {
		localMgr.Enable(local.CategoryAgents, p.Local.Agents)
	}
	// Enable commands
	if len(p.Local.Commands) > 0 {
		localMgr.Enable(local.CategoryCommands, p.Local.Commands)
	}
	// Enable skills
	if len(p.Local.Skills) > 0 {
		localMgr.Enable(local.CategorySkills, p.Local.Skills)
	}
	// Enable hooks
	if len(p.Local.Hooks) > 0 {
		localMgr.Enable(local.CategoryHooks, p.Local.Hooks)
	}
	// Enable rules
	if len(p.Local.Rules) > 0 {
		localMgr.Enable(local.CategoryRules, p.Local.Rules)
	}
	// Enable output-styles
	if len(p.Local.OutputStyles) > 0 {
		localMgr.Enable(local.CategoryOutputStyles, p.Local.OutputStyles)
	}
}

// Apply settings hooks
if len(p.SettingsHooks) > 0 {
	settings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		// Create new settings if none exist
		settings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	// Convert HookEntry to map format for MergeHooks
	hooksMap := make(map[string][]map[string]interface{})
	for eventType, entries := range p.SettingsHooks {
		for _, entry := range entries {
			hooksMap[eventType] = append(hooksMap[eventType], map[string]interface{}{
				"type":    entry.Type,
				"command": entry.Command,
			})
		}
	}

	if err := settings.MergeHooks(hooksMap); err != nil {
		return fmt.Errorf("failed to merge hooks: %w", err)
	}

	if err := claude.SaveSettings(claudeDir, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/profile/... -v -run "TestApplyWithLocal|TestApplyWithSettingsHooks"`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/profile/apply.go internal/profile/apply_test.go
git commit -m "$(cat <<'EOF'
feat(profile): apply Local items and SettingsHooks

Profile apply now enables local items (agents, commands, etc.) based
on patterns and merges settingsHooks into settings.json.
EOF
)"
```

---

## Task 12: Profile Save - Capture Local Items

**Files:**
- Modify: `internal/profile/snapshot.go`
- Test: `internal/profile/snapshot_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/profile/snapshot_test.go

func TestSnapshotCapturesLocal(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")

	// Create library and enabled.json
	libraryDir := filepath.Join(claudeDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)

	enabledJSON := `{"agents": {"gsd-planner.md": true}}`
	os.WriteFile(filepath.Join(claudeDir, "enabled.json"), []byte(enabledJSON), 0644)

	// Create minimal settings.json
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{"enabledPlugins": {}}`), 0644)

	// Snapshot
	p, err := Snapshot("test", claudeDir, "")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	// Verify Local was captured
	if p.Local == nil {
		t.Fatal("Local is nil")
	}
	if len(p.Local.Agents) != 1 || p.Local.Agents[0] != "gsd-planner.md" {
		t.Errorf("Local.Agents = %v, want [gsd-planner.md]", p.Local.Agents)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/profile/... -v -run TestSnapshotCapturesLocal`
Expected: FAIL - Snapshot doesn't capture Local

**Step 3: Write minimal implementation**

Modify Snapshot in `internal/profile/snapshot.go`:

```go
// Add to Snapshot function

// Capture local items from enabled.json
localMgr := local.NewManager(claudeDir)
config, err := localMgr.LoadConfig()
if err == nil && len(config) > 0 {
	p.Local = &LocalSettings{}

	for category, items := range config {
		var enabledItems []string
		for item, enabled := range items {
			if enabled {
				enabledItems = append(enabledItems, item)
			}
		}
		sort.Strings(enabledItems)

		switch category {
		case local.CategoryAgents:
			p.Local.Agents = enabledItems
		case local.CategoryCommands:
			p.Local.Commands = enabledItems
		case local.CategorySkills:
			p.Local.Skills = enabledItems
		case local.CategoryHooks:
			p.Local.Hooks = enabledItems
		case local.CategoryRules:
			p.Local.Rules = enabledItems
		case local.CategoryOutputStyles:
			p.Local.OutputStyles = enabledItems
		}
	}

	// Remove Local if empty
	if len(p.Local.Agents) == 0 && len(p.Local.Commands) == 0 &&
		len(p.Local.Skills) == 0 && len(p.Local.Hooks) == 0 &&
		len(p.Local.Rules) == 0 && len(p.Local.OutputStyles) == 0 {
		p.Local = nil
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./internal/profile/... -v -run TestSnapshotCapturesLocal`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add internal/profile/snapshot.go internal/profile/snapshot_test.go
git commit -m "$(cat <<'EOF'
feat(profile): capture Local items in Snapshot

Profile save now captures enabled local items from enabled.json,
preserving the user's local configuration in the profile.
EOF
)"
```

---

## Task 13: Integration Tests

**Files:**
- Create: `test/integration/local/local_suite_test.go`

**Step 1: Write integration test**

```go
// test/integration/local/local_suite_test.go
// ABOUTME: Integration tests for local item management
// ABOUTME: Tests end-to-end enable/disable/list/view flows
package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/v4/internal/local"
)

func TestLocalIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create full library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	commandsDir := filepath.Join(libraryDir, "commands", "gsd")
	hooksDir := filepath.Join(libraryDir, "hooks")

	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(commandsDir, 0755)
	os.MkdirAll(hooksDir, 0755)

	// Create test items
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(commandsDir, "new-project.md"), []byte("# New Project"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gsd-check-update.js"), []byte("// JS"), 0644)

	manager := local.NewManager(tmpDir)

	// Test: List items
	agents, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems(agents) error = %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("ListItems(agents) = %d items, want 2", len(agents))
	}

	// Test: Enable with wildcard
	enabled, notFound, err := manager.Enable("agents", []string{"gsd-*"})
	if err != nil {
		t.Fatalf("Enable(agents, gsd-*) error = %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("Enable() enabled %d items, want 2", len(enabled))
	}
	if len(notFound) != 0 {
		t.Errorf("Enable() notFound = %v, want []", notFound)
	}

	// Verify symlinks
	for _, item := range enabled {
		symlinkPath := filepath.Join(tmpDir, "agents", item)
		if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
			t.Errorf("Symlink not created for %s", item)
		}
	}

	// Test: Disable
	disabled, _, err := manager.Disable("agents", []string{"gsd-planner"})
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if len(disabled) != 1 {
		t.Errorf("Disable() disabled %d items, want 1", len(disabled))
	}

	// Verify symlink removed
	if _, err := os.Lstat(filepath.Join(tmpDir, "agents", "gsd-planner.md")); !os.IsNotExist(err) {
		t.Error("Symlink was not removed")
	}

	// Test: View
	content, err := manager.View("agents", "gsd-executor")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "# Executor" {
		t.Errorf("View() content = %q, want '# Executor'", content)
	}

	// Test: Sync
	err = manager.Sync()
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
}
```

**Step 2: Run integration tests**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./test/integration/local/... -v`
Expected: PASS

**Step 3: Commit**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git add test/integration/local/local_suite_test.go
git commit -m "$(cat <<'EOF'
test(local): add integration tests

End-to-end tests for local item management including enable with
wildcards, disable, view, and sync operations.
EOF
)"
```

---

## Task 14: Final - Run Full Test Suite

**Files:** None (verification only)

**Step 1: Run all tests**

Run: `cd /Users/markalston/code/claudeup/.worktrees/local-management && go test ./... -v`
Expected: All tests pass

**Step 2: Build and manual verification**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
go build -o bin/claudeup ./cmd/claudeup

# Test commands
./bin/claudeup local --help
./bin/claudeup local list
```

**Step 3: Final commit if any fixes needed**

```bash
cd /Users/markalston/code/claudeup/.worktrees/local-management
git status
# If clean, we're done
# If changes needed, fix and commit
```

---

## Summary

This plan implements the local management feature in 14 tasks:

1. **Tasks 1-7:** Core `internal/local` package (types, config, list, resolve, wildcard, symlinks, view)
2. **Task 8:** CLI commands (`claudeup local list/enable/disable/view/sync`)
3. **Tasks 9-10:** Profile schema extension (Local, SettingsHooks) and settings.json hook merging
4. **Tasks 11-12:** Profile apply/save integration
5. **Tasks 13-14:** Integration tests and verification

Each task follows TDD: write failing test, implement, verify, commit.
