// ABOUTME: Cleanup command implementation for fixing and removing plugin entries
// ABOUTME: Fixes correctable path issues and removes truly broken plugins
package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cleanupReinstall bool
	cleanupDryRun    bool
	cleanupFixOnly   bool
	cleanupRemoveOnly bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Fix plugin path issues, remove stale entries, and reinstall missing plugins",
	Long: `Fix plugin configuration drift and installation issues.

By default, this command:
  1. Fixes plugins with correctable path issues (missing subdirectories)
  2. Removes plugin entries that are truly broken (no valid path found)
  3. Offers to reinstall plugins enabled in settings but not installed

Use --fix-only or --remove-only for granular control.`,
	Example: `  # Preview changes without applying them
  claudeup cleanup --dry-run

  # Only fix path issues, don't remove entries
  claudeup cleanup --fix-only

  # Only remove stale entries, don't fix paths
  claudeup cleanup --remove-only

  # Show reinstall commands for removed plugins
  claudeup cleanup --reinstall`,
	Args: cobra.NoArgs,
	RunE: runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().BoolVar(&cleanupReinstall, "reinstall", false, "Show reinstall commands for removed plugins")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would happen without making changes")
	cleanupCmd.Flags().BoolVar(&cleanupFixOnly, "fix-only", false, "Only fix path issues, don't remove entries")
	cleanupCmd.Flags().BoolVar(&cleanupRemoveOnly, "remove-only", false, "Only remove broken entries, don't fix paths")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	// Validate flag combinations
	if cleanupFixOnly && cleanupRemoveOnly {
		return fmt.Errorf("cannot use --fix-only and --remove-only together")
	}

	// Get current directory for scope-aware settings
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Load settings from all scopes to find enabled plugins
	scopes := []string{"user", "project", "local"}
	enabledInSettings := make(map[string]bool)

	for _, scope := range scopes {
		settings, err := claude.LoadSettingsForScope(scope, claudeDir, projectDir)
		if err == nil {
			for name, enabled := range settings.EnabledPlugins {
				if enabled {
					enabledInSettings[name] = true
				}
			}
		}
	}

	// Detect plugins enabled in settings but not installed
	missingPlugins := []string{}
	for name := range enabledInSettings {
		if _, installed := plugins.GetAllPlugins()[name]; !installed {
			missingPlugins = append(missingPlugins, name)
		}
	}
	sort.Strings(missingPlugins)

	// Analyze issues
	pathIssues := analyzePathIssues(plugins)

	// Separate fixable and unfixable issues
	fixableIssues := []PathIssue{}
	unfixableIssues := []PathIssue{}
	for _, issue := range pathIssues {
		if issue.CanAutoFix {
			fixableIssues = append(fixableIssues, issue)
		} else {
			unfixableIssues = append(unfixableIssues, issue)
		}
	}

	// Apply flag filtering
	shouldFix := !cleanupRemoveOnly
	shouldRemove := !cleanupFixOnly

	if shouldFix {
		fixableIssues = filterByFlag(fixableIssues, shouldFix)
	} else {
		fixableIssues = []PathIssue{}
	}

	if shouldRemove {
		unfixableIssues = filterByFlag(unfixableIssues, shouldRemove)
	} else {
		unfixableIssues = []PathIssue{}
	}

	// Check if there's anything to do
	if len(fixableIssues) == 0 && len(unfixableIssues) == 0 && len(missingPlugins) == 0 {
		ui.PrintSuccess("No issues found")
		return nil
	}

	// Show missing plugins first (most common issue)
	if len(missingPlugins) > 0 {
		if cleanupDryRun {
			fmt.Println(ui.RenderSection("Would reinstall missing plugins", len(missingPlugins)))
		} else {
			fmt.Println(ui.RenderSection("Plugins enabled but not installed", len(missingPlugins)))
		}
		fmt.Println()
		for _, name := range missingPlugins {
			fmt.Printf("  %s %s\n", ui.Error(ui.SymbolError), ui.Bold(name))
		}
		fmt.Println()
	}

	// Show what will be done
	if len(fixableIssues) > 0 {
		if cleanupDryRun {
			fmt.Println(ui.RenderSection("Would fix path issues", len(fixableIssues)))
		} else {
			fmt.Println(ui.RenderSection("Fixable path issues", len(fixableIssues)))
		}
		fmt.Println()
		for _, issue := range fixableIssues {
			fmt.Printf("  %s %s\n", ui.Warning(ui.SymbolWarning), ui.Bold(issue.PluginName))
			fmt.Println(ui.Indent(fmt.Sprintf("%s %s %s", ui.Muted(issue.InstallPath), ui.SymbolArrow, issue.ExpectedPath), 2))
		}
		fmt.Println()
	}

	if len(unfixableIssues) > 0 {
		if cleanupDryRun {
			fmt.Println(ui.RenderSection("Would remove broken entries", len(unfixableIssues)))
		} else {
			fmt.Println(ui.RenderSection("Plugins to remove", len(unfixableIssues)))
		}
		fmt.Println()
		for _, issue := range unfixableIssues {
			fmt.Printf("  %s %s\n", ui.Error(ui.SymbolError), ui.Bold(issue.PluginName))
			fmt.Println(ui.Indent(ui.RenderDetail("Path", ui.Muted(issue.InstallPath)), 2))
		}
		fmt.Println()
	}

	if cleanupDryRun {
		ui.PrintInfo("Run without --dry-run to apply these changes")
		return nil
	}

	// Apply fixes with prompt
	fixed := 0
	if len(fixableIssues) > 0 {
		confirm, err := ui.ConfirmYesNo("Fix these paths?")
		if err != nil {
			return err
		}
		if confirm {
			for _, issue := range fixableIssues {
				if plugin, exists := plugins.GetPlugin(issue.PluginName); exists {
					plugin.InstallPath = issue.ExpectedPath
					plugins.SetPlugin(issue.PluginName, plugin)
					fixed++
				}
			}
		}
	}

	// Remove unfixable entries with prompt
	removed := 0
	removedIssues := []PathIssue{}
	if len(unfixableIssues) > 0 {
		confirm, err := ui.ConfirmYesNo("Remove broken entries?")
		if err != nil {
			return err
		}
		if confirm {
			for _, issue := range unfixableIssues {
				if plugins.DisablePlugin(issue.PluginName) {
					removed++
					removedIssues = append(removedIssues, issue)
				}
			}
		}
	}

	// Save updated plugins
	if err := claude.SavePlugins(claudeDir, plugins); err != nil {
		return fmt.Errorf("failed to save plugins: %w", err)
	}

	// Handle missing plugins (enabled but not installed)
	if len(missingPlugins) > 0 && !cleanupDryRun {
		// Get active profile for smarter recommendations
		activeProfile, _ := getActiveProfile(projectDir)

		fmt.Println()
		if activeProfile != "" && activeProfile != "none" {
			// Use profile to reinstall
			ui.PrintInfo(fmt.Sprintf("Reinstall %d missing plugin%s from profile '%s'?", len(missingPlugins), pluralS(len(missingPlugins)), activeProfile))
			confirm, err := ui.ConfirmYesNo(fmt.Sprintf("Run 'claudeup profile apply %s --reinstall'?", activeProfile))
			if err != nil {
				return err
			}
			if confirm {
				ui.PrintInfo("To reinstall missing plugins, run:")
				fmt.Printf("  %s\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s --reinstall", activeProfile)))
				fmt.Println()
				ui.PrintMuted("(This will be automated in a future version)")
			}
		} else {
			// No active profile - show manual install commands
			ui.PrintInfo("To reinstall missing plugins, run:")
			for _, name := range missingPlugins {
				fmt.Printf("  %s\n", ui.Muted("claude plugin install "+name))
			}
		}
	}

	// Report results
	fmt.Println()
	if fixed > 0 {
		ui.PrintSuccess(fmt.Sprintf("Fixed %d plugin paths", fixed))
	}
	if removed > 0 {
		ui.PrintSuccess(fmt.Sprintf("Removed %d plugin entries", removed))
	}

	if cleanupReinstall && removed > 0 {
		fmt.Println()
		ui.PrintInfo("To reinstall these plugins, use:")
		for _, issue := range removedIssues {
			fmt.Printf("  %s\n", ui.Muted("claude plugin install "+issue.PluginName))
		}
	}

	if fixed > 0 || removed > 0 || len(missingPlugins) > 0 {
		fmt.Println()
		fmt.Printf("%s Run 'claudeup status' to verify the changes\n", ui.Muted(ui.SymbolArrow))
	}

	return nil
}

func filterByFlag(issues []PathIssue, include bool) []PathIssue {
	if include {
		return issues
	}
	return []PathIssue{}
}
