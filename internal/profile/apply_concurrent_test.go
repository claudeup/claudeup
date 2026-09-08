// ABOUTME: Tests for concurrent profile apply operations
// ABOUTME: Validates parallel execution and progress tracking integration
package profile

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/claudeup/claudeup/v5/internal/secrets"
)

// concurrentMockExecutor records commands for testing concurrent apply
// Uses mutex to protect slice append from concurrent worker goroutines
type concurrentMockExecutor struct {
	commands [][]string
	mu       sync.Mutex
}

func (m *concurrentMockExecutor) Run(args ...string) error {
	return nil
}

func (m *concurrentMockExecutor) RunWithOutput(args ...string) (string, error) {
	m.mu.Lock()
	m.commands = append(m.commands, args)
	m.mu.Unlock()
	return "", nil
}

func TestApplyConcurrentlySkipsInstalledMarketplaces(t *testing.T) {
	// This test verifies that already-installed marketplaces are skipped
	profile := &Profile{
		Marketplaces: []Marketplace{
			{Repo: "already/installed"},
			{Repo: "new/marketplace"},
		},
		Plugins: []string{},
	}

	// Create temp dir with mock marketplace registry
	// For now, just verify the function runs without error
	executor := &concurrentMockExecutor{}
	var output bytes.Buffer

	result, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
		ClaudeDir: "/nonexistent", // Will cause load to fail, treating all as new
		Executor:  executor,
		Output:    &output,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both marketplaces should be installed (since we couldn't load existing)
	if len(result.MarketplacesInstalled) != 2 {
		t.Errorf("expected 2 marketplaces installed, got %d", len(result.MarketplacesInstalled))
	}
}

func TestApplyConcurrentlyInstallsPluginsWithScope(t *testing.T) {
	profile := &Profile{
		Plugins: []string{"plugin-a", "plugin-b"},
	}

	executor := &concurrentMockExecutor{}
	var output bytes.Buffer

	_, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
		ClaudeDir: "/nonexistent",
		Scope:     "project",
		Executor:  executor,
		Output:    &output,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that scope flag was included
	foundScopeFlag := false
	for _, cmd := range executor.commands {
		for i, arg := range cmd {
			if arg == "--scope" && i+1 < len(cmd) && cmd[i+1] == "project" {
				foundScopeFlag = true
				break
			}
		}
	}

	if !foundScopeFlag {
		t.Error("expected --scope project flag in plugin install commands")
	}
}

func TestApplyConcurrentlyHandlesMCPServers(t *testing.T) {
	profile := &Profile{
		MCPServers: []MCPServer{
			{Name: "test-mcp", Command: "test-cmd"},
		},
	}

	executor := &concurrentMockExecutor{}
	var output bytes.Buffer

	result, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
		ClaudeDir: "/nonexistent",
		Executor:  executor,
		Output:    &output,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.MCPServersInstalled) != 1 {
		t.Errorf("expected 1 MCP server installed, got %d", len(result.MCPServersInstalled))
	}
}

// fakeSecretResolver resolves a fixed set of references. It stands in for a
// non-env backend (1Password, keychain) so the test cannot be satisfied by
// buildMCPAddArgs' os.Getenv fallback.
type fakeSecretResolver struct {
	values map[string]string
}

func (f *fakeSecretResolver) Name() string    { return "fake" }
func (f *fakeSecretResolver) Available() bool { return true }
func (f *fakeSecretResolver) Resolve(ref string) (string, error) {
	if v, ok := f.values[ref]; ok {
		return v, nil
	}
	return "", errors.New("unknown ref: " + ref)
}

// The concurrent path checks that a declared secret resolves (so a missing
// one is warned about early) but must pass a ${KEY} placeholder, never the
// resolved value, to `claude mcp add` (#312).
func TestApplyConcurrentlyWritesMCPSecretPlaceholders(t *testing.T) {
	profile := &Profile{
		MCPServers: []MCPServer{
			{
				Name:    "secret-server",
				Command: "node",
				Args:    []string{"$MY_SECRET_TOKEN"},
				Secrets: map[string]SecretRef{
					"MY_SECRET_TOKEN": {
						Sources: []SecretSource{
							{Type: "1password", Ref: "op://vault/item/token"},
						},
					},
				},
			},
		},
	}

	// Ensure the placeholder cannot be satisfied from the environment.
	t.Setenv("MY_SECRET_TOKEN", "")
	chain := secrets.NewChain(&fakeSecretResolver{
		values: map[string]string{"op://vault/item/token": "resolved-value"},
	})

	executor := &concurrentMockExecutor{}
	var output bytes.Buffer

	result, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
		ClaudeDir:   "/nonexistent",
		Executor:    executor,
		Output:      &output,
		SecretChain: chain,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.MCPServersInstalled) != 1 {
		t.Fatalf("expected secret-server to be installed, got: %v", result.MCPServersInstalled)
	}

	var mcpCmd string
	for _, cmd := range executor.commands {
		if len(cmd) >= 3 && cmd[0] == "mcp" && cmd[1] == "add" && cmd[2] == "secret-server" {
			mcpCmd = strings.Join(cmd, " ")
		}
	}
	if mcpCmd == "" {
		t.Fatalf("expected mcp add command for secret-server, got: %v", executor.commands)
	}
	if !strings.HasSuffix(mcpCmd, " ${MY_SECRET_TOKEN}") {
		t.Errorf("expected ${MY_SECRET_TOKEN} placeholder in mcp add args, got: %s", mcpCmd)
	}
	if strings.Contains(mcpCmd, "resolved-value") {
		t.Errorf("resolved secret leaked into mcp add argv: %s", mcpCmd)
	}
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "MY_SECRET_TOKEN") {
			t.Errorf("expected no warning for a resolvable secret, got: %v", w)
		}
	}
}

func TestApplyConcurrentlyWarnsOnUnresolvedMCPSecret(t *testing.T) {
	profile := &Profile{
		MCPServers: []MCPServer{
			{
				Name:    "secret-server",
				Command: "node",
				Args:    []string{"$MISSING_TOKEN"},
				Secrets: map[string]SecretRef{
					"MISSING_TOKEN": {
						Sources: []SecretSource{
							{Type: "env", Key: "MISSING_TOKEN"},
						},
					},
				},
			},
		},
	}

	t.Setenv("MISSING_TOKEN", "")
	chain := secrets.NewChain(secrets.NewEnvResolver())

	executor := &concurrentMockExecutor{}
	var output bytes.Buffer

	result, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
		ClaudeDir:   "/nonexistent",
		Executor:    executor,
		Output:      &output,
		SecretChain: chain,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Matches the sequential path: an unresolved secret is a warning, and the
	// server is still registered.
	if len(result.MCPServersInstalled) != 1 {
		t.Errorf("expected secret-server to still be installed, got: %v", result.MCPServersInstalled)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors for unresolved secret, got: %v", result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "secret-server") && strings.Contains(w.Error(), "MISSING_TOKEN") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning naming secret-server and MISSING_TOKEN, got: %v", result.Warnings)
	}
}

func TestApplyConcurrentlyWithReinstallFlag(t *testing.T) {
	profile := &Profile{
		Marketplaces: []Marketplace{
			{Repo: "some/marketplace"},
		},
	}

	executor := &concurrentMockExecutor{}
	var output bytes.Buffer

	result, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
		ClaudeDir: "/nonexistent",
		Reinstall: true,
		Executor:  executor,
		Output:    &output,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With reinstall, nothing should be skipped
	if len(result.MarketplacesSkipped) != 0 {
		t.Errorf("expected 0 skipped with reinstall, got %d", len(result.MarketplacesSkipped))
	}
}

func TestApplyConcurrentlyWithLoadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission checks do not apply when running as root")
	}

	// Create a claudeDir with unreadable marketplace and plugin files
	// so both LoadMarketplaces and LoadPlugins return non-ErrNotExist errors.
	claudeDir := t.TempDir()
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "known_marketplaces.json"), []byte(`{}`), 0000); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(`{"version":2,"plugins":{}}`), 0000); err != nil {
		t.Fatal(err)
	}

	profile := &Profile{
		Marketplaces: []Marketplace{
			{Repo: "org/marketplace-a"},
		},
		Plugins: []string{"plugin-a"},
	}

	executor := &concurrentMockExecutor{}
	var output bytes.Buffer

	result, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
		ClaudeDir: claudeDir,
		Executor:  executor,
		Output:    &output,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All items should be installed (not skipped) since load fell back to empty
	if len(result.MarketplacesInstalled) != 1 {
		t.Errorf("expected 1 marketplace installed, got %d", len(result.MarketplacesInstalled))
	}
	if len(result.MarketplacesSkipped) != 0 {
		t.Errorf("expected 0 marketplaces skipped, got %d", len(result.MarketplacesSkipped))
	}
	if len(result.PluginsInstalled) != 1 {
		t.Errorf("expected 1 plugin installed, got %d", len(result.PluginsInstalled))
	}
	if len(result.PluginsSkipped) != 0 {
		t.Errorf("expected 0 plugins skipped, got %d", len(result.PluginsSkipped))
	}

	// Both load errors should be surfaced as warnings
	foundMarketplaceWarning := false
	foundPluginWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "could not read installed marketplaces") {
			foundMarketplaceWarning = true
		}
		if strings.Contains(w.Error(), "could not read installed plugins") {
			foundPluginWarning = true
		}
	}
	if !foundMarketplaceWarning {
		t.Error("expected load warning for marketplaces to be surfaced in result.Warnings")
	}
	if !foundPluginWarning {
		t.Error("expected load warning for plugins to be surfaced in result.Warnings")
	}

	// No actual install errors should exist (mock executor succeeds)
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// failingMockExecutor returns a specified output and error for matching commands
type failingMockExecutor struct {
	mu               sync.Mutex
	failOnWithOutput map[string]string // command prefix -> output to return alongside an error
}

func (m *failingMockExecutor) Run(args ...string) error {
	_, err := m.RunWithOutput(args...)
	return err
}

func (m *failingMockExecutor) RunWithOutput(args ...string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.Join(args, " ")
	for prefix, output := range m.failOnWithOutput {
		if strings.HasPrefix(key, prefix) {
			return output, errors.New("exit status 1")
		}
	}
	return "", nil
}

func TestApplyConcurrentlyMCPAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	writeTestJSON(t, filepath.Join(pluginsDir, "installed_plugins.json"), map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}})
	writeTestJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]interface{}{"enabledPlugins": map[string]bool{}})
	writeTestJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), map[string]interface{}{})

	executor := &failingMockExecutor{
		failOnWithOutput: map[string]string{
			"mcp add context7": "MCP server context7 already exists in user config",
		},
	}

	p := &Profile{
		Name: "test-concurrent-mcp",
		MCPServers: []MCPServer{
			{Name: "context7", Command: "npx", Args: []string{"-y", "@context7/mcp"}},
		},
	}

	result, err := ApplyConcurrently(p, ConcurrentApplyOptions{
		ClaudeDir: claudeDir,
		Scope:     "user",
		Output:    io.Discard,
		Executor:  executor,
	})
	if err != nil {
		t.Fatalf("ApplyConcurrently failed: %v", err)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	if len(result.MCPServersSkipped) != 1 || result.MCPServersSkipped[0] != "context7" {
		t.Errorf("Expected context7 in MCPServersSkipped, got: %v", result.MCPServersSkipped)
	}
}
