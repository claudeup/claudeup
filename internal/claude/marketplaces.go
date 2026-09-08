// ABOUTME: Data structures and functions for managing Claude Code marketplaces
// ABOUTME: Handles reading known_marketplaces.json
package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// MarketplaceRegistry represents the known_marketplaces.json file structure
type MarketplaceRegistry map[string]MarketplaceMetadata

// MarketplaceMetadata represents metadata for an installed marketplace
type MarketplaceMetadata struct {
	Source          MarketplaceSource `json:"source"`
	InstallLocation string            `json:"installLocation"`
	LastUpdated     string            `json:"lastUpdated"`
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

// PluginSource represents a plugin's source location.
// In marketplace.json, source is either a string holding a path relative to the
// marketplace root, or an object whose "source" field names an external source
// type alongside type-specific location fields. The recognized types are npm,
// url, github and git-subdir, per
// https://www.schemastore.org/claude-code-marketplace.json
//
// Only the discriminator and url are retained; the location fields belonging to
// the other forms are not read, because external plugins are fetched by Claude
// Code rather than by claudeup. The type is decode-only and has no MarshalJSON,
// so it does not round-trip.
type PluginSource struct {
	// RelativePath holds the path inside the marketplace, set only for the
	// string form. It is named RelativePath rather than Path because the
	// git-subdir form has its own required "path" field, which means a
	// subdirectory of a remote repository.
	RelativePath string `json:"-"`

	// Kind names the external source type, set only for the object form.
	Kind string `json:"source,omitempty"`

	// URL is the location field of the url and git-subdir forms. The github and
	// npm forms carry repo and package instead, so an empty URL does not mean
	// the source is local.
	URL string `json:"url,omitempty"`
}

// UnmarshalJSON handles source being either a string or an object
func (s *PluginSource) UnmarshalJSON(data []byte) error {
	// Clear the receiver so a value decoded more than once cannot carry a field
	// from one source form into another.
	*s = PluginSource{}

	// Try string first (relative path like "./plugins/hookify")
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.RelativePath = str
		return nil
	}

	// Try object (like {"source": "url", "url": "https://github.com/org/repo.git"}).
	// Decoding into a method-less copy of the type keeps the field list in one
	// place, so a field added above is picked up here rather than staying empty.
	// RelativePath is excluded by its json tag, so the string form stays distinct.
	type plain PluginSource
	return json.Unmarshal(data, (*plain)(s))
}

// IsRelativePath returns true if the source names a marketplace-relative path.
// Only the string form does. Object forms are treated as external, including
// source types this build does not recognize. Whether the path stays inside the
// marketplace is the caller's check, not this one.
func (s *PluginSource) IsRelativePath() bool {
	return s.RelativePath != ""
}

// IsURL returns true if the source carries a git URL, which only the url and
// git-subdir forms do. It is not a test for whether the source is external:
// the github and npm forms are external and carry no URL.
func (s *PluginSource) IsURL() bool {
	return s.URL != ""
}

// MarketplacePluginInfo represents a plugin entry in the marketplace index
type MarketplacePluginInfo struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Version     string        `json:"version,omitempty"`
	Source      *PluginSource `json:"source,omitempty"`
}

// LoadMarketplaces reads and parses the known_marketplaces.json file.
// Any fs.ErrNotExist along the full path -- including a missing claudeDir,
// plugins directory, or marketplaces file -- is treated as a fresh install
// and returns an empty registry with nil error.
func LoadMarketplaces(claudeDir string) (MarketplaceRegistry, error) {
	marketplacesPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	data, err := os.ReadFile(marketplacesPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No plugins directory or no marketplaces file yet - treat as fresh install
			return make(MarketplaceRegistry), nil
		}
		return nil, fmt.Errorf("cannot read marketplaces from %s: %w", marketplacesPath, err)
	}

	var registry MarketplaceRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse marketplaces JSON from %s: %w", marketplacesPath, err)
	}

	return registry, nil
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
