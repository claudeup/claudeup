// ABOUTME: Update command implementation for checking and applying updates
// ABOUTME: Checks marketplaces and plugins for available updates
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
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

	var outdatedMarketplaces []string
	for _, update := range marketplaceUpdates {
		if update.HasUpdate {
			fmt.Printf("  ⚠ %s: Update available\n", update.Name)
			outdatedMarketplaces = append(outdatedMarketplaces, update.Name)
		} else {
			fmt.Printf("  ✓ %s: Up to date\n", update.Name)
		}
	}

	// Check plugin updates
	fmt.Println("\n━━━ Checking Plugins ━━━")
	pluginUpdates := checkPluginUpdates(plugins, marketplaces)

	var outdatedPlugins []string
	for _, update := range pluginUpdates {
		if update.HasUpdate {
			fmt.Printf("  ⚠ %s: Update available\n", update.Name)
			outdatedPlugins = append(outdatedPlugins, update.Name)
		}
	}

	if len(outdatedPlugins) == 0 {
		fmt.Println("  ✓ All plugins up to date")
	}

	// Summary
	fmt.Println("\n━━━ Summary ━━━")
	if len(outdatedMarketplaces) == 0 && len(outdatedPlugins) == 0 {
		fmt.Println("✓ Everything is up to date!")
		return nil
	}

	if updateCheckOnly {
		if len(outdatedMarketplaces) > 0 {
			fmt.Println("\nMarketplace updates available:")
			for _, name := range outdatedMarketplaces {
				fmt.Printf("  • %s\n", name)
			}
		}
		if len(outdatedPlugins) > 0 {
			fmt.Println("\nPlugin updates available:")
			for _, name := range outdatedPlugins {
				fmt.Printf("  • %s\n", name)
			}
		}
		fmt.Println("\nRun without --check-only to apply updates")
		return nil
	}

	// Interactive selection for marketplaces
	if len(outdatedMarketplaces) > 0 {
		selectedMarketplaces, err := ui.SelectFromList(
			"\nMarketplaces with updates available:",
			outdatedMarketplaces,
		)
		if err != nil {
			return err
		}
		outdatedMarketplaces = selectedMarketplaces
	}

	// Interactive selection for plugins
	if len(outdatedPlugins) > 0 {
		selectedPlugins, err := ui.SelectFromList(
			"\nPlugins with updates available:",
			outdatedPlugins,
		)
		if err != nil {
			return err
		}
		outdatedPlugins = selectedPlugins
	}

	// Check if user selected anything
	if len(outdatedMarketplaces) == 0 && len(outdatedPlugins) == 0 {
		fmt.Println("No updates selected")
		return nil
	}

	// Apply marketplace updates
	if len(outdatedMarketplaces) > 0 {
		fmt.Println("\n━━━ Updating Marketplaces ━━━")
		for _, name := range outdatedMarketplaces {
			if err := updateMarketplace(name, marketplaces[name].InstallLocation); err != nil {
				fmt.Printf("  ✗ %s: %v\n", name, err)
			} else {
				fmt.Printf("  ✓ %s: Updated\n", name)
			}
		}
	}

	// Apply plugin updates
	if len(outdatedPlugins) > 0 {
		fmt.Println("\n━━━ Updating Plugins ━━━")
		for _, name := range outdatedPlugins {
			if err := updatePlugin(name, plugins); err != nil {
				fmt.Printf("  ✗ %s: %v\n", name, err)
			} else {
				fmt.Printf("  ✓ %s: Updated\n", name)
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
	plugin, exists := plugins.Plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found")
	}

	// Find marketplace path from plugin install path
	var marketplacePath string
	parts := strings.Split(plugin.InstallPath, string(filepath.Separator))
	for i, part := range parts {
		if part == "marketplaces" && i+1 < len(parts) {
			marketplacePath = strings.Join(parts[:i+2], string(filepath.Separator))
			break
		}
	}

	if marketplacePath == "" {
		return fmt.Errorf("marketplace not found in path")
	}

	// Get latest commit from marketplace
	cmd := exec.Command("git", "-C", marketplacePath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get latest commit: %w", err)
	}
	latestCommit := strings.TrimSpace(string(output))

	// For cached plugins (isLocal: false), re-copy from marketplace to cache
	if !plugin.IsLocal {
		// Extract plugin name from full name (e.g., "hookify@claude-code-plugins" -> "hookify")
		pluginBaseName := strings.Split(name, "@")[0]

		// Find source plugin in marketplace (try /plugins/ and /skills/ subdirectories)
		var sourcePath string
		possiblePaths := []string{
			filepath.Join(marketplacePath, "plugins", pluginBaseName),
			filepath.Join(marketplacePath, "skills", pluginBaseName),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				sourcePath = path
				break
			}
		}

		if sourcePath == "" {
			return fmt.Errorf("plugin source not found in marketplace")
		}

		// Remove old cached version
		if err := os.RemoveAll(plugin.InstallPath); err != nil {
			return fmt.Errorf("failed to remove old cached plugin: %w", err)
		}

		// Copy updated plugin to cache
		if err := copyDir(sourcePath, plugin.InstallPath); err != nil {
			return fmt.Errorf("failed to copy updated plugin: %w", err)
		}
	}

	// Update the gitCommitSha
	plugin.GitCommitSha = latestCommit
	plugins.Plugins[name] = plugin

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Get source file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, srcInfo.Mode())
}
