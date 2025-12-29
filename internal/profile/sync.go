// ABOUTME: Syncs plugins from .claudeup.json for team members
// ABOUTME: Installs missing plugins at project scope from project configuration
package profile

import (
	"fmt"
	"strings"

	"github.com/claudeup/claudeup/internal/claude"
)

// SyncResult contains the results of syncing from .claudeup.json
type SyncResult struct {
	MarketplacesAdded int
	PluginsInstalled  int
	PluginsSkipped    int
	Errors            []error
}

// SyncOptions controls sync behavior
type SyncOptions struct {
	DryRun bool
}

// Sync installs plugins from .claudeup.json at project scope
func Sync(profilesDir, projectDir, claudeDir string, opts SyncOptions) (*SyncResult, error) {
	return SyncWithExecutor(profilesDir, projectDir, claudeDir, opts, &DefaultExecutor{ClaudeDir: claudeDir})
}

// SyncWithExecutor syncs using the provided executor
func SyncWithExecutor(profilesDir, projectDir, claudeDir string, opts SyncOptions, executor CommandExecutor) (*SyncResult, error) {
	if profilesDir == "" {
		return nil, fmt.Errorf("profiles directory not specified")
	}

	// Load .claudeup.json
	cfg, err := LoadProjectConfig(projectDir)
	if err != nil {
		return nil, fmt.Errorf("no %s found: %w", ProjectConfigFile, err)
	}

	// Load the profile - check project first, then user profiles
	prof, _, err := LoadWithFallback(profilesDir, projectDir, cfg.Profile)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %q: %w", cfg.Profile, err)
	}

	result := &SyncResult{}

	if opts.DryRun {
		return dryRunSync(prof, claudeDir)
	}

	// 1. Add marketplaces (user-level, idempotent)
	for _, m := range prof.Marketplaces {
		key := marketplaceKey(m)
		if key == "" {
			continue
		}
		output, err := executor.RunWithOutput("plugin", "marketplace", "add", key)
		if err != nil {
			// Marketplace may already exist, not an error
			if !strings.Contains(output, "already") {
				result.Errors = append(result.Errors, fmt.Errorf("marketplace %s: %w", key, err))
			}
			continue
		}
		result.MarketplacesAdded++
	}

	// 2. Get currently installed plugins
	installedPlugins := getInstalledPluginsFromDir(claudeDir)

	// 3. Install plugins with project scope
	for _, plugin := range prof.Plugins {
		if installedPlugins[plugin] {
			result.PluginsSkipped++
			continue
		}

		output, err := executor.RunWithOutput("plugin", "install", "--scope", "project", plugin)
		if err != nil {
			if strings.Contains(output, "already installed") {
				result.PluginsSkipped++
			} else {
				result.Errors = append(result.Errors, fmt.Errorf("plugin %s: %w", plugin, err))
			}
		} else {
			result.PluginsInstalled++
		}
	}

	return result, nil
}

func dryRunSync(prof *Profile, claudeDir string) (*SyncResult, error) {
	result := &SyncResult{}
	installedPlugins := getInstalledPluginsFromDir(claudeDir)

	for _, plugin := range prof.Plugins {
		if installedPlugins[plugin] {
			result.PluginsSkipped++
		} else {
			result.PluginsInstalled++
		}
	}

	result.MarketplacesAdded = len(prof.Marketplaces)
	return result, nil
}

func getInstalledPluginsFromDir(claudeDir string) map[string]bool {
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return make(map[string]bool)
	}

	result := make(map[string]bool)
	for name := range plugins.GetAllPlugins() {
		result[name] = true
	}
	return result
}
