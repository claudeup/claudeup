// ABOUTME: Creates a profile from current Claude Code state
// ABOUTME: Reads installed plugins, marketplaces, MCP servers, and extensions
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/ext"
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
func Snapshot(name, claudeDir, claudeJSONPath, claudeupHome string) (*Profile, error) {
	return SnapshotWithScope(name, claudeDir, claudeJSONPath, claudeupHome, SnapshotOptions{
		Scope: "user",
	})
}

// SnapshotWithScope creates a Profile from a specific scope
func SnapshotWithScope(name, claudeDir, claudeJSONPath, claudeupHome string, opts SnapshotOptions) (*Profile, error) {
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
	mcpServers, err := ReadMCPServersForScope(claudeJSONPath, opts.ProjectDir, opts.Scope)
	if err == nil {
		p.MCPServers = mcpServers
	}

	// Read extensions from enabled.json
	extensions, err := ReadExtensions(claudeDir, claudeupHome)
	if err == nil && extensions != nil {
		p.Extensions = extensions
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

// UsedMarketplaces returns marketplaces referenced by the given plugins.
func UsedMarketplaces(claudeDir string, plugins []string) ([]Marketplace, error) {
	return readUsedMarketplaces(claudeDir, plugins)
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

// ReadExtensions reads enabled extensions from enabled.json
func ReadExtensions(claudeDir, claudeupHome string) (*ExtensionSettings, error) {
	manager := ext.NewManager(claudeDir, claudeupHome)
	config, err := manager.LoadConfig()
	if err != nil {
		return nil, err
	}

	// If config is empty, return nil (no extensions to capture)
	if len(config) == 0 {
		return nil, nil
	}

	settings := &ExtensionSettings{}
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
	if agents := extractEnabled(ext.CategoryAgents); len(agents) > 0 {
		settings.Agents = agents
		hasItems = true
	}
	if commands := extractEnabled(ext.CategoryCommands); len(commands) > 0 {
		settings.Commands = commands
		hasItems = true
	}
	if skills := extractEnabled(ext.CategorySkills); len(skills) > 0 {
		settings.Skills = skills
		hasItems = true
	}
	if hooks := extractEnabled(ext.CategoryHooks); len(hooks) > 0 {
		settings.Hooks = hooks
		hasItems = true
	}
	if rules := extractEnabled(ext.CategoryRules); len(rules) > 0 {
		settings.Rules = rules
		hasItems = true
	}
	if outputStyles := extractEnabled(ext.CategoryOutputStyles); len(outputStyles) > 0 {
		settings.OutputStyles = outputStyles
		hasItems = true
	}

	if !hasItems {
		return nil, nil
	}

	return settings, nil
}

// ReadMCPServersForScope reads MCP servers from the appropriate config file for the given scope.
func ReadMCPServersForScope(claudeJSONPath, projectDir, scope string) ([]MCPServer, error) {
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
func SnapshotAllScopes(name, claudeDir, claudeJSONPath, projectDir, claudeupHome string) (*Profile, error) {
	p := &Profile{
		Name:     name,
		PerScope: &PerScopeSettings{},
	}

	// Collect all plugins across scopes for marketplace filtering.
	// Non-nil empty slice means "filter strictly" (no marketplaces if no plugins).
	allPlugins := []string{}

	// Capture user scope
	userPlugins, _ := readPluginsForScope(claudeDir, projectDir, "user")
	userMCP, _ := ReadMCPServersForScope(claudeJSONPath, projectDir, "user")
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
		projectMCP, _ := ReadMCPServersForScope(claudeJSONPath, projectDir, "project")
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
		localMCP, _ := ReadMCPServersForScope(claudeJSONPath, projectDir, "local")
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

	// Read user-scoped extensions from enabled.json into PerScope.User
	userExtensions, err := ReadExtensions(claudeDir, claudeupHome)
	if err == nil && userExtensions != nil {
		if p.PerScope.User == nil {
			p.PerScope.User = &ScopeSettings{}
		}
		p.PerScope.User.Extensions = userExtensions
	}

	// Read project-scoped extensions from project .claude/{agents,rules}/
	if projectDir != "" {
		projectExtensions := ReadProjectExtensions(projectDir)
		if projectExtensions != nil {
			if p.PerScope.Project == nil {
				p.PerScope.Project = &ScopeSettings{}
			}
			p.PerScope.Project.Extensions = projectExtensions
		}
	}

	// Auto-generate description based on contents
	p.Description = p.GenerateDescription()

	return p, nil
}

// ReadProjectExtensions scans .claude/{agents,rules}/ in the project directory
// for regular files (not symlinks). Regular files are project-scoped extensions;
// symlinks are user-scoped extensions managed by claudeup and should be skipped.
func ReadProjectExtensions(projectDir string) *ExtensionSettings {
	settings := &ExtensionSettings{}
	hasItems := false

	for _, category := range []string{"agents", "rules"} {
		dir := filepath.Join(projectDir, ".claude", category)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		var items []string
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "CLAUDE.md" {
				continue
			}

			path := filepath.Join(dir, name)
			info, err := os.Lstat(path)
			if err != nil {
				continue
			}

			// Skip symlinks (user-scoped items managed by claudeup)
			if info.Mode()&os.ModeSymlink != 0 {
				continue
			}

			items = append(items, name)
		}

		sort.Strings(items)

		if len(items) > 0 {
			switch category {
			case "agents":
				settings.Agents = items
			case "rules":
				settings.Rules = items
			}
			hasItems = true
		}
	}

	if !hasItems {
		return nil
	}
	return settings
}
