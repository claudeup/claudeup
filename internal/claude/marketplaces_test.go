// ABOUTME: Unit tests for marketplace registry management
// ABOUTME: Tests loading and marketplace operations
package claude

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMarketplacesNonExistent(t *testing.T) {
	// A non-existent plugins directory is treated as a fresh install (no marketplaces yet).
	// The stat check is intentionally absent; ReadFile handles the missing path.
	registry, err := LoadMarketplaces("/non/existent/path")
	if err != nil {
		t.Errorf("LoadMarketplaces should return empty registry for non-existent path, got error: %v", err)
	}
	if registry == nil {
		t.Error("Registry should be initialized, not nil")
	}
	if len(registry) != 0 {
		t.Errorf("Expected 0 marketplaces for non-existent path, got %d", len(registry))
	}
}

func TestLoadMarketplacesFreshInstall(t *testing.T) {
	// Create temp directory with plugins subdirectory but no known_marketplaces.json
	// This simulates a fresh Claude Code install that hasn't added any marketplaces yet
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugins directory (Claude creates this on install)
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Load marketplaces from fresh install (no known_marketplaces.json yet)
	registry, err := LoadMarketplaces(tempDir)
	if err != nil {
		t.Fatalf("LoadMarketplaces should not error on fresh install, got: %v", err)
	}

	// Should return empty registry
	if registry == nil {
		t.Error("Registry should be initialized, not nil")
	}

	if len(registry) != 0 {
		t.Errorf("Expected 0 marketplaces in fresh install, got %d", len(registry))
	}
}

func TestLoadMarketplacesPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission checks do not apply when running as root")
	}
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create an unreadable known_marketplaces.json
	marketplacesPath := filepath.Join(pluginsDir, "known_marketplaces.json")
	if err := os.WriteFile(marketplacesPath, []byte(`{}`), 0000); err != nil {
		t.Fatal(err)
	}

	_, err := LoadMarketplaces(tempDir)
	if err == nil {
		t.Fatal("LoadMarketplaces should return error for unreadable file")
	}
	if !strings.Contains(err.Error(), "cannot read marketplaces from") {
		t.Errorf("expected context-wrapped error, got: %v", err)
	}
	// Should NOT be a not-found error -- it's a permission error
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected permission error, not ErrNotExist, got: %v", err)
	}
}

func TestLoadMarketplacesCorruptJSON(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}
	marketplacesPath := filepath.Join(pluginsDir, "known_marketplaces.json")
	if err := os.WriteFile(marketplacesPath, []byte(`{not valid json`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadMarketplaces(tempDir)
	if err == nil {
		t.Fatal("LoadMarketplaces should return error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse marketplaces JSON from") {
		t.Errorf("expected parse error with path context, got: %v", err)
	}
}

func TestMarketplaceRegistryJSONMarshaling(t *testing.T) {
	registry := MarketplaceRegistry{
		"marketplace-1": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "org/repo1",
			},
			InstallLocation: "/path/1",
			LastUpdated:     "2024-01-01T00:00:00Z",
		},
		"marketplace-2": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "git",
				Repo:   "org/repo2",
			},
			InstallLocation: "/path/2",
			LastUpdated:     "2024-01-02T00:00:00Z",
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Unmarshal from JSON
	var loaded MarketplaceRegistry
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	// Verify data integrity
	if len(loaded) != len(registry) {
		t.Error("Marketplace count mismatch after JSON round-trip")
	}

	m1 := loaded["marketplace-1"]
	if m1.Source.Repo != "org/repo1" {
		t.Error("Marketplace-1 repo mismatch after JSON round-trip")
	}

	m2 := loaded["marketplace-2"]
	if m2.Source.Repo != "org/repo2" {
		t.Error("Marketplace-2 repo mismatch after JSON round-trip")
	}
}

func TestMarketplaceExists(t *testing.T) {
	registry := MarketplaceRegistry{
		"claude-mem": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "thedotmack/claude-mem",
			},
		},
		"superpowers": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "git",
				URL:    "https://github.com/obra/superpowers-marketplace.git",
			},
		},
	}

	// Test exists by repo
	if !registry.MarketplaceExists("thedotmack/claude-mem") {
		t.Error("Should find marketplace by repo")
	}

	// Test exists by URL
	if !registry.MarketplaceExists("https://github.com/obra/superpowers-marketplace.git") {
		t.Error("Should find marketplace by URL")
	}

	// Test not found
	if registry.MarketplaceExists("nonexistent/repo") {
		t.Error("Should not find nonexistent marketplace")
	}
}

func TestGetMarketplaceByRepo(t *testing.T) {
	registry := MarketplaceRegistry{
		"claude-mem": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "thedotmack/claude-mem",
			},
		},
	}

	// Test found
	name := registry.GetMarketplaceByRepo("thedotmack/claude-mem")
	if name != "claude-mem" {
		t.Errorf("Expected 'claude-mem', got '%s'", name)
	}

	// Test not found
	name = registry.GetMarketplaceByRepo("nonexistent/repo")
	if name != "" {
		t.Errorf("Expected empty string for not found, got '%s'", name)
	}
}
