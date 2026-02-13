// ABOUTME: Upgrade command for updating marketplaces and plugins
// ABOUTME: Checks git repos for updates and applies them
package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/ui"
	"github.com/spf13/cobra"
)

const gitTimeout = 30 * time.Second

// truncateHash safely truncates a git commit hash to 7 characters
func truncateHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [targets...]",
	Short: "Update marketplaces and plugins to latest versions",
	Long:  `Update installed marketplaces and plugins to their latest versions.`,
	Example: `  # Upgrade all outdated marketplaces and plugins
  claudeup upgrade

  # Upgrade a specific marketplace
  claudeup upgrade superpowers-marketplace

  # Upgrade a specific plugin
  claudeup upgrade hookify@claude-code-plugins`,
	Args: cobra.ArbitraryArgs,
	RunE: runUpgrade,
}

var upgradeAll bool

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&upgradeAll, "all", false, "Upgrade plugins across all scopes, not just the current context")
}

func availableScopes(allFlag bool) []string {
	if allFlag {
		return claude.ValidScopes
	}
	projectDir, err := os.Getwd()
	if err != nil {
		return []string{"user"}
	}
	scopes := []string{"user"}
	if claude.IsProjectContext(claudeDir, projectDir) {
		scopes = append(scopes, "project", "local")
	}
	return scopes
}

// MarketplaceUpdate represents the update status of an installed marketplace.
// HasUpdate indicates whether a newer version is available on the remote.
type MarketplaceUpdate struct {
	Name          string
	HasUpdate     bool
	CurrentCommit string
	LatestCommit  string
}

// PluginUpdate represents the update status of an installed plugin.
// HasUpdate indicates whether the plugin's source marketplace has newer commits.
type PluginUpdate struct {
	Name          string
	Scope         string
	HasUpdate     bool
	CurrentCommit string
	LatestCommit  string
}

// parseUpgradeTargets separates positional args into marketplaces and plugins
// Plugins contain '@' (e.g., "hookify@plugins"), marketplaces don't
func parseUpgradeTargets(args []string) (marketplaces, plugins []string) {
	for _, arg := range args {
		if strings.Contains(arg, "@") {
			plugins = append(plugins, arg)
		} else {
			marketplaces = append(marketplaces, arg)
		}
	}
	return
}

// findUnmatchedTargets returns targets that don't match any known marketplace or plugin
func findUnmatchedTargets(targetMarketplaces, targetPlugins []string, marketplaceUpdates []MarketplaceUpdate, pluginUpdates []PluginUpdate) []string {
	var unmatched []string

	// Check marketplace targets
	for _, target := range targetMarketplaces {
		found := false
		for _, update := range marketplaceUpdates {
			if update.Name == target {
				found = true
				break
			}
		}
		if !found {
			unmatched = append(unmatched, target)
		}
	}

	// Check plugin targets
	for _, target := range targetPlugins {
		found := false
		for _, update := range pluginUpdates {
			if update.Name == target {
				found = true
				break
			}
		}
		if !found {
			unmatched = append(unmatched, target)
		}
	}

	return unmatched
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	ui.PrintInfo("Checking for updates...")

	// Parse target filters (if any)
	targetMarketplaces, targetPlugins := parseUpgradeTargets(args)
	hasTargets := len(targetMarketplaces) > 0 || len(targetPlugins) > 0

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
	fmt.Println(ui.RenderSection("Checking Marketplaces", len(marketplaces)))
	marketplaceUpdates := checkMarketplaceUpdates(marketplaces)

	var outdatedMarketplaces []string
	for _, update := range marketplaceUpdates {
		if update.HasUpdate {
			// Filter by target if specified
			if hasTargets {
				if len(targetMarketplaces) > 0 {
					found := false
					for _, target := range targetMarketplaces {
						if target == update.Name {
							found = true
							break
						}
					}
					if !found {
						fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), update.Name, ui.Warning("Update available (skipped)"))
						continue
					}
				} else {
					// User specified plugins only, skip marketplaces
					fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), update.Name, ui.Warning("Update available (skipped)"))
					continue
				}
			}
			fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), update.Name, ui.Warning("Update available"))
			outdatedMarketplaces = append(outdatedMarketplaces, update.Name)
		} else {
			fmt.Printf("  %s %s: %s\n", ui.Success(ui.SymbolSuccess), update.Name, ui.Muted("Up to date"))
		}
	}

	// Check plugin updates
	fmt.Println()
	scopes := availableScopes(upgradeAll)
	scopedPlugins := plugins.GetPluginsAtScopes(scopes)
	fmt.Println(ui.RenderSection("Checking Plugins", len(scopedPlugins)))
	pluginUpdates := checkPluginUpdates(plugins, marketplaces, scopes)

	var outdatedUpdates []PluginUpdate
	for _, update := range pluginUpdates {
		if update.HasUpdate {
			displayName := fmt.Sprintf("%s (%s)", update.Name, update.Scope)
			// Filter by target if specified.
			// Targeting a plugin by name upgrades it at all scopes where it's outdated.
			if hasTargets {
				if len(targetPlugins) > 0 {
					found := false
					for _, target := range targetPlugins {
						if target == update.Name {
							found = true
							break
						}
					}
					if !found {
						fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), displayName, ui.Warning("Update available (skipped)"))
						continue
					}
				} else {
					// User specified marketplaces only, skip plugins
					fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), displayName, ui.Warning("Update available (skipped)"))
					continue
				}
			}
			fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), displayName, ui.Warning("Update available"))
			outdatedUpdates = append(outdatedUpdates, update)
		}
	}

	if len(outdatedUpdates) == 0 {
		fmt.Printf("  %s All plugins up to date\n", ui.Success(ui.SymbolSuccess))
	}

	// Warn about unmatched targets
	if hasTargets {
		unmatchedTargets := findUnmatchedTargets(targetMarketplaces, targetPlugins, marketplaceUpdates, pluginUpdates)
		if len(unmatchedTargets) > 0 {
			fmt.Println()
			for _, target := range unmatchedTargets {
				ui.PrintWarning(fmt.Sprintf("Unknown target: %s", target))
			}
		}
	}

	// If nothing outdated, we're done
	if len(outdatedMarketplaces) == 0 && len(outdatedUpdates) == 0 {
		fmt.Println()
		ui.PrintSuccess("Everything is up to date!")
		return nil
	}

	// Apply all marketplace updates
	if len(outdatedMarketplaces) > 0 {
		fmt.Println()
		fmt.Println(ui.RenderSection("Updating Marketplaces", len(outdatedMarketplaces)))
		for _, name := range outdatedMarketplaces {
			if err := updateMarketplace(name, marketplaces[name].InstallLocation); err != nil {
				ui.PrintError(fmt.Sprintf("%s: %v", name, err))
			} else {
				ui.PrintSuccess(fmt.Sprintf("%s: Updated", name))
			}
		}
	}

	// Apply plugin updates
	if len(outdatedUpdates) > 0 {
		fmt.Println()
		fmt.Println(ui.RenderSection("Updating Plugins", len(outdatedUpdates)))
		for _, update := range outdatedUpdates {
			displayName := fmt.Sprintf("%s (%s)", update.Name, update.Scope)
			if err := updatePlugin(update.Name, update.Scope, plugins); err != nil {
				ui.PrintError(fmt.Sprintf("%s: %v", displayName, err))
			} else {
				ui.PrintSuccess(fmt.Sprintf("%s: Updated", displayName))
			}
		}

		// Save updated plugin registry
		if err := claude.SavePlugins(claudeDir, plugins); err != nil {
			return fmt.Errorf("failed to save plugins: %w", err)
		}
	}

	fmt.Println()
	ui.PrintSuccess("Updates complete!")

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

		// Fetch from remote with timeout
		ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
		fetchCmd := exec.CommandContext(ctx, "git", "-C", marketplace.InstallLocation, "fetch", "origin")
		fetchErr := fetchCmd.Run()
		cancel()
		if fetchErr != nil {
			// Fetch failed - warn user and mark as unable to check
			fmt.Fprintf(os.Stderr, "  %s %s: %s (git fetch failed)\n", ui.Warning(ui.SymbolWarning), name, ui.Muted("Unable to check for updates"))
			updates = append(updates, MarketplaceUpdate{
				Name:      name,
				HasUpdate: false,
			})
			continue
		}

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
			CurrentCommit: truncateHash(currentCommit),
			LatestCommit:  truncateHash(remoteCommit),
		})
	}

	return updates
}

func checkPluginUpdates(plugins *claude.PluginRegistry, marketplaces claude.MarketplaceRegistry, scopes []string) []PluginUpdate {
	var updates []PluginUpdate

	for _, sp := range plugins.GetPluginsAtScopes(scopes) {
		name := sp.Name
		plugin := sp.PluginMetadata

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
				Scope:         plugin.Scope,
				HasUpdate:     true,
				CurrentCommit: truncateHash(plugin.GitCommitSha),
				LatestCommit:  truncateHash(currentCommit),
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

func updatePlugin(name string, scope string, plugins *claude.PluginRegistry) error {
	plugin, exists := plugins.GetPluginAtScope(name, scope)
	if !exists {
		return fmt.Errorf("plugin not found at scope %s", scope)
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

		// Sanitize: prevent path traversal attacks
		if strings.Contains(pluginBaseName, "..") || strings.Contains(pluginBaseName, string(filepath.Separator)) {
			return fmt.Errorf("invalid plugin name: %s", pluginBaseName)
		}

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
	plugins.SetPlugin(name, plugin)

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
