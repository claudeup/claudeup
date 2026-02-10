// ABOUTME: Tests for snapshot functionality
// ABOUTME: Validates creating a profile from current Claude state
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotFromState(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock installed_plugins.json
	pluginsData := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"superpowers@superpowers-marketplace": []map[string]interface{}{{
				"scope":       "user",
				"version":     "1.0.0",
				"installPath": "/path/to/plugin",
			}},
			"frontend-design@claude-code-plugins": []map[string]interface{}{{
				"scope":       "user",
				"version":     "2.0.0",
				"installPath": "/path/to/plugin2",
			}},
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData)

	// Create mock settings.json with enabled plugins
	settingsData := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"superpowers@superpowers-marketplace": true,
			"frontend-design@claude-code-plugins": true,
		},
	}
	writeJSON(t, filepath.Join(claudeDir, "settings.json"), settingsData)

	// Create mock known_marketplaces.json
	marketplacesData := map[string]interface{}{
		"superpowers-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/superpowers-marketplace",
			},
		},
		"claude-code-plugins": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/claude-code-plugins",
			},
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplacesData)

	// Create mock ~/.claude.json with MCP servers
	claudeJSON := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"context7": map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@upstash/context7-mcp"},
				"env":     map[string]string{},
			},
		},
	}
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, claudeJSON)

	// Create snapshot
	p, err := Snapshot("test-snapshot", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Verify profile
	if p.Name != "test-snapshot" {
		t.Errorf("Name mismatch: got %q", p.Name)
	}

	if len(p.Plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(p.Plugins))
	}

	if len(p.Marketplaces) != 2 {
		t.Errorf("Expected 2 marketplaces, got %d", len(p.Marketplaces))
	}

	if len(p.MCPServers) != 1 {
		t.Errorf("Expected 1 MCP server, got %d", len(p.MCPServers))
	}

	// Verify MCP server details
	if len(p.MCPServers) > 0 {
		mcp := p.MCPServers[0]
		if mcp.Name != "context7" {
			t.Errorf("MCP name mismatch: got %q", mcp.Name)
		}
		if mcp.Command != "npx" {
			t.Errorf("MCP command mismatch: got %q", mcp.Command)
		}
	}
}

func TestSnapshotEmptyState(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")

	// Don't create any files - test with empty state
	p, err := Snapshot("empty", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed on empty state: %v", err)
	}

	if p.Name != "empty" {
		t.Errorf("Name mismatch: got %q", p.Name)
	}

	if len(p.Plugins) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(p.Plugins))
	}
}

func TestSnapshotWithGitSourceMarketplace(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock installed_plugins.json (empty)
	pluginsData := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{},
	}
	writeJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData)

	// Create mock known_marketplaces.json with both github and git sources
	marketplacesData := map[string]interface{}{
		"claude-code-plugins": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/claude-code",
			},
		},
		"every-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "git",
				"url":    "https://github.com/EveryInc/compound-engineering-plugin.git",
			},
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplacesData)

	// Create mock ~/.claude.json (empty MCP servers)
	claudeJSON := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, claudeJSON)

	// Create snapshot
	p, err := Snapshot("test-git-url", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Verify profile
	if len(p.Marketplaces) != 2 {
		t.Fatalf("Expected 2 marketplaces, got %d", len(p.Marketplaces))
	}

	// Check that both marketplaces have proper display names
	foundGithub := false
	foundGit := false
	for _, m := range p.Marketplaces {
		displayName := m.DisplayName()
		if displayName == "" {
			t.Errorf("Marketplace has empty display name: source=%s repo=%s url=%s", m.Source, m.Repo, m.URL)
		}
		if m.Source == "github" && m.Repo == "anthropics/claude-code" {
			foundGithub = true
		}
		if m.Source == "git" && m.URL == "https://github.com/EveryInc/compound-engineering-plugin.git" {
			foundGit = true
		}
	}

	if !foundGithub {
		t.Error("GitHub marketplace not found in snapshot")
	}
	if !foundGit {
		t.Error("Git URL marketplace not found in snapshot")
	}
}

func TestMarketplaceDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		market   Marketplace
		expected string
	}{
		{
			name:     "github source uses repo",
			market:   Marketplace{Source: "github", Repo: "anthropics/claude-code"},
			expected: "anthropics/claude-code",
		},
		{
			name:     "git source uses url",
			market:   Marketplace{Source: "git", URL: "https://github.com/example/repo.git"},
			expected: "https://github.com/example/repo.git",
		},
		{
			name:     "both set prefers repo",
			market:   Marketplace{Source: "github", Repo: "owner/repo", URL: "https://example.com"},
			expected: "owner/repo",
		},
		{
			name:     "empty returns empty",
			market:   Marketplace{Source: "git"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.market.DisplayName()
			if got != tt.expected {
				t.Errorf("DisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSnapshotFiltersInvalidMarketplaces(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock installed_plugins.json (empty)
	pluginsData := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{},
	}
	writeJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData)

	// Create mock known_marketplaces.json with one valid and one invalid marketplace
	// This reproduces the bug where a "git" source has no url field
	marketplacesData := map[string]interface{}{
		"valid-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/claude-code",
			},
		},
		"invalid-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "git",
				// Missing both repo and url - this is invalid data
			},
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplacesData)

	// Create mock ~/.claude.json (empty MCP servers)
	claudeJSON := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, claudeJSON)

	// Create snapshot
	p, err := Snapshot("test-invalid-filter", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Should only have 1 marketplace (the valid one)
	if len(p.Marketplaces) != 1 {
		t.Errorf("Expected 1 marketplace (invalid one should be filtered), got %d", len(p.Marketplaces))
		for i, m := range p.Marketplaces {
			t.Logf("  marketplace[%d]: source=%s repo=%q url=%q", i, m.Source, m.Repo, m.URL)
		}
	}

	// Verify the remaining marketplace is the valid one
	if len(p.Marketplaces) > 0 {
		m := p.Marketplaces[0]
		if m.Repo != "anthropics/claude-code" {
			t.Errorf("Expected valid marketplace with repo 'anthropics/claude-code', got repo=%q url=%q", m.Repo, m.URL)
		}
	}
}

func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotCapturesLocalItems(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock enabled.json with enabled local items
	enabledData := map[string]map[string]bool{
		"agents": {
			"gsd-planner":  true,
			"gsd-executor": true,
			"other-agent":  false, // disabled, should not be captured
		},
		"commands": {
			"gsd/start": true,
			"gsd/stop":  true,
		},
		"hooks": {
			"gsd-check-update.js": true,
		},
	}
	writeJSON(t, filepath.Join(claudeDir, "enabled.json"), enabledData)

	// Create corresponding active directory entries (symlinks/dirs)
	for _, dir := range []string{
		filepath.Join(claudeDir, "agents", "gsd-planner"),
		filepath.Join(claudeDir, "agents", "gsd-executor"),
		filepath.Join(claudeDir, "commands", "gsd"),
		filepath.Join(claudeDir, "hooks"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Create files for commands and hooks
	for _, f := range []string{
		filepath.Join(claudeDir, "commands", "gsd", "start"),
		filepath.Join(claudeDir, "commands", "gsd", "stop"),
		filepath.Join(claudeDir, "hooks", "gsd-check-update.js"),
	} {
		if err := os.WriteFile(f, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create minimal settings.json
	settingsData := map[string]interface{}{
		"enabledPlugins": map[string]bool{},
	}
	writeJSON(t, filepath.Join(claudeDir, "settings.json"), settingsData)

	// Create mock known_marketplaces.json (empty)
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})

	// Create mock ~/.claude.json
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	})

	// Create snapshot
	p, err := Snapshot("test-local-items", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Verify LocalItems was captured
	if p.LocalItems == nil {
		t.Fatal("Expected LocalItems to be captured, got nil")
	}

	// Check agents - should only include enabled ones, sorted
	expectedAgents := []string{"gsd-executor", "gsd-planner"}
	if len(p.LocalItems.Agents) != len(expectedAgents) {
		t.Errorf("Expected %d agents, got %d: %v", len(expectedAgents), len(p.LocalItems.Agents), p.LocalItems.Agents)
	} else {
		for i, expected := range expectedAgents {
			if p.LocalItems.Agents[i] != expected {
				t.Errorf("Agent[%d]: expected %q, got %q", i, expected, p.LocalItems.Agents[i])
			}
		}
	}

	// Check commands - should be sorted
	expectedCommands := []string{"gsd/start", "gsd/stop"}
	if len(p.LocalItems.Commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d: %v", len(expectedCommands), len(p.LocalItems.Commands), p.LocalItems.Commands)
	} else {
		for i, expected := range expectedCommands {
			if p.LocalItems.Commands[i] != expected {
				t.Errorf("Command[%d]: expected %q, got %q", i, expected, p.LocalItems.Commands[i])
			}
		}
	}

	// Check hooks
	expectedHooks := []string{"gsd-check-update.js"}
	if len(p.LocalItems.Hooks) != len(expectedHooks) {
		t.Errorf("Expected %d hooks, got %d: %v", len(expectedHooks), len(p.LocalItems.Hooks), p.LocalItems.Hooks)
	}
}

func TestSnapshotExcludesStaleLocalItems(t *testing.T) {
	// enabled.json may have stale entries (marked true but no corresponding
	// symlink in the active directory). These should be excluded from snapshots.
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Config has both a real entry and a stale entry
	enabledData := map[string]map[string]bool{
		"agents": {
			"real-agent":  true, // has active directory entry
			"stale-agent": true, // NO active directory entry (stale)
		},
	}
	writeJSON(t, filepath.Join(claudeDir, "enabled.json"), enabledData)

	// Only create active entry for real-agent
	if err := os.MkdirAll(filepath.Join(claudeDir, "agents", "real-agent"), 0755); err != nil {
		t.Fatal(err)
	}

	settingsData := map[string]interface{}{
		"enabledPlugins": map[string]bool{},
	}
	writeJSON(t, filepath.Join(claudeDir, "settings.json"), settingsData)
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})

	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	})

	p, err := Snapshot("test-stale", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Should only include real-agent, not stale-agent
	if p.LocalItems == nil {
		t.Fatal("Expected LocalItems to be non-nil")
	}
	if len(p.LocalItems.Agents) != 1 {
		t.Errorf("Expected 1 agent (real-agent only), got %d: %v", len(p.LocalItems.Agents), p.LocalItems.Agents)
	}
	if len(p.LocalItems.Agents) > 0 && p.LocalItems.Agents[0] != "real-agent" {
		t.Errorf("Expected real-agent, got %q", p.LocalItems.Agents[0])
	}
}

func TestSnapshotAllScopesNoPluginsReturnsNoMarketplaces(t *testing.T) {
	// When no plugins are enabled, SnapshotAllScopes should return no
	// marketplaces (empty plugins means filter strictly, not include all).
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// No plugins enabled
	writeJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]interface{}{
		"enabledPlugins": map[string]bool{},
	})

	// Registry has marketplaces, but no plugins reference them
	registry := map[string]interface{}{
		"some-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "someone/repo",
			},
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), registry)

	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	})

	p, err := SnapshotAllScopes("no-plugins", claudeDir, claudeJSONPath, "", claudeDir)
	if err != nil {
		t.Fatalf("SnapshotAllScopes failed: %v", err)
	}

	// No plugins means no marketplaces should be included
	if len(p.Marketplaces) != 0 {
		t.Errorf("Expected 0 marketplaces when no plugins exist, got %d: %v", len(p.Marketplaces), p.Marketplaces)
	}
}

func TestSnapshotOnlyCapturesMarketplacesUsedByPlugins(t *testing.T) {
	// SnapshotAllScopes (used by profile save) filters marketplaces to only
	// those referenced by enabled plugins. Marketplaces installed by other
	// tools should be excluded.
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Plugins reference marketplaces via the @marketplace suffix
	settingsData := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"superpowers@superpowers-marketplace": true,
			"claude-hud@claude-hud":               true,
		},
	}
	writeJSON(t, filepath.Join(claudeDir, "settings.json"), settingsData)

	// Registry has 3 marketplaces, but only 2 are used by plugins
	registry := map[string]interface{}{
		"superpowers-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "obra/superpowers-marketplace",
			},
		},
		"claude-hud": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "jarrodwatts/claude-hud",
			},
		},
		"unused-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "someone/unused-repo",
			},
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), registry)

	// Create mock ~/.claude.json
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	})

	// SnapshotAllScopes filters marketplaces by plugin references
	p, err := SnapshotAllScopes("test-filter", claudeDir, claudeJSONPath, "", claudeDir)
	if err != nil {
		t.Fatalf("SnapshotAllScopes failed: %v", err)
	}

	// Should only have the 2 marketplaces referenced by plugins
	if len(p.Marketplaces) != 2 {
		t.Errorf("Expected 2 marketplaces (used by plugins), got %d: %v", len(p.Marketplaces), p.Marketplaces)
	}

	// Verify the unused marketplace was excluded
	for _, m := range p.Marketplaces {
		if m.Repo == "someone/unused-repo" {
			t.Errorf("Marketplace %q should not be in snapshot (no plugins use it)", m.Repo)
		}
	}
}

func TestSnapshotReturnsAllMarketplacesForDiff(t *testing.T) {
	// Snapshot (used by ComputeDiff) returns all marketplaces without
	// filtering, so removals can be detected.
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Only one plugin enabled
	settingsData := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"superpowers@superpowers-marketplace": true,
		},
	}
	writeJSON(t, filepath.Join(claudeDir, "settings.json"), settingsData)

	// Registry has 2 marketplaces, but only 1 is used by plugins
	registry := map[string]interface{}{
		"superpowers-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "obra/superpowers-marketplace",
			},
		},
		"unused-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "someone/unused-repo",
			},
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), registry)

	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	})

	p, err := Snapshot("test-all", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Snapshot returns ALL marketplaces (no filtering) for diff computation
	if len(p.Marketplaces) != 2 {
		t.Errorf("Expected 2 marketplaces (all, unfiltered), got %d: %v", len(p.Marketplaces), p.Marketplaces)
	}
}

func TestSnapshotNoLocalItemsWhenConfigMissing(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// No enabled.json file - should still work but LocalItems should be nil

	// Create minimal settings.json
	settingsData := map[string]interface{}{
		"enabledPlugins": map[string]bool{},
	}
	writeJSON(t, filepath.Join(claudeDir, "settings.json"), settingsData)

	// Create mock known_marketplaces.json (empty)
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})

	// Create mock ~/.claude.json
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	writeJSON(t, claudeJSONPath, map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	})

	// Create snapshot
	p, err := Snapshot("test-no-local", claudeDir, claudeJSONPath, claudeDir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// LocalItems should be nil when no enabled.json exists
	if p.LocalItems != nil {
		t.Errorf("Expected LocalItems to be nil when no enabled.json, got %+v", p.LocalItems)
	}
}
