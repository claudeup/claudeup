// ABOUTME: Profile subcommands for managing Claude Code profiles
// ABOUTME: Implements all profile lifecycle commands (list, apply, save, diff, delete, rename, etc.)
package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/claudeup/claudeup/v5/internal/backup"
	"github.com/claudeup/claudeup/v5/internal/breadcrumb"
	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/config"
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/internal/ui"
	"github.com/spf13/cobra"
)

var (
	profileSaveDescription  string
	profileCloneFromFlag    string
	profileCloneDescription string
	profileDiffOriginal     bool
	profileDiffScope        string
	profileDiffUser         bool
	profileDiffProject      bool
	profileDiffLocal        bool
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage Claude Code configuration profiles",
	Long: `Profiles are saved configurations of plugins, MCP servers, and marketplaces.

Use profiles to:
  - Create custom profiles with the interactive wizard (create)
  - Clone existing profiles (clone)
  - Save your current setup for later (save)
  - Switch between different configurations (use)
  - Share configurations between machines`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles",
	Long: `List all available profiles.

Profiles marked (customized) are built-in profiles that have been modified and saved locally.

Use 'claudeup profile status' to see effective configuration across all scopes.`,
	Args: cobra.NoArgs,
	RunE: runProfileList,
}

var profileApplyCmd = &cobra.Command{
	Use:     "apply <name>",
	Aliases: []string{"use"},
	Short:   "Apply a profile to Claude Code",
	Long: `Apply a profile's configuration to your Claude Code installation.

SCOPES:
  --project        Apply to current project (.claude/settings.json)
  --local          Apply to current project, but not shared (personal overrides)
  --user           Apply globally (default, affects all projects)

REPLACE MODE:
  --replace        Replace user-scope settings instead of adding to them.
                   By default, user-scope plugins are preserved (additive).
                   Project and local scopes are always replaced.

When applying multi-scope profiles, existing user-scope plugins not in the
profile are preserved by default. If extras are detected, you will be prompted
to choose between keeping them or replacing to match the profile exactly.

Use --replace to skip the prompt and always replace.
Use -y to skip the prompt and always keep extras (additive).

DRY RUN:
  --dry-run        Show what would change without making any modifications.

Precedence: local > project > user. Plugins from all scopes are active simultaneously.

For team projects, use --project to create a shareable configuration that
teammates can apply with 'claudeup profile apply'.

Shows a diff of changes before applying. Prompts for confirmation unless -y is used.`,
	Example: `  # Apply profile (adds to existing user config)
  claudeup profile apply backend-stack

  # Preview changes without applying
  claudeup profile apply backend-stack --dry-run

  # Replace user-scope config with profile
  claudeup profile apply backend-stack --replace

  # Replace without prompts (for scripting)
  claudeup profile apply backend-stack --replace -y

  # Set up a profile for your team (creates .claude/settings.json)
  claudeup profile apply backend-stack --project

  # Force the post-apply setup wizard to run
  claudeup profile apply my-profile --setup`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileApply,
}

var profileSaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save current Claude Code state to a profile",
	Long: `Saves your current Claude Code configuration (plugins, MCP servers, marketplaces) to a profile.

When no name is given, defaults to the last-applied profile (from 'profile apply').

MULTI-SCOPE CAPTURE:
  Save captures settings from ALL scopes (user, project, local) and stores them
  in a structured format. When the profile is applied, each scope's settings are
  restored to the correct location.

  Profiles are always saved to the user profiles directory.
  For team sharing, use 'profile apply <name> --project' to apply the
  profile at project scope, which creates .claude/settings.json for version control.

If the profile exists, prompts for confirmation unless -y is used.`,
	Example: `  # Save current state (defaults to last-applied profile)
  claudeup profile save

  # Save current state to a named profile
  claudeup profile save my-tools

  # Save with confirmation prompt
  claudeup profile save team-config`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileSave,
}

var profileCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new profile with interactive wizard",
	Long: `Interactive wizard for creating custom profiles.

Guides you through:
  1. Selecting marketplaces
  2. Choosing plugins (by category or individually)
  3. Setting a description
  4. Optionally applying the profile

If name is not provided, you'll be prompted to enter one.`,
	Example: `  # Create profile with wizard
  claudeup profile create my-profile

  # Create profile, wizard prompts for name
  claudeup profile create`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileCreate,
}

var profileCloneCmd = &cobra.Command{
	Use:   "clone <name>",
	Short: "Clone an existing profile",
	Long: `Creates a new profile by copying an existing one.

Use --from to specify the source profile, or select interactively.
With -y flag, --from is required (no interactive selection).`,
	Example: `  # Clone from specific profile
  claudeup profile clone new-profile --from existing-profile

  # Interactive selection
  claudeup profile clone new-profile`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileClone,
}

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Display a profile's contents",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileShow,
}

var profileStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show effective configuration across all scopes",
	Long: `Display the live effective configuration by reading settings files directly.

Shows plugins from all active scopes (user, project, local) with:
  - Scope grouping (user, project, local)
  - Enabled/disabled status
  - Marketplace summary`,
	Example: `  # Show what Claude is actually running
  claudeup profile status`,
	Args: cobra.NoArgs,
	RunE: runProfileStatus,
}

var profileDiffCmd = &cobra.Command{
	Use:   "diff [name]",
	Short: "Compare a profile against live Claude Code state",
	Long: `Compare a saved profile against the live Claude Code configuration.

Shows what has changed between the profile and the current state across
all scopes (user, project, local). Use this to see drift before running
'profile save' to pull changes back into the profile.

When called without arguments, diffs against the last-applied profile.

Use --original to compare a customized built-in profile against its
embedded original.`,
	Example: `  # Diff against the last-applied profile
  claudeup profile diff

  # Diff a specific profile against live state
  claudeup profile diff my-setup

  # Compare a customized built-in profile to its original
  claudeup profile diff default --original`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileDiff,
}

var profileSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest a profile for the current directory",
	Args:  cobra.NoArgs,
	RunE:  runProfileSuggest,
}

var profileResetCmd = &cobra.Command{
	Use:   "reset <name>",
	Short: "Remove all components installed by a profile",
	Long: `Removes all plugins, MCP servers, and marketplaces that a profile would install.

This is useful for:
  - Testing a profile from scratch
  - Cleaning up before switching to a different profile
  - Removing a profile's effects without applying a new one`,
	Example: `  # Remove everything installed by a profile
  claudeup profile reset my-profile

  # Reset without confirmation prompts
  claudeup profile reset my-profile -y`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileReset,
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a custom user profile",
	Long: `Permanently deletes a custom user profile from disk.

This command only works on custom profiles you've created. For customized
built-in profiles, use 'profile restore' instead.

Note: This only deletes the profile file. It does not uninstall any plugins
or remove any configuration. Use 'profile reset' first if you want to
remove the profile's installed components.`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileDelete,
}

var profileRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore a built-in profile to its original state",
	Long: `Removes your customizations from a built-in profile, restoring it to the
original version embedded in claudeup.

This command only works on built-in profiles that you've customized (shown
with "(customized)" in the profile list). For custom profiles, use
'profile delete' instead.

Note: This only removes your customized profile file. It does not uninstall
any plugins or remove any configuration. Use 'profile reset' first if you
want to remove the profile's installed components.`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileRestore,
}

var profileRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a custom user profile",
	Long: `Renames a custom user profile.

This command only works on custom profiles you've created. Built-in profiles
cannot be renamed.`,
	Args: cobra.ExactArgs(2),
	RunE: runProfileRename,
}

// Flags for profile apply command
var (
	profileApplySetup         bool
	profileApplyNoInteractive bool
	profileApplyForce         bool
	profileApplyScope         string
	profileApplyReinstall     bool
	profileApplyNoProgress    bool
	profileApplyReplace       bool
	profileApplyDryRun        bool
	// Scope aliases (shorthand for --scope)
	profileApplyUser    bool
	profileApplyProject bool
	profileApplyLocal   bool
)

// Flags for profile save command
var (
	profileSaveScope   string
	profileSaveUser    bool
	profileSaveProject bool
	profileSaveLocal   bool
)

// Flags for profile list command
var profileListAll bool

// Flags for profile clean command
var (
	profileCleanScope   string
	profileCleanProject bool
	profileCleanLocal   bool
)

// Flags for profile create command
var (
	profileCreateDescription  string
	profileCreateMarketplaces []string
	profileCreatePlugins      []string
	profileCreateFromFile     string
	profileCreateFromStdin    bool
)

var profileCleanCmd = &cobra.Command{
	Use:   "clean <plugin>",
	Short: "Remove orphaned plugin from config and profile",
	Long: `Remove orphaned plugins from project or local scope.

This command removes plugins that are enabled but no longer installed.
If the plugin is also in your saved profile definition, this command will offer to
remove it from the profile as well (preventing future reinstall attempts).

Use this to clean up issues detected by 'claudeup status' or 'claudeup doctor'.`,
	Example: `  # Remove plugin from project scope
  claudeup profile clean --project nextjs-vercel-pro@claude-code-templates

  # Remove plugin from local scope
  claudeup profile clean --local my-plugin@marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileClean,
}

// resolveScopeFlags resolves scope from either --scope string or boolean aliases (--user, --project, --local).
// Returns empty string if no scope specified. Returns error if multiple scopes specified.
func resolveScopeFlags(scopeStr string, userFlag, projectFlag, localFlag bool) (string, error) {
	count := 0
	if scopeStr != "" {
		count++
	}
	if userFlag {
		count++
	}
	if projectFlag {
		count++
	}
	if localFlag {
		count++
	}

	if count > 1 {
		return "", fmt.Errorf("cannot specify multiple scope flags; use only one of --scope, --user, --project, or --local")
	}

	if userFlag {
		return "user", nil
	}
	if projectFlag {
		return "project", nil
	}
	if localFlag {
		return "local", nil
	}
	return scopeStr, nil
}

func runProfileClean(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	// Resolve scope from --scope or boolean aliases
	resolvedScope, err := resolveScopeFlags(profileCleanScope, false, profileCleanProject, profileCleanLocal)
	if err != nil {
		return err
	}
	profileCleanScope = resolvedScope

	// Validate scope flag is provided
	if profileCleanScope == "" {
		return fmt.Errorf("scope required: use --project or --local")
	}

	// Validate scope value
	var scope profile.Scope
	switch profileCleanScope {
	case "project":
		scope = profile.ScopeProject
	case "local":
		scope = profile.ScopeLocal
	default:
		return fmt.Errorf("invalid scope %q: must be 'project' or 'local'", profileCleanScope)
	}

	// Get current directory
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Remove plugin from Claude settings
	// Note: Plugins are defined in profiles, so we only need to disable them
	// in settings. To fully remove a plugin from a profile, edit the profile
	// definition itself.
	scopeForSettings := profileCleanScope // "project" or "local"
	settings, err := claude.LoadSettingsForScope(scopeForSettings, claudeDir, projectDir)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	if settings == nil {
		return fmt.Errorf("no settings file found at %s scope", scope.String())
	}
	if !settings.IsPluginEnabled(pluginName) {
		return fmt.Errorf("plugin %q not found in %s scope settings", pluginName, scope.String())
	}

	// Remove the plugin entirely (not just disable) to prevent Claude validation errors
	if settings.EnabledPlugins != nil {
		delete(settings.EnabledPlugins, pluginName)
	}

	// Save updated settings
	if err := claude.SaveSettingsForScope(scopeForSettings, claudeDir, projectDir, settings); err != nil {
		return fmt.Errorf("failed to remove from settings: %w", err)
	}

	// Success message
	scopeName := scope.String()
	settingsFile := ".claude/settings.json"
	if scope == profile.ScopeLocal {
		settingsFile = ".claude/settings.local.json"
	}
	ui.PrintSuccess(fmt.Sprintf("Removed %s from %s scope (%s)", pluginName, scopeName, settingsFile))

	return nil
}

// profileExists checks if a profile file exists on disk, searching subdirectories
func profileExists(profilesDir, name string) bool {
	paths, err := profile.FindProfilePaths(profilesDir, name)
	if err != nil {
		return false
	}
	return len(paths) > 0
}

// profileExistsAtRoot checks if a profile file exists at the root of profilesDir.
// Use this instead of profileExists when the operation writes to root (Save always writes to root).
func profileExistsAtRoot(profilesDir, name string) bool {
	_, err := os.Stat(filepath.Join(profilesDir, name+".json"))
	return err == nil
}

// resolveProfileArg resolves a profile name or path reference to an absolute file path.
// If the name is ambiguous (multiple profiles share the same name), it prompts interactively
// or returns an error in non-interactive (--yes) mode.
func resolveProfileArg(profilesDir, nameOrPath string) (string, error) {
	paths, err := profile.FindProfilePaths(profilesDir, nameOrPath)
	if err != nil {
		return "", fmt.Errorf("failed to search profiles: %w", err)
	}

	switch len(paths) {
	case 0:
		return "", fmt.Errorf("profile %q not found", nameOrPath)
	case 1:
		return paths[0], nil
	default:
		// Multiple matches -- need disambiguation
		relPaths := make([]string, len(paths))
		for i, p := range paths {
			rel, err := filepath.Rel(profilesDir, p)
			if err != nil {
				rel = p
			}
			relPaths[i] = strings.TrimSuffix(filepath.ToSlash(rel), ".json")
		}

		if config.YesFlag {
			return "", &profile.AmbiguousProfileError{Name: nameOrPath, Paths: relPaths}
		}

		// Interactive disambiguation
		fmt.Printf("\nMultiple profiles match %q:\n\n", nameOrPath)
		for i, rp := range relPaths {
			fmt.Printf("  %d) %s\n", i+1, rp)
		}
		fmt.Println()

		fmt.Print("Enter number: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		input = strings.TrimSpace(input)

		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(paths) {
			return "", fmt.Errorf("invalid selection: %s (must be 1-%d)", input, len(paths))
		}

		return paths[num-1], nil
	}
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileListCmd.Flags().BoolVarP(&profileListAll, "all", "a", false, "Show all profiles including hidden ones (prefixed with _)")

	profileCmd.AddCommand(profileApplyCmd)
	profileCmd.AddCommand(profileSaveCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileCloneCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileStatusCmd)
	profileCmd.AddCommand(profileDiffCmd)
	profileCmd.AddCommand(profileSuggestCmd)
	profileCmd.AddCommand(profileResetCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileRestoreCmd)
	profileCmd.AddCommand(profileRenameCmd)
	profileCmd.AddCommand(profileCleanCmd)

	// Add flags to profile create command
	profileCreateCmd.Flags().StringVar(&profileCreateDescription, "description", "", "Profile description")
	profileCreateCmd.Flags().StringSliceVar(&profileCreateMarketplaces, "marketplace", nil, "Marketplace in owner/repo format (can be repeated)")
	profileCreateCmd.Flags().StringSliceVar(&profileCreatePlugins, "plugin", nil, "Plugin in name@marketplace-ref format (can be repeated)")
	profileCreateCmd.Flags().StringVar(&profileCreateFromFile, "from-file", "", "Create profile from JSON file")
	profileCreateCmd.Flags().BoolVar(&profileCreateFromStdin, "from-stdin", false, "Create profile from JSON on stdin")

	profileCloneCmd.Flags().StringVar(&profileCloneFromFlag, "from", "", "Source profile to copy from")
	profileCloneCmd.Flags().StringVar(&profileCloneDescription, "description", "", "Custom description for the profile")

	profileSaveCmd.Flags().StringVar(&profileSaveDescription, "description", "", "Custom description for the profile")
	profileSaveCmd.Flags().StringVar(&profileSaveScope, "scope", "", "Save only settings from specified scope: user, project, local")
	profileSaveCmd.Flags().BoolVar(&profileSaveUser, "user", false, "Save only user scope settings")
	profileSaveCmd.Flags().BoolVar(&profileSaveProject, "project", false, "Save only project scope settings")
	profileSaveCmd.Flags().BoolVar(&profileSaveLocal, "local", false, "Save only local scope settings")
	// Add flags to profile apply command
	profileApplyCmd.Flags().BoolVar(&profileApplySetup, "setup", false, "Force post-apply setup wizard to run")
	profileApplyCmd.Flags().BoolVar(&profileApplyNoInteractive, "no-interactive", false, "Skip post-apply setup wizard (for CI/scripting)")
	profileApplyCmd.Flags().BoolVarP(&profileApplyForce, "force", "f", false, "Force reapply even with unsaved changes")
	profileApplyCmd.Flags().StringVar(&profileApplyScope, "scope", "", "Apply scope: user, project, or local (default: user)")
	profileApplyCmd.Flags().BoolVar(&profileApplyUser, "user", false, fmt.Sprintf("Apply to user scope (%s/)", config.ClaudeDirDisplay()))
	profileApplyCmd.Flags().BoolVar(&profileApplyProject, "project", false, "Apply to project scope (.claude/settings.json)")
	profileApplyCmd.Flags().BoolVar(&profileApplyLocal, "local", false, "Apply to local scope (.claude/settings.local.json)")
	profileApplyCmd.Flags().BoolVar(&profileApplyReinstall, "reinstall", false, "Force reinstall all plugins and marketplaces")
	profileApplyCmd.Flags().BoolVar(&profileApplyNoProgress, "no-progress", false, "Disable progress display (for CI/scripting)")
	profileApplyCmd.Flags().BoolVar(&profileApplyReplace, "replace", false, "Replace user-scope settings instead of adding to them")
	profileApplyCmd.Flags().BoolVar(&profileApplyDryRun, "dry-run", false, "Show what would be changed without making modifications")

	// Add flags to profile diff command
	profileDiffCmd.Flags().BoolVar(&profileDiffOriginal, "original", false, "Compare a customized built-in profile against its embedded original")
	profileDiffCmd.Flags().StringVar(&profileDiffScope, "scope", "", "Use the last-applied profile at this scope: user, project, local")
	profileDiffCmd.Flags().BoolVar(&profileDiffUser, "user", false, "Use the last-applied profile at user scope")
	profileDiffCmd.Flags().BoolVar(&profileDiffProject, "project", false, "Use the last-applied profile at project scope")
	profileDiffCmd.Flags().BoolVar(&profileDiffLocal, "local", false, "Use the last-applied profile at local scope")

	// Add flags to profile clean command
	profileCleanCmd.Flags().StringVar(&profileCleanScope, "scope", "", "Config scope to clean: project or local (required)")
	profileCleanCmd.Flags().BoolVar(&profileCleanProject, "project", false, "Clean from project scope (.claude/settings.json)")
	profileCleanCmd.Flags().BoolVar(&profileCleanLocal, "local", false, "Clean from local scope (.claude/settings.local.json)")

}

func runProfileList(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()
	cwd, _ := os.Getwd()

	// Load all profiles from both user and project directories
	allProfiles, err := profile.ListAll(profilesDir, cwd)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	// Load embedded (built-in) profiles
	embeddedProfiles, embeddedErr := profile.ListEmbeddedProfiles()
	if embeddedErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load built-in profiles: %v\n", embeddedErr)
		embeddedProfiles = []*profile.Profile{} // Prevent nil slice panic
	}

	// Track which profiles exist on disk
	profileOnDisk := make(map[string]bool)
	for _, p := range allProfiles {
		profileOnDisk[p.Name] = true
	}

	// Collect custom profiles (exclude those that shadow built-ins)
	var customProfiles []*profile.ProfileWithSource
	for _, p := range allProfiles {
		if !profile.IsEmbeddedProfile(p.Name) {
			customProfiles = append(customProfiles, p)
		}
	}

	// Check if we have any profiles to show
	if len(embeddedProfiles) == 0 && len(customProfiles) == 0 {
		ui.PrintInfo("No profiles found.")
		fmt.Printf("  %s Create one with: claudeup profile save <name>\n", ui.Muted(ui.SymbolArrow))
		return nil
	}

	// Load breadcrumb once and compute applied markers for all profiles
	appliedMarkers := loadAppliedProfiles(profilesDir)

	// Show built-in profiles section
	if len(embeddedProfiles) > 0 {
		fmt.Println(ui.Bold("Built-in profiles"))
		fmt.Println()
		for _, p := range embeddedProfiles {
			desc := p.Description

			// If shadowed on disk, check if content actually differs
			if profileOnDisk[p.Name] {
				for _, dp := range allProfiles {
					if dp.Name == p.Name {
						desc = dp.Description
						if !p.Equal(dp.Profile) {
							desc += " " + ui.Muted("(customized)")
						}
						break
					}
				}
			}

			if info, ok := appliedMarkers[p.Name]; ok {
				if info.Modified {
					desc += " " + ui.Muted("(applied, modified)")
				} else {
					desc += " " + ui.Muted("(applied)")
				}
			}

			if desc == "" {
				desc = ui.Muted("(no description)")
			}
			fmt.Printf("  %-20s %s\n", p.Name, desc)
		}
		fmt.Println()
	}

	// Filter and group user profiles
	var visibleProfiles []*profile.ProfileWithSource
	hiddenCount := 0
	for _, p := range customProfiles {
		displayName := p.DisplayName()
		if displayName == "" {
			continue
		}
		// Use the leaf segment for hidden-prefix detection (e.g. "team/backend/_internal" → "_internal")
		baseName := displayName
		if idx := strings.LastIndex(displayName, "/"); idx >= 0 {
			baseName = displayName[idx+1:]
		}
		if !profileListAll && strings.HasPrefix(baseName, "_") {
			hiddenCount++
			continue
		}
		visibleProfiles = append(visibleProfiles, p)
	}

	if len(visibleProfiles) > 0 {
		fmt.Println(ui.Bold("Your profiles"))
		fmt.Println()

		// Separate into ungrouped and grouped by path prefix
		type groupedProfile struct {
			shortName string
			profile   *profile.ProfileWithSource
		}
		ungrouped := []groupedProfile{}
		groups := map[string][]groupedProfile{}
		var groupNames []string

		for _, p := range visibleProfiles {
			displayName := p.DisplayName()
			// Split on first "/" to group by top-level directory
			if idx := strings.Index(displayName, "/"); idx >= 0 {
				groupName := displayName[:idx]
				shortName := displayName[idx+1:]
				if _, exists := groups[groupName]; !exists {
					groupNames = append(groupNames, groupName)
				}
				groups[groupName] = append(groups[groupName], groupedProfile{shortName, p})
			} else {
				ungrouped = append(ungrouped, groupedProfile{displayName, p})
			}
		}
		sort.Strings(groupNames)

		// Calculate column width from longest short name across all profiles
		maxNameLen := 0
		for _, gp := range ungrouped {
			if len(gp.shortName) > maxNameLen {
				maxNameLen = len(gp.shortName)
			}
		}
		for _, name := range groupNames {
			for _, gp := range groups[name] {
				if len(gp.shortName) > maxNameLen {
					maxNameLen = len(gp.shortName)
				}
			}
		}
		if maxNameLen < 20 {
			maxNameLen = 20
		}
		colFmt := fmt.Sprintf("%%-%ds %%s\n", maxNameLen)

		// Render ungrouped profiles first
		for _, gp := range ungrouped {
			desc := gp.profile.Description
			if desc == "" {
				desc = ui.Muted("(no description)")
			}
			if gp.profile.IsStack() {
				desc += " " + ui.Muted("[stack]")
			}
			if info, ok := appliedMarkers[gp.profile.DisplayName()]; ok {
				if info.Modified {
					desc += " " + ui.Muted("(applied, modified)")
				} else {
					desc += " " + ui.Muted("(applied)")
				}
			}
			fmt.Printf("  "+colFmt, gp.shortName, desc)
		}

		// Render grouped profiles
		for _, groupName := range groupNames {
			// Blank line before first group only when ungrouped profiles precede it,
			// and always before subsequent groups
			if len(ungrouped) > 0 || groupName != groupNames[0] {
				fmt.Println()
			}
			fmt.Printf("  %s/\n", groupName)
			for _, gp := range groups[groupName] {
				desc := gp.profile.Description
				if desc == "" {
					desc = ui.Muted("(no description)")
				}
				if gp.profile.IsStack() {
					desc += " " + ui.Muted("[stack]")
				}
				if info, ok := appliedMarkers[gp.profile.DisplayName()]; ok {
					if info.Modified {
						desc += " " + ui.Muted("(applied, modified)")
					} else {
						desc += " " + ui.Muted("(applied)")
					}
				}
				fmt.Printf("    "+colFmt, gp.shortName, desc)
			}
		}
		fmt.Println()

		if hiddenCount > 0 {
			fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("(%d hidden profiles not shown, use --all to include them)", hiddenCount)))
			fmt.Println()
		}
	} else if hiddenCount > 0 {
		// All custom profiles are hidden
		fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("(%d hidden profiles not shown, use --all to include them)", hiddenCount)))
		fmt.Println()
	}

	fmt.Printf("%s Use 'claudeup profile status' to see effective configuration\n", ui.Muted(ui.SymbolArrow))
	fmt.Printf("%s Use 'claudeup profile show <name>' for profile details\n", ui.Muted(ui.SymbolArrow))
	fmt.Printf("%s Use 'claudeup profile apply <name>' to apply a profile\n", ui.Muted(ui.SymbolArrow))

	return nil
}

func runProfileApply(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	// Resolve scope from --scope or boolean aliases
	resolvedScope, err := resolveScopeFlags(profileApplyScope, profileApplyUser, profileApplyProject, profileApplyLocal)
	if err != nil {
		return err
	}
	profileApplyScope = resolvedScope

	name := args[0]

	// "current" is reserved for the live status view (profile show current)
	if name == "current" {
		return fmt.Errorf("'current' is a reserved name. Use 'claudeup profile status' to view current configuration")
	}

	var scope profile.Scope

	if profileApplyScope != "" {
		// Explicit scope from flag
		var err error
		scope, err = profile.ParseScope(profileApplyScope)
		if err != nil {
			return err
		}
	} else {
		// Auto-detect: if .mcp.json exists, use project scope
		if profile.MCPJSONExists(cwd) {
			scope = profile.ScopeProject
			ui.PrintInfo("Detected existing project configuration, using project scope")
			fmt.Println()
		} else {
			scope = profile.ScopeUser
		}
	}

	// Handle --replace flag: clear scope before applying
	if profileApplyReplace {
		scopeStr := string(scope)
		settingsPath, err := claude.SettingsPathForScope(scopeStr, claudeDir, cwd)
		if err != nil {
			return err
		}

		// Check if there's anything to clear
		if _, err := os.Stat(settingsPath); err == nil {
			claudeupHome := config.MustClaudeupHome()

			// Create backup unless -y (silent mode) is used
			if !config.YesFlag {
				var backupPath string
				if scopeStr == "local" {
					backupPath, err = backup.SaveLocalScopeBackup(claudeupHome, cwd, settingsPath)
				} else {
					backupPath, err = backup.SaveScopeBackup(claudeupHome, scopeStr, settingsPath)
				}
				if err != nil {
					ui.PrintWarning(fmt.Sprintf("Could not create backup: %v", err))
				} else {
					fmt.Printf("  Backup saved: %s\n", backupPath)
				}
			}

			// Clear the scope
			if err := clearScope(scopeStr, settingsPath, claudeDir); err != nil {
				return fmt.Errorf("failed to clear %s scope: %w", scopeStr, err)
			}
			fmt.Printf("Cleared %s scope\n", scopeStr)
			fmt.Println()
		}
	}

	explicitScope := profileApplyScope != ""
	return applyProfileWithScope(name, scope, explicitScope)
}

// applyProfileWithScope applies a profile at the specified scope.
// This is the core implementation shared by runProfileApply and runProfileCreate.
// explicitScope indicates whether the user explicitly passed a scope flag.
func applyProfileWithScope(name string, scope profile.Scope, explicitScope bool) error {
	profilesDir := getProfilesDir()
	cwd, _ := os.Getwd()

	// Resolve to exact path if ambiguous (handles nested profiles)
	var p *profile.Profile
	resolvedPath, resolveErr := resolveProfileArg(profilesDir, name)
	if resolveErr == nil {
		// Found on disk -- load from resolved path
		var loadErr error
		p, loadErr = profile.LoadFromPath(resolvedPath)
		if loadErr != nil {
			return fmt.Errorf("failed to load profile %q: %w", name, loadErr)
		}
		// Normalize name to the display name format (relative path without .json)
		// so breadcrumbs match what profile list uses for lookups.
		if relPath, err := filepath.Rel(profilesDir, resolvedPath); err == nil {
			name = strings.TrimSuffix(relPath, ".json")
		}
	} else {
		// Surface ambiguity and other non-not-found errors directly
		var ambigErr *profile.AmbiguousProfileError
		if errors.As(resolveErr, &ambigErr) {
			return resolveErr
		}
		// Not found on disk -- try embedded profiles
		var embeddedErr error
		p, embeddedErr = profile.GetEmbeddedProfile(name)
		if embeddedErr != nil {
			return fmt.Errorf("profile %q not found: %w", name, resolveErr)
		}
	}

	// Resolve stack profiles (composable includes).
	// Track whether the original profile was a stack so we always route through
	// ApplyAllScopes, even if the resolved profile has only flat fields.
	wasStack := p.IsStack()
	if wasStack {
		if explicitScope {
			return fmt.Errorf("stack profiles define their own scopes; --scope is not supported with stacks")
		}
		loader := &profile.DirLoader{ProfilesDir: profilesDir}
		resolved, resolveIncludesErr := profile.ResolveIncludes(p, loader)
		if resolveIncludesErr != nil {
			return fmt.Errorf("failed to resolve includes: %w", resolveIncludesErr)
		}
		p = resolved
	}

	// Security check FIRST: warn about hooks from non-embedded profiles
	// Users should know about hooks before seeing the diff
	if p.PostApply != nil && !profile.IsEmbeddedProfile(name) {
		fmt.Println()
		ui.PrintWarning("Security Warning: This profile contains a post-apply hook.")
		fmt.Println("  Hooks execute arbitrary commands on your system.")
		fmt.Println("  Only proceed if you trust the source of this profile.")
		if p.PostApply.Script != "" {
			fmt.Printf("  Script: %s\n", p.PostApply.Script)
		}
		if p.PostApply.Command != "" {
			fmt.Printf("  Command: %s\n", p.PostApply.Command)
		}
		fmt.Println()
		if !confirmProceed() {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Use the global claudeDir from root.go (set via --claude-dir flag)
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Compute and show diff (scope-aware to avoid confusing Remove actions for user-scope items)
	diff, err := profile.ComputeDiffWithScope(p, claudeDir, claudeJSONPath, claudeupHome, profile.DiffOptions{
		Scope:      scope,
		ProjectDir: cwd,
	})
	if err != nil {
		return fmt.Errorf("failed to compute changes: %w", err)
	}

	// Check if we need to run the hook (before early return)
	scriptDir := profile.GetEmbeddedProfileScriptDir(name)
	if scriptDir != "" {
		defer os.RemoveAll(scriptDir)
	}

	hookOpts := profile.HookOptions{
		ForceSetup:    profileApplySetup,
		NoInteractive: profileApplyNoInteractive,
		ScriptDir:     scriptDir,
	}

	shouldRunHook := profile.ShouldRunHook(p, claudeDir, claudeJSONPath, claudeupHome, hookOpts)

	// Multi-scope profiles and stacks always need to apply (diff only checks one scope)
	needsApply := p.IsMultiScope() || wasStack || hasDiffChanges(diff) || shouldRunHook

	// If no changes and no hook to run, we're done
	if !needsApply {
		if profileApplyDryRun {
			ui.PrintInfo("Dry run: no changes would be applied.")
			return nil
		}

		// Record breadcrumb even when no changes needed -- user applied this profile
		recordBreadcrumb(name, cwd, scopesForBreadcrumb(scope, p))

		if p.SkipPluginDiff {
			ui.PrintSuccess("No configuration changes needed.")
			fmt.Println()

			// If profile has a post-apply hook (wizard), show instructions
			if p.PostApply != nil {
				ui.PrintInfo("This profile uses an interactive wizard to configure plugins.")
				fmt.Printf("  Run: %s\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s --setup", name)))
			} else {
				ui.PrintInfo("Note: This profile does not manage plugins.")
				fmt.Println("      Your existing plugins will remain unchanged.")
			}
			return nil
		} else {
			ui.PrintSuccess("No changes needed - profile already matches current state.")
			return nil
		}
	}

	// Show diff and confirm if there are changes
	if hasDiffChanges(diff) || p.IsMultiScope() || wasStack {
		fmt.Println(ui.RenderDetail("Profile", ui.Bold(name)))
		fmt.Println()
		if p.IsMultiScope() || wasStack {
			// Show per-scope summary for multi-scope profiles
			showMultiScopeSummary(p)
		} else {
			showDiff(diff)
		}
		fmt.Println()

		// Dry run mode: show what would change, then exit
		if profileApplyDryRun {
			ui.PrintInfo("Dry run - no changes made")
			return nil
		}

		// Detect extras (live items not in profile) for multi-scope profiles.
		// Show interactive prompt so users can choose add vs replace mode.
		extrasPrompted := false
		if (p.IsMultiScope() || wasStack) && !profileApplyReplace && !config.YesFlag && !profileApplyForce {
			live, snapErr := profile.SnapshotAllScopes("live", claudeDir, claudeJSONPath, cwd, claudeupHome)
			if snapErr != nil {
				ui.PrintWarning(fmt.Sprintf("Could not snapshot live config for extras detection: %v", snapErr))
			} else {
				extras := profile.UserScopeExtras(p.AsPerScope(), live.AsPerScope())
				if len(extras) > 0 {
					if promptApplyMode(extras) {
						profileApplyReplace = true
					}
					extrasPrompted = true
				}
			}
		}

		// Skip confirmation if extras prompt was already shown (it serves as confirmation)
		if !extrasPrompted && !profileApplyForce && !confirmProceed() {
			ui.PrintMuted("Cancelled.")
			return nil
		}
	} else {
		// No changes, but hook needs to run
		fmt.Println(ui.RenderDetail("Profile", ui.Bold(name)))
		fmt.Println()
		if profileApplyDryRun {
			ui.PrintInfo("Dry run - no changes would be made")
			return nil
		}
		ui.PrintInfo("No configuration changes needed.")
		if profileApplySetup {
			fmt.Println("Running setup wizard...")
		}
		fmt.Println()
	}

	// Apply (hook decision was already made above)
	fmt.Println()

	chain := buildSecretChain()

	var result *profile.ApplyResult

	// Use ApplyAllScopes for multi-scope profiles and stacks, ApplyWithOptions for legacy
	if p.IsMultiScope() || wasStack {
		ui.PrintInfo("Applying profile (all scopes)...")
		applyOpts := &profile.ApplyAllScopesOptions{
			ReplaceUserScope: profileApplyReplace, // --replace flag controls user scope behavior
		}
		result, err = profile.ApplyAllScopes(p, claudeDir, claudeJSONPath, cwd, claudeupHome, chain, applyOpts)
		if err != nil {
			return fmt.Errorf("failed to apply profile: %w", err)
		}

	} else {
		ui.PrintInfo(fmt.Sprintf("Applying profile (%s scope)...", scope))

		// Build apply options with progress tracking enabled by default
		opts := profile.ApplyOptions{
			Scope:        scope,
			ProjectDir:   cwd,
			Reinstall:    profileApplyReinstall,
			ShowProgress: !profileApplyNoProgress, // Enable concurrent apply with progress UI
		}
		// Add progress callback for sequential installs (user scope)
		if !profileApplyNoProgress {
			opts.Progress = ui.PluginProgress()
		}

		result, err = profile.ApplyWithOptions(p, claudeDir, claudeJSONPath, claudeupHome, chain, opts)
		if err != nil {
			return fmt.Errorf("failed to apply profile: %w", err)
		}
	}

	showApplyResults(result)

	// Silently clean up stale plugin entries
	cleanupStalePlugins(claudeDir)

	fmt.Println()
	ui.PrintSuccess("Profile applied!")
	recordBreadcrumb(name, cwd, scopesForBreadcrumb(scope, p))

	// Scope-specific post-apply messages
	if scope == profile.ScopeProject {
		fmt.Println()
		ui.PrintInfo("Project files created:")

		filesToAdd := []string{".claude/settings.json"}
		fmt.Printf("  %s %s (project plugins)\n", ui.Success(ui.SymbolSuccess), ".claude/settings.json")

		if profile.MCPJSONExists(cwd) {
			fmt.Printf("  %s %s (MCP servers - Claude auto-loads)\n", ui.Success(ui.SymbolSuccess), profile.MCPConfigFile)
			filesToAdd = append(filesToAdd, profile.MCPConfigFile)
		}

		fmt.Println()
		fmt.Printf("%s Consider adding these to git:\n", ui.Muted(ui.SymbolArrow))
		fmt.Printf("  git add %s\n", strings.Join(filesToAdd, " "))
	}

	// Run post-apply hook if applicable (decision was made before apply)
	if shouldRunHook {
		fmt.Println()
		if err := profile.RunHook(p, hookOpts); err != nil {
			ui.PrintError(fmt.Sprintf("Post-apply hook failed: %v", err))
			return fmt.Errorf("hook execution failed: %w", err)
		}
	}

	return nil
}

// dirExists returns true if path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// sameResolvedDir returns true if a and b resolve to the same directory
// after cleaning paths and resolving symlinks.
func sameResolvedDir(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return ra == rb
}

// recordBreadcrumb writes a breadcrumb entry recording which profile was applied.
// Errors are logged but do not fail the operation.
func recordBreadcrumb(name, projectDir string, scopes []string) {
	if err := breadcrumb.Record(claudeupHome, name, projectDir, scopes); err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not save breadcrumb: %v", err))
	}
}

// scopesForBreadcrumb determines which scopes a profile apply touched.
func scopesForBreadcrumb(scope profile.Scope, p *profile.Profile) []string {
	if p.PerScope != nil {
		var scopes []string
		if p.PerScope.User != nil {
			scopes = append(scopes, "user")
		}
		if p.PerScope.Project != nil {
			scopes = append(scopes, "project")
		}
		if p.PerScope.Local != nil {
			scopes = append(scopes, "local")
		}
		if len(scopes) > 0 {
			return scopes
		}
	}
	// Flat profiles and flat-only stacks apply to the requested scope
	return []string{string(scope)}
}

// cleanupStalePlugins removes plugin entries with invalid paths
// This is called automatically after profile apply to clean up zombie entries
func cleanupStalePlugins(claudeDir string) {
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		// Only warn if not a simple "file not found" - that's expected on fresh installs
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "  Warning: could not load plugins for cleanup: %v\n", err)
		}
		return
	}

	removed := 0
	for _, sp := range plugins.GetPluginsAtScopes(claude.ValidScopes) {
		if !sp.PathExists() {
			if plugins.RemovePluginAtScope(sp.Name, sp.Scope) {
				removed++
			}
		}
	}

	if removed > 0 {
		if err := claude.SavePlugins(claudeDir, plugins); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not save cleaned plugins: %v\n", err)
		} else {
			fmt.Printf("  Cleaned up %d stale plugin entries\n", removed)
		}
	}
}

func runProfileSave(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Profiles are always saved to user profiles directory
	profilesDir := getProfilesDir()

	// Resolve scope flags
	resolvedScope, err := resolveScopeFlags(profileSaveScope, profileSaveUser, profileSaveProject, profileSaveLocal)
	if err != nil {
		return err
	}
	if resolvedScope != "" {
		if err := claude.ValidateScope(resolvedScope); err != nil {
			return err
		}
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Default to last-applied profile from breadcrumb
		bc, err := breadcrumb.Load(claudeupHome)
		if err != nil {
			return fmt.Errorf("failed to read breadcrumb: %w", err)
		}
		bc = breadcrumb.FilterByDir(bc, cwd)

		var bcScope string
		if resolvedScope != "" {
			profileName, appliedAt, ok := breadcrumb.ForScope(bc, resolvedScope)
			if !ok {
				return fmt.Errorf("no profile has been applied at %s scope. Run: claudeup profile save <name>", resolvedScope)
			}
			name = profileName
			bcScope = fmt.Sprintf("applied %s, %s scope", appliedAt.Format("Jan 2, 2006"), resolvedScope)
		} else {
			profileName, scope := breadcrumb.HighestPrecedence(bc)
			if profileName == "" {
				return fmt.Errorf("no profile has been applied yet. Run: claudeup profile save <name>")
			}
			name = profileName
			entry := bc[scope]
			bcScope = fmt.Sprintf("applied %s, %s scope", entry.AppliedAt.Format("Jan 2, 2006"), scope)
		}
		fmt.Printf("Saving to %q (%s)\n\n", name, bcScope)
	}

	// "current" is reserved as a keyword for live status view
	if name == "current" {
		return fmt.Errorf("'current' is a reserved name. Use a different profile name")
	}

	// Check if profile already exists at root.
	// Only checks root-level since Save() writes to root profiles directory.
	if profileExistsAtRoot(profilesDir, name) {
		if !config.YesFlag {
			fmt.Printf("%s Profile %q already exists. ", ui.Warning(ui.SymbolWarning), name)
			choice := promptChoice("Overwrite?", "n")
			if choice != "y" && choice != "yes" {
				ui.PrintMuted("Cancelled.")
				return nil
			}
		}
	}

	// Use the global claudeDir from root.go (set via --claude-dir flag)
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create snapshot, optionally filtered to a single scope
	p, err := profile.SnapshotAllScopes(name, claudeDir, claudeJSONPath, cwd, claudeupHome)
	if err != nil {
		return fmt.Errorf("failed to snapshot current state: %w", err)
	}
	if resolvedScope != "" {
		p.FilterToScope(resolvedScope)
	}

	// When overwriting, preserve localItems from the existing profile.
	// LocalItems accumulate from various sources and the snapshot would pick
	// up items enabled by other tools that aren't part of this profile.
	// Marketplaces are already filtered by plugin references at snapshot time.
	existingProfile, _ := profile.Load(profilesDir, name) // OK if doesn't exist
	if existingProfile != nil {
		p.PreserveFrom(existingProfile)
	}

	// Handle description
	if profileSaveDescription != "" {
		// User provided explicit description via flag
		p.Description = profileSaveDescription
	} else if existingProfile != nil {
		// Preserve custom descriptions from existing profile
		if existingProfile.Description != "" && existingProfile.Description != "Snapshot of current Claude Code configuration" {
			p.Description = existingProfile.Description
		}
	}

	// Save
	if err := profile.Save(profilesDir, p); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	scopeLabel := resolvedScope + " scope"
	if resolvedScope == "" {
		scopeLabel = scopeLabelFromProfile(p)
	}
	ui.PrintSuccess(fmt.Sprintf("Saved profile %q (%s)", name, scopeLabel))
	fmt.Println()

	// Show per-scope plugin counts for multi-scope profiles
	if p.IsMultiScope() {
		for _, s := range []struct {
			label    string
			settings *profile.ScopeSettings
		}{
			{"User", p.PerScope.User},
			{"Project", p.PerScope.Project},
			{"Local", p.PerScope.Local},
		} {
			if s.settings == nil {
				continue
			}
			if len(s.settings.Plugins) > 0 {
				fmt.Println(ui.Indent(ui.RenderDetail(s.label+" plugins", fmt.Sprintf("%d", len(s.settings.Plugins))), 1))
			}
			if n := countExtensions(s.settings.Extensions); n > 0 {
				fmt.Println(ui.Indent(ui.RenderDetail(s.label+" extensions", fmt.Sprintf("%d", n)), 1))
			}
		}
	} else {
		fmt.Println(ui.Indent(ui.RenderDetail("Plugins", fmt.Sprintf("%d", len(p.Plugins))), 1))
	}
	fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", fmt.Sprintf("%d", len(p.Marketplaces))), 1))

	return nil
}

// scopeLabelFromProfile returns a human-readable label describing which
// scopes a profile contains (e.g. "user scope", "all scopes").
func scopeLabelFromProfile(p *profile.Profile) string {
	if p == nil || p.PerScope == nil {
		return "all scopes"
	}
	var scopes []string
	if p.PerScope.User != nil {
		scopes = append(scopes, "user")
	}
	if p.PerScope.Project != nil {
		scopes = append(scopes, "project")
	}
	if p.PerScope.Local != nil {
		scopes = append(scopes, "local")
	}
	switch len(scopes) {
	case 0:
		return "all scopes"
	case 1:
		return scopes[0] + " scope"
	default:
		return "all scopes"
	}
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// "show current" delegates to live effective config view
	if name == "current" {
		return runProfileStatus(cmd, nil)
	}

	// Load the profile (try disk first, then embedded)
	p, err := loadProfileWithFallback(profilesDir, name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	fmt.Printf("Profile: %s\n", p.Name)
	if p.Description != "" {
		fmt.Printf("Description: %s\n", p.Description)
	}

	// Stack profiles: show include tree and resolved summary
	if p.IsStack() {
		fmt.Printf("Type:     stack\n")
		fmt.Println()
		showStackIncludes(p, profilesDir)

		// Resolve and show the merged profile details
		loader := &profile.DirLoader{ProfilesDir: profilesDir}
		resolved, resolveErr := profile.ResolveIncludes(p, loader)
		if resolveErr != nil {
			fmt.Printf("\n%s Failed to resolve includes: %v\n", ui.SymbolError, resolveErr)
			return nil
		}

		showResolvedSummary(resolved)
		fmt.Println()
		p = resolved
	}

	fmt.Println()

	if p.IsMultiScope() {
		showMultiScopeProfile(p)
	} else {
		showLegacyProfile(p)
	}

	if len(p.Marketplaces) > 0 {
		fmt.Println("  Marketplaces:")
		for _, m := range p.Marketplaces {
			fmt.Printf("    - %s\n", m.DisplayName())
		}
		fmt.Println()
	}

	return nil
}

// scopeEntry pairs a scope label with its settings for display.
type scopeEntry struct {
	label    string
	settings *profile.ScopeSettings
}

func showMultiScopeProfile(p *profile.Profile) {
	scopes := []scopeEntry{
		{"User", p.PerScope.User},
		{"Project", p.PerScope.Project},
		{"Local", p.PerScope.Local},
	}

	// Top-level extensions are unscoped; use as fallback for user scope display
	unscopedExt := p.Extensions

	for _, s := range scopes {
		var plugins []string
		var mcpServers []profile.MCPServer
		var ext *profile.ExtensionSettings

		if s.settings != nil {
			plugins = s.settings.Plugins
			mcpServers = s.settings.MCPServers
			ext = s.settings.Extensions
		}

		// For user scope, fall back to top-level extensions if scope has none
		if s.label == "User" && ext == nil && unscopedExt != nil {
			ext = unscopedExt
		}

		if len(plugins) == 0 && len(mcpServers) == 0 && countExtensions(ext) == 0 {
			continue
		}

		fmt.Printf("  %s\n", ui.Bold(s.label+" scope"))

		indent := "    "

		if len(plugins) > 0 {
			fmt.Printf("%sPlugins:\n", indent)
			for _, plug := range plugins {
				fmt.Printf("%s  - %s\n", indent, plug)
			}
		}

		displayMCPServers(mcpServers, indent)
		displayExtensionCategories(ext, indent)

		fmt.Println()
	}
}

func showLegacyProfile(p *profile.Profile) {
	indent := "  "

	if len(p.Plugins) > 0 {
		fmt.Printf("%sPlugins:\n", indent)
		for _, plug := range p.Plugins {
			fmt.Printf("%s  - %s\n", indent, plug)
		}
	}

	displayMCPServers(p.MCPServers, indent)

	displayExtensionCategories(p.Extensions, indent)

	fmt.Println()
}

// extensionCategory maps a display label to a getter for that category's extensions.
type extensionCategory struct {
	label  string
	getter func(*profile.ExtensionSettings) []string
}

// extensionCategories defines the display order and accessors for extension categories.
var extensionCategories = []extensionCategory{
	{"Agents", func(l *profile.ExtensionSettings) []string { return l.Agents }},
	{"Commands", func(l *profile.ExtensionSettings) []string { return l.Commands }},
	{"Skills", func(l *profile.ExtensionSettings) []string { return l.Skills }},
	{"Hooks", func(l *profile.ExtensionSettings) []string { return l.Hooks }},
	{"Rules", func(l *profile.ExtensionSettings) []string { return l.Rules }},
	{"Output Styles", func(l *profile.ExtensionSettings) []string { return l.OutputStyles }},
}

// displayMCPServers prints MCP server list at the given indent level.
func displayMCPServers(servers []profile.MCPServer, indent string) {
	if len(servers) == 0 {
		return
	}
	fmt.Printf("%sMCP Servers:\n", indent)
	for _, m := range servers {
		fmt.Printf("%s  - %s (%s)\n", indent, m.Name, m.Command)
		secretKeys := make([]string, 0, len(m.Secrets))
		for envVar := range m.Secrets {
			secretKeys = append(secretKeys, envVar)
		}
		sort.Strings(secretKeys)
		for _, envVar := range secretKeys {
			fmt.Printf("%s      requires: %s\n", indent, envVar)
		}
	}
}

// displayExtensionCategories prints extensions grouped by category at the given indent level.
func displayExtensionCategories(ext *profile.ExtensionSettings, indent string) {
	if ext == nil {
		return
	}
	hasItems := false
	for _, c := range extensionCategories {
		if len(c.getter(ext)) > 0 {
			hasItems = true
			break
		}
	}
	if !hasItems {
		return
	}
	fmt.Printf("%sExtensions:\n", indent)
	for _, c := range extensionCategories {
		items := c.getter(ext)
		if len(items) == 0 {
			continue
		}
		fmt.Printf("%s  %s:\n", indent, c.label)
		for _, item := range items {
			fmt.Printf("%s    - %s\n", indent, item)
		}
	}
}

// countExtensions returns the total number of extensions across all categories.
func countExtensions(items *profile.ExtensionSettings) int {
	if items == nil {
		return 0
	}
	count := 0
	for _, c := range extensionCategories {
		count += len(c.getter(items))
	}
	return count
}

// showStackIncludes displays the include tree for a stack profile.
// Nested stacks are expanded one level to show their sub-includes.
// Uses DirLoader for consistent fallback semantics with ResolveIncludes.
func showStackIncludes(p *profile.Profile, profilesDir string) {
	loader := &profile.DirLoader{ProfilesDir: profilesDir}
	fmt.Println("Includes:")
	for _, name := range p.Includes {
		// Try to load the included profile to check if it's also a stack
		included, err := loader.LoadProfile(name)
		if err != nil {
			fmt.Printf("  %s %s\n", name, ui.Muted(fmt.Sprintf("(%v)", err)))
			continue
		}
		if included.IsStack() {
			subNames := make([]string, len(included.Includes))
			copy(subNames, included.Includes)
			fmt.Printf("  %s %s [%s]\n", name, ui.Muted("->"), strings.Join(subNames, ", "))
		} else {
			fmt.Printf("  %s\n", name)
		}
	}
}

// showResolvedSummary displays a summary of the resolved (merged) profile contents.
func showResolvedSummary(p *profile.Profile) {
	var parts []string

	if len(p.Marketplaces) > 0 {
		parts = append(parts, fmt.Sprintf("%d marketplaces", len(p.Marketplaces)))
	}

	var pluginTotal int
	var scopeParts []string

	if len(p.Plugins) > 0 {
		pluginTotal += len(p.Plugins)
		scopeParts = append(scopeParts, fmt.Sprintf("%d unscoped", len(p.Plugins)))
	}
	if p.PerScope != nil {
		if p.PerScope.User != nil && len(p.PerScope.User.Plugins) > 0 {
			pluginTotal += len(p.PerScope.User.Plugins)
			scopeParts = append(scopeParts, fmt.Sprintf("%d user", len(p.PerScope.User.Plugins)))
		}
		if p.PerScope.Project != nil && len(p.PerScope.Project.Plugins) > 0 {
			pluginTotal += len(p.PerScope.Project.Plugins)
			scopeParts = append(scopeParts, fmt.Sprintf("%d project", len(p.PerScope.Project.Plugins)))
		}
		if p.PerScope.Local != nil && len(p.PerScope.Local.Plugins) > 0 {
			pluginTotal += len(p.PerScope.Local.Plugins)
			scopeParts = append(scopeParts, fmt.Sprintf("%d local", len(p.PerScope.Local.Plugins)))
		}
	}
	if pluginTotal > 0 {
		if len(scopeParts) > 1 {
			parts = append(parts, fmt.Sprintf("%d plugins (%s)", pluginTotal, strings.Join(scopeParts, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("%d plugins", pluginTotal))
		}
	}

	mcpCount := len(p.MCPServers)
	if p.PerScope != nil {
		if p.PerScope.User != nil {
			mcpCount += len(p.PerScope.User.MCPServers)
		}
		if p.PerScope.Project != nil {
			mcpCount += len(p.PerScope.Project.MCPServers)
		}
		if p.PerScope.Local != nil {
			mcpCount += len(p.PerScope.Local.MCPServers)
		}
	}
	if mcpCount > 0 {
		parts = append(parts, fmt.Sprintf("%d MCP servers", mcpCount))
	}

	if len(parts) > 0 {
		fmt.Printf("\nResolved: %s\n", strings.Join(parts, ", "))
	}
}

func runProfileStatus(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	// Header
	fmt.Printf("Effective configuration for %s\n\n", ui.Bold(cwd))

	// Display the highest-precedence applied profile and its drift status
	profilesDir := getProfilesDir()
	applied := loadAppliedProfiles(profilesDir)
	if info := highestPrecedenceApplied(applied); info != nil {
		modifiedMarker := ""
		if info.Modified {
			modifiedMarker = " " + ui.Muted("(modified)")
		}
		fmt.Printf("  Last applied: %s%s\n", ui.Bold(info.Name), modifiedMarker)
		fmt.Printf("                %s\n\n",
			ui.Muted(fmt.Sprintf("applied %s, %s scope", info.AppliedAt.Format("Jan 2, 2006"), info.Scope)))
	}

	anyContent := false
	var allPluginNames []string
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Determine whether cwd has a distinct project scope.
	// When cwd/.claude is the same directory as claudeDir (e.g., running
	// from ~), project/local settings overlap with user settings.
	hasDistinctProjectScope := false
	if projectClaudeDir := filepath.Join(cwd, ".claude"); dirExists(projectClaudeDir) {
		hasDistinctProjectScope = !sameResolvedDir(projectClaudeDir, claudeDir)
	}

	for _, scope := range []string{"user", "project", "local"} {
		// Skip project/local if cwd has no distinct project scope
		if scope != "user" && !hasDistinctProjectScope {
			continue
		}

		settings, err := claude.LoadSettingsForScope(scope, claudeDir, cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s Failed to load %s scope settings: %v\n",
				ui.Warning("Warning:"), scope, err)
			continue
		}

		var enabled, disabled []string
		for name, isEnabled := range settings.EnabledPlugins {
			if isEnabled {
				enabled = append(enabled, name)
			} else {
				disabled = append(disabled, name)
			}
		}
		sort.Strings(enabled)
		sort.Strings(disabled)

		mcpServers, mcpErr := profile.ReadMCPServersForScope(claudeJSONPath, cwd, scope)
		if mcpErr != nil {
			fmt.Fprintf(os.Stderr, "%s Failed to load %s MCP servers: %v\n",
				ui.Warning("Warning:"), scope, mcpErr)
		}

		var extensions *profile.ExtensionSettings
		if scope == "user" {
			ext, extErr := profile.ReadExtensions(claudeDir, claudeupHome)
			if extErr != nil {
				fmt.Fprintf(os.Stderr, "%s Failed to load user extensions: %v\n",
					ui.Warning("Warning:"), extErr)
			}
			extensions = ext
		} else if scope == "project" {
			extensions = profile.ReadProjectExtensions(cwd)
		}

		if len(enabled) == 0 && len(disabled) == 0 && len(mcpServers) == 0 && countExtensions(extensions) == 0 {
			continue
		}

		anyContent = true
		allPluginNames = append(allPluginNames, enabled...)

		// Scope header
		scopeLabel := formatScopeName(scope)
		fmt.Printf("  %s\n", ui.Bold(scopeLabel+" scope"))

		// Enabled plugins
		if len(enabled) > 0 {
			fmt.Println("    Plugins:")
			for _, name := range enabled {
				fmt.Printf("      - %s\n", name)
			}
		}

		// Disabled plugins
		if len(disabled) > 0 {
			fmt.Println("    Disabled:")
			for _, name := range disabled {
				fmt.Printf("      - %s\n", name)
			}
		}

		displayMCPServers(mcpServers, "    ")
		displayExtensionCategories(extensions, "    ")

		fmt.Println()
	}

	if !anyContent {
		fmt.Printf("  %s\n\n", ui.Muted("No configuration found at any scope."))
	}

	// Marketplaces section
	marketplaces, err := profile.UsedMarketplaces(claudeDir, allPluginNames)
	if err == nil && len(marketplaces) > 0 {
		fmt.Println("  Marketplaces:")
		for _, m := range marketplaces {
			fmt.Printf("    - %s\n", m.DisplayName())
		}
		fmt.Println()
	}

	return nil
}

// pluralS returns "s" if count != 1, otherwise empty string
func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func hasDiffChanges(diff *profile.Diff) bool {
	return len(diff.PluginsToRemove) > 0 ||
		len(diff.PluginsToInstall) > 0 ||
		len(diff.MCPToRemove) > 0 ||
		len(diff.MCPToInstall) > 0 ||
		len(diff.MarketplacesToAdd) > 0 ||
		len(diff.MarketplacesToRemove) > 0
}

func showMultiScopeSummary(p *profile.Profile) {
	if p.PerScope == nil {
		return
	}

	fmt.Printf("  %s\n", ui.Info("Multi-scope profile:"))

	for _, s := range []struct {
		label    string
		settings *profile.ScopeSettings
	}{
		{"User", p.PerScope.User},
		{"Project", p.PerScope.Project},
		{"Local", p.PerScope.Local},
	} {
		if s.settings == nil {
			continue
		}
		var parts []string
		if len(s.settings.Plugins) > 0 {
			parts = append(parts, fmt.Sprintf("%d plugins", len(s.settings.Plugins)))
		}
		if s.settings.Extensions != nil {
			n := countExtensions(s.settings.Extensions)
			if n > 0 {
				parts = append(parts, fmt.Sprintf("%d extensions", n))
			}
		}
		if len(parts) > 0 {
			fmt.Printf("    %-15s %s\n", s.label+" scope:", strings.Join(parts, ", "))
		}
	}
}

func showDiff(diff *profile.Diff) {
	if len(diff.PluginsToRemove) > 0 || len(diff.MCPToRemove) > 0 || len(diff.MarketplacesToRemove) > 0 {
		fmt.Printf("  %s\n", ui.Warning("Remove:"))
		for _, m := range diff.MarketplacesToRemove {
			fmt.Printf("    %s %s\n", ui.Warning("-"), ui.Muted("Marketplace: ")+m.DisplayName())
		}
		for _, p := range diff.PluginsToRemove {
			fmt.Printf("    %s %s\n", ui.Warning("-"), p)
		}
		for _, m := range diff.MCPToRemove {
			fmt.Printf("    %s %s\n", ui.Warning("-"), ui.Muted("MCP: ")+m)
		}
	}

	if len(diff.PluginsToInstall) > 0 || len(diff.MCPToInstall) > 0 || len(diff.MarketplacesToAdd) > 0 {
		fmt.Printf("  %s\n", ui.Success("Install:"))
		for _, m := range diff.MarketplacesToAdd {
			fmt.Printf("    %s %s\n", ui.Success("+"), ui.Muted("Marketplace: ")+m.DisplayName())
		}
		for _, p := range diff.PluginsToInstall {
			fmt.Printf("    %s %s\n", ui.Success("+"), p)
		}
		for _, m := range diff.MCPToInstall {
			secretInfo := ""
			if len(m.Secrets) > 0 {
				for k := range m.Secrets {
					secretInfo = ui.Muted(fmt.Sprintf(" (requires %s)", k))
					break
				}
			}
			fmt.Printf("    %s %s%s\n", ui.Success("+"), ui.Muted("MCP: ")+m.Name, secretInfo)
		}
	}
}

func runProfileDiff(cmd *cobra.Command, args []string) error {
	if profileDiffOriginal {
		if len(args) != 1 {
			return fmt.Errorf("--original requires exactly 1 profile name")
		}
		return runProfileDiffOriginal(cmd, args)
	}

	var name string
	var breadcrumbScope string

	// Resolve scope flags early to detect conflicts with explicit name
	resolvedScope, err := resolveScopeFlags(profileDiffScope, profileDiffUser, profileDiffProject, profileDiffLocal)
	if err != nil {
		return err
	}

	if resolvedScope != "" {
		if err := claude.ValidateScope(resolvedScope); err != nil {
			return err
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if len(args) > 0 {
		if resolvedScope != "" {
			return fmt.Errorf("cannot use --scope/--user/--project/--local with an explicit profile name")
		}
		name = args[0]
	} else {
		// Load breadcrumb for default profile
		bc, err := breadcrumb.Load(claudeupHome)
		if err != nil {
			return fmt.Errorf("failed to read breadcrumb: %w", err)
		}
		bc = breadcrumb.FilterByDir(bc, cwd)

		if resolvedScope != "" {
			profileName, appliedAt, ok := breadcrumb.ForScope(bc, resolvedScope)
			if !ok {
				return fmt.Errorf("no profile has been applied at %s scope. Run: claudeup profile diff <name>", resolvedScope)
			}
			name = profileName
			breadcrumbScope = fmt.Sprintf("applied %s, %s scope", appliedAt.Format("Jan 2, 2006"), resolvedScope)
		} else {
			profileName, scope := breadcrumb.HighestPrecedence(bc)
			if profileName == "" {
				return fmt.Errorf("no profile has been applied yet. Run: claudeup profile diff <name>")
			}
			name = profileName
			entry := bc[scope]
			breadcrumbScope = fmt.Sprintf("applied %s, %s scope", entry.AppliedAt.Format("Jan 2, 2006"), scope)
		}
	}

	profilesDir := getProfilesDir()

	// Load saved profile (disk first, fallback to embedded)
	saved, err := loadProfileWithFallback(profilesDir, name)
	if err != nil {
		var ambigErr *profile.AmbiguousProfileError
		if errors.As(err, &ambigErr) {
			return ambigErr
		}
		if breadcrumbScope != "" {
			return fmt.Errorf("profile %q not found (breadcrumb referenced it). Run: claudeup profile diff <name>", name)
		}
		return fmt.Errorf("profile '%s' not found", name)
	}

	if breadcrumbScope != "" {
		fmt.Printf("Comparing against %q (%s)\n\n", name, breadcrumbScope)
	}

	// Snapshot live state
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")
	live, err := profile.SnapshotAllScopes("live", claudeDir, claudeJSONPath, cwd, claudeupHome)
	if err != nil {
		return fmt.Errorf("failed to snapshot live state: %w", err)
	}

	// Normalize both to PerScope form
	savedNorm := saved.AsPerScope()
	liveNorm := live.AsPerScope()

	// Compute and display diff (skip description -- live snapshots auto-generate descriptions)
	diff := profile.ComputeProfileDiff(savedNorm, liveNorm)
	diff.DescriptionChange = nil
	if diff.IsEmpty() {
		fmt.Printf("Profile '%s' matches live state. No differences.\n", name)
		return nil
	}

	showProfileDiff(diff)
	fmt.Println()
	fmt.Printf("%s Run 'claudeup profile save %s' to update the profile.\n", ui.Info(ui.SymbolArrow), name)
	return nil
}

// showProfileDiff displays a formatted diff between a profile and live state
func showProfileDiff(diff *profile.ProfileDiff) {
	fmt.Printf("Profile '%s' vs live configuration:\n", diff.ProfileName)

	if diff.DescriptionChange != nil {
		fmt.Printf("\n  %s description: %q %s %q\n",
			ui.Warning("~"),
			diff.DescriptionChange[0],
			ui.SymbolArrow,
			diff.DescriptionChange[1])
	}

	for _, sd := range diff.Scopes {
		if len(sd.Items) == 0 {
			continue
		}
		fmt.Printf("\n  %s:\n", capitalizeFirst(sd.Scope))
		for _, item := range sd.Items {
			symbol := ui.Warning("~")
			switch item.Op {
			case profile.DiffAdded:
				symbol = ui.Success("+")
			case profile.DiffRemoved:
				symbol = ui.Error("-")
			}

			detail := ""
			if item.Detail != "" {
				detail = fmt.Sprintf(" (%s)", item.Detail)
			}

			fmt.Printf("    %s %s: %s%s\n", symbol, item.Kind, item.Name, detail)
		}
	}

	added, removed, modified := diff.Counts()
	if diff.DescriptionChange != nil {
		modified++
	}
	total := added + removed + modified
	parts := []string{}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", added))
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", removed))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	fmt.Printf("\n%d %s (%s)\n",
		total,
		pluralize(total, "difference", "differences"),
		strings.Join(parts, ", "))
}

// capitalizeFirst uppercases the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// pluralize returns singular or plural form based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// promptApplyMode shows extra plugins (live items not in profile) and asks the
// user whether to keep them or replace to match the profile exactly.
// Returns true if the user chose replace mode, false for additive.
// Callers receive only DiffPlugin items (UserScopeExtras filters to plugins).
func promptApplyMode(extras []profile.DiffItem) bool {
	n := len(extras)
	fmt.Printf("  %s %d %s in your current config %s not in this profile:\n",
		ui.Info(ui.SymbolInfo), n, pluralize(n, "plugin", "plugins"), pluralize(n, "is", "are"))
	for _, e := range extras {
		fmt.Printf("    - %s\n", e.Name)
	}
	fmt.Println()
	fmt.Println("  How would you like to apply?")
	fmt.Printf("    [A] Add profile settings, keep extras (default)\n")
	fmt.Printf("    [R] Replace %s match profile exactly (removes extras)\n", ui.Muted("--"))
	fmt.Println()

	for {
		choice := strings.TrimSpace(promptChoice("  Choice", "A"))
		if strings.EqualFold(choice, "A") {
			return false
		}
		if strings.EqualFold(choice, "R") {
			return true
		}
		fmt.Printf("  Please enter A or R.\n")
	}
}

// runProfileDiffOriginal compares a customized built-in profile against its embedded original
func runProfileDiffOriginal(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Check if the profile is a built-in
	if !profile.IsEmbeddedProfile(name) {
		return fmt.Errorf("'%s' is not a built-in profile. Use 'profile diff --original' only with built-in profiles", name)
	}

	// Get the embedded original
	embedded, err := profile.GetEmbeddedProfile(name)
	if err != nil {
		return fmt.Errorf("failed to load embedded profile: %w", err)
	}

	// Check if there's a customized version on disk
	resolvedPath, err := resolveProfileArg(profilesDir, name)
	if err != nil {
		// Surface ambiguity errors so the user can disambiguate
		var ambigErr *profile.AmbiguousProfileError
		if errors.As(err, &ambigErr) {
			return err
		}
		// No customized version found - no differences
		fmt.Printf("Profile '%s' has not been customized.\n", name)
		fmt.Println("No differences from the built-in version.")
		return nil
	}

	// Load the customized version
	customized, err := profile.LoadFromPath(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to load customized profile: %w", err)
	}

	// Compare using Profile.Equal()
	if embedded.Equal(customized) {
		fmt.Printf("Profile '%s' matches the built-in version.\n", name)
		fmt.Println("No differences.")
		return nil
	}

	// Show differences
	fmt.Printf("Differences in '%s' from built-in:\n\n", name)

	// Compare description
	if embedded.Description != customized.Description {
		fmt.Printf("  %s description: %q %s %q\n",
			ui.Warning("~"),
			embedded.Description,
			ui.SymbolArrow,
			customized.Description)
	}

	// Compare plugins
	embeddedPlugins := make(map[string]bool)
	for _, p := range embedded.Plugins {
		embeddedPlugins[p] = true
	}
	customizedPlugins := make(map[string]bool)
	for _, p := range customized.Plugins {
		customizedPlugins[p] = true
	}

	// Added plugins
	for _, p := range customized.Plugins {
		if !embeddedPlugins[p] {
			fmt.Printf("  %s plugin: %s\n", ui.Success("+"), p)
		}
	}

	// Removed plugins
	for _, p := range embedded.Plugins {
		if !customizedPlugins[p] {
			fmt.Printf("  %s plugin: %s\n", ui.Error("-"), p)
		}
	}

	// Compare marketplaces
	embeddedMarkets := make(map[string]bool)
	for _, m := range embedded.Marketplaces {
		embeddedMarkets[m.DisplayName()] = true
	}
	customizedMarkets := make(map[string]bool)
	for _, m := range customized.Marketplaces {
		customizedMarkets[m.DisplayName()] = true
	}

	// Added marketplaces
	for _, m := range customized.Marketplaces {
		if !embeddedMarkets[m.DisplayName()] {
			fmt.Printf("  %s marketplace: %s\n", ui.Success("+"), m.DisplayName())
		}
	}

	// Removed marketplaces
	for _, m := range embedded.Marketplaces {
		if !customizedMarkets[m.DisplayName()] {
			fmt.Printf("  %s marketplace: %s\n", ui.Error("-"), m.DisplayName())
		}
	}

	// Compare MCP servers
	embeddedMCP := make(map[string]bool)
	for _, m := range embedded.MCPServers {
		embeddedMCP[m.Name] = true
	}
	customizedMCP := make(map[string]bool)
	for _, m := range customized.MCPServers {
		customizedMCP[m.Name] = true
	}

	// Added MCP servers
	for _, m := range customized.MCPServers {
		if !embeddedMCP[m.Name] {
			fmt.Printf("  %s MCP server: %s\n", ui.Success("+"), m.Name)
		}
	}

	// Removed MCP servers
	for _, m := range embedded.MCPServers {
		if !customizedMCP[m.Name] {
			fmt.Printf("  %s MCP server: %s\n", ui.Error("-"), m.Name)
		}
	}

	return nil
}

func runProfileSuggest(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load all profiles
	entries, err := profile.List(profilesDir)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No profiles available.")
		fmt.Println("Create one with: claudeup profile save <name>")
		return nil
	}

	// Extract profile pointers for SuggestProfile
	profiles := make([]*profile.Profile, len(entries))
	for i, e := range entries {
		profiles[i] = e.Profile
	}

	// Find matching profiles
	suggested := profile.SuggestProfile(cwd, profiles)

	if suggested == nil {
		fmt.Println("No profile matches the current directory.")
		fmt.Println()
		fmt.Println("Available profiles:")
		for _, p := range entries {
			fmt.Printf("  - %s\n", p.DisplayName())
		}
		return nil
	}

	fmt.Println(ui.RenderDetail("Suggested profile", ui.Bold(suggested.Name)))
	if suggested.Description != "" {
		fmt.Printf("  %s\n", ui.Muted(suggested.Description))
	}
	fmt.Println()

	fmt.Printf("%s Apply this profile? %s: ", ui.Info(ui.SymbolArrow), ui.Muted("[Y/n]"))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		ui.PrintMuted("Cancelled.")
		return nil
	}
	choice := strings.TrimSpace(strings.ToLower(input))
	if choice == "" || choice == "y" || choice == "yes" {
		// Run the use command
		return runProfileApply(cmd, []string{suggested.Name})
	}

	ui.PrintMuted("Cancelled.")
	return nil
}

// loadProfileWithFallback tries to load a profile from disk first,
// falling back to embedded profiles if not found on disk.
// Ambiguity errors (multiple profiles with the same name) are not swallowed.
func loadProfileWithFallback(profilesDir, name string) (*profile.Profile, error) {
	// Try disk first
	p, err := profile.Load(profilesDir, name)
	if err == nil {
		return p, nil
	}

	// Propagate ambiguity errors -- don't fall back to embedded
	var ambigErr *profile.AmbiguousProfileError
	if errors.As(err, &ambigErr) {
		return nil, err
	}

	// Fall back to embedded profiles
	return profile.GetEmbeddedProfile(name)
}

// appliedProfileInfo holds a breadcrumbed profile's name, scope, timestamp,
// and whether live settings have drifted from the saved profile.
type appliedProfileInfo struct {
	Name      string
	Scope     string
	AppliedAt time.Time
	Modified  bool
}

// loadAppliedProfiles loads the breadcrumb file once, then checks each
// breadcrumbed profile for drift against the live configuration.
// Returns a map from profile name to its applied info.
// At most 3 entries exist (one per scope: user, project, local).
func loadAppliedProfiles(profilesDir string) map[string]appliedProfileInfo {
	bc, err := breadcrumb.Load(claudeupHome)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not read breadcrumb: %v", err))
		return nil
	}
	if len(bc) == 0 {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		// Keep only user-scope breadcrumbs (directory-independent).
		if userEntry, ok := bc["user"]; ok {
			bc = breadcrumb.File{"user": userEntry}
			cwd = ""
		} else {
			return nil
		}
	} else {
		bc = breadcrumb.FilterByDir(bc, cwd)
		if len(bc) == 0 {
			return nil
		}
	}

	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")
	live, err := profile.SnapshotAllScopes("live", claudeDir, claudeJSONPath, cwd, claudeupHome)
	if err != nil {
		return nil
	}

	result := make(map[string]appliedProfileInfo, len(bc))
	for scope, entry := range bc {
		if _, exists := result[entry.Profile]; exists {
			continue
		}

		saved, err := loadProfileWithFallback(profilesDir, entry.Profile)
		if err != nil {
			continue
		}

		savedPerScope := saved.AsPerScope()

		// Narrow the saved profile to only scopes with active breadcrumbs.
		// This prevents marketplace diffs caused by plugins at scopes whose
		// breadcrumbs were filtered out (e.g., project plugins when the
		// project breadcrumb is from a different directory).
		activeScopes := make(map[string]bool, len(bc))
		for s := range bc {
			activeScopes[s] = true
		}
		savedForDiff := profile.FilterToScopes(savedPerScope, activeScopes)
		diff := profile.ComputeProfileDiff(savedForDiff, live.AsPerScope())
		// Snapshot descriptions are auto-generated and always differ from
		// saved profile descriptions; exclude them from drift detection.
		diff.DescriptionChange = nil

		// Exclude marketplace diffs from drift detection. Marketplaces are
		// infrastructure managed automatically with plugin installs -- they
		// never change independently of plugins. Including them causes false
		// positives due to mismatches between registry key matching (used by
		// snapshots) and DisplayName matching (used by diff comparison).
		for i := range diff.Scopes {
			filtered := diff.Scopes[i].Items[:0]
			for _, item := range diff.Scopes[i].Items {
				if item.Kind != profile.DiffMarketplace {
					filtered = append(filtered, item)
				}
			}
			diff.Scopes[i].Items = filtered
		}

		// Only check drift at scopes where BOTH the saved profile defines
		// settings AND a breadcrumb is active. This prevents false positives
		// from scopes the profile covers but whose breadcrumbs were filtered
		// out (e.g., project-scope breadcrumb from a different directory).
		filtered := diff.Scopes[:0]
		for _, sd := range diff.Scopes {
			if _, hasBreadcrumb := bc[sd.Scope]; !hasBreadcrumb {
				continue
			}
			switch sd.Scope {
			case "user":
				if savedPerScope.PerScope.User != nil {
					filtered = append(filtered, sd)
				}
			case "project":
				if savedPerScope.PerScope.Project != nil {
					filtered = append(filtered, sd)
				}
			case "local":
				if savedPerScope.PerScope.Local != nil {
					filtered = append(filtered, sd)
				}
			}
		}
		diff.Scopes = filtered

		result[entry.Profile] = appliedProfileInfo{
			Name:      entry.Profile,
			Scope:     scope,
			AppliedAt: entry.AppliedAt,
			Modified:  !diff.IsEmpty(),
		}
	}
	return result
}

// highestPrecedenceApplied returns the applied profile info for the
// highest-precedence scope (local > project > user), or nil if none.
func highestPrecedenceApplied(applied map[string]appliedProfileInfo) *appliedProfileInfo {
	if applied == nil {
		return nil
	}
	for _, scope := range []string{"local", "project", "user"} {
		for _, info := range applied {
			if info.Scope == scope {
				return &info
			}
		}
	}
	return nil
}

// getAllProfiles returns all available profiles (user + embedded), with user profiles taking precedence
func getAllProfiles(profilesDir string) ([]*profile.Profile, error) {
	// Load user profiles
	userEntries, err := profile.List(profilesDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list user profiles: %w", err)
	}

	// Extract profile pointers and track names
	userNames := make(map[string]bool)
	userProfiles := make([]*profile.Profile, 0, len(userEntries))
	for _, e := range userEntries {
		userNames[e.Name] = true
		userProfiles = append(userProfiles, e.Profile)
	}

	// Load embedded profiles (skip ones that exist on disk)
	embeddedProfiles, err := profile.ListEmbeddedProfiles()
	if err != nil {
		// Non-fatal - just use user profiles
		return userProfiles, nil
	}

	// Combine: user profiles + embedded profiles not on disk
	result := make([]*profile.Profile, 0, len(userProfiles)+len(embeddedProfiles))
	result = append(result, userProfiles...)
	for _, p := range embeddedProfiles {
		if !userNames[p.Name] {
			result = append(result, p)
		}
	}

	return result, nil
}

// promptProfileSelection displays an interactive menu to select a profile
func promptProfileSelection(profilesDir, newName string) (*profile.Profile, error) {
	profiles, err := getAllProfiles(profilesDir)
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles available to copy from")
	}

	fmt.Printf("\nWhich profile should %q be based on?\n\n", newName)
	for i, p := range profiles {
		desc := p.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Printf("  %d) %-20s %s\n", i+1, p.Name, desc)
	}
	fmt.Println()

	fmt.Print("Enter number or name: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	// Validate non-empty input
	if input == "" {
		return nil, fmt.Errorf("no selection made")
	}

	// Try as number first
	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(profiles) {
			return profiles[num-1], nil
		}
		return nil, fmt.Errorf("invalid selection: %d (must be 1-%d)", num, len(profiles))
	}

	// Try as name
	for _, p := range profiles {
		if p.Name == input {
			return p, nil
		}
	}

	return nil, fmt.Errorf("profile %q not found", input)
}

// validateNewProfileName validates a profile name and checks it doesn't already exist at root.
// Only checks root-level since Save() always writes to the root profiles directory.
func validateNewProfileName(name, profilesDir string) error {
	if err := profile.ValidateName(name); err != nil {
		return err
	}
	if profileExistsAtRoot(profilesDir, name) {
		return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
	}
	return nil
}

// registryKeysFromInstalled loads marketplace registry keys from the user's
// Claude installation. Returns nil if the registry cannot be loaded (e.g.,
// no plugins directory or fresh install).
func registryKeysFromInstalled() ([]string, error) {
	registry, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load marketplace registry: %w", err)
	}
	keys := make([]string, 0, len(registry))
	for key := range registry {
		keys = append(keys, key)
	}
	return keys, nil
}

// saveAndPrintNewProfile saves a new profile and prints a success message.
func saveAndPrintNewProfile(p *profile.Profile, profilesDir string) error {
	registryKeys, err := registryKeysFromInstalled()
	if err != nil {
		return fmt.Errorf("cannot validate plugin marketplaces: %w", err)
	}
	if err := p.ValidateMarketplaceRefs(registryKeys); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	if err := profile.Save(profilesDir, p); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}
	fmt.Printf("Profile %q created successfully.\n\n", p.Name)
	fmt.Printf("  Marketplaces: %d\n", len(p.Marketplaces))
	fmt.Printf("  Plugins: %d\n", len(p.Plugins))
	fmt.Printf("\nRun 'claudeup profile apply %s' to use it.\n", p.Name)
	return nil
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()

	// Detect mode: file, flags, or wizard
	hasFileInput := profileCreateFromFile != "" || profileCreateFromStdin
	hasFlagsInput := len(profileCreateMarketplaces) > 0 || len(profileCreatePlugins) > 0 || profileCreateDescription != ""

	// Mutual exclusivity checks
	if profileCreateFromFile != "" && profileCreateFromStdin {
		return fmt.Errorf("cannot use both --from-file and --from-stdin")
	}
	if hasFileInput && hasFlagsInput && (len(profileCreateMarketplaces) > 0 || len(profileCreatePlugins) > 0) {
		return fmt.Errorf("cannot combine --from-file/--from-stdin with --marketplace/--plugin flags")
	}

	// Name is required for non-interactive modes
	if (hasFileInput || hasFlagsInput) && len(args) == 0 {
		return fmt.Errorf("profile name is required for non-interactive mode")
	}

	// Flags mode
	if hasFlagsInput && !hasFileInput {
		name := args[0]
		if err := validateNewProfileName(name, profilesDir); err != nil {
			return err
		}

		newProfile, err := profile.CreateFromFlags(name, profileCreateDescription, profileCreateMarketplaces, profileCreatePlugins)
		if err != nil {
			return err
		}

		return saveAndPrintNewProfile(newProfile, profilesDir)
	}

	// File/stdin mode
	if hasFileInput {
		name := args[0]
		if err := validateNewProfileName(name, profilesDir); err != nil {
			return err
		}

		var reader io.Reader
		if profileCreateFromStdin {
			reader = os.Stdin
		} else {
			f, err := os.Open(profileCreateFromFile)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()
			reader = f
		}

		newProfile, err := profile.CreateFromReader(name, reader, profileCreateDescription)
		if err != nil {
			return err
		}

		return saveAndPrintNewProfile(newProfile, profilesDir)
	}

	// Wizard mode (interactive) - existing code continues...

	// Step 1: Get profile name
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		promptedName, err := profile.PromptForName()
		if err != nil {
			return fmt.Errorf("failed to get profile name: %w", err)
		}
		name = promptedName
	}

	// Validate name
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	// Check if profile already exists at root (Save writes to root)
	if profileExistsAtRoot(profilesDir, name) {
		return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
	}

	// Welcome message
	fmt.Println(ui.RenderDetail("Creating profile", ui.Bold(name)))

	// Step 2: Select marketplaces
	availableMarketplaces := profile.GetAvailableMarketplaces()
	selectedMarketplaces, err := profile.SelectMarketplaces(availableMarketplaces)
	if err != nil {
		return fmt.Errorf("failed to select marketplaces: %w", err)
	}

	if len(selectedMarketplaces) == 0 {
		return fmt.Errorf("no marketplaces selected (at least one required)")
	}

	// Step 3: Select plugins for each marketplace
	allPlugins := make([]string, 0)
	for _, marketplace := range selectedMarketplaces {
		fmt.Println()
		fmt.Printf("Selecting plugins from %s...\n", marketplace.DisplayName())

		plugins, err := profile.SelectPluginsForMarketplace(marketplace)
		if err != nil {
			return fmt.Errorf("failed to select plugins from %s: %w", marketplace.DisplayName(), err)
		}

		allPlugins = append(allPlugins, plugins...)
	}

	// Step 4: Generate and edit description
	autoDesc := profile.GenerateWizardDescription(len(selectedMarketplaces), len(allPlugins))
	description, err := profile.PromptForDescription(autoDesc)
	if err != nil {
		return fmt.Errorf("failed to get description: %w", err)
	}

	// Step 5: Create profile
	newProfile := &profile.Profile{
		Name:         name,
		Description:  description,
		Marketplaces: selectedMarketplaces,
		Plugins:      allPlugins,
		MCPServers:   []profile.MCPServer{},
	}

	// Step 6: Show summary
	fmt.Println()
	fmt.Println(ui.RenderDetail("Profile summary", ""))
	fmt.Println(ui.Indent(ui.RenderDetail("Name", name), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("Description", description), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", fmt.Sprintf("%d", len(selectedMarketplaces))), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("Plugins", fmt.Sprintf("%d", len(allPlugins))), 1))
	fmt.Println()

	// Step 7: Save profile
	if err := profile.Save(profilesDir, newProfile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Saved profile %q", name))
	fmt.Println()

	// Step 8: Prompt to apply
	fmt.Print("Apply this profile now? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	applyInput, err := reader.ReadString('\n')
	if err != nil {
		// Default to not applying on error
		fmt.Printf("Profile saved. Use '%s' to apply.\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s", name)))
		return nil
	}
	applyChoice := strings.TrimSpace(strings.ToLower(applyInput))

	if applyChoice == "" || applyChoice == "y" || applyChoice == "yes" {
		// Apply the profile at user scope (not project scope).
		// This prevents accidentally overwriting existing project configs when
		// the user just wants to create and try a new profile.
		if err := applyProfileWithScope(name, profile.ScopeUser, true); err != nil {
			return err
		}

		// Save snapshot after applying to sync profile with actual installed state
		// This prevents the profile from showing as "modified" immediately after creation
		fmt.Println()
		ui.PrintInfo("Saving profile snapshot...")
		claudeJSONPath := filepath.Join(claudeDir, ".claude.json")
		snapshot, err := profile.Snapshot(name, claudeDir, claudeJSONPath, claudeupHome)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to save snapshot: %v", err))
			return nil
		}

		// Preserve the wizard-created description
		snapshot.Description = newProfile.Description

		if err := profile.Save(profilesDir, snapshot); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to save snapshot: %v", err))
			return nil
		}

		return nil
	}

	fmt.Printf("Profile saved. Use '%s' to apply.\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s", name)))
	return nil
}

func runProfileClone(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// "current" is reserved as a keyword for live status view
	if name == "current" {
		return fmt.Errorf("'current' is a reserved name. Use a different profile name")
	}

	// Check if target profile already exists at root (Save writes to root)
	if profileExistsAtRoot(profilesDir, name) {
		return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
	}

	// Determine source profile
	var sourceProfile *profile.Profile
	var err error

	if profileCloneFromFlag != "" {
		// Explicit --from flag
		sourceProfile, err = loadProfileWithFallback(profilesDir, profileCloneFromFlag)
		if err != nil {
			return fmt.Errorf("profile %q not found: %w", profileCloneFromFlag, err)
		}
	} else if config.YesFlag {
		return fmt.Errorf("--from <profile> is required when using -y")
	} else {
		// Interactive selection
		sourceProfile, err = promptProfileSelection(profilesDir, name)
		if err != nil {
			return err
		}
	}

	// Clone the profile with the new name
	newProfile := sourceProfile.Clone(name)

	// Handle description
	if profileCloneDescription != "" {
		// User provided explicit description via flag
		newProfile.Description = profileCloneDescription
	} else if newProfile.Description == "Snapshot of current Claude Code configuration" {
		// Source has old generic description, replace with auto-generated
		newProfile.Description = newProfile.GenerateDescription()
	}
	// Otherwise preserve source's custom description

	// Save
	if err := profile.Save(profilesDir, newProfile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Created profile %q (based on %q)", name, sourceProfile.Name))
	fmt.Println()
	fmt.Printf("  MCP Servers:   %d\n", len(newProfile.MCPServers))
	fmt.Printf("  Marketplaces:  %d\n", len(newProfile.Marketplaces))
	fmt.Printf("  Plugins:       %d\n", len(newProfile.Plugins))

	return nil
}

func runProfileReset(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Load the profile
	p, err := loadProfileWithFallback(profilesDir, name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	// Show what will be removed
	fmt.Println(ui.RenderDetail("Reset profile", ui.Bold(name)))
	fmt.Println()

	// Use the global claudeDir from root.go (set via --claude-dir flag)
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Get current state to show what plugins will be removed
	current, _ := profile.Snapshot("current", claudeDir, claudeJSONPath, claudeupHome)

	// Build lookup from repo to marketplace name
	repoToName := profile.BuildRepoToNameLookup(claudeDir)

	// Find plugins that match profile's marketplaces
	var pluginsToRemove []string
	if current != nil {
		for _, m := range p.Marketplaces {
			suffix := "@" + strings.ReplaceAll(m.Repo, "/", "-")
			for _, plugin := range current.Plugins {
				if strings.HasSuffix(plugin, suffix) {
					pluginsToRemove = append(pluginsToRemove, plugin)
				}
			}
		}
	}

	hasChanges := len(pluginsToRemove) > 0 || len(p.MCPServers) > 0 || len(p.Marketplaces) > 0

	if !hasChanges {
		fmt.Println("Nothing to remove - profile has no installed components.")
		return nil
	}

	fmt.Println("  Will remove:")
	for _, plugin := range pluginsToRemove {
		fmt.Printf("    - Plugin: %s\n", plugin)
	}
	for _, mcp := range p.MCPServers {
		fmt.Printf("    - MCP: %s\n", mcp.Name)
	}
	for _, m := range p.Marketplaces {
		// Show the registered marketplace name, falling back to repo if not found
		displayName := m.DisplayName()
		if name, found := repoToName[m.Repo]; found {
			displayName = name
		}
		fmt.Printf("    - Marketplace: %s\n", displayName)
	}
	fmt.Println()

	if !confirmProceed() {
		fmt.Println("Cancelled.")
		return nil
	}

	// Execute reset
	fmt.Println()
	fmt.Println("Removing profile components...")

	result, err := profile.Reset(p, claudeDir, claudeJSONPath, claudeupHome)
	if err != nil {
		return fmt.Errorf("failed to reset profile: %w", err)
	}

	// Show results
	if len(result.PluginsRemoved) > 0 {
		fmt.Printf("  Removed %d plugins\n", len(result.PluginsRemoved))
	}
	if len(result.MCPServersRemoved) > 0 {
		fmt.Printf("  Removed %d MCP servers\n", len(result.MCPServersRemoved))
	}
	if len(result.MarketplacesRemoved) > 0 {
		fmt.Printf("  Removed %d marketplaces\n", len(result.MarketplacesRemoved))
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("Some errors occurred:")
		for _, err := range result.Errors {
			ui.PrintError(fmt.Sprintf("%v", err))
		}
	}

	fmt.Println()
	ui.PrintSuccess("Profile reset complete!")

	return nil
}

func runProfileDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Resolve the profile to an exact path (single walk handles both existence and disambiguation)
	profilePath, resolveErr := resolveProfileArg(profilesDir, name)

	// Check if it's a built-in profile
	if profile.IsEmbeddedProfile(name) {
		// Surface ambiguity errors so the user can disambiguate
		var ambigErr *profile.AmbiguousProfileError
		if errors.As(resolveErr, &ambigErr) {
			return resolveErr
		}
		if resolveErr == nil {
			return fmt.Errorf("profile %q is a customized built-in profile. Use 'claudeup profile restore %s' instead", name, name)
		}
		return fmt.Errorf("profile %q is a built-in profile and cannot be deleted", name)
	}

	if resolveErr != nil {
		return resolveErr
	}

	// Show what we're about to do
	fmt.Println(ui.RenderDetail("Delete profile", ui.Bold(name)))
	fmt.Println()
	ui.PrintWarning("This will permanently remove this profile.")
	fmt.Println()

	if !confirmProceed() {
		ui.PrintMuted("Cancelled.")
		return nil
	}

	// Delete the file
	if err := os.Remove(profilePath); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	// Clean breadcrumb entries referencing the deleted profile
	if err := breadcrumb.Remove(claudeupHome, name); err != nil {
		ui.PrintWarning(fmt.Sprintf("Breadcrumb cleanup failed: %v. 'profile diff' may reference the deleted profile until you apply another.", err))
	}

	ui.PrintSuccess(fmt.Sprintf("Deleted profile %q", name))

	return nil
}

func runProfileRestore(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Check if it's a built-in profile
	if !profile.IsEmbeddedProfile(name) {
		return fmt.Errorf("profile %q is not a built-in profile. Use 'claudeup profile delete %s' instead", name, name)
	}

	// Resolve the customization file on disk
	profilePath, err := resolveProfileArg(profilesDir, name)
	if err != nil {
		// Surface ambiguity errors so the user can disambiguate
		var ambigErr *profile.AmbiguousProfileError
		if errors.As(err, &ambigErr) {
			return err
		}
		return fmt.Errorf("profile %q has no customizations to restore from", name)
	}

	// Show what we're about to do
	fmt.Printf("Restore profile: %s\n", name)
	fmt.Println()
	fmt.Println("  This will remove your customizations and restore the original built-in version.")
	fmt.Println()

	if !confirmProceed() {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete the customization file
	if err := os.Remove(profilePath); err != nil {
		return fmt.Errorf("failed to restore profile: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Restored built-in profile %q", name))

	return nil
}

func runProfileRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]
	profilesDir := getProfilesDir()

	// "current" is reserved as a keyword for live status view
	if newName == "current" {
		return fmt.Errorf("'current' is a reserved name. Use a different profile name")
	}

	// Check if it's a built-in profile
	if profile.IsEmbeddedProfile(oldName) {
		return fmt.Errorf("profile %q is a built-in profile and cannot be renamed", oldName)
	}

	// Resolve source profile to exact path (handles nested profiles)
	oldPath, err := resolveProfileArg(profilesDir, oldName)
	if err != nil {
		return err
	}

	// Check if target profile already exists at root (Save writes to root)
	newPath := filepath.Join(profilesDir, newName+".json")
	if profileExistsAtRoot(profilesDir, newName) {
		if !config.YesFlag {
			return fmt.Errorf("profile %q already exists. Use -y to overwrite", newName)
		}
		// Remove the existing target at the known root path (Save writes to root)
		if err := os.Remove(newPath); err != nil {
			return fmt.Errorf("failed to remove existing profile: %w", err)
		}
	}

	// Load the profile from its resolved path
	p, err := profile.LoadFromPath(oldPath)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	// Update the name
	p.Name = newName

	// Save with new name (always writes to root profilesDir)
	if err := profile.Save(profilesDir, p); err != nil {
		return fmt.Errorf("failed to save renamed profile: %w", err)
	}

	// Remove old profile file
	if err := os.Remove(oldPath); err != nil {
		// Rollback: remove the new file we just created to avoid inconsistent state
		os.Remove(newPath)
		return fmt.Errorf("failed to remove old profile: %w", err)
	}

	// Update breadcrumb entries referencing the old name
	if err := breadcrumb.Rename(claudeupHome, oldName, newName); err != nil {
		ui.PrintWarning(fmt.Sprintf("Breadcrumb update failed: %v. 'profile diff' may reference the old name until you apply again.", err))
	}

	ui.PrintSuccess(fmt.Sprintf("Renamed profile %q to %q", oldName, newName))

	return nil
}
