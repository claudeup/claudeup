// ABOUTME: Creates a profile from current Claude Code state
// ABOUTME: Reads installed plugins, marketplaces, and MCP servers
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/claudeup/claudeup/v2/internal/claude"
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

// SnapshotOptions controls how a snapshot is taken
type SnapshotOptions struct {
	Scope      string // user, project, or local
	ProjectDir string // Required for project/local scope
}

// Snapshot creates a Profile from the current Claude Code state (user scope)
func Snapshot(name, claudeDir, claudeJSONPath string) (*Profile, error) {
	return SnapshotWithScope(name, claudeDir, claudeJSONPath, SnapshotOptions{
		Scope: "user",
	})
}

// SnapshotWithScope creates a Profile from a specific scope
func SnapshotWithScope(name, claudeDir, claudeJSONPath string, opts SnapshotOptions) (*Profile, error) {
	p := &Profile{
		Name: name,
	}

	if opts.Scope == "" {
		opts.Scope = "user"
	}

	// Read plugins from scope-specific settings
	plugins, err := readPluginsForScope(claudeDir, opts.ProjectDir, opts.Scope)
	if err == nil {
		p.Plugins = plugins
	}

	// Read marketplaces
	// For project scope, we could read from .claudeup.json if we want project-specific marketplaces
	// For now, marketplaces are always user-scoped
	marketplaces, err := readMarketplaces(claudeDir)
	if err == nil {
		p.Marketplaces = marketplaces
	}

	// Read MCP servers
	// For project scope, read from .mcp.json
	mcpServers, err := readMCPServersForScope(claudeJSONPath, opts.ProjectDir, opts.Scope)
	if err == nil {
		p.MCPServers = mcpServers
	}

	// Auto-generate description based on contents
	p.Description = p.GenerateDescription()

	return p, nil
}

// SnapshotCombined creates a Profile combining all scopes (user + project + local)
// This represents the effective Claude Code configuration, since Claude accumulates
// settings from user → project → local (later scopes override earlier ones)
func SnapshotCombined(name, claudeDir, claudeJSONPath, projectDir string) (*Profile, error) {
	// Combine plugins from all scopes
	plugins, err := readPluginsCombined(claudeDir, projectDir)
	if err != nil {
		return nil, err
	}

	// Read marketplaces (always user-scoped)
	marketplaces, err := readMarketplaces(claudeDir)
	if err != nil {
		marketplaces = []Marketplace{} // Continue even if marketplace read fails
	}

	// Combine MCP servers from all scopes
	mcpServers, err := readMCPServersCombined(claudeJSONPath, projectDir)
	if err != nil {
		mcpServers = []MCPServer{} // Continue even if MCP read fails
	}

	p := &Profile{
		Name:          name,
		Plugins:       plugins,
		Marketplaces:  marketplaces,
		MCPServers:    mcpServers,
	}

	// Auto-generate description based on contents
	p.Description = p.GenerateDescription()

	return p, nil
}

// readPluginsCombined reads enabled plugins from all scopes and combines them
// Claude Code accumulates settings: user → project → local (later overrides earlier)
func readPluginsCombined(claudeDir, projectDir string) ([]string, error) {
	// Start with empty map to track enabled state
	enabledPlugins := make(map[string]bool)

	// Layer 1: User scope
	userSettings, err := claude.LoadSettingsForScope("user", claudeDir, projectDir)
	if err == nil {
		for name, enabled := range userSettings.EnabledPlugins {
			enabledPlugins[name] = enabled
		}
	}

	// Layer 2: Project scope (overrides user)
	projectSettings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
	if err == nil {
		for name, enabled := range projectSettings.EnabledPlugins {
			enabledPlugins[name] = enabled
		}
	}

	// Layer 3: Local scope (overrides project and user)
	localSettings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
	if err == nil {
		for name, enabled := range localSettings.EnabledPlugins {
			enabledPlugins[name] = enabled
		}
	}

	// Extract only enabled plugins
	plugins := make([]string, 0, len(enabledPlugins))
	for name, enabled := range enabledPlugins {
		if enabled {
			plugins = append(plugins, name)
		}
	}
	sort.Strings(plugins)

	return plugins, nil
}

// readMCPServersCombined reads MCP servers from all scopes and combines them
// Claude Code uses REPLACEMENT for MCP servers: user → project → local
// When the same server name exists in multiple scopes, the higher-precedence
// scope COMPLETELY REPLACES the lower-precedence definition (no merging).
func readMCPServersCombined(claudeJSONPath, projectDir string) ([]MCPServer, error) {
	// Use map to track servers by name - higher-precedence scopes completely replace lower ones
	serverMap := make(map[string]MCPServer)

	// Layer 1: User scope (global ~/.claude.json)
	userServers, err := readMCPServersForScope(claudeJSONPath, projectDir, "user")
	if err == nil {
		for _, server := range userServers {
			serverMap[server.Name] = server
		}
	}

	// Layer 2: Project scope (.mcp.json in project root)
	projectServers, err := readMCPServersForScope(claudeJSONPath, projectDir, "project")
	if err == nil {
		for _, server := range projectServers {
			serverMap[server.Name] = server
		}
	}

	// Layer 3: Local scope (.mcp.local.json in project root)
	localServers, err := readMCPServersForScope(claudeJSONPath, projectDir, "local")
	if err == nil {
		for _, server := range localServers {
			serverMap[server.Name] = server
		}
	}

	// Convert map back to slice
	combined := make([]MCPServer, 0, len(serverMap))
	for _, server := range serverMap {
		combined = append(combined, server)
	}

	return combined, nil
}

func readPlugins(claudeDir string) ([]string, error) {
	return readPluginsForScope(claudeDir, "", "user")
}

func readPluginsForScope(claudeDir, projectDir, scope string) ([]string, error) {
	// Read enabled plugins from scope-specific settings.json
	settings, err := claude.LoadSettingsForScope(scope, claudeDir, projectDir)
	if err != nil {
		return nil, err
	}

	// Extract enabled plugin names
	plugins := make([]string, 0, len(settings.EnabledPlugins))
	for name, enabled := range settings.EnabledPlugins {
		if enabled {
			plugins = append(plugins, name)
		}
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
		m := Marketplace{
			Source: meta.Source.Source,
			Repo:   meta.Source.Repo,
			URL:    meta.Source.URL,
		}
		// Filter out invalid marketplaces (no repo or url)
		if m.DisplayName() == "" {
			continue
		}
		marketplaces = append(marketplaces, m)
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
	return readMCPServersForScope(claudeJSONPath, "", "user")
}

func readMCPServersForScope(claudeJSONPath, projectDir, scope string) ([]MCPServer, error) {
	var mcpPath string

	switch scope {
	case "project":
		// Project scope reads from .mcp.json in project directory
		if projectDir == "" {
			return nil, nil // No project directory, return empty
		}
		mcpPath = filepath.Join(projectDir, ".mcp.json")
	case "local":
		// Local scope reads from .claude-local/mcp.json (if we implement it)
		// For now, return empty for local scope
		return nil, nil
	default:
		// User scope reads from ~/.claude.json
		mcpPath = claudeJSONPath
	}

	data, err := os.ReadFile(mcpPath)
	if err != nil {
		// File not existing is not an error for optional scopes
		if os.IsNotExist(err) {
			return nil, nil
		}
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
			Scope:   scope,
		})
	}

	// Sort by name for consistent output
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers, nil
}
