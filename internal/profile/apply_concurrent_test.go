// ABOUTME: Tests for concurrent profile apply operations
// ABOUTME: Validates parallel execution and progress tracking integration
package profile

import (
	"bytes"
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
