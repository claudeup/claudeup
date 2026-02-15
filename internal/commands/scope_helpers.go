// ABOUTME: Scope-aware operations used across commands
// ABOUTME: Provides profile resolution, plugin-by-scope rendering, and scope clearing
package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/config"
	"github.com/claudeup/claudeup/v5/internal/ui"
)

// ActiveProfileInfo represents an active profile at a specific scope
type ActiveProfileInfo struct {
	Name  string
	Scope string
}

// getActiveProfile returns the active profile name and scope using the hierarchy:
// 1. Local scope (projects.json registry) - highest priority
// 2. Project scope (projects.json registry) - middle priority
// 3. User scope (~/.claudeup/config.json) - lowest priority
//
// Returns empty strings if no profile is active.
func getActiveProfile(cwd string) (profileName, scope string) {
	if registry, err := config.LoadProjectsRegistry(); err == nil {
		// Check local scope first (highest precedence)
		if entry, ok := registry.GetProject(cwd); ok && entry.Profile != "" {
			return entry.Profile, "local"
		}
		// Check project scope (middle precedence)
		if name, ok := registry.GetProjectScope(cwd); ok {
			return name, "project"
		}
	}

	// Fall back to user-level config
	if cfg, _ := config.Load(); cfg != nil && cfg.Preferences.ActiveProfile != "" {
		return cfg.Preferences.ActiveProfile, "user"
	}

	return "", ""
}

// getAllActiveProfiles returns active profiles from all scopes that have a profile set
// Returns profiles in order: local, project, user (matching precedence order)
// Used to display all active profiles when they differ across scopes
func getAllActiveProfiles(cwd string) []ActiveProfileInfo {
	var profiles []ActiveProfileInfo

	if registry, err := config.LoadProjectsRegistry(); err == nil {
		// Local scope
		if entry, ok := registry.GetProject(cwd); ok && entry.Profile != "" {
			profiles = append(profiles, ActiveProfileInfo{
				Name:  entry.Profile,
				Scope: "local",
			})
		}

		// Project scope
		if name, ok := registry.GetProjectScope(cwd); ok {
			profiles = append(profiles, ActiveProfileInfo{
				Name:  name,
				Scope: "project",
			})
		}
	}

	// User scope
	if cfg, _ := config.Load(); cfg != nil && cfg.Preferences.ActiveProfile != "" {
		profiles = append(profiles, ActiveProfileInfo{
			Name:  cfg.Preferences.ActiveProfile,
			Scope: "user",
		})
	}

	return profiles
}

// UntrackedScopeInfo describes a scope that has settings but no tracked profile
type UntrackedScopeInfo struct {
	Scope        string // "project" or "local"
	PluginCount  int
	SettingsFile string // relative path like ".claude/settings.json"
}

// getUntrackedScopes checks project and local scopes for settings files with
// enabled plugins that have no corresponding tracked profile.
func getUntrackedScopes(cwd, claudeDir string, trackedProfiles []ActiveProfileInfo) []UntrackedScopeInfo {
	trackedScopes := make(map[string]bool)
	for _, p := range trackedProfiles {
		trackedScopes[p.Scope] = true
	}

	var untracked []UntrackedScopeInfo
	for _, scope := range []string{"project", "local"} {
		if trackedScopes[scope] {
			continue
		}

		settings, err := claude.LoadSettingsForScope(scope, claudeDir, cwd)
		if err != nil {
			continue
		}

		count := 0
		for _, enabled := range settings.EnabledPlugins {
			if enabled {
				count++
			}
		}
		if count == 0 {
			continue
		}

		settingsFile := ".claude/settings.json"
		if scope == "local" {
			settingsFile = ".claude/settings.local.json"
		}

		untracked = append(untracked, UntrackedScopeInfo{
			Scope:        scope,
			PluginCount:  count,
			SettingsFile: settingsFile,
		})
	}

	return untracked
}

// renderUntrackedScopeHints displays warnings for scopes with enabled plugins but no tracked profile
func renderUntrackedScopeHints(untrackedScopes []UntrackedScopeInfo) {
	for _, us := range untrackedScopes {
		pluginWord := "plugins"
		if us.PluginCount == 1 {
			pluginWord = "plugin"
		}
		fmt.Printf("  %s %d %s in %s (no profile tracked)\n",
			ui.Warning(us.Scope+":"),
			us.PluginCount, pluginWord, us.SettingsFile)
		fmt.Printf("    %s Save with: claudeup profile save <name> --%s && claudeup profile apply <name> --%s\n",
			ui.Muted(ui.SymbolArrow), us.Scope, us.Scope)
	}
	if len(untrackedScopes) > 0 {
		fmt.Println()
	}
}

// formatScopeName returns a capitalized scope name for display
func formatScopeName(scope string) string {
	switch scope {
	case "user":
		return "User"
	case "project":
		return "Project"
	case "local":
		return "Local"
	default:
		return scope
	}
}

// RenderPluginsByScope displays enabled plugins grouped by scope.
func RenderPluginsByScope(claudeDir, projectDir, filterScope string) error {
	// Validate scope if specified
	if filterScope != "" {
		if err := claude.ValidateScope(filterScope); err != nil {
			return err
		}
	}

	// Determine which scopes to show
	scopesToShow := []string{}
	if filterScope != "" {
		scopesToShow = append(scopesToShow, filterScope)
	} else {
		scopesToShow = []string{"user", "project", "local"}
	}

	// Track if we're in a project directory
	inProjectDir := false
	if _, err := os.Stat(projectDir + "/.claude"); err == nil {
		inProjectDir = true
	}

	// Load settings for each scope and display
	effectivePlugins := make(map[string]bool)

	for _, scope := range scopesToShow {
		// Skip project/local if not in project directory and showing all scopes
		if filterScope == "" && !inProjectDir && (scope == "project" || scope == "local") {
			continue
		}

		// Get settings path for this scope
		settingsPath, err := claude.SettingsPathForScope(scope, claudeDir, projectDir)
		if err != nil {
			return err
		}

		// Load settings for this scope
		settings, err := claude.LoadSettingsForScope(scope, claudeDir, projectDir)
		if err != nil {
			// If file doesn't exist, show appropriate message
			if os.IsNotExist(err) || (scope != "user" && err != nil) {
				if filterScope == "" {
					// When showing all scopes, mention it's not configured
					fmt.Println(ui.RenderSection(fmt.Sprintf("Scope: %s (%s)", formatScopeName(scope), settingsPath), -1))
					fmt.Printf("  %s Not configured\n", ui.Muted("—"))
					fmt.Println()
					continue
				} else {
					// When showing specific scope, it's an error if it doesn't exist
					return fmt.Errorf("%s scope not configured", scope)
				}
			}
			return err
		}

		// Get enabled plugins for this scope
		enabledPlugins := []string{}
		for plugin, enabled := range settings.EnabledPlugins {
			if enabled {
				enabledPlugins = append(enabledPlugins, plugin)
				effectivePlugins[plugin] = true
			}
		}
		sort.Strings(enabledPlugins)

		// Display scope header
		fmt.Println(ui.RenderSection(fmt.Sprintf("Scope: %s (%s)", formatScopeName(scope), settingsPath), len(enabledPlugins)))

		// Display plugins
		if len(enabledPlugins) > 0 {
			for _, plugin := range enabledPlugins {
				fmt.Printf("  %s %s\n", ui.Success(ui.SymbolSuccess), plugin)
			}
		} else {
			fmt.Printf("  %s No plugins enabled\n", ui.Muted("—"))
		}

		fmt.Println()
	}

	// Show messages for non-project directory if showing all scopes
	if filterScope == "" && !inProjectDir {
		fmt.Println(ui.Muted("Project scope: Not in a project directory"))
		fmt.Println(ui.Muted("Local scope: Not configured for this directory"))
		fmt.Println()
	}

	// Show effective configuration when showing all scopes
	if filterScope == "" {
		fmt.Println(ui.Bold(fmt.Sprintf("Effective Configuration: %d unique plugins enabled", len(effectivePlugins))))
	}

	return nil
}

// clearScope removes settings at the specified scope
func clearScope(scope string, settingsPath string, claudeDir string) error {
	switch scope {
	case "user":
		// Load existing settings and only clear enabledPlugins
		settings, err := claude.LoadSettings(claudeDir)
		if err != nil {
			// If settings don't exist, create minimal settings
			settings = &claude.Settings{
				EnabledPlugins: make(map[string]bool),
			}
		} else {
			// Clear only the enabledPlugins field, preserve everything else
			settings.EnabledPlugins = make(map[string]bool)
		}
		return claude.SaveSettings(claudeDir, settings)

	case "project":
		// Remove project settings file
		if err := os.Remove(settingsPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil

	case "local":
		// Remove local settings file
		if err := os.Remove(settingsPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil

	default:
		return fmt.Errorf("invalid scope: %s", scope)
	}
}
