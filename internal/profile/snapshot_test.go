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
	p, err := Snapshot("test-snapshot", claudeDir, claudeJSONPath)
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
	p, err := Snapshot("empty", claudeDir, claudeJSONPath)
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
	p, err := Snapshot("test-git-url", claudeDir, claudeJSONPath)
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
	p, err := Snapshot("test-invalid-filter", claudeDir, claudeJSONPath)
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
	p, err := Snapshot("test-local-items", claudeDir, claudeJSONPath)
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
	p, err := Snapshot("test-no-local", claudeDir, claudeJSONPath)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// LocalItems should be nil when no enabled.json exists
	if p.LocalItems != nil {
		t.Errorf("Expected LocalItems to be nil when no enabled.json, got %+v", p.LocalItems)
	}
}
