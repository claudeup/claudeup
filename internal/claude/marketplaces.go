// ABOUTME: Data structures and functions for managing Claude Code marketplaces
// ABOUTME: Handles reading and writing known_marketplaces.json
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// LoadMarketplaces reads and parses the known_marketplaces.json file
func LoadMarketplaces(claudeDir string) (MarketplaceRegistry, error) {
	marketplacesPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	data, err := os.ReadFile(marketplacesPath)
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

	return os.WriteFile(marketplacesPath, data, 0644)
}
