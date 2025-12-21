// ABOUTME: Concurrent execution engine for profile apply operations
// ABOUTME: Handles parallel marketplace/plugin installs with progress tracking
package profile

import (
	"fmt"
	"io"
	"os"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
)

// ConcurrentApplyOptions configures concurrent apply behavior
type ConcurrentApplyOptions struct {
	ClaudeDir   string
	Scope       string // "user", "project", "local"
	Reinstall   bool   // Force reinstall even if already installed
	Output      io.Writer
	Executor    CommandExecutor
}

// ConcurrentApplyResult contains results from concurrent apply
type ConcurrentApplyResult struct {
	MarketplacesInstalled []string
	MarketplacesSkipped   []string
	PluginsInstalled      []string
	PluginsSkipped        []string
	MCPServersInstalled   []string
	Errors                []error
}

// ApplyConcurrently installs marketplaces and plugins concurrently with progress tracking
func ApplyConcurrently(profile *Profile, opts ConcurrentApplyOptions) (*ConcurrentApplyResult, error) {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	result := &ConcurrentApplyResult{}

	// Load current state to determine what needs installing
	currentMarketplaces, _ := claude.LoadMarketplaces(opts.ClaudeDir)
	if currentMarketplaces == nil {
		currentMarketplaces = make(claude.MarketplaceRegistry)
	}

	currentPlugins, _ := claude.LoadPlugins(opts.ClaudeDir)

	// Filter marketplaces - skip already installed unless reinstall
	var marketplacesToInstall []Marketplace
	for _, m := range profile.Marketplaces {
		key := marketplaceKey(m)
		if opts.Reinstall || !currentMarketplaces.MarketplaceExists(key) {
			marketplacesToInstall = append(marketplacesToInstall, m)
		} else {
			result.MarketplacesSkipped = append(result.MarketplacesSkipped, key)
		}
	}

	// Filter plugins - skip already installed unless reinstall
	var pluginsToInstall []string
	for _, plugin := range profile.Plugins {
		if opts.Reinstall || currentPlugins == nil || !currentPlugins.PluginExists(plugin) {
			pluginsToInstall = append(pluginsToInstall, plugin)
		} else {
			result.PluginsSkipped = append(result.PluginsSkipped, plugin)
		}
	}

	// Create progress tracker
	tracker := ui.NewProgressTracker(ui.TrackerConfig{
		Phases: []string{"Marketplaces", "Plugins", "MCP Servers"},
		Window: 5,
	})
	tracker.SetPhaseTotals("Marketplaces", len(marketplacesToInstall), len(profile.Marketplaces))
	tracker.SetPhaseTotals("Plugins", len(pluginsToInstall), len(profile.Plugins))
	tracker.SetPhaseTotals("MCP Servers", len(profile.MCPServers), len(profile.MCPServers))

	// Initial render
	tracker.Render(opts.Output)

	// Phase 1: Install marketplaces (must complete before plugins)
	if len(marketplacesToInstall) > 0 {
		marketplaceJobs := make([]Job, len(marketplacesToInstall))
		for i, m := range marketplacesToInstall {
			m := m // capture for closure
			key := marketplaceKey(m)
			marketplaceJobs[i] = Job{
				Name: key,
				Type: "marketplace",
				Execute: func() error {
					_, err := opts.Executor.RunWithOutput("plugin", "marketplace", "add", key)
					return err
				},
			}
		}

		jobResults := RunWorkerPoolWithCallback(marketplaceJobs, DefaultWorkers, func(jr JobResult) {
			tracker.RecordResult("Marketplaces", ui.ItemResult{
				Name:    jr.Name,
				Success: jr.Success,
				Error:   errorString(jr.Error),
			})
			tracker.RenderUpdate(opts.Output, "Marketplaces", ui.ItemResult{
				Name:    jr.Name,
				Success: jr.Success,
				Error:   errorString(jr.Error),
			})
		})

		for _, jr := range jobResults {
			if jr.Success {
				result.MarketplacesInstalled = append(result.MarketplacesInstalled, jr.Name)
			} else {
				result.Errors = append(result.Errors, fmt.Errorf("marketplace %s: %w", jr.Name, jr.Error))
			}
		}
	}

	// Phase 2: Install plugins (concurrent with MCP servers)
	pluginJobs := make([]Job, len(pluginsToInstall))
	for i, plugin := range pluginsToInstall {
		plugin := plugin // capture
		args := []string{"plugin", "install"}
		if opts.Scope != "" && opts.Scope != "user" {
			args = append(args, "--scope", opts.Scope)
		}
		args = append(args, plugin)

		pluginJobs[i] = Job{
			Name: plugin,
			Type: "plugin",
			Execute: func() error {
				_, err := opts.Executor.RunWithOutput(args...)
				return err
			},
		}
	}

	// MCP server jobs (can run in parallel with plugins)
	mcpJobs := make([]Job, len(profile.MCPServers))
	for i, mcp := range profile.MCPServers {
		mcp := mcp // capture
		mcpJobs[i] = Job{
			Name: mcp.Name,
			Type: "mcp",
			Execute: func() error {
				mcpCopy := mcp
				if opts.Scope != "" && opts.Scope != "user" {
					mcpCopy.Scope = opts.Scope
				}
				args := buildMCPAddArgs(mcpCopy, nil)
				_, err := opts.Executor.RunWithOutput(args...)
				return err
			},
		}
	}

	// Combine plugin and MCP jobs for concurrent execution
	allPhase2Jobs := append(pluginJobs, mcpJobs...)

	if len(allPhase2Jobs) > 0 {
		jobResults := RunWorkerPoolWithCallback(allPhase2Jobs, DefaultWorkers, func(jr JobResult) {
			phase := "Plugins"
			if jr.Type == "mcp" {
				phase = "MCP Servers"
			}
			tracker.RecordResult(phase, ui.ItemResult{
				Name:    jr.Name,
				Success: jr.Success,
				Error:   errorString(jr.Error),
			})
			tracker.RenderUpdate(opts.Output, phase, ui.ItemResult{
				Name:    jr.Name,
				Success: jr.Success,
				Error:   errorString(jr.Error),
			})
		})

		for _, jr := range jobResults {
			if jr.Type == "plugin" {
				if jr.Success {
					result.PluginsInstalled = append(result.PluginsInstalled, jr.Name)
				} else {
					result.Errors = append(result.Errors, fmt.Errorf("plugin %s: %w", jr.Name, jr.Error))
				}
			} else if jr.Type == "mcp" {
				if jr.Success {
					result.MCPServersInstalled = append(result.MCPServersInstalled, jr.Name)
				} else {
					result.Errors = append(result.Errors, fmt.Errorf("mcp %s: %w", jr.Name, jr.Error))
				}
			}
		}
	}

	// Finish progress display
	tracker.Finish(opts.Output)

	return result, nil
}

// errorString returns error string or empty if nil
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
