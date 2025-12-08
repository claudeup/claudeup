// ABOUTME: Applies a profile to Claude Code using replace strategy
// ABOUTME: Computes diff, resolves secrets, executes via claude CLI
package profile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/internal/secrets"
)

// CommandExecutor runs claude CLI commands
type CommandExecutor interface {
	Run(args ...string) error
	RunWithOutput(args ...string) (string, error)
}

// DefaultExecutor runs commands using the real claude CLI
type DefaultExecutor struct{}

// Run executes the claude CLI with the given arguments
func (e *DefaultExecutor) Run(args ...string) error {
	return runClaude(args...)
}

// RunWithOutput executes the claude CLI and returns captured output
func (e *DefaultExecutor) RunWithOutput(args ...string) (string, error) {
	return runClaudeWithOutput(args...)
}

// ApplyResult contains the results of applying a profile
type ApplyResult struct {
	PluginsRemoved      []string
	PluginsInstalled    []string
	MCPServersRemoved   []string
	MCPServersInstalled []string
	MarketplacesAdded   []string
	Errors              []error
}

// Diff represents what needs to change to apply a profile
type Diff struct {
	PluginsToRemove  []string
	PluginsToInstall []string
	MCPToRemove      []string
	MCPToInstall     []MCPServer
	MarketplacesToAdd []Marketplace
}

// ComputeDiff calculates what changes are needed to apply a profile
func ComputeDiff(profile *Profile, claudeDir, claudeJSONPath string) (*Diff, error) {
	current, err := Snapshot("current", claudeDir, claudeJSONPath)
	if err != nil {
		// If we can't read current state, treat as empty
		current = &Profile{}
	}

	diff := &Diff{}

	// Plugins to remove (in current but not in profile)
	currentPlugins := toSet(current.Plugins)
	profilePlugins := toSet(profile.Plugins)

	for plugin := range currentPlugins {
		if _, exists := profilePlugins[plugin]; !exists {
			diff.PluginsToRemove = append(diff.PluginsToRemove, plugin)
		}
	}

	// Plugins to install (in profile but not in current)
	for plugin := range profilePlugins {
		if _, exists := currentPlugins[plugin]; !exists {
			diff.PluginsToInstall = append(diff.PluginsToInstall, plugin)
		}
	}

	// MCP servers to remove/install
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

	// Marketplaces to add (we don't remove marketplaces - just add missing ones)
	currentMarketplaces := make(map[string]bool)
	for _, m := range current.Marketplaces {
		currentMarketplaces[m.Repo] = true
	}

	for _, m := range profile.Marketplaces {
		if !currentMarketplaces[m.Repo] {
			diff.MarketplacesToAdd = append(diff.MarketplacesToAdd, m)
		}
	}

	return diff, nil
}

// Apply executes the profile changes using the default executor
func Apply(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain) (*ApplyResult, error) {
	return ApplyWithExecutor(profile, claudeDir, claudeJSONPath, secretChain, &DefaultExecutor{})
}

// ApplyWithExecutor executes the profile changes using the provided executor
func ApplyWithExecutor(profile *Profile, claudeDir, claudeJSONPath string, secretChain *secrets.Chain, executor CommandExecutor) (*ApplyResult, error) {
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

	// Remove plugins
	for _, plugin := range diff.PluginsToRemove {
		output, err := executor.RunWithOutput("plugin", "uninstall", plugin)
		if err != nil {
			// Check if the error is just "already uninstalled" - treat as success
			if strings.Contains(output, "already uninstalled") {
				fmt.Printf("✔ Plugin %s was already uninstalled\n", plugin)
				result.PluginsRemoved = append(result.PluginsRemoved, plugin)
			} else {
				fmt.Print(output) // Show the actual error output
				result.Errors = append(result.Errors, fmt.Errorf("failed to uninstall plugin %s: %w", plugin, err))
			}
		} else {
			fmt.Print(output) // Show success message
			result.PluginsRemoved = append(result.PluginsRemoved, plugin)
		}
	}

	// Remove MCP servers
	for _, mcp := range diff.MCPToRemove {
		if err := executor.Run("mcp", "remove", mcp); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove MCP server %s: %w", mcp, err))
		} else {
			result.MCPServersRemoved = append(result.MCPServersRemoved, mcp)
		}
	}

	// Add marketplaces
	for _, m := range diff.MarketplacesToAdd {
		if m.Repo != "" {
			if err := executor.Run("plugin", "marketplace", "add", m.Repo); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to add marketplace %s: %w", m.Repo, err))
			} else {
				result.MarketplacesAdded = append(result.MarketplacesAdded, m.Repo)
			}
		}
	}

	// Install plugins
	for _, plugin := range diff.PluginsToInstall {
		if err := executor.Run("plugin", "install", plugin); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to install plugin %s: %w", plugin, err))
		} else {
			result.PluginsInstalled = append(result.PluginsInstalled, plugin)
		}
	}

	// Install MCP servers
	for _, mcp := range diff.MCPToInstall {
		args := buildMCPAddArgs(mcp, resolvedMCP[mcp.Name])
		if err := executor.Run(args...); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to add MCP server %s: %w", mcp.Name, err))
		} else {
			result.MCPServersInstalled = append(result.MCPServersInstalled, mcp.Name)
		}
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

func runClaude(args ...string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found: %w", err)
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// runClaudeWithOutput runs claude and captures combined output
// Returns (output, error) - useful for checking error messages
func runClaudeWithOutput(args ...string) (string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found: %w", err)
	}

	cmd := exec.Command(claudePath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DefaultClaudeDir returns the Claude configuration directory
// Respects CLAUDE_CONFIG_DIR environment variable if set
func DefaultClaudeDir() string {
	if override := os.Getenv("CLAUDE_CONFIG_DIR"); override != "" {
		return override
	}
	return filepath.Join(MustHomeDir(), ".claude")
}

// DefaultClaudeJSONPath returns the path to .claude.json
// When CLAUDE_CONFIG_DIR is set, it's inside that directory
// Otherwise it's at ~/.claude.json
func DefaultClaudeJSONPath() string {
	if override := os.Getenv("CLAUDE_CONFIG_DIR"); override != "" {
		return filepath.Join(override, ".claude.json")
	}
	return filepath.Join(MustHomeDir(), ".claude.json")
}

func toSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, item := range slice {
		set[item] = struct{}{}
	}
	return set
}

// MustHomeDir returns the user's home directory or panics if it cannot be determined.
// This is appropriate because the tool cannot function without knowing the home directory.
func MustHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("cannot determine home directory: %v", err))
	}
	return homeDir
}
