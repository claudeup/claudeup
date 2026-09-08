// ABOUTME: Upgrade command for updating marketplaces and plugins
// ABOUTME: Checks git repos for updates and applies them
package commands

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
  claudeup upgrade hookify@claude-plugins-official`,
	Args: cobra.ArbitraryArgs,
	RunE: runUpgrade,
}

var upgradeAll bool

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&upgradeAll, "all", false, "Upgrade plugins across all scopes, not just the current context")
}

func availableScopes(allFlag bool, projectDir string) []string {
	if allFlag {
		return claude.ValidScopes
	}
	scopes := []string{"user"}
	if projectDir != "" && claude.IsProjectContext(claudeDir, projectDir) {
		scopes = append(scopes, "project", "local")
	}
	return scopes
}

// MarketplaceUpdate represents the update status of an installed marketplace.
// HasUpdate indicates whether a newer version is available on the remote.
type MarketplaceUpdate struct {
	Name          string
	HasUpdate     bool
	CheckFailed   bool
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

	// Check and apply marketplace updates first, so plugin checks see current HEAD
	fmt.Println()
	fmt.Println(ui.RenderSection("Checking Marketplaces", len(marketplaces)))
	var outdatedMarketplaces []string
	marketplaceUpdates := checkMarketplaceUpdates(marketplaces, func(update MarketplaceUpdate) {
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
						return
					}
				} else {
					// User specified plugins only, skip marketplaces
					fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), update.Name, ui.Warning("Update available (skipped)"))
					return
				}
			}
			fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), update.Name, ui.Warning("Update available"))
			outdatedMarketplaces = append(outdatedMarketplaces, update.Name)
		} else if update.CheckFailed {
			fmt.Printf("  %s %s: %s\n", ui.Warning(ui.SymbolWarning), update.Name, ui.Muted("Unable to check for updates"))
		} else {
			fmt.Printf("  %s %s: %s\n", ui.Success(ui.SymbolSuccess), update.Name, ui.Muted("Up to date"))
		}
	})

	// Apply marketplace updates before checking plugins.
	// Plugin update detection compares against the marketplace's local HEAD,
	// so marketplaces must be pulled first for plugins to see the latest commits.
	//
	// When the user targets specific plugins, their parent marketplaces must
	// still be pulled even though they aren't in the display list.
	marketplacesToPull := outdatedMarketplaces
	if hasTargets && len(targetPlugins) > 0 {
		needed := make(map[string]bool)
		for _, name := range outdatedMarketplaces {
			needed[name] = true
		}
		for _, target := range targetPlugins {
			if parts := strings.SplitN(target, "@", 2); len(parts) == 2 {
				if _, ok := marketplaces[parts[1]]; ok && !needed[parts[1]] {
					for _, update := range marketplaceUpdates {
						if update.Name == parts[1] && update.HasUpdate {
							marketplacesToPull = append(marketplacesToPull, parts[1])
							needed[parts[1]] = true
						}
					}
				}
			}
		}
	}
	attempted, failures := 0, 0
	if len(marketplacesToPull) > 0 {
		fmt.Println()
		fmt.Println(ui.RenderSection("Updating Marketplaces", len(marketplacesToPull)))
		for _, name := range marketplacesToPull {
			attempted++
			if err := updateMarketplace(name, marketplaces[name].InstallLocation); err != nil {
				ui.PrintError(fmt.Sprintf("%s: %v", name, err))
				failures++
			} else {
				ui.PrintSuccess(fmt.Sprintf("%s: Updated", name))
			}
		}
	}

	// Check plugin updates (after marketplace pull so HEAD is current)
	fmt.Println()
	projectDir := ""
	if !upgradeAll {
		if dir, err := os.Getwd(); err == nil {
			projectDir = dir
		}
	}
	scopes := availableScopes(upgradeAll, projectDir)
	scopedPlugins := plugins.GetPluginsForContext(scopes, projectDir)
	fmt.Println(ui.RenderSection("Checking Plugins", len(scopedPlugins)))
	pluginUpdates := checkPluginUpdates(scopedPlugins, marketplaces)

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

	// If nothing was outdated, we're done
	if len(outdatedMarketplaces) == 0 && len(outdatedUpdates) == 0 {
		fmt.Println()
		ui.PrintSuccess("Everything is up to date!")
		return nil
	}

	// Apply plugin updates
	if len(outdatedUpdates) > 0 {
		fmt.Println()
		fmt.Println(ui.RenderSection("Updating Plugins", len(outdatedUpdates)))
		for _, update := range outdatedUpdates {
			displayName := fmt.Sprintf("%s (%s)", update.Name, update.Scope)
			attempted++
			err := updatePlugin(claudeDir, update.Name, update.Scope, plugins, marketplaces)
			switch {
			case err == nil:
				ui.PrintSuccess(fmt.Sprintf("%s: Updated", displayName))
			case isStalePluginError(err):
				ui.PrintWarning(fmt.Sprintf("%s: %v", displayName, err))
				confirmed, promptErr := ui.ConfirmYesNo(
					fmt.Sprintf("  Remove stale registry entry for %s?", displayName))
				switch {
				case promptErr != nil:
					// Leaving the entry in place without saying so hides the
					// stale plugin from the next run's summary too.
					ui.PrintError(fmt.Sprintf("%s: cannot ask whether to remove the stale entry: %v", displayName, promptErr))
					failures++
				case confirmed:
					plugins.RemovePluginAtScope(update.Name, update.Scope)
					ui.PrintSuccess(fmt.Sprintf("%s: Removed stale entry", displayName))
				default:
					ui.PrintWarning(fmt.Sprintf("%s: Kept stale entry", displayName))
				}
			default:
				ui.PrintError(fmt.Sprintf("%s: %v", displayName, err))
				failures++
			}
		}

		// Save updated plugin registry
		if err := claude.SavePlugins(claudeDir, plugins); err != nil {
			return fmt.Errorf("failed to save plugins: %w", err)
		}
	}

	fmt.Println()
	if failures > 0 {
		return fmt.Errorf("%d of %d updates failed", failures, attempted)
	}
	ui.PrintSuccess("Updates complete!")

	return nil
}

func checkMarketplaceUpdates(marketplaces claude.MarketplaceRegistry, onResult func(MarketplaceUpdate)) []MarketplaceUpdate {
	var updates []MarketplaceUpdate

	for name, marketplace := range marketplaces {
		var update MarketplaceUpdate

		// Fetch latest from remote
		gitDir := filepath.Join(marketplace.InstallLocation, ".git")
		if _, err := os.Stat(gitDir); errors.Is(err, fs.ErrNotExist) {
			// Not a git repo, skip
			update = MarketplaceUpdate{
				Name:      name,
				HasUpdate: false,
			}
			updates = append(updates, update)
			if onResult != nil {
				onResult(update)
			}
			continue
		}

		// Get current commit
		currentCmd := exec.Command("git", "-C", marketplace.InstallLocation, "rev-parse", "HEAD")
		currentOutput, err := currentCmd.Output()
		if err != nil {
			update = MarketplaceUpdate{
				Name:      name,
				HasUpdate: false,
			}
			updates = append(updates, update)
			if onResult != nil {
				onResult(update)
			}
			continue
		}
		currentCommit := strings.TrimSpace(string(currentOutput))

		// Fetch from remote with timeout
		ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
		fetchCmd := exec.CommandContext(ctx, "git", "-C", marketplace.InstallLocation, "fetch", "origin")
		fetchErr := fetchCmd.Run()
		cancel()
		if fetchErr != nil {
			update = MarketplaceUpdate{
				Name:        name,
				CheckFailed: true,
			}
			updates = append(updates, update)
			if onResult != nil {
				onResult(update)
			}
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
					update = MarketplaceUpdate{
						Name:      name,
						HasUpdate: false,
					}
					updates = append(updates, update)
					if onResult != nil {
						onResult(update)
					}
					continue
				}
			}
		}
		remoteCommit := strings.TrimSpace(string(remoteOutput))

		// Only flag as update available when the remote has commits not in local.
		// A simple != check falsely triggers when local is ahead of remote.
		hasUpdate := false
		if currentCommit != remoteCommit {
			ancestorCmd := exec.Command("git", "-C", marketplace.InstallLocation, "merge-base", "--is-ancestor", remoteCommit, currentCommit)
			hasUpdate = ancestorCmd.Run() != nil
		}

		update = MarketplaceUpdate{
			Name:          name,
			HasUpdate:     hasUpdate,
			CurrentCommit: truncateHash(currentCommit),
			LatestCommit:  truncateHash(remoteCommit),
		}
		updates = append(updates, update)
		if onResult != nil {
			onResult(update)
		}
	}

	return updates
}

// findMarketplacePath resolves the marketplace install location for a plugin.
// It first tries to extract the marketplace name from the plugin name (format: "plugin@marketplace"),
// then falls back to checking if the plugin's install path is inside a marketplace directory.
// Returns empty string when neither lookup matches.
func findMarketplacePath(pluginName string, installPath string, marketplaces claude.MarketplaceRegistry) string {
	if parts := strings.SplitN(pluginName, "@", 2); len(parts) == 2 {
		if marketplace, ok := marketplaces[parts[1]]; ok {
			return marketplace.InstallLocation
		}
	}
	for _, marketplace := range marketplaces {
		prefix := marketplace.InstallLocation + string(filepath.Separator)
		if strings.HasPrefix(installPath, prefix) {
			return marketplace.InstallLocation
		}
	}
	return ""
}

func checkPluginUpdates(scopedPlugins []claude.ScopedPlugin, marketplaces claude.MarketplaceRegistry) []PluginUpdate {
	var updates []PluginUpdate

	// Cache marketplace indexes to avoid re-reading for each plugin
	indexCache := make(map[string]*claude.MarketplaceIndex)

	for _, sp := range scopedPlugins {
		name := sp.Name
		plugin := sp.PluginMetadata

		// Skip if plugin path doesn't exist
		if !plugin.PathExists() {
			updates = append(updates, PluginUpdate{Name: name, Scope: plugin.Scope})
			continue
		}

		marketplacePath := findMarketplacePath(name, plugin.InstallPath, marketplaces)
		if marketplacePath == "" {
			updates = append(updates, PluginUpdate{Name: name, Scope: plugin.Scope})
			continue
		}

		// Get current commit from marketplace
		gitDir := filepath.Join(marketplacePath, ".git")
		if _, err := os.Stat(gitDir); errors.Is(err, fs.ErrNotExist) {
			updates = append(updates, PluginUpdate{Name: name, Scope: plugin.Scope})
			continue
		}

		currentCmd := exec.Command("git", "-C", marketplacePath, "rev-parse", "HEAD")
		currentOutput, err := currentCmd.Output()
		if err != nil {
			updates = append(updates, PluginUpdate{Name: name, Scope: plugin.Scope})
			continue
		}
		currentCommit := strings.TrimSpace(string(currentOutput))

		declaredVersion, external := marketplaceIndexEntry(marketplacePath, name, indexCache)

		var hasUpdate bool
		if external && declaredVersion != "" {
			// The plugin's content lives in another repository, so a commit to
			// the marketplace is no evidence about the plugin. The version the
			// marketplace declares is the only signal available locally.
			hasUpdate = declaredVersion != plugin.Version
		} else {
			// The plugin is a directory in the marketplace checkout, so the
			// marketplace commit is the plugin's own history. This also covers an
			// external plugin whose marketplace declares no version, where the
			// commit is a coarse proxy but the only thing to go on.
			hasUpdate = plugin.GitCommitSha != currentCommit
			if !hasUpdate && declaredVersion != "" {
				hasUpdate = declaredVersion != plugin.Version
			}
		}

		updates = append(updates, PluginUpdate{
			Name:          name,
			Scope:         plugin.Scope,
			HasUpdate:     hasUpdate,
			CurrentCommit: truncateHash(plugin.GitCommitSha),
			LatestCommit:  truncateHash(currentCommit),
		})
	}

	return updates
}

// marketplaceIndexEntry looks up what the marketplace index declares about a
// plugin: the version it publishes, and whether the plugin is fetched from
// elsewhere rather than living in the marketplace checkout. A missing index or
// a plugin the index does not list yields an empty version and reports the
// plugin as local, since the marketplace commit is then all there is.
func marketplaceIndexEntry(marketplacePath, qualifiedName string, cache map[string]*claude.MarketplaceIndex) (version string, external bool) {
	index, ok := cache[marketplacePath]
	if !ok {
		loaded, err := claude.LoadMarketplaceIndex(marketplacePath)
		if err != nil {
			cache[marketplacePath] = nil
			return "", false
		}
		index = loaded
		cache[marketplacePath] = index
	}
	if index == nil {
		return "", false
	}

	baseName := qualifiedName
	if parts := strings.SplitN(qualifiedName, "@", 2); len(parts) == 2 {
		baseName = parts[0]
	}
	for _, p := range index.Plugins {
		if p.Name == baseName {
			return p.Version, p.Source != nil && !p.Source.IsRelativePath()
		}
	}
	return "", false
}

func updateMarketplace(name, path string) error {
	// Git pull to update
	cmd := exec.Command("git", "-C", path, "pull", "--ff-only")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}
	return nil
}

func updatePlugin(claudeDir string, name string, scope string, plugins *claude.PluginRegistry, marketplaces claude.MarketplaceRegistry) error {
	plugin, exists := plugins.GetPluginAtScope(name, scope)
	if !exists {
		return fmt.Errorf("plugin not found at scope %s", scope)
	}

	marketplacePath := findMarketplacePath(name, plugin.InstallPath, marketplaces)
	if marketplacePath == "" {
		return fmt.Errorf("marketplace not found for plugin")
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
		pluginBaseName := strings.Split(name, "@")[0]

		// Sanitize: prevent path traversal attacks
		if strings.Contains(pluginBaseName, "..") || strings.Contains(pluginBaseName, string(filepath.Separator)) {
			return fmt.Errorf("invalid plugin name: %s", pluginBaseName)
		}

		sourcePath, newVersion, err := resolvePluginSource(marketplacePath, pluginBaseName)
		if err != nil {
			return err
		}

		if sourcePath == "" {
			// External sources are fetched by Claude Code, not from the
			// marketplace checkout. Delegate to its plugin update command.
			if err := updatePluginViaCLI(name, scope); err != nil {
				return err
			}

			// Claude Code rewrites the registry file itself. Read its entry back
			// rather than patching the copy loaded before delegating, which the
			// save at the end of the run would write over the top of.
			fresh, err := claude.LoadPlugins(claudeDir)
			if err != nil {
				return fmt.Errorf("cannot re-read the plugin registry after updating %s: %w", name, err)
			}
			updated, ok := fresh.GetPluginAtScope(name, scope)
			if !ok {
				return fmt.Errorf("claude plugin update reported success but %s is no longer registered at scope %s", name, scope)
			}

			// A zero exit status is not proof the version landed.
			if newVersion != "" && updated.Version != newVersion {
				return fmt.Errorf("claude plugin update reported success but %s is still at version %s, not %s",
					name, updated.Version, newVersion)
			}

			// The commit claudeup compared against is its own bookkeeping; the
			// rest of the entry belongs to Claude Code.
			updated.GitCommitSha = latestCommit
			plugins.SetPlugin(name, updated)
			return nil
		}

		// Determine cache destination. When the marketplace provides a new version,
		// write to a new versioned directory. The old cache directory is removed either way.
		destPath := plugin.InstallPath
		if newVersion != "" && newVersion != plugin.Version {
			destPath = filepath.Join(filepath.Dir(plugin.InstallPath), newVersion)
		}

		// Remove old cached version
		if err := os.RemoveAll(plugin.InstallPath); err != nil {
			return fmt.Errorf("failed to remove old cached plugin: %w", err)
		}

		// Copy updated plugin to cache
		if err := copyDir(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy updated plugin: %w", err)
		}

		plugin.InstallPath = destPath
		if newVersion != "" {
			plugin.Version = newVersion
		}
	}

	// Update the gitCommitSha
	plugin.GitCommitSha = latestCommit
	plugins.SetPlugin(name, plugin)

	return nil
}

// updatePluginViaCLI delegates plugin updates to Claude Code's `claude plugin update` command.
// This handles externally sourced plugins, which Claude Code fetches from npm or
// from a remote repository rather than from the marketplace checkout.
func updatePluginViaCLI(pluginName, scope string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "plugin", "update", "--scope", scope, pluginName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("claude plugin update timed out after 60s for %s", pluginName)
		}
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return fmt.Errorf("claude plugin update failed: %s", trimmed)
		}
		return fmt.Errorf("claude plugin update failed: %w", err)
	}
	return nil
}

// resolvePluginSource finds the source directory for a plugin within its marketplace.
// The marketplace index decides where a plugin comes from. Only when the index
// cannot be read, or does not list the plugin, does it fall back to the
// conventional plugins/ and skills/ directories. Returns (sourcePath, version, error).
// Returns empty sourcePath (not an error) when the plugin uses an external
// source, signaling the caller to delegate to `claude plugin update`.
func resolvePluginSource(marketplacePath, pluginBaseName string) (string, string, error) {
	index, indexErr := claude.LoadMarketplaceIndex(marketplacePath)

	var pluginInfo *claude.MarketplacePluginInfo
	if indexErr == nil {
		for i := range index.Plugins {
			if index.Plugins[i].Name == pluginBaseName {
				pluginInfo = &index.Plugins[i]
				break
			}
		}
	}

	if pluginInfo == nil {
		// Fall back to the conventional directories. A marketplace may ship no
		// index at all, and one that does may not list every directory it holds.
		for _, subdir := range []string{"plugins", "skills"} {
			p := filepath.Join(marketplacePath, subdir, pluginBaseName)
			_, err := os.Stat(p)
			if err == nil {
				return p, "", nil
			}
			// Only a missing entry means the plugin is not here. An unreadable one
			// would otherwise fall through and resolve from somewhere else.
			if !errors.Is(err, fs.ErrNotExist) {
				return "", "", fmt.Errorf("cannot read %s in marketplace: %w", filepath.Join(subdir, pluginBaseName), err)
			}
		}
		if indexErr != nil {
			return "", "", fmt.Errorf("plugin source not found in marketplace and cannot read index: %w", indexErr)
		}
		return "", "", fmt.Errorf("plugin %q not found in marketplace index", pluginBaseName)
	}

	// A listed plugin with no source is a malformed marketplace, not a registry
	// entry the marketplace has dropped. Keep the two apart so the caller does
	// not offer to delete the user's entry over someone else's bad index.
	if pluginInfo.Source == nil {
		return "", "", fmt.Errorf("plugin %q is listed in the marketplace index without a source", pluginBaseName)
	}

	if pluginInfo.Source.IsRelativePath() {
		// Resolve relative path within marketplace, ensuring it stays within bounds
		resolved := filepath.Join(marketplacePath, pluginInfo.Source.RelativePath)
		resolved = filepath.Clean(resolved)
		cleanMarketplace := filepath.Clean(marketplacePath)
		if resolved != cleanMarketplace && !strings.HasPrefix(resolved, cleanMarketplace+string(filepath.Separator)) {
			return "", "", fmt.Errorf("plugin source %q resolves outside marketplace directory", pluginInfo.Source.RelativePath)
		}
		if _, err := os.Stat(resolved); err != nil {
			return "", "", fmt.Errorf("plugin source path %s does not exist: %w", resolved, err)
		}

		// The comparison above is lexical, so it cannot see a symlink inside the
		// marketplace that points out of it. Compare the real paths as well.
		realResolved, err := filepath.EvalSymlinks(resolved)
		if err != nil {
			return "", "", fmt.Errorf("cannot resolve plugin source path %s: %w", resolved, err)
		}
		realMarketplace, err := filepath.EvalSymlinks(cleanMarketplace)
		if err != nil {
			return "", "", fmt.Errorf("cannot resolve marketplace directory %s: %w", cleanMarketplace, err)
		}
		if realResolved != realMarketplace && !strings.HasPrefix(realResolved, realMarketplace+string(filepath.Separator)) {
			return "", "", fmt.Errorf("plugin source %q resolves outside marketplace directory", pluginInfo.Source.RelativePath)
		}
		return resolved, pluginInfo.Version, nil
	}

	// External source; Claude Code fetches it. Return empty to signal delegation.
	return "", pluginInfo.Version, nil
}

// isStalePluginError returns true when the error indicates a registry entry
// that no longer corresponds to an installed or indexed plugin.
func isStalePluginError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found in marketplace index") ||
		strings.Contains(msg, "not installed at scope")
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
