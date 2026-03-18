// ABOUTME: Unit tests for ApplyAllScopes function
// ABOUTME: Tests applying multi-scope profiles to correct scope locations
package profile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/secrets"
)

func TestApplyAllScopesMultiScope(t *testing.T) {
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

	// Initialize with empty settings
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{
		"enabledPlugins": map[string]bool{},
	})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	// Create a multi-scope profile
	profile := &Profile{
		Name: "test-multi",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-plugin@marketplace"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-plugin@marketplace"},
			},
			Local: &ScopeSettings{
				Plugins: []string{"local-plugin@marketplace"},
			},
		},
	}

	// Apply the profile (nil opts = additive user scope by default)
	result, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Verify result is non-nil
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify user-scope settings were written
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load user settings: %v", err)
	}
	if !userSettings.IsPluginEnabled("user-plugin@marketplace") {
		t.Error("expected user-plugin@marketplace to be enabled at user scope")
	}
	// User scope should NOT have project/local plugins
	if userSettings.IsPluginEnabled("project-plugin@marketplace") {
		t.Error("project-plugin should not be in user scope settings")
	}
	if userSettings.IsPluginEnabled("local-plugin@marketplace") {
		t.Error("local-plugin should not be in user scope settings")
	}

	// Verify project-scope settings were written
	projectSettings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("failed to load project settings: %v", err)
	}
	if !projectSettings.IsPluginEnabled("project-plugin@marketplace") {
		t.Error("expected project-plugin@marketplace to be enabled at project scope")
	}

	// Verify local-scope settings were written
	localSettings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("failed to load local settings: %v", err)
	}
	if !localSettings.IsPluginEnabled("local-plugin@marketplace") {
		t.Error("expected local-plugin@marketplace to be enabled at local scope")
	}
}

func TestApplyAllScopesLegacyProfile(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create directory structure
	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Initialize with empty settings
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{
		"enabledPlugins": map[string]bool{},
	})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	// Create a legacy (flat) profile
	profile := &Profile{
		Name:    "test-legacy",
		Plugins: []string{"legacy-plugin@marketplace"},
	}

	// Apply the profile (nil opts = additive user scope by default)
	result, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Legacy profiles should apply to user scope only
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load user settings: %v", err)
	}
	if !userSettings.IsPluginEnabled("legacy-plugin@marketplace") {
		t.Error("expected legacy-plugin@marketplace to be enabled at user scope")
	}

	// Project scope should NOT have the plugin
	projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if _, err := os.Stat(projectSettingsPath); err == nil {
		projectSettings, _ := claude.LoadSettingsForScope("project", claudeDir, projectDir)
		if projectSettings != nil && projectSettings.IsPluginEnabled("legacy-plugin@marketplace") {
			t.Error("legacy profile should not apply to project scope")
		}
	}
}

func TestApplyAllScopesPartialScopes(t *testing.T) {
	// Test profile with only some scopes populated
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{
		"enabledPlugins": map[string]bool{},
	})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	// Profile with only user and project scope (no local)
	profile := &Profile{
		Name: "partial",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-only@marketplace"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-only@marketplace"},
			},
			// Local is nil
		},
	}

	result, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// User scope should have its plugin
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load user settings: %v", err)
	}
	if !userSettings.IsPluginEnabled("user-only@marketplace") {
		t.Error("expected user-only@marketplace to be enabled")
	}

	// Project scope should have its plugin
	projectSettings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("failed to load project settings: %v", err)
	}
	if !projectSettings.IsPluginEnabled("project-only@marketplace") {
		t.Error("expected project-only@marketplace to be enabled")
	}
}

func TestApplyAllScopesPreservesExistingSettings(t *testing.T) {
	// Test that applying doesn't wipe out other settings or existing plugins (additive behavior)
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Pre-existing settings with other fields and existing plugins
	existingSettings := map[string]any{
		"enabledPlugins": map[string]bool{
			"existing-plugin@marketplace": true,
		},
		"someOtherSetting": "should-be-preserved",
	}
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), existingSettings)
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	profile := &Profile{
		Name: "new-profile",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"new-plugin@marketplace"},
			},
		},
	}

	// Apply with default options (additive for user scope)
	_, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Load raw settings to check other fields
	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	// Check that other settings are preserved
	if settings["someOtherSetting"] != "should-be-preserved" {
		t.Error("expected someOtherSetting to be preserved")
	}

	// Check that existing plugins are preserved (additive behavior)
	enabledPlugins := settings["enabledPlugins"].(map[string]any)
	if enabledPlugins["existing-plugin@marketplace"] != true {
		t.Error("expected existing-plugin@marketplace to be preserved (additive behavior)")
	}
	if enabledPlugins["new-plugin@marketplace"] != true {
		t.Error("expected new-plugin@marketplace to be added")
	}
}

func TestApplyAllScopesReplaceUserScope(t *testing.T) {
	// Test that ReplaceUserScope option removes existing plugins
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Pre-existing settings with existing plugins
	existingSettings := map[string]any{
		"enabledPlugins": map[string]bool{
			"existing-plugin@marketplace":  true,
			"another-existing@marketplace": true,
		},
	}
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), existingSettings)
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	profile := &Profile{
		Name: "new-profile",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"new-plugin@marketplace"},
			},
		},
	}

	// Apply with ReplaceUserScope = true
	opts := &ApplyAllScopesOptions{
		ReplaceUserScope: true,
	}
	_, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Load settings
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	// Existing plugins should be removed (replace behavior)
	if userSettings.IsPluginEnabled("existing-plugin@marketplace") {
		t.Error("expected existing-plugin@marketplace to be removed (replace behavior)")
	}
	if userSettings.IsPluginEnabled("another-existing@marketplace") {
		t.Error("expected another-existing@marketplace to be removed (replace behavior)")
	}

	// New plugin should be present
	if !userSettings.IsPluginEnabled("new-plugin@marketplace") {
		t.Error("expected new-plugin@marketplace to be enabled")
	}
}

// allScopesTestEnv creates an isolated test environment for ApplyAllScopes tests
type allScopesTestEnv struct {
	tempDir        string
	claudeDir      string
	projectDir     string
	claudeJSONPath string
	claudeupHome   string
}

func setupAllScopesTestEnv(t *testing.T) *allScopesTestEnv {
	t.Helper()
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeupHome := filepath.Join(tempDir, ".claudeup")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)
	mustMkdir(t, filepath.Join(projectDir, ".claude"))
	mustMkdir(t, claudeupHome)

	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{
		"enabledPlugins": map[string]bool{},
	})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, filepath.Join(claudeDir, ".claude.json"), map[string]any{"mcpServers": map[string]any{}})

	return &allScopesTestEnv{
		tempDir:        tempDir,
		claudeDir:      claudeDir,
		projectDir:     projectDir,
		claudeJSONPath: filepath.Join(claudeDir, ".claude.json"),
		claudeupHome:   claudeupHome,
	}
}

// allScopesMockExecutor records commands for ApplyAllScopes tests
type allScopesMockExecutor struct {
	commands         [][]string
	failOn           map[string]bool   // command prefix → fail with empty output
	failOnWithOutput map[string]string // command prefix → fail with this output string
}

func (m *allScopesMockExecutor) Run(args ...string) error {
	m.commands = append(m.commands, args)
	if len(args) > 0 && m.failOn != nil {
		key := strings.Join(args[:min(3, len(args))], " ")
		if m.failOn[key] {
			return fmt.Errorf("mock failure for: %s", key)
		}
	}
	return nil
}

func (m *allScopesMockExecutor) RunWithOutput(args ...string) (string, error) {
	m.commands = append(m.commands, args)
	if len(args) > 0 {
		key := strings.Join(args[:min(3, len(args))], " ")
		if m.failOnWithOutput != nil {
			if output, ok := m.failOnWithOutput[key]; ok {
				return output, fmt.Errorf("mock failure for: %s", key)
			}
		}
		if m.failOn != nil && m.failOn[key] {
			return "", fmt.Errorf("mock failure for: %s", key)
		}
	}
	return "", nil
}

func (m *allScopesMockExecutor) hasCommand(prefix ...string) bool {
	target := strings.Join(prefix, " ")
	for _, cmd := range m.commands {
		if strings.HasPrefix(strings.Join(cmd, " "), target) {
			return true
		}
	}
	return false
}

func (m *allScopesMockExecutor) commandsWithPrefix(prefix ...string) [][]string {
	target := strings.Join(prefix, " ")
	var matches [][]string
	for _, cmd := range m.commands {
		if strings.HasPrefix(strings.Join(cmd, " "), target) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

func TestApplyAllScopesRegistersMarketplaces(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-marketplaces",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/marketplace-one"},
			{Source: "github", Repo: "org/marketplace-two"},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"plugin-a@marketplace-one"},
			},
		},
	}

	opts := &ApplyAllScopesOptions{
		Executor: executor,
	}
	_, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Verify marketplace add commands were executed
	marketplaceCmds := executor.commandsWithPrefix("plugin", "marketplace", "add")
	if len(marketplaceCmds) != 2 {
		t.Errorf("expected 2 marketplace add commands, got %d: %v", len(marketplaceCmds), executor.commands)
	}

	if !executor.hasCommand("plugin", "marketplace", "add", "org/marketplace-one") {
		t.Error("expected marketplace add for org/marketplace-one")
	}
	if !executor.hasCommand("plugin", "marketplace", "add", "org/marketplace-two") {
		t.Error("expected marketplace add for org/marketplace-two")
	}
}

func TestApplyAllScopesInstallsPlugins(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-plugins",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-plugin@mp"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-plugin@mp"},
			},
		},
	}

	opts := &ApplyAllScopesOptions{
		Executor: executor,
	}
	_, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Verify plugin install commands were issued
	pluginCmds := executor.commandsWithPrefix("plugin", "install")
	if len(pluginCmds) != 2 {
		t.Errorf("expected exactly 2 plugin install commands, got %d: %v", len(pluginCmds), executor.commands)
	}

	if !executor.hasCommand("plugin", "install", "user-plugin@mp") {
		t.Error("expected plugin install for user-plugin@mp")
	}
	// Project-scope plugins should have --scope project
	if !executor.hasCommand("plugin", "install", "--scope", "project", "project-plugin@mp") {
		t.Error("expected plugin install --scope project for project-plugin@mp")
	}
}

func TestApplyAllScopesWritesMCPJSON(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-mcp",
		PerScope: &PerScopeSettings{
			Project: &ScopeSettings{
				Plugins: []string{"some-plugin@mp"},
				MCPServers: []MCPServer{
					{Name: "test-server", Command: "node", Args: []string{"server.js"}},
				},
			},
		},
	}

	opts := &ApplyAllScopesOptions{
		Executor: executor,
	}
	_, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Verify .mcp.json was written in project directory
	mcpPath := filepath.Join(env.projectDir, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("expected .mcp.json to be written: %v", err)
	}

	var mcpConfig map[string]any
	if err := json.Unmarshal(data, &mcpConfig); err != nil {
		t.Fatalf("failed to parse .mcp.json: %v", err)
	}

	servers, ok := mcpConfig["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("expected mcpServers key in .mcp.json")
	}
	if _, exists := servers["test-server"]; !exists {
		t.Error("expected test-server in .mcp.json")
	}
}

func TestApplyAllScopesMergesSettingsHooks(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-hooks",
		SettingsHooks: map[string][]HookEntry{
			"PostToolUse": {
				{Type: "command", Command: "echo hello"},
			},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"some-plugin@mp"},
			},
		},
	}

	opts := &ApplyAllScopesOptions{
		Executor: executor,
	}
	_, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Verify hooks were merged into settings.json
	data, err := os.ReadFile(filepath.Join(env.claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var rawSettings map[string]any
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		t.Fatalf("failed to parse settings.json: %v", err)
	}

	hooks, ok := rawSettings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("expected hooks in settings.json")
	}

	postToolUse, ok := hooks["PostToolUse"].([]any)
	if !ok {
		t.Fatal("expected PostToolUse hooks array")
	}
	if len(postToolUse) == 0 {
		t.Error("expected at least one PostToolUse hook entry")
	}
}

func TestApplyAllScopesMarketplacesBeforePlugins(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-ordering",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/my-marketplace"},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"my-plugin@my-marketplace"},
			},
		},
	}

	opts := &ApplyAllScopesOptions{
		Executor: executor,
	}
	_, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Find indices of first marketplace add and first plugin install
	firstMarketplace := -1
	firstPlugin := -1
	for i, cmd := range executor.commands {
		cmdStr := strings.Join(cmd, " ")
		if firstMarketplace == -1 && strings.HasPrefix(cmdStr, "plugin marketplace add") {
			firstMarketplace = i
		}
		if firstPlugin == -1 && strings.HasPrefix(cmdStr, "plugin install") {
			firstPlugin = i
		}
	}

	if firstMarketplace == -1 {
		t.Fatal("expected marketplace add command")
	}
	if firstPlugin == -1 {
		t.Fatal("expected plugin install command")
	}
	if firstMarketplace >= firstPlugin {
		t.Errorf("marketplace add (index %d) must come before plugin install (index %d)", firstMarketplace, firstPlugin)
	}
}

func TestApplyAllScopesDefaultExecutor(t *testing.T) {
	// Verify that nil Executor in options creates a DefaultExecutor internally.
	// Use a profile with only hooks (no plugins/marketplaces) to avoid real CLI calls.
	env := setupAllScopesTestEnv(t)

	p := &Profile{
		Name: "test-default-exec",
		SettingsHooks: map[string][]HookEntry{
			"PreToolUse": {{Type: "command", Command: "echo test"}},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{},
		},
	}

	opts := &ApplyAllScopesOptions{}
	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes with nil Executor failed: %v", err)
	}

	// Verify hooks were applied (proves the function ran through successfully)
	data, err := os.ReadFile(filepath.Join(env.claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}
	var rawSettings map[string]any
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}
	if _, ok := rawSettings["hooks"].(map[string]any); !ok {
		t.Error("expected hooks to be written with default executor")
	}

	// No errors expected (no plugins/marketplaces to install)
	if len(result.Errors) > 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestApplyAllScopesMarketplaceAlreadyInstalled(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{
		// Return "already installed" output with an error (mimics real CLI behavior)
		failOnWithOutput: map[string]string{
			"plugin marketplace add": "marketplace org/existing: already installed",
		},
	}

	p := &Profile{
		Name: "test-already-installed",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/existing"},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{},
		},
	}

	opts := &ApplyAllScopesOptions{Executor: executor}
	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// "already installed" should count as added, not an error
	if len(result.MarketplacesAdded) != 1 || result.MarketplacesAdded[0] != "org/existing" {
		t.Errorf("expected org/existing in MarketplacesAdded, got: %v", result.MarketplacesAdded)
	}
	if len(result.Errors) > 0 {
		t.Errorf("expected no errors for already-installed marketplace, got: %v", result.Errors)
	}
}

func TestApplyAllScopesUserMCPViaCLI(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-user-mcp",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "user-server", Command: "node", Args: []string{"srv.js"}},
				},
			},
		},
	}

	opts := &ApplyAllScopesOptions{Executor: executor}
	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Should issue `mcp add user-server -s user -- node srv.js`
	if !executor.hasCommand("mcp", "add", "user-server") {
		t.Errorf("expected mcp add for user-server, got commands: %v", executor.commands)
	}
	if len(result.MCPServersInstalled) == 0 || result.MCPServersInstalled[0] != "user-server" {
		t.Errorf("expected user-server in MCPServersInstalled, got: %v", result.MCPServersInstalled)
	}
}

func TestApplyAllScopesLocalMCPViaCLI(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-local-mcp",
		PerScope: &PerScopeSettings{
			Local: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "local-server", Command: "python", Args: []string{"server.py"}},
				},
			},
		},
	}

	opts := &ApplyAllScopesOptions{Executor: executor}
	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Should issue mcp add with -s local
	if !executor.hasCommand("mcp", "add", "local-server", "-s", "local") {
		t.Errorf("expected mcp add with -s local for local-server, got commands: %v", executor.commands)
	}
	if len(result.MCPServersInstalled) == 0 || result.MCPServersInstalled[0] != "local-server" {
		t.Errorf("expected local-server in MCPServersInstalled, got: %v", result.MCPServersInstalled)
	}
}

func TestApplyAllScopesPluginCountExact(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-exact-count",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-p@mp"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-p@mp"},
			},
		},
	}

	opts := &ApplyAllScopesOptions{Executor: executor}
	_, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Exactly 2 plugin install commands -- no duplicates
	pluginCmds := executor.commandsWithPrefix("plugin", "install")
	if len(pluginCmds) != 2 {
		t.Errorf("expected exactly 2 plugin install commands, got %d: %v", len(pluginCmds), pluginCmds)
	}
}

func TestApplyAllScopesNoPluginDoubleCounting(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-no-double-count",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"my-plugin@mp"},
			},
		},
	}

	opts := &ApplyAllScopesOptions{Executor: executor}
	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Plugin should appear exactly once in PluginsInstalled (from CLI install), not twice
	count := 0
	for _, p := range result.PluginsInstalled {
		if p == "my-plugin@mp" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected my-plugin@mp exactly once in PluginsInstalled, got %d times. Full list: %v", count, result.PluginsInstalled)
	}
}

func TestApplyAllScopesMCPSecretResolution(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-mcp-secrets",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{
					{
						Name:    "secret-server",
						Command: "node",
						Args:    []string{"$MY_SECRET_TOKEN"},
						Secrets: map[string]SecretRef{
							"MY_SECRET_TOKEN": {
								Sources: []SecretSource{
									{Type: "env", Key: "MY_SECRET_TOKEN"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set the env var and create a real secret chain to resolve it
	t.Setenv("MY_SECRET_TOKEN", "resolved-value")
	chain := secrets.NewChain(secrets.NewEnvResolver())

	opts := &ApplyAllScopesOptions{Executor: executor}
	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, chain, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	if len(result.MCPServersInstalled) == 0 {
		t.Fatal("expected secret-server to be installed")
	}

	// Verify the resolved value was passed, not the raw $MY_SECRET_TOKEN
	mcpCmds := executor.commandsWithPrefix("mcp", "add", "secret-server")
	if len(mcpCmds) == 0 {
		t.Fatal("expected mcp add command for secret-server")
	}
	cmdStr := strings.Join(mcpCmds[0], " ")
	if strings.Contains(cmdStr, "$MY_SECRET_TOKEN") {
		t.Error("expected $MY_SECRET_TOKEN to be resolved, but raw variable was passed")
	}
	if !strings.Contains(cmdStr, "resolved-value") {
		t.Errorf("expected resolved-value in mcp add args, got: %s", cmdStr)
	}
}

func TestApplyAllScopesMarketplaceErrorIncludesOutput(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{
		failOnWithOutput: map[string]string{
			"plugin marketplace add": "network timeout: connection refused",
		},
	}

	p := &Profile{
		Name: "test-marketplace-error",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/failing-mp"},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{},
		},
	}

	opts := &ApplyAllScopesOptions{Executor: executor}
	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Fatal("expected error for failing marketplace add")
	}

	errMsg := result.Errors[0].Error()
	if !strings.Contains(errMsg, "network timeout") {
		t.Errorf("expected error to include CLI output, got: %s", errMsg)
	}
}

func TestApplyAllScopesMCPAlreadyExists(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	executor := &allScopesMockExecutor{
		failOnWithOutput: map[string]string{
			"mcp add context7": "MCP server context7 already exists in user config",
		},
	}

	p := &Profile{
		Name: "test-mcp-exists",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "context7", Command: "npx", Args: []string{"-y", "@context7/mcp"}},
				},
			},
		},
	}

	result, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, &ApplyAllScopesOptions{
		Executor: executor,
		Output:   io.Discard,
	})
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}

	if len(result.MCPServersAlreadyPresent) != 1 || result.MCPServersAlreadyPresent[0] != "context7" {
		t.Errorf("Expected context7 in MCPServersAlreadyPresent, got: %v", result.MCPServersAlreadyPresent)
	}

	if len(result.MCPServersInstalled) > 0 {
		t.Errorf("Expected no MCPServersInstalled, got: %v", result.MCPServersInstalled)
	}
}

func TestApplyAllScopesInstallMarketplacesOutput(t *testing.T) {
	env := setupAllScopesTestEnv(t)
	var buf strings.Builder
	executor := &allScopesMockExecutor{}

	p := &Profile{
		Name: "test-output",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/my-mp"},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{},
		},
	}

	opts := &ApplyAllScopesOptions{
		Executor: executor,
		Output:   &buf,
	}
	_, err := ApplyAllScopes(p, env.claudeDir, env.claudeJSONPath, env.projectDir, env.claudeupHome, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "org/my-mp") {
		t.Errorf("expected marketplace name in output, got: %q", output)
	}
}
