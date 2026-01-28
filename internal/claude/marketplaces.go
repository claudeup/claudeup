// ABOUTME: Data structures and functions for managing Claude Code marketplaces
// ABOUTME: Handles reading and writing known_marketplaces.json
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v3/internal/events"
)

// MarketplaceRegistry represents the known_marketplaces.json file structure
type MarketplaceRegistry map[string]MarketplaceMetadata

// MarketplaceMetadata represents metadata for an installed marketplace
type MarketplaceMetadata struct {
	Source           MarketplaceSource `json:"source"`
	InstallLocation  string            `json:"installLocation"`
	LastUpdated      string            `json:"lastUpdated"`
}

// MarketplaceSource represents the source of a marketplace
type MarketplaceSource struct {
	Source string `json:"source"`
	Repo   string `json:"repo,omitempty"`
	URL    string `json:"url,omitempty"`
}

// MarketplaceIndex represents the .claude-plugin/marketplace.json file
type MarketplaceIndex struct {
	Name    string                  `json:"name"`
	Plugins []MarketplacePluginInfo `json:"plugins"`
}

// MarketplacePluginInfo represents a plugin entry in the marketplace index
type MarketplacePluginInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// LoadMarketplaces reads and parses the known_marketplaces.json file
func LoadMarketplaces(claudeDir string) (MarketplaceRegistry, error) {
	// Check if plugins directory exists
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		return nil, err
	}

	marketplacesPath := filepath.Join(pluginsDir, "known_marketplaces.json")

	data, err := os.ReadFile(marketplacesPath)
	if os.IsNotExist(err) {
		// Fresh Claude install - no marketplaces added yet
		return make(MarketplaceRegistry), nil
	}
	if err != nil {
		return nil, err
	}

	var registry MarketplaceRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}

	return registry, nil
}

// SaveMarketplaces writes the marketplace registry back to known_marketplaces.json
func SaveMarketplaces(claudeDir string, registry MarketplaceRegistry) error {
	marketplacesPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"marketplace update",
		marketplacesPath,
		"user",
		func() error {
			return os.WriteFile(marketplacesPath, data, 0644)
		},
	)
}

// MarketplaceExists checks if a marketplace with the given repo or URL is installed
func (r MarketplaceRegistry) MarketplaceExists(repoOrURL string) bool {
	for _, meta := range r {
		if meta.Source.Repo == repoOrURL || meta.Source.URL == repoOrURL {
			return true
		}
	}
	return false
}

// GetMarketplaceByRepo returns the marketplace name for a given repo, or empty string if not found
func (r MarketplaceRegistry) GetMarketplaceByRepo(repoOrURL string) string {
	for name, meta := range r {
		if meta.Source.Repo == repoOrURL || meta.Source.URL == repoOrURL {
			return name
		}
	}
	return ""
}

// LoadMarketplaceIndex reads the .claude-plugin/marketplace.json from a marketplace
func LoadMarketplaceIndex(installLocation string) (*MarketplaceIndex, error) {
	// Validate path is absolute to prevent path traversal attacks
	if !filepath.IsAbs(installLocation) {
		return nil, fmt.Errorf("install location must be absolute path")
	}
	cleanPath := filepath.Clean(installLocation)

	indexPath := filepath.Join(cleanPath, ".claude-plugin", "marketplace.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read marketplace index: %w", err)
	}

	var index MarketplaceIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse marketplace index: %w", err)
	}

	// Validate required fields
	if index.Name == "" {
		return nil, fmt.Errorf("marketplace index missing required 'name' field")
	}

	return &index, nil
}

// FindMarketplace finds a marketplace by name, repo, or URL
// Returns the marketplace metadata, its key in the registry, and any error
func FindMarketplace(claudeDir string, identifier string) (*MarketplaceMetadata, string, error) {
	registry, err := LoadMarketplaces(claudeDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load marketplaces: %w", err)
	}

	// First, check by key (marketplace name in registry)
	if meta, exists := registry[identifier]; exists {
		return &meta, identifier, nil
	}

	// Check by repo or URL
	for name, meta := range registry {
		if meta.Source.Repo == identifier || meta.Source.URL == identifier {
			return &meta, name, nil
		}
	}

	// Check by marketplace name from index files
	for name, meta := range registry {
		index, err := LoadMarketplaceIndex(meta.InstallLocation)
		if err != nil {
			continue
		}
		if index.Name == identifier {
			return &meta, name, nil
		}
	}

	return nil, "", fmt.Errorf("marketplace %q not found", identifier)
}
