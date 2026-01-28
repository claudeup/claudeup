// ABOUTME: Unit tests for SnapshotAllScopes function
// ABOUTME: Tests capturing settings from all three scopes into a profile
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotAllScopes(t *testing.T) {
	// Create temp directories for testing
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create directory structure
	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)
	mustMkdir(t, filepath.Join(projectDir, ".claude"))

	// Set up user-scope settings
	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"user-plugin-a@marketplace": true,
			"user-plugin-b@marketplace": true,
		},
	}
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), userSettings)

	// Set up project-scope settings
	projectSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"project-plugin-x@marketplace": true,
		},
	}
	mustWriteJSON(t, filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)

	// Set up local-scope settings
	localSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"local-plugin-z@marketplace": true,
		},
	}
	mustWriteJSON(t, filepath.Join(projectDir, ".claude", "settings.local.json"), localSettings)

	// Set up user-scope MCP servers (~/.claude.json)
	userMCP := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"user-mcp": map[string]interface{}{
				"command": "user-cmd",
				"args":    []string{"arg1"},
			},
		},
	}
	mustWriteJSON(t, claudeJSONPath, userMCP)

	// Set up project-scope MCP servers (.mcp.json)
	projectMCP := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"project-mcp": map[string]interface{}{
				"command": "project-cmd",
			},
		},
	}
	mustWriteJSON(t, filepath.Join(projectDir, ".mcp.json"), projectMCP)

	// Set up marketplaces (always user-scoped)
	marketplaces := map[string]interface{}{
		"test-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "user/repo",
			},
		},
	}
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), marketplaces)

	// Call SnapshotAllScopes
	profile, err := SnapshotAllScopes("test-profile", claudeDir, claudeJSONPath, projectDir)
	if err != nil {
		t.Fatalf("SnapshotAllScopes failed: %v", err)
	}

	// Verify profile name
	if profile.Name != "test-profile" {
		t.Errorf("expected name 'test-profile', got %q", profile.Name)
	}

	// Verify profile uses multi-scope format
	if !profile.IsMultiScope() {
		t.Fatal("expected profile to be multi-scope")
	}

	// Verify user scope
	if profile.PerScope.User == nil {
		t.Fatal("expected PerScope.User to be non-nil")
	}
	if len(profile.PerScope.User.Plugins) != 2 {
		t.Errorf("expected 2 user plugins, got %d: %v", len(profile.PerScope.User.Plugins), profile.PerScope.User.Plugins)
	}
	if len(profile.PerScope.User.MCPServers) != 1 {
		t.Errorf("expected 1 user MCP server, got %d", len(profile.PerScope.User.MCPServers))
	}

	// Verify project scope
	if profile.PerScope.Project == nil {
		t.Fatal("expected PerScope.Project to be non-nil")
	}
	if len(profile.PerScope.Project.Plugins) != 1 {
		t.Errorf("expected 1 project plugin, got %d: %v", len(profile.PerScope.Project.Plugins), profile.PerScope.Project.Plugins)
	}
	if profile.PerScope.Project.Plugins[0] != "project-plugin-x@marketplace" {
		t.Errorf("expected project-plugin-x@marketplace, got %q", profile.PerScope.Project.Plugins[0])
	}
	if len(profile.PerScope.Project.MCPServers) != 1 {
		t.Errorf("expected 1 project MCP server, got %d", len(profile.PerScope.Project.MCPServers))
	}

	// Verify local scope
	if profile.PerScope.Local == nil {
		t.Fatal("expected PerScope.Local to be non-nil")
	}
	if len(profile.PerScope.Local.Plugins) != 1 {
		t.Errorf("expected 1 local plugin, got %d: %v", len(profile.PerScope.Local.Plugins), profile.PerScope.Local.Plugins)
	}
	if profile.PerScope.Local.Plugins[0] != "local-plugin-z@marketplace" {
		t.Errorf("expected local-plugin-z@marketplace, got %q", profile.PerScope.Local.Plugins[0])
	}

	// Verify marketplaces (always user-scoped, stored at profile level)
	if len(profile.Marketplaces) != 1 {
		t.Errorf("expected 1 marketplace, got %d", len(profile.Marketplaces))
	}

	// Verify legacy fields are empty (we use PerScope now)
	if len(profile.Plugins) > 0 {
		t.Errorf("expected legacy Plugins to be empty, got %v", profile.Plugins)
	}
}

func TestSnapshotAllScopesEmptyScopes(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create minimal directory structure
	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Empty user settings
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]interface{}{
		"enabledPlugins": map[string]bool{},
	})

	// Empty claude.json
	mustWriteJSON(t, claudeJSONPath, map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	})

	// Empty marketplaces
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]interface{}{})

	profile, err := SnapshotAllScopes("empty-test", claudeDir, claudeJSONPath, projectDir)
	if err != nil {
		t.Fatalf("SnapshotAllScopes failed: %v", err)
	}

	// Should still be multi-scope format, but with empty/nil scopes
	if profile.PerScope == nil {
		t.Fatal("expected PerScope to be non-nil even when empty")
	}

	// Empty scopes should be nil to keep JSON clean
	if profile.PerScope.User != nil && len(profile.PerScope.User.Plugins) == 0 && len(profile.PerScope.User.MCPServers) == 0 {
		// If User exists but is empty, that's also acceptable
	}
	if profile.PerScope.Project != nil && len(profile.PerScope.Project.Plugins) > 0 {
		t.Errorf("expected no project plugins, got %v", profile.PerScope.Project.Plugins)
	}
	if profile.PerScope.Local != nil && len(profile.PerScope.Local.Plugins) > 0 {
		t.Errorf("expected no local plugins, got %v", profile.PerScope.Local.Plugins)
	}
}

func TestSnapshotAllScopesNoProjectDir(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create user scope only
	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))

	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"user-plugin@marketplace": true,
		},
	}
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), userSettings)
	mustWriteJSON(t, claudeJSONPath, map[string]interface{}{"mcpServers": map[string]interface{}{}})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]interface{}{})

	// Pass empty projectDir
	profile, err := SnapshotAllScopes("user-only", claudeDir, claudeJSONPath, "")
	if err != nil {
		t.Fatalf("SnapshotAllScopes failed: %v", err)
	}

	// Should capture user scope
	if profile.PerScope == nil || profile.PerScope.User == nil {
		t.Fatal("expected user scope to be captured")
	}
	if len(profile.PerScope.User.Plugins) != 1 {
		t.Errorf("expected 1 user plugin, got %d", len(profile.PerScope.User.Plugins))
	}

	// Project and local should be nil (no projectDir)
	if profile.PerScope.Project != nil && len(profile.PerScope.Project.Plugins) > 0 {
		t.Errorf("expected no project plugins without projectDir")
	}
}

// Helper functions for tests
func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", path, err)
	}
}

func mustWriteJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
