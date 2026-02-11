// ABOUTME: Update command for self-updating the claudeup CLI
// ABOUTME: Downloads and installs the latest version from GitHub
package commands

import (
	"fmt"

	"github.com/claudeup/claudeup/v5/internal/selfupdate"
	"github.com/claudeup/claudeup/v5/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the claudeup CLI to the latest version",
	Long: `Update the claudeup CLI binary to the latest version from GitHub.

This command checks GitHub releases for a newer version and, if found,
downloads and installs it automatically.`,
	Example: `  # Update claudeup to the latest version
  claudeup update`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	currentVersion := rootCmd.Version

	ui.PrintInfo("Checking for updates...")

	// Check latest version
	latestVersion, err := selfupdate.CheckLatestVersion(selfupdate.DefaultAPIURL)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Check if update needed
	if !selfupdate.IsNewer(currentVersion, latestVersion) {
		ui.PrintSuccess(fmt.Sprintf("Already up to date (%s)", currentVersion))
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("Updating %s → %s...", currentVersion, latestVersion))

	// Perform update
	result := selfupdate.Update(currentVersion, latestVersion, "")
	if result.Error != nil {
		return fmt.Errorf("update failed: %w", result.Error)
	}

	ui.PrintSuccess(fmt.Sprintf("Updated claudeup %s → %s", result.OldVersion, result.NewVersion))
	return nil
}
