// ABOUTME: Helper functions for scope-aware operations
// ABOUTME: Provides profile resolution and plugin-by-scope rendering
package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/claudeup/claudeup/v2/internal/claude"
	"github.com/claudeup/claudeup/v2/internal/config"
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/internal/ui"
)

// ActiveProfileInfo represents an active profile at a specific scope
type ActiveProfileInfo struct {
	Name  string
	Scope string
}

// getActiveProfile returns the active profile name and scope using the hierarchy:
// 1. Project scope (.claudeup.json in cwd) - highest priority
// 2. Local scope (projects.json registry)
// 3. User scope (~/.claudeup/config.json) - lowest priority
//
// Returns empty strings if no profile is active.
func getActiveProfile(cwd string) (profileName, scope string) {
	// Check project scope first (highest precedence)
	if profile.ProjectConfigExists(cwd) {
		if projectCfg, err := profile.LoadProjectConfig(cwd); err == nil && projectCfg.Profile != "" {
			return projectCfg.Profile, "project"
		}
	}

	// Check local scope in registry
	if registry, err := config.LoadProjectsRegistry(); err == nil {
		if entry, ok := registry.GetProject(cwd); ok && entry.Profile != "" {
			return entry.Profile, "local"
		}
	}

	// Fall back to user-level config
	if cfg, _ := config.Load(); cfg != nil && cfg.Preferences.ActiveProfile != "" {
		return cfg.Preferences.ActiveProfile, "user"
	}

	return "", ""
}

// getAllActiveProfiles returns active profiles from all scopes that have a profile set
// Returns profiles in order: project, local, user (matching precedence order)
// Used to display all active profiles when they differ across scopes
func getAllActiveProfiles(cwd string) []ActiveProfileInfo {
	var profiles []ActiveProfileInfo

	// Project scope
	if profile.ProjectConfigExists(cwd) {
		if projectCfg, err := profile.LoadProjectConfig(cwd); err == nil && projectCfg.Profile != "" {
			profiles = append(profiles, ActiveProfileInfo{
				Name:  projectCfg.Profile,
				Scope: "project",
			})
		}
	}

	// Local scope
	if registry, err := config.LoadProjectsRegistry(); err == nil {
		if entry, ok := registry.GetProject(cwd); ok && entry.Profile != "" {
			profiles = append(profiles, ActiveProfileInfo{
				Name:  entry.Profile,
				Scope: "local",
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

// RenderPluginsByScope displays enabled plugins grouped by scope.
// This is the shared implementation used by both 'scope list' and 'plugin list --by-scope'.
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
