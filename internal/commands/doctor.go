// ABOUTME: Doctor command implementation for diagnosing Claude installation issues
// ABOUTME: Detects stale paths, missing directories, and provides fix recommendations
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v4/internal/claude"
	"github.com/claudeup/claudeup/v4/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose common issues with Claude Code installation",
	Long: `Run diagnostics to identify and explain issues with plugins, marketplaces, and paths.

Checks:
  - Marketplace directories exist
  - Plugin paths are valid
  - Fixable path issues vs truly broken entries

Use 'claudeup cleanup' to fix any detected issues.`,
	Args: cobra.NoArgs,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type PathIssue struct {
	PluginName   string
	InstallPath  string
	ExpectedPath string
	IssueType    string
	CanAutoFix   bool
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ui.PrintInfo("Running diagnostics...")

	// Get current directory for scope-aware settings
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get active profile for recommendations
	activeProfile, _ := getActiveProfile(projectDir)

	// Load plugins (gracefully handle fresh installs with no plugins)
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		if os.IsNotExist(err) {
			plugins = &claude.PluginRegistry{Plugins: make(map[string][]claude.PluginMetadata)}
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

	// Load settings from all scopes to find enabled plugins
	scopes := []string{"user", "project", "local"}
	scopeSettings := make(map[string]*claude.Settings)
	enabledInSettings := make(map[string]bool)

	for _, scope := range scopes {
		settings, err := claude.LoadSettingsForScope(scope, claudeDir, projectDir)
		if err == nil {
			scopeSettings[scope] = settings
			for name, enabled := range settings.EnabledPlugins {
				if enabled {
					enabledInSettings[name] = true
				}
			}
		}
	}

	// Check marketplaces
	fmt.Println()
	fmt.Println(ui.RenderSection("Checking Marketplaces", len(marketplaces)))
	marketplaceIssues := 0
	for name, marketplace := range marketplaces {
		if _, err := os.Stat(marketplace.InstallLocation); os.IsNotExist(err) {
			fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+" "+name+": Directory not found at "+marketplace.InstallLocation, 1))
			marketplaceIssues++
		} else {
			fmt.Println(ui.Indent(ui.Success(ui.SymbolSuccess)+" "+name, 1))
		}
	}
	if marketplaceIssues == 0 {
		fmt.Println(ui.Indent(ui.Success("All marketplaces OK"), 1))
	}
	fmt.Println()

	// Detect plugins enabled in settings but not installed
	missingPlugins := []string{}
	for name := range enabledInSettings {
		if _, installed := plugins.GetAllPlugins()[name]; !installed {
			missingPlugins = append(missingPlugins, name)
		}
	}
	sort.Strings(missingPlugins)

	// Analyze path issues
	fmt.Println(ui.RenderSection("Analyzing Plugin Paths", -1))
	pathIssues := analyzePathIssues(plugins)

	if len(pathIssues) == 0 && len(missingPlugins) == 0 {
		fmt.Println(ui.Indent(ui.Success(ui.SymbolSuccess)+" All plugin paths are valid", 1))
	} else {
		// Show plugins enabled but not installed
		if len(missingPlugins) > 0 {
			// Check which missing plugins are in the saved profile
			pluginsInProfile := make(map[string]bool)
			if activeProfile != "" && activeProfile != "none" {
				profilesDir := getProfilesDir()
				savedProfile, err := loadProfileWithFallback(profilesDir, activeProfile)
				if err == nil {
					for _, p := range savedProfile.Plugins {
						pluginsInProfile[p] = true
					}
				}
			}

			fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+fmt.Sprintf(" %d plugin%s enabled but not installed:", len(missingPlugins), pluralS(len(missingPlugins))), 1))
			for _, name := range missingPlugins {
				suffix := ""
				if pluginsInProfile[name] {
					suffix = ui.Muted(" (in profile)")
				}
				fmt.Println(ui.Indent(ui.SymbolBullet+" "+name+suffix, 2))
			}
		}

		// Group by issue type
		byType := make(map[string][]PathIssue)
		for _, issue := range pathIssues {
			byType[issue.IssueType] = append(byType[issue.IssueType], issue)
		}

		// Report fixable issues
		if fixable, ok := byType["missing_subdirectory"]; ok {
			fmt.Println(ui.Indent(ui.Warning(ui.SymbolWarning)+fmt.Sprintf(" %d plugins with fixable path issues:", len(fixable)), 1))
			for _, issue := range fixable {
				fmt.Println(ui.Indent(ui.SymbolBullet+" "+issue.PluginName, 2))
				fmt.Println(ui.Indent(ui.RenderDetail("Current", issue.InstallPath), 3))
				fmt.Println(ui.Indent(ui.RenderDetail("Expected", issue.ExpectedPath), 3))
			}
		}

		// Report truly missing plugins
		if missing, ok := byType["not_found"]; ok {
			if len(byType["missing_subdirectory"]) > 0 {
				fmt.Println()
			}
			fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+fmt.Sprintf(" %d plugins with missing directories:", len(missing)), 1))
			for _, issue := range missing {
				fmt.Println(ui.Indent(ui.SymbolBullet+" "+issue.PluginName, 2))
				fmt.Println(ui.Indent(ui.RenderDetail("Path", issue.InstallPath), 3))
			}
		}

		// Recommendations
		fmt.Println()
		fmt.Println(ui.Indent(ui.Bold("Recommendations:"), 1))

		// Get active profile for better recommendations
		activeProfile, _ := getActiveProfile(projectDir)

		if len(missingPlugins) > 0 {
			if activeProfile != "" && activeProfile != "none" {
				fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+fmt.Sprintf(" Reinstall missing plugins from profile: %s", ui.Bold("claudeup profile apply"))), 1))
				fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Or remove from settings: "+ui.Bold("claudeup profile clean <plugin-name>")), 1))
			} else {
				fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Install missing plugins: "+ui.Bold("claude plugin install <name>")), 1))
				fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Or remove from settings: "+ui.Bold("claudeup profile clean <plugin-name>")), 1))
			}
		}

		if len(pathIssues) > 0 {
			fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Fix path issues: "+ui.Bold("claudeup cleanup")), 1))
			fmt.Println(ui.Indent(ui.Muted("(use --fix-only or --remove-only for granular control)"), 2))
		}
	}
	fmt.Println()

	// Summary
	fmt.Println(ui.RenderSection("Summary", -1))
	marketplaceSummary := fmt.Sprintf("%d installed", len(marketplaces))
	if marketplaceIssues > 0 {
		marketplaceSummary += fmt.Sprintf(", %d issues", marketplaceIssues)
	}
	fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", marketplaceSummary), 1))

	pluginSummary := fmt.Sprintf("%d installed", len(plugins.Plugins))
	totalIssues := len(pathIssues) + len(missingPlugins)
	if totalIssues > 0 {
		pluginSummary += fmt.Sprintf(", %d issues", totalIssues)
	}
	fmt.Println(ui.Indent(ui.RenderDetail("Plugins", pluginSummary), 1))

	fmt.Println()
	if totalIssues > 0 || marketplaceIssues > 0 {
		ui.PrintInfo("Run the suggested commands to fix these issues.")
	} else {
		ui.PrintSuccess("No issues detected!")
	}

	return nil
}

func analyzePathIssues(plugins *claude.PluginRegistry) []PathIssue {
	var issues []PathIssue

	for name, plugin := range plugins.GetAllPlugins() {
		if !plugin.PathExists() {
			// Check if this is a fixable path issue
			expectedPath := getExpectedPath(plugin.InstallPath)
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

func getExpectedPath(currentPath string) string {
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
