// ABOUTME: Shared plugin installation logic with progress reporting
// ABOUTME: Used by apply command for DRY compliance

package profile

import (
	"fmt"
	"strings"
)

// ProgressCallback reports installation progress for multi-item operations
type ProgressCallback func(current, total int, item string)

// InstallPluginsResult contains the result of installing plugins
type InstallPluginsResult struct {
	Installed []string
	Skipped   []string // Already installed
	Errors    []error
}

// InstallPluginsOptions configures plugin installation behavior
type InstallPluginsOptions struct {
	// Scope for plugin installation: "", "project", or "local"
	// Empty string means user scope (no --scope flag)
	Scope string

	// InstalledPlugins is an optional map of already-installed plugins.
	// If provided, plugins in this map are skipped before running install.
	// If nil, no pre-filtering occurs (relies on "already installed" output).
	InstalledPlugins map[string]bool

	// Progress is an optional callback for reporting installation progress.
	// Called with (current, total, pluginName) for each plugin being installed.
	Progress ProgressCallback
}

// InstallPluginsWithProgress installs plugins and reports progress.
// It handles both pre-filtering (when InstalledPlugins is provided) and
// fallback detection of "already installed" from command output.
func InstallPluginsWithProgress(
	plugins []string,
	executor CommandExecutor,
	opts InstallPluginsOptions,
) *InstallPluginsResult {
	result := &InstallPluginsResult{}

	// Filter out already-installed plugins if a map was provided
	var toInstall []string
	if opts.InstalledPlugins != nil {
		for _, plugin := range plugins {
			if opts.InstalledPlugins[plugin] {
				result.Skipped = append(result.Skipped, plugin)
			} else {
				toInstall = append(toInstall, plugin)
			}
		}
	} else {
		// No pre-filtering, try to install all
		toInstall = plugins
	}

	// Install plugins with optional progress reporting
	for i, plugin := range toInstall {
		if opts.Progress != nil {
			opts.Progress(i+1, len(toInstall), plugin)
		}

		// Build command based on scope
		args := []string{"plugin", "install"}
		if opts.Scope != "" {
			args = append(args, "--scope", opts.Scope)
		}
		args = append(args, plugin)

		output, err := executor.RunWithOutput(args...)
		if err != nil {
			// Check if it's just "already installed" - treat as skipped, not error
			if strings.Contains(output, "already installed") {
				result.Skipped = append(result.Skipped, plugin)
			} else {
				result.Errors = append(result.Errors, fmt.Errorf("plugin %s: %w", plugin, err))
			}
		} else {
			result.Installed = append(result.Installed, plugin)
		}
	}

	return result
}
