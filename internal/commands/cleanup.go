// ABOUTME: Cleanup command implementation for fixing and removing plugin entries
// ABOUTME: Fixes correctable path issues and removes truly broken plugins
package commands

import (
	"fmt"

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
	Short: "Fix and remove plugin issues",
	Long: `Fix plugin path issues and remove entries that can't be fixed.

By default, this command:
  1. Fixes plugins with correctable path issues (missing subdirectories)
  2. Removes plugin entries that are truly broken (no valid path found)

Use --fix-only or --remove-only for granular control.`,
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

	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

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
	if len(fixableIssues) == 0 && len(unfixableIssues) == 0 {
		fmt.Println("✓ No issues found")
		return nil
	}

	// Show what will be done
	if len(fixableIssues) > 0 {
		if cleanupDryRun {
			fmt.Printf("Would fix %d path issues:\n\n", len(fixableIssues))
		} else {
			fmt.Printf("Found %d fixable path issues:\n\n", len(fixableIssues))
		}
		for _, issue := range fixableIssues {
			fmt.Printf("  %s\n", issue.PluginName)
			fmt.Printf("    %s → %s\n", issue.InstallPath, issue.ExpectedPath)
		}
		fmt.Println()
	}

	if len(unfixableIssues) > 0 {
		if cleanupDryRun {
			fmt.Printf("Would remove %d broken plugin entries:\n\n", len(unfixableIssues))
		} else {
			fmt.Printf("Found %d plugins to remove:\n\n", len(unfixableIssues))
		}
		for _, issue := range unfixableIssues {
			fmt.Printf("  • %s\n", issue.PluginName)
			fmt.Printf("    Path: %s\n", issue.InstallPath)
		}
		fmt.Println()
	}

	if cleanupDryRun {
		fmt.Println("Run without --dry-run to apply these changes")
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
				if plugin, exists := plugins.Plugins[issue.PluginName]; exists {
					plugin.InstallPath = issue.ExpectedPath
					plugins.Plugins[issue.PluginName] = plugin
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

	// Report results
	fmt.Println()
	if fixed > 0 {
		fmt.Printf("✓ Fixed %d plugin paths\n", fixed)
	}
	if removed > 0 {
		fmt.Printf("✓ Removed %d plugin entries\n", removed)
	}

	if cleanupReinstall && removed > 0 {
		fmt.Println("\nTo reinstall these plugins, use:")
		for _, issue := range removedIssues {
			fmt.Printf("  claude plugin install %s\n", issue.PluginName)
		}
	}

	if fixed > 0 || removed > 0 {
		fmt.Println("\nRun 'claudeup status' to verify the changes")
	}

	return nil
}

func filterByFlag(issues []PathIssue, include bool) []PathIssue {
	if include {
		return issues
	}
	return []PathIssue{}
}
