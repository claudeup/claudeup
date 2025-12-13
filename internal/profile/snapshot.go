// ABOUTME: Creates a profile from current Claude Code state
// ABOUTME: Reads installed plugins, marketplaces, and MCP servers
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
)

// ClaudeJSON represents the ~/.claude.json file structure (relevant parts)
type ClaudeJSON struct {
	MCPServers map[string]ClaudeMCPServer `json:"mcpServers"`
}

// ClaudeMCPServer represents an MCP server in ~/.claude.json
type ClaudeMCPServer struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// MarketplaceRegistry represents known_marketplaces.json
type MarketplaceRegistry map[string]MarketplaceMetadata

// MarketplaceMetadata represents metadata for a marketplace
type MarketplaceMetadata struct {
	Source MarketplaceSource `json:"source"`
}

// MarketplaceSource represents the source of a marketplace
type MarketplaceSource struct {
	Source string `json:"source"`
	Repo   string `json:"repo,omitempty"`
	URL    string `json:"url,omitempty"`
}

// Snapshot creates a Profile from the current Claude Code state
func Snapshot(name, claudeDir, claudeJSONPath string) (*Profile, error) {
	p := &Profile{
		Name:        name,
		Description: "Snapshot of current Claude Code configuration",
	}

	// Read plugins
	plugins, err := readPlugins(claudeDir)
	if err == nil {
		p.Plugins = plugins
	}

	// Read marketplaces
	marketplaces, err := readMarketplaces(claudeDir)
	if err == nil {
		p.Marketplaces = marketplaces
	}

	// Read MCP servers
	mcpServers, err := readMCPServers(claudeJSONPath)
	if err == nil {
		p.MCPServers = mcpServers
	}

	return p, nil
}

func readPlugins(claudeDir string) ([]string, error) {
	// Use claude.LoadPlugins to get V1→V2 migration for free
	registry, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return nil, err
	}

	// Extract plugin names using GetAllPlugins (returns user-scoped plugins)
	allPlugins := registry.GetAllPlugins()
	plugins := make([]string, 0, len(allPlugins))
	for name := range allPlugins {
		plugins = append(plugins, name)
	}
	sort.Strings(plugins)

	return plugins, nil
}

func readMarketplaces(claudeDir string) ([]Marketplace, error) {
	marketplacesPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	data, err := os.ReadFile(marketplacesPath)
	if err != nil {
		return nil, err
	}

	var registry MarketplaceRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}

	var marketplaces []Marketplace
	for _, meta := range registry {
		marketplaces = append(marketplaces, Marketplace{
			Source: meta.Source.Source,
			Repo:   meta.Source.Repo,
			URL:    meta.Source.URL,
		})
	}

	// Sort by repo (or URL for git sources) for consistent output
	sort.Slice(marketplaces, func(i, j int) bool {
		keyI := marketplaces[i].Repo
		if keyI == "" {
			keyI = marketplaces[i].URL
		}
		keyJ := marketplaces[j].Repo
		if keyJ == "" {
			keyJ = marketplaces[j].URL
		}
		return keyI < keyJ
	})

	return marketplaces, nil
}

func readMCPServers(claudeJSONPath string) ([]MCPServer, error) {
	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		return nil, err
	}

	var claudeJSON ClaudeJSON
	if err := json.Unmarshal(data, &claudeJSON); err != nil {
		return nil, err
	}

	var servers []MCPServer
	for name, server := range claudeJSON.MCPServers {
		servers = append(servers, MCPServer{
			Name:    name,
			Command: server.Command,
			Args:    server.Args,
			Scope:   "user",
		})
	}

	// Sort by name for consistent output
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers, nil
}
