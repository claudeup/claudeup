// ABOUTME: Status command implementation showing overview of Claude installation
// ABOUTME: Displays marketplaces, plugins, MCP servers, and detected issues
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	statusScope string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show an overview of Claude Code installation",
	Long: `Display the current state of your Claude Code installation.

Shows:
  - Active profile
  - Installed marketplaces
  - Plugin counts and status
  - Any detected issues

For detailed plugin information, use 'claudeup plugins'.
For diagnostics, use 'claudeup doctor'.`,
	Args: cobra.NoArgs,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&statusScope, "scope", "", "Check status for specific scope (user, project, or local)")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Get current directory for scope-aware settings
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load marketplaces
	marketplaces, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load marketplaces: %w", err)
	}

	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Load settings - scope-aware
	var settings *claude.Settings
	if statusScope != "" {
		// Validate scope
		if err := claude.ValidateScope(statusScope); err != nil {
			return err
		}
		// Load specific scope
		settings, err = claude.LoadSettingsForScope(statusScope, claudeDir, projectDir)
		if err != nil {
			return fmt.Errorf("failed to load %s scope settings: %w", statusScope, err)
		}
	} else {
		// Load merged settings from all scopes
		settings, err = claude.LoadMergedSettings(claudeDir, projectDir)
		if err != nil {
			return fmt.Errorf("failed to load settings: %w", err)
		}
	}

	// Print header
	fmt.Println(ui.RenderSection("claudeup Status", -1))

	// Print active profile
	cfg, _ := config.Load()
	activeProfile := "none"
	if cfg != nil && cfg.Preferences.ActiveProfile != "" {
		activeProfile = cfg.Preferences.ActiveProfile
	}
	fmt.Println()
	fmt.Println(ui.RenderDetail("Active Profile", ui.Bold(activeProfile)))

	// Check for unsaved profile changes (scope-aware)
	if cfg != nil && cfg.Preferences.ActiveProfile != "" {
		homeDir, _ := os.UserHomeDir()
		profilesDir := filepath.Join(homeDir, ".claudeup", "profiles")
		claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

		// Check if profile file exists (both disk and embedded)
		_, diskErr := profile.Load(profilesDir, cfg.Preferences.ActiveProfile)
		_, embeddedErr := profile.GetEmbeddedProfile(cfg.Preferences.ActiveProfile)

		if diskErr != nil && embeddedErr != nil {
			// Profile doesn't exist anywhere - auto-clear
			ui.PrintWarning(fmt.Sprintf("Active profile '%s' not found. Clearing active profile.", cfg.Preferences.ActiveProfile))
			cfg.Preferences.ActiveProfile = ""
			if err := config.Save(cfg); err != nil {
				ui.PrintWarning(fmt.Sprintf("Could not clear active profile: %v", err))
			}
		} else {
			// Determine which scopes to check
			scopesToCheck := []string{}
			if statusScope != "" {
				// User specified a specific scope
				scopesToCheck = append(scopesToCheck, statusScope)
			} else {
				// Check all scopes (user always, project/local if settings exist)
				scopesToCheck = append(scopesToCheck, "user")

				// Check if project scope settings exist
				projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				if _, err := os.Stat(projectSettingsPath); err == nil {
					scopesToCheck = append(scopesToCheck, "project")
				}

				// Check if local scope settings exist
				localSettingsPath := filepath.Join(projectDir, ".claude", "settings-local.json")
				if _, err := os.Stat(localSettingsPath); err == nil {
					scopesToCheck = append(scopesToCheck, "local")
				}
			}

			// Check each scope for drift
			hasAnyDrift := false
			for _, scope := range scopesToCheck {
				modified, comparisonErr := profile.IsProfileModifiedAtScope(
					cfg.Preferences.ActiveProfile,
					profilesDir,
					claudeDir,
					claudeJSONPath,
					projectDir,
					scope,
				)

				if comparisonErr != nil {
					// Subtle warning for debugging - don't alarm users
					ui.PrintMuted(fmt.Sprintf("Note: Could not check %s scope for profile changes (%v)", scope, comparisonErr))
					continue
				}

				if modified {
					if !hasAnyDrift {
						fmt.Println()
						ui.PrintWarning(fmt.Sprintf("Active profile '%s' has unsaved changes:", cfg.Preferences.ActiveProfile))
						hasAnyDrift = true
					}

					// Load profile again to get diff details for summary
					savedProfile, err := profile.Load(profilesDir, cfg.Preferences.ActiveProfile)
					if err != nil {
						savedProfile, err = profile.GetEmbeddedProfile(cfg.Preferences.ActiveProfile)
					}
					if err == nil {
						diff, err := profile.CompareWithScope(savedProfile, claudeDir, claudeJSONPath, projectDir, scope)
						if err == nil && diff.HasChanges() {
							ui.PrintInfo(fmt.Sprintf("  • %s scope: %s", ui.Bold(scope), diff.Summary()))
						}
					}
				}
			}

			if hasAnyDrift {
				if statusScope != "" {
					ui.PrintInfo(fmt.Sprintf("Run 'claudeup profile save --scope %s' to persist changes at %s scope.", statusScope, statusScope))
				} else {
					ui.PrintInfo("Run 'claudeup profile save --scope <scope>' to persist changes for each scope.")
				}
			}
		}
	}

	// Print marketplaces
	fmt.Println()
	fmt.Println(ui.RenderSection("Marketplaces", len(marketplaces)))
	for name := range marketplaces {
		fmt.Printf("  %s %s\n", ui.Success(ui.SymbolSuccess), name)
	}

	// Count enabled plugins and detect issues
	enabledCount := 0
	stalePlugins := []string{}

	for name, plugin := range plugins.GetAllPlugins() {
		// Check if plugin is enabled in settings.json
		if settings.IsPluginEnabled(name) {
			enabledCount++
			// Also check if enabled plugin has stale path
			if !plugin.PathExists() {
				stalePlugins = append(stalePlugins, name)
			}
		}
	}

	// Print plugins summary (only show enabled plugins, like marketplaces)
	fmt.Println()
	fmt.Println(ui.RenderSection("Plugins", enabledCount))
	if enabledCount > 0 {
		for name := range plugins.GetAllPlugins() {
			if settings.IsPluginEnabled(name) {
				fmt.Printf("  %s %s\n", ui.Success(ui.SymbolSuccess), name)
			}
		}
	}
	// Only show stale plugins if there are any
	if len(stalePlugins) > 0 {
		fmt.Printf("  %s %d stale\n", ui.Warning(ui.SymbolWarning), len(stalePlugins))
	}

	// Print MCP servers placeholder
	fmt.Println()
	fmt.Println(ui.RenderSection("MCP Servers", -1))
	fmt.Printf("  %s Run 'claudeup mcp list' for details\n", ui.Muted(ui.SymbolArrow))

	// Print issues if any
	if len(stalePlugins) > 0 {
		fmt.Println()
		fmt.Println(ui.RenderSection("Issues Detected", -1))
		fmt.Printf("  %s %d plugins have stale paths\n", ui.Warning(ui.SymbolWarning), len(stalePlugins))
		for _, name := range stalePlugins {
			fmt.Printf("    - %s\n", name)
		}
		fmt.Printf("  %s Run 'claudeup doctor' for details\n", ui.Muted(ui.SymbolArrow))
	}

	return nil
}
