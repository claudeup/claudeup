// ABOUTME: Applies a profile to Claude Code using replace strategy
// ABOUTME: Computes diff, resolves secrets, executes via claude CLI
package profile

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/v4/internal/claude"
	"github.com/claudeup/claudeup/v4/internal/config"
	"github.com/claudeup/claudeup/v4/internal/local"
	"github.com/claudeup/claudeup/v4/internal/secrets"
)

// ApplyOptions controls how a profile is applied
type ApplyOptions struct {
	Scope        Scope            // user, project, or local
	ProjectDir   string           // Required for project/local scope
	DryRun       bool             // If true, don't make changes (not yet implemented)
	Reinstall    bool             // If true, reinstall even if already installed
	ShowProgress bool             // If true, use concurrent apply with progress UI (project/local scope only)
	Progress     ProgressCallback // Optional progress callback for sequential installs
}

// CommandExecutor runs claude CLI commands
type CommandExecutor interface {
	Run(args ...string) error
	RunWithOutput(args ...string) (string, error)
}

// DefaultExecutor runs commands using the real claude CLI
type DefaultExecutor struct {
	ClaudeDir string // Directory to use for CLAUDE_CONFIG_DIR env var
}

// Run executes the claude CLI with the given arguments
func (e *DefaultExecutor) Run(args ...string) error {
	return runClaude(e.ClaudeDir, args...)
}

// RunWithOutput executes the claude CLI and returns captured output
func (e *DefaultExecutor) RunWithOutput(args ...string) (string, error) {
	return runClaudeWithOutput(e.ClaudeDir, args...)
}

// ApplyResult contains the results of applying a profile
type ApplyResult struct {
	PluginsRemoved        []string
	PluginsInstalled      []string
	PluginsAlreadyRemoved []string // Plugins that were already uninstalled
	PluginsAlreadyPresent []string // Plugins that were already installed
	MCPServersRemoved     []string
	MCPServersInstalled   []string
	MarketplacesAdded     []string
	MarketplacesRemoved   []string
	Errors                []error
}

// Diff represents what needs to change to apply a profile
type Diff struct {
	PluginsToRemove      []string
	PluginsToInstall     []string
	MCPToRemove          []string
	MCPToInstall         []MCPServer
	MarketplacesToAdd    []Marketplace
	MarketplacesToRemove []Marketplace
}

// DiffOptions controls how a diff is computed
type DiffOptions struct {
	Scope      Scope  // Target scope for comparison
	ProjectDir string // Required for project/local scope
}

// ComputeDiff calculates what changes are needed to apply a profile.
// This compares against user scope by default; use ComputeDiffWithScope for scope-aware comparison.
func ComputeDiff(profile *Profile, claudeDir, claudeJSONPath string) (*Diff, error) {
	return ComputeDiffWithScope(profile, claudeDir, claudeJSONPath, DiffOptions{Scope: ScopeUser})
}

// ComputeDiffWithScope calculates what changes are needed to apply a profile at a specific scope.
// For project/local scope, it compares only against that scope's current state (not user scope).
// This prevents confusing "Remove" actions for user-scope items when applying at project scope.
func ComputeDiffWithScope(profile *Profile, claudeDir, claudeJSONPath string, opts DiffOptions) (*Diff, error) {
	var current *Profile
	var err error

	scope := opts.Scope
	if scope == "" {
		scope = ScopeUser
	}

	// Snapshot the current state for the target scope only
	switch scope {
	case ScopeProject, ScopeLocal:
		current, err = SnapshotWithScope("current", claudeDir, claudeJSONPath, SnapshotOptions{
			Scope:      string(scope),
			ProjectDir: opts.ProjectDir,
		})
	default:
		current, err = Snapshot("current", claudeDir, claudeJSONPath)
	}

	if err != nil {
		// If we can't read current state, treat as empty
		current = &Profile{}
	}

	diff := &Diff{}

	// Skip plugin diff if profile opts out (e.g., wizard-managed plugins)
	if !profile.SkipPluginDiff {
		// Plugins to remove (in current but not in profile)
		currentPlugins := toSet(current.Plugins)
		profilePlugins := toSet(profile.Plugins)

		for plugin := range currentPlugins {
			if _, exists := profilePlugins[plugin]; !exists {
				diff.PluginsToRemove = append(diff.PluginsToRemove, plugin)
			}
		}

		// Plugins to install - always include ALL profile plugins to ensure
		// they're properly registered with Claude CLI, even if they appear
		// in the current state (they may be in a broken state where JSON
		// shows them but Claude CLI doesn't recognize them)
		for plugin := range profilePlugins {
			diff.PluginsToInstall = append(diff.PluginsToInstall, plugin)
		}
	}

	// MCP servers to remove/install
	// Note: MCP servers respect scope during apply (--scope flag), so we compare them
	// directly without special scope handling. Unlike marketplaces (always user-scoped),
	// MCP servers can be added at project or local scope and will be properly scoped.
	currentMCP := make(map[string]bool)
	for _, mcp := range current.MCPServers {
		currentMCP[mcp.Name] = true
	}

	profileMCP := make(map[string]MCPServer)
	for _, mcp := range profile.MCPServers {
		profileMCP[mcp.Name] = mcp
	}

	for name := range currentMCP {
		if _, exists := profileMCP[name]; !exists {
			diff.MCPToRemove = append(diff.MCPToRemove, name)
		}
	}

	for name, mcp := range profileMCP {
		if !currentMCP[name] {
			diff.MCPToInstall = append(diff.MCPToInstall, mcp)
		}
	}

	// Marketplaces: always user-scoped in Claude Code
	// For project/local scope, we only ADD missing marketplaces (needed to resolve plugins)
	// but never REMOVE user-scope marketplaces
	currentMarketplaces := make(map[string]Marketplace)
	for _, m := range current.Marketplaces {
		currentMarketplaces[marketplaceKey(m)] = m
	}

	profileMarketplaces := make(map[string]bool)
	for _, m := range profile.Marketplaces {
		profileMarketplaces[marketplaceKey(m)] = true
	}

	// Only remove marketplaces for user scope (declarative behavior)
	// For project/local scope, don't show removal of user-scope marketplaces
	if scope == ScopeUser {
		for key, m := range currentMarketplaces {
			if !profileMarketplaces[key] {
				diff.MarketplacesToRemove = append(diff.MarketplacesToRemove, m)
			}
		}
	}

	// Add marketplaces missing from current (all scopes need this for plugin resolution)
	for _, m := range profile.Marketplaces {
		if _, exists := currentMarketplaces[marketplaceKey(m)]; !exists {
			diff.MarketplacesToAdd = append(diff.MarketplacesToAdd, m)
		}
	}

	return diff, nil
}

// Apply executes the profile changes using the default executor
func Apply(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain) (*ApplyResult, error) {
	return ApplyWithExecutor(profile, claudeDir, claudeJSONPath, secretChain, &DefaultExecutor{ClaudeDir: claudeDir})
}

// ApplyWithOptions applies a profile with the specified scope options
func ApplyWithOptions(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain, opts ApplyOptions) (*ApplyResult, error) {
	// Validate options
	if opts.Scope == "" {
		opts.Scope = ScopeUser
	}

	if (opts.Scope == ScopeProject || opts.Scope == ScopeLocal) && opts.ProjectDir == "" {
		return nil, fmt.Errorf("project directory required for %s scope", opts.Scope)
	}

	executor := &DefaultExecutor{ClaudeDir: claudeDir}

	// Use concurrent apply with progress tracking for project/local scope.
	// User scope always uses sequential apply because it needs declarative behavior
	// (removes plugins not in profile, then adds missing ones). Concurrent apply
	// is additive-only, suitable for project/local where we don't remove plugins.
	if opts.ShowProgress && opts.Scope != ScopeUser {
		concurrentResult, err := ApplyConcurrently(profile, ConcurrentApplyOptions{
			ClaudeDir: claudeDir,
			Scope:     string(opts.Scope),
			Reinstall: opts.Reinstall,
			Output:    os.Stdout,
			Executor:  executor,
		})
		if err != nil {
			return nil, err
		}

		// Convert to ApplyResult and handle post-apply tasks
		result := convertConcurrentResult(concurrentResult)

		// Write scope-specific config files
		if opts.Scope == ScopeProject {
			if err := writeProjectScopeConfigs(profile, claudeDir, opts.ProjectDir); err != nil {
				return nil, err
			}
		} else if opts.Scope == ScopeLocal {
			if err := writeLocalScopeConfigs(profile, claudeDir, opts.ProjectDir); err != nil {
				return nil, err
			}
		}

		return result, nil
	}

	// Sequential apply (user scope for declarative behavior, or when progress disabled)
	switch opts.Scope {
	case ScopeProject:
		return applyProjectScope(profile, claudeDir, claudeJSONPath, secretChain, opts, executor)
	case ScopeLocal:
		return applyLocalScope(profile, claudeDir, claudeJSONPath, secretChain, opts, executor)
	default:
		// User scope: declarative behavior (removes extras, adds missing)
		return applyUserScope(profile, claudeDir, claudeJSONPath, secretChain, opts, executor)
	}
}

// convertConcurrentResult converts ConcurrentApplyResult to ApplyResult
func convertConcurrentResult(cr *ConcurrentApplyResult) *ApplyResult {
	return &ApplyResult{
		PluginsInstalled:      cr.PluginsInstalled,
		PluginsAlreadyPresent: cr.PluginsSkipped,
		MCPServersInstalled:   cr.MCPServersInstalled,
		MarketplacesAdded:     cr.MarketplacesInstalled,
		Errors:                cr.Errors,
	}
}

// writeProjectScopeConfigs writes .mcp.json and settings.json for project scope
func writeProjectScopeConfigs(profile *Profile, claudeDir, projectDir string) error {
	// Write .mcp.json for MCP servers
	if len(profile.MCPServers) > 0 {
		if err := WriteMCPJSON(projectDir, profile.MCPServers); err != nil {
			return fmt.Errorf("failed to write %s: %w", MCPConfigFile, err)
		}
	}

	// Write project settings.json with enabled plugins
	projectSettings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		projectSettings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}
	projectSettings.EnabledPlugins = make(map[string]bool)
	for _, plugin := range profile.Plugins {
		projectSettings.EnabledPlugins[plugin] = true
	}
	if err := claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings); err != nil {
		return fmt.Errorf("failed to write project settings.json: %w", err)
	}

	// Save profile to project profiles directory for team sharing
	if err := SaveToProject(projectDir, profile); err != nil {
		return fmt.Errorf("failed to save profile to project: %w", err)
	}

	return nil
}

// writeLocalScopeConfigs writes settings.local.json for local scope
func writeLocalScopeConfigs(profile *Profile, claudeDir, projectDir string) error {
	localSettings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
	if err != nil {
		localSettings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}
	localSettings.EnabledPlugins = make(map[string]bool)
	for _, plugin := range profile.Plugins {
		localSettings.EnabledPlugins[plugin] = true
	}
	if err := claude.SaveSettingsForScope("local", claudeDir, projectDir, localSettings); err != nil {
		return fmt.Errorf("failed to write local settings.json: %w", err)
	}

	// Update projects registry
	registry, err := config.LoadProjectsRegistry()
	if err == nil {
		registry.SetProject(projectDir, profile.Name)
		_ = config.SaveProjectsRegistry(registry)
	}

	return nil
}

// applyProjectScope applies a profile at project scope, creating .claude/settings.json and .mcp.json
func applyProjectScope(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain, opts ApplyOptions, executor CommandExecutor) (*ApplyResult, error) {
	result := &ApplyResult{}

	// 1. Write .mcp.json for MCP servers (Claude native format)
	if len(profile.MCPServers) > 0 {
		if err := WriteMCPJSON(opts.ProjectDir, profile.MCPServers); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", MCPConfigFile, err)
		}
		// Track as "installed" even though we're just writing a file
		for _, mcp := range profile.MCPServers {
			result.MCPServersInstalled = append(result.MCPServersInstalled, mcp.Name)
		}
	}

	// 2. Add marketplaces (user-level, needed to resolve plugins)
	validMarketplaces := filterValidMarketplaceKeys(profile.Marketplaces)
	for i, key := range validMarketplaces {
		fmt.Printf("  [%d/%d] Adding marketplace %s\n", i+1, len(validMarketplaces), key)
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

	// 3. Install plugins with project scope using shared function
	installResult := InstallPluginsWithProgress(profile.Plugins, executor, InstallPluginsOptions{
		Scope: "project",
	})
	result.PluginsInstalled = installResult.Installed
	result.PluginsAlreadyPresent = installResult.Skipped
	result.Errors = append(result.Errors, installResult.Errors...)

	// 4. Write project scope settings.json with enabled plugins (declarative replace)
	// CRITICAL: Load existing settings to preserve non-plugin fields
	projectSettings, err := claude.LoadSettingsForScope("project", claudeDir, opts.ProjectDir)
	if err != nil {
		// If settings don't exist, create new minimal settings
		projectSettings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	// Update only enabledPlugins field (preserve all other fields)
	projectSettings.EnabledPlugins = make(map[string]bool)
	for _, plugin := range profile.Plugins {
		projectSettings.EnabledPlugins[plugin] = true
	}

	if err := claude.SaveSettingsForScope("project", claudeDir, opts.ProjectDir, projectSettings); err != nil {
		return nil, fmt.Errorf("failed to write project settings.json: %w", err)
	}

	// 5. Save profile to project profiles directory for team sharing
	if err := SaveToProject(opts.ProjectDir, profile); err != nil {
		return nil, fmt.Errorf("failed to save profile to project: %w", err)
	}

	// 6. Apply local items if present
	if err := applyLocalItems(profile, claudeDir); err != nil {
		result.Errors = append(result.Errors, err)
	}

	// 7. Apply settings hooks if present
	if err := applySettingsHooks(profile, claudeDir); err != nil {
		result.Errors = append(result.Errors, err)
	}

	return result, nil
}

// applyLocalScope applies a profile at local scope (private to this machine)
func applyLocalScope(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain, opts ApplyOptions, executor CommandExecutor) (*ApplyResult, error) {
	result := &ApplyResult{}

	// 1. Resolve secrets for MCP servers
	resolvedMCP := make(map[string]map[string]string)
	for _, mcp := range profile.MCPServers {
		if len(mcp.Secrets) > 0 {
			resolved := make(map[string]string)
			for envVar, ref := range mcp.Secrets {
				var value string
				var resolveErr error
				for _, source := range ref.Sources {
					switch source.Type {
					case "env":
						value, _, resolveErr = secretChain.Resolve(source.Key)
					case "1password":
						value, _, resolveErr = secretChain.Resolve(source.Ref)
					case "keychain":
						keychainRef := source.Service
						if source.Account != "" {
							keychainRef = source.Service + ":" + source.Account
						}
						value, _, resolveErr = secretChain.Resolve(keychainRef)
					}
					if resolveErr == nil && value != "" {
						break
					}
				}
				if value == "" {
					result.Errors = append(result.Errors, fmt.Errorf("could not resolve secret %s for MCP server %s", envVar, mcp.Name))
					continue
				}
				resolved[envVar] = value
			}
			resolvedMCP[mcp.Name] = resolved
		}
	}

	// 2. Add MCP servers with local scope
	for _, mcp := range profile.MCPServers {
		mcpCopy := mcp
		mcpCopy.Scope = "local" // Override to local
		args := buildMCPAddArgs(mcpCopy, resolvedMCP[mcp.Name])
		output, err := executor.RunWithOutput(args...)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("MCP %s: %w\n  Output: %s", mcp.Name, err, strings.TrimSpace(output)))
		} else {
			result.MCPServersInstalled = append(result.MCPServersInstalled, mcp.Name)
		}
	}

	// 3. Add marketplaces (user-level)
	validMarketplaces := filterValidMarketplaceKeys(profile.Marketplaces)
	for i, key := range validMarketplaces {
		fmt.Printf("  [%d/%d] Adding marketplace %s\n", i+1, len(validMarketplaces), key)
		output, err := executor.RunWithOutput("plugin", "marketplace", "add", key)
		if err != nil {
			if strings.Contains(output, "already installed") {
				result.MarketplacesAdded = append(result.MarketplacesAdded, key)
			} else {
				result.Errors = append(result.Errors, fmt.Errorf("marketplace %s: %w", key, err))
			}
		} else {
			result.MarketplacesAdded = append(result.MarketplacesAdded, key)
		}
	}

	// 4. Install plugins with local scope using shared function
	installResult := InstallPluginsWithProgress(profile.Plugins, executor, InstallPluginsOptions{
		Scope: "local",
	})
	result.PluginsInstalled = installResult.Installed
	result.PluginsAlreadyPresent = installResult.Skipped
	result.Errors = append(result.Errors, installResult.Errors...)

	// 5. Write local scope settings.json with enabled plugins (declarative replace)
	// CRITICAL: Load existing settings to preserve non-plugin fields
	localSettings, err := claude.LoadSettingsForScope("local", claudeDir, opts.ProjectDir)
	if err != nil {
		// If settings don't exist, create new minimal settings
		localSettings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	// Update only enabledPlugins field (preserve all other fields)
	localSettings.EnabledPlugins = make(map[string]bool)
	for _, plugin := range profile.Plugins {
		localSettings.EnabledPlugins[plugin] = true
	}

	if err := claude.SaveSettingsForScope("local", claudeDir, opts.ProjectDir, localSettings); err != nil {
		return nil, fmt.Errorf("failed to write local settings.json: %w", err)
	}

	// 6. Update projects registry
	registry, err := config.LoadProjectsRegistry()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("load registry: %w", err))
	} else {
		registry.SetProject(opts.ProjectDir, profile.Name)
		if err := config.SaveProjectsRegistry(registry); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save registry: %w", err))
		}
	}

	// 7. Apply local items if present
	if err := applyLocalItems(profile, claudeDir); err != nil {
		result.Errors = append(result.Errors, err)
	}

	// 8. Apply settings hooks if present
	if err := applySettingsHooks(profile, claudeDir); err != nil {
		result.Errors = append(result.Errors, err)
	}

	return result, nil
}

// ApplyWithExecutor executes the profile changes using the provided executor.
// This is the legacy API for backward compatibility; use ApplyWithOptions for new code.
func ApplyWithExecutor(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain, executor CommandExecutor) (*ApplyResult, error) {
	return applyUserScope(profile, claudeDir, claudeJSONPath, secretChain, ApplyOptions{}, executor)
}

// applyUserScope applies a profile at user scope with declarative behavior.
// It computes a diff, removes extras, adds missing plugins/MCP servers.
func applyUserScope(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain, opts ApplyOptions, executor CommandExecutor) (*ApplyResult, error) {
	diff, err := ComputeDiff(profile, claudeDir, claudeJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	result := &ApplyResult{}

	// Resolve secrets for MCP servers before making any changes
	resolvedMCP := make(map[string]map[string]string) // mcp name -> env var -> value
	for _, mcp := range diff.MCPToInstall {
		if len(mcp.Secrets) > 0 {
			resolved := make(map[string]string)
			for envVar, ref := range mcp.Secrets {
				// Try each source in order
				var value string
				var resolveErr error
				for _, source := range ref.Sources {
					switch source.Type {
					case "env":
						value, _, resolveErr = secretChain.Resolve(source.Key)
					case "1password":
						value, _, resolveErr = secretChain.Resolve(source.Ref)
					case "keychain":
						keychainRef := source.Service
						if source.Account != "" {
							keychainRef = source.Service + ":" + source.Account
						}
						value, _, resolveErr = secretChain.Resolve(keychainRef)
					}
					if resolveErr == nil && value != "" {
						break
					}
				}
				if value == "" {
					return nil, fmt.Errorf("could not resolve secret %s for MCP server %s", envVar, mcp.Name)
				}
				resolved[envVar] = value
			}
			resolvedMCP[mcp.Name] = resolved
		}
	}

	// Remove plugins by disabling them in settings.json
	// Note: We disable in settings.json instead of uninstalling because plugins
	// may exist at multiple scopes (user/project/local) and claude CLI only
	// uninstalls from one scope at a time, leaving them enabled
	if len(diff.PluginsToRemove) > 0 {
		settings, err := claude.LoadSettings(claudeDir)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to load settings: %w", err))
		} else {
			for _, plugin := range diff.PluginsToRemove {
				if settings.IsPluginEnabled(plugin) {
					settings.RemovePlugin(plugin)
					result.PluginsRemoved = append(result.PluginsRemoved, plugin)
				} else {
					result.PluginsAlreadyRemoved = append(result.PluginsAlreadyRemoved, plugin)
				}
			}

			// Save updated settings
			if err := claude.SaveSettings(claudeDir, settings); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to save settings: %w", err))
			}
		}
	}

	// Remove MCP servers
	for _, mcp := range diff.MCPToRemove {
		output, err := executor.RunWithOutput("mcp", "remove", mcp)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove MCP server %s: %w\n  Output: %s", mcp, err, strings.TrimSpace(output)))
		} else {
			result.MCPServersRemoved = append(result.MCPServersRemoved, mcp)
		}
	}

	// Remove marketplaces
	// Load current marketplaces to find names by repo/URL
	marketplaceRegistry, err := claude.LoadMarketplaces(claudeDir)
	if err == nil {
		for _, m := range diff.MarketplacesToRemove {
			// Find the marketplace name by matching repo/URL
			var marketplaceName string
			repoKey := marketplaceKey(m)
			for name, meta := range marketplaceRegistry {
				metaKey := meta.Source.Repo
				if metaKey == "" {
					metaKey = meta.Source.URL
				}
				if metaKey == repoKey {
					marketplaceName = name
					break
				}
			}

			if marketplaceName != "" {
				output, err := executor.RunWithOutput("plugin", "marketplace", "remove", marketplaceName)
				if err != nil {
					// Check if already removed - treat as success
					if strings.Contains(output, "not found") || strings.Contains(output, "not installed") {
						result.MarketplacesRemoved = append(result.MarketplacesRemoved, repoKey)
					} else {
						result.Errors = append(result.Errors, fmt.Errorf("failed to remove marketplace %s (%s): %w\n  Output: %s", marketplaceName, repoKey, err, strings.TrimSpace(output)))
					}
				} else {
					result.MarketplacesRemoved = append(result.MarketplacesRemoved, repoKey)
				}
			}
		}
	}

	// Add marketplaces
	validMarketplaces := filterValidMarketplaceKeys(diff.MarketplacesToAdd)
	for i, key := range validMarketplaces {
		fmt.Printf("  [%d/%d] Adding marketplace %s\n", i+1, len(validMarketplaces), key)
		output, err := executor.RunWithOutput("plugin", "marketplace", "add", key)
		if err != nil {
			// Check if already installed - treat as success
			if strings.Contains(output, "already installed") {
				result.MarketplacesAdded = append(result.MarketplacesAdded, key)
			} else {
				result.Errors = append(result.Errors, fmt.Errorf("failed to add marketplace %s: %w\n  Output: %s", key, err, strings.TrimSpace(output)))
			}
		} else {
			result.MarketplacesAdded = append(result.MarketplacesAdded, key)
		}
	}

	// Install plugins using shared function (user scope - no --scope flag)
	installResult := InstallPluginsWithProgress(diff.PluginsToInstall, executor, InstallPluginsOptions{
		Scope:    "", // empty = user scope (no --scope flag)
		Progress: opts.Progress,
	})
	result.PluginsInstalled = append(result.PluginsInstalled, installResult.Installed...)
	result.PluginsAlreadyPresent = append(result.PluginsAlreadyPresent, installResult.Skipped...)
	result.Errors = append(result.Errors, installResult.Errors...)

	// Write user scope settings.json with enabled plugins (declarative replace)
	// This ensures settings.json exactly matches the profile
	// CRITICAL: Load existing settings to preserve non-plugin fields (mcpServers, etc.)
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		// If settings don't exist, create new minimal settings
		userSettings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	// Update only enabledPlugins field (preserve all other fields)
	userSettings.EnabledPlugins = make(map[string]bool)
	for _, plugin := range profile.Plugins {
		userSettings.EnabledPlugins[plugin] = true
	}

	if err := claude.SaveSettings(claudeDir, userSettings); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to write user settings.json: %w", err))
	}

	// Install MCP servers
	for _, mcp := range diff.MCPToInstall {
		args := buildMCPAddArgs(mcp, resolvedMCP[mcp.Name])
		output, err := executor.RunWithOutput(args...)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to add MCP server %s: %w\n  Output: %s", mcp.Name, err, strings.TrimSpace(output)))
		} else {
			result.MCPServersInstalled = append(result.MCPServersInstalled, mcp.Name)
		}
	}

	// Apply local items if present
	if err := applyLocalItems(profile, claudeDir); err != nil {
		result.Errors = append(result.Errors, err)
	}

	// Apply settings hooks if present
	if err := applySettingsHooks(profile, claudeDir); err != nil {
		result.Errors = append(result.Errors, err)
	}

	return result, nil
}

func buildMCPAddArgs(mcp MCPServer, resolvedSecrets map[string]string) []string {
	args := []string{"mcp", "add", mcp.Name}

	// Add scope if specified
	scope := mcp.Scope
	if scope == "" {
		scope = "user"
	}
	args = append(args, "-s", scope)

	// Add separator and command
	args = append(args, "--", mcp.Command)

	// Add command args, substituting secrets
	for _, arg := range mcp.Args {
		if strings.HasPrefix(arg, "$") {
			envVar := strings.TrimPrefix(arg, "$")
			if value, ok := resolvedSecrets[envVar]; ok {
				args = append(args, value)
			} else if value := os.Getenv(envVar); value != "" {
				args = append(args, value)
			} else {
				args = append(args, arg) // Keep as-is if not resolved
			}
		} else {
			args = append(args, arg)
		}
	}

	return args
}

func runClaude(claudeDir string, args ...string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found: %w", err)
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set CLAUDE_CONFIG_DIR if a custom directory was specified
	if claudeDir != "" && claudeDir != DefaultClaudeDir() {
		cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+claudeDir)
	}

	return cmd.Run()
}

// runClaudeWithOutput runs claude and captures combined output
// Returns (output, error) - useful for checking error messages
func runClaudeWithOutput(claudeDir string, args ...string) (string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found: %w", err)
	}

	cmd := exec.Command(claudePath, args...)

	// Set CLAUDE_CONFIG_DIR if a custom directory was specified
	if claudeDir != "" && claudeDir != DefaultClaudeDir() {
		cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+claudeDir)
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DefaultClaudeDir returns the Claude configuration directory
// Respects CLAUDE_CONFIG_DIR environment variable if set
func DefaultClaudeDir() string {
	return config.MustClaudeDir()
}

func toSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, item := range slice {
		set[item] = struct{}{}
	}
	return set
}

// HookOptions controls post-apply hook behavior
type HookOptions struct {
	ForceSetup    bool   // Run hook even if first-run check would skip
	NoInteractive bool   // Skip hook entirely (for CI/scripting)
	ScriptDir     string // Directory containing hook scripts (for built-in profiles)
}

// ShouldRunHook checks if the post-apply hook should run based on condition and current state
func ShouldRunHook(profile *Profile, claudeDir, claudeJSONPath string, opts HookOptions) bool {
	if opts.NoInteractive {
		return false
	}

	if profile.PostApply == nil {
		return false
	}

	if opts.ForceSetup {
		return true
	}

	// Check condition
	switch profile.PostApply.Condition {
	case "always", "":
		return true
	case "first-run":
		return isFirstRun(profile, claudeDir, claudeJSONPath)
	default:
		return false
	}
}

// isFirstRun checks if any plugins from the profile's marketplaces are enabled
func isFirstRun(profile *Profile, claudeDir, claudeJSONPath string) bool {
	current, err := Snapshot("current", claudeDir, claudeJSONPath)
	if err != nil {
		// Can't read current state - treat as first run
		return true
	}

	// Build set of marketplace suffixes from profile
	marketplaceSuffixes := make([]string, 0, len(profile.Marketplaces))
	for _, m := range profile.Marketplaces {
		// Extract marketplace name (e.g., "user/repo" -> "user-repo")
		name := marketplaceName(m)
		if name != "" {
			marketplaceSuffixes = append(marketplaceSuffixes, "@"+name)
		}
	}

	// Check if any current plugins match these marketplaces
	for _, plugin := range current.Plugins {
		for _, suffix := range marketplaceSuffixes {
			if strings.HasSuffix(plugin, suffix) {
				return false // Found a plugin from this marketplace - not first run
			}
		}
	}

	return true
}

// marketplaceKey returns the lookup key for a marketplace (Repo or URL)
func marketplaceKey(m Marketplace) string {
	if m.Repo != "" {
		return m.Repo
	}
	return m.URL
}

// filterValidMarketplaceKeys returns only non-empty marketplace keys
func filterValidMarketplaceKeys(marketplaces []Marketplace) []string {
	var keys []string
	for _, m := range marketplaces {
		if key := marketplaceKey(m); key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// marketplaceName extracts the marketplace name from a repo path or URL
func marketplaceName(m Marketplace) string {
	key := m.Repo
	if key == "" {
		key = m.URL
	}
	if key == "" {
		return ""
	}

	// Handle URLs by extracting the path portion
	// e.g., "https://github.com/user/repo.git" -> "user/repo"
	if strings.Contains(key, "://") {
		parsed, err := url.Parse(key)
		if err != nil {
			// Fall back to treating as plain path
			return strings.ReplaceAll(key, "/", "-")
		}
		// Get path and trim leading slash
		key = strings.TrimPrefix(parsed.Path, "/")
		// Remove .git suffix
		key = strings.TrimSuffix(key, ".git")
	}

	// "user/repo" -> "user-repo"
	return strings.ReplaceAll(key, "/", "-")
}

// ResetResult contains the results of resetting a profile
type ResetResult struct {
	PluginsRemoved      []string
	MCPServersRemoved   []string
	MarketplacesRemoved []string
	Errors              []error
}

// Reset removes everything a profile installed (plugins, MCP servers, marketplaces)
func Reset(profile *Profile, claudeDir, claudeJSONPath string) (*ResetResult, error) {
	return ResetWithExecutor(profile, claudeDir, claudeJSONPath, &DefaultExecutor{ClaudeDir: claudeDir})
}

// ResetWithExecutor removes everything a profile installed using the provided executor
func ResetWithExecutor(profile *Profile, claudeDir, claudeJSONPath string, executor CommandExecutor) (*ResetResult, error) {
	result := &ResetResult{}

	// Get current state to find installed plugins
	current, err := Snapshot("current", claudeDir, claudeJSONPath)
	if err != nil {
		// Can't read current state - nothing to remove
		return result, nil
	}

	// Build lookup from repo to marketplace name for removal
	repoToName := BuildRepoToNameLookup(claudeDir)

	// Build marketplace suffixes to find matching plugins
	marketplaceSuffixes := make(map[string]string) // suffix -> key
	for _, m := range profile.Marketplaces {
		name := marketplaceName(m)
		if name != "" {
			marketplaceSuffixes["@"+name] = marketplaceKey(m)
		}
	}

	// Remove plugins that belong to profile's marketplaces
	for _, plugin := range current.Plugins {
		for suffix := range marketplaceSuffixes {
			if strings.HasSuffix(plugin, suffix) {
				output, err := executor.RunWithOutput("plugin", "uninstall", plugin)
				if err != nil {
					// Check if the error is just "already uninstalled" or "not found" - treat as success
					if strings.Contains(output, "already uninstalled") || strings.Contains(output, "not found") {
						result.PluginsRemoved = append(result.PluginsRemoved, plugin)
					} else {
						result.Errors = append(result.Errors, fmt.Errorf("failed to uninstall plugin %s: %w\n  Output: %s", plugin, err, strings.TrimSpace(output)))
					}
				} else {
					result.PluginsRemoved = append(result.PluginsRemoved, plugin)
				}
				break
			}
		}
	}

	// Remove MCP servers defined in the profile
	for _, mcp := range profile.MCPServers {
		output, err := executor.RunWithOutput("mcp", "remove", mcp.Name)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove MCP server %s: %w\n  Output: %s", mcp.Name, err, strings.TrimSpace(output)))
		} else {
			result.MCPServersRemoved = append(result.MCPServersRemoved, mcp.Name)
		}
	}

	// Remove marketplaces using their registered name (not repo/url)
	for _, m := range profile.Marketplaces {
		// Determine the lookup key (repo or url)
		lookupKey := m.Repo
		if lookupKey == "" {
			lookupKey = m.URL
		}
		if lookupKey == "" {
			continue
		}

		// Look up the marketplace name from the registry
		name, found := repoToName[lookupKey]
		if !found {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove marketplace %s: not found in registry", lookupKey))
			continue
		}
		output, err := executor.RunWithOutput("plugin", "marketplace", "remove", name)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove marketplace %s: %w\n  Output: %s", lookupKey, err, strings.TrimSpace(output)))
		} else {
			result.MarketplacesRemoved = append(result.MarketplacesRemoved, lookupKey)
		}
	}

	return result, nil
}

// BuildRepoToNameLookup reads known_marketplaces.json and builds a map from repo to name
func BuildRepoToNameLookup(claudeDir string) map[string]string {
	result := make(map[string]string)

	marketplacesPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	data, err := os.ReadFile(marketplacesPath)
	if err != nil {
		return result
	}

	var registry MarketplaceRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return result
	}

	for name, meta := range registry {
		if meta.Source.Repo != "" {
			result[meta.Source.Repo] = name
		}
		if meta.Source.URL != "" {
			result[meta.Source.URL] = name
		}
	}

	return result
}

// RunHook executes the post-apply hook
func RunHook(profile *Profile, opts HookOptions) error {
	if profile.PostApply == nil {
		return nil
	}

	hook := profile.PostApply

	// Determine what to run
	var cmd *exec.Cmd
	if hook.Script != "" {
		// Script path - resolve relative to ScriptDir
		scriptPath := hook.Script
		if opts.ScriptDir != "" && !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(opts.ScriptDir, scriptPath)
		}
		// Verify script exists before attempting to run
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("hook script not found: %s", scriptPath)
		}
		cmd = exec.Command("bash", scriptPath)
	} else if hook.Command != "" {
		// Direct command
		cmd = exec.Command("bash", "-c", hook.Command)
	} else {
		return nil // Nothing to run
	}

	// Run interactively
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ApplyAllScopesOptions controls how multi-scope profiles are applied.
type ApplyAllScopesOptions struct {
	// ReplaceUserScope controls whether user-scope settings are replaced (true)
	// or merged additively (false). Default is false (additive).
	// Project and local scopes always use declarative (replace) semantics.
	ReplaceUserScope bool
}

// ApplyAllScopes applies a profile to all scope levels.
// For multi-scope profiles (with PerScope), it applies each scope independently.
// For legacy profiles (flat format), it applies to user scope only.
// The secretChain parameter is optional and used for MCP server secret resolution.
func ApplyAllScopes(profile *Profile, claudeDir, claudeJSONPath, projectDir string, secretChain *secrets.Chain, opts *ApplyAllScopesOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	if opts == nil {
		opts = &ApplyAllScopesOptions{}
	}

	// If legacy profile (no PerScope), apply to user scope only
	if !profile.IsMultiScope() {
		return applyUserScopeSettings(profile, claudeDir, projectDir, opts.ReplaceUserScope)
	}

	// Apply each scope in order: user → project → local
	if profile.PerScope.User != nil {
		userResult, err := applyUserScopeSettings(profile.ForScope("user"), claudeDir, projectDir, opts.ReplaceUserScope)
		if err != nil {
			return nil, fmt.Errorf("failed to apply user scope: %w", err)
		}
		aggregateResults(result, userResult)
	}

	if profile.PerScope.Project != nil && projectDir != "" {
		projectResult, err := applyProjectScopeSettings(profile.ForScope("project"), claudeDir, projectDir)
		if err != nil {
			return nil, fmt.Errorf("failed to apply project scope: %w", err)
		}
		aggregateResults(result, projectResult)
	}

	if profile.PerScope.Local != nil && projectDir != "" {
		localResult, err := applyLocalScopeSettings(profile.ForScope("local"), claudeDir, projectDir)
		if err != nil {
			return nil, fmt.Errorf("failed to apply local scope: %w", err)
		}
		aggregateResults(result, localResult)
	}

	return result, nil
}

// applyUserScopeSettings writes plugins to user-scope settings.json.
// If replace is false (default), existing plugins are preserved and profile plugins are added.
// If replace is true, existing plugins are replaced with profile plugins.
func applyUserScopeSettings(profile *Profile, claudeDir, projectDir string, replace bool) (*ApplyResult, error) {
	result := &ApplyResult{}

	// Load existing user settings (preserves other fields)
	settings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		settings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	if replace {
		// Declarative: replace all plugins with profile plugins
		settings.EnabledPlugins = make(map[string]bool)
		for _, plugin := range profile.Plugins {
			settings.EnabledPlugins[plugin] = true
			result.PluginsInstalled = append(result.PluginsInstalled, plugin)
		}
	} else {
		// Additive: preserve existing plugins, add profile plugins
		if settings.EnabledPlugins == nil {
			settings.EnabledPlugins = make(map[string]bool)
		}
		for _, plugin := range profile.Plugins {
			if !settings.EnabledPlugins[plugin] {
				result.PluginsInstalled = append(result.PluginsInstalled, plugin)
			}
			settings.EnabledPlugins[plugin] = true
		}
	}

	// Save settings
	if err := claude.SaveSettings(claudeDir, settings); err != nil {
		return nil, fmt.Errorf("failed to save user settings: %w", err)
	}

	return result, nil
}

// applyProjectScopeSettings writes plugins to project-scope settings.json
func applyProjectScopeSettings(profile *Profile, claudeDir, projectDir string) (*ApplyResult, error) {
	result := &ApplyResult{}

	// Load existing project settings (preserves other fields)
	settings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		settings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	// Update enabledPlugins with profile plugins
	settings.EnabledPlugins = make(map[string]bool)
	for _, plugin := range profile.Plugins {
		settings.EnabledPlugins[plugin] = true
		result.PluginsInstalled = append(result.PluginsInstalled, plugin)
	}

	// Save settings
	if err := claude.SaveSettingsForScope("project", claudeDir, projectDir, settings); err != nil {
		return nil, fmt.Errorf("failed to save project settings: %w", err)
	}

	return result, nil
}

// applyLocalScopeSettings writes plugins to local-scope settings.local.json
func applyLocalScopeSettings(profile *Profile, claudeDir, projectDir string) (*ApplyResult, error) {
	result := &ApplyResult{}

	// Load existing local settings (preserves other fields)
	settings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
	if err != nil {
		settings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	// Update enabledPlugins with profile plugins
	settings.EnabledPlugins = make(map[string]bool)
	for _, plugin := range profile.Plugins {
		settings.EnabledPlugins[plugin] = true
		result.PluginsInstalled = append(result.PluginsInstalled, plugin)
	}

	// Save settings
	if err := claude.SaveSettingsForScope("local", claudeDir, projectDir, settings); err != nil {
		return nil, fmt.Errorf("failed to save local settings: %w", err)
	}

	return result, nil
}

// aggregateResults combines results from multiple scope applications
func aggregateResults(target, source *ApplyResult) {
	if source == nil || target == nil {
		return
	}
	target.PluginsInstalled = append(target.PluginsInstalled, source.PluginsInstalled...)
	target.PluginsRemoved = append(target.PluginsRemoved, source.PluginsRemoved...)
	target.PluginsAlreadyPresent = append(target.PluginsAlreadyPresent, source.PluginsAlreadyPresent...)
	target.PluginsAlreadyRemoved = append(target.PluginsAlreadyRemoved, source.PluginsAlreadyRemoved...)
	target.MCPServersInstalled = append(target.MCPServersInstalled, source.MCPServersInstalled...)
	target.MCPServersRemoved = append(target.MCPServersRemoved, source.MCPServersRemoved...)
	target.MarketplacesAdded = append(target.MarketplacesAdded, source.MarketplacesAdded...)
	target.MarketplacesRemoved = append(target.MarketplacesRemoved, source.MarketplacesRemoved...)
	target.Errors = append(target.Errors, source.Errors...)
}

// applyLocalItems enables local items from the profile
func applyLocalItems(profile *Profile, claudeDir string) error {
	if profile.LocalItems == nil {
		return nil
	}

	manager := local.NewManager(claudeDir)

	// Enable agents
	if len(profile.LocalItems.Agents) > 0 {
		if _, _, err := manager.Enable(local.CategoryAgents, profile.LocalItems.Agents); err != nil {
			return fmt.Errorf("failed to enable agents: %w", err)
		}
	}

	// Enable commands
	if len(profile.LocalItems.Commands) > 0 {
		if _, _, err := manager.Enable(local.CategoryCommands, profile.LocalItems.Commands); err != nil {
			return fmt.Errorf("failed to enable commands: %w", err)
		}
	}

	// Enable skills
	if len(profile.LocalItems.Skills) > 0 {
		if _, _, err := manager.Enable(local.CategorySkills, profile.LocalItems.Skills); err != nil {
			return fmt.Errorf("failed to enable skills: %w", err)
		}
	}

	// Enable hooks
	if len(profile.LocalItems.Hooks) > 0 {
		if _, _, err := manager.Enable(local.CategoryHooks, profile.LocalItems.Hooks); err != nil {
			return fmt.Errorf("failed to enable hooks: %w", err)
		}
	}

	// Enable rules
	if len(profile.LocalItems.Rules) > 0 {
		if _, _, err := manager.Enable(local.CategoryRules, profile.LocalItems.Rules); err != nil {
			return fmt.Errorf("failed to enable rules: %w", err)
		}
	}

	// Enable output-styles
	if len(profile.LocalItems.OutputStyles) > 0 {
		if _, _, err := manager.Enable(local.CategoryOutputStyles, profile.LocalItems.OutputStyles); err != nil {
			return fmt.Errorf("failed to enable output-styles: %w", err)
		}
	}

	return nil
}

// applySettingsHooks merges profile hooks into settings.json
func applySettingsHooks(profile *Profile, claudeDir string) error {
	if len(profile.SettingsHooks) == 0 {
		return nil
	}

	settings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		// Create new settings if none exist
		settings = &claude.Settings{
			EnabledPlugins: make(map[string]bool),
		}
	}

	// Convert HookEntry to map format for MergeHooks
	hooksMap := make(map[string][]map[string]interface{})
	for eventType, entries := range profile.SettingsHooks {
		for _, entry := range entries {
			hooksMap[eventType] = append(hooksMap[eventType], map[string]interface{}{
				"type":    entry.Type,
				"command": entry.Command,
			})
		}
	}

	if err := settings.MergeHooks(hooksMap); err != nil {
		return fmt.Errorf("failed to merge hooks: %w", err)
	}

	if err := claude.SaveSettings(claudeDir, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}
