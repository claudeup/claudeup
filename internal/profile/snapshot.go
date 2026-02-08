// ABOUTME: Creates a profile from current Claude Code state
// ABOUTME: Reads installed plugins, marketplaces, MCP servers, and local items
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v4/internal/claude"
	"github.com/claudeup/claudeup/v4/internal/local"
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

	marketplaces, err := readAllMarketplaces(claudeDir)
	if err == nil {
		p.Marketplaces = marketplaces
	}

	// Read MCP servers
	// For project scope, read from .mcp.json
	mcpServers, err := readMCPServersForScope(claudeJSONPath, opts.ProjectDir, opts.Scope)
	if err == nil {
		p.MCPServers = mcpServers
	}

	// Read local items from enabled.json
	localItems, err := readLocalItems(claudeDir)
	if err == nil && localItems != nil {
		p.LocalItems = localItems
	}

	// Auto-generate description based on contents
	p.Description = p.GenerateDescription()

	return p, nil
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

// marketplaceNamesFromPlugins extracts the set of marketplace names
// referenced by plugin identifiers (the part after @).
func marketplaceNamesFromPlugins(plugins []string) map[string]bool {
	names := make(map[string]bool)
	for _, plugin := range plugins {
		parts := strings.SplitN(plugin, "@", 2)
		if len(parts) == 2 && parts[1] != "" {
			names[parts[1]] = true
		}
	}
	return names
}

// readAllMarketplaces returns all valid marketplaces from the registry.
// Used by ComputeDiff which needs the full set to detect removals.
func readAllMarketplaces(claudeDir string) ([]Marketplace, error) {
	return readMarketplaces(claudeDir, nil)
}

// readUsedMarketplaces returns only marketplaces referenced by at least one
// of the given plugins (matched by the @suffix in plugin names).
// If plugins is empty, no marketplaces are returned.
func readUsedMarketplaces(claudeDir string, plugins []string) ([]Marketplace, error) {
	return readMarketplaces(claudeDir, plugins)
}

// readMarketplaces reads the marketplace registry. When plugins is nil, all
// marketplaces are returned. When plugins is non-nil, only marketplaces
// referenced by at least one plugin are included (empty slice returns none).
func readMarketplaces(claudeDir string, plugins []string) ([]Marketplace, error) {
	marketplacesPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	data, err := os.ReadFile(marketplacesPath)
	if err != nil {
		return nil, err
	}

	var registry MarketplaceRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}

	var usedNames map[string]bool
	if plugins != nil {
		usedNames = marketplaceNamesFromPlugins(plugins)
	}

	var marketplaces []Marketplace
	for name, meta := range registry {
		// When filtering, only include marketplaces referenced by plugins
		if usedNames != nil && !usedNames[name] {
			continue
		}
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

// readLocalItems reads enabled local items from enabled.json
func readLocalItems(claudeDir string) (*LocalItemSettings, error) {
	manager := local.NewManager(claudeDir)
	config, err := manager.LoadConfig()
	if err != nil {
		return nil, err
	}

	// If config is empty, return nil (no local items to capture)
	if len(config) == 0 {
		return nil, nil
	}

	settings := &LocalItemSettings{}
	hasItems := false

	// Helper to extract enabled items for a category, verifying each
	// item actually exists in the active directory (filters stale entries)
	extractEnabled := func(category string) []string {
		items, ok := config[category]
		if !ok {
			return nil
		}
		activeDir := filepath.Join(claudeDir, category)
		var enabled []string
		for name, isEnabled := range items {
			if !isEnabled {
				continue
			}
			// Verify the item exists in the active directory (skip stale entries)
			if _, err := os.Stat(filepath.Join(activeDir, name)); os.IsNotExist(err) {
				continue
			}
			enabled = append(enabled, name)
		}
		sort.Strings(enabled)
		return enabled
	}

	// Extract each category
	if agents := extractEnabled(local.CategoryAgents); len(agents) > 0 {
		settings.Agents = agents
		hasItems = true
	}
	if commands := extractEnabled(local.CategoryCommands); len(commands) > 0 {
		settings.Commands = commands
		hasItems = true
	}
	if skills := extractEnabled(local.CategorySkills); len(skills) > 0 {
		settings.Skills = skills
		hasItems = true
	}
	if hooks := extractEnabled(local.CategoryHooks); len(hooks) > 0 {
		settings.Hooks = hooks
		hasItems = true
	}
	if rules := extractEnabled(local.CategoryRules); len(rules) > 0 {
		settings.Rules = rules
		hasItems = true
	}
	if outputStyles := extractEnabled(local.CategoryOutputStyles); len(outputStyles) > 0 {
		settings.OutputStyles = outputStyles
		hasItems = true
	}

	if !hasItems {
		return nil, nil
	}

	return settings, nil
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

// SnapshotAllScopes creates a Profile capturing settings from all three scopes
// (user, project, local) and organizing them in the PerScope structure.
// This is the preferred way to save profiles as it preserves scope information.
func SnapshotAllScopes(name, claudeDir, claudeJSONPath, projectDir string) (*Profile, error) {
	p := &Profile{
		Name:     name,
		PerScope: &PerScopeSettings{},
	}

	// Collect all plugins across scopes for marketplace filtering.
	// Non-nil empty slice means "filter strictly" (no marketplaces if no plugins).
	allPlugins := []string{}

	// Capture user scope
	userPlugins, _ := readPluginsForScope(claudeDir, projectDir, "user")
	userMCP, _ := readMCPServersForScope(claudeJSONPath, projectDir, "user")
	allPlugins = append(allPlugins, userPlugins...)
	if len(userPlugins) > 0 || len(userMCP) > 0 {
		p.PerScope.User = &ScopeSettings{
			Plugins:    userPlugins,
			MCPServers: userMCP,
		}
	}

	// Capture project scope
	if projectDir != "" {
		projectPlugins, _ := readPluginsForScope(claudeDir, projectDir, "project")
		projectMCP, _ := readMCPServersForScope(claudeJSONPath, projectDir, "project")
		allPlugins = append(allPlugins, projectPlugins...)
		if len(projectPlugins) > 0 || len(projectMCP) > 0 {
			p.PerScope.Project = &ScopeSettings{
				Plugins:    projectPlugins,
				MCPServers: projectMCP,
			}
		}
	}

	// Capture local scope
	if projectDir != "" {
		localPlugins, _ := readPluginsForScope(claudeDir, projectDir, "local")
		localMCP, _ := readMCPServersForScope(claudeJSONPath, projectDir, "local")
		allPlugins = append(allPlugins, localPlugins...)
		if len(localPlugins) > 0 || len(localMCP) > 0 {
			p.PerScope.Local = &ScopeSettings{
				Plugins:    localPlugins,
				MCPServers: localMCP,
			}
		}
	}

	// Marketplaces are always user-scoped; only include those used by plugins
	marketplaces, err := readUsedMarketplaces(claudeDir, allPlugins)
	if err == nil {
		p.Marketplaces = marketplaces
	}

	// Read local items from enabled.json
	localItems, err := readLocalItems(claudeDir)
	if err == nil && localItems != nil {
		p.LocalItems = localItems
	}

	// Auto-generate description based on contents
	p.Description = p.GenerateDescription()

	return p, nil
}
