// ABOUTME: Profile comparison and diff functionality
// ABOUTME: Detects changes between saved profiles and current Claude state
package profile

import (
	"fmt"
	"strings"

	"github.com/claudeup/claudeup/internal/claude"
)

// ProfileDiff represents differences between a saved profile and current state
type ProfileDiff struct {
	PluginsAdded         []string
	PluginsRemoved       []string
	MarketplacesAdded    []Marketplace
	MarketplacesRemoved  []Marketplace
	MCPServersAdded      []MCPServer
	MCPServersRemoved    []MCPServer
	MCPServersModified   []MCPServer
}

// HasChanges returns true if there are any differences between profiles
func (d *ProfileDiff) HasChanges() bool {
	return len(d.PluginsAdded) > 0 || len(d.PluginsRemoved) > 0 ||
		len(d.MarketplacesAdded) > 0 || len(d.MarketplacesRemoved) > 0 ||
		len(d.MCPServersAdded) > 0 || len(d.MCPServersRemoved) > 0 ||
		len(d.MCPServersModified) > 0
}

// Summary returns a human-readable summary of the changes
func (d *ProfileDiff) Summary() string {
	if !d.HasChanges() {
		return ""
	}

	var parts []string

	// Plugins
	if len(d.PluginsAdded) > 0 {
		parts = append(parts, pluralize(len(d.PluginsAdded), "plugin", "plugins")+" added")
	}
	if len(d.PluginsRemoved) > 0 {
		parts = append(parts, pluralize(len(d.PluginsRemoved), "plugin", "plugins")+" removed")
	}

	// Marketplaces
	if len(d.MarketplacesAdded) > 0 {
		parts = append(parts, pluralize(len(d.MarketplacesAdded), "marketplace", "marketplaces")+" added")
	}
	if len(d.MarketplacesRemoved) > 0 {
		parts = append(parts, pluralize(len(d.MarketplacesRemoved), "marketplace", "marketplaces")+" removed")
	}

	// MCP servers
	if len(d.MCPServersAdded) > 0 {
		parts = append(parts, pluralize(len(d.MCPServersAdded), "MCP server", "MCP servers")+" added")
	}
	if len(d.MCPServersRemoved) > 0 {
		parts = append(parts, pluralize(len(d.MCPServersRemoved), "MCP server", "MCP servers")+" removed")
	}
	if len(d.MCPServersModified) > 0 {
		parts = append(parts, pluralize(len(d.MCPServersModified), "MCP server", "MCP servers")+" modified")
	}

	return strings.Join(parts, ", ")
}

// pluralize returns a formatted string with proper pluralization
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// CompareWithCurrent compares a saved profile with the current Claude state
func CompareWithCurrent(savedProfile *Profile, claudeDir, claudeJSONPath string) (*ProfileDiff, error) {
	// Create snapshot of current state
	current, err := Snapshot("", claudeDir, claudeJSONPath)
	if err != nil {
		return nil, err
	}

	// Compare current vs saved, respecting skipPluginDiff
	return compare(savedProfile, current), nil
}

// CompareWithScope compares a saved profile with the current state at a specific scope
func CompareWithScope(savedProfile *Profile, claudeDir, claudeJSONPath, projectDir, scope string) (*ProfileDiff, error) {
	// Validate scope
	if err := claude.ValidateScope(scope); err != nil {
		return nil, err
	}

	// Create snapshot of current state at the specified scope
	current, err := SnapshotWithScope("", claudeDir, claudeJSONPath, SnapshotOptions{
		Scope:      scope,
		ProjectDir: projectDir,
	})
	if err != nil {
		return nil, err
	}

	// Compare current vs saved, respecting skipPluginDiff
	return compare(savedProfile, current), nil
}

// IsActiveProfileModified checks if the active profile has unsaved changes
// Returns (hasChanges, comparisonError)
// - hasChanges: true if profile exists and has modifications
// - comparisonError: non-nil if comparison failed (file read error, corrupt data, etc.)
// Gracefully returns (false, err) on any errors
func IsActiveProfileModified(activeProfileName, profilesDir, claudeDir, claudeJSONPath string) (bool, error) {
	if activeProfileName == "" {
		return false, nil
	}

	// Try to load the profile (disk first, then embedded)
	savedProfile, err := Load(profilesDir, activeProfileName)
	if err != nil {
		// Try embedded profile
		savedProfile, err = GetEmbeddedProfile(activeProfileName)
		if err != nil {
			return false, fmt.Errorf("failed to load profile %q: %w", activeProfileName, err)
		}
	}

	// Compare with current state
	diff, err := CompareWithCurrent(savedProfile, claudeDir, claudeJSONPath)
	if err != nil {
		return false, fmt.Errorf("failed to compare profile %q: %w", activeProfileName, err)
	}

	return diff.HasChanges(), nil
}

// IsProfileModifiedAtScope checks if a profile has unsaved changes at a specific scope
// Returns (hasChanges, comparisonError)
// - hasChanges: true if profile exists and has modifications at the specified scope
// - comparisonError: non-nil if comparison failed (file read error, corrupt data, etc.)
// Gracefully returns (false, err) on any errors
func IsProfileModifiedAtScope(profileName, profilesDir, claudeDir, claudeJSONPath, projectDir, scope string) (bool, error) {
	if profileName == "" {
		return false, nil
	}

	// Try to load the profile (disk first, then embedded)
	savedProfile, err := Load(profilesDir, profileName)
	if err != nil {
		// Try embedded profile
		savedProfile, err = GetEmbeddedProfile(profileName)
		if err != nil {
			return false, fmt.Errorf("failed to load profile %q: %w", profileName, err)
		}
	}

	// Compare with current state at the specified scope
	diff, err := CompareWithScope(savedProfile, claudeDir, claudeJSONPath, projectDir, scope)
	if err != nil {
		return false, fmt.Errorf("failed to compare profile %q at scope %s: %w", profileName, scope, err)
	}

	return diff.HasChanges(), nil
}

// compare compares a saved profile with current state and returns differences
func compare(saved, current *Profile) *ProfileDiff {
	diff := &ProfileDiff{}

	// Compare plugins (unless skipPluginDiff is set)
	if !saved.SkipPluginDiff {
		diff.PluginsAdded, diff.PluginsRemoved = compareStringSlices(saved.Plugins, current.Plugins)
	}

	// Compare marketplaces
	diff.MarketplacesAdded, diff.MarketplacesRemoved = compareMarketplaces(saved.Marketplaces, current.Marketplaces)

	// Compare MCP servers
	diff.MCPServersAdded, diff.MCPServersRemoved, diff.MCPServersModified = compareMCPServers(saved.MCPServers, current.MCPServers)

	return diff
}

// compareStringSlices compares two string slices and returns added and removed items
func compareStringSlices(saved, current []string) (added, removed []string) {
	savedSet := make(map[string]bool)
	for _, s := range saved {
		savedSet[s] = true
	}

	currentSet := make(map[string]bool)
	for _, s := range current {
		currentSet[s] = true
	}

	// Find added items (in current but not in saved)
	for _, s := range current {
		if !savedSet[s] {
			added = append(added, s)
		}
	}

	// Find removed items (in saved but not in current)
	for _, s := range saved {
		if !currentSet[s] {
			removed = append(removed, s)
		}
	}

	return added, removed
}

// compareMarketplaces compares two marketplace slices and returns added and removed
func compareMarketplaces(saved, current []Marketplace) (added, removed []Marketplace) {
	savedMap := make(map[string]Marketplace)
	for _, m := range saved {
		savedMap[m.DisplayName()] = m
	}

	currentMap := make(map[string]Marketplace)
	for _, m := range current {
		currentMap[m.DisplayName()] = m
	}

	// Find added marketplaces
	for name, m := range currentMap {
		if _, exists := savedMap[name]; !exists {
			added = append(added, m)
		}
	}

	// Find removed marketplaces
	for name, m := range savedMap {
		if _, exists := currentMap[name]; !exists {
			removed = append(removed, m)
		}
	}

	return added, removed
}

// compareMCPServers compares MCP servers and returns added, removed, and modified
func compareMCPServers(saved, current []MCPServer) (added, removed, modified []MCPServer) {
	savedMap := make(map[string]MCPServer)
	for _, s := range saved {
		savedMap[s.Name] = s
	}

	currentMap := make(map[string]MCPServer)
	for _, s := range current {
		currentMap[s.Name] = s
	}

	// Find added and modified servers
	for name, curr := range currentMap {
		if sav, exists := savedMap[name]; !exists {
			// Server added
			added = append(added, curr)
		} else if !mcpServersEqual(sav, curr) {
			// Server modified (same name but different command/args)
			modified = append(modified, curr)
		}
	}

	// Find removed servers
	for name, sav := range savedMap {
		if _, exists := currentMap[name]; !exists {
			removed = append(removed, sav)
		}
	}

	return added, removed, modified
}

// mcpServersEqual checks if two MCP servers are equal
// Compares: command, args, scope, and secrets
func mcpServersEqual(a, b MCPServer) bool {
	// Compare command
	if a.Command != b.Command {
		return false
	}

	// Compare scope
	if a.Scope != b.Scope {
		return false
	}

	// Compare args
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if a.Args[i] != b.Args[i] {
			return false
		}
	}

	// Compare secrets
	if len(a.Secrets) != len(b.Secrets) {
		return false
	}
	for key, aSecret := range a.Secrets {
		bSecret, exists := b.Secrets[key]
		if !exists {
			return false
		}
		if !secretRefsEqual(aSecret, bSecret) {
			return false
		}
	}

	return true
}

// secretRefsEqual compares two SecretRef values
func secretRefsEqual(a, b SecretRef) bool {
	if a.Description != b.Description {
		return false
	}

	if len(a.Sources) != len(b.Sources) {
		return false
	}

	for i := range a.Sources {
		if a.Sources[i] != b.Sources[i] {
			return false
		}
	}

	return true
}
