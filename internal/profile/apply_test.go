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

	// Current state: plugins A and B installed and enabled
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"plugin-b@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	currentSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin-a@marketplace": true,
			"plugin-b@marketplace": true,
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), currentSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile wants: plugins B and C
	profile := &Profile{
		Name:    "test",
		Plugins: []string{"plugin-b@marketplace", "plugin-c@marketplace"},
	}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir)
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

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir)
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

	// Current state: has plugins and MCP servers installed and enabled
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"plugin-b@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	currentSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin-a@marketplace": true,
			"plugin-b@marketplace": true,
		},
	}
	claudeJSON := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"server-a": map[string]interface{}{"command": "cmd-a"},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), currentSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), claudeJSON)

	// Empty profile - should remove everything
	profile := &Profile{Name: "empty"}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir)
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

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir)
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

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir)
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

	// Profile only has marketplace B (not A) - marketplaces are declarative
	profile := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/marketplace-b"},
		},
	}

	diff, err := ComputeDiff(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	// Should add B and remove A (declarative behavior)
	if len(diff.MarketplacesToAdd) != 1 || diff.MarketplacesToAdd[0].Repo != "org/marketplace-b" {
		t.Errorf("Expected to add marketplace-b, got: %v", diff.MarketplacesToAdd)
	}
	if len(diff.MarketplacesToRemove) != 1 || diff.MarketplacesToRemove[0].Repo != "org/marketplace-a" {
		t.Errorf("Expected to remove marketplace-a, got: %v", diff.MarketplacesToRemove)
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

func TestMarketplaceKey(t *testing.T) {
	tests := []struct {
		name        string
		marketplace Marketplace
		expected    string
	}{
		{"repo only", Marketplace{Repo: "user/repo"}, "user/repo"},
		{"url only", Marketplace{URL: "https://github.com/user/repo.git"}, "https://github.com/user/repo.git"},
		{"both prefers repo", Marketplace{Repo: "user/repo", URL: "https://example.com"}, "user/repo"},
		{"empty", Marketplace{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := marketplaceKey(tc.marketplace)
			if result != tc.expected {
				t.Errorf("marketplaceKey(%+v) = %q, want %q", tc.marketplace, result, tc.expected)
			}
		})
	}
}

func TestMarketplaceName(t *testing.T) {
	tests := []struct {
		name        string
		marketplace Marketplace
		expected    string
	}{
		{"repo path", Marketplace{Repo: "wshobson/agents"}, "wshobson-agents"},
		{"repo path with org", Marketplace{Repo: "anthropics/claude-code"}, "anthropics-claude-code"},
		{"empty", Marketplace{}, ""},
		{"simple repo", Marketplace{Repo: "simple"}, "simple"},
		{"https url", Marketplace{URL: "https://github.com/user/repo.git"}, "user-repo"},
		{"https url no .git", Marketplace{URL: "https://github.com/user/repo"}, "user-repo"},
		{"git url", Marketplace{URL: "git://example.com/org/project.git"}, "org-project"},
		{"self-hosted single path", Marketplace{URL: "https://git.example.com/repo"}, "repo"},
		{"self-hosted with org", Marketplace{URL: "https://gitlab.corp.com/team/project.git"}, "team-project"},
		{"deep path", Marketplace{URL: "https://github.com/org/group/subgroup/repo.git"}, "org-group-subgroup-repo"},
		{"url with port", Marketplace{URL: "https://git.example.com:8443/user/repo.git"}, "user-repo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := marketplaceName(tc.marketplace)
			if result != tc.expected {
				t.Errorf("marketplaceName(%+v) = %q, want %q", tc.marketplace, result, tc.expected)
			}
		})
	}
}

func TestIsFirstRun(t *testing.T) {
	tests := []struct {
		name           string
		profileMarkets []Marketplace
		currentPlugins []string
		expectedResult bool
	}{
		{
			name:           "no plugins - first run",
			profileMarkets: []Marketplace{{Repo: "wshobson/agents"}},
			currentPlugins: []string{},
			expectedResult: true,
		},
		{
			name:           "plugins from other marketplace - first run",
			profileMarkets: []Marketplace{{Repo: "wshobson/agents"}},
			currentPlugins: []string{"superpowers@superpowers-marketplace", "frontend@claude-code-plugins"},
			expectedResult: true,
		},
		{
			name:           "plugins from same marketplace - not first run",
			profileMarkets: []Marketplace{{Repo: "wshobson/agents"}},
			currentPlugins: []string{"debugging-toolkit@wshobson-agents", "superpowers@superpowers-marketplace"},
			expectedResult: false,
		},
		{
			name:           "multiple marketplaces - any match stops first run",
			profileMarkets: []Marketplace{{Repo: "wshobson/agents"}, {Repo: "other/marketplace"}},
			currentPlugins: []string{"something@other-marketplace"},
			expectedResult: false,
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
			enabledPlugins := make(map[string]bool)
			for _, p := range tc.currentPlugins {
				pluginsMap[p] = []map[string]interface{}{{"scope": "user", "version": "1.0"}}
				enabledPlugins[p] = true
			}
			currentPlugins := map[string]interface{}{
				"version": 2,
				"plugins": pluginsMap,
			}
			currentSettings := map[string]interface{}{
				"enabledPlugins": enabledPlugins,
			}
			writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
			writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), currentSettings)
			writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
			writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

			profile := &Profile{
				Name:         "test",
				Marketplaces: tc.profileMarkets,
			}

			result := isFirstRun(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir)
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
			result := ShouldRunHook(tc.profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, tc.opts)
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

// mockExecutor records commands for testing
type mockExecutor struct {
	commands [][]string
	failOn   map[string]bool // command prefixes that should fail
}

func (m *mockExecutor) Run(args ...string) error {
	m.commands = append(m.commands, args)
	// Check if this command should fail
	if len(args) > 0 && m.failOn != nil {
		key := strings.Join(args[:min(3, len(args))], " ")
		if m.failOn[key] {
			return fmt.Errorf("mock failure for: %s", key)
		}
	}
	return nil
}

func (m *mockExecutor) RunWithOutput(args ...string) (string, error) {
	m.commands = append(m.commands, args)
	// Check if this command should fail
	if len(args) > 0 && m.failOn != nil {
		key := strings.Join(args[:min(3, len(args))], " ")
		if m.failOn[key] {
			return "mock failure output", fmt.Errorf("mock failure for: %s", key)
		}
	}
	return "", nil
}

func TestResetRemovesPluginsFromMarketplace(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: plugins from wshobson-agents marketplace installed and enabled
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"debugging-toolkit@wshobson-agents": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"code-review-ai@wshobson-agents":    []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"superpowers@other-marketplace":     []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	currentSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"debugging-toolkit@wshobson-agents": true,
			"code-review-ai@wshobson-agents":    true,
			"superpowers@other-marketplace":     true,
		},
	}
	// Marketplace registry maps repo to name
	marketplaces := map[string]interface{}{
		"wshobson-agents": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "wshobson/agents",
			},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), currentSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile with wshobson/agents marketplace
	profile := &Profile{
		Name: "hobson",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "wshobson/agents"},
		},
	}

	executor := &mockExecutor{}
	result, err := ResetWithExecutor(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, executor)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should have removed 2 plugins from wshobson-agents
	if len(result.PluginsRemoved) != 2 {
		t.Errorf("Expected 2 plugins removed, got %d: %v", len(result.PluginsRemoved), result.PluginsRemoved)
	}

	// Should have removed the marketplace
	if len(result.MarketplacesRemoved) != 1 {
		t.Errorf("Expected 1 marketplace removed, got %d: %v", len(result.MarketplacesRemoved), result.MarketplacesRemoved)
	}

	// Verify correct commands were issued
	pluginUninstalls := 0
	marketplaceRemoves := 0
	for _, cmd := range executor.commands {
		if len(cmd) >= 2 && cmd[0] == "plugin" && cmd[1] == "uninstall" {
			pluginUninstalls++
		}
		if len(cmd) >= 3 && cmd[0] == "plugin" && cmd[1] == "marketplace" && cmd[2] == "remove" {
			marketplaceRemoves++
		}
	}
	if pluginUninstalls != 2 {
		t.Errorf("Expected 2 plugin uninstall commands, got %d", pluginUninstalls)
	}
	if marketplaceRemoves != 1 {
		t.Errorf("Expected 1 marketplace remove command, got %d", marketplaceRemoves)
	}
}

func TestResetRemovesMCPServers(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Empty current state
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}})
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile with MCP servers
	profile := &Profile{
		Name: "test",
		MCPServers: []MCPServer{
			{Name: "server-a", Command: "cmd-a"},
			{Name: "server-b", Command: "cmd-b"},
		},
	}

	executor := &mockExecutor{}
	result, err := ResetWithExecutor(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, executor)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should have removed 2 MCP servers
	if len(result.MCPServersRemoved) != 2 {
		t.Errorf("Expected 2 MCP servers removed, got %d: %v", len(result.MCPServersRemoved), result.MCPServersRemoved)
	}

	// Verify correct commands were issued
	mcpRemoves := 0
	for _, cmd := range executor.commands {
		if len(cmd) >= 2 && cmd[0] == "mcp" && cmd[1] == "remove" {
			mcpRemoves++
		}
	}
	if mcpRemoves != 2 {
		t.Errorf("Expected 2 mcp remove commands, got %d", mcpRemoves)
	}
}

func TestResetUsesMarketplaceNameNotRepo(t *testing.T) {
	// Bug: Reset was using repo (wshobson/agents) instead of marketplace name
	// (claude-code-workflows) when calling "claude plugin marketplace remove"
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: marketplace registered with name "claude-code-workflows"
	// but source repo is "wshobson/agents"
	marketplaces := map[string]interface{}{
		"claude-code-workflows": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "wshobson/agents",
			},
			"installLocation": "/Users/test/.claude/plugins/marketplaces/claude-code-workflows",
			"lastUpdated":     "2025-12-13T21:26:57.258Z",
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}})
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile uses repo "wshobson/agents"
	profile := &Profile{
		Name: "hobson",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "wshobson/agents"},
		},
	}

	executor := &mockExecutor{}
	_, err := ResetWithExecutor(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, executor)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Find the marketplace remove command
	var marketplaceRemoveArg string
	for _, cmd := range executor.commands {
		if len(cmd) >= 4 && cmd[0] == "plugin" && cmd[1] == "marketplace" && cmd[2] == "remove" {
			marketplaceRemoveArg = cmd[3]
			break
		}
	}

	// The command should use the marketplace NAME (claude-code-workflows),
	// NOT the repo (wshobson/agents)
	if marketplaceRemoveArg != "claude-code-workflows" {
		t.Errorf("Expected marketplace remove to use name 'claude-code-workflows', got: %q", marketplaceRemoveArg)
	}
}

func TestResetHandlesErrors(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state with a plugin installed and enabled
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"test-plugin@test-marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	currentSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"test-plugin@test-marketplace": true,
		},
	}
	// Marketplace registry maps repo to name
	marketplaces := map[string]interface{}{
		"test-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "test/marketplace",
			},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), currentSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	profile := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "test/marketplace"},
		},
	}

	// Executor that fails on plugin uninstall
	executor := &mockExecutor{
		failOn: map[string]bool{
			"plugin uninstall test-plugin@test-marketplace": true,
		},
	}
	result, err := ResetWithExecutor(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, executor)
	if err != nil {
		t.Fatalf("Reset should not return error, but collect errors: %v", err)
	}

	// Should have recorded the error
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error recorded, got %d", len(result.Errors))
	}
}

// mockExecutorWithOutput allows controlling both error and output
type mockExecutorWithOutput struct {
	commands [][]string
	outputs  map[string]string // command prefix -> output
}

func (m *mockExecutorWithOutput) Run(args ...string) error {
	m.commands = append(m.commands, args)
	return nil
}

func (m *mockExecutorWithOutput) RunWithOutput(args ...string) (string, error) {
	m.commands = append(m.commands, args)
	// Check if we have output/error for this command
	if len(args) > 0 && m.outputs != nil {
		key := strings.Join(args, " ")
		if output, ok := m.outputs[key]; ok {
			// If output contains error keywords, return error
			if strings.Contains(output, "not found") || strings.Contains(output, "Failed") {
				return output, fmt.Errorf("exit status 1")
			}
		}
	}
	return "", nil
}

func TestApplyHandlesPluginNotFoundError(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: has plugin-a installed and enabled (but not plugin-b)
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	currentSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin-a@marketplace": true,
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), currentSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile wants to remove both plugin-a and plugin-b (plugin-b doesn't exist)
	profile := &Profile{
		Name:    "test",
		Plugins: []string{}, // Empty = remove everything
	}

	// Executor that simulates "not found" error for plugin-b
	executor := &mockExecutorWithOutput{
		outputs: map[string]string{
			"plugin uninstall plugin-a@marketplace": "", // Success
		},
	}

	result, err := ApplyWithExecutor(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, nil, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have removed plugin-a successfully
	if len(result.PluginsRemoved) != 1 {
		t.Errorf("Expected 1 plugin removed, got %d: %v", len(result.PluginsRemoved), result.PluginsRemoved)
	}

	// Should have NO errors (plugin not found is not an error)
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestComputeDiffWithScopeProjectNoRemovesForUserScope(t *testing.T) {
	// Issue #101: When applying a profile at project scope in a fresh directory,
	// the diff should NOT show "Remove" actions for user-scope items.
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(pluginsDir, 0755)
	os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)

	// User scope has plugins A and B installed and enabled
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"plugin-b@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin-a@marketplace": true,
			"plugin-b@marketplace": true,
		},
	}
	marketplaces := map[string]interface{}{
		"user-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "user/marketplace",
			},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), userSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Project scope is empty (fresh project)
	// No .claude/settings.json in projectDir yet

	// Profile wants plugin-c only
	profile := &Profile{
		Name:    "project-profile",
		Plugins: []string{"plugin-c@new-marketplace"},
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "new/marketplace"},
		},
	}

	// Compute diff for PROJECT scope
	diff, err := ComputeDiffWithScope(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, DiffOptions{
		Scope:      ScopeProject,
		ProjectDir: projectDir,
	})
	if err != nil {
		t.Fatalf("ComputeDiffWithScope failed: %v", err)
	}

	// Should NOT show user-scope plugins as removals
	if len(diff.PluginsToRemove) != 0 {
		t.Errorf("Project scope diff should not remove user-scope plugins, got: %v", diff.PluginsToRemove)
	}

	// Should NOT show user-scope marketplaces as removals
	if len(diff.MarketplacesToRemove) != 0 {
		t.Errorf("Project scope diff should not remove user-scope marketplaces, got: %v", diff.MarketplacesToRemove)
	}

	// Should show plugin-c as install
	if len(diff.PluginsToInstall) != 1 || diff.PluginsToInstall[0] != "plugin-c@new-marketplace" {
		t.Errorf("Expected to install plugin-c, got: %v", diff.PluginsToInstall)
	}

	// Should show new marketplace as add
	if len(diff.MarketplacesToAdd) != 1 || diff.MarketplacesToAdd[0].Repo != "new/marketplace" {
		t.Errorf("Expected to add new/marketplace, got: %v", diff.MarketplacesToAdd)
	}
}

func TestComputeDiffWithScopeUserStillRemoves(t *testing.T) {
	// Verify that user scope diff still has declarative behavior (removes extras)
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// User scope has plugins A and B
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"plugin-b@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin-a@marketplace": true,
			"plugin-b@marketplace": true,
		},
	}
	marketplaces := map[string]interface{}{
		"old-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "old/marketplace",
			},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), userSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile wants only plugin-c (not A or B)
	profile := &Profile{
		Name:    "user-profile",
		Plugins: []string{"plugin-c@new-marketplace"},
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "new/marketplace"},
		},
	}

	// Compute diff for USER scope (default)
	diff, err := ComputeDiffWithScope(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, DiffOptions{
		Scope: ScopeUser,
	})
	if err != nil {
		t.Fatalf("ComputeDiffWithScope failed: %v", err)
	}

	// Should show user-scope plugins as removals (declarative behavior)
	if len(diff.PluginsToRemove) != 2 {
		t.Errorf("User scope diff should remove extra plugins, got: %v", diff.PluginsToRemove)
	}

	// Should show old marketplace as removal
	if len(diff.MarketplacesToRemove) != 1 {
		t.Errorf("User scope diff should remove extra marketplaces, got: %v", diff.MarketplacesToRemove)
	}

	// Should show plugin-c as install
	if len(diff.PluginsToInstall) != 1 {
		t.Errorf("Expected to install 1 plugin, got: %v", diff.PluginsToInstall)
	}
}

func TestComputeDiffWithScopeProjectExistingState(t *testing.T) {
	// Test project scope diff when project already has some plugins
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(pluginsDir, 0755)
	os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)

	// User scope has plugins A and B
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin-a@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
			"plugin-b@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin-a@marketplace": true,
			"plugin-b@marketplace": true,
		},
	}
	// Project scope has plugin-x enabled
	projectSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin-x@marketplace": true,
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), userSettings)
	writeTestJSON(t, filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile wants plugin-c only
	profile := &Profile{
		Name:    "project-profile",
		Plugins: []string{"plugin-c@marketplace"},
	}

	// Compute diff for PROJECT scope
	diff, err := ComputeDiffWithScope(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, DiffOptions{
		Scope:      ScopeProject,
		ProjectDir: projectDir,
	})
	if err != nil {
		t.Fatalf("ComputeDiffWithScope failed: %v", err)
	}

	// Should show plugin-x as removal (it's in project scope)
	if len(diff.PluginsToRemove) != 1 || diff.PluginsToRemove[0] != "plugin-x@marketplace" {
		t.Errorf("Project scope diff should remove plugin-x (project-scope), got: %v", diff.PluginsToRemove)
	}

	// Should NOT remove user-scope plugins A and B
	for _, p := range diff.PluginsToRemove {
		if p == "plugin-a@marketplace" || p == "plugin-b@marketplace" {
			t.Errorf("Project scope diff should NOT remove user-scope plugin %s", p)
		}
	}
}

func TestComputeDiffWithScopeLocalBehavesLikeProject(t *testing.T) {
	// Test that local scope behaves like project scope:
	// doesn't show user-scope items as removals
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(pluginsDir, 0755)
	os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)

	// User scope has marketplaces and plugins
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"user-plugin@marketplace": []map[string]interface{}{{"scope": "user", "version": "1.0"}},
		},
	}
	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"user-plugin@marketplace": true,
		},
	}
	marketplaces := map[string]interface{}{
		"user-marketplace": map[string]interface{}{
			"source": map[string]interface{}{"source": "github", "repo": "user/marketplace"},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), userSettings)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	// Profile with different marketplace
	profile := &Profile{
		Name:    "local-profile",
		Plugins: []string{"local-plugin@new-marketplace"},
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "new/marketplace"},
		},
	}

	// Compute diff for LOCAL scope
	diff, err := ComputeDiffWithScope(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, DiffOptions{
		Scope:      ScopeLocal,
		ProjectDir: projectDir,
	})
	if err != nil {
		t.Fatalf("ComputeDiffWithScope failed: %v", err)
	}

	// Should NOT show user-scope marketplaces as removals
	if len(diff.MarketplacesToRemove) != 0 {
		t.Errorf("Local scope diff should not remove user-scope marketplaces, got: %v", diff.MarketplacesToRemove)
	}

	// Should still add the new marketplace
	if len(diff.MarketplacesToAdd) != 1 {
		t.Errorf("Local scope diff should add new marketplace, got: %v", diff.MarketplacesToAdd)
	}
}

func TestResetHandlesPluginNotFoundError(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Current state: no plugins actually installed
	currentPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{},
	}
	// But profile thinks there should be plugins from this marketplace
	marketplaces := map[string]interface{}{
		"test-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "test/marketplace",
			},
		},
	}
	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), currentPlugins)
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)
	writeTestJSON(t, filepath.Join(tmpDir, ".claude.json"), map[string]interface{}{})

	profile := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "test/marketplace"},
		},
	}

	// Executor that simulates "not found" error
	executor := &mockExecutorWithOutput{
		outputs: map[string]string{
			"plugin uninstall gopls-lsp@claude-plugins-official": `✘ Failed to uninstall plugin "gopls-lsp@claude-plugins-official": Plugin "gopls-lsp@claude-plugins-official" not found in installed plugins`,
		},
	}

	result, err := ResetWithExecutor(profile, claudeDir, filepath.Join(tmpDir, ".claude.json"), claudeDir, executor)
	if err != nil {
		t.Fatalf("Reset should not return error: %v", err)
	}

	// Should have NO errors (plugin not found is not an error when trying to remove it)
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors for 'not found' during reset, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestApplyWithExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := tmpDir

	// Create local directory structure (in claudeupHome/local/)
	localDir := filepath.Join(claudeDir, "ext")
	agentsDir := filepath.Join(localDir, "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)

	// Create profile with Extensions
	p := &Profile{
		Name: "test-local",
		Extensions: &ExtensionSettings{
			Agents: []string{"gsd-*"},
		},
	}

	// Apply extensions
	err := applyExtensions(p, claudeDir, claudeDir)
	if err != nil {
		t.Fatalf("applyExtensions() error = %v", err)
	}

	// Verify symlinks created
	symlinkPath := filepath.Join(claudeDir, "agents", "gsd-planner.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created for gsd-planner.md")
	}

	// Verify second symlink
	symlinkPath2 := filepath.Join(claudeDir, "agents", "gsd-executor.md")
	if _, err := os.Lstat(symlinkPath2); os.IsNotExist(err) {
		t.Error("Symlink was not created for gsd-executor.md")
	}
}

func TestApplyWithSettingsHooks(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := tmpDir

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

	// Apply settings hooks
	err := applySettingsHooks(p, claudeDir)
	if err != nil {
		t.Fatalf("applySettingsHooks() error = %v", err)
	}

	// Verify settings.json was updated
	data, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if !strings.Contains(string(data), "SessionStart") {
		t.Error("SessionStart hook was not added to settings.json")
	}
	if !strings.Contains(string(data), "test.js") {
		t.Error("Hook command was not added to settings.json")
	}
}

func TestApplyExtensionsNilProfile(t *testing.T) {
	tmpDir := t.TempDir()

	// Profile with no Extensions
	p := &Profile{
		Name: "test-no-local",
	}

	// Should return nil with no error
	err := applyExtensions(p, tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("applyExtensions() should return nil for nil Extensions, got error = %v", err)
	}
}

func TestApplySettingsHooksEmptyHooks(t *testing.T) {
	tmpDir := t.TempDir()

	// Profile with empty SettingsHooks
	p := &Profile{
		Name:          "test-empty-hooks",
		SettingsHooks: map[string][]HookEntry{},
	}

	// Should return nil with no error
	err := applySettingsHooks(p, tmpDir)
	if err != nil {
		t.Fatalf("applySettingsHooks() should return nil for empty hooks, got error = %v", err)
	}
}

func TestFilterValidMarketplaceKeys(t *testing.T) {
	tests := []struct {
		name         string
		marketplaces []Marketplace
		want         []string
	}{
		{
			name:         "empty list",
			marketplaces: []Marketplace{},
			want:         nil,
		},
		{
			name: "all valid repos",
			marketplaces: []Marketplace{
				{Source: "github", Repo: "owner/repo1"},
				{Source: "github", Repo: "owner/repo2"},
			},
			want: []string{"owner/repo1", "owner/repo2"},
		},
		{
			name: "mixed repo and url",
			marketplaces: []Marketplace{
				{Source: "github", Repo: "owner/repo"},
				{Source: "git", URL: "https://example.com/repo.git"},
			},
			want: []string{"owner/repo", "https://example.com/repo.git"},
		},
		{
			name: "filters empty keys",
			marketplaces: []Marketplace{
				{Source: "github", Repo: "owner/repo1"},
				{Source: "github", Repo: ""}, // empty repo
				{Source: "git", URL: ""},     // empty url
				{Source: "github", Repo: "owner/repo2"},
			},
			want: []string{"owner/repo1", "owner/repo2"},
		},
		{
			name: "all empty",
			marketplaces: []Marketplace{
				{Source: "github", Repo: ""},
				{Source: "git", URL: ""},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterValidMarketplaceKeys(tt.marketplaces)
			if len(got) != len(tt.want) {
				t.Errorf("filterValidMarketplaceKeys() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("filterValidMarketplaceKeys()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
