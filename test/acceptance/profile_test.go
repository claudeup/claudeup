// ABOUTME: Acceptance tests for profile save and create commands
// ABOUTME: Tests complete end-to-end workflows with real file operations
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/mcp"
	"github.com/claudeup/claudeup/internal/profile"
)

func TestProfileSaveAndLoad(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	// Setup: Create a complete Claude installation state
	// 1. Create marketplace
	env.CreateMarketplace("test-marketplace", "github.com/test/marketplace")

	marketplaceDir := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace")
	env.CreateMarketplaceRegistry(map[string]claude.MarketplaceMetadata{
		"test-marketplace": {
			Source: claude.MarketplaceSource{
				Source: "github",
				Repo:   "github.com/test/marketplace",
			},
			InstallLocation: marketplaceDir,
			LastUpdated:     "2024-01-01T00:00:00Z",
		},
	})

	// 2. Create plugins with MCP servers
	env.CreatePlugin("plugin1", "test-marketplace", "1.0.0", map[string]mcp.ServerDefinition{
		"server1": {
			Command: "node",
			Args:    []string{"server.js"},
			Env: map[string]string{
				"API_KEY": "secret123",
			},
		},
	})

	env.CreatePlugin("plugin2", "test-marketplace", "2.0.0", map[string]mcp.ServerDefinition{
		"server2": {
			Command: "python",
			Args:    []string{"-m", "server"},
		},
	})

	// 3. Register plugins
	plugin1Path := filepath.Join(marketplaceDir, "plugins", "plugin1")
	plugin2Path := filepath.Join(marketplaceDir, "plugins", "plugin2")

	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"plugin1@test-marketplace": {
			Version:      "1.0.0",
			InstallPath:  plugin1Path,
			GitCommitSha: "abc123",
			IsLocal:      false,
		},
		"plugin2@test-marketplace": {
			Version:      "2.0.0",
			InstallPath:  plugin2Path,
			GitCommitSha: "def456",
			IsLocal:      true,
		},
	})

	// 4. Create claude_desktop_config.json with MCP servers
	env.CreateClaudeJSON(map[string]interface{}{
		"server1": map[string]interface{}{
			"command": "node",
			"args":    []interface{}{"server.js"},
			"env": map[string]interface{}{
				"API_KEY": "secret123",
			},
		},
		"server2": map[string]interface{}{
			"command": "python",
			"args":    []interface{}{"-m", "server"},
		},
	})

	// Action: Take a snapshot and save as profile
	snapshot, err := profile.Snapshot("my-dev-setup", env.ClaudeDir, env.ClaudeJSONPath)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if err := profile.Save(env.ProfilesDir, snapshot); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Verify: Profile file exists
	if !env.ProfileExists("my-dev-setup") {
		t.Error("Profile file should exist")
	}

	// Verify: Load and check profile contents
	loaded, err := profile.Load(env.ProfilesDir, "my-dev-setup")
	if err != nil {
		t.Fatalf("Failed to load profile: %v", err)
	}

	// Check profile metadata
	if loaded.Name != "my-dev-setup" {
		t.Errorf("Expected name 'my-dev-setup', got %q", loaded.Name)
	}

	// Check marketplaces
	if len(loaded.Marketplaces) != 1 {
		t.Fatalf("Expected 1 marketplace, got %d", len(loaded.Marketplaces))
	}

	marketplace := loaded.Marketplaces[0]
	if marketplace.Source != "github" {
		t.Errorf("Expected source 'github', got %q", marketplace.Source)
	}
	if marketplace.Repo != "github.com/test/marketplace" {
		t.Errorf("Expected repo 'github.com/test/marketplace', got %q", marketplace.Repo)
	}

	// Check plugins (stored as strings like "plugin1@test-marketplace")
	if len(loaded.Plugins) != 2 {
		t.Fatalf("Expected 2 plugins, got %d", len(loaded.Plugins))
	}

	// Verify plugin names
	pluginSet := make(map[string]bool)
	for _, p := range loaded.Plugins {
		pluginSet[p] = true
	}

	if !pluginSet["plugin1@test-marketplace"] {
		t.Error("plugin1@test-marketplace not found in profile")
	}
	if !pluginSet["plugin2@test-marketplace"] {
		t.Error("plugin2@test-marketplace not found in profile")
	}

	// Check MCP servers
	if len(loaded.MCPServers) != 2 {
		t.Fatalf("Expected 2 MCP servers, got %d", len(loaded.MCPServers))
	}

	// Verify server1 configuration
	var server1Found bool
	for _, server := range loaded.MCPServers {
		if server.Name == "server1" {
			server1Found = true
			if server.Command != "node" {
				t.Errorf("Expected command 'node', got %q", server.Command)
			}
			if len(server.Args) != 1 || server.Args[0] != "server.js" {
				t.Errorf("Expected args [server.js], got %v", server.Args)
			}
		}
	}
	if !server1Found {
		t.Error("server1 not found in MCP servers")
	}
}

func TestProfileCreate(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	// Setup: Create minimal Claude installation
	env.CreateMarketplace("minimal-marketplace", "github.com/minimal/repo")

	marketplaceDir := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "minimal-marketplace")
	env.CreateMarketplaceRegistry(map[string]claude.MarketplaceMetadata{
		"minimal-marketplace": {
			Source: claude.MarketplaceSource{
				Source: "github",
				Repo:   "github.com/minimal/repo",
			},
			InstallLocation: marketplaceDir,
			LastUpdated:     "2024-01-01T00:00:00Z",
		},
	})

	env.CreatePlugin("simple-plugin", "minimal-marketplace", "1.0.0", nil)

	pluginPath := filepath.Join(marketplaceDir, "plugins", "simple-plugin")
	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"simple-plugin@minimal-marketplace": {
			Version:     "1.0.0",
			InstallPath: pluginPath,
		},
	})

	// Action: Create profile from current state
	snapshot, err := profile.Snapshot("minimal-profile", env.ClaudeDir, env.ClaudeJSONPath)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if err := profile.Save(env.ProfilesDir, snapshot); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Verify: Profile exists and has correct structure
	if !env.ProfileExists("minimal-profile") {
		t.Fatal("Profile should exist")
	}

	loaded, err := profile.Load(env.ProfilesDir, "minimal-profile")
	if err != nil {
		t.Fatalf("Failed to load profile: %v", err)
	}

	if loaded.Name != "minimal-profile" {
		t.Errorf("Expected name 'minimal-profile', got %q", loaded.Name)
	}

	if len(loaded.Marketplaces) != 1 {
		t.Errorf("Expected 1 marketplace, got %d", len(loaded.Marketplaces))
	}

	if len(loaded.Plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(loaded.Plugins))
	}

	if len(loaded.MCPServers) != 0 {
		t.Errorf("Expected 0 MCP servers, got %d", len(loaded.MCPServers))
	}
}

func TestProfileOverwrite(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	// Setup: Create initial profile
	initialProfile := &profile.Profile{
		Name:         "test-profile",
		Marketplaces: []profile.Marketplace{},
		Plugins:      []string{},
		MCPServers:   []profile.MCPServer{},
	}

	if err := profile.Save(env.ProfilesDir, initialProfile); err != nil {
		t.Fatalf("Failed to save initial profile: %v", err)
	}

	// Action: Create new state and overwrite
	env.CreateMarketplace("new-marketplace", "github.com/new/repo")

	marketplaceDir := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "new-marketplace")
	env.CreateMarketplaceRegistry(map[string]claude.MarketplaceMetadata{
		"new-marketplace": {
			Source: claude.MarketplaceSource{
				Source: "github",
				Repo:   "github.com/new/repo",
			},
			InstallLocation: marketplaceDir,
		},
	})

	snapshot, err := profile.Snapshot("test-profile", env.ClaudeDir, env.ClaudeJSONPath)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if err := profile.Save(env.ProfilesDir, snapshot); err != nil {
		t.Fatalf("Failed to overwrite profile: %v", err)
	}

	// Verify: Profile was updated
	loaded, err := profile.Load(env.ProfilesDir, "test-profile")
	if err != nil {
		t.Fatalf("Failed to load profile: %v", err)
	}

	if len(loaded.Marketplaces) != 1 {
		t.Errorf("Expected 1 marketplace after overwrite, got %d", len(loaded.Marketplaces))
	}

	if loaded.Marketplaces[0].Repo != "github.com/new/repo" {
		t.Errorf("Expected repo 'github.com/new/repo', got %q", loaded.Marketplaces[0].Repo)
	}
}

func TestEmptyStateSnapshot(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	// No setup - test with empty Claude installation

	// Action: Create snapshot of empty state
	snapshot, err := profile.Snapshot("empty-profile", env.ClaudeDir, env.ClaudeJSONPath)
	if err != nil {
		t.Fatalf("Failed to create snapshot of empty state: %v", err)
	}

	// Verify: Snapshot should have zero items but be valid
	if len(snapshot.Marketplaces) != 0 {
		t.Errorf("Expected 0 marketplaces, got %d", len(snapshot.Marketplaces))
	}

	if len(snapshot.Plugins) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(snapshot.Plugins))
	}

	if len(snapshot.MCPServers) != 0 {
		t.Errorf("Expected 0 MCP servers, got %d", len(snapshot.MCPServers))
	}

	// Save and reload to ensure serialization works
	if err := profile.Save(env.ProfilesDir, snapshot); err != nil {
		t.Fatalf("Failed to save empty profile: %v", err)
	}

	loaded, err := profile.Load(env.ProfilesDir, "empty-profile")
	if err != nil {
		t.Fatalf("Failed to load empty profile: %v", err)
	}

	if loaded.Name != "empty-profile" {
		t.Errorf("Expected name 'empty-profile', got %q", loaded.Name)
	}
}

func TestProfileJSONFormat(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	// Create simple profile
	testProfile := &profile.Profile{
		Name: "format-test",
		Marketplaces: []profile.Marketplace{
			{
				Source: "github",
				Repo:   "test/repo",
			},
		},
		Plugins: []string{
			"test-plugin@test-marketplace",
		},
		MCPServers: []profile.MCPServer{
			{
				Name:    "test-server",
				Command: "node",
				Args:    []string{"server.js"},
			},
		},
	}

	// Save profile
	if err := profile.Save(env.ProfilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Read raw JSON and verify format
	profilePath := filepath.Join(env.ProfilesDir, "format-test.json")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("Failed to read profile file: %v", err)
	}

	var rawJSON map[string]interface{}
	if err := json.Unmarshal(data, &rawJSON); err != nil {
		t.Fatalf("Profile is not valid JSON: %v", err)
	}

	// Verify top-level structure
	requiredFields := []string{"name", "marketplaces", "plugins", "mcpServers"}
	for _, field := range requiredFields {
		if _, exists := rawJSON[field]; !exists {
			t.Errorf("Profile JSON missing required field: %s", field)
		}
	}

	// Verify name
	if name, ok := rawJSON["name"].(string); !ok || name != "format-test" {
		t.Errorf("Expected name 'format-test', got %v", rawJSON["name"])
	}
}
