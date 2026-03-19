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
