// ABOUTME: Tests for profile apply logic
// ABOUTME: Validates diff computation and arg building
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeDiffPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: plugins A and B installed
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"plugin-b@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
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
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}})
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
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"plugin-b@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
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
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}})
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

func TestMarketplaceNameFromRepo(t *testing.T) {
	tests := []struct {
		repo     string
		expected string
	}{
		{"wshobson/agents", "wshobson-agents"},
		{"anthropics/claude-code", "anthropics-claude-code"},
		{"", ""},
		{"simple", "simple"},
	}

	for _, tc := range tests {
		result := marketplaceNameFromRepo(tc.repo)
		if result != tc.expected {
			t.Errorf("marketplaceNameFromRepo(%q) = %q, want %q", tc.repo, result, tc.expected)
		}
	}
}

func TestIsFirstRun(t *testing.T) {
	tests := []struct {
		name            string
		profileMarkets  []Marketplace
		currentPlugins  []string
		expectedResult  bool
	}{
		{
			name:            "no plugins - first run",
			profileMarkets:  []Marketplace{{Repo: "wshobson/agents"}},
			currentPlugins:  []string{},
			expectedResult:  true,
		},
		{
			name:            "plugins from other marketplace - first run",
			profileMarkets:  []Marketplace{{Repo: "wshobson/agents"}},
			currentPlugins:  []string{"superpowers@superpowers-marketplace", "frontend@claude-code-plugins"},
			expectedResult:  true,
		},
		{
			name:            "plugins from same marketplace - not first run",
			profileMarkets:  []Marketplace{{Repo: "wshobson/agents"}},
			currentPlugins:  []string{"debugging-toolkit@wshobson-agents", "superpowers@superpowers-marketplace"},
			expectedResult:  false,
		},
		{
			name:            "multiple marketplaces - any match stops first run",
			profileMarkets:  []Marketplace{{Repo: "wshobson/agents"}, {Repo: "other/marketplace"}},
			currentPlugins:  []string{"something@other-marketplace"},
			expectedResult:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			claudeDir := filepath.Join(tmpDir, ".claude")
			pluginsDir := filepath.Join(claudeDir, "plugins")
			os.MkdirAll(pluginsDir, 0755)

			// Build current plugins JSON
			pluginsMap := make(map[string]interface{})
			for _, p := range tc.currentPlugins {
				pluginsMap[p] = []map[string]interface{}{{"scope": "user", "version": "1.0"}}
			}
			currentPlugins := map[string]interface{}{
				"version": 2,
				"plugins": pluginsMap,
			}
			writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
			writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
			writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

			profile := &Profile{
				Name:         "test",
				Marketplaces: tc.profileMarkets,
			}

			result := isFirstRun(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"))
			if result != tc.expectedResult {
				t.Errorf("isFirstRun() = %v, want %v", result, tc.expectedResult)
			}
		})
	}
}

func TestShouldRunHook(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Empty current state
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}})
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	tests := []struct {
		name     string
		profile  *Profile
		opts     HookOptions
		expected bool
	}{
		{
			name:     "no hook defined",
			profile:  &Profile{Name: "test"},
			opts:     HookOptions{},
			expected: false,
		},
		{
			name: "no-interactive skips hook",
			profile: &Profile{
				Name:      "test",
				PostApply: &PostApplyHook{Script: "setup.sh", Condition: "always"},
			},
			opts:     HookOptions{NoInteractive: true},
			expected: false,
		},
		{
			name: "always condition runs hook",
			profile: &Profile{
				Name:      "test",
				PostApply: &PostApplyHook{Script: "setup.sh", Condition: "always"},
			},
			opts:     HookOptions{},
			expected: true,
		},
		{
			name: "empty condition defaults to always",
			profile: &Profile{
				Name:      "test",
				PostApply: &PostApplyHook{Script: "setup.sh"},
			},
			opts:     HookOptions{},
			expected: true,
		},
		{
			name: "force-setup overrides first-run check",
			profile: &Profile{
				Name:         "test",
				Marketplaces: []Marketplace{{Repo: "wshobson/agents"}},
				PostApply:    &PostApplyHook{Script: "setup.sh", Condition: "first-run"},
			},
			opts:     HookOptions{ForceSetup: true},
			expected: true,
		},
		{
			name: "first-run condition with fresh install",
			profile: &Profile{
				Name:         "test",
				Marketplaces: []Marketplace{{Repo: "wshobson/agents"}},
				PostApply:    &PostApplyHook{Script: "setup.sh", Condition: "first-run"},
			},
			opts:     HookOptions{},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ShouldRunHook(tc.profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), tc.opts)
			if result != tc.expected {
				t.Errorf("ShouldRunHook() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestRunHookWithCommand(t *testing.T) {
	profile := &Profile{
		Name: "test",
		PostApply: &PostApplyHook{
			Command: "echo 'hook ran'",
		},
	}

	// Should not error for a simple echo command
	err := RunHook(profile, HookOptions{})
	if err != nil {
		t.Errorf("RunHook() unexpected error: %v", err)
	}
}

func TestRunHookWithScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test script
	scriptPath := filepath.Join(tmpDir, "test-setup.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'script ran'\n"), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	profile := &Profile{
		Name: "test",
		PostApply: &PostApplyHook{
			Script: "test-setup.sh",
		},
	}

	err := RunHook(profile, HookOptions{ScriptDir: tmpDir})
	if err != nil {
		t.Errorf("RunHook() unexpected error: %v", err)
	}
}

func TestRunHookNoHook(t *testing.T) {
	profile := &Profile{Name: "test"}

	// Should not error when no hook is defined
	err := RunHook(profile, HookOptions{})
	if err != nil {
		t.Errorf("RunHook() unexpected error: %v", err)
	}
}

func TestRunHookReturnsErrorOnFailure(t *testing.T) {
	profile := &Profile{
		Name: "test",
		PostApply: &PostApplyHook{
			Command: "exit 1", // Command that fails
		},
	}

	err := RunHook(profile, HookOptions{})
	if err == nil {
		t.Error("RunHook() expected error for failing command, got nil")
	}
}

func TestRunHookScriptFailureReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that fails
	scriptPath := filepath.Join(tmpDir, "failing-setup.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nexit 42\n"), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	profile := &Profile{
		Name: "test",
		PostApply: &PostApplyHook{
			Script: "failing-setup.sh",
		},
	}

	err := RunHook(profile, HookOptions{ScriptDir: tmpDir})
	if err == nil {
		t.Error("RunHook() expected error for failing script, got nil")
	}
}

func TestRunHookScriptTakesPrecedenceOverCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test script that creates a marker file
	scriptPath := filepath.Join(tmpDir, "test-setup.sh")
	markerPath := filepath.Join(tmpDir, "script-ran")
	scriptContent := fmt.Sprintf("#!/bin/bash\ntouch %s\n", markerPath)
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Profile has both Script and Command - Script should take precedence
	profile := &Profile{
		Name: "test",
		PostApply: &PostApplyHook{
			Script:  "test-setup.sh",
			Command: "echo 'command ran'", // This should NOT run
		},
	}

	err := RunHook(profile, HookOptions{ScriptDir: tmpDir})
	if err != nil {
		t.Errorf("RunHook() unexpected error: %v", err)
	}

	// Verify script ran (marker file exists)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Errorf("Script did not run - marker file not created")
	}
}

func TestRunHookScriptNotFound(t *testing.T) {
	profile := &Profile{
		Name: "test",
		PostApply: &PostApplyHook{
			Script: "nonexistent-script.sh",
		},
	}

	err := RunHook(profile, HookOptions{ScriptDir: t.TempDir()})
	if err == nil {
		t.Error("RunHook() expected error for missing script, got nil")
	}
	if !strings.Contains(err.Error(), "hook script not found") {
		t.Errorf("Expected 'hook script not found' error, got: %v", err)
	}
}

func TestIsEmbeddedProfile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"hobson", true},       // Known embedded profile
		{"frontend", true},     // Known embedded profile
		{"default", true},      // Known embedded profile
		{"nonexistent", false}, // Not embedded
		{"", false},            // Empty name
	}

	for _, tc := range tests {
		result := IsEmbeddedProfile(tc.name)
		if result != tc.expected {
			t.Errorf("IsEmbeddedProfile(%q) = %v, want %v", tc.name, result, tc.expected)
		}
	}
}
