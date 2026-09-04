// ABOUTME: Doctor command implementation for diagnosing Claude installation issues
// ABOUTME: Detects stale paths, missing directories, and provides fix recommendations
package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/ext"
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

	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Load marketplaces (LoadMarketplaces treats missing files as fresh install)
	marketplaces, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load marketplaces: %w", err)
	}

	// Load settings from all scopes to find enabled plugins
	scopeSettings := make(map[string]*claude.Settings)
	enabledInSettings := make(map[string]bool)

	type scopeIssue struct {
		scope string
		path  string
		err   error
	}
	var scopeIssues []scopeIssue

	for _, scope := range claude.ValidScopes {
		settings, err := claude.LoadSettingsForScope(scope, claudeDir, projectDir)
		if err == nil {
			scopeSettings[scope] = settings
			for name, enabled := range settings.EnabledPlugins {
				if enabled {
					enabledInSettings[name] = true
				}
			}
		} else {
			// LoadSettingsForScope handles missing files internally (returns empty settings, nil error).
			// Any error reaching here is a real I/O or parse failure.
			path, pathErr := claude.SettingsPathForScope(scope, claudeDir, projectDir)
			if pathErr != nil {
				path = ""
			}
			scopeIssues = append(scopeIssues, scopeIssue{scope: scope, path: path, err: err})
		}
	}

	// Report any scope settings loading errors
	if len(scopeIssues) > 0 {
		fmt.Println()
		fmt.Println(ui.RenderSection("Checking Settings Scopes", -1))
		for _, se := range scopeIssues {
			fmt.Println(ui.Indent(fmt.Sprintf("%s %s scope: failed to load settings: %v", ui.Warning(ui.SymbolWarning), se.scope, se.err), 1))
			if se.path != "" {
				fmt.Println(ui.Indent(ui.Muted("Restore or delete the corrupted file: "+se.path), 2))
			} else {
				fmt.Println(ui.Indent(ui.Muted("Could not determine settings file path for this scope."), 2))
			}
		}
	}

	// Check marketplaces
	fmt.Println()
	fmt.Println(ui.RenderSection("Checking Marketplaces", len(marketplaces)))
	marketplaceIssues := 0
	for name, marketplace := range marketplaces {
		_, err := os.Stat(marketplace.InstallLocation)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+" "+name+": Directory not found at "+marketplace.InstallLocation, 1))
			marketplaceIssues++
		case err != nil:
			fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+" "+name+": Cannot access directory at "+marketplace.InstallLocation+": "+underlyingError(err), 1))
			marketplaceIssues++
		default:
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
			for _, scope := range claude.ValidScopes {
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

	if len(scopeIssues) > 0 {
		fmt.Println(ui.Indent(fmt.Sprintf("%s Plugin analysis may be incomplete: %d scope%s could not be loaded",
			ui.Warning(ui.SymbolWarning), len(scopeIssues), pluralS(len(scopeIssues))), 1))
	}

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
	brokenSymlinks, uncheckedFromWalk := checkBrokenSymlinks(claudeDir)

	// Check for directory symlinks that bypass enable/disable controls
	dirSymlinks, uncheckedFromDirs := checkDirectorySymlinks(claudeDir)

	// The same entry can fail in both scans; report each path once
	uncheckedPaths := mergeUncheckedPaths(uncheckedFromWalk, uncheckedFromDirs)

	if len(brokenSymlinks) == 0 && len(uncheckedPaths) == 0 {
		fmt.Println(ui.Indent(ui.Success(ui.SymbolSuccess)+" All local symlinks are valid", 1))
	}
	if len(brokenSymlinks) > 0 {
		fmt.Println(ui.Indent(ui.Error(ui.SymbolError)+fmt.Sprintf(" %d broken symlink%s:", len(brokenSymlinks), pluralS(len(brokenSymlinks))), 1))
		for _, bs := range brokenSymlinks {
			fmt.Println(ui.Indent(ui.SymbolBullet+" "+bs.Path, 2))
			fmt.Println(ui.Indent(ui.Muted("-> "+bs.Target), 3))
		}
		fmt.Println()
		fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Fix with: "+ui.Bold("claudeup extensions sync")), 1))
	}

	if len(uncheckedPaths) > 0 {
		if len(brokenSymlinks) > 0 {
			fmt.Println()
		}
		fmt.Println(ui.Indent(ui.Warning(ui.SymbolWarning)+fmt.Sprintf(" %d path%s could not be checked:", len(uncheckedPaths), pluralS(len(uncheckedPaths))), 1))
		for _, up := range uncheckedPaths {
			fmt.Println(ui.Indent(ui.SymbolBullet+" "+up.Path, 2))
			fmt.Println(ui.Indent(ui.Muted(underlyingError(up.Err)), 3))
		}
		fmt.Println()
		fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Symlink results above may be incomplete; check permissions on the listed paths"), 1))
	}

	if len(dirSymlinks) > 0 {
		fmt.Println()
		fmt.Println(ui.Indent(ui.Warning(ui.SymbolWarning)+fmt.Sprintf(" %d directory symlink%s bypassing enable/disable controls:", len(dirSymlinks), pluralS(len(dirSymlinks))), 1))
		for _, ds := range dirSymlinks {
			fmt.Println(ui.Indent(ui.SymbolBullet+" "+filepath.Base(ds.Path)+ui.Muted(" ("+ds.Category+", "+fmt.Sprintf("%d", ds.ItemCount)+" items exposed)"), 2))
			fmt.Println(ui.Indent(ui.Muted("-> "+ds.Target), 3))
		}
		fmt.Println()
		fmt.Println(ui.Indent(ui.Info(ui.SymbolArrow+" Fix with: "+ui.Bold("claudeup extensions disable <category> <directory-name>")), 1))
		fmt.Println(ui.Indent(ui.Muted("This removes the directory symlink; re-enable individual items with claudeup extensions enable"), 2))
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

	symSummary := fmt.Sprintf("%d broken", len(brokenSymlinks))
	if len(uncheckedPaths) > 0 {
		symSummary += fmt.Sprintf(", %d unchecked", len(uncheckedPaths))
	}
	if len(dirSymlinks) > 0 {
		symSummary += fmt.Sprintf(", %d directory symlinks", len(dirSymlinks))
	}
	if len(brokenSymlinks) == 0 && len(uncheckedPaths) == 0 && len(dirSymlinks) == 0 {
		symSummary = "all valid"
	}
	fmt.Println(ui.Indent(ui.RenderDetail("Symlinks", symSummary), 1))

	if len(scopeIssues) > 0 {
		fmt.Println(ui.Indent(ui.RenderDetail("Settings", fmt.Sprintf("%d scope load error%s", len(scopeIssues), pluralS(len(scopeIssues)))), 1))
	}

	fmt.Println()
	allIssues := totalIssues + marketplaceIssues + len(brokenSymlinks) + len(uncheckedPaths) + len(dirSymlinks) + len(scopeIssues)
	if allIssues > 0 {
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

// pluginsSubdirMarketplaces are marketplaces whose plugins live under a
// plugins/ subdirectory that older registry entries omitted.
var pluginsSubdirMarketplaces = []string{
	"claude-code-plugins",
	"claude-plugins-official",
	"claude-code-templates",
	"every-marketplace",
	"awesome-claude-code-plugins",
}

func getExpectedPath(currentPath string) string {
	// Based on fix-plugin-paths.sh logic
	for _, marker := range pluginsSubdirMarketplaces {
		if strings.Contains(currentPath, marker) {
			// Add /plugins/ subdirectory
			return filepath.Join(filepath.Dir(currentPath), "plugins", filepath.Base(currentPath))
		}
	}
	if strings.Contains(currentPath, "anthropic-agent-skills") {
		// Add /skills/ subdirectory
		return filepath.Join(filepath.Dir(currentPath), "skills", filepath.Base(currentPath))
	}
	if strings.Contains(currentPath, "platform-k8s-architect") {
		// Remove duplicate directory name
		dir := filepath.Dir(currentPath)
		base := filepath.Base(currentPath)
		if filepath.Base(dir) == base {
			return dir
		}
	}
	return ""
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirectorySymlink represents a symlink in a category directory that points to a
// directory instead of a file. Directory symlinks bypass individual enable/disable
// controls in enabled.json, exposing all items in the target directory.
type DirectorySymlink struct {
	Path      string
	Target    string
	Category  string
	ItemCount int
}

// UncheckedPath represents a path doctor could not inspect because of an OS
// error other than "not found" (for example permission denied or a symlink
// loop). Doctor cannot tell whether such an entry is valid, so it is reported
// separately from broken symlinks instead of being silently skipped.
type UncheckedPath struct {
	Path string
	Err  error
}

// mergeUncheckedPaths combines unchecked entries from several scans, keeping
// the first entry seen for each path, sorted by path.
func mergeUncheckedPaths(lists ...[]UncheckedPath) []UncheckedPath {
	var merged []UncheckedPath
	seen := make(map[string]bool)
	for _, list := range lists {
		for _, up := range list {
			if seen[up.Path] {
				continue
			}
			seen[up.Path] = true
			merged = append(merged, up)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Path < merged[j].Path
	})
	return merged
}

// underlyingError returns the OS-level cause of a path error without the
// repeated operation and path prefix, so it can be printed next to the path.
func underlyingError(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}
	return err.Error()
}

// checkDirectorySymlinks scans category directories for symlinks that point to
// directories. These bypass the per-item enable/disable controls in enabled.json.
// Symlinks that cannot be resolved for a reason other than a missing target are
// returned as unchecked; missing targets are left to checkBrokenSymlinks.
func checkDirectorySymlinks(baseDir string) ([]DirectorySymlink, []UncheckedPath) {
	var results []DirectorySymlink
	var unchecked []UncheckedPath
	for _, category := range ext.AllCategories() {
		catDir := filepath.Join(baseDir, category)
		entries, err := os.ReadDir(catDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			unchecked = append(unchecked, UncheckedPath{Path: catDir, Err: err})
			continue
		}

		for _, entry := range entries {
			path := filepath.Join(catDir, entry.Name())

			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}

			// Resolve the symlink and check if the target is a directory
			resolved, err := filepath.EvalSymlinks(path)
			if err == nil {
				var resolvedInfo os.FileInfo
				resolvedInfo, err = os.Stat(resolved)
				if err == nil && !resolvedInfo.IsDir() {
					continue
				}
			}
			if errors.Is(err, fs.ErrNotExist) {
				continue // dangling symlink: reported by checkBrokenSymlinks
			}
			if err != nil {
				unchecked = append(unchecked, UncheckedPath{Path: path, Err: err})
				continue
			}

			// Skill directories (containing SKILL.md) are legitimate directory symlinks
			if category == ext.CategorySkills {
				if _, err := os.Stat(filepath.Join(resolved, "SKILL.md")); err == nil {
					continue
				}
			}

			// Count exposed items
			items, err := os.ReadDir(path)
			itemCount := 0
			if err == nil {
				for _, item := range items {
					if !item.IsDir() {
						itemCount++
					}
				}
			}

			results = append(results, DirectorySymlink{
				Path:      path,
				Target:    resolved,
				Category:  category,
				ItemCount: itemCount,
			})
		}
	}
	return results, unchecked
}

// BrokenSymlink represents a symlink whose target no longer exists
type BrokenSymlink struct {
	Path   string
	Target string
}

// checkBrokenSymlinks scans category directories recursively for symlinks with
// missing targets. Entries that could not be read or resolved for any other
// reason are returned as unchecked rather than silently skipped, so doctor
// never reports "all valid" for a tree it could not fully inspect.
func checkBrokenSymlinks(baseDir string) ([]BrokenSymlink, []UncheckedPath) {
	var broken []BrokenSymlink
	var unchecked []UncheckedPath
	for _, category := range ext.AllCategories() {
		catDir := filepath.Join(baseDir, category)
		if _, err := os.Stat(catDir); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			unchecked = append(unchecked, UncheckedPath{Path: catDir, Err: err})
			continue
		}

		// WalkDir reports read errors through the callback; the callback never
		// returns an error, so the walk itself cannot fail.
		_ = filepath.WalkDir(catDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				unchecked = append(unchecked, UncheckedPath{Path: path, Err: err})
				return nil // keep walking the rest of the tree
			}

			info, err := d.Info()
			if err != nil {
				unchecked = append(unchecked, UncheckedPath{Path: path, Err: err})
				return nil
			}

			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					unchecked = append(unchecked, UncheckedPath{Path: path, Err: err})
					return nil
				}
				_, err = os.Stat(path)
				switch {
				case errors.Is(err, fs.ErrNotExist):
					broken = append(broken, BrokenSymlink{Path: path, Target: target})
				case err != nil:
					unchecked = append(unchecked, UncheckedPath{Path: path, Err: err})
				}
			}
			return nil
		})
	}
	return broken, unchecked
}
