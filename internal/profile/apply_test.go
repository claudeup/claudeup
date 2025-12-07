// ABOUTME: Tests for profile apply logic
// ABOUTME: Validates diff computation and arg building
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeDiffPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: plugins A and B installed
	currentPlugins := map[string]interface{}{
		"version": 1,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": map[string]interface{}{"version": "1.0"},
			"plugin-b@marketplace": map[string]interface{}{"version": "1.0"},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile wants: plugins B and C
	profile := &Profile{
		Name:    "test",
		Plugins: []string{"plugin-b@marketplace", "plugin-c@marketplace"},
	}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"))
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	// Should remove A (in current, not in profile)
	if len(diff.PluginsToRemove) != 1 || diff.PluginsToRemove[0] != "plugin-a@marketplace" {
		t.Errorf("Expected to remove plugin-a, got: %v", diff.PluginsToRemove)
	}

	// Should install C (in profile, not in current)
	if len(diff.PluginsToInstall) != 1 || diff.PluginsToInstall[0] != "plugin-c@marketplace" {
		t.Errorf("Expected to install plugin-c, got: %v", diff.PluginsToInstall)
	}
}

func TestComputeDiffMCPServers(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: MCP servers A and B
	claudeJSON := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"server-a": map[string]interface{}{"command": "cmd-a"},
			"server-b": map[string]interface{}{"command": "cmd-b"},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 1, "plugins": map[string]interface{}{}})
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), claudeJSON)

	// Profile wants: servers B and C
	profile := &Profile{
		Name: "test",
		MCPServers: []MCPServer{
			{Name: "server-b", Command: "cmd-b"},
			{Name: "server-c", Command: "cmd-c"},
		},
	}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"))
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	// Should remove A
	if len(diff.MCPToRemove) != 1 || diff.MCPToRemove[0] != "server-a" {
		t.Errorf("Expected to remove server-a, got: %v", diff.MCPToRemove)
	}

	// Should install C
	if len(diff.MCPToInstall) != 1 || diff.MCPToInstall[0].Name != "server-c" {
		t.Errorf("Expected to install server-c, got: %v", diff.MCPToInstall)
	}
}

func TestBuildMCPAddArgs(t *testing.T) {
	mcp := MCPServer{
		Name:    "test-mcp",
		Command: "npx",
		Args:    []string{"-y", "some-package", "$API_KEY"},
		Scope:   "user",
	}

	resolvedSecrets := map[string]string{
		"API_KEY": "secret-value-123",
	}

	args := buildMCPAddArgs(mcp, resolvedSecrets)

	expected := []string{"mcp", "add", "test-mcp", "-s", "user", "--", "npx", "-y", "some-package", "secret-value-123"}

	if len(args) != len(expected) {
		t.Fatalf("Expected %d args, got %d: %v", len(expected), len(args), args)
	}

	for i, exp := range expected {
		if args[i] != exp {
			t.Errorf("Arg %d: expected %q, got %q", i, exp, args[i])
		}
	}
}

func TestBuildMCPAddArgsDefaultScope(t *testing.T) {
	mcp := MCPServer{
		Name:    "test-mcp",
		Command: "node",
		Args:    []string{"server.js"},
		// Scope not set - should default to "user"
	}

	args := buildMCPAddArgs(mcp, nil)

	// Check that -s user is present
	foundScope := false
	for i, arg := range args {
		if arg == "-s" && i+1 < len(args) && args[i+1] == "user" {
			foundScope = true
			break
		}
	}

	if !foundScope {
		t.Errorf("Expected default scope 'user' in args: %v", args)
	}
}

func writeTestJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		t.Fatal(err)
	}
}
