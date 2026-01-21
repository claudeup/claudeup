// ABOUTME: CLI command for searching plugins by capability
// ABOUTME: Integrates pluginsearch package with Cobra CLI
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/internal/claude"
	"github.com/claudeup/claudeup/v2/internal/pluginsearch"
	"github.com/spf13/cobra"
)

var (
	searchAll         bool
	searchType        string
	searchMarketplace string
	searchByComponent bool
	searchContent     bool
	searchRegex       bool
	searchFormat      string
)

var pluginSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search plugins by capability",
	Long: `Search across installed plugins to find those with specific capabilities.

By default, searches only installed plugins. Use --all to search the entire
plugin cache (all synced marketplaces).

Searches plugin names, descriptions, keywords, and component names/descriptions.`,
	Example: `  # Find TDD-related plugins
  claudeup plugin search tdd

  # Search all cached plugins for skill-creation capabilities
  claudeup plugin search "skill" --all --type skills --by-component

  # Find commit commands in a specific marketplace
  claudeup plugin search commit --type commands --marketplace superpowers-marketplace

  # Regex search
  claudeup plugin search "front.?end|react" --regex --all`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginSearch,
}

func init() {
	pluginCmd.AddCommand(pluginSearchCmd)

	pluginSearchCmd.Flags().BoolVar(&searchAll, "all", false, "Search all cached plugins, not just installed")
	pluginSearchCmd.Flags().StringVar(&searchType, "type", "", "Filter by component type: skills, commands, agents")
	pluginSearchCmd.Flags().StringVar(&searchMarketplace, "marketplace", "", "Limit to specific marketplace")
	pluginSearchCmd.Flags().BoolVar(&searchByComponent, "by-component", false, "Group results by component type")
	pluginSearchCmd.Flags().BoolVar(&searchContent, "content", false, "Also search SKILL.md body content")
	pluginSearchCmd.Flags().BoolVar(&searchRegex, "regex", false, "Treat query as regular expression")
	pluginSearchCmd.Flags().StringVar(&searchFormat, "format", "", "Output format: json, table")
}

func runPluginSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Validate --type flag
	if searchType != "" && searchType != "skills" && searchType != "commands" && searchType != "agents" {
		return fmt.Errorf("invalid --type: must be skills, commands, or agents")
	}

	// Warn about unimplemented --content flag
	if searchContent {
		fmt.Fprintln(os.Stderr, "Warning: --content flag is not yet implemented, searching metadata only")
	}

	// Determine cache directory
	cacheDir := filepath.Join(claudeDir, "plugins", "cache")

	// Check cache exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin cache not found at %s\n\nRun 'claude marketplace sync' to populate the cache", cacheDir)
	}

	// Build index using scanner
	scanner := pluginsearch.NewScanner()
	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to scan plugin cache: %w", err)
	}

	// If not --all, filter to installed plugins only
	if !searchAll {
		installed, err := claude.LoadPlugins(claudeDir)
		if err != nil {
			return fmt.Errorf("failed to load installed plugins: %w", err)
		}

		// Filter plugins to only those that are installed
		var installedPlugins []pluginsearch.PluginSearchIndex
		for _, plugin := range plugins {
			fullName := plugin.Name + "@" + plugin.Marketplace
			if installed.PluginExists(fullName) {
				installedPlugins = append(installedPlugins, plugin)
			}
		}

		if len(installedPlugins) == 0 {
			return fmt.Errorf("no installed plugins found\n\nInstall plugins first or use --all to search all cached plugins")
		}

		plugins = installedPlugins
	}

	// Build search options
	searchOpts := pluginsearch.SearchOptions{
		UseRegex:      searchRegex,
		FilterType:    searchType,
		FilterMarket:  searchMarketplace,
		SearchContent: searchContent,
	}

	// Search
	matcher := pluginsearch.NewMatcher()
	results := matcher.Search(plugins, query, searchOpts)

	// Build format options
	formatOpts := pluginsearch.FormatOptions{
		Format:      searchFormat,
		ByComponent: searchByComponent,
	}

	// Render results
	formatter := pluginsearch.NewFormatter(os.Stdout)
	formatter.Render(results, query, formatOpts)

	return nil
}
