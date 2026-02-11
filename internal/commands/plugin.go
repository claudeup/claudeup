// ABOUTME: Plugin subcommand group for managing Claude Code plugins
// ABOUTME: Provides list, browse, show, and search subcommands
package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v4/internal/claude"
	"github.com/claudeup/claudeup/v4/internal/ui"
	"github.com/spf13/cobra"
)

var (
	pluginListSummary    bool
	pluginFilterEnabled  bool
	pluginFilterDisabled bool
	pluginListFormat     string
	pluginListByScope    bool
	pluginBrowseFormat   string
	pluginBrowseShow     string
	pluginShowRaw        bool
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long: `Manage Claude Code plugins - list, browse, show, and search.

Use 'claude plugin install' and 'claude plugin uninstall' to add or remove plugins.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `Display information about all installed plugins.

Shows each plugin's name, version, status, enabled scope, and active source
in a compact table format. Use --format detail for verbose per-plugin output.`,
	Args: cobra.NoArgs,
	RunE: runPluginList,
}

var pluginBrowseCmd = &cobra.Command{
	Use:   "browse <marketplace>",
	Short: "Browse available plugins in a marketplace",
	Long: `Display plugins available in a marketplace before installing.

Accepts marketplace name, repo (user/repo), or URL as identifier.`,
	Example: `  claudeup plugin browse claude-code-workflows
  claudeup plugin browse wshobson/agents
  claudeup plugin browse --format table my-marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginBrowse,
}

var pluginShowCmd = &cobra.Command{
	Use:   "show <plugin>@<marketplace> [file]",
	Short: "Show plugin contents",
	Long: `Display the directory structure or file contents of a plugin.

Without a file argument, shows the plugin directory tree.
With a file argument, displays the file contents. Markdown files are
rendered for the terminal; use --raw for unformatted output
(useful for piping to other tools like glow or bat).

File paths are relative to the plugin root. Extension inference is
supported (e.g. "agents/test" resolves to "agents/test.md").
Skill directories resolve to their SKILL.md file.`,
	Example: `  claudeup plugin show observability-monitoring@claude-code-workflows
  claudeup plugin show my-plugin@acme-marketplace agents/test
  claudeup plugin show my-plugin@acme-marketplace skills/awesome-skill
  claudeup plugin show my-plugin@acme-marketplace agents/test --raw
  claudeup plugin show my-plugin@acme-marketplace agents/test --raw | glow`,
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runPluginShow,
	ValidArgsFunction: pluginShowCompletionFunc,
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginBrowseCmd)
	pluginCmd.AddCommand(pluginShowCmd)

	pluginListCmd.Flags().BoolVar(&pluginListSummary, "summary", false, "Show only summary statistics")
	pluginListCmd.Flags().BoolVar(&pluginFilterEnabled, "enabled", false, "Show only enabled plugins")
	pluginListCmd.Flags().BoolVar(&pluginFilterDisabled, "disabled", false, "Show only disabled plugins")
	pluginListCmd.Flags().StringVar(&pluginListFormat, "format", "", "Output format (table, detail)")
	pluginListCmd.Flags().BoolVar(&pluginListByScope, "by-scope", false, "Group enabled plugins by scope")
	pluginBrowseCmd.Flags().StringVar(&pluginBrowseFormat, "format", "", "Output format (table)")
	pluginBrowseCmd.Flags().StringVar(&pluginBrowseShow, "show", "", "Show contents of a specific plugin")
	pluginShowCmd.Flags().BoolVar(&pluginShowRaw, "raw", false, "Output raw content without rendering")
}

func runPluginList(cmd *cobra.Command, args []string) error {
	// Validate mutually exclusive flags
	if pluginFilterEnabled && pluginFilterDisabled {
		return fmt.Errorf("--enabled and --disabled are mutually exclusive")
	}

	// Validate --by-scope incompatibilities
	if pluginListByScope {
		if pluginListSummary || pluginListFormat != "" || pluginFilterEnabled || pluginFilterDisabled {
			return fmt.Errorf("--by-scope cannot be combined with --summary, --format, --enabled, or --disabled")
		}
	}

	// Get current directory for project scope
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Handle --by-scope: show plugins grouped by scope
	if pluginListByScope {
		return RenderPluginsByScope(claudeDir, projectDir, "")
	}

	// Analyze plugins across all scopes (including orphan detection)
	result, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
	if err != nil {
		return fmt.Errorf("failed to analyze plugins: %w", err)
	}

	analysis := result.Installed

	// Sort plugin names for consistent output
	names := make([]string, 0, len(analysis))
	for name := range analysis {
		names = append(names, name)
	}
	sort.Strings(names)

	// Calculate statistics (before filtering)
	stats := calculatePluginStatistics(analysis)

	// Apply filters
	totalCount := len(names)
	filterLabel := ""
	if pluginFilterEnabled {
		filtered := make([]string, 0)
		for _, name := range names {
			if analysis[name].IsEnabled() {
				filtered = append(filtered, name)
			}
		}
		names = filtered
		filterLabel = "enabled"
	} else if pluginFilterDisabled {
		filtered := make([]string, 0)
		for _, name := range names {
			if !analysis[name].IsEnabled() {
				filtered = append(filtered, name)
			}
		}
		names = filtered
		filterLabel = "disabled"
	}

	// Display based on output mode
	if pluginListSummary {
		printPluginSummary(stats)
		printEnabledNotInstalled(result.EnabledNotInstalled)
		return nil
	}

	switch pluginListFormat {
	case "detail":
		printPluginDetails(names, analysis)
		printPluginListFooterFiltered(stats, len(names), totalCount, filterLabel)
	default:
		// Table is the default format
		printPluginTable(names, analysis)
	}

	printEnabledNotInstalled(result.EnabledNotInstalled)

	return nil
}

// formatScopeList formats a list of scopes as a comma-separated string
func formatScopeList(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	if len(scopes) == 1 {
		return scopes[0]
	}

	result := ""
	for i, scope := range scopes {
		if i > 0 {
			result += ", "
		}
		result += scope
	}
	return result
}

func runPluginBrowse(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Find the marketplace
	meta, marketplaceName, err := claude.FindMarketplace(claudeDir, identifier)
	if err != nil {
		// Build helpful error message
		registry, loadErr := claude.LoadMarketplaces(claudeDir)
		if loadErr != nil {
			return fmt.Errorf("marketplace %q not found\n\nTo add a marketplace:\n  claude marketplace add <repo-or-url>", identifier)
		}

		var installed []string
		for name, m := range registry {
			if m.Source.Repo != "" {
				installed = append(installed, fmt.Sprintf("  %s (%s)", name, m.Source.Repo))
			} else {
				installed = append(installed, fmt.Sprintf("  %s", name))
			}
		}
		sort.Strings(installed)

		msg := fmt.Sprintf("marketplace %q not found\n\nTo add a marketplace:\n  claude marketplace add <repo-or-url>", identifier)
		if len(installed) > 0 {
			msg += "\n\nInstalled marketplaces:\n" + strings.Join(installed, "\n")
		}
		return errors.New(msg)
	}

	// If --show flag is set, delegate to showPluginTree
	if pluginBrowseShow != "" {
		return showPluginTree(pluginBrowseShow, marketplaceName)
	}

	// Load the marketplace index
	index, err := claude.LoadMarketplaceIndex(meta.InstallLocation)
	if err != nil {
		return fmt.Errorf("marketplace %q has no plugin index\n\nThe marketplace at %s is missing .claude-plugin/marketplace.json", marketplaceName, meta.InstallLocation)
	}

	// Handle empty marketplace
	if len(index.Plugins) == 0 {
		fmt.Printf("No plugins available in %s\n", index.Name)
		return nil
	}

	// Load installed plugins to check status (non-fatal if unavailable)
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		// Can still show marketplace plugins, just without installation status
		plugins = nil
	}

	// Sort plugins alphabetically
	sortedPlugins := make([]claude.MarketplacePluginInfo, len(index.Plugins))
	copy(sortedPlugins, index.Plugins)
	sort.Slice(sortedPlugins, func(i, j int) bool {
		return sortedPlugins[i].Name < sortedPlugins[j].Name
	})

	// Display based on format
	switch pluginBrowseFormat {
	case "json":
		printBrowseJSON(sortedPlugins, index.Name, marketplaceName, plugins)
	case "table":
		printBrowseTable(sortedPlugins, index.Name, marketplaceName, plugins)
	default:
		printBrowseDefault(sortedPlugins, index.Name, marketplaceName, plugins)
	}

	return nil
}

func printBrowseDefault(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	fmt.Println(ui.RenderSection("Available in "+indexName, len(plugins)))
	fmt.Println()

	// Calculate max name width for alignment
	nameWidth := 20
	for _, p := range plugins {
		if len(p.Name) > nameWidth {
			nameWidth = len(p.Name)
		}
	}
	nameWidth += 2

	for _, p := range plugins {
		// Check if installed
		fullName := p.Name + "@" + marketplaceName
		var status string
		if installed != nil && installed.PluginExists(fullName) {
			status = ui.Success(ui.SymbolSuccess)
		}

		// Truncate description if needed (rune-safe for UTF-8)
		desc := p.Description
		descRunes := []rune(desc)
		if len(descRunes) > 60 {
			desc = string(descRunes[:57]) + "..."
		}

		// Format with styling - fixed width columns
		nameFmt := fmt.Sprintf("%%-%ds", nameWidth)
		nameCol := fmt.Sprintf(nameFmt, p.Name)
		descCol := fmt.Sprintf("%-60s", desc)
		versionCol := fmt.Sprintf("%-8s", p.Version)

		fmt.Printf("%s %s  %s %s\n",
			ui.Bold(nameCol),
			ui.Muted(descCol),
			ui.Muted(versionCol),
			status)
	}
}

func printBrowseTable(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	// Calculate max name width for alignment
	nameWidth := 6 // minimum "PLUGIN" length
	for _, p := range plugins {
		if len(p.Name) > nameWidth {
			nameWidth = len(p.Name)
		}
	}
	nameWidth += 2 // add padding

	descWidth := 60

	// Print header with bold styling
	headerFmt := fmt.Sprintf("%%-%ds %%-%ds %%-10s %%s", nameWidth, descWidth)
	header := fmt.Sprintf(headerFmt, "PLUGIN", "DESCRIPTION", "VERSION", "STATUS")
	fmt.Println(ui.Bold(header))
	fmt.Println(ui.Muted(strings.Repeat("─", nameWidth+descWidth+10+12)))

	// Print rows
	for _, p := range plugins {
		fullName := p.Name + "@" + marketplaceName

		// Truncate description (rune-safe for UTF-8)
		desc := p.Description
		descRunes := []rune(desc)
		if len(descRunes) > descWidth {
			desc = string(descRunes[:descWidth-3]) + "..."
		}

		// Format columns with padding first (before applying ANSI styles)
		nameFmt := fmt.Sprintf("%%-%ds", nameWidth)
		nameCol := fmt.Sprintf(nameFmt, p.Name)
		descCol := fmt.Sprintf("%-*s", descWidth, desc)
		versionCol := fmt.Sprintf("%-10s", p.Version)

		// Check installed status
		var statusCol string
		if installed != nil && installed.PluginExists(fullName) {
			statusCol = ui.Success("installed")
		}

		fmt.Printf("%s %s %s %s\n",
			ui.Bold(nameCol),
			ui.Muted(descCol),
			ui.Muted(versionCol),
			statusCol)
	}
}

func printBrowseJSON(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	type pluginOutput struct {
		Name        string `json:"name"`
		FullName    string `json:"fullName"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Installed   bool   `json:"installed"`
	}

	output := struct {
		Marketplace string         `json:"marketplace"`
		Count       int            `json:"count"`
		Plugins     []pluginOutput `json:"plugins"`
	}{
		Marketplace: indexName,
		Count:       len(plugins),
		Plugins:     make([]pluginOutput, len(plugins)),
	}

	for i, p := range plugins {
		fullName := p.Name + "@" + marketplaceName
		output.Plugins[i] = pluginOutput{
			Name:        p.Name,
			FullName:    fullName,
			Description: p.Description,
			Version:     p.Version,
			Installed:   installed != nil && installed.PluginExists(fullName),
		}
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func runPluginShow(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Parse plugin@marketplace
	parts := strings.SplitN(identifier, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid format: expected <plugin>@<marketplace>\n\nExample: claudeup plugin show my-plugin@my-marketplace")
	}

	pluginName := parts[0]
	marketplaceID := parts[1]

	// If a file argument is provided, show file contents
	if len(args) == 2 {
		loc, err := resolvePluginPath(claudeDir, pluginName, marketplaceID)
		if err != nil {
			return err
		}
		return showPluginFile(loc.Path, args[1], pluginShowRaw)
	}

	return showPluginTree(pluginName, marketplaceID)
}

// pluginLocation holds the resolved filesystem path and version of a plugin.
type pluginLocation struct {
	Path            string
	Version         string
	MarketplaceName string
}

// resolvePluginPath locates a plugin on disk given its name and marketplace identifier.
// It checks bundled (monorepo) and cache (external) locations.
func resolvePluginPath(claudeConfigDir, pluginName, marketplaceID string) (pluginLocation, error) {
	meta, marketplaceName, err := claude.FindMarketplace(claudeConfigDir, marketplaceID)
	if err != nil {
		return pluginLocation{}, fmt.Errorf("marketplace %q not found\n\nRun 'claudeup marketplace add %s' first", marketplaceID, marketplaceID)
	}

	index, err := claude.LoadMarketplaceIndex(meta.InstallLocation)
	if err != nil {
		return pluginLocation{}, fmt.Errorf("failed to load marketplace index: %w", err)
	}

	var indexVersion string
	var pluginFound bool
	for _, p := range index.Plugins {
		if p.Name == pluginName {
			indexVersion = p.Version
			pluginFound = true
			break
		}
	}

	if !pluginFound {
		return pluginLocation{}, fmt.Errorf("plugin %q not found in marketplace %q\n\nRun 'claudeup plugin browse %s' to see available plugins", pluginName, marketplaceName, marketplaceName)
	}

	// Check bundled location first (monorepo marketplaces)
	var pluginPath string
	var installedVersion string

	bundledPath := filepath.Join(meta.InstallLocation, "plugins", pluginName)
	if info, err := os.Stat(bundledPath); err == nil && info.IsDir() {
		pluginPath = bundledPath
		installedVersion = indexVersion
	}

	// Try cache location (external source marketplaces)
	if pluginPath == "" {
		cacheDir := filepath.Join(claudeConfigDir, "plugins", "cache", marketplaceName, pluginName)
		if entries, err := os.ReadDir(cacheDir); err == nil && len(entries) > 0 {
			for _, entry := range entries {
				if entry.IsDir() {
					installedVersion = entry.Name()
					pluginPath = filepath.Join(cacheDir, installedVersion)
					break
				}
			}
		}
	}

	if pluginPath == "" {
		return pluginLocation{}, fmt.Errorf("plugin %q is not cached locally\n\nThe marketplace index lists it, but it hasn't been downloaded.\nRun 'claudeup plugin install %s@%s' first", pluginName, pluginName, marketplaceName)
	}

	version := indexVersion
	if version == "" {
		version = installedVersion
	}

	return pluginLocation{
		Path:            pluginPath,
		Version:         version,
		MarketplaceName: marketplaceName,
	}, nil
}

func showPluginTree(pluginName, marketplaceID string) error {
	loc, err := resolvePluginPath(claudeDir, pluginName, marketplaceID)
	if err != nil {
		return err
	}

	// Generate tree
	tree, dirs, files := generateTree(loc.Path)

	// Print header
	fullName := pluginName + "@" + loc.MarketplaceName
	if loc.Version != "" {
		fmt.Printf("%s (v%s)\n\n", ui.Bold(fullName), loc.Version)
	} else {
		fmt.Printf("%s\n\n", ui.Bold(fullName))
	}

	// Print tree
	if tree == "" {
		fmt.Println("(empty)")
	} else {
		fmt.Print(tree)
	}

	// Print summary
	fmt.Println()
	dirWord := "directories"
	if dirs == 1 {
		dirWord = "directory"
	}
	fileWord := "files"
	if files == 1 {
		fileWord = "file"
	}
	fmt.Printf("%d %s, %d %s\n", dirs, dirWord, files, fileWord)

	return nil
}

// pluginShowCompletionFunc provides tab completion for plugin show command
// Format: <plugin>@<marketplace>
func pluginShowCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Load marketplaces
	registry, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	// Check if user has typed plugin@marketplace pattern
	if strings.Contains(toComplete, "@") {
		// Format: <plugin-prefix>@<marketplace-prefix>
		parts := strings.SplitN(toComplete, "@", 2)
		pluginPrefix := parts[0]
		marketplacePrefix := parts[1]

		// Search all marketplaces for matching plugins
		for marketplaceName, meta := range registry {
			// Filter by marketplace prefix (after @)
			if marketplacePrefix != "" && !strings.HasPrefix(marketplaceName, marketplacePrefix) {
				continue
			}

			// List plugins from this marketplace
			pluginsDir := filepath.Join(meta.InstallLocation, "plugins")
			entries, err := os.ReadDir(pluginsDir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				pluginName := entry.Name()
				// Filter by plugin prefix (before @)
				if pluginPrefix != "" && !strings.HasPrefix(pluginName, pluginPrefix) {
					continue
				}
				completions = append(completions, pluginName+"@"+marketplaceName)
			}
		}
	} else {
		// No @ yet, suggest marketplace names (helps user discover marketplaces)
		for name := range registry {
			completions = append(completions, name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
