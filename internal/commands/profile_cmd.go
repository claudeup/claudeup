// ABOUTME: Profile subcommands for managing Claude Code profiles
// ABOUTME: Implements list, use, save, and show operations
package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/claudeup/claudeup/v2/internal/backup"
	"github.com/claudeup/claudeup/v2/internal/claude"
	"github.com/claudeup/claudeup/v2/internal/config"
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/internal/ui"
	"github.com/spf13/cobra"
)

var (
	profileSaveDescription  string
	profileCloneFromFlag    string
	profileCloneDescription string
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
	Long: `List all available profiles with their status.

INDICATORS:
  *            Active profile with highest precedence (this is what Claude Code uses)
  ○            Active profile at lower precedence scope (overridden, not in effect)
  [scope]      Scope where profile is active (project/local/user)
  (customized) Built-in profile has been modified and saved locally
  (modified)   Active profile's effective configuration differs from saved definition

The (modified) indicator only appears on the highest precedence profile (*).
It compares the saved profile against the EFFECTIVE configuration, which
combines all scopes: user → project → local (later scopes override earlier).

SCOPE PRECEDENCE:
  project > local > user

Multiple profiles can be active at different scopes, but only the highest
precedence profile affects Claude Code's behavior. Lower precedence profiles
are overridden.

SCOPE FILTERING:
  --scope user      Show only the profile active at user scope
  --scope project   Show only the profile active at project scope
  --scope local     Show only the profile active at local scope

Note: --scope project and --scope local require the corresponding
.claude/settings.json or .claude/settings.local.json file to exist.

Example: If "base-tools" is active at user scope but "claudeup" is
active at project scope in ~/claudeup/, you'll see:
  * claudeup    [project] (modified)  ← This is what Claude Code uses
  ○ base-tools  [user]                ← Overridden, not in effect

Use 'claudeup profile status' to see how profiles differ from your current setup.`,
	Args: cobra.NoArgs,
	RunE: runProfileList,
}

var profileApplyCmd = &cobra.Command{
	Use:     "apply <name>",
	Aliases: []string{"use"},
	Short:   "Apply a profile to Claude Code",
	Long: `Apply a profile's configuration to your Claude Code installation.

SCOPES:
  --scope project  Apply to current project (creates .claudeup.json for team sharing)
  --scope local    Apply to current project, but not shared (personal overrides)
  --scope user     Apply globally to ~/.claude/ (default, affects all projects)

RESET MODE:
  --reset          Clear the target scope before applying the profile.
                   This replaces existing configuration instead of adding to it.
                   A backup is created automatically (skip with -y).

Precedence: local > project > user. Plugins from all scopes are active simultaneously.

For team projects, use --scope project to create a shareable configuration that
teammates can sync with 'claudeup profile sync'.

Shows a diff of changes before applying. Prompts for confirmation unless -y is used.`,
	Example: `  # Apply profile (adds to existing config)
  claudeup profile apply backend-stack

  # Replace existing config with profile
  claudeup profile apply backend-stack --reset

  # Replace without prompts (for scripting)
  claudeup profile apply backend-stack --reset -y

  # Set up a profile for your team (creates .claudeup.json)
  claudeup profile apply backend-stack --scope project

  # Force the post-apply setup wizard to run
  claudeup profile apply my-profile --setup`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileApply,
}

var profileSaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save current Claude Code state to a profile",
	Long: `Saves your current Claude Code configuration (plugins, MCP servers, marketplaces) to a profile.

SCOPES:
  --scope user     Save to ~/.claudeup/profiles/ (personal profile)
  --scope project  Save to .claudeup/profiles/ (shareable with team)

If --scope is not specified, uses context-aware default:
  - project scope if .claudeup.json exists in current directory
  - user scope otherwise

If no name is given, saves to the currently active profile.
If the profile exists, prompts for confirmation unless -y is used.`,
	Example: `  # Save to user scope (default in non-project directories)
  claudeup profile save my-tools

  # Save to project scope (for team sharing)
  claudeup profile save team-config --scope project

  # Explicitly save to user scope
  claudeup profile save my-tools --scope user`,
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
With -y flag, uses the currently active profile as the source.`,
	Example: `  # Clone from specific profile
  claudeup profile clone new-profile --from existing-profile

  # Clone from active profile
  claudeup profile clone new-profile -y

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
	Use:   "status [name]",
	Short: "Show differences between profile and current state by scope",
	Long: `Display how a profile differs from current state, broken down by scope.

Shows:
  - Which scope the profile is active in
  - Differences at each scope (user, project, local)
  - Effective configuration (all scopes combined)

If no name is given, uses the currently active profile.

This helps distinguish between:
  - Profile modifications at the active scope (profile truly changed)
  - Scope layering (user/project/local scopes adding to the profile)`,
	Example: `  # Show status for active profile
  claudeup profile status

  # Show status for specific profile
  claudeup profile status backend-stack`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileStatus,
}

var profileDiffCmd = &cobra.Command{
	Use:   "diff <name>",
	Short: "Compare customized built-in profile to its original",
	Long: `Compare a customized built-in profile to its embedded original.

This command shows what you've changed from the original built-in profile.
Use this to see your customizations before restoring or sharing profiles.

Only works with built-in profiles that have been customized (saved to disk).`,
	Example: `  # Show changes made to the default profile
  claudeup profile diff default

  # Show changes made to the frontend profile
  claudeup profile diff frontend`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileDiff,
}

var profileSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest a profile for the current directory",
	Args:  cobra.NoArgs,
	RunE:  runProfileSuggest,
}

var profileCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the currently active profile",
	Args:  cobra.NoArgs,
	RunE:  runProfileCurrent,
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
cannot be renamed.

If the profile being renamed is currently active, the active profile config
will be updated to point to the new name.`,
	Args: cobra.ExactArgs(2),
	RunE: runProfileRename,
}

var profileSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync plugins from .claudeup.json",
	Long: `Install plugins defined in the project's .claudeup.json file.

This command is intended for team members who clone a repository
that has been configured with 'claudeup profile apply --scope project'.

MCP servers from .mcp.json are loaded automatically by Claude Code.
This command syncs plugins that require explicit installation.`,
	Args: cobra.NoArgs,
	RunE: runProfileSync,
}

// Flags for profile apply command
var (
	profileApplySetup         bool
	profileApplyNoInteractive bool
	profileApplyForce         bool
	profileApplyScope         string
	profileApplyReinstall     bool
	profileApplyNoProgress    bool
	profileApplyReset         bool
)

// Flags for profile sync command
var profileSyncDryRun bool

// Flags for profile save command
var profileSaveScope string

// Flags for profile list command
var profileListScope string

// Flags for profile clean command
var profileCleanScope string

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

Use this to clean up drift detected by 'claudeup status' or 'claudeup doctor'.`,
	Example: `  # Remove plugin from project scope
  claudeup profile clean --scope project nextjs-vercel-pro@claude-code-templates

  # Remove plugin from local scope
  claudeup profile clean --scope local my-plugin@marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileClean,
}

func runProfileClean(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	// Validate scope flag is provided
	if profileCleanScope == "" {
		return fmt.Errorf("--scope flag is required (project or local)")
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
	// Note: Plugins are defined in profiles, not in .claudeup.json, so we only
	// need to disable them in settings. To fully remove a plugin from a profile,
	// edit the profile definition itself.
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

	// Check if plugin is also in the saved profile definition
	var profileName string
	if scope == profile.ScopeProject && profile.ConfigExistsForScope(projectDir, scope) {
		cfg, err := profile.LoadConfigForScope(projectDir, scope)
		if err == nil && cfg != nil {
			profileName = cfg.Profile
		}
	}
	if profileName != "" {
		profilesDir := getProfilesDir()
		savedProfile, err := loadProfileWithFallback(profilesDir, profileName)
		if err == nil {
			// Check if plugin is in the profile
			pluginInProfile := false
			for _, p := range savedProfile.Plugins {
				if p == pluginName {
					pluginInProfile = true
					break
				}
			}

			if pluginInProfile {
				fmt.Println()
				ui.PrintWarning(fmt.Sprintf("Plugin %s is also in your saved profile %q", pluginName, profileName))
				fmt.Println("  If not removed from the profile, it will be reinstalled when you run:")
				fmt.Printf("    %s\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s --reinstall", profileName)))
				fmt.Println()

				// Ask if user wants to remove it from the profile too
				confirm, err := ui.ConfirmYesNo("Remove from saved profile too?")
				if err != nil {
					return err
				}

				if confirm {
					// Remove plugin from profile
					newPlugins := []string{}
					for _, p := range savedProfile.Plugins {
						if p != pluginName {
							newPlugins = append(newPlugins, p)
						}
					}
					savedProfile.Plugins = newPlugins

					// Save updated profile (only save to disk if it's a user profile or customized built-in)
					if !profile.IsEmbeddedProfile(profileName) || profileExists(profilesDir, profileName) {
						if err := profile.Save(profilesDir, savedProfile); err != nil {
							return fmt.Errorf("failed to update profile: %w", err)
						}
						ui.PrintSuccess(fmt.Sprintf("Removed %s from profile %q", pluginName, profileName))
					} else {
						ui.PrintWarning("Cannot modify embedded profile. Create a custom version with:")
						fmt.Printf("  %s\n", ui.Bold(fmt.Sprintf("claudeup profile save %s", profileName)))
					}
				} else {
					ui.PrintMuted("Plugin remains in profile definition.")
				}
			}
		}
	}

	return nil
}

// profileExists checks if a profile file exists on disk
func profileExists(profilesDir, name string) bool {
	path := filepath.Join(profilesDir, name+".json")
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileApplyCmd)
	profileCmd.AddCommand(profileSaveCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileCloneCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileStatusCmd)
	profileCmd.AddCommand(profileDiffCmd)
	profileCmd.AddCommand(profileSuggestCmd)
	profileCmd.AddCommand(profileCurrentCmd)
	profileCmd.AddCommand(profileResetCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileRestoreCmd)
	profileCmd.AddCommand(profileRenameCmd)
	profileCmd.AddCommand(profileSyncCmd)
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
	profileSaveCmd.Flags().StringVar(&profileSaveScope, "scope", "", "Where to save: 'user' or 'project' (default: project if .claudeup.json exists, otherwise user)")

	// Add flags to profile apply command
	profileApplyCmd.Flags().BoolVar(&profileApplySetup, "setup", false, "Force post-apply setup wizard to run")
	profileApplyCmd.Flags().BoolVar(&profileApplyNoInteractive, "no-interactive", false, "Skip post-apply setup wizard (for CI/scripting)")
	profileApplyCmd.Flags().BoolVarP(&profileApplyForce, "force", "f", false, "Force reapply even with unsaved changes")
	profileApplyCmd.Flags().StringVar(&profileApplyScope, "scope", "", "Apply scope: user, project, or local (default: user, or project if .claudeup.json exists)")
	profileApplyCmd.Flags().BoolVar(&profileApplyReinstall, "reinstall", false, "Force reinstall all plugins and marketplaces")
	profileApplyCmd.Flags().BoolVar(&profileApplyNoProgress, "no-progress", false, "Disable progress display (for CI/scripting)")
	profileApplyCmd.Flags().BoolVar(&profileApplyReset, "reset", false, "Clear target scope before applying profile")

	// Add flags to profile clean command
	profileCleanCmd.Flags().StringVar(&profileCleanScope, "scope", "", "Config scope to clean: project or local (required)")
	profileCleanCmd.MarkFlagRequired("scope")

	// Add flags to profile sync command
	profileSyncCmd.Flags().BoolVar(&profileSyncDryRun, "dry-run", false, "Show what would be synced without making changes")

	// Add flags to profile list command
	profileListCmd.Flags().StringVar(&profileListScope, "scope", "", "Show only the profile active at specified scope: user, project, local")
}

func runProfileList(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()
	cwd, _ := os.Getwd()

	// Validate scope value if provided
	validScopes := map[string]bool{
		"":        true,
		"user":    true,
		"project": true,
		"local":   true,
	}
	if !validScopes[profileListScope] {
		return fmt.Errorf("invalid scope %q: must be one of user, project, local", profileListScope)
	}

	// Check for scope-specific file requirements
	projectSettingsPath := filepath.Join(cwd, ".claude", "settings.json")
	localSettingsPath := filepath.Join(cwd, ".claude", "settings.local.json")

	if profileListScope == "project" {
		if _, err := os.Stat(projectSettingsPath); os.IsNotExist(err) {
			ui.PrintWarning("No .claude/settings.json found in current directory.")
			fmt.Printf("  %s Use --scope project inside a project with Claude settings.\n", ui.Muted(ui.SymbolArrow))
			return nil
		}
	}
	if profileListScope == "local" {
		if _, err := os.Stat(localSettingsPath); os.IsNotExist(err) {
			ui.PrintWarning("No .claude/settings.local.json found in current directory.")
			fmt.Printf("  %s Use --scope local inside a project with local Claude settings.\n", ui.Muted(ui.SymbolArrow))
			return nil
		}
	}

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

	// Get active profiles from all scopes (project, local, user)
	allActiveProfiles := getAllActiveProfiles(cwd)

	// Create maps for quick lookup
	activeProfileNames := make(map[string]bool)
	activeProfileByScope := make(map[string]string) // scope -> profile name
	for _, ap := range allActiveProfiles {
		activeProfileNames[ap.Name] = true
		activeProfileByScope[ap.Scope] = ap.Name
	}

	// The highest precedence active profile (project > local > user)
	activeProfile, _ := getActiveProfile(cwd)

	// Filter helper function
	shouldShowProfile := func(profileName string) bool {
		switch profileListScope {
		case "":
			return true
		case "user":
			return activeProfileByScope["user"] == profileName
		case "project":
			return activeProfileByScope["project"] == profileName
		case "local":
			return activeProfileByScope["local"] == profileName
		default:
			return true
		}
	}

	// Filter embedded profiles
	var filteredEmbedded []*profile.Profile
	for _, p := range embeddedProfiles {
		if shouldShowProfile(p.Name) {
			filteredEmbedded = append(filteredEmbedded, p)
		}
	}

	// Filter custom profiles (exclude those that shadow built-ins)
	var customProfiles []*profile.ProfileWithSource
	for _, p := range allProfiles {
		if !profile.IsEmbeddedProfile(p.Name) && shouldShowProfile(p.Name) {
			customProfiles = append(customProfiles, p)
		}
	}

	// Check if we have any profiles to show after filtering
	if len(filteredEmbedded) == 0 && len(customProfiles) == 0 {
		if profileListScope != "" {
			ui.PrintInfo(fmt.Sprintf("No profile is active at %s scope.", profileListScope))
		} else {
			ui.PrintInfo("No profiles found.")
			fmt.Printf("  %s Create one with: claudeup profile save <name>\n", ui.Muted(ui.SymbolArrow))
		}
		return nil
	}

	// Helper to get profile marker (active indicator)
	getProfileMarker := func(profileName string) string {
		for _, ap := range allActiveProfiles {
			if ap.Name == profileName {
				if ap.Name == activeProfile {
					return ui.Success("* ")
				}
				return ui.Muted("○ ")
			}
		}
		return "  "
	}

	// Show scope info if filtering
	if profileListScope != "" {
		fmt.Printf("%s Showing profile active at: %s scope\n", ui.Muted(ui.SymbolArrow), ui.Bold(profileListScope))
		fmt.Println()
	}

	// Show built-in profiles section
	if len(filteredEmbedded) > 0 {
		fmt.Println(ui.Bold("Built-in profiles"))
		fmt.Println()
		for _, p := range filteredEmbedded {
			marker := getProfileMarker(p.Name)
			desc := p.Description

			// If shadowed on disk, check if content actually differs
			if profileOnDisk[p.Name] {
				// Find the disk version
				for _, dp := range allProfiles {
					if dp.Name == p.Name {
						desc = dp.Description
						// Only show (customized) if content actually differs
						if !p.Equal(dp.Profile) {
							desc += " " + ui.Muted("(customized)")
						}
						break
					}
				}
			}

			if desc == "" {
				desc = ui.Muted("(no description)")
			}
			fmt.Printf("%s%-20s %s\n", marker, p.Name, desc)
		}
		fmt.Println()
	}

	// Show user profiles section
	if len(customProfiles) > 0 {
		fmt.Println(ui.Bold("Your profiles"))
		fmt.Println()
		for _, p := range customProfiles {
			marker := getProfileMarker(p.Name)
			desc := p.Description
			if desc == "" {
				desc = ui.Muted("(no description)")
			}
			fmt.Printf("%s%-20s %s\n", marker, p.Name, desc)
		}
		fmt.Println()
	}

	// Warn if user has a profile named "current" (now reserved)
	if profileOnDisk["current"] && profileListScope == "" {
		ui.PrintWarning("Profile \"current\" uses a reserved name. Rename it with:")
		fmt.Println("  claudeup profile rename current <new-name>")
		fmt.Println()
	}

	fmt.Printf("%s Use 'claudeup profile show <name>' for details\n", ui.Muted(ui.SymbolArrow))
	fmt.Printf("%s Use 'claudeup profile apply <name>' to apply a profile\n", ui.Muted(ui.SymbolArrow))

	return nil
}

func runProfileApply(cmd *cobra.Command, args []string) error {
	name := args[0]

	// "current" is reserved as a keyword for the active profile
	if name == "current" {
		return fmt.Errorf("'current' is a reserved name. Use 'claudeup profile show current' to see the active profile")
	}

	// Determine scope from flag or auto-detect
	cwd, _ := os.Getwd()
	var scope profile.Scope

	if profileApplyScope != "" {
		// Explicit scope from flag
		var err error
		scope, err = profile.ParseScope(profileApplyScope)
		if err != nil {
			return err
		}
	} else {
		// Auto-detect: if project files exist, use project scope
		if profile.ProjectConfigExists(cwd) || profile.MCPJSONExists(cwd) {
			scope = profile.ScopeProject
			ui.PrintInfo("Detected existing project configuration, using project scope")
			fmt.Println()
		} else {
			scope = profile.ScopeUser
		}
	}

	// Handle --reset flag: clear scope before applying
	if profileApplyReset {
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

	return applyProfileWithScope(name, scope)
}

// applyProfileWithScope applies a profile at the specified scope.
// This is the core implementation shared by runProfileApply and runProfileCreate.
func applyProfileWithScope(name string, scope profile.Scope) error {
	profilesDir := getProfilesDir()
	cwd, _ := os.Getwd()

	// Warn if we're about to overwrite an existing .claudeup.json with a different profile.
	// This only applies to project scope - user scope doesn't touch project config files.
	if scope == profile.ScopeProject && profile.ProjectConfigExists(cwd) {
		existingConfig, err := profile.LoadProjectConfig(cwd)
		if err != nil {
			// Log error but continue - malformed config will be overwritten
			ui.PrintWarning(fmt.Sprintf("Could not read existing project config: %v", err))
		} else if existingConfig.Profile != "" && existingConfig.Profile != name {
			ui.PrintWarning(fmt.Sprintf("Project is already configured with profile %q", existingConfig.Profile))
			fmt.Printf("  Applying %q will overwrite the existing configuration.\n", name)
			fmt.Println()
			if !config.YesFlag {
				if !confirmProceed() {
					ui.PrintMuted("Cancelled.")
					return nil
				}
			}
		}
	}

	// Load the profile (try disk first, then embedded)
	p, err := loadProfileWithFallback(profilesDir, name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	// In a declarative system, reapplying a profile should always be allowed
	// It simply syncs the current state to match the profile definition
	cfg, _ := config.Load()

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
	diff, err := profile.ComputeDiffWithScope(p, claudeDir, claudeJSONPath, profile.DiffOptions{
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

	shouldRunHook := profile.ShouldRunHook(p, claudeDir, claudeJSONPath, hookOpts)

	// If no changes and no hook to run, we're done
	if !hasDiffChanges(diff) && !shouldRunHook {
		// Update active profile in config even when no changes needed
		cfg, err = config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}
		cfg.Preferences.ActiveProfile = name
		if err := config.Save(cfg); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not save active profile: %v", err))
		}

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
	if hasDiffChanges(diff) {
		fmt.Println(ui.RenderDetail("Profile", ui.Bold(name)))
		fmt.Println()
		showDiff(diff)
		fmt.Println()

		// Skip confirmation if using --force flag
		if !profileApplyForce && !confirmProceed() {
			ui.PrintMuted("Cancelled.")
			return nil
		}
	} else {
		// No changes, but hook needs to run
		fmt.Println(ui.RenderDetail("Profile", ui.Bold(name)))
		fmt.Println()
		ui.PrintInfo("No configuration changes needed.")
		if profileApplySetup {
			fmt.Println("Running setup wizard...")
		}
		fmt.Println()
	}

	// Apply (hook decision was already made above)
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Applying profile (%s scope)...", scope))

	chain := buildSecretChain()

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

	result, err := profile.ApplyWithOptions(p, claudeDir, claudeJSONPath, chain, opts)
	if err != nil {
		return fmt.Errorf("failed to apply profile: %w", err)
	}

	showApplyResults(result)

	// Update active profile in config (only for user scope)
	if scope == profile.ScopeUser {
		cfg, err = config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}
		cfg.Preferences.ActiveProfile = name
		if err := config.Save(cfg); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not save active profile: %v", err))
		}

		// Silently clean up stale plugin entries (only for user scope)
		cleanupStalePlugins(claudeDir)
	}

	fmt.Println()
	ui.PrintSuccess("Profile applied!")

	// Scope-specific post-apply messages
	if scope == profile.ScopeProject {
		fmt.Println()
		ui.PrintInfo("Project files created:")

		var filesToAdd []string
		if profile.MCPJSONExists(cwd) {
			fmt.Printf("  %s %s (MCP servers - Claude auto-loads)\n", ui.Success(ui.SymbolSuccess), profile.MCPConfigFile)
			filesToAdd = append(filesToAdd, profile.MCPConfigFile)
		}
		if profile.ProjectConfigExists(cwd) {
			fmt.Printf("  %s %s (plugins manifest)\n", ui.Success(ui.SymbolSuccess), profile.ProjectConfigFile)
			filesToAdd = append(filesToAdd, profile.ProjectConfigFile)
		}

		if len(filesToAdd) > 0 {
			fmt.Println()
			fmt.Printf("%s Consider adding these to git:\n", ui.Muted(ui.SymbolArrow))
			fmt.Printf("  git add %s\n", strings.Join(filesToAdd, " "))
		}
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
	for name, plugin := range plugins.GetAllPlugins() {
		if !plugin.PathExists() {
			if plugins.DisablePlugin(name) {
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
	// Determine scope from flag or context-aware default
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	var scope string
	if profileSaveScope != "" {
		// Explicit scope from flag
		if profileSaveScope != "user" && profileSaveScope != "project" {
			return fmt.Errorf("invalid scope %q: must be 'user' or 'project'", profileSaveScope)
		}
		scope = profileSaveScope
	} else {
		// Context-aware default: project if .claudeup.json exists, otherwise user
		if profile.ProjectConfigExists(cwd) {
			scope = "project"
		} else {
			scope = "user"
		}
	}

	// Determine profiles directory based on scope
	var profilesDir string
	if scope == "project" {
		profilesDir = profile.ProjectProfilesDir(cwd)
	} else {
		profilesDir = getProfilesDir()
	}

	// Determine profile name
	var name string
	isActiveProfile := false
	if len(args) > 0 {
		name = args[0]
		// "current" is reserved as a keyword for the active profile
		// Only check when explicitly passed as argument (not when resolved from active profile)
		if name == "current" {
			return fmt.Errorf("'current' is a reserved name. Use a different profile name")
		}
	} else {
		// Use active profile name
		// Error ignored: missing/corrupt config is handled same as no active profile
		cfg, _ := config.Load()
		if cfg == nil || cfg.Preferences.ActiveProfile == "" {
			return fmt.Errorf("no profile name given and no active profile set. Use 'claudeup profile save <name>' or 'claudeup profile apply <name>' first")
		}
		name = cfg.Preferences.ActiveProfile
		isActiveProfile = true
		ui.PrintInfo(fmt.Sprintf("Saving to active profile: %s", name))
	}

	// Check if profile already exists (only prompt if explicitly named, not when saving to active profile)
	existingPath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(existingPath); err == nil {
		if !isActiveProfile && !config.YesFlag {
			fmt.Printf("%s Profile %q already exists at %s scope. ", ui.Warning(ui.SymbolWarning), name, scope)
			choice := promptChoice("Overwrite?", "n")
			if choice != "y" && choice != "yes" {
				ui.PrintMuted("Cancelled.")
				return nil
			}
		}
	}

	// Use the global claudeDir from root.go (set via --claude-dir flag)
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create snapshot
	p, err := profile.Snapshot(name, claudeDir, claudeJSONPath)
	if err != nil {
		return fmt.Errorf("failed to snapshot current state: %w", err)
	}

	// Handle description: preserve existing custom descriptions, update generic ones
	if profileSaveDescription != "" {
		// User provided explicit description via flag
		p.Description = profileSaveDescription
	} else {
		// Check if existing profile has custom description
		existingProfile, err := profile.Load(profilesDir, name)
		if err == nil {
			// Profile exists - preserve custom descriptions
			if existingProfile.Description != "" && existingProfile.Description != "Snapshot of current Claude Code configuration" {
				// Has custom description, preserve it
				p.Description = existingProfile.Description
			}
			// Otherwise use auto-generated description (already set by Snapshot)
		}
		// If profile doesn't exist, use auto-generated description (already set by Snapshot)
	}

	// Save
	if err := profile.Save(profilesDir, p); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	// Only update active profile in config for user-scope saves
	if scope == "user" {
		cfg, err := config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}
		cfg.Preferences.ActiveProfile = name
		if err := config.Save(cfg); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not save active profile: %v", err))
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Saved profile %q (%s scope)", name, scope))
	fmt.Println()
	fmt.Println(ui.Indent(ui.RenderDetail("MCP Servers", fmt.Sprintf("%d", len(p.MCPServers))), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", fmt.Sprintf("%d", len(p.Marketplaces))), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("Plugins", fmt.Sprintf("%d", len(p.Plugins))), 1))

	// Show commit hint for project scope
	if scope == "project" {
		fmt.Println()
		fmt.Printf("%s Consider adding to git: %s\n", ui.Muted(ui.SymbolArrow), ui.Bold(".claudeup/profiles/"))
	}

	return nil
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Handle "current" as a special keyword for the active profile
	if name == "current" {
		// Error ignored: missing/corrupt config is handled same as no active profile
		cfg, _ := config.Load()
		if cfg == nil || cfg.Preferences.ActiveProfile == "" {
			return fmt.Errorf("no active profile set. Use 'claudeup profile apply <name>' to apply a profile")
		}
		name = cfg.Preferences.ActiveProfile
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
	fmt.Println()

	if len(p.MCPServers) > 0 {
		fmt.Println("MCP Servers:")
		for _, m := range p.MCPServers {
			fmt.Printf("  - %s (%s)\n", m.Name, m.Command)
			if len(m.Secrets) > 0 {
				for envVar := range m.Secrets {
					fmt.Printf("      requires: %s\n", envVar)
				}
			}
		}
		fmt.Println()
	}

	if len(p.Marketplaces) > 0 {
		fmt.Println("Marketplaces:")
		for _, m := range p.Marketplaces {
			fmt.Printf("  - %s\n", m.DisplayName())
		}
		fmt.Println()
	}

	if len(p.Plugins) > 0 {
		fmt.Println("Plugins:")
		for _, plug := range p.Plugins {
			fmt.Printf("  - %s\n", plug)
		}
		fmt.Println()
	}

	return nil
}

func runProfileStatus(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()
	cwd, _ := os.Getwd()
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Determine profile name
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use active profile
		activeProfile, _ := getActiveProfile(cwd)
		if activeProfile == "" {
			return fmt.Errorf("no active profile set. Specify a profile name or use 'claudeup profile apply <name>' first")
		}
		name = activeProfile
	}

	// Load the profile
	savedProfile, err := loadProfileWithFallback(profilesDir, name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	// Get all active profiles to determine which scope this profile is active in
	allActiveProfiles := getAllActiveProfiles(cwd)
	var activeScope string
	for _, ap := range allActiveProfiles {
		if ap.Name == name {
			activeScope = ap.Scope
			break
		}
	}

	// Header
	fmt.Println(ui.RenderDetail("Profile", ui.Bold(name)))
	if activeScope != "" {
		fmt.Printf("  %s\n", ui.Info(fmt.Sprintf("[active at %s scope]", activeScope)))
	}
	fmt.Println()

	// Compare the active scope against the profile
	if activeScope != "" {
		diff, err := profile.CompareWithScope(savedProfile, claudeDir, claudeJSONPath, cwd, activeScope)
		if err == nil {
			scopeLabel := fmt.Sprintf("%s scope (%s)", activeScope, getScopeFile(activeScope))
			if !diff.HasSignificantChanges() {
				fmt.Printf("%s %s\n", ui.Success("✓"), ui.Bold(scopeLabel))
				fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("Matches saved profile (%d plugins)", len(savedProfile.Plugins))))
			} else {
				fmt.Printf("%s %s\n", ui.Warning("○"), ui.Bold(scopeLabel))
				if len(diff.PluginsAdded) > 0 {
					fmt.Printf("  %s\n", ui.Success(fmt.Sprintf("%d additional plugin%s:", len(diff.PluginsAdded), pluralS(len(diff.PluginsAdded)))))
					for _, p := range diff.PluginsAdded {
						fmt.Printf("    + %s\n", p)
					}
				}
				if len(diff.PluginsRemoved) > 0 {
					fmt.Printf("  %s\n", ui.Warning(fmt.Sprintf("%d missing plugin%s:", len(diff.PluginsRemoved), pluralS(len(diff.PluginsRemoved)))))
					for _, p := range diff.PluginsRemoved {
						fmt.Printf("    - %s\n", p)
					}
				}

				// Show actionable guidance based on drift type
				fmt.Println()
				hasExtra := len(diff.PluginsAdded) > 0
				hasMissing := len(diff.PluginsRemoved) > 0

				if hasExtra && hasMissing {
					// Both types of drift
					fmt.Printf("  %s Reset to profile (removes extra, installs missing):\n", ui.Muted(ui.SymbolArrow))
					fmt.Printf("    %s\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s --scope %s --reset", name, activeScope)))
				} else if hasExtra {
					// Only extra plugins
					fmt.Printf("  %s Remove extra plugin%s:\n", ui.Muted(ui.SymbolArrow), pluralS(len(diff.PluginsAdded)))
					fmt.Printf("    %s\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s --scope %s --reset", name, activeScope)))
					if len(diff.PluginsAdded) == 1 {
						fmt.Printf("  %s Or remove specific plugin:\n", ui.Muted(ui.SymbolArrow))
						fmt.Printf("    %s\n", ui.Bold(fmt.Sprintf("claudeup profile clean --scope %s %s", activeScope, diff.PluginsAdded[0])))
					} else {
						fmt.Printf("  %s Or remove specific plugins:\n", ui.Muted(ui.SymbolArrow))
						fmt.Printf("    %s\n", ui.Bold(fmt.Sprintf("claudeup profile clean --scope %s <plugin>", activeScope)))
					}
				} else if hasMissing {
					// Only missing plugins
					if activeScope == "project" {
						fmt.Printf("  %s Install missing plugin%s:\n", ui.Muted(ui.SymbolArrow), pluralS(len(diff.PluginsRemoved)))
						fmt.Printf("    %s\n", ui.Bold("claudeup profile sync"))
					} else {
						fmt.Printf("  %s Install missing plugin%s:\n", ui.Muted(ui.SymbolArrow), pluralS(len(diff.PluginsRemoved)))
						fmt.Printf("    %s\n", ui.Bold(fmt.Sprintf("claudeup profile apply %s --scope %s", name, activeScope)))
					}
				}
			}
			fmt.Println()
		}
	}

	// For other scopes, show what they're adding on top
	scopes := []string{"user", "project", "local"}
	for _, scope := range scopes {
		if scope == activeScope {
			continue // Already showed this above
		}

		// Get snapshot of just this scope
		scopeSnapshot, err := profile.SnapshotWithScope("", claudeDir, claudeJSONPath, profile.SnapshotOptions{
			Scope:      scope,
			ProjectDir: cwd,
		})
		if err != nil {
			continue
		}

		scopeLabel := fmt.Sprintf("%s scope (%s)", scope, getScopeFile(scope))

		// Show what this scope adds
		if len(scopeSnapshot.Plugins) > 0 {
			fmt.Printf("%s %s\n", ui.Info("○"), scopeLabel)
			fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("%d plugin%s layered on top:", len(scopeSnapshot.Plugins), pluralS(len(scopeSnapshot.Plugins)))))
			for _, p := range scopeSnapshot.Plugins {
				fmt.Printf("    + %s\n", p)
			}
		} else {
			fmt.Printf("%s %s\n", ui.Success("✓"), scopeLabel)
			fmt.Printf("  %s\n", ui.Muted("No additional plugins"))
		}

		fmt.Println()
	}

	// Show effective configuration (all scopes combined)
	combinedDiff, err := profile.CompareWithCombinedScopes(savedProfile, claudeDir, claudeJSONPath, cwd)
	if err != nil {
		return fmt.Errorf("failed to compare combined scopes: %w", err)
	}

	fmt.Println(ui.RenderDetail("Effective configuration", "(all scopes combined)"))
	if !combinedDiff.HasSignificantChanges() {
		fmt.Printf("  %s\n", ui.Success("✓ Matches saved profile"))
	} else {
		totalPlugins := len(savedProfile.Plugins)
		effectivePlugins := totalPlugins + len(combinedDiff.PluginsAdded) - len(combinedDiff.PluginsRemoved)
		fmt.Printf("  %d plugins total (%d from profile", effectivePlugins, totalPlugins)
		if len(combinedDiff.PluginsAdded) > 0 {
			fmt.Printf(" + %d from other scope%s", len(combinedDiff.PluginsAdded), pluralS(len(combinedDiff.PluginsAdded)))
		}
		if len(combinedDiff.PluginsRemoved) > 0 {
			fmt.Printf(" - %d missing", len(combinedDiff.PluginsRemoved))
		}
		fmt.Println(")")
	}
	fmt.Println()

	// Check for config drift (orphaned tracking entries)
	plugins, err := claude.LoadPlugins(claudeDir)
	if err == nil {
		profilesDir := getProfilesDir()
		configDrift, err := profile.DetectConfigDrift(profilesDir, claudeDir, cwd, plugins)
		if err == nil && len(configDrift) > 0 {
			// Group by scope
			driftByScope := make(map[profile.Scope][]string)
			for _, d := range configDrift {
				driftByScope[d.Scope] = append(driftByScope[d.Scope], d.PluginName)
			}

			// Check which drifted plugins are in the saved profile
			pluginsInProfile := make(map[string]bool)
			if savedProfile != nil {
				for _, p := range savedProfile.Plugins {
					pluginsInProfile[p] = true
				}
			}

			fmt.Printf("%s %s\n", ui.Warning("⚠"), ui.Bold("Config Drift Detected"))
			fmt.Println()

			// Show project scope drift
			if projectDrift, ok := driftByScope[profile.ScopeProject]; ok {
				fmt.Printf("  %s %d orphaned entr%s:\n", ui.Muted(".claudeup.json"), len(projectDrift), pluralYIES(len(projectDrift)))
				for _, pluginName := range projectDrift {
					suffix := ""
					if pluginsInProfile[pluginName] {
						suffix = ui.Muted(" (also in profile)")
					}
					fmt.Printf("    - %s%s\n", pluginName, suffix)
				}
				fmt.Println()
				for _, pluginName := range projectDrift {
					fmt.Printf("  %s Remove from config and profile: %s\n",
						ui.Muted(ui.SymbolArrow), ui.Bold(fmt.Sprintf("claudeup profile clean --scope project %s", pluginName)))
				}
				fmt.Printf("  %s Or sync from profile: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold("claudeup profile sync"))
				fmt.Println()
			}

			// Show local scope drift
			if localDrift, ok := driftByScope[profile.ScopeLocal]; ok {
				fmt.Printf("  %s %d orphaned entr%s:\n", ui.Muted(".claude/settings.local.json"), len(localDrift), pluralYIES(len(localDrift)))
				for _, pluginName := range localDrift {
					suffix := ""
					if pluginsInProfile[pluginName] {
						suffix = ui.Muted(" (also in profile)")
					}
					fmt.Printf("    - %s%s\n", pluginName, suffix)
				}
				fmt.Println()
				for _, pluginName := range localDrift {
					fmt.Printf("  %s Remove from config and profile: %s\n",
						ui.Muted(ui.SymbolArrow), ui.Bold(fmt.Sprintf("claudeup profile clean --scope local %s", pluginName)))
				}
				fmt.Printf("  %s Or reinstall if available: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold(fmt.Sprintf("claudeup profile apply %s --reinstall", name)))
				fmt.Println()
			}
		}
	}

	return nil
}

// getScopeFile returns the settings file name for a scope
func getScopeFile(scope string) string {
	switch scope {
	case "user":
		return "~/.claude/settings.json"
	case "project":
		return ".claude/settings.json"
	case "local":
		return ".claude/settings.local.json"
	default:
		return ""
	}
}

// pluralS returns "s" if count != 1, otherwise empty string
func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func pluralYIES(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}

func hasDiffChanges(diff *profile.Diff) bool {
	return len(diff.PluginsToRemove) > 0 ||
		len(diff.PluginsToInstall) > 0 ||
		len(diff.MCPToRemove) > 0 ||
		len(diff.MCPToInstall) > 0 ||
		len(diff.MarketplacesToAdd) > 0 ||
		len(diff.MarketplacesToRemove) > 0
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
	name := args[0]
	profilesDir := getProfilesDir()

	// Check if the profile is a built-in
	if !profile.IsEmbeddedProfile(name) {
		return fmt.Errorf("'%s' is not a built-in profile. Use 'profile diff' only with built-in profiles", name)
	}

	// Get the embedded original
	embedded, err := profile.GetEmbeddedProfile(name)
	if err != nil {
		return fmt.Errorf("failed to load embedded profile: %w", err)
	}

	// Check if there's a customized version on disk
	customizedPath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(customizedPath); os.IsNotExist(err) {
		// No customized version - no differences
		fmt.Printf("Profile '%s' has not been customized.\n", name)
		fmt.Println("No differences from the built-in version.")
		return nil
	}

	// Load the customized version
	customized, err := profile.Load(profilesDir, name)
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
		fmt.Printf("  %s description: %q → %q\n",
			ui.Warning("~"),
			embedded.Description,
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
	profiles, err := profile.List(profilesDir)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles available.")
		fmt.Println("Create one with: claudeup profile save <name>")
		return nil
	}

	// Find matching profiles
	suggested := profile.SuggestProfile(cwd, profiles)

	if suggested == nil {
		fmt.Println("No profile matches the current directory.")
		fmt.Println()
		fmt.Println("Available profiles:")
		for _, p := range profiles {
			fmt.Printf("  - %s\n", p.Name)
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
// falling back to embedded profiles if not found
func loadProfileWithFallback(profilesDir, name string) (*profile.Profile, error) {
	// Try disk first
	p, err := profile.Load(profilesDir, name)
	if err == nil {
		return p, nil
	}

	// Fall back to embedded profiles
	return profile.GetEmbeddedProfile(name)
}

// getAllProfiles returns all available profiles (user + embedded), with user profiles taking precedence
func getAllProfiles(profilesDir string) ([]*profile.Profile, error) {
	// Load user profiles
	userProfiles, err := profile.List(profilesDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list user profiles: %w", err)
	}

	// Track user profile names
	userNames := make(map[string]bool)
	for _, p := range userProfiles {
		userNames[p.Name] = true
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
		if err := profile.ValidateName(name); err != nil {
			return err
		}

		existingPath := filepath.Join(profilesDir, name+".json")
		if _, err := os.Stat(existingPath); err == nil {
			return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
		}

		newProfile, err := profile.CreateFromFlags(name, profileCreateDescription, profileCreateMarketplaces, profileCreatePlugins)
		if err != nil {
			return err
		}

		if err := profile.Save(profilesDir, newProfile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Printf("Profile %q created successfully.\n\n", name)
		fmt.Printf("  Marketplaces: %d\n", len(newProfile.Marketplaces))
		fmt.Printf("  Plugins: %d\n", len(newProfile.Plugins))
		fmt.Printf("\nRun 'claudeup profile apply %s' to use it.\n", name)
		return nil
	}

	// File/stdin mode
	if hasFileInput {
		name := args[0]
		if err := profile.ValidateName(name); err != nil {
			return err
		}

		existingPath := filepath.Join(profilesDir, name+".json")
		if _, err := os.Stat(existingPath); err == nil {
			return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
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

		if err := profile.Save(profilesDir, newProfile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Printf("Profile %q created successfully.\n\n", name)
		fmt.Printf("  Marketplaces: %d\n", len(newProfile.Marketplaces))
		fmt.Printf("  Plugins: %d\n", len(newProfile.Plugins))
		fmt.Printf("\nRun 'claudeup profile apply %s' to use it.\n", name)
		return nil
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

	// Check if profile already exists
	existingPath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(existingPath); err == nil {
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
		// Note: User scope intentionally skips the project overwrite warning since
		// it doesn't modify .claudeup.json.
		if err := applyProfileWithScope(name, profile.ScopeUser); err != nil {
			return err
		}

		// If there's an existing project config, inform the user about scope choice
		cwd, _ := os.Getwd()
		if profile.ProjectConfigExists(cwd) {
			existingConfig, loadErr := profile.LoadProjectConfig(cwd)
			if loadErr == nil && existingConfig.Profile != name {
				fmt.Println()
				ui.PrintInfo(fmt.Sprintf("Applied profile %q at user scope.", name))
				fmt.Printf("  Note: This directory has an existing project config (%q).\n", existingConfig.Profile)
				fmt.Printf("  Use '%s' to apply at project level.\n",
					ui.Bold(fmt.Sprintf("claudeup profile apply %s --scope project", name)))
			}
		}

		// Save snapshot after applying to sync profile with actual installed state
		// This prevents the profile from showing as "modified" immediately after creation
		fmt.Println()
		ui.PrintInfo("Saving profile snapshot...")
		claudeJSONPath := filepath.Join(claudeDir, ".claude.json")
		snapshot, err := profile.Snapshot(name, claudeDir, claudeJSONPath)
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

	// "current" is reserved as a keyword for the active profile
	if name == "current" {
		return fmt.Errorf("'current' is a reserved name. Use a different profile name")
	}

	// Check if target profile already exists
	existingPath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(existingPath); err == nil {
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
		// -y flag: use active profile
		cfg, _ := config.Load()
		if cfg == nil || cfg.Preferences.ActiveProfile == "" {
			return fmt.Errorf("no active profile. Use --from <profile> to specify base")
		}
		sourceProfile, err = loadProfileWithFallback(profilesDir, cfg.Preferences.ActiveProfile)
		if err != nil {
			return fmt.Errorf("active profile %q not found: %w", cfg.Preferences.ActiveProfile, err)
		}
		fmt.Printf("Using active profile: %s\n", cfg.Preferences.ActiveProfile)
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

func runProfileCurrent(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	profilesDir := getProfilesDir()

	// Check for project-level profile first (takes precedence)
	if profile.ProjectConfigExists(cwd) {
		projectCfg, err := profile.LoadProjectConfig(cwd)
		if err == nil {
			p, err := loadProfileWithFallback(profilesDir, projectCfg.Profile)

			fmt.Println(ui.RenderDetail("Current profile", ui.Bold(projectCfg.Profile)))
			fmt.Printf("  %s\n", ui.Info("(project scope)"))

			if err != nil {
				// Profile doesn't exist, show warning but don't fail
				fmt.Printf("  %s\n", ui.Warning(fmt.Sprintf("Profile definition not found: %v", err)))
			} else {
				// Profile exists, show full details
				if p.Description != "" {
					fmt.Printf("  %s\n", ui.Muted(p.Description))
				}
				fmt.Println()
				fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", fmt.Sprintf("%d", len(p.Marketplaces))), 1))
				fmt.Println(ui.Indent(ui.RenderDetail("Plugins", fmt.Sprintf("%d", len(p.Plugins))), 1))

				// Check for .mcp.json
				if profile.MCPJSONExists(cwd) {
					mcpCfg, _ := profile.LoadMCPJSON(cwd)
					if mcpCfg != nil {
						fmt.Println(ui.Indent(ui.RenderDetail("MCP Servers", fmt.Sprintf("%d", len(mcpCfg.MCPServers))), 1))
					}
				}
			}

			return nil
		}
	}

	// Check for local-scope profile in registry
	registry, err := config.LoadProjectsRegistry()
	if err == nil {
		if entry, ok := registry.GetProject(cwd); ok {
			p, err := loadProfileWithFallback(profilesDir, entry.Profile)

			fmt.Println(ui.RenderDetail("Current profile", ui.Bold(entry.Profile)))
			fmt.Printf("  %s\n", ui.Info("(local scope)"))

			if err != nil {
				// Profile doesn't exist, show warning but don't fail
				fmt.Printf("  %s\n", ui.Warning(fmt.Sprintf("Profile definition not found: %v", err)))
			} else {
				// Profile exists, show full details
				if p.Description != "" {
					fmt.Printf("  %s\n", ui.Muted(p.Description))
				}
				fmt.Println()
				fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", fmt.Sprintf("%d", len(p.Marketplaces))), 1))
				fmt.Println(ui.Indent(ui.RenderDetail("Plugins", fmt.Sprintf("%d", len(p.Plugins))), 1))
				fmt.Println(ui.Indent(ui.RenderDetail("MCP Servers", fmt.Sprintf("%d", len(p.MCPServers))), 1))
			}
			return nil
		}
	}

	// Fall back to user-level profile
	cfg, _ := config.Load()
	activeProfile := ""
	if cfg != nil {
		activeProfile = cfg.Preferences.ActiveProfile
	}

	if activeProfile == "" {
		ui.PrintInfo("No profile is currently active.")
		fmt.Printf("  %s Use 'claudeup profile apply <name>' to apply a profile.\n", ui.Muted(ui.SymbolArrow))
		return nil
	}

	// Load the profile to show details
	p, err := loadProfileWithFallback(profilesDir, activeProfile)
	if err != nil {
		// Profile was set but can't be loaded - show name and error
		ui.PrintWarning(fmt.Sprintf("Current profile: %s (details unavailable: %v)", activeProfile, err))
		return nil
	}

	fmt.Println(ui.RenderDetail("Current profile", ui.Bold(p.Name)))
	fmt.Printf("  %s\n", ui.Info("(user scope)"))
	if p.Description != "" {
		fmt.Printf("  %s\n", ui.Muted(p.Description))
	}
	fmt.Println()
	fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", fmt.Sprintf("%d", len(p.Marketplaces))), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("Plugins", fmt.Sprintf("%d", len(p.Plugins))), 1))
	fmt.Println(ui.Indent(ui.RenderDetail("MCP Servers", fmt.Sprintf("%d", len(p.MCPServers))), 1))

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
	current, _ := profile.Snapshot("current", claudeDir, claudeJSONPath)

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

	result, err := profile.Reset(p, claudeDir, claudeJSONPath)
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

	// Clear active profile if it matches
	cfg, _ := config.Load()
	if cfg != nil && cfg.Preferences.ActiveProfile == name {
		cfg.Preferences.ActiveProfile = ""
		if err := config.Save(cfg); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not clear active profile: %v", err))
		}
	}

	fmt.Println()
	ui.PrintSuccess("Profile reset complete!")

	return nil
}

func runProfileDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Check if it's a built-in profile
	if profile.IsEmbeddedProfile(name) {
		// Check if it has customizations
		profilePath := filepath.Join(profilesDir, name+".json")
		if _, err := os.Stat(profilePath); err == nil {
			return fmt.Errorf("profile %q is a customized built-in profile. Use 'claudeup profile restore %s' instead", name, name)
		}
		return fmt.Errorf("profile %q is a built-in profile and cannot be deleted", name)
	}

	// Check if profile file exists on disk
	profilePath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found", name)
	}

	// Check if this is the currently active profile
	cfg, _ := config.Load()
	isActive := cfg != nil && cfg.Preferences.ActiveProfile == name

	// Show what we're about to do
	fmt.Println(ui.RenderDetail("Delete profile", ui.Bold(name)))
	fmt.Println()
	ui.PrintWarning("This will permanently remove this profile.")
	if isActive {
		ui.PrintWarning("This profile is currently active.")
	}
	fmt.Println()

	if !confirmProceed() {
		ui.PrintMuted("Cancelled.")
		return nil
	}

	// Delete the file
	if err := os.Remove(profilePath); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	// Clear active profile if it matches
	if isActive {
		cfg.Preferences.ActiveProfile = ""
		if err := config.Save(cfg); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not clear active profile: %v", err))
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Deleted profile %q", name))

	// If we deleted the active profile, tell user to select a new one
	if isActive {
		fmt.Println()
		fmt.Println(ui.Muted("→ Run 'claudeup profile apply <name>' to select a new active profile"))
	}

	return nil
}

func runProfileRestore(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Check if it's a built-in profile
	if !profile.IsEmbeddedProfile(name) {
		return fmt.Errorf("profile %q is not a built-in profile. Use 'claudeup profile delete %s' instead", name, name)
	}

	// Check if profile file exists on disk (customization)
	profilePath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
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

	// Clear active profile if it matches (will use built-in now)
	cfg, _ := config.Load()
	if cfg != nil && cfg.Preferences.ActiveProfile == name {
		// Keep the active profile set - it will now use the built-in version
	}

	ui.PrintSuccess(fmt.Sprintf("Restored built-in profile %q", name))

	return nil
}

func runProfileRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]
	profilesDir := getProfilesDir()

	// "current" is reserved as a keyword for the active profile
	if newName == "current" {
		return fmt.Errorf("'current' is a reserved name. Use a different profile name")
	}

	// Check if it's a built-in profile
	if profile.IsEmbeddedProfile(oldName) {
		return fmt.Errorf("profile %q is a built-in profile and cannot be renamed", oldName)
	}

	// Check if source profile exists on disk
	oldPath := filepath.Join(profilesDir, oldName+".json")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found", oldName)
	}

	// Check if target profile already exists
	newPath := filepath.Join(profilesDir, newName+".json")
	if _, err := os.Stat(newPath); err == nil {
		if !config.YesFlag {
			return fmt.Errorf("profile %q already exists. Use -y to overwrite", newName)
		}
		// Remove existing target profile
		if err := os.Remove(newPath); err != nil {
			return fmt.Errorf("failed to remove existing profile: %w", err)
		}
	}

	// Load the profile
	p, err := profile.Load(profilesDir, oldName)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	// Update the name
	p.Name = newName

	// Save with new name
	if err := profile.Save(profilesDir, p); err != nil {
		return fmt.Errorf("failed to save renamed profile: %w", err)
	}

	// Remove old profile file
	if err := os.Remove(oldPath); err != nil {
		// Rollback: remove the new file we just created to avoid inconsistent state
		os.Remove(newPath)
		return fmt.Errorf("failed to remove old profile: %w", err)
	}

	// Update active profile if it matches
	// Error ignored: missing/corrupt config means no active profile to update
	cfg, _ := config.Load()
	if cfg != nil && cfg.Preferences.ActiveProfile == oldName {
		cfg.Preferences.ActiveProfile = newName
		if err := config.Save(cfg); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not update active profile: %v", err))
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Renamed profile %q to %q", oldName, newName))

	return nil
}

func runProfileSync(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check for .claudeup.json
	if !profile.ProjectConfigExists(cwd) {
		return fmt.Errorf("no %s found in current directory.\nRun `claudeup profile apply <name> --scope project` to create one", profile.ProjectConfigFile)
	}

	opts := profile.SyncOptions{
		DryRun: profileSyncDryRun,
	}

	// Add progress callback when not in dry-run mode
	if !profileSyncDryRun {
		opts.Progress = ui.PluginProgress()
	}

	if profileSyncDryRun {
		ui.PrintInfo("Dry run - no changes will be made")
		fmt.Println()
	}

	profilesDir := getProfilesDir()
	result, err := profile.Sync(profilesDir, cwd, claudeDir, opts)
	if err != nil {
		return err
	}

	// Show results
	if profileSyncDryRun {
		fmt.Println("Would sync:")
	} else {
		fmt.Println("Synced:")
	}

	if result.MarketplacesAdded > 0 {
		fmt.Printf("  Marketplaces: %d\n", result.MarketplacesAdded)
	}
	if result.PluginsInstalled > 0 {
		fmt.Printf("  Plugins installed: %d\n", result.PluginsInstalled)
	}
	if result.PluginsSkipped > 0 {
		fmt.Printf("  Plugins already installed: %d\n", result.PluginsSkipped)
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		ui.PrintWarning("Some items failed to sync:")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	if !profileSyncDryRun && result.PluginsInstalled > 0 {
		fmt.Println()
		ui.PrintSuccess("Sync complete!")
	} else if !profileSyncDryRun && result.PluginsInstalled == 0 && len(result.Errors) == 0 {
		fmt.Println()
		ui.PrintSuccess("Already synced - no changes needed")
	}

	return nil
}
