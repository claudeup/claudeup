// ABOUTME: Plugin statistics calculation for the plugin list command
// ABOUTME: Provides PluginStatistics struct, calculation logic, and display functions
package commands

import (
	"fmt"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
)

// PluginStatistics holds aggregated counts for plugin analysis
type PluginStatistics struct {
	Total    int // Total number of unique plugins
	Cached   int // Plugins stored in ~/.claude/plugins/cache/
	Local    int // Plugins referenced from marketplace
	Enabled  int // Plugins enabled at any scope
	Disabled int // Plugins not enabled at any scope
	Stale    int // Plugins with missing install paths
}

// calculatePluginStatistics computes plugin counts from analysis data
func calculatePluginStatistics(analysis map[string]*claude.PluginScopeInfo) PluginStatistics {
	stats := PluginStatistics{
		Total: len(analysis),
	}

	for _, info := range analysis {
		// Determine primary installation (active source or first installation)
		var primaryInst *claude.PluginMetadata
		if info.ActiveSource != "" {
			primaryInst = info.GetInstallationForScope(info.ActiveSource)
		}
		if primaryInst == nil && len(info.InstalledAt) > 0 {
			primaryInst = &info.InstalledAt[0]
		}

		// Count by type (cached vs local)
		if primaryInst != nil {
			if primaryInst.IsLocal {
				stats.Local++
			} else {
				stats.Cached++
			}

			// Check for stale installations
			if !primaryInst.PathExists() {
				stats.Stale++
			}
		}

		// Count enabled/disabled
		if info.IsEnabled() {
			stats.Enabled++
		} else {
			stats.Disabled++
		}
	}

	return stats
}

// printPluginSummary displays summary statistics for plugins
func printPluginSummary(stats PluginStatistics) {
	fmt.Println(ui.RenderHeader("Plugin Summary"))
	fmt.Println()
	fmt.Println(ui.RenderDetail("Total", fmt.Sprintf("%d plugins", stats.Total)))
	fmt.Println(ui.RenderDetail("Enabled", fmt.Sprintf("%d", stats.Enabled)))
	if stats.Disabled > 0 {
		fmt.Println(ui.RenderDetail("Disabled", fmt.Sprintf("%d", stats.Disabled)))
	}
	if stats.Stale > 0 {
		fmt.Println(ui.RenderDetail("Stale", fmt.Sprintf("%d", stats.Stale)))
	}
	fmt.Println()
	fmt.Println(ui.Bold("By Type:"))
	fmt.Println(ui.Indent(fmt.Sprintf("Cached: %d %s", stats.Cached, ui.Muted("(copied to ~/.claude/plugins/cache/)")), 1))
	fmt.Println(ui.Indent(fmt.Sprintf("Local:  %d %s", stats.Local, ui.Muted("(referenced from marketplace)")), 1))
}

// printPluginDetails displays detailed information for each plugin
func printPluginDetails(names []string, analysis map[string]*claude.PluginScopeInfo) {
	fmt.Println(ui.RenderSection("Installed Plugins", len(names)))
	fmt.Println()

	for _, name := range names {
		info := analysis[name]
		printSinglePlugin(name, info)
	}
}

// printSinglePlugin displays details for one plugin
func printSinglePlugin(name string, info *claude.PluginScopeInfo) {
	var statusSymbol, statusText string

	// Check if any installation is stale
	hasStale := false
	for _, inst := range info.InstalledAt {
		if !inst.PathExists() {
			hasStale = true
			break
		}
	}

	if hasStale {
		statusSymbol = ui.Error(ui.SymbolError)
		statusText = ui.Error("stale (path not found)")
	} else if info.IsEnabled() {
		statusSymbol = ui.Success(ui.SymbolSuccess)
		statusText = "enabled"
	} else {
		statusSymbol = ui.Error(ui.SymbolError)
		statusText = "disabled"
	}

	// Get version from active source or first installation
	version := ""
	if info.ActiveSource != "" {
		if activeInst := info.GetInstallationForScope(info.ActiveSource); activeInst != nil {
			version = activeInst.Version
		}
	}
	if version == "" && len(info.InstalledAt) > 0 {
		version = info.InstalledAt[0].Version
	}

	fmt.Printf("%s %s\n", statusSymbol, ui.Bold(name))

	if version != "" {
		fmt.Println(ui.Indent(ui.RenderDetail("Version", version), 1))
	}

	fmt.Println(ui.Indent(ui.RenderDetail("Status", statusText), 1))

	// Show scope information
	if len(info.EnabledAt) > 0 {
		enabledAtText := formatScopeList(info.EnabledAt)
		fmt.Println(ui.Indent(ui.RenderDetail("Enabled at", enabledAtText), 1))
	}

	if info.ActiveSource != "" {
		fmt.Println(ui.Indent(ui.RenderDetail("Active source", info.ActiveSource), 1))
	}

	// Show all installation locations (deduplicated)
	if len(info.InstalledAt) > 1 {
		printOtherInstallations(info)
	}

	// Show primary installation path
	printInstallationPath(info)

	fmt.Println()
}

// printOtherInstallations shows additional scopes where the plugin is installed
func printOtherInstallations(info *claude.PluginScopeInfo) {
	// Use map to deduplicate scopes
	otherScopesMap := make(map[string]bool)
	for _, inst := range info.InstalledAt {
		if inst.Scope != info.ActiveSource {
			otherScopesMap[inst.Scope] = true
		}
	}

	// Convert to sorted slice
	otherInstalls := make([]string, 0, len(otherScopesMap))
	for scope := range otherScopesMap {
		otherInstalls = append(otherInstalls, scope)
	}
	claude.SortScopesByPrecedence(otherInstalls)

	if len(otherInstalls) > 0 {
		fmt.Println(ui.Indent(ui.RenderDetail("Also installed at", formatScopeList(otherInstalls)), 1))
	}
}

// printInstallationPath shows the primary installation path and metadata
func printInstallationPath(info *claude.PluginScopeInfo) {
	var inst *claude.PluginMetadata

	if info.ActiveSource != "" {
		inst = info.GetInstallationForScope(info.ActiveSource)
	} else if len(info.InstalledAt) > 0 {
		inst = &info.InstalledAt[0]
	}

	if inst == nil {
		return
	}

	fmt.Println(ui.Indent(ui.RenderDetail("Path", inst.InstallPath), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("Installed", inst.InstalledAt), 1))

	pluginType := "cached"
	if inst.IsLocal {
		pluginType = "local"
	}
	fmt.Println(ui.Indent(ui.RenderDetail("Type", pluginType), 1))
}

// printPluginListFooter displays the summary footer after plugin details
func printPluginListFooter(stats PluginStatistics) {
	printPluginListFooterFiltered(stats, stats.Total, stats.Total, "")
}

// printPluginListFooterFiltered displays the summary footer with filter info
func printPluginListFooterFiltered(stats PluginStatistics, shown int, total int, filterLabel string) {
	fmt.Println(ui.RenderSection("Summary", -1))
	if filterLabel != "" {
		fmt.Printf("Showing: %d %s (of %d total)\n", shown, filterLabel, total)
	} else {
		fmt.Printf("Total: %d plugins (%d cached, %d local)\n", stats.Total, stats.Cached, stats.Local)
	}
	if stats.Stale > 0 {
		ui.PrintWarning(fmt.Sprintf("%d stale plugins detected", stats.Stale))
	}
}
