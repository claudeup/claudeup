// ABOUTME: Integration tests for profile Apply flow
// ABOUTME: Uses mock executor to verify command sequences
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/internal/secrets"
)

// MockExecutor records commands for verification
type MockExecutor struct {
	Commands [][]string
	Errors   map[string]error // command prefix -> error to return
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Commands: [][]string{},
		Errors:   make(map[string]error),
	}
}

func (m *MockExecutor) Run(args ...string) error {
	m.Commands = append(m.Commands, args)

	// Check if we should return an error
	cmdKey := strings.Join(args[:min(3, len(args))], " ")
	if err, ok := m.Errors[cmdKey]; ok {
		return err
	}
	return nil
}

func (m *MockExecutor) CommandCount() int {
	return len(m.Commands)
}

func (m *MockExecutor) HasCommand(prefix ...string) bool {
	for _, cmd := range m.Commands {
		if len(cmd) >= len(prefix) {
			match := true
			for i, p := range prefix {
				if cmd[i] != p {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestApplyInstallsPlugins(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// Current state: no plugins
	// Profile: wants plugin-a
	p := &profile.Profile{
		Name:    "test",
		Plugins: []string{"plugin-a@marketplace"},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have called plugin install
	if !executor.HasCommand("plugin", "install", "plugin-a@marketplace") {
		t.Error("Expected plugin install command")
		t.Logf("Commands: %v", executor.Commands)
	}

	if len(result.PluginsInstalled) != 1 {
		t.Errorf("Expected 1 plugin installed, got %d", len(result.PluginsInstalled))
	}
}

func TestApplyRemovesPlugins(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// Current state: plugin-a and plugin-b installed
	env.createPluginRegistry(map[string]interface{}{
		"plugin-a@marketplace": map[string]interface{}{"version": "1.0"},
		"plugin-b@marketplace": map[string]interface{}{"version": "1.0"},
	})

	// Profile: only wants plugin-a (should remove plugin-b)
	p := &profile.Profile{
		Name:    "test",
		Plugins: []string{"plugin-a@marketplace"},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have called plugin uninstall for plugin-b
	if !executor.HasCommand("plugin", "uninstall", "plugin-b@marketplace") {
		t.Error("Expected plugin uninstall command for plugin-b")
		t.Logf("Commands: %v", executor.Commands)
	}

	if len(result.PluginsRemoved) != 1 {
		t.Errorf("Expected 1 plugin removed, got %d", len(result.PluginsRemoved))
	}
}

func TestApplyAddsMCPServers(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// Profile: wants an MCP server
	p := &profile.Profile{
		Name: "test",
		MCPServers: []profile.MCPServer{
			{
				Name:    "test-mcp",
				Command: "npx",
				Args:    []string{"-y", "test-package"},
				Scope:   "user",
			},
		},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have called mcp add
	if !executor.HasCommand("mcp", "add", "test-mcp") {
		t.Error("Expected mcp add command")
		t.Logf("Commands: %v", executor.Commands)
	}

	if len(result.MCPServersInstalled) != 1 {
		t.Errorf("Expected 1 MCP server installed, got %d", len(result.MCPServersInstalled))
	}
}

func TestApplyRemovesMCPServers(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// Current state: has an MCP server
	env.createClaudeJSON(map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"old-mcp": map[string]interface{}{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	})

	// Profile: no MCP servers (should remove old-mcp)
	p := &profile.Profile{
		Name:       "test",
		MCPServers: []profile.MCPServer{},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have called mcp remove
	if !executor.HasCommand("mcp", "remove", "old-mcp") {
		t.Error("Expected mcp remove command")
		t.Logf("Commands: %v", executor.Commands)
	}

	if len(result.MCPServersRemoved) != 1 {
		t.Errorf("Expected 1 MCP server removed, got %d", len(result.MCPServersRemoved))
	}
}

func TestApplyAddsMarketplaces(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// Profile: wants a marketplace
	p := &profile.Profile{
		Name: "test",
		Marketplaces: []profile.Marketplace{
			{Source: "github", Repo: "test-org/test-marketplace"},
		},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have called marketplace add
	if !executor.HasCommand("plugin", "marketplace", "add") {
		t.Error("Expected marketplace add command")
		t.Logf("Commands: %v", executor.Commands)
	}

	if len(result.MarketplacesAdded) != 1 {
		t.Errorf("Expected 1 marketplace added, got %d", len(result.MarketplacesAdded))
	}
}

func TestApplyWithSecrets(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// Set up an env var for the secret
	os.Setenv("TEST_API_KEY", "secret-value-123")
	defer os.Unsetenv("TEST_API_KEY")

	// Profile: MCP server that needs a secret
	p := &profile.Profile{
		Name: "test",
		MCPServers: []profile.MCPServer{
			{
				Name:    "secret-mcp",
				Command: "npx",
				Args:    []string{"-y", "package", "$TEST_API_KEY"},
				Secrets: map[string]profile.SecretRef{
					"TEST_API_KEY": {
						Description: "Test API key",
						Sources: []profile.SecretSource{
							{Type: "env", Key: "TEST_API_KEY"},
						},
					},
				},
			},
		},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have resolved the secret
	if len(result.MCPServersInstalled) != 1 {
		t.Errorf("Expected 1 MCP server installed, got %d", len(result.MCPServersInstalled))
	}

	// Check that the command includes the resolved secret
	found := false
	for _, cmd := range executor.Commands {
		for _, arg := range cmd {
			if arg == "secret-value-123" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected resolved secret in command args")
		t.Logf("Commands: %v", executor.Commands)
	}
}

func TestApplyMissingSecretFails(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// DON'T set up the env var - secret should fail

	p := &profile.Profile{
		Name: "test",
		MCPServers: []profile.MCPServer{
			{
				Name:    "secret-mcp",
				Command: "npx",
				Args:    []string{"package", "$MISSING_SECRET"},
				Secrets: map[string]profile.SecretRef{
					"MISSING_SECRET": {
						Sources: []profile.SecretSource{
							{Type: "env", Key: "MISSING_SECRET"},
						},
					},
				},
			},
		},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	_, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err == nil {
		t.Error("Expected error for missing secret")
	}
}

func TestApplyCommandOrder(t *testing.T) {
	env := setupApplyTestEnv(t)
	defer env.cleanup()

	// Current state: has plugins to remove
	env.createPluginRegistry(map[string]interface{}{
		"old-plugin@marketplace": map[string]interface{}{"version": "1.0"},
	})

	// Profile: different plugins and marketplaces
	p := &profile.Profile{
		Name: "test",
		Marketplaces: []profile.Marketplace{
			{Source: "github", Repo: "new-marketplace"},
		},
		Plugins: []string{"new-plugin@marketplace"},
	}

	executor := NewMockExecutor()
	chain := secrets.NewChain(secrets.NewEnvResolver())

	_, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify order: removals first, then adds
	// 1. plugin uninstall (remove old)
	// 2. marketplace add (add new)
	// 3. plugin install (add new)

	uninstallIdx := -1
	marketplaceIdx := -1
	installIdx := -1

	for i, cmd := range executor.Commands {
		if len(cmd) >= 2 {
			if cmd[0] == "plugin" && cmd[1] == "uninstall" {
				uninstallIdx = i
			}
			if cmd[0] == "plugin" && cmd[1] == "marketplace" {
				marketplaceIdx = i
			}
			if cmd[0] == "plugin" && cmd[1] == "install" {
				installIdx = i
			}
		}
	}

	if uninstallIdx == -1 || marketplaceIdx == -1 || installIdx == -1 {
		t.Fatalf("Missing expected commands: uninstall=%d, marketplace=%d, install=%d",
			uninstallIdx, marketplaceIdx, installIdx)
	}

	if uninstallIdx > marketplaceIdx {
		t.Error("Expected uninstall before marketplace add")
	}
	if marketplaceIdx > installIdx {
		t.Error("Expected marketplace add before plugin install")
	}
}

// Test environment helpers

type applyTestEnv struct {
	claudeDir  string
	claudeJSON string
	t          *testing.T
}

func setupApplyTestEnv(t *testing.T) *applyTestEnv {
	t.Helper()
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	claudeJSON := filepath.Join(tmpDir, ".claude.json")

	// Create empty registries
	env := &applyTestEnv{
		claudeDir:  claudeDir,
		claudeJSON: claudeJSON,
		t:          t,
	}

	env.createPluginRegistry(map[string]interface{}{})
	env.createMarketplaceRegistry(map[string]interface{}{})
	env.createClaudeJSON(map[string]interface{}{})

	return env
}

func (e *applyTestEnv) cleanup() {
	// t.TempDir() handles cleanup
}

func (e *applyTestEnv) createPluginRegistry(plugins map[string]interface{}) {
	e.t.Helper()
	data := map[string]interface{}{
		"version": 1,
		"plugins": plugins,
	}
	e.writeJSON(filepath.Join(e.claudeDir, "plugins", "installed_plugins.json"), data)
}

func (e *applyTestEnv) createMarketplaceRegistry(marketplaces map[string]interface{}) {
	e.t.Helper()
	e.writeJSON(filepath.Join(e.claudeDir, "plugins", "known_marketplaces.json"), marketplaces)
}

func (e *applyTestEnv) createClaudeJSON(data map[string]interface{}) {
	e.t.Helper()
	e.writeJSON(e.claudeJSON, data)
}

func (e *applyTestEnv) writeJSON(path string, data interface{}) {
	e.t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		e.t.Fatal(err)
	}
}
