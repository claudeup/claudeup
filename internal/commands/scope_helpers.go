// ABOUTME: Scope-aware operations used across commands
// ABOUTME: Provides profile resolution, plugin-by-scope rendering, and scope clearing
package commands

import (
	"fmt"
	"os"

	"github.com/claudeup/claudeup/v4/internal/claude"
	"github.com/claudeup/claudeup/v4/internal/config"
)

// ActiveProfileInfo represents an active profile at a specific scope
type ActiveProfileInfo struct {
	Name  string
	Scope string
}

// getActiveProfile returns the active profile name and scope using the hierarchy:
// 1. Local scope (projects.json registry) - highest priority
// 2. User scope (~/.claudeup/config.json) - lowest priority
//
// Returns empty strings if no profile is active.
func getActiveProfile(cwd string) (profileName, scope string) {
	// Check local scope in registry first (highest precedence)
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
// Returns profiles in order: local, user (matching precedence order)
// Used to display all active profiles when they differ across scopes
func getAllActiveProfiles(cwd string) []ActiveProfileInfo {
	var profiles []ActiveProfileInfo

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
