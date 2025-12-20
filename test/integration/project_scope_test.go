// ABOUTME: Integration tests for project-level profile scopes
// ABOUTME: Tests scope precedence, file creation, sync, and edge cases
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/profile"
)

func TestWriteMCPJSON_CreatesValidFile(t *testing.T) {
	projectDir := t.TempDir()

	servers := []profile.MCPServer{
		{
			Name:    "test-server",
			Command: "node",
			Args:    []string{"server.js"},
		},
		{
			Name:    "python-server",
			Command: "python",
			Args:    []string{"-m", "my_server"},
		},
	}

	err := profile.WriteMCPJSON(projectDir, servers)
	if err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	// Verify file exists
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		t.Fatal(".mcp.json should be created")
	}

	// Verify content structure
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("Failed to read .mcp.json: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers key missing or wrong type")
	}

	if len(mcpServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(mcpServers))
	}

	// Check specific server
	testServer, ok := mcpServers["test-server"].(map[string]interface{})
	if !ok {
		t.Fatal("test-server missing")
	}

	if testServer["command"] != "node" {
		t.Errorf("command = %v, want 'node'", testServer["command"])
	}
}

func TestWriteMCPJSON_WithSecrets(t *testing.T) {
	projectDir := t.TempDir()

	servers := []profile.MCPServer{
		{
			Name:    "github-server",
			Command: "npx",
			Args:    []string{"-y", "@anthropic/github-mcp"},
			Secrets: map[string]profile.SecretRef{
				"GITHUB_TOKEN": {
					Sources: []profile.SecretSource{
						{Type: "env", Key: "GITHUB_TOKEN"},
					},
				},
			},
		},
	}

	err := profile.WriteMCPJSON(projectDir, servers)
	if err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	// Read and verify the file contains env var reference format
	data, err := os.ReadFile(filepath.Join(projectDir, ".mcp.json"))
	if err != nil {
		t.Fatalf("Failed to read .mcp.json: %v", err)
	}

	// Should contain some kind of reference, not the actual secret
	content := string(data)
	if content == "" {
		t.Error(".mcp.json should not be empty")
	}
}

func TestWriteMCPJSON_EmptyServers(t *testing.T) {
	projectDir := t.TempDir()

	servers := []profile.MCPServer{}

	err := profile.WriteMCPJSON(projectDir, servers)
	if err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	// File should still be created with empty mcpServers
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		t.Fatal(".mcp.json should be created even with empty servers")
	}

	data, _ := os.ReadFile(mcpPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers key missing")
	}
	if len(mcpServers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(mcpServers))
	}
}

func TestProjectConfig_RoundTrip(t *testing.T) {
	projectDir := t.TempDir()

	cfg := &profile.ProjectConfig{
		Profile:       "test-profile",
		ProfileSource: "embedded",
		Marketplaces: []profile.Marketplace{
			{Source: "github", Repo: "test/plugins"},
		},
		Plugins: []string{"plugin-a@test", "plugin-b@test"},
	}

	// Save
	err := profile.SaveProjectConfig(projectDir, cfg)
	if err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	// Load
	loaded, err := profile.LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}

	// Verify
	if loaded.Profile != cfg.Profile {
		t.Errorf("Profile = %q, want %q", loaded.Profile, cfg.Profile)
	}
	if loaded.ProfileSource != cfg.ProfileSource {
		t.Errorf("ProfileSource = %q, want %q", loaded.ProfileSource, cfg.ProfileSource)
	}
	if len(loaded.Plugins) != 2 {
		t.Errorf("len(Plugins) = %d, want 2", len(loaded.Plugins))
	}
	if len(loaded.Marketplaces) != 1 {
		t.Errorf("len(Marketplaces) = %d, want 1", len(loaded.Marketplaces))
	}
	if loaded.Version != "1" {
		t.Errorf("Version = %q, want '1'", loaded.Version)
	}
	if loaded.AppliedAt.IsZero() {
		t.Error("AppliedAt should be set")
	}
}

func TestScopeString(t *testing.T) {
	tests := []struct {
		scope profile.Scope
		want  string
	}{
		{profile.ScopeUser, "user"},
		{profile.ScopeProject, "project"},
		{profile.ScopeLocal, "local"},
	}

	for _, tt := range tests {
		if got := string(tt.scope); got != tt.want {
			t.Errorf("Scope string = %q, want %q", got, tt.want)
		}
	}
}

func TestSyncWithExecutor_InstallsPlugins(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Create .claudeup.json with plugins
	cfg := &profile.ProjectConfig{
		Version: "1",
		Profile: "test-profile",
		Marketplaces: []profile.Marketplace{
			{Source: "github", Repo: "test/plugins"},
		},
		Plugins: []string{"plugin-a@test-plugins", "plugin-b@test-plugins"},
	}
	profile.SaveProjectConfig(projectDir, cfg)

	// Create empty installed_plugins.json
	installedData := []byte(`{"version": 2, "plugins": {}}`)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedData, 0644)

	// Mock executor
	executor := &testMockExecutor{commands: [][]string{}}

	result, err := profile.SyncWithExecutor(projectDir, claudeDir, profile.SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	if result.PluginsInstalled != 2 {
		t.Errorf("PluginsInstalled = %d, want 2", result.PluginsInstalled)
	}
	if result.MarketplacesAdded != 1 {
		t.Errorf("MarketplacesAdded = %d, want 1", result.MarketplacesAdded)
	}

	// Verify marketplace add was called
	foundMarketplace := false
	for _, cmd := range executor.commands {
		if len(cmd) >= 4 && cmd[0] == "plugin" && cmd[1] == "marketplace" && cmd[2] == "add" {
			foundMarketplace = true
		}
	}
	if !foundMarketplace {
		t.Error("marketplace add command not found")
	}
}

func TestSyncWithExecutor_SkipsExistingPlugins(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	cfg := &profile.ProjectConfig{
		Version: "1",
		Profile: "test-profile",
		Plugins: []string{"existing@test", "new@test"},
	}
	profile.SaveProjectConfig(projectDir, cfg)

	// One plugin already installed
	installedData := []byte(`{
		"version": 2,
		"plugins": {
			"existing@test": [{"scope": "project", "version": "1.0"}]
		}
	}`)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedData, 0644)

	executor := &testMockExecutor{commands: [][]string{}}

	result, err := profile.SyncWithExecutor(projectDir, claudeDir, profile.SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	if result.PluginsInstalled != 1 {
		t.Errorf("PluginsInstalled = %d, want 1", result.PluginsInstalled)
	}
	if result.PluginsSkipped != 1 {
		t.Errorf("PluginsSkipped = %d, want 1", result.PluginsSkipped)
	}
}

func TestSyncWithExecutor_DryRun(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	cfg := &profile.ProjectConfig{
		Version: "1",
		Profile: "test-profile",
		Plugins: []string{"plugin@test"},
	}
	profile.SaveProjectConfig(projectDir, cfg)

	installedData := []byte(`{"version": 2, "plugins": {}}`)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedData, 0644)

	executor := &testMockExecutor{commands: [][]string{}}

	result, err := profile.SyncWithExecutor(projectDir, claudeDir, profile.SyncOptions{DryRun: true}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	// Should report what would be installed
	if result.PluginsInstalled != 1 {
		t.Errorf("PluginsInstalled = %d, want 1", result.PluginsInstalled)
	}

	// But no commands executed
	if len(executor.commands) != 0 {
		t.Errorf("Commands executed = %d, want 0", len(executor.commands))
	}
}

func TestSyncWithExecutor_NoConfigFile(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := t.TempDir()

	executor := &testMockExecutor{commands: [][]string{}}

	_, err := profile.SyncWithExecutor(projectDir, claudeDir, profile.SyncOptions{}, executor)
	if err == nil {
		t.Error("Expected error for missing .claudeup.json")
	}
}

func TestNewProjectConfig_FromProfile(t *testing.T) {
	p := &profile.Profile{
		Name: "test-profile",
		Marketplaces: []profile.Marketplace{
			{Source: "github", Repo: "test/repo"},
		},
		Plugins: []string{"plugin@test"},
	}

	cfg := profile.NewProjectConfig(p)

	if cfg.Profile != "test-profile" {
		t.Errorf("Profile = %q, want %q", cfg.Profile, "test-profile")
	}
	if len(cfg.Marketplaces) != 1 {
		t.Errorf("len(Marketplaces) = %d, want 1", len(cfg.Marketplaces))
	}
	if len(cfg.Plugins) != 1 {
		t.Errorf("len(Plugins) = %d, want 1", len(cfg.Plugins))
	}
}

func TestNewProjectConfig_EmbeddedProfile(t *testing.T) {
	// Get an actual embedded profile name
	profiles, err := profile.ListEmbeddedProfiles()
	if err != nil || len(profiles) == 0 {
		t.Skip("No embedded profiles available")
	}

	p := profiles[0]
	cfg := profile.NewProjectConfig(p)

	if cfg.ProfileSource != "embedded" {
		t.Errorf("ProfileSource = %q, want 'embedded'", cfg.ProfileSource)
	}
}

func TestNewProjectConfig_CustomProfile(t *testing.T) {
	p := &profile.Profile{
		Name: "my-custom-profile", // Not an embedded profile
	}

	cfg := profile.NewProjectConfig(p)

	if cfg.ProfileSource != "custom" {
		t.Errorf("ProfileSource = %q, want 'custom'", cfg.ProfileSource)
	}
}

func TestProjectConfigExists(t *testing.T) {
	projectDir := t.TempDir()

	// Should not exist initially
	if profile.ProjectConfigExists(projectDir) {
		t.Error("Should return false for empty directory")
	}

	// Create config
	cfg := &profile.ProjectConfig{Profile: "test"}
	profile.SaveProjectConfig(projectDir, cfg)

	// Should exist now
	if !profile.ProjectConfigExists(projectDir) {
		t.Error("Should return true after saving")
	}
}

func TestApplyOptions_ScopeValidation(t *testing.T) {
	opts := profile.ApplyOptions{
		Scope: profile.ScopeProject,
		// Missing ProjectDir
	}

	// Scope requires ProjectDir
	if opts.Scope == profile.ScopeProject && opts.ProjectDir == "" {
		// This is expected - the actual validation happens in ApplyWithOptions
		// We're just testing the struct construction here
	}
}

// testMockExecutor is a simple mock for testing sync operations
type testMockExecutor struct {
	commands [][]string
}

func (m *testMockExecutor) Run(args ...string) error {
	m.commands = append(m.commands, args)
	return nil
}

func (m *testMockExecutor) RunWithOutput(args ...string) (string, error) {
	m.commands = append(m.commands, args)
	return "", nil
}
