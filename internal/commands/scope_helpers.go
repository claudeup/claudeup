// ABOUTME: Scope-aware operations used across commands
// ABOUTME: Provides plugin-by-scope rendering and scope clearing
package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/ui"
)

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
		scopesToShow = claude.ValidScopes
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

// clearScope removes plugin settings at the given scope.
// For "user" scope, only enabledPlugins is cleared while other settings are preserved.
// For "project" and "local" scope, the entire settings file is removed.
// If the file does not exist, the operation succeeds silently.
// Unrecognised scope values return an error.
func clearScope(scope string, settingsPath string, claudeDir string) error {
	switch scope {
	case "user":
		// Load existing settings and only clear enabledPlugins
		settings, err := claude.LoadSettingsOrEmpty(claudeDir)
		if err != nil {
			return fmt.Errorf("failed to load settings: %w", err)
		}
		settings.EnabledPlugins = make(map[string]bool)
		return claude.SaveSettings(claudeDir, settings)

	case "project", "local":
		if err := os.Remove(settingsPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil

	default:
		return fmt.Errorf("invalid scope: %s", scope)
	}
}
