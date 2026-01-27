// ABOUTME: Functions for analyzing plugin state across multiple scopes
// ABOUTME: Provides comprehensive view of where plugins are enabled and installed
package claude

import (
	"sort"
)

// PluginAnalysisResult contains both installed plugin analysis and orphaned enabled entries
type PluginAnalysisResult struct {
	Installed           map[string]*PluginScopeInfo // Plugins in the registry
	EnabledNotInstalled []string                    // Plugin names enabled but not in registry
}

// PluginScopeInfo provides detailed information about a plugin's state across all scopes
type PluginScopeInfo struct {
	Name         string           // Plugin name (e.g., "my-plugin@marketplace")
	EnabledAt    []string         // Scopes where plugin is enabled: "user", "project", "local"
	InstalledAt  []PluginMetadata // All installation instances across scopes
	ActiveSource string           // Which scope's installation is used (based on precedence: local > project > user)
}

// AnalyzePluginScopes examines all scopes and returns comprehensive plugin information
// showing where each plugin is installed and enabled
func AnalyzePluginScopes(claudeDir string, projectDir string) (map[string]*PluginScopeInfo, error) {
	// Load plugin registry (shows all installations across all scopes)
	registry, err := LoadPlugins(claudeDir)
	if err != nil {
		return nil, err
	}

	// Load settings from all scopes
	scopes := []string{"user", "project", "local"}
	scopeSettings := make(map[string]*Settings)

	for _, scope := range scopes {
		settings, err := LoadSettingsForScope(scope, claudeDir, projectDir)
		if err != nil {
			// For user scope, this is an error
			if scope == "user" {
				return nil, err
			}
			// For project/local, missing settings is ok (empty settings)
			settings = &Settings{
				EnabledPlugins: make(map[string]bool),
			}
		}
		scopeSettings[scope] = settings
	}

	// Build analysis map
	analysis := make(map[string]*PluginScopeInfo)

	// Iterate through all installed plugins
	for pluginName, instances := range registry.Plugins {
		info := &PluginScopeInfo{
			Name:        pluginName,
			EnabledAt:   []string{},
			InstalledAt: instances,
		}

		// Check which scopes have this plugin enabled
		for _, scope := range scopes {
			if scopeSettings[scope].IsPluginEnabled(pluginName) {
				info.EnabledAt = append(info.EnabledAt, scope)
			}
		}

		// Determine active source based on precedence (local > project > user)
		// Only set if plugin is enabled at least somewhere
		if len(info.EnabledAt) > 0 {
			info.ActiveSource = determineActiveSource(instances, info.EnabledAt)
		}

		analysis[pluginName] = info
	}

	return analysis, nil
}

// AnalyzePluginScopesWithOrphans examines all scopes and returns comprehensive plugin information
// including plugins that are enabled in settings but not actually installed
func AnalyzePluginScopesWithOrphans(claudeDir string, projectDir string) (*PluginAnalysisResult, error) {
	// Load plugin registry (shows all installations across all scopes)
	registry, err := LoadPlugins(claudeDir)
	if err != nil {
		return nil, err
	}

	// Load settings from all scopes
	scopes := []string{"user", "project", "local"}
	scopeSettings := make(map[string]*Settings)

	for _, scope := range scopes {
		settings, err := LoadSettingsForScope(scope, claudeDir, projectDir)
		if err != nil {
			// For user scope, this is an error
			if scope == "user" {
				return nil, err
			}
			// For project/local, missing settings is ok (empty settings)
			settings = &Settings{
				EnabledPlugins: make(map[string]bool),
			}
		}
		scopeSettings[scope] = settings
	}

	// Build analysis map for installed plugins
	analysis := make(map[string]*PluginScopeInfo)

	// Iterate through all installed plugins
	for pluginName, instances := range registry.Plugins {
		info := &PluginScopeInfo{
			Name:        pluginName,
			EnabledAt:   []string{},
			InstalledAt: instances,
		}

		// Check which scopes have this plugin enabled
		for _, scope := range scopes {
			if scopeSettings[scope].IsPluginEnabled(pluginName) {
				info.EnabledAt = append(info.EnabledAt, scope)
			}
		}

		// Determine active source based on precedence (local > project > user)
		// Only set if plugin is enabled at least somewhere
		if len(info.EnabledAt) > 0 {
			info.ActiveSource = determineActiveSource(instances, info.EnabledAt)
		}

		analysis[pluginName] = info
	}

	// Collect enabled-but-not-installed plugins (orphans)
	orphanSet := make(map[string]bool)
	for _, scope := range scopes {
		for pluginName, enabled := range scopeSettings[scope].EnabledPlugins {
			// Only include if enabled (true) and not in registry
			if enabled && !registry.PluginExists(pluginName) {
				orphanSet[pluginName] = true
			}
		}
	}

	// Convert set to sorted slice
	orphans := make([]string, 0, len(orphanSet))
	for name := range orphanSet {
		orphans = append(orphans, name)
	}
	sort.Strings(orphans)

	return &PluginAnalysisResult{
		Installed:           analysis,
		EnabledNotInstalled: orphans,
	}, nil
}

// determineActiveSource finds which installation is active based on scope precedence.
//
// Claude Code's precedence model (from docs.claude.com/settings):
//   - More specific scopes always take precedence: local > project > user
//   - When a plugin is enabled at any scope, it uses the highest-precedence installation available
//
// Examples:
//   - Installed at project, enabled at user → uses project (higher precedence installation)
//   - Installed at user, enabled at local → uses user (only available installation)
//   - Installed at local+user, enabled at project → uses local (highest precedence)
func determineActiveSource(installations []PluginMetadata, enabledScopes []string) string {
	if len(enabledScopes) == 0 {
		return ""
	}

	// Build map of available installations by scope
	installMap := make(map[string]bool)
	for _, inst := range installations {
		installMap[inst.Scope] = true
	}

	// Return highest-precedence installation available
	precedence := []string{"local", "project", "user"}
	for _, scope := range precedence {
		if installMap[scope] {
			return scope
		}
	}

	return ""
}

// GetEnabledPlugins returns a sorted list of all plugins enabled at any scope
func (a *PluginScopeInfo) GetEnabledPlugins() []string {
	return a.EnabledAt
}

// IsEnabled returns true if the plugin is enabled at any scope
func (a *PluginScopeInfo) IsEnabled() bool {
	return len(a.EnabledAt) > 0
}

// GetInstallationForScope returns the installation metadata for a specific scope
func (a *PluginScopeInfo) GetInstallationForScope(scope string) *PluginMetadata {
	for _, inst := range a.InstalledAt {
		if inst.Scope == scope {
			return &inst
		}
	}
	return nil
}

// GetSortedScopeNames returns scope names sorted by precedence (highest first)
func GetSortedScopeNames() []string {
	return []string{"local", "project", "user"}
}

// SortScopesByPrecedence sorts a slice of scope names by precedence (highest first)
func SortScopesByPrecedence(scopes []string) {
	precedence := map[string]int{
		"local":   0,
		"project": 1,
		"user":    2,
	}
	sort.Slice(scopes, func(i, j int) bool {
		return precedence[scopes[i]] < precedence[scopes[j]]
	})
}
