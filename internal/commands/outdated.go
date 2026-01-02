// ABOUTME: Outdated command shows available updates
// ABOUTME: Checks CLI, marketplaces, and plugins for newer versions
package commands

import (
	"fmt"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/selfupdate"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "Show available updates for CLI, marketplaces, and plugins",
	Long: `Check for available updates across all components:
- claudeup CLI binary
- Installed marketplaces
- Installed plugins

This is a read-only command that only checks for updates without applying them.`,
	Example: `  # Check what's outdated
  claudeup outdated`,
	Args: cobra.NoArgs,
	RunE: runOutdated,
}

func init() {
	rootCmd.AddCommand(outdatedCmd)
}

func runOutdated(cmd *cobra.Command, args []string) error {
	currentVersion := rootCmd.Version

	// Check CLI updates
	fmt.Println()
	fmt.Println(ui.RenderSection("CLI", -1))

	latestVersion, err := selfupdate.CheckLatestVersion(selfupdate.DefaultAPIURL)
	if err != nil {
		fmt.Printf("  %s claudeup: %s\n", ui.Warning(ui.SymbolWarning), ui.Muted("Unable to check for updates"))
		fmt.Printf("    %s %s\n", ui.Muted("Error:"), ui.Muted(err.Error()))
		fmt.Printf("    %s\n", ui.Muted("Check your network connection or try again later"))
	} else if selfupdate.IsNewer(currentVersion, latestVersion) {
		fmt.Printf("  %s claudeup %s %s %s\n", ui.Warning(ui.SymbolWarning), currentVersion, ui.SymbolArrow, ui.Success(latestVersion))
	} else {
		fmt.Printf("  %s claudeup %s %s\n", ui.Success(ui.SymbolSuccess), currentVersion, ui.Muted("(up to date)"))
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

	// Check marketplace updates
	fmt.Println()
	fmt.Println(ui.RenderSection("Marketplaces", len(marketplaces)))
	if len(marketplaces) == 0 {
		fmt.Printf("  %s\n", ui.Muted("No marketplaces installed"))
	} else {
		marketplaceUpdates := checkMarketplaceUpdates(marketplaces)
		for _, update := range marketplaceUpdates {
			if update.HasUpdate {
				fmt.Printf("  %s %s %s %s %s\n", ui.Warning(ui.SymbolWarning), update.Name, update.CurrentCommit, ui.SymbolArrow, ui.Success(update.LatestCommit))
			} else {
				fmt.Printf("  %s %s %s\n", ui.Success(ui.SymbolSuccess), update.Name, ui.Muted("(up to date)"))
			}
		}
	}

	// Check plugin updates
	fmt.Println()
	fmt.Println(ui.RenderSection("Plugins", len(plugins.GetAllPlugins())))
	if len(plugins.GetAllPlugins()) == 0 {
		fmt.Printf("  %s\n", ui.Muted("No plugins installed"))
	} else {
		pluginUpdates := checkPluginUpdates(plugins, marketplaces)
		if len(pluginUpdates) == 0 {
			fmt.Printf("  %s All plugins up to date\n", ui.Success(ui.SymbolSuccess))
		} else {
			hasOutdated := false
			for _, update := range pluginUpdates {
				if update.HasUpdate {
					hasOutdated = true
					fmt.Printf("  %s %s %s %s %s\n", ui.Warning(ui.SymbolWarning), update.Name, update.CurrentCommit, ui.SymbolArrow, ui.Success(update.LatestCommit))
				}
			}
			if !hasOutdated {
				fmt.Printf("  %s All plugins up to date\n", ui.Success(ui.SymbolSuccess))
			}
		}
	}

	// Footer with suggested commands
	fmt.Println()
	fmt.Printf("%s Run '%s' to update the CLI\n", ui.Muted(ui.SymbolArrow), ui.Bold("claudeup update"))
	fmt.Printf("%s Run '%s' to update marketplaces and plugins\n", ui.Muted(ui.SymbolArrow), ui.Bold("claudeup upgrade"))

	return nil
}
