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

	// Should install ALL profile plugins (B and C) to ensure proper registration
	if len(diff.PluginsToInstall) != 2 {
		t.Errorf("Expected to install 2 plugins (all profile plugins), got: %v", diff.PluginsToInstall)
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

func TestComputeDiffEmptyProfileRemovesEverything(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: has plugins and MCP servers
	currentPlugins := map[string]interface{}{
		"version": 1,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": map[string]interface{}{"version": "1.0"},
			"plugin-b@marketplace": map[string]interface{}{"version": "1.0"},
		},
	}
	claudeJSON := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"server-a": map[string]interface{}{"command": "cmd-a"},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), claudeJSON)

	// Empty profile - should remove everything
	profile := &Profile{Name: "empty"}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"))
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	if len(diff.PluginsToRemove) != 2 {
		t.Errorf("Expected 2 plugins to remove, got %d: %v", len(diff.PluginsToRemove), diff.PluginsToRemove)
	}
	if len(diff.MCPToRemove) != 1 {
		t.Errorf("Expected 1 MCP server to remove, got %d: %v", len(diff.MCPToRemove), diff.MCPToRemove)
	}
	if len(diff.PluginsToInstall) != 0 {
		t.Errorf("Expected no plugins to install, got: %v", diff.PluginsToInstall)
	}
	if len(diff.MCPToInstall) != 0 {
		t.Errorf("Expected no MCP servers to install, got: %v", diff.MCPToInstall)
	}
}

func TestComputeDiffFreshInstall(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	// Don't create any files - simulates fresh install

	// Profile with content
	profile := &Profile{
		Name:    "full",
		Plugins: []string{"plugin-a@marketplace", "plugin-b@marketplace"},
		MCPServers: []MCPServer{
			{Name: "server-a", Command: "cmd-a"},
		},
		Marketplaces: []Marketplace{
			{Repo: "org/marketplace"},
		},
	}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"))
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	// Should install everything, remove nothing
	if len(diff.PluginsToInstall) != 2 {
		t.Errorf("Expected 2 plugins to install, got %d: %v", len(diff.PluginsToInstall), diff.PluginsToInstall)
	}
	if len(diff.MCPToInstall) != 1 {
		t.Errorf("Expected 1 MCP server to install, got %d: %v", len(diff.MCPToInstall), diff.MCPToInstall)
	}
	if len(diff.MarketplacesToAdd) != 1 {
		t.Errorf("Expected 1 marketplace to add, got %d: %v", len(diff.MarketplacesToAdd), diff.MarketplacesToAdd)
	}
	if len(diff.PluginsToRemove) != 0 {
		t.Errorf("Expected no plugins to remove, got: %v", diff.PluginsToRemove)
	}
	if len(diff.MCPToRemove) != 0 {
		t.Errorf("Expected no MCP servers to remove, got: %v", diff.MCPToRemove)
	}
}

func TestComputeDiffIdenticalStates(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state matches profile exactly
	currentPlugins := map[string]interface{}{
		"version": 1,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": map[string]interface{}{"version": "1.0"},
		},
	}
	claudeJSON := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"server-a": map[string]interface{}{"command": "cmd-a"},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), claudeJSON)

	// Profile identical to current state
	profile := &Profile{
		Name:    "identical",
		Plugins: []string{"plugin-a@marketplace"},
		MCPServers: []MCPServer{
			{Name: "server-a", Command: "cmd-a"},
		},
	}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"))
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	// Nothing should be removed
	if len(diff.PluginsToRemove) != 0 {
		t.Errorf("Expected no plugins to remove, got: %v", diff.PluginsToRemove)
	}
	// All profile plugins should be in install list (to ensure proper registration)
	if len(diff.PluginsToInstall) != 1 {
		t.Errorf("Expected 1 plugin to install (all profile plugins), got: %v", diff.PluginsToInstall)
	}
	if len(diff.MCPToRemove) != 0 {
		t.Errorf("Expected no MCP servers to remove, got: %v", diff.MCPToRemove)
	}
	if len(diff.MCPToInstall) != 0 {
		t.Errorf("Expected no MCP servers to install, got: %v", diff.MCPToInstall)
	}
}

func TestComputeDiffMarketplacesOnlyAdd(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state has marketplace A
	marketplaces := map[string]interface{}{
		"marketplace-a": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "org/marketplace-a",
			},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 1, "plugins": map[string]interface{}{}})
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile only has marketplace B (not A) - but marketplaces are additive
	profile := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/marketplace-b"},
		},
	}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"))
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	// Should add B, not remove A (marketplaces are additive only)
	if len(diff.MarketplacesToAdd) != 1 || diff.MarketplacesToAdd[0].Repo != "org/marketplace-b" {
		t.Errorf("Expected to add marketplace-b, got: %v", diff.MarketplacesToAdd)
	}
	// Verify no mechanism exists to remove marketplaces (by design)
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
