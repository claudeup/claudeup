// ABOUTME: Unit tests for marketplace registry management
// ABOUTME: Tests loading, saving, and marketplace operations
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSaveMarketplaces(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugins directory
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test registry
	registry := MarketplaceRegistry{
		"test-marketplace": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "test/repo",
			},
			InstallLocation: "/test/location",
			LastUpdated:     "2024-01-01T00:00:00Z",
		},
	}

	// Save registry
	if err := SaveMarketplaces(tempDir, registry); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	marketplacesFile := filepath.Join(tempDir, "plugins", "known_marketplaces.json")
	if _, err := os.Stat(marketplacesFile); os.IsNotExist(err) {
		t.Error("known_marketplaces.json should exist after save")
	}

	// Load registry
	loaded, err := LoadMarketplaces(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify loaded data
	marketplace, exists := loaded["test-marketplace"]
	if !exists {
		t.Error("Marketplace should exist in loaded registry")
	}

	if marketplace.Source.Source != "github" {
		t.Errorf("Expected source github, got %s", marketplace.Source.Source)
	}

	if marketplace.Source.Repo != "test/repo" {
		t.Errorf("Expected repo test/repo, got %s", marketplace.Source.Repo)
	}

	if marketplace.InstallLocation != "/test/location" {
		t.Errorf("Expected location /test/location, got %s", marketplace.InstallLocation)
	}
}

func TestLoadMarketplacesNonExistent(t *testing.T) {
	// Try to load from non-existent directory
	_, err := LoadMarketplaces("/non/existent/path")
	if err == nil {
		t.Error("LoadMarketplaces should return error for non-existent path")
	}
}

func TestSaveMarketplacesInvalidPath(t *testing.T) {
	registry := MarketplaceRegistry{
		"test": MarketplaceMetadata{},
	}

	// Try to save to invalid path
	err := SaveMarketplaces("/invalid/path/that/does/not/exist", registry)
	if err == nil {
		t.Error("SaveMarketplaces should return error for invalid path")
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

func TestEmptyMarketplaceRegistry(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugins directory
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Save empty registry
	registry := MarketplaceRegistry{}
	if err := SaveMarketplaces(tempDir, registry); err != nil {
		t.Fatal(err)
	}

	// Load it back
	loaded, err := LoadMarketplaces(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 0 {
		t.Errorf("Expected empty registry, got %d entries", len(loaded))
	}
}
