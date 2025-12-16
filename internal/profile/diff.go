// ABOUTME: Profile comparison and diff functionality
// ABOUTME: Detects changes between saved profiles and current Claude state
package profile

import (
	"fmt"
	"strings"
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

// mcpServersEqual checks if two MCP servers are equal (same command and args)
func mcpServersEqual(a, b MCPServer) bool {
	if a.Command != b.Command {
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

	return true
}
