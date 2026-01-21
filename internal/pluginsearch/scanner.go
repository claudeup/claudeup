// ABOUTME: Scans plugin cache directory to build search index
// ABOUTME: Parses plugin.json and component directories

package pluginsearch

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Scanner scans the plugin cache to build a search index.
type Scanner struct{}

// NewScanner creates a new Scanner.
func NewScanner() *Scanner {
	return &Scanner{}
}

// pluginJSON represents the structure of .claude-plugin/plugin.json
type pluginJSON struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Keywords    []string `json:"keywords"`
}

// Scan walks the cache directory and builds an index of all plugins.
func (s *Scanner) Scan(cacheDir string) ([]PluginSearchIndex, error) {
	var plugins []PluginSearchIndex

	// Walk marketplace directories
	marketplaces, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	for _, marketplace := range marketplaces {
		if !marketplace.IsDir() {
			continue
		}
		marketplacePath := filepath.Join(cacheDir, marketplace.Name())

		// Walk plugin directories within marketplace
		pluginDirs, err := os.ReadDir(marketplacePath)
		if err != nil {
			continue
		}

		for _, pluginDir := range pluginDirs {
			if !pluginDir.IsDir() {
				continue
			}
			pluginPath := filepath.Join(marketplacePath, pluginDir.Name())

			// Walk version directories within plugin
			versions, err := os.ReadDir(pluginPath)
			if err != nil {
				continue
			}

			for _, version := range versions {
				if !version.IsDir() {
					continue
				}
				versionPath := filepath.Join(pluginPath, version.Name())

				// Parse plugin.json
				plugin, err := s.parsePlugin(versionPath, marketplace.Name())
				if err != nil {
					continue // Skip malformed plugins
				}

				plugins = append(plugins, *plugin)
			}
		}
	}

	return plugins, nil
}

func (s *Scanner) parsePlugin(pluginPath, marketplace string) (*PluginSearchIndex, error) {
	jsonPath := filepath.Join(pluginPath, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	var pj pluginJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return nil, err
	}

	return &PluginSearchIndex{
		Name:        pj.Name,
		Description: pj.Description,
		Version:     pj.Version,
		Keywords:    pj.Keywords,
		Marketplace: marketplace,
		Path:        pluginPath,
	}, nil
}
