// ABOUTME: Functions for analyzing plugin state across multiple scopes
// ABOUTME: Provides comprehensive view of where plugins are enabled and installed
package claude

import (
	"sort"
)

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

// determineActiveSource finds which installation is active based on scope precedence
// Precedence: local > project > user
// Only considers scopes where the plugin is enabled
func determineActiveSource(installations []PluginMetadata, enabledScopes []string) string {
	// Build map of enabled scopes for quick lookup
	enabledMap := make(map[string]bool)
	for _, scope := range enabledScopes {
		enabledMap[scope] = true
	}

	// Build map of available installations by scope
	installMap := make(map[string]bool)
	for _, inst := range installations {
		installMap[inst.Scope] = true
	}

	// Check in precedence order (highest to lowest)
	precedence := []string{"local", "project", "user"}
	for _, scope := range precedence {
		// Use this scope if:
		// 1. Plugin is enabled at this scope OR
		// 2. Plugin is enabled at a lower-precedence scope and installed at this scope
		if enabledMap[scope] && installMap[scope] {
			return scope
		}
	}

	// Fall back to highest-precedence installation if enabled at lower scope
	for _, scope := range precedence {
		if installMap[scope] {
			// Check if any enabled scope has lower precedence
			scopeIndex := indexOf(precedence, scope)
			for _, enabledScope := range enabledScopes {
				enabledIndex := indexOf(precedence, enabledScope)
				if enabledIndex >= scopeIndex {
					return scope
				}
			}
		}
	}

	// Shouldn't reach here if enabledScopes is not empty, but handle gracefully
	if len(installations) > 0 {
		return installations[0].Scope
	}
	return ""
}

// indexOf returns the index of a string in a slice, or -1 if not found
func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
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
