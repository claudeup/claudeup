// ABOUTME: Profile struct and Load/Save functionality for claudeup
// ABOUTME: Profiles define a desired state of Claude Code configuration
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/events"
)

// AmbiguousProfileError is returned when a profile name matches multiple files
// in the profiles directory (e.g. both "profiles/api.json" and "profiles/backend/api.json").
type AmbiguousProfileError struct {
	Name  string   // the profile name that was searched for
	Paths []string // relative paths of all matching profiles (forward-slash separated, without .json)
}

func (e *AmbiguousProfileError) Error() string {
	return fmt.Sprintf("ambiguous profile name %q matches %d profiles: %s",
		e.Name, len(e.Paths), strings.Join(e.Paths, ", "))
}

// Profile represents a Claude Code configuration profile
type Profile struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	Includes       []string       `json:"includes,omitempty"`
	MCPServers     []MCPServer    `json:"mcpServers,omitempty"`
	Marketplaces   []Marketplace  `json:"marketplaces,omitempty"`
	Plugins        []string       `json:"plugins,omitempty"`
	SkipPluginDiff bool           `json:"skipPluginDiff,omitempty"` // If true, don't add/remove plugins (managed externally e.g. by wizard)
	Detect         DetectRules    `json:"detect,omitempty"`
	PostApply      *PostApplyHook `json:"postApply,omitempty"`

	// PerScope contains settings organized by scope (user, project, local).
	// When present, this takes precedence over the flat Plugins/MCPServers fields.
	// When absent, the flat fields are treated as user-scope (backward compatibility).
	PerScope *PerScopeSettings `json:"perScope,omitempty"`

	// Extensions contains patterns for extensions to enable (agents, commands, etc.)
	Extensions *ExtensionSettings `json:"extensions,omitempty"`

	// SettingsHooks contains hooks to merge into settings.json by event type
	SettingsHooks map[string][]HookEntry `json:"settingsHooks,omitempty"`
}

// PerScopeSettings organizes configuration by scope level.
// This enables profiles to capture and restore settings to the correct scope.
type PerScopeSettings struct {
	User    *ScopeSettings `json:"user,omitempty"`
	Project *ScopeSettings `json:"project,omitempty"`
	Local   *ScopeSettings `json:"local,omitempty"`
}

// ScopeSettings contains settings for a single scope level.
type ScopeSettings struct {
	Plugins    []string           `json:"plugins,omitempty"`
	MCPServers []MCPServer        `json:"mcpServers,omitempty"`
	Extensions *ExtensionSettings `json:"extensions,omitempty"`
}

// IsMultiScope returns true if this profile uses per-scope settings.
func (p *Profile) IsMultiScope() bool {
	if p == nil {
		return false
	}
	return p.PerScope != nil
}

// IsStack returns true if this profile composes other profiles via includes.
func (p *Profile) IsStack() bool {
	return p != nil && len(p.Includes) > 0
}

// HasConfigFields returns true if the profile has any configuration
// fields beyond name, description, and includes.
func (p *Profile) HasConfigFields() bool {
	if p == nil {
		return false
	}
	return len(p.Marketplaces) > 0 ||
		len(p.Plugins) > 0 ||
		len(p.MCPServers) > 0 ||
		p.PerScope != nil ||
		p.Extensions != nil ||
		len(p.SettingsHooks) > 0 ||
		len(p.Detect.Files) > 0 ||
		len(p.Detect.Contains) > 0 ||
		p.PostApply != nil ||
		p.SkipPluginDiff
}

// HasMCPServersWithSecrets returns true if any MCP server in the profile has secrets defined.
// This is used to warn users that sync cannot resolve secrets.
func (p *Profile) HasMCPServersWithSecrets() bool {
	if p == nil {
		return false
	}

	// Check legacy MCP servers
	for _, server := range p.MCPServers {
		if len(server.Secrets) > 0 {
			return true
		}
	}

	// Check multi-scope MCP servers
	if p.PerScope != nil {
		for _, scope := range []*ScopeSettings{p.PerScope.User, p.PerScope.Project, p.PerScope.Local} {
			if scope != nil {
				for _, server := range scope.MCPServers {
					if len(server.Secrets) > 0 {
						return true
					}
				}
			}
		}
	}

	return false
}

// CombinedScopes returns a flat Profile combining all scopes (user + project + local).
// This aggregates plugins and MCP servers from all scopes into single lists,
// matching how Claude Code accumulates settings from user → project → local.
// Useful for comparing a multi-scope profile against the combined system state.
func (p *Profile) CombinedScopes() *Profile {
	if p == nil {
		return &Profile{}
	}

	result := &Profile{
		Name:         p.Name,
		Description:  p.Description,
		Marketplaces: p.Marketplaces,
	}

	if p.PerScope == nil {
		// Legacy profile - all data is already flat
		result.Plugins = p.Plugins
		result.MCPServers = p.MCPServers
		return result
	}

	// Aggregate plugins from all scopes (use a set to avoid duplicates)
	pluginSet := make(map[string]bool)
	for _, scope := range []*ScopeSettings{p.PerScope.User, p.PerScope.Project, p.PerScope.Local} {
		if scope != nil {
			for _, plugin := range scope.Plugins {
				pluginSet[plugin] = true
			}
		}
	}
	for plugin := range pluginSet {
		result.Plugins = append(result.Plugins, plugin)
	}

	// Aggregate MCP servers from all scopes (later scopes override earlier)
	serverMap := make(map[string]MCPServer)
	for _, scope := range []*ScopeSettings{p.PerScope.User, p.PerScope.Project, p.PerScope.Local} {
		if scope != nil {
			for _, server := range scope.MCPServers {
				serverMap[server.Name] = server
			}
		}
	}
	for _, server := range serverMap {
		result.MCPServers = append(result.MCPServers, server)
	}

	// Aggregate extensions from all scopes (union with dedup)
	for _, scope := range []*ScopeSettings{p.PerScope.User, p.PerScope.Project, p.PerScope.Local} {
		if scope != nil && scope.Extensions != nil {
			if result.Extensions == nil {
				result.Extensions = &ExtensionSettings{}
			}
			result.Extensions.Agents = mergeStringSlice(result.Extensions.Agents, scope.Extensions.Agents)
			result.Extensions.Commands = mergeStringSlice(result.Extensions.Commands, scope.Extensions.Commands)
			result.Extensions.Skills = mergeStringSlice(result.Extensions.Skills, scope.Extensions.Skills)
			result.Extensions.Hooks = mergeStringSlice(result.Extensions.Hooks, scope.Extensions.Hooks)
			result.Extensions.Rules = mergeStringSlice(result.Extensions.Rules, scope.Extensions.Rules)
			result.Extensions.OutputStyles = mergeStringSlice(result.Extensions.OutputStyles, scope.Extensions.OutputStyles)
		}
	}

	return result
}

// ForScope returns a flat Profile containing only settings for the specified scope.
// This is useful for applying a single scope from a multi-scope profile.
// Marketplaces are always included since they're user-scoped.
func (p *Profile) ForScope(scope string) *Profile {
	if p == nil {
		return &Profile{}
	}

	result := &Profile{
		Name:         p.Name,
		Description:  p.Description,
		Marketplaces: p.Marketplaces,
	}

	if p.PerScope == nil {
		// Legacy profile - all data is user-scoped.
		// Multi-scope profiles should use PerScope for project/local data.
		if scope == "user" {
			result.Plugins = p.Plugins
			result.MCPServers = p.MCPServers
		}
		return result
	}

	var settings *ScopeSettings
	switch scope {
	case "user":
		settings = p.PerScope.User
	case "project":
		settings = p.PerScope.Project
	case "local":
		settings = p.PerScope.Local
	}

	if settings != nil {
		result.Plugins = settings.Plugins
		result.MCPServers = settings.MCPServers
		result.Extensions = settings.Extensions
	}

	return result
}

// FilterToScope removes all scope data except the specified scope.
// This is used when saving a profile for a single scope.
func (p *Profile) FilterToScope(scope string) {
	if p.PerScope == nil {
		return
	}
	switch scope {
	case "user":
		p.PerScope.Project = nil
		p.PerScope.Local = nil
		// Only keep marketplaces referenced by remaining user-scope plugins
		p.filterMarketplacesToPlugins()
	case "project":
		p.PerScope.User = nil
		p.PerScope.Local = nil
		// Marketplaces are user-scoped; only keep those referenced by remaining plugins
		p.filterMarketplacesToPlugins()
	case "local":
		p.PerScope.User = nil
		p.PerScope.Project = nil
		// Marketplaces are user-scoped; only keep those referenced by remaining plugins
		p.filterMarketplacesToPlugins()
	}
	// Clear flat fields that may have been populated
	p.Plugins = nil
	p.MCPServers = nil
	p.Extensions = nil
	// Regenerate description
	p.Description = p.GenerateDescription()
}

// filterMarketplacesToPlugins removes marketplaces not referenced by remaining plugins
func (p *Profile) filterMarketplacesToPlugins() {
	remaining := p.CombinedScopes()
	if len(remaining.Plugins) == 0 {
		p.Marketplaces = nil
		return
	}
	// Build set of marketplace refs from plugin names (part after @)
	refs := make(map[string]bool)
	for _, plugin := range remaining.Plugins {
		parts := strings.SplitN(plugin, "@", 2)
		if len(parts) == 2 {
			refs[parts[1]] = true
		}
	}
	var filtered []Marketplace
	for _, m := range p.Marketplaces {
		// Plugin refs are short names (e.g., "claude-plugins-official") while
		// Marketplace.Repo is "owner/repo" (e.g., "anthropics/claude-plugins-official").
		// Match if any ref equals the repo or its last path segment.
		for ref := range refs {
			if m.Repo == ref || strings.HasSuffix(m.Repo, "/"+ref) {
				filtered = append(filtered, m)
				break
			}
		}
	}
	p.Marketplaces = filtered
}

// PostApplyHook defines a hook to run after a profile is applied.
//
// Execution order: Script takes precedence over Command. If both are set,
// only Script will be executed.
//
// Condition types:
//   - "always" (default): Hook runs every time the profile is applied
//   - "first-run": Hook only runs if no plugins from the profile's marketplaces
//     are currently enabled
//
// Security note: Hooks execute arbitrary shell commands. Only use profiles
// from trusted sources.
type PostApplyHook struct {
	Script    string `json:"script,omitempty"`    // Script path relative to profile (takes precedence)
	Command   string `json:"command,omitempty"`   // Direct command to run (used if Script is empty)
	Condition string `json:"condition,omitempty"` // "always" (default) or "first-run"
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name    string               `json:"name"`
	Command string               `json:"command"`
	Args    []string             `json:"args,omitempty"`
	Scope   string               `json:"scope,omitempty"`
	Secrets map[string]SecretRef `json:"secrets,omitempty"`
}

// Marketplace represents a plugin marketplace source
type Marketplace struct {
	Source string `json:"source"`
	Repo   string `json:"repo,omitempty"` // Used for github sources
	URL    string `json:"url,omitempty"`  // Used for git sources
}

// DisplayName returns the repo or URL for display purposes
func (m Marketplace) DisplayName() string {
	if m.Repo != "" {
		return m.Repo
	}
	return m.URL
}

// SecretRef defines a secret requirement with multiple resolution sources
type SecretRef struct {
	Description string         `json:"description,omitempty"`
	Sources     []SecretSource `json:"sources"`
}

// SecretSource defines a single source for resolving a secret
type SecretSource struct {
	Type    string `json:"type"`              // env, 1password, keychain
	Key     string `json:"key,omitempty"`     // for env
	Ref     string `json:"ref,omitempty"`     // for 1password
	Service string `json:"service,omitempty"` // for keychain
	Account string `json:"account,omitempty"` // for keychain
}

// DetectRules defines how to auto-detect if a profile matches a project
type DetectRules struct {
	Files    []string          `json:"files,omitempty"`
	Contains map[string]string `json:"contains,omitempty"`
}

// profileJSON is the raw JSON shape used for unmarshaling profiles.
// It accepts both the old "localItems" field and the new "extensions" field.
type profileJSON struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Includes       []string               `json:"includes,omitempty"`
	MCPServers     []MCPServer            `json:"mcpServers,omitempty"`
	Marketplaces   []Marketplace          `json:"marketplaces,omitempty"`
	Plugins        []string               `json:"plugins,omitempty"`
	SkipPluginDiff bool                   `json:"skipPluginDiff,omitempty"`
	Detect         DetectRules            `json:"detect,omitempty"`
	PostApply      *PostApplyHook         `json:"postApply,omitempty"`
	PerScope       *perScopeSettingsJSON  `json:"perScope,omitempty"`
	Extensions     *ExtensionSettings     `json:"extensions,omitempty"`
	LocalItems     *ExtensionSettings     `json:"localItems,omitempty"` // deprecated field
	SettingsHooks  map[string][]HookEntry `json:"settingsHooks,omitempty"`
}

// perScopeSettingsJSON is the raw JSON shape for per-scope settings,
// accepting both "localItems" and "extensions" in each scope.
type perScopeSettingsJSON struct {
	User    *scopeSettingsJSON `json:"user,omitempty"`
	Project *scopeSettingsJSON `json:"project,omitempty"`
	Local   *scopeSettingsJSON `json:"local,omitempty"`
}

// scopeSettingsJSON is the raw JSON shape for a single scope,
// accepting both "localItems" and "extensions".
type scopeSettingsJSON struct {
	Plugins    []string           `json:"plugins,omitempty"`
	MCPServers []MCPServer        `json:"mcpServers,omitempty"`
	Extensions *ExtensionSettings `json:"extensions,omitempty"`
	LocalItems *ExtensionSettings `json:"localItems,omitempty"` // deprecated field
}

// UnmarshalJSON handles migration from the old "localItems" JSON field
// to the new "extensions" field. Profiles saved before the rename used
// "localItems"; this ensures they load correctly.
func (p *Profile) UnmarshalJSON(data []byte) error {
	var raw profileJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.Name = raw.Name
	p.Description = raw.Description
	p.Includes = raw.Includes
	p.MCPServers = raw.MCPServers
	p.Marketplaces = raw.Marketplaces
	p.Plugins = raw.Plugins
	p.SkipPluginDiff = raw.SkipPluginDiff
	p.Detect = raw.Detect
	p.PostApply = raw.PostApply
	p.SettingsHooks = raw.SettingsHooks

	// Migrate top-level localItems → extensions
	p.Extensions = raw.Extensions
	if p.Extensions == nil && raw.LocalItems != nil {
		p.Extensions = raw.LocalItems
	}

	// Migrate per-scope settings
	if raw.PerScope != nil {
		p.PerScope = &PerScopeSettings{
			User:    migrateScopeSettings(raw.PerScope.User),
			Project: migrateScopeSettings(raw.PerScope.Project),
			Local:   migrateScopeSettings(raw.PerScope.Local),
		}
	}

	return nil
}

// migrateScopeSettings converts a scopeSettingsJSON (with possible old
// "localItems" field) into a ScopeSettings with the canonical "extensions" field.
func migrateScopeSettings(raw *scopeSettingsJSON) *ScopeSettings {
	if raw == nil {
		return nil
	}
	s := &ScopeSettings{
		Plugins:    raw.Plugins,
		MCPServers: raw.MCPServers,
		Extensions: raw.Extensions,
	}
	if s.Extensions == nil && raw.LocalItems != nil {
		s.Extensions = raw.LocalItems
	}
	return s
}

// ExtensionSettings contains extension patterns to enable.
// These are items from ~/.claudeup/ext/ that get symlinked to ~/.claude/
type ExtensionSettings struct {
	Agents       []string `json:"agents,omitempty"`
	Commands     []string `json:"commands,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	Hooks        []string `json:"hooks,omitempty"`
	Rules        []string `json:"rules,omitempty"`
	OutputStyles []string `json:"output-styles,omitempty"`
}

// HookEntry represents a single hook configuration for settings.json
type HookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// PreserveFrom copies extensions from an existing profile.
// When re-saving, this keeps only the extensions the user originally saved,
// preventing accumulation of items enabled by other tools.
func (p *Profile) PreserveFrom(existing *Profile) {
	p.Extensions = existing.Extensions
}

// Save writes a profile to the profiles directory
func Save(profilesDir string, p *Profile) error {
	profilePath := filepath.Join(profilesDir, p.Name+".json")

	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	// Ensure trailing newline (POSIX text file convention)
	data = append(data, '\n')

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"profile save",
		profilePath,
		"local",
		func() error {
			return os.WriteFile(profilePath, data, 0644)
		},
	)
}

// Load reads a profile from the profiles directory.
// If name contains "/", it is treated as a relative path within profilesDir.
// Otherwise, profilesDir is searched recursively for a matching .json file.
// Returns an error if the name matches multiple profiles (ambiguous).
func Load(profilesDir, name string) (*Profile, error) {
	paths, err := FindProfilePaths(profilesDir, name)
	if err != nil {
		return nil, err
	}

	switch len(paths) {
	case 0:
		return nil, &os.PathError{Op: "open", Path: filepath.Join(profilesDir, name+".json"), Err: os.ErrNotExist}
	case 1:
		return LoadFromPath(paths[0])
	default:
		// Build relative paths for the error
		relPaths := make([]string, 0, len(paths))
		for _, p := range paths {
			rel, err := filepath.Rel(profilesDir, p)
			if err != nil {
				rel = p
			}
			relPaths = append(relPaths, strings.TrimSuffix(filepath.ToSlash(rel), ".json"))
		}
		return nil, &AmbiguousProfileError{Name: name, Paths: relPaths}
	}
}

// ProfileEntry is a profile with its location relative to the profiles directory.
// RelPath uses forward slashes (e.g. "backend/api.json" or "mobile.json").
type ProfileEntry struct {
	*Profile
	RelPath string
}

// DisplayName returns the profile's display name for listing.
// For root profiles, this is just the profile name.
// For nested profiles, this is the relative path without the .json extension.
func (e ProfileEntry) DisplayName() string {
	return strings.TrimSuffix(e.RelPath, ".json")
}

// FindProfilePaths walks profilesDir recursively and returns absolute paths
// to .json files whose filename stem matches name.
// If name contains a "/", it is treated as a relative path reference:
// only profilesDir/name.json is checked (after validating the path stays within profilesDir).
// Returns an empty slice (not an error) if profilesDir does not exist.
// The profilesDir argument is resolved to an absolute path internally.
func FindProfilePaths(profilesDir, name string) ([]string, error) {
	// Ensure absolute profilesDir so WalkDir returns absolute paths
	var err error
	profilesDir, err = filepath.Abs(profilesDir)
	if err != nil {
		return nil, fmt.Errorf("invalid profiles directory: %w", err)
	}

	// Path reference mode: name contains "/"
	if strings.Contains(name, "/") {
		// Normalize to OS-specific separators for correct filepath operations
		name = filepath.FromSlash(name)
		target := filepath.Clean(filepath.Join(profilesDir, name+".json"))
		// Validate the resolved path stays within profilesDir to prevent traversal
		if target != profilesDir && !strings.HasPrefix(target, profilesDir+string(filepath.Separator)) {
			return nil, fmt.Errorf("invalid profile path %q: escapes profiles directory", name)
		}
		if _, err := os.Stat(target); err == nil {
			return []string{target}, nil
		}
		return []string{}, nil
	}

	// Name-based search: walk recursively
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	matches := make([]string, 0, 8)
	err = filepath.WalkDir(profilesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir // skip unreadable subdirectories
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		stem := strings.TrimSuffix(d.Name(), ".json")
		if stem == name {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(matches)
	return matches, nil
}

// LoadFromPath loads a profile from an absolute file path.
// If the JSON does not contain a name field, the name is derived from the filename.
func LoadFromPath(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	// Set name from filename if not present in JSON
	if p.Name == "" {
		p.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	}

	return &p, nil
}

// ProjectProfilesDir returns the path to project-local profiles directory
func ProjectProfilesDir(projectDir string) string {
	return filepath.Join(projectDir, ".claudeup", "profiles")
}

// SaveToProject saves a profile to the project's .claudeup/profiles/ directory
func SaveToProject(projectDir string, p *Profile) error {
	profilesDir := ProjectProfilesDir(projectDir)
	return Save(profilesDir, p)
}

// List returns all profiles in the profiles directory (including subdirectories),
// sorted by name then by relative path for duplicates.
func List(profilesDir string) ([]ProfileEntry, error) {
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		return []ProfileEntry{}, nil
	}

	var entries []ProfileEntry
	err := filepath.WalkDir(profilesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir // skip unreadable subdirectories
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		p, loadErr := LoadFromPath(path)
		if loadErr != nil {
			return nil // skip invalid profiles (bad JSON, etc.)
		}

		relPath, relErr := filepath.Rel(profilesDir, path)
		if relErr != nil {
			return relErr
		}

		entries = append(entries, ProfileEntry{Profile: p, RelPath: filepath.ToSlash(relPath)})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].RelPath < entries[j].RelPath
	})

	return entries, nil
}

// ProfileWithSource wraps a profile with its source location and relative path
type ProfileWithSource struct {
	*Profile
	Source  string // "user" or "project"
	RelPath string // relative path within profiles dir (e.g. "backend/api.json")
}

// DisplayName returns the profile's display name for listing.
// For root profiles, this is just the profile name.
// For nested profiles, this is the relative path without the .json extension.
func (p *ProfileWithSource) DisplayName() string {
	return strings.TrimSuffix(p.RelPath, ".json")
}

// ListAll returns profiles from both user and project directories.
// Project profiles take precedence over user profiles with the same name.
func ListAll(userProfilesDir, projectDir string) ([]*ProfileWithSource, error) {
	var all []*ProfileWithSource
	seen := make(map[string]bool)

	// List project profiles first (higher precedence)
	projectProfilesDir := ProjectProfilesDir(projectDir)
	projectProfiles, err := List(projectProfilesDir)
	if err == nil {
		for _, entry := range projectProfiles {
			all = append(all, &ProfileWithSource{
				Profile: entry.Profile,
				Source:  "project",
				RelPath: entry.RelPath,
			})
			seen[entry.Name] = true
		}
	}

	// List user profiles (skip if already in project)
	userProfiles, err := List(userProfilesDir)
	if err == nil {
		for _, entry := range userProfiles {
			if !seen[entry.Name] {
				all = append(all, &ProfileWithSource{
					Profile: entry.Profile,
					Source:  "user",
					RelPath: entry.RelPath,
				})
			}
		}
	}

	// Sort by name, then by RelPath for duplicates
	sort.Slice(all, func(i, j int) bool {
		if all[i].Name != all[j].Name {
			return all[i].Name < all[j].Name
		}
		return all[i].RelPath < all[j].RelPath
	})

	return all, nil
}

// Clone creates a deep copy of the profile with a new name
func (p *Profile) Clone(newName string) *Profile {
	clone := &Profile{
		Name:        newName,
		Description: p.Description,
	}

	// Deep copy Includes
	if len(p.Includes) > 0 {
		clone.Includes = make([]string, len(p.Includes))
		copy(clone.Includes, p.Includes)
	}

	// Deep copy MCPServers
	if len(p.MCPServers) > 0 {
		clone.MCPServers = make([]MCPServer, len(p.MCPServers))
		for i, srv := range p.MCPServers {
			clone.MCPServers[i] = MCPServer{
				Name:    srv.Name,
				Command: srv.Command,
				Scope:   srv.Scope,
			}
			if len(srv.Args) > 0 {
				clone.MCPServers[i].Args = make([]string, len(srv.Args))
				copy(clone.MCPServers[i].Args, srv.Args)
			}
			if len(srv.Secrets) > 0 {
				clone.MCPServers[i].Secrets = make(map[string]SecretRef)
				for k, v := range srv.Secrets {
					sources := make([]SecretSource, len(v.Sources))
					copy(sources, v.Sources)
					clone.MCPServers[i].Secrets[k] = SecretRef{
						Description: v.Description,
						Sources:     sources,
					}
				}
			}
		}
	}

	// Deep copy Marketplaces
	if len(p.Marketplaces) > 0 {
		clone.Marketplaces = make([]Marketplace, len(p.Marketplaces))
		copy(clone.Marketplaces, p.Marketplaces)
	}

	// Deep copy Plugins
	if len(p.Plugins) > 0 {
		clone.Plugins = make([]string, len(p.Plugins))
		copy(clone.Plugins, p.Plugins)
	}

	// Deep copy Detect
	if len(p.Detect.Files) > 0 {
		clone.Detect.Files = make([]string, len(p.Detect.Files))
		copy(clone.Detect.Files, p.Detect.Files)
	}
	if len(p.Detect.Contains) > 0 {
		clone.Detect.Contains = make(map[string]string)
		for k, v := range p.Detect.Contains {
			clone.Detect.Contains[k] = v
		}
	}

	// Deep copy PerScope
	if p.PerScope != nil {
		clone.PerScope = &PerScopeSettings{}
		if p.PerScope.User != nil {
			clone.PerScope.User = cloneScopeSettings(p.PerScope.User)
		}
		if p.PerScope.Project != nil {
			clone.PerScope.Project = cloneScopeSettings(p.PerScope.Project)
		}
		if p.PerScope.Local != nil {
			clone.PerScope.Local = cloneScopeSettings(p.PerScope.Local)
		}
	}

	return clone
}

// Equal compares two profiles for semantic equality, ignoring the Name field.
// Name is treated as an identifier, not content - two profiles with different names
// but identical content are considered equal.
// Nil and empty slices are treated as equivalent.
func (p *Profile) Equal(other *Profile) bool {
	if other == nil {
		return false
	}

	// Compare description
	if p.Description != other.Description {
		return false
	}

	// Compare Includes
	if !strSlicesEqual(p.Includes, other.Includes) {
		return false
	}

	// Compare SkipPluginDiff
	if p.SkipPluginDiff != other.SkipPluginDiff {
		return false
	}

	// Compare slices (nil and empty are equivalent)
	if !strSlicesEqual(p.Plugins, other.Plugins) {
		return false
	}

	if !marketplaceSlicesEqual(p.Marketplaces, other.Marketplaces) {
		return false
	}

	if !mcpServerSlicesEqual(p.MCPServers, other.MCPServers) {
		return false
	}

	// Compare DetectRules
	if !detectRulesStructEqual(p.Detect, other.Detect) {
		return false
	}

	// Compare PostApplyHook
	if !postApplyHookPtrEqual(p.PostApply, other.PostApply) {
		return false
	}

	// Compare PerScope
	if !perScopeSettingsEqual(p.PerScope, other.PerScope) {
		return false
	}

	return true
}

// strSlicesEqual compares two string slices, treating nil and empty as equal
func strSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// marketplaceSlicesEqual compares two marketplace slices
func marketplaceSlicesEqual(a, b []Marketplace) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Source != b[i].Source || a[i].Repo != b[i].Repo || a[i].URL != b[i].URL {
			return false
		}
	}
	return true
}

// mcpServersEqual checks if two MCP servers are equal
// Compares: command, args, scope, and secrets
func mcpServersEqual(a, b MCPServer) bool {
	// Compare command
	if a.Command != b.Command {
		return false
	}

	// Compare scope
	if a.Scope != b.Scope {
		return false
	}

	// Compare args
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if a.Args[i] != b.Args[i] {
			return false
		}
	}

	// Compare secrets
	if len(a.Secrets) != len(b.Secrets) {
		return false
	}
	for key, aSecret := range a.Secrets {
		bSecret, exists := b.Secrets[key]
		if !exists {
			return false
		}
		if !secretRefsEqual(aSecret, bSecret) {
			return false
		}
	}

	return true
}

// secretRefsEqual compares two SecretRef values
func secretRefsEqual(a, b SecretRef) bool {
	if a.Description != b.Description {
		return false
	}

	if len(a.Sources) != len(b.Sources) {
		return false
	}

	for i := range a.Sources {
		if a.Sources[i] != b.Sources[i] {
			return false
		}
	}

	return true
}

// mcpServerSlicesEqual compares two MCP server slices using the mcpServersEqual helper
func mcpServerSlicesEqual(a, b []MCPServer) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		// Also compare Name which mcpServersEqual doesn't check
		if a[i].Name != b[i].Name {
			return false
		}
		if !mcpServersEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// detectRulesStructEqual compares two DetectRules structs
func detectRulesStructEqual(a, b DetectRules) bool {
	if !strSlicesEqual(a.Files, b.Files) {
		return false
	}
	if !strMapsEqual(a.Contains, b.Contains) {
		return false
	}
	return true
}

// strMapsEqual compares two string maps
func strMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if other, ok := b[k]; !ok || v != other {
			return false
		}
	}
	return true
}

// postApplyHookPtrEqual compares two PostApplyHook pointers
func postApplyHookPtrEqual(a, b *PostApplyHook) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Script == b.Script &&
		a.Command == b.Command &&
		a.Condition == b.Condition
}

// GenerateDescription creates a human-readable description of the profile contents
func (p *Profile) GenerateDescription() string {
	if p.IsStack() {
		n := len(p.Includes)
		if n == 1 {
			return "stack: 1 include"
		}
		return fmt.Sprintf("stack: %d includes", n)
	}

	var parts []string

	marketplaceCount := len(p.Marketplaces)
	pluginCount := len(p.Plugins)
	mcpCount := len(p.MCPServers)

	if marketplaceCount > 0 {
		if marketplaceCount == 1 {
			parts = append(parts, "1 marketplace")
		} else {
			parts = append(parts, fmt.Sprintf("%d marketplaces", marketplaceCount))
		}
	}

	if pluginCount > 0 {
		if pluginCount == 1 {
			parts = append(parts, "1 plugin")
		} else {
			parts = append(parts, fmt.Sprintf("%d plugins", pluginCount))
		}
	}

	if mcpCount > 0 {
		if mcpCount == 1 {
			parts = append(parts, "1 MCP server")
		} else {
			parts = append(parts, fmt.Sprintf("%d MCP servers", mcpCount))
		}
	}

	if len(parts) == 0 {
		return "Empty profile"
	}

	// Join with commas: "1 marketplace, 3 plugins, 2 MCP servers"
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// cloneScopeSettings deep-copies a ScopeSettings
func cloneScopeSettings(s *ScopeSettings) *ScopeSettings {
	if s == nil {
		return nil
	}
	clone := &ScopeSettings{}
	if len(s.Plugins) > 0 {
		clone.Plugins = make([]string, len(s.Plugins))
		copy(clone.Plugins, s.Plugins)
	}
	if len(s.MCPServers) > 0 {
		clone.MCPServers = make([]MCPServer, len(s.MCPServers))
		copy(clone.MCPServers, s.MCPServers)
	}
	if s.Extensions != nil {
		clone.Extensions = cloneExtensionSettings(s.Extensions)
	}
	return clone
}

// cloneExtensionSettings deep-copies an ExtensionSettings
func cloneExtensionSettings(l *ExtensionSettings) *ExtensionSettings {
	if l == nil {
		return nil
	}
	clone := &ExtensionSettings{}
	if len(l.Agents) > 0 {
		clone.Agents = make([]string, len(l.Agents))
		copy(clone.Agents, l.Agents)
	}
	if len(l.Commands) > 0 {
		clone.Commands = make([]string, len(l.Commands))
		copy(clone.Commands, l.Commands)
	}
	if len(l.Skills) > 0 {
		clone.Skills = make([]string, len(l.Skills))
		copy(clone.Skills, l.Skills)
	}
	if len(l.Hooks) > 0 {
		clone.Hooks = make([]string, len(l.Hooks))
		copy(clone.Hooks, l.Hooks)
	}
	if len(l.Rules) > 0 {
		clone.Rules = make([]string, len(l.Rules))
		copy(clone.Rules, l.Rules)
	}
	if len(l.OutputStyles) > 0 {
		clone.OutputStyles = make([]string, len(l.OutputStyles))
		copy(clone.OutputStyles, l.OutputStyles)
	}
	return clone
}

// perScopeSettingsEqual compares two PerScopeSettings pointers
func perScopeSettingsEqual(a, b *PerScopeSettings) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !scopeSettingsEqual(a.User, b.User) {
		return false
	}
	if !scopeSettingsEqual(a.Project, b.Project) {
		return false
	}
	if !scopeSettingsEqual(a.Local, b.Local) {
		return false
	}
	return true
}

// scopeSettingsEqual compares two ScopeSettings pointers
func scopeSettingsEqual(a, b *ScopeSettings) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !strSlicesEqual(a.Plugins, b.Plugins) {
		return false
	}
	if !mcpServerSlicesEqual(a.MCPServers, b.MCPServers) {
		return false
	}
	if !extensionSettingsEqual(a.Extensions, b.Extensions) {
		return false
	}
	return true
}

// extensionSettingsEqual compares two ExtensionSettings pointers
func extensionSettingsEqual(a, b *ExtensionSettings) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return strSlicesEqual(a.Agents, b.Agents) &&
		strSlicesEqual(a.Commands, b.Commands) &&
		strSlicesEqual(a.Skills, b.Skills) &&
		strSlicesEqual(a.Hooks, b.Hooks) &&
		strSlicesEqual(a.Rules, b.Rules) &&
		strSlicesEqual(a.OutputStyles, b.OutputStyles)
}
