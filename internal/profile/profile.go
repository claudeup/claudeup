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

	"github.com/claudeup/claudeup/v4/internal/events"
)

// Profile represents a Claude Code configuration profile
type Profile struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
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

	// LocalItems contains patterns for local items to enable (agents, commands, etc.)
	LocalItems *LocalItemSettings `json:"localItems,omitempty"`

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
	Plugins    []string    `json:"plugins,omitempty"`
	MCPServers []MCPServer `json:"mcpServers,omitempty"`
}

// IsMultiScope returns true if this profile uses per-scope settings.
func (p *Profile) IsMultiScope() bool {
	if p == nil {
		return false
	}
	return p.PerScope != nil
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
	}

	return result
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

// LocalItemSettings contains local item patterns to enable.
// These are items from ~/.claude/.library that get symlinked to ~/.claude/
type LocalItemSettings struct {
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

// PreserveFrom copies localItems from an existing profile.
// When re-saving, this keeps only the local items the user originally saved,
// preventing accumulation of items enabled by other tools.
func (p *Profile) PreserveFrom(existing *Profile) {
	p.LocalItems = existing.LocalItems
}

// Save writes a profile to the profiles directory
func Save(profilesDir string, p *Profile) error {
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return err
	}

	profilePath := filepath.Join(profilesDir, p.Name+".json")

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

// Load reads a profile from the profiles directory
func Load(profilesDir, name string) (*Profile, error) {
	profilePath := filepath.Join(profilesDir, name+".json")

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, err
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	// Set name from filename if not present in JSON
	if p.Name == "" {
		p.Name = name
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

// List returns all profiles in the profiles directory, sorted by name
func List(profilesDir string) ([]*Profile, error) {
	entries, err := os.ReadDir(profilesDir)
	if os.IsNotExist(err) {
		return []*Profile{}, nil
	}
	if err != nil {
		return nil, err
	}

	var profiles []*Profile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		p, err := Load(profilesDir, name)
		if err != nil {
			continue // Skip invalid profiles
		}
		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// ProfileWithSource wraps a profile with its source location
type ProfileWithSource struct {
	*Profile
	Source string // "user" or "project"
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
		for _, p := range projectProfiles {
			all = append(all, &ProfileWithSource{Profile: p, Source: "project"})
			seen[p.Name] = true
		}
	}

	// List user profiles (skip if already in project)
	userProfiles, err := List(userProfilesDir)
	if err == nil {
		for _, p := range userProfiles {
			if !seen[p.Name] {
				all = append(all, &ProfileWithSource{Profile: p, Source: "user"})
			}
		}
	}

	// Sort by name
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	return all, nil
}

// Clone creates a deep copy of the profile with a new name
func (p *Profile) Clone(newName string) *Profile {
	clone := &Profile{
		Name:        newName,
		Description: p.Description,
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
