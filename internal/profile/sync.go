// ABOUTME: Syncs configuration from .claudeup.json for team members
// ABOUTME: Installs plugins and applies profile settings to all scopes
package profile

import (
	"fmt"
	"strings"
)

// SyncResult contains the results of syncing from .claudeup.json
type SyncResult struct {
	ProfileName       string
	ProfileCreated    bool // True if profile was created/updated locally
	PluginsInstalled  int
	PluginsSkipped    int
	MarketplacesAdded []string
	Warnings          []string // Non-fatal issues (e.g., embedded marketplace failures)
	Errors            []error
}

// ProgressCallback reports installation progress for multi-item operations
type ProgressCallback func(current, total int, item string)

// SyncOptions controls sync behavior
type SyncOptions struct {
	DryRun           bool
	ReplaceUserScope bool             // If true, replace user scope; if false, additive (default)
	Progress         ProgressCallback // Optional progress reporting
}

// Sync applies the profile from .claudeup.json to all scopes.
// It saves a local copy of the profile, installs plugins, and applies settings:
// - User scope: additive by default, declarative with ReplaceUserScope=true
// - Project/local scopes: always declarative (replaces settings)
func Sync(profilesDir, projectDir, claudeDir, claudeJSONPath string, opts SyncOptions) (*SyncResult, error) {
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
		// Profile doesn't exist - bootstrap by creating from current state
		// This handles the case where:
		// 1. Project was set up with an older version that didn't save to .claudeup/profiles/
		// 2. User is syncing for the first time without the profile
		prof, err = SnapshotAllScopes(cfg.Profile, claudeDir, claudeJSONPath, projectDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile %q from current state: %w", cfg.Profile, err)
		}
	}

	result := &SyncResult{
		ProfileName: prof.Name,
	}

	if opts.DryRun {
		return dryRunSync(prof, claudeDir, projectDir)
	}

	// Save/update a local copy of the profile to user's profiles directory
	if err := Save(profilesDir, prof); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to save local profile copy: %w", err))
	} else {
		result.ProfileCreated = true
	}

	// Create executor for running claude CLI commands
	executor := &DefaultExecutor{ClaudeDir: claudeDir}

	// 1. Add marketplaces from embedded profiles first (best-effort)
	// This ensures known marketplaces (like wshobson/agents from hobson profile) are available
	embeddedMarketplaces := collectEmbeddedMarketplaces()
	for _, m := range embeddedMarketplaces {
		key := marketplaceKey(m)
		if key == "" {
			continue
		}
		output, err := executor.RunWithOutput("plugin", "marketplace", "add", key)
		if err == nil {
			result.MarketplacesAdded = append(result.MarketplacesAdded, key)
		} else if strings.Contains(output, "already installed") {
			// Already installed is fine, just don't add to MarketplacesAdded
		} else {
			// Track as warning - these are best-effort additions, not fatal errors
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("embedded marketplace %s: %v", key, err))
		}
	}

	// 2. Add marketplaces from the profile (needed to resolve plugin names)
	for _, m := range prof.Marketplaces {
		key := marketplaceKey(m)
		if key == "" {
			continue
		}
		output, err := executor.RunWithOutput("plugin", "marketplace", "add", key)
		if err != nil {
			// Check if already installed - treat as success
			if strings.Contains(output, "already installed") {
				result.MarketplacesAdded = append(result.MarketplacesAdded, key)
			} else {
				result.Errors = append(result.Errors, fmt.Errorf("marketplace %s: %w", key, err))
			}
		} else {
			result.MarketplacesAdded = append(result.MarketplacesAdded, key)
		}
	}

	// 3. Install plugins and write settings for each scope
	// IMPORTANT: Decouple installation from settings write so that empty plugin
	// lists still write settings (clearing stale plugins in declarative mode)
	if prof.IsMultiScope() {
		// Multi-scope profile: install plugins at each scope
		if prof.PerScope.User != nil {
			// Install plugins only if there are any
			if len(prof.PerScope.User.Plugins) > 0 {
				installResult := InstallPluginsWithProgress(prof.PerScope.User.Plugins, executor, InstallPluginsOptions{
					Scope:    "", // User scope (no --scope flag)
					Progress: opts.Progress,
				})
				result.PluginsInstalled += len(installResult.Installed)
				result.PluginsSkipped += len(installResult.Skipped)
				result.Errors = append(result.Errors, installResult.Errors...)
			}

			// Always write user settings (even for empty plugin list, to clear stale state)
			if _, err := applyUserScopeSettings(prof.ForScope("user"), claudeDir, projectDir, opts.ReplaceUserScope); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to write user settings: %w", err))
			}
		}

		if prof.PerScope.Project != nil && projectDir != "" {
			// Install plugins only if there are any
			if len(prof.PerScope.Project.Plugins) > 0 {
				installResult := InstallPluginsWithProgress(prof.PerScope.Project.Plugins, executor, InstallPluginsOptions{
					Scope:    "project",
					Progress: opts.Progress,
				})
				result.PluginsInstalled += len(installResult.Installed)
				result.PluginsSkipped += len(installResult.Skipped)
				result.Errors = append(result.Errors, installResult.Errors...)
			}

			// Always write project settings (project scope is always declarative)
			if _, err := applyProjectScopeSettings(prof.ForScope("project"), claudeDir, projectDir); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to write project settings: %w", err))
			}
		}

		if prof.PerScope.Local != nil && projectDir != "" {
			// Install plugins only if there are any
			if len(prof.PerScope.Local.Plugins) > 0 {
				installResult := InstallPluginsWithProgress(prof.PerScope.Local.Plugins, executor, InstallPluginsOptions{
					Scope:    "local",
					Progress: opts.Progress,
				})
				result.PluginsInstalled += len(installResult.Installed)
				result.PluginsSkipped += len(installResult.Skipped)
				result.Errors = append(result.Errors, installResult.Errors...)
			}

			// Always write local settings (local scope is always declarative)
			if _, err := applyLocalScopeSettings(prof.ForScope("local"), claudeDir, projectDir); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to write local settings: %w", err))
			}
		}
	} else {
		// Legacy single-scope profile: install at user scope
		// Install plugins only if there are any
		if len(prof.Plugins) > 0 {
			installResult := InstallPluginsWithProgress(prof.Plugins, executor, InstallPluginsOptions{
				Scope:    "", // User scope
				Progress: opts.Progress,
			})
			result.PluginsInstalled += len(installResult.Installed)
			result.PluginsSkipped += len(installResult.Skipped)
			result.Errors = append(result.Errors, installResult.Errors...)
		}

		// Write settings if there are plugins OR if in replace mode (to clear stale plugins)
		if len(prof.Plugins) > 0 || opts.ReplaceUserScope {
			if _, err := applyUserScopeSettings(prof, claudeDir, projectDir, opts.ReplaceUserScope); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to write user settings: %w", err))
			}
		}
	}

	// Warn if profile has MCP servers with secrets that won't be resolved
	if prof.HasMCPServersWithSecrets() {
		result.Errors = append(result.Errors, fmt.Errorf(
			"profile contains MCP servers with secrets; secrets will not be resolved during sync. "+
				"Use 'claudeup profile apply %s' with --secrets flag for secret resolution", prof.Name))
	}

	return result, nil
}

func dryRunSync(prof *Profile, claudeDir, projectDir string) (*SyncResult, error) {
	result := &SyncResult{
		ProfileName:    prof.Name,
		ProfileCreated: true, // Would create/update
	}

	// Count plugins that would be installed
	pluginCount := 0
	if prof.IsMultiScope() {
		if prof.PerScope.User != nil {
			pluginCount += len(prof.PerScope.User.Plugins)
		}
		if prof.PerScope.Project != nil {
			pluginCount += len(prof.PerScope.Project.Plugins)
		}
		if prof.PerScope.Local != nil {
			pluginCount += len(prof.PerScope.Local.Plugins)
		}
	} else {
		pluginCount = len(prof.Plugins)
	}

	result.PluginsInstalled = pluginCount

	// Count marketplaces
	for _, m := range prof.Marketplaces {
		key := marketplaceKey(m)
		if key != "" {
			result.MarketplacesAdded = append(result.MarketplacesAdded, key)
		}
	}

	return result, nil
}

// collectEmbeddedMarketplaces returns all unique marketplaces from embedded profiles
func collectEmbeddedMarketplaces() []Marketplace {
	embeddedProfiles, err := ListEmbeddedProfiles()
	if err != nil {
		return nil
	}

	// Use a map to deduplicate marketplaces by repo/URL
	seen := make(map[string]bool)
	var marketplaces []Marketplace

	for _, p := range embeddedProfiles {
		for _, m := range p.Marketplaces {
			key := marketplaceKey(m)
			if key != "" && !seen[key] {
				seen[key] = true
				marketplaces = append(marketplaces, m)
			}
		}
	}

	return marketplaces
}
