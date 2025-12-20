// ABOUTME: Acceptance test helpers for end-to-end testing
// ABOUTME: Provides real Claude installation fixtures and utilities
package acceptance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/mcp"
)

// AcceptanceTestEnv represents a complete test environment with real directories
type AcceptanceTestEnv struct {
	// Test directories
	ClaudeDir   string
	ProfilesDir string
	HomeDir     string
	ConfigDir   string

	// Paths
	ClaudeJSONPath string

	t *testing.T
}

// SetupAcceptanceTestEnv creates a complete fake Claude installation
func SetupAcceptanceTestEnv(t *testing.T) *AcceptanceTestEnv {
	t.Helper()

	// Create temp directories
	tempDir, err := os.MkdirTemp("", "claudeup-acceptance-*")
	if err != nil {
		t.Fatal(err)
	}

	homeDir := filepath.Join(tempDir, "home")
	claudeDir := filepath.Join(homeDir, "Library", "Application Support", "Claude")
	configDir := filepath.Join(homeDir, ".config", "claudeup")
	profilesDir := filepath.Join(homeDir, ".claudeup", "profiles")
	claudeJSONPath := filepath.Join(claudeDir, "claude_desktop_config.json")

	env := &AcceptanceTestEnv{
		ClaudeDir:      claudeDir,
		ProfilesDir:    profilesDir,
		HomeDir:        homeDir,
		ConfigDir:      configDir,
		ClaudeJSONPath: claudeJSONPath,
		t:              t,
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(claudeDir, "plugins"),
		filepath.Join(claudeDir, "plugins", "marketplaces"),
		profilesDir,
		configDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	return env
}

// Cleanup removes all test directories
func (e *AcceptanceTestEnv) Cleanup() {
	os.RemoveAll(filepath.Dir(e.HomeDir))
}

// CreateMarketplace creates a fake marketplace with git repo
func (e *AcceptanceTestEnv) CreateMarketplace(name, repo string) {
	e.t.Helper()

	marketplaceDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", name)
	if err := os.MkdirAll(marketplaceDir, 0755); err != nil {
		e.t.Fatal(err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = marketplaceDir
	if err := cmd.Run(); err != nil {
		e.t.Fatal(err)
	}

	// Create a commit so git rev-parse HEAD works
	readmePath := filepath.Join(marketplaceDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# "+name), 0644); err != nil {
		e.t.Fatal(err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = marketplaceDir
	if err := cmd.Run(); err != nil {
		e.t.Fatal(err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = marketplaceDir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if err := cmd.Run(); err != nil {
		e.t.Fatal(err)
	}
}

// CreatePlugin creates a fake plugin in a marketplace
func (e *AcceptanceTestEnv) CreatePlugin(name, marketplace, version string, mcpServers map[string]mcp.ServerDefinition) {
	e.t.Helper()

	pluginDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", marketplace, "plugins", name)
	if err := os.MkdirAll(filepath.Join(pluginDir, ".claude-plugin"), 0755); err != nil {
		e.t.Fatal(err)
	}

	// Create plugin.json
	pluginJSON := mcp.PluginJSON{
		Name:       name,
		Version:    version,
		MCPServers: mcpServers,
	}

	data, err := json.MarshalIndent(pluginJSON, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}

	pluginJSONPath := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
	if err := os.WriteFile(pluginJSONPath, data, 0644); err != nil {
		e.t.Fatal(err)
	}
}

// CreatePluginRegistry creates installed_plugins.json
func (e *AcceptanceTestEnv) CreatePluginRegistry(plugins map[string]claude.PluginMetadata) {
	e.t.Helper()

	// Convert to V2 format
	pluginsV2 := make(map[string][]claude.PluginMetadata)
	for name, meta := range plugins {
		// Ensure scope is set
		if meta.Scope == "" {
			meta.Scope = "user"
		}
		pluginsV2[name] = []claude.PluginMetadata{meta}
	}

	registry := &claude.PluginRegistry{
		Version: 2,
		Plugins: pluginsV2,
	}

	if err := claude.SavePlugins(e.ClaudeDir, registry); err != nil {
		e.t.Fatal(err)
	}
}

// CreateMarketplaceRegistry creates known_marketplaces.json
func (e *AcceptanceTestEnv) CreateMarketplaceRegistry(marketplaces map[string]claude.MarketplaceMetadata) {
	e.t.Helper()

	registry := claude.MarketplaceRegistry(marketplaces)

	if err := claude.SaveMarketplaces(e.ClaudeDir, registry); err != nil {
		e.t.Fatal(err)
	}
}

// CreateClaudeJSON creates claude_desktop_config.json with MCP servers
func (e *AcceptanceTestEnv) CreateClaudeJSON(mcpServers map[string]interface{}) {
	e.t.Helper()

	config := map[string]interface{}{
		"mcpServers": mcpServers,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}

	if err := os.WriteFile(e.ClaudeJSONPath, data, 0644); err != nil {
		e.t.Fatal(err)
	}
}

// LoadPluginRegistry loads the plugin registry
func (e *AcceptanceTestEnv) LoadPluginRegistry() *claude.PluginRegistry {
	e.t.Helper()

	registry, err := claude.LoadPlugins(e.ClaudeDir)
	if err != nil {
		e.t.Fatal(err)
	}
	return registry
}

// LoadMarketplaceRegistry loads the marketplace registry
func (e *AcceptanceTestEnv) LoadMarketplaceRegistry() claude.MarketplaceRegistry {
	e.t.Helper()

	registry, err := claude.LoadMarketplaces(e.ClaudeDir)
	if err != nil {
		e.t.Fatal(err)
	}
	return registry
}

// LoadClaudeJSON loads claude_desktop_config.json
func (e *AcceptanceTestEnv) LoadClaudeJSON() map[string]interface{} {
	e.t.Helper()

	data, err := os.ReadFile(e.ClaudeJSONPath)
	if err != nil {
		e.t.Fatal(err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		e.t.Fatal(err)
	}

	return config
}

// ProfileExists checks if a profile file exists
func (e *AcceptanceTestEnv) ProfileExists(name string) bool {
	profilePath := filepath.Join(e.ProfilesDir, name+".json")
	_, err := os.Stat(profilePath)
	return err == nil
}

// PluginCount returns the number of installed plugins
func (e *AcceptanceTestEnv) PluginCount() int {
	e.t.Helper()

	registry := e.LoadPluginRegistry()
	return len(registry.Plugins)
}

// MarketplaceCount returns the number of installed marketplaces
func (e *AcceptanceTestEnv) MarketplaceCount() int {
	e.t.Helper()

	registry := e.LoadMarketplaceRegistry()
	return len(registry)
}

// ProjectDir returns a project directory path (creates if needed)
func (e *AcceptanceTestEnv) ProjectDir() string {
	e.t.Helper()

	projectDir := filepath.Join(e.HomeDir, "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		e.t.Fatal(err)
	}
	return projectDir
}

// ProjectDirWithGit creates a project directory with git initialized
func (e *AcceptanceTestEnv) ProjectDirWithGit() string {
	e.t.Helper()

	projectDir := e.ProjectDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		e.t.Fatal(err)
	}

	return projectDir
}

// MCPJSONExists checks if .mcp.json exists in the given directory
func (e *AcceptanceTestEnv) MCPJSONExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".mcp.json"))
	return err == nil
}

// ClaudeupJSONExists checks if .claudeup.json exists in the given directory
func (e *AcceptanceTestEnv) ClaudeupJSONExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".claudeup.json"))
	return err == nil
}

// LoadMCPJSON loads .mcp.json from the given directory
func (e *AcceptanceTestEnv) LoadMCPJSON(dir string) map[string]interface{} {
	e.t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		e.t.Fatal(err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		e.t.Fatal(err)
	}

	return config
}

// LoadClaudeupJSON loads .claudeup.json from the given directory
func (e *AcceptanceTestEnv) LoadClaudeupJSON(dir string) map[string]interface{} {
	e.t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, ".claudeup.json"))
	if err != nil {
		e.t.Fatal(err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		e.t.Fatal(err)
	}

	return config
}

// CreateClaudeupJSON creates a .claudeup.json file in the given directory
func (e *AcceptanceTestEnv) CreateClaudeupJSON(dir string, cfg map[string]interface{}) {
	e.t.Helper()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, ".claudeup.json"), data, 0644); err != nil {
		e.t.Fatal(err)
	}
}

// CreateMCPJSON creates a .mcp.json file in the given directory
func (e *AcceptanceTestEnv) CreateMCPJSON(dir string, cfg map[string]interface{}) {
	e.t.Helper()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0644); err != nil {
		e.t.Fatal(err)
	}
}

// ClaudeupdDir returns the claudeup config directory path
func (e *AcceptanceTestEnv) ClaudeupDir() string {
	return filepath.Join(e.HomeDir, ".claudeup")
}

// CreateProjectsJSON creates projects.json with local scope mappings
func (e *AcceptanceTestEnv) CreateProjectsJSON(projects map[string]interface{}) {
	e.t.Helper()

	claudeupDir := e.ClaudeupDir()
	if err := os.MkdirAll(claudeupDir, 0755); err != nil {
		e.t.Fatal(err)
	}

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(claudeupDir, "projects.json"), data, 0644); err != nil {
		e.t.Fatal(err)
	}
}
