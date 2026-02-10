// ABOUTME: CLI commands for managing local Claude Code extensions
// ABOUTME: Provides list, enable, disable, view, sync, and import subcommands
package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v4/internal/local"
	"github.com/claudeup/claudeup/v4/internal/ui"
	"github.com/spf13/cobra"
)

// markdownCategories are categories where all items are markdown files.
var markdownCategories = map[string]bool{
	local.CategoryAgents: true,
	local.CategorySkills: true,
	local.CategoryRules:  true,
}

var (
	localFilterEnabled  bool
	localFilterDisabled bool
	localViewRaw        bool
)

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Manage local extensions (agents, commands, skills, hooks, rules, output-styles)",
	Long: `Manage local Claude Code extensions from ~/.claudeup/local.

These are local files (not marketplace plugins) that extend Claude Code
with custom agents, commands, skills, hooks, rules, and output-styles.`,
}

var localListCmd = &cobra.Command{
	Use:   "list [category]",
	Short: "List local items and their enabled status",
	Long: `List all local items and their enabled status.

Optionally filter by category. Use --enabled or --disabled to filter by status.`,
	Example: `  claudeup local list
  claudeup local list agents
  claudeup local list --enabled
  claudeup local list hooks --disabled`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLocalList,
}

var localEnableCmd = &cobra.Command{
	Use:   "enable <category> <items...>",
	Short: "Enable local items",
	Long: `Enable one or more local items by creating symlinks.

Supports wildcards:
  - gsd-* matches items starting with "gsd-"
  - gsd/* matches all items in the "gsd/" directory
  - * matches all items in the category`,
	Example: `  claudeup local enable agents gsd-*
  claudeup local enable commands gsd/*
  claudeup local enable hooks format-on-save`,
	Args: cobra.MinimumNArgs(2),
	RunE: runLocalEnable,
}

var localDisableCmd = &cobra.Command{
	Use:   "disable <category> <items...>",
	Short: "Disable local items",
	Long: `Disable one or more local items by removing symlinks.

Supports the same wildcards as enable.`,
	Example: `  claudeup local disable agents gsd-*
  claudeup local disable hooks gsd-check-update`,
	Args: cobra.MinimumNArgs(2),
	RunE: runLocalDisable,
}

var localViewCmd = &cobra.Command{
	Use:   "view <category> <item>",
	Short: "View contents of a local item",
	Long: `Display the contents of a local item.

Markdown files are rendered for the terminal. Use --raw for
unformatted output (useful for piping to other tools like glow or bat).`,
	Example: `  claudeup local view agents gsd-planner
  claudeup local view hooks format-on-save
  claudeup local view skills bash
  claudeup local view agents gsd-planner --raw
  claudeup local view agents gsd-planner --raw | glow`,
	Args: cobra.ExactArgs(2),
	RunE: runLocalView,
}

var localSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync symlinks from enabled.json",
	Long:  `Recreate all symlinks based on the enabled.json configuration.`,
	Args:  cobra.NoArgs,
	RunE:  runLocalSync,
}

var localImportCmd = &cobra.Command{
	Use:   "import <category> <items...>",
	Short: "Import items from active directory to local storage",
	Long: `Import items that were installed directly to active directories (like GSD).

This command moves files from ~/.claude/<category>/ to ~/.claudeup/local/<category>/
and creates symlinks back, enabling management via claudeup.

This is useful when tools install directly to active directories instead of local storage.
Existing symlinks (already managed items) are skipped.

Supports wildcards:
  - gsd-* matches items starting with "gsd-"
  - * matches all non-symlink items in the category`,
	Example: `  claudeup local import agents gsd-*
  claudeup local import commands gsd
  claudeup local import hooks gsd-*`,
	Args: cobra.MinimumNArgs(2),
	RunE: runLocalImport,
}

var localImportAllCmd = &cobra.Command{
	Use:   "import-all [patterns...]",
	Short: "Import items from all categories to local storage",
	Long: `Import items from all active directories to local storage.

Scans all category directories (agents, commands, skills, hooks, rules, output-styles)
for items that are not already symlinks, moves them to local storage, and enables them.

If patterns are provided, only items matching the patterns are imported.
Without patterns, all non-symlink items are imported.`,
	Example: `  claudeup local import-all           # Import everything
  claudeup local import-all gsd-* gsd # Import only GSD items`,
	RunE: runLocalImportAll,
}

var localInstallCmd = &cobra.Command{
	Use:   "install <category> <path>",
	Short: "Install items from an external path to local storage",
	Long: `Install items from an external path (file or directory) to local storage.

This copies files to local storage and automatically enables them.
Use this to install items from a git repo, downloads folder, or other location.

For single files/directories: installed as-is.
For directories containing multiple items: each item is installed individually.

Existing items with the same name are skipped (not overwritten).`,
	Example: `  claudeup local install agents ~/code/my-agents/
  claudeup local install hooks ~/Downloads/format-on-save.sh
  claudeup local install skills ~/code/my-skills/awesome-skill`,
	Args: cobra.ExactArgs(2),
	RunE: runLocalInstall,
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.AddCommand(localListCmd)
	localCmd.AddCommand(localEnableCmd)
	localCmd.AddCommand(localDisableCmd)
	localCmd.AddCommand(localViewCmd)
	localCmd.AddCommand(localSyncCmd)
	localCmd.AddCommand(localImportCmd)
	localCmd.AddCommand(localImportAllCmd)
	localCmd.AddCommand(localInstallCmd)

	localListCmd.Flags().BoolVarP(&localFilterEnabled, "enabled", "e", false, "Show only enabled items")
	localListCmd.Flags().BoolVarP(&localFilterDisabled, "disabled", "d", false, "Show only disabled items")
	localViewCmd.Flags().BoolVar(&localViewRaw, "raw", false, "Output raw content without rendering")
}

type itemStatus struct {
	name    string
	enabled bool
}

func runLocalList(cmd *cobra.Command, args []string) error {
	if localFilterEnabled && localFilterDisabled {
		return fmt.Errorf("--enabled and --disabled are mutually exclusive")
	}

	manager := local.NewManager(claudeDir, claudeupHome)
	config, err := manager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var categories []string
	if len(args) > 0 {
		if err := local.ValidateCategory(args[0]); err != nil {
			return err
		}
		categories = []string{args[0]}
	} else {
		categories = local.AllCategories()
	}

	totalItems := 0
	for _, category := range categories {
		items, err := manager.ListItems(category)
		if err != nil {
			continue
		}

		catConfig := config[category]
		if catConfig == nil {
			catConfig = make(map[string]bool)
		}

		// Filter items by status
		var filtered []itemStatus

		for _, item := range items {
			enabled := catConfig[item]
			if localFilterEnabled && !enabled {
				continue
			}
			if localFilterDisabled && enabled {
				continue
			}
			filtered = append(filtered, itemStatus{item, enabled})
		}

		totalItems += len(filtered)

		if len(filtered) == 0 {
			if len(args) > 0 {
				// User requested specific category
				fmt.Printf("\n%s/: (empty)\n", category)
			}
			continue
		}

		fmt.Printf("\n%s/:\n", category)

		if category == local.CategoryAgents {
			// Group agents by their group directory
			printGroupedAgents(filtered)
		} else {
			for _, item := range filtered {
				status := ui.Error("x")
				if item.enabled {
					status = ui.Success("*")
				}
				fmt.Printf("  %s %s\n", status, item.name)
			}
		}
	}

	if totalItems == 0 {
		fmt.Println("No items in library. Use 'claudeup local install' or 'claudeup local import' to add items.")
	}

	return nil
}

func printGroupedAgents(items []itemStatus) {
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
		status := ui.Error("x")
		if item.enabled {
			status = ui.Success("*")
		}
		fmt.Printf("  %s %s\n", status, item.name)
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
			status := ui.Error("x")
			if item.enabled {
				status = ui.Success("*")
			}
			fmt.Printf("    %s %s\n", status, strings.TrimSuffix(item.name, ".md"))
		}
	}
}

func runLocalEnable(cmd *cobra.Command, args []string) error {
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

func runLocalDisable(cmd *cobra.Command, args []string) error {
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

func runLocalView(cmd *cobra.Command, args []string) error {
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
		fmt.Print(ui.RenderMarkdown(content, localViewRaw))
	} else {
		fmt.Println(content)
	}
	return nil
}

func runLocalSync(cmd *cobra.Command, args []string) error {
	manager := local.NewManager(claudeDir, claudeupHome)

	fmt.Println("Syncing local items from enabled.json...")
	if err := manager.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	ui.PrintSuccess("Sync complete")
	return nil
}

func runLocalImport(cmd *cobra.Command, args []string) error {
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
		ui.PrintSuccess(fmt.Sprintf("Linked (already in local storage): %s/%s", category, item))
	}

	for _, pattern := range notFound {
		ui.PrintWarning(fmt.Sprintf("Not found: %s/%s", category, pattern))
	}

	if len(notFound) > 0 && len(imported) == 0 && len(skipped) == 0 {
		return fmt.Errorf("no items found matching patterns")
	}

	return nil
}

func runLocalImportAll(cmd *cobra.Command, args []string) error {
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
			ui.PrintSuccess(fmt.Sprintf("Linked (already in local storage): %s/%s", category, item))
			totalProcessed++
		}
	}

	if totalProcessed == 0 {
		fmt.Println("No items to import (all items are already symlinks or no matching items found)")
	}

	return nil
}

func runLocalInstall(cmd *cobra.Command, args []string) error {
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
		fmt.Println("All items already exist in local storage")
	}

	return nil
}
