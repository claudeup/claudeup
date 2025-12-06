// ABOUTME: Update command implementation for checking and applying updates
// ABOUTME: Checks marketplaces and plugins for available updates
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/malston/claude-pm/internal/claude"
	"github.com/spf13/cobra"
)

var (
	updateCheckOnly bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and apply updates to marketplaces and plugins",
	Long: `Check if marketplaces or plugins have updates available and optionally apply them.

By default, checks for updates and prompts to install them.
Use --check-only to see what's available without making changes.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check-only", false, "Check for updates without applying them")
}

type MarketplaceUpdate struct {
	Name          string
	HasUpdate     bool
	CurrentCommit string
	LatestCommit  string
}

type PluginUpdate struct {
	Name          string
	HasUpdate     bool
	CurrentCommit string
	LatestCommit  string
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking for updates...")

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
	fmt.Println("━━━ Checking Marketplaces ━━━")
	marketplaceUpdates := checkMarketplaceUpdates(marketplaces)

	hasMarketplaceUpdates := false
	for _, update := range marketplaceUpdates {
		if update.HasUpdate {
			fmt.Printf("  ⚠ %s: Update available\n", update.Name)
			hasMarketplaceUpdates = true
		} else {
			fmt.Printf("  ✓ %s: Up to date\n", update.Name)
		}
	}

	// Check plugin updates
	fmt.Println("\n━━━ Checking Plugins ━━━")
	pluginUpdates := checkPluginUpdates(plugins, marketplaces)

	hasPluginUpdates := false
	for _, update := range pluginUpdates {
		if update.HasUpdate {
			fmt.Printf("  ⚠ %s: Update available\n", update.Name)
			hasPluginUpdates = true
		}
	}

	if !hasPluginUpdates {
		fmt.Println("  ✓ All plugins up to date")
	}

	// Summary
	fmt.Println("\n━━━ Summary ━━━")
	if !hasMarketplaceUpdates && !hasPluginUpdates {
		fmt.Println("✓ Everything is up to date!")
		return nil
	}

	if hasMarketplaceUpdates {
		fmt.Println("\nMarketplace updates available:")
		for _, update := range marketplaceUpdates {
			if update.HasUpdate {
				fmt.Printf("  • %s\n", update.Name)
			}
		}
	}

	if hasPluginUpdates {
		fmt.Println("\nPlugin updates available:")
		for _, update := range pluginUpdates {
			if update.HasUpdate {
				fmt.Printf("  • %s\n", update.Name)
			}
		}
	}

	if updateCheckOnly {
		fmt.Println("\nRun without --check-only to apply updates")
		return nil
	}

	// Prompt to update
	fmt.Print("\nApply updates? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled")
		return nil
	}

	// Apply marketplace updates
	if hasMarketplaceUpdates {
		fmt.Println("\n━━━ Updating Marketplaces ━━━")
		for _, update := range marketplaceUpdates {
			if update.HasUpdate {
				if err := updateMarketplace(update.Name, marketplaces[update.Name].InstallLocation); err != nil {
					fmt.Printf("  ✗ %s: %v\n", update.Name, err)
				} else {
					fmt.Printf("  ✓ %s: Updated\n", update.Name)
				}
			}
		}
	}

	// Apply plugin updates
	if hasPluginUpdates {
		fmt.Println("\n━━━ Updating Plugins ━━━")
		for _, update := range pluginUpdates {
			if update.HasUpdate {
				if err := updatePlugin(update.Name, plugins); err != nil {
					fmt.Printf("  ✗ %s: %v\n", update.Name, err)
				} else {
					fmt.Printf("  ✓ %s: Updated\n", update.Name)
				}
			}
		}

		// Save updated plugin registry
		if err := claude.SavePlugins(claudeDir, plugins); err != nil {
			return fmt.Errorf("failed to save plugins: %w", err)
		}
	}

	fmt.Println("\n✓ Updates complete!")

	return nil
}

func checkMarketplaceUpdates(marketplaces claude.MarketplaceRegistry) []MarketplaceUpdate {
	var updates []MarketplaceUpdate

	for name, marketplace := range marketplaces {
		// Fetch latest from remote
		gitDir := filepath.Join(marketplace.InstallLocation, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			// Not a git repo, skip
			updates = append(updates, MarketplaceUpdate{
				Name:      name,
				HasUpdate: false,
			})
			continue
		}

		// Get current commit
		currentCmd := exec.Command("git", "-C", marketplace.InstallLocation, "rev-parse", "HEAD")
		currentOutput, err := currentCmd.Output()
		if err != nil {
			updates = append(updates, MarketplaceUpdate{
				Name:      name,
				HasUpdate: false,
			})
			continue
		}
		currentCommit := strings.TrimSpace(string(currentOutput))

		// Fetch from remote
		fetchCmd := exec.Command("git", "-C", marketplace.InstallLocation, "fetch", "origin")
		fetchCmd.Run() // Ignore errors

		// Get remote commit
		remoteCmd := exec.Command("git", "-C", marketplace.InstallLocation, "rev-parse", "origin/HEAD")
		remoteOutput, err := remoteCmd.Output()
		if err != nil {
			// Try main branch
			remoteCmd = exec.Command("git", "-C", marketplace.InstallLocation, "rev-parse", "origin/main")
			remoteOutput, err = remoteCmd.Output()
			if err != nil {
				// Try master branch
				remoteCmd = exec.Command("git", "-C", marketplace.InstallLocation, "rev-parse", "origin/master")
				remoteOutput, err = remoteCmd.Output()
				if err != nil {
					updates = append(updates, MarketplaceUpdate{
						Name:      name,
						HasUpdate: false,
					})
					continue
				}
			}
		}
		remoteCommit := strings.TrimSpace(string(remoteOutput))

		updates = append(updates, MarketplaceUpdate{
			Name:          name,
			HasUpdate:     currentCommit != remoteCommit,
			CurrentCommit: currentCommit[:7],
			LatestCommit:  remoteCommit[:7],
		})
	}

	return updates
}

func checkPluginUpdates(plugins *claude.PluginRegistry, marketplaces claude.MarketplaceRegistry) []PluginUpdate {
	var updates []PluginUpdate

	for name, plugin := range plugins.Plugins {
		// Skip if plugin path doesn't exist
		if !plugin.PathExists() {
			continue
		}

		// Find the marketplace this plugin belongs to
		var marketplacePath string
		for _, marketplace := range marketplaces {
			if strings.Contains(plugin.InstallPath, marketplace.InstallLocation) {
				marketplacePath = marketplace.InstallLocation
				break
			}
		}

		if marketplacePath == "" {
			continue
		}

		// Get current commit from marketplace
		gitDir := filepath.Join(marketplacePath, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			continue
		}

		currentCmd := exec.Command("git", "-C", marketplacePath, "rev-parse", "HEAD")
		currentOutput, err := currentCmd.Output()
		if err != nil {
			continue
		}
		currentCommit := strings.TrimSpace(string(currentOutput))

		// Compare with plugin's gitCommitSha
		if plugin.GitCommitSha != currentCommit {
			updates = append(updates, PluginUpdate{
				Name:          name,
				HasUpdate:     true,
				CurrentCommit: plugin.GitCommitSha[:7],
				LatestCommit:  currentCommit[:7],
			})
		}
	}

	return updates
}

func updateMarketplace(name, path string) error {
	// Git pull to update
	cmd := exec.Command("git", "-C", path, "pull", "--ff-only")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}
	return nil
}

func updatePlugin(name string, plugins *claude.PluginRegistry) error {
	// Update the gitCommitSha to the latest marketplace commit
	plugin, exists := plugins.Plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found")
	}

	// Find marketplace path
	var marketplacePath string
	parts := strings.Split(plugin.InstallPath, string(filepath.Separator))
	for i, part := range parts {
		if part == "marketplaces" && i+1 < len(parts) {
			marketplacePath = strings.Join(parts[:i+2], string(filepath.Separator))
			break
		}
	}

	if marketplacePath == "" {
		return fmt.Errorf("marketplace not found")
	}

	// Get latest commit
	cmd := exec.Command("git", "-C", marketplacePath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get latest commit: %w", err)
	}

	latestCommit := strings.TrimSpace(string(output))
	plugin.GitCommitSha = latestCommit
	plugins.Plugins[name] = plugin

	return nil
}
