// ABOUTME: Syncs configuration from .claudeup.json for team members
// ABOUTME: Applies profile settings to all scopes, creating local profile copy
package profile

import (
	"fmt"

	"github.com/claudeup/claudeup/v2/internal/secrets"
)

// SyncResult contains the results of syncing from .claudeup.json
type SyncResult struct {
	ProfileName      string
	ProfileCreated   bool // True if profile was created/updated locally
	PluginsInstalled int
	PluginsSkipped   int
	Errors           []error
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
// It saves a local copy of the profile and applies settings:
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
		return nil, fmt.Errorf("failed to load profile %q: %w", cfg.Profile, err)
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

	// Apply the profile to all scopes using ApplyAllScopes
	applyOpts := &ApplyAllScopesOptions{
		ReplaceUserScope: opts.ReplaceUserScope,
	}

	var chain *secrets.Chain // nil chain - no secret resolution during sync
	applyResult, err := ApplyAllScopes(prof, claudeDir, claudeJSONPath, projectDir, chain, applyOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to apply profile: %w", err)
	}

	// Aggregate results
	result.PluginsInstalled = len(applyResult.PluginsInstalled)
	result.Errors = append(result.Errors, applyResult.Errors...)

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
	return result, nil
}
