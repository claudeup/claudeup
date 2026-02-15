// ABOUTME: Doctor command implementation for diagnosing Claude installation issues
// ABOUTME: Detects stale paths, missing directories, and provides fix recommendations
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/local"
	"github.com/claudeup/claudeup/v5/internal/ui"
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
	Scope        string
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

	// Detect plugins enabled in settings but not installed,
	// tracking which scope each one is enabled in
	missingPlugins := []string{}
	missingPluginScope := make(map[string]string)
	for name := range enabledInSettings {
		if !plugins.PluginExistsAtAnyScope(name) {
			missingPlugins = append(missingPlugins, name)
			for _, scope := range scopes {
				if scopeSettings[scope] != nil && scopeSettings[scope].IsPluginEnabled(name) {
					missingPluginScope[name] = scope
					break
				}
			}
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
			fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+fmt.Sprintf(" %d plugin%s enabled but not installed:", len(missingPlugins), pluralS(len(missingPlugins))), 1))
			for _, name := range missingPlugins {
				scope := missingPluginScope[name]
				fmt.Println(ui.Indent(ui.SymbolBullet+" "+name+" "+ui.Muted("("+scope+")"), 2))
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
				fmt.Println(ui.Indent(ui.SymbolBullet+" "+issue.PluginName+ui.Muted(" ("+issue.Scope+")"), 2))
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
				fmt.Println(ui.Indent(ui.SymbolBullet+" "+issue.PluginName+ui.Muted(" ("+issue.Scope+")"), 2))
				fmt.Println(ui.Indent(ui.RenderDetail("Path", issue.InstallPath), 3))
			}
		}

		// Recommendations
		fmt.Println()
		fmt.Println(ui.Indent(ui.Bold("Recommendations:"), 1))

		if len(missingPlugins) > 0 {
			fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Install the plugin:"), 1))
			fmt.Println(ui.Indent(ui.Bold("claude plugin install --scope <scope> <plugin-name>"), 2))
			fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Remove the stale settings entry:"), 1))
			fmt.Println(ui.Indent(ui.Bold("claudeup profile clean --<scope> <plugin-name>"), 2))
		}

		if len(pathIssues) > 0 {
			fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Fix path issues: "+ui.Bold("claudeup cleanup")), 1))
			fmt.Println(ui.Indent(ui.Muted("(use --fix-only or --remove-only for granular control)"), 2))
		}
	}
	// Check for broken symlinks in extensions
	fmt.Println(ui.RenderSection("Checking Local Symlinks", -1))
	brokenSymlinks := checkBrokenSymlinks()
	if len(brokenSymlinks) == 0 {
		fmt.Println(ui.Indent(ui.Success(ui.SymbolSuccess)+" All local symlinks are valid", 1))
	} else {
		fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+fmt.Sprintf(" %d broken symlink%s:", len(brokenSymlinks), pluralS(len(brokenSymlinks))), 1))
		for _, bs := range brokenSymlinks {
			fmt.Println(ui.Indent(ui.SymbolBullet+" "+bs.Path, 2))
			fmt.Println(ui.Indent(ui.Muted("-> "+bs.Target), 3))
		}
		fmt.Println()
		fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Fix with: "+ui.Bold("claudeup extensions sync")), 1))
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

	for _, sp := range plugins.GetPluginsAtScopes(claude.ValidScopes) {
		name := sp.Name
		plugin := sp.PluginMetadata
		if !plugin.PathExists() {
			// Check if this is a fixable path issue
			expectedPath := getExpectedPath(plugin.InstallPath)
			if expectedPath != "" && pathExists(expectedPath) {
				issues = append(issues, PathIssue{
					PluginName:   name,
					Scope:        plugin.Scope,
					InstallPath:  plugin.InstallPath,
					ExpectedPath: expectedPath,
					IssueType:    "missing_subdirectory",
					CanAutoFix:   true,
				})
			} else {
				issues = append(issues, PathIssue{
					PluginName:  name,
					Scope:       plugin.Scope,
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

// BrokenSymlink represents a symlink whose target no longer exists
type BrokenSymlink struct {
	Path   string
	Target string
}

// checkBrokenSymlinks scans category directories recursively for symlinks with missing targets
func checkBrokenSymlinks() []BrokenSymlink {
	var broken []BrokenSymlink
	for _, category := range local.AllCategories() {
		catDir := filepath.Join(claudeDir, category)
		if _, err := os.Stat(catDir); err != nil {
			continue
		}

		filepath.WalkDir(catDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip entries we can't read
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					broken = append(broken, BrokenSymlink{Path: path, Target: "(unreadable)"})
					return nil
				}
				if _, err := os.Stat(path); os.IsNotExist(err) {
					broken = append(broken, BrokenSymlink{Path: path, Target: target})
				}
			}
			return nil
		})
	}
	return broken
}
