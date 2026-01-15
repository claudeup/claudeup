// ABOUTME: Unit tests for plugin show shell completion
// ABOUTME: Tests tab completion for plugin@marketplace format
package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestPluginShowCompletion_NoInput(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	originalClaudeDir := claudeDir
	claudeDir = filepath.Join(tempDir, ".claude")
	defer func() { claudeDir = originalClaudeDir }()

	os.MkdirAll(filepath.Join(claudeDir, "plugins", "marketplaces"), 0755)

	// Create a mock marketplace
	marketplacePath := filepath.Join(claudeDir, "plugins", "marketplaces", "test-market")
	os.MkdirAll(filepath.Join(marketplacePath, "plugins", "plugin-a"), 0755)
	os.MkdirAll(filepath.Join(marketplacePath, "plugins", "plugin-b"), 0755)

	// Create known_marketplaces.json
	knownPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	os.WriteFile(knownPath, []byte(`{
		"test-market": {
			"source": {"source": "github", "repo": "test/market"},
			"installLocation": "`+marketplacePath+`"
		}
	}`), 0644)

	// Test completion with empty input
	completions, directive := pluginShowCompletionFunc(nil, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	// Should suggest marketplace names
	found := false
	for _, c := range completions {
		if c == "test-market" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected completion to include 'test-market', got %v", completions)
	}
}

func TestPluginShowCompletion_WithMarketplace(t *testing.T) {
	tempDir := t.TempDir()
	originalClaudeDir := claudeDir
	claudeDir = filepath.Join(tempDir, ".claude")
	defer func() { claudeDir = originalClaudeDir }()

	marketplacePath := filepath.Join(claudeDir, "plugins", "marketplaces", "test-market")
	os.MkdirAll(filepath.Join(marketplacePath, "plugins", "plugin-a"), 0755)
	os.MkdirAll(filepath.Join(marketplacePath, "plugins", "plugin-b"), 0755)

	knownPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	os.WriteFile(knownPath, []byte(`{
		"test-market": {
			"source": {"source": "github", "repo": "test/market"},
			"installLocation": "`+marketplacePath+`"
		}
	}`), 0644)

	// Test completion with marketplace@ prefix
	completions, directive := pluginShowCompletionFunc(nil, []string{}, "test-market@")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	// Should suggest plugin@marketplace combinations
	foundA := false
	foundB := false
	for _, c := range completions {
		if c == "plugin-a@test-market" {
			foundA = true
		}
		if c == "plugin-b@test-market" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Errorf("expected completions to include plugin-a@test-market and plugin-b@test-market, got %v", completions)
	}
}

func TestPluginShowCompletion_AlreadyHasArg(t *testing.T) {
	// When an argument is already provided, don't suggest more
	completions, directive := pluginShowCompletionFunc(nil, []string{"plugin@market"}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	if len(completions) != 0 {
		t.Errorf("expected no completions when arg already provided, got %v", completions)
	}
}

func TestPluginShowCompletion_NoMarketplaces(t *testing.T) {
	// When no marketplaces exist, return empty completions
	tempDir := t.TempDir()
	originalClaudeDir := claudeDir
	claudeDir = filepath.Join(tempDir, ".claude")
	defer func() { claudeDir = originalClaudeDir }()

	// Create plugins dir but no known_marketplaces.json
	os.MkdirAll(filepath.Join(claudeDir, "plugins"), 0755)

	completions, directive := pluginShowCompletionFunc(nil, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	if len(completions) != 0 {
		t.Errorf("expected no completions when no marketplaces exist, got %v", completions)
	}
}

func TestPluginShowCompletion_PartialMarketplaceName(t *testing.T) {
	// Test completion filtering by partial marketplace name
	tempDir := t.TempDir()
	originalClaudeDir := claudeDir
	claudeDir = filepath.Join(tempDir, ".claude")
	defer func() { claudeDir = originalClaudeDir }()

	marketplacePath1 := filepath.Join(claudeDir, "plugins", "marketplaces", "test-market")
	marketplacePath2 := filepath.Join(claudeDir, "plugins", "marketplaces", "other-market")
	os.MkdirAll(filepath.Join(marketplacePath1, "plugins", "plugin-a"), 0755)
	os.MkdirAll(filepath.Join(marketplacePath2, "plugins", "plugin-b"), 0755)

	knownPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	os.WriteFile(knownPath, []byte(`{
		"test-market": {
			"source": {"source": "github", "repo": "test/market"},
			"installLocation": "`+marketplacePath1+`"
		},
		"other-market": {
			"source": {"source": "github", "repo": "other/market"},
			"installLocation": "`+marketplacePath2+`"
		}
	}`), 0644)

	// Test completion with partial marketplace name (no @)
	completions, directive := pluginShowCompletionFunc(nil, []string{}, "test")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	// Should return all marketplaces, shell filters them
	if len(completions) != 2 {
		t.Errorf("expected 2 completions (shell does filtering), got %v", completions)
	}
}
