// ABOUTME: Helper functions for scope-aware profile resolution
// ABOUTME: Provides getActiveProfile to determine active profile using scope hierarchy
package commands

import (
	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/profile"
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
