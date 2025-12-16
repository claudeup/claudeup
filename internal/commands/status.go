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
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Check for unsaved profile changes
	if cfg != nil && cfg.Preferences.ActiveProfile != "" {
		homeDir, _ := os.UserHomeDir()
		profilesDir := filepath.Join(homeDir, ".claudeup", "profiles")
		profilePath := filepath.Join(profilesDir, cfg.Preferences.ActiveProfile+".json")
		claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

		// Check if profile file exists
		if _, err := os.Stat(profilePath); os.IsNotExist(err) {
			// Auto-clear missing profile
			ui.PrintWarning(fmt.Sprintf("Active profile '%s' not found. Clearing active profile.", cfg.Preferences.ActiveProfile))
			cfg.Preferences.ActiveProfile = ""
			if err := config.Save(cfg); err != nil {
				ui.PrintWarning(fmt.Sprintf("Could not clear active profile: %v", err))
			}
		} else {
			// Load and compare
			savedProfile, err := profile.Load(profilesDir, cfg.Preferences.ActiveProfile)
			if err == nil {
				diff, err := profile.CompareWithCurrent(savedProfile, claudeDir, claudeJSONPath)
				if err == nil && diff.HasChanges() {
					fmt.Println()
					ui.PrintWarning(fmt.Sprintf("Active profile '%s' has unsaved changes:", cfg.Preferences.ActiveProfile))
					ui.PrintInfo("  • " + diff.Summary())
					ui.PrintInfo("Run 'claudeup profile save' to persist them.")
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

	// Count enabled/disabled plugins and detect issues
	enabledCount := 0
	stalePlugins := []string{}

	for name, plugin := range plugins.GetAllPlugins() {
		if plugin.PathExists() {
			enabledCount++
		} else {
			stalePlugins = append(stalePlugins, name)
		}
	}

	// Print plugins summary
	fmt.Println()
	fmt.Println(ui.RenderSection("Plugins", len(plugins.GetAllPlugins())))
	fmt.Printf("  %s %d enabled\n", ui.Success(ui.SymbolSuccess), enabledCount)
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
