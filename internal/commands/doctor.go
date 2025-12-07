// ABOUTME: Doctor command implementation for diagnosing Claude installation issues
// ABOUTME: Detects stale paths, missing directories, and provides fix recommendations
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose common issues with Claude Code installation",
	Long:  `Run diagnostics to identify and explain issues with plugins, marketplaces, and paths.`,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type PathIssue struct {
	PluginName    string
	InstallPath   string
	ExpectedPath  string
	IssueType     string
	CanAutoFix    bool
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("Running diagnostics...")

	// Load plugins (gracefully handle fresh installs with no plugins)
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		if os.IsNotExist(err) {
			plugins = &claude.PluginRegistry{Plugins: make(map[string]claude.PluginMetadata)}
		} else {
			return fmt.Errorf("failed to load plugins: %w", err)
		}
	}

	// Load marketplaces (gracefully handle fresh installs)
	marketplaces, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		if os.IsNotExist(err) {
			marketplaces = make(claude.MarketplaceRegistry)
		} else {
			return fmt.Errorf("failed to load marketplaces: %w", err)
		}
	}

	// Check marketplaces
	fmt.Println("━━━ Checking Marketplaces ━━━")
	marketplaceIssues := 0
	for name, marketplace := range marketplaces {
		if _, err := os.Stat(marketplace.InstallLocation); os.IsNotExist(err) {
			fmt.Printf("  ✗ %s: Directory not found at %s\n", name, marketplace.InstallLocation)
			marketplaceIssues++
		} else {
			fmt.Printf("  ✓ %s\n", name)
		}
	}
	if marketplaceIssues == 0 {
		fmt.Println("  All marketplaces OK")
	}
	fmt.Println()

	// Analyze path issues
	fmt.Println("━━━ Analyzing Plugin Paths ━━━")
	pathIssues := analyzePathIssues(plugins)

	if len(pathIssues) == 0 {
		fmt.Println("  ✓ All plugin paths are valid")
	} else {
		// Group by issue type
		byType := make(map[string][]PathIssue)
		for _, issue := range pathIssues {
			byType[issue.IssueType] = append(byType[issue.IssueType], issue)
		}

		// Report fixable issues
		if fixable, ok := byType["missing_subdirectory"]; ok {
			fmt.Printf("  ⚠ %d plugins with fixable path issues:\n", len(fixable))
			for _, issue := range fixable {
				fmt.Printf("    - %s\n", issue.PluginName)
				fmt.Printf("      Current:  %s\n", issue.InstallPath)
				fmt.Printf("      Expected: %s\n", issue.ExpectedPath)
			}
		}

		// Report truly missing plugins
		if missing, ok := byType["not_found"]; ok {
			if len(byType["missing_subdirectory"]) > 0 {
				fmt.Println()
			}
			fmt.Printf("  ✗ %d plugins with missing directories:\n", len(missing))
			for _, issue := range missing {
				fmt.Printf("    - %s\n", issue.PluginName)
				fmt.Printf("      Path: %s\n", issue.InstallPath)
			}
		}

		// Unified recommendation
		fmt.Println("\n  → Run 'claudeup cleanup' to fix and remove these issues")
		fmt.Println("     (use --fix-only or --remove-only for granular control)")
	}
	fmt.Println()

	// Summary
	fmt.Println("━━━ Summary ━━━")
	fmt.Printf("  Marketplaces: %d installed", len(marketplaces))
	if marketplaceIssues > 0 {
		fmt.Printf(", %d issues", marketplaceIssues)
	}
	fmt.Println()

	fmt.Printf("  Plugins:      %d installed", len(plugins.Plugins))
	if len(pathIssues) > 0 {
		fmt.Printf(", %d issues", len(pathIssues))
	}
	fmt.Println()

	if len(pathIssues) > 0 || marketplaceIssues > 0 {
		fmt.Println("\nRun the suggested commands to fix these issues.")
	} else {
		fmt.Println("\n✓ No issues detected!")
	}

	return nil
}

func analyzePathIssues(plugins *claude.PluginRegistry) []PathIssue {
	var issues []PathIssue

	for name, plugin := range plugins.Plugins {
		if !plugin.PathExists() {
			// Check if this is a fixable path issue
			expectedPath := getExpectedPath(name, plugin.InstallPath)
			if expectedPath != "" && pathExists(expectedPath) {
				issues = append(issues, PathIssue{
					PluginName:   name,
					InstallPath:  plugin.InstallPath,
					ExpectedPath: expectedPath,
					IssueType:    "missing_subdirectory",
					CanAutoFix:   true,
				})
			} else {
				issues = append(issues, PathIssue{
					PluginName:  name,
					InstallPath: plugin.InstallPath,
					IssueType:   "not_found",
					CanAutoFix:  false,
				})
			}
		}
	}

	// Sort by plugin name
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].PluginName < issues[j].PluginName
	})

	return issues
}

func getExpectedPath(pluginName, currentPath string) string {
	// Based on fix-plugin-paths.sh logic
	if strings.Contains(currentPath, "claude-code-plugins") {
		// Add /plugins/ subdirectory
		base := filepath.Dir(currentPath)
		plugin := filepath.Base(currentPath)
		return filepath.Join(base, "plugins", plugin)
	}
	if strings.Contains(currentPath, "claude-code-templates") {
		base := filepath.Dir(currentPath)
		plugin := filepath.Base(currentPath)
		return filepath.Join(base, "plugins", plugin)
	}
	if strings.Contains(currentPath, "anthropic-agent-skills") {
		base := filepath.Dir(currentPath)
		plugin := filepath.Base(currentPath)
		return filepath.Join(base, "skills", plugin)
	}
	if strings.Contains(currentPath, "every-marketplace") {
		base := filepath.Dir(currentPath)
		plugin := filepath.Base(currentPath)
		return filepath.Join(base, "plugins", plugin)
	}
	if strings.Contains(currentPath, "awesome-claude-code-plugins") {
		base := filepath.Dir(currentPath)
		plugin := filepath.Base(currentPath)
		return filepath.Join(base, "plugins", plugin)
	}
	if strings.Contains(currentPath, "tanzu-cf-architect") {
		// Remove duplicate directory name
		parts := strings.Split(currentPath, string(filepath.Separator))
		lastPart := parts[len(parts)-1]
		if len(parts) >= 2 && parts[len(parts)-2] == lastPart {
			return filepath.Join(parts[:len(parts)-1]...)
		}
	}
	return ""
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
