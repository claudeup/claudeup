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
