// ABOUTME: CLI commands for managing Claude Code extensions
// ABOUTME: Provides list, enable, disable, view, sync, and import subcommands
package commands

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/local"
	"github.com/claudeup/claudeup/v5/internal/ui"
	"github.com/spf13/cobra"
)

// markdownCategories are categories where all items are markdown files.
var markdownCategories = map[string]bool{
	local.CategoryAgents: true,
	local.CategorySkills: true,
	local.CategoryRules:  true,
}

var (
	extFilterEnabled  bool
	extFilterDisabled bool
	extListFull       bool
	extListLong       bool
	extViewRaw        bool
)

var extensionsCmd = &cobra.Command{
	Use:     "extensions",
	Aliases: []string{"ext"},
	Short:   "Manage extensions (agents, commands, skills, hooks, rules, output-styles)",
	Long: `Manage Claude Code extensions stored in ~/.claudeup/ext.

These are files (not marketplace plugins) that extend Claude Code
with custom agents, commands, skills, hooks, rules, and output-styles.

Adding extensions:
  install     Copy items from external paths (git repos, downloads)
  import      Move items from active directories to extension storage
  import-all  Bulk import across all categories at once

Removing extensions:
  uninstall   Remove extensions and clean up symlinks`,
}

var extensionsListCmd = &cobra.Command{
	Use:   "list [category]",
	Short: "List extensions and their enabled status",
	Long: `List all extensions and their enabled status.

Without arguments, shows a summary with item counts per category.
Use a category argument to see individual items, or --full to show all items.
Use --enabled or --disabled to filter by status (implies full listing).
Use --long (-l) to show file type and path for each item.`,
	Example: `  claudeup extensions list
  claudeup extensions list agents
  claudeup extensions list --full
  claudeup extensions list --enabled
  claudeup extensions list hooks --disabled`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtensionsList,
}

var extensionsEnableCmd = &cobra.Command{
	Use:   "enable <category> <items...>",
	Short: "Enable extensions",
	Long: `Enable one or more extensions by creating symlinks.

Supports wildcards:
  - gsd-* matches items starting with "gsd-"
  - gsd/* matches all items in the "gsd/" directory
  - * matches all items in the category`,
	Example: `  claudeup extensions enable agents gsd-*
  claudeup extensions enable commands gsd/*
  claudeup extensions enable hooks format-on-save`,
	Args: cobra.MinimumNArgs(2),
	RunE: runExtensionsEnable,
}

var extensionsDisableCmd = &cobra.Command{
	Use:   "disable <category> <items...>",
	Short: "Disable extensions",
	Long: `Disable one or more extensions by removing symlinks.

Supports the same wildcards as enable.`,
	Example: `  claudeup extensions disable agents gsd-*
  claudeup extensions disable hooks gsd-check-update`,
	Args: cobra.MinimumNArgs(2),
	RunE: runExtensionsDisable,
}

var extensionsViewCmd = &cobra.Command{
	Use:   "view <category> <item>",
	Short: "View contents of an extension",
	Long: `Display the contents of an extension.

Markdown files are rendered for the terminal. Use --raw for
unformatted output (useful for piping to other tools like glow or bat).`,
	Example: `  claudeup extensions view agents gsd-planner
  claudeup extensions view hooks format-on-save
  claudeup extensions view skills bash
  claudeup extensions view agents gsd-planner --raw
  claudeup extensions view agents gsd-planner --raw | glow`,
	Args: cobra.ExactArgs(2),
	RunE: runExtensionsView,
}

var extensionsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync symlinks from enabled.json",
	Long:  `Recreate all symlinks based on the enabled.json configuration.`,
	Args:  cobra.NoArgs,
	RunE:  runExtensionsSync,
}

var extensionsImportCmd = &cobra.Command{
	Use:   "import <category> <items...>",
	Short: "Import items from active directory to extension storage",
	Long: `Import items that were installed directly to active directories (like GSD).

This command moves files from ~/.claude/<category>/ to ~/.claudeup/ext/<category>/
and creates symlinks back, enabling management via claudeup.

This is useful when tools install directly to active directories instead of extension storage.
Existing symlinks (already managed extensions) are skipped.

Supports wildcards:
  - gsd-* matches items starting with "gsd-"
  - * matches all non-symlink items in the category`,
	Example: `  claudeup extensions import agents gsd-*
  claudeup extensions import commands gsd
  claudeup extensions import hooks gsd-*`,
	Args: cobra.MinimumNArgs(2),
	RunE: runExtensionsImport,
}

var extensionsImportAllCmd = &cobra.Command{
	Use:   "import-all [patterns...]",
	Short: "Import items from all categories to extension storage",
	Long: `Import items from all active directories to extension storage.

Scans all category directories (agents, commands, skills, hooks, rules, output-styles)
for items that are not already symlinks, moves them to extension storage, and enables them.

If patterns are provided, only items matching the patterns are imported.
Without patterns, all non-symlink items are imported.`,
	Example: `  claudeup extensions import-all           # Import everything
  claudeup extensions import-all gsd-* gsd # Import only GSD items`,
	RunE: runExtensionsImportAll,
}

var extensionsUninstallCmd = &cobra.Command{
	Use:   "uninstall <category> <items...>",
	Short: "Remove extensions",
	Long: `Remove extensions and clean up symlinks.

For each matched item: removes the symlink from the active directory,
deletes the file from extension storage, and removes the config entry.

Supports the same wildcards as enable/disable.`,
	Example: `  claudeup extensions uninstall rules my-rule.md
  claudeup extensions uninstall agents gsd-*
  claudeup extensions uninstall hooks format-on-save`,
	Args: cobra.MinimumNArgs(2),
	RunE: runExtensionsUninstall,
}

var extensionsInstallCmd = &cobra.Command{
	Use:   "install <category> <path>",
	Short: "Install extensions from an external path",
	Long: `Install extensions from an external path (file or directory).

This copies files to extension storage and automatically enables them.
Use this to install extensions from a git repo, downloads folder, or other location.

For single files/directories: installed as-is.
For directories containing multiple items: each item is installed individually.

Existing items with the same name are skipped (not overwritten).`,
	Example: `  claudeup extensions install agents ~/code/my-agents/
  claudeup extensions install hooks ~/Downloads/format-on-save.sh
  claudeup extensions install skills ~/code/my-skills/awesome-skill`,
	Args: cobra.ExactArgs(2),
	RunE: runExtensionsInstall,
}

func init() {
	rootCmd.AddCommand(extensionsCmd)
	extensionsCmd.AddCommand(extensionsListCmd)
	extensionsCmd.AddCommand(extensionsEnableCmd)
	extensionsCmd.AddCommand(extensionsDisableCmd)
	extensionsCmd.AddCommand(extensionsViewCmd)
	extensionsCmd.AddCommand(extensionsSyncCmd)
	extensionsCmd.AddCommand(extensionsImportCmd)
	extensionsCmd.AddCommand(extensionsImportAllCmd)
	extensionsCmd.AddCommand(extensionsInstallCmd)
	extensionsCmd.AddCommand(extensionsUninstallCmd)

	extensionsListCmd.Flags().BoolVarP(&extFilterEnabled, "enabled", "e", false, "Show only enabled items")
	extensionsListCmd.Flags().BoolVarP(&extFilterDisabled, "disabled", "d", false, "Show only disabled items")
	extensionsListCmd.Flags().BoolVar(&extListFull, "full", false, "Show all items instead of summary counts")
	extensionsListCmd.Flags().BoolVarP(&extListLong, "long", "l", false, "Show file type and path for each item")
	extensionsViewCmd.Flags().BoolVar(&extViewRaw, "raw", false, "Output raw content without rendering")
}

type itemStatus struct {
	name    string
	enabled bool
}

func runExtensionsList(cmd *cobra.Command, args []string) error {
	if extFilterEnabled && extFilterDisabled {
		return fmt.Errorf("--enabled and --disabled are mutually exclusive")
	}

	manager := local.NewManager(claudeDir, claudeupHome)
	config, err := manager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	specificCategory := len(args) > 0
	hasFilter := extFilterEnabled || extFilterDisabled
	showSummary := !specificCategory && !hasFilter && !extListFull && !extListLong

	var categories []string
	if specificCategory {
		if err := local.ValidateCategory(args[0]); err != nil {
			return err
		}
		categories = []string{args[0]}
	} else {
		categories = local.AllCategories()
	}

	totalItems := 0
	categoriesWithItems := 0
	for _, category := range categories {
		items, err := manager.ListItems(category)
		if err != nil {
			continue
		}

		catConfig := config[category]
		if catConfig == nil {
			catConfig = make(map[string]bool)
		}

		totalItems += len(items)

		if showSummary {
			if len(items) == 0 {
				continue
			}
			categoriesWithItems++
			enabledCount := 0
			for _, item := range items {
				if catConfig[item] {
					enabledCount++
				}
			}
			fmt.Printf("  %s/:  %d items (%d enabled)\n", category, len(items), enabledCount)
			continue
		}

		// Full listing mode
		var filtered []itemStatus
		for _, item := range items {
			enabled := catConfig[item]
			if extFilterEnabled && !enabled {
				continue
			}
			if extFilterDisabled && enabled {
				continue
			}
			filtered = append(filtered, itemStatus{item, enabled})
		}

		if len(filtered) == 0 {
			if specificCategory {
				fmt.Printf("\n%s/: (empty)\n", category)
			}
			continue
		}

		fmt.Printf("\n%s/:\n", category)

		if category == local.CategoryAgents {
			// Group agents by their group directory
			printGroupedAgents(filtered, extListLong, category)
		} else {
			for _, item := range filtered {
				status := ui.Muted("·")
				if item.enabled {
					status = ui.Success(ui.SymbolSuccess)
				}
				if extListLong {
					fileType := fileTypeLabel(item.name)
					relPath := filepath.Join("ext", category, item.name)
					fmt.Printf("  %s %-30s [%s]  %s\n", status, item.name, fileType, relPath)
				} else {
					fmt.Printf("  %s %s\n", status, item.name)
				}
			}
		}

		// Per-category count line
		enabledCount := 0
		for _, item := range items {
			if catConfig[item] {
				enabledCount++
			}
		}
		fmt.Printf("  %d items (%d enabled)\n", len(items), enabledCount)
	}

	if totalItems == 0 && len(args) == 0 {
		fmt.Println("No extensions found. Use 'claudeup extensions install' or 'claudeup extensions import' to add items.")
	} else if showSummary && categoriesWithItems > 0 {
		categoryWord := "categories"
		if categoriesWithItems == 1 {
			categoryWord = "category"
		}
		fmt.Printf("\n  Total: %d items across %d %s\n", totalItems, categoriesWithItems, categoryWord)
	}

	return nil
}

func printGroupedAgents(items []itemStatus, long bool, category string) {
	// Group by directory
	groups := make(map[string][]itemStatus)
	var flatItems []itemStatus

	for _, item := range items {
		if strings.Contains(item.name, "/") {
			parts := strings.SplitN(item.name, "/", 2)
			group := parts[0]
			groups[group] = append(groups[group], itemStatus{
				name:    parts[1],
				enabled: item.enabled,
			})
		} else {
			flatItems = append(flatItems, item)
		}
	}

	// Print flat items first
	for _, item := range flatItems {
		status := ui.Muted("·")
		if item.enabled {
			status = ui.Success(ui.SymbolSuccess)
		}
		if long {
			fileType := fileTypeLabel(item.name)
			relPath := filepath.Join("ext", category, item.name)
			fmt.Printf("  %s %-30s [%s]  %s\n", status, item.name, fileType, relPath)
		} else {
			fmt.Printf("  %s %s\n", status, item.name)
		}
	}

	// Print grouped items
	groupNames := make([]string, 0, len(groups))
	for g := range groups {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	for _, group := range groupNames {
		fmt.Printf("  %s/\n", group)
		for _, item := range groups[group] {
			status := ui.Muted("·")
			if item.enabled {
				status = ui.Success(ui.SymbolSuccess)
			}
			displayName := strings.TrimSuffix(item.name, ".md")
			if long {
				fullName := group + "/" + item.name
				fileType := fileTypeLabel(item.name)
				relPath := filepath.Join("ext", category, fullName)
				fmt.Printf("    %s %-28s [%s]  %s\n", status, displayName, fileType, relPath)
			} else {
				fmt.Printf("    %s %s\n", status, displayName)
			}
		}
	}
}

// fileTypeLabel returns a short label for the file type based on extension.
func fileTypeLabel(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".sh", ".bash":
		return "bash"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".md":
		return "markdown"
	default:
		if ext != "" {
			return ext[1:] // strip the dot
		}
		return "directory"
	}
}

func runExtensionsEnable(cmd *cobra.Command, args []string) error {
	category := args[0]
	patterns := args[1:]

	manager := local.NewManager(claudeDir, claudeupHome)
	enabled, notFound, err := manager.Enable(category, patterns)
	if err != nil {
		return err
	}

	for _, item := range enabled {
		ui.PrintSuccess(fmt.Sprintf("Enabled: %s/%s", category, item))
	}

	for _, pattern := range notFound {
		ui.PrintWarning(fmt.Sprintf("Not found: %s/%s", category, pattern))
	}

	if len(notFound) > 0 && len(enabled) == 0 {
		return fmt.Errorf("no items found matching patterns")
	}

	return nil
}

func runExtensionsDisable(cmd *cobra.Command, args []string) error {
	category := args[0]
	patterns := args[1:]

	manager := local.NewManager(claudeDir, claudeupHome)
	disabled, notFound, err := manager.Disable(category, patterns)
	if err != nil {
		return err
	}

	for _, item := range disabled {
		ui.PrintSuccess(fmt.Sprintf("Disabled: %s/%s", category, item))
	}

	for _, pattern := range notFound {
		ui.PrintWarning(fmt.Sprintf("Not found: %s/%s", category, pattern))
	}

	if len(notFound) > 0 && len(disabled) == 0 {
		return fmt.Errorf("no items found matching patterns")
	}

	return nil
}

func runExtensionsView(cmd *cobra.Command, args []string) error {
	category := args[0]
	item := args[1]

	manager := local.NewManager(claudeDir, claudeupHome)
	content, err := manager.View(category, item)
	if err != nil {
		return err
	}

	isMarkdown := markdownCategories[category]
	if !isMarkdown {
		// For non-markdown categories, check the resolved filename
		if resolved, err := manager.ResolveItemName(category, item); err == nil {
			isMarkdown = strings.HasSuffix(resolved, ".md")
		}
	}

	if isMarkdown {
		fmt.Print(ui.RenderMarkdown(content, extViewRaw))
	} else {
		fmt.Println(content)
	}
	return nil
}

func runExtensionsSync(cmd *cobra.Command, args []string) error {
	manager := local.NewManager(claudeDir, claudeupHome)

	fmt.Println("Syncing extensions from enabled.json...")
	if err := manager.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	ui.PrintSuccess("Sync complete")
	return nil
}

func runExtensionsImport(cmd *cobra.Command, args []string) error {
	category := args[0]
	patterns := args[1:]

	manager := local.NewManager(claudeDir, claudeupHome)
	imported, skipped, notFound, err := manager.Import(category, patterns)
	if err != nil {
		return err
	}

	for _, item := range imported {
		ui.PrintSuccess(fmt.Sprintf("Imported: %s/%s", category, item))
	}

	for _, item := range skipped {
		ui.PrintSuccess(fmt.Sprintf("Linked (already installed): %s/%s", category, item))
	}

	for _, pattern := range notFound {
		ui.PrintWarning(fmt.Sprintf("Not found: %s/%s", category, pattern))
	}

	if len(notFound) > 0 && len(imported) == 0 && len(skipped) == 0 {
		return fmt.Errorf("no items found matching patterns")
	}

	return nil
}

func runExtensionsImportAll(cmd *cobra.Command, args []string) error {
	manager := local.NewManager(claudeDir, claudeupHome)

	var patterns []string
	if len(args) > 0 {
		patterns = args
	}

	imported, linked, err := manager.ImportAll(patterns)
	if err != nil {
		return err
	}

	totalProcessed := 0
	for category, items := range imported {
		for _, item := range items {
			ui.PrintSuccess(fmt.Sprintf("Imported: %s/%s", category, item))
			totalProcessed++
		}
	}
	for category, items := range linked {
		for _, item := range items {
			ui.PrintSuccess(fmt.Sprintf("Linked (already installed): %s/%s", category, item))
			totalProcessed++
		}
	}

	if totalProcessed == 0 {
		fmt.Println("No items to import (all items are already symlinks or no matching items found)")
	}

	return nil
}

func runExtensionsInstall(cmd *cobra.Command, args []string) error {
	category := args[0]
	sourcePath := args[1]

	manager := local.NewManager(claudeDir, claudeupHome)
	installed, skipped, err := manager.Install(category, sourcePath)
	if err != nil {
		return err
	}

	for _, item := range installed {
		ui.PrintSuccess(fmt.Sprintf("Installed: %s/%s", category, item))
	}

	for _, item := range skipped {
		ui.PrintWarning(fmt.Sprintf("Skipped (already exists): %s/%s", category, item))
	}

	if len(installed) == 0 && len(skipped) > 0 {
		fmt.Println("All items already installed")
	}

	return nil
}

func runExtensionsUninstall(cmd *cobra.Command, args []string) error {
	category := args[0]
	patterns := args[1:]

	manager := local.NewManager(claudeDir, claudeupHome)
	removed, notFound, err := manager.Uninstall(category, patterns)
	if err != nil {
		return err
	}

	for _, item := range removed {
		ui.PrintSuccess(fmt.Sprintf("Removed: %s/%s", category, item))
	}

	for _, pattern := range notFound {
		ui.PrintWarning(fmt.Sprintf("Not found: %s/%s", category, pattern))
	}

	if len(notFound) > 0 && len(removed) == 0 {
		return fmt.Errorf("no items found matching patterns")
	}

	return nil
}
