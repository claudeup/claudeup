// ABOUTME: Resolution engine for profile includes (composable stacks)
// ABOUTME: Flattens include trees into a single merged profile with cycle detection
package profile

import (
	"fmt"
	"os"
	"strings"
)

// MaxIncludeDepth limits how deeply nested include chains can be.
// Prevents resource exhaustion from pathological deep nesting.
const MaxIncludeDepth = 50

// ProfileLoader loads a profile by name or path-qualified name.
type ProfileLoader interface {
	LoadProfile(name string) (*Profile, error)
}

// DirLoader loads profiles from disk via Load() with embedded fallback.
type DirLoader struct {
	ProfilesDir string
}

// LoadProfile loads a profile by name, delegating to Load() which handles
// both short names (recursive search) and path-qualified names (direct lookup).
// Falls back to embedded profiles only when the profile is not found on disk.
// Other errors (ambiguous names, invalid JSON) propagate without fallback.
func (l *DirLoader) LoadProfile(name string) (*Profile, error) {
	p, err := Load(l.ProfilesDir, name)
	if err == nil {
		return p, nil
	}

	// Only fall back to embedded when the profile is genuinely not found.
	// Let AmbiguousProfileError, JSON parse errors, etc. propagate.
	if os.IsNotExist(err) {
		return GetEmbeddedProfile(name)
	}
	return nil, err
}

// ResolveIncludes recursively resolves includes and returns a merged profile.
// Returns an error if:
//   - p is nil
//   - the profile has includes alongside config fields (stacks must be pure)
//   - a cycle is detected
//   - an included profile cannot be loaded
//
// If the profile has no includes, it is returned as-is.
func ResolveIncludes(p *Profile, loader ProfileLoader) (*Profile, error) {
	if p == nil {
		return nil, fmt.Errorf("cannot resolve includes: profile is nil")
	}

	if !p.IsStack() {
		return p, nil
	}

	if loader == nil {
		return nil, fmt.Errorf("cannot resolve includes for stack profile %q: loader is nil", p.Name)
	}

	if err := validatePureStack(p); err != nil {
		return nil, err
	}

	// Collect all leaf profiles in include order
	resolved := make(map[string]*Profile)
	visitingSet := make(map[string]bool)
	var visitingPath []string

	var leaves []*Profile
	var collectErr error

	collectLeaves := func(name string) {}
	collectLeaves = func(name string) {
		if collectErr != nil {
			return
		}

		if len(visitingPath) >= MaxIncludeDepth {
			collectErr = fmt.Errorf("include depth limit exceeded (%d levels)", MaxIncludeDepth)
			return
		}

		// Check for cached resolution (diamond support)
		if _, ok := resolved[name]; ok {
			return
		}

		if visitingSet[name] {
			// Build full cycle path: find where the cycle starts and append the duplicate
			cyclePath := []string{name}
			for i := len(visitingPath) - 1; i >= 0; i-- {
				cyclePath = append([]string{visitingPath[i]}, cyclePath...)
				if visitingPath[i] == name {
					break
				}
			}
			collectErr = fmt.Errorf("include cycle detected: %s", strings.Join(cyclePath, " -> "))
			return
		}

		visitingSet[name] = true
		visitingPath = append(visitingPath, name)
		defer func() {
			delete(visitingSet, name)
			visitingPath = visitingPath[:len(visitingPath)-1]
		}()

		included, err := loader.LoadProfile(name)
		if err != nil {
			collectErr = fmt.Errorf("failed to load included profile %q: %w", name, err)
			return
		}

		if included.IsStack() {
			if err := validatePureStack(included); err != nil {
				collectErr = fmt.Errorf("included profile %q: %w", name, err)
				return
			}
			for _, sub := range included.Includes {
				collectLeaves(sub)
				if collectErr != nil {
					return
				}
			}
		} else {
			leaves = append(leaves, included)
		}

		resolved[name] = included
	}

	for _, name := range p.Includes {
		collectLeaves(name)
		if collectErr != nil {
			return nil, collectErr
		}
	}

	result := mergeProfiles(leaves)
	result.Name = p.Name
	result.Description = p.Description
	result.Includes = nil

	return result, nil
}

// validatePureStack checks that a stack profile has no config fields alongside includes.
func validatePureStack(p *Profile) error {
	if p.IsStack() && p.HasConfigFields() {
		return fmt.Errorf("stack profiles must be pure: %q has config fields alongside includes", p.Name)
	}
	return nil
}

// mergeProfiles merges a flat list of profiles left-to-right into a single profile.
func mergeProfiles(profiles []*Profile) *Profile {
	result := &Profile{}
	for _, p := range profiles {
		mergeProfile(result, p)
	}
	return result
}

// mergeProfile merges src into dst, applying the merge strategy for each field.
func mergeProfile(dst, src *Profile) {
	mergeMarketplaces(dst, src)
	mergePerScope(dst, src)
	mergeFlatPlugins(dst, src)
	mergeFlatMCPServers(dst, src)
	mergeExtensions(dst, src)
	mergeSettingsHooks(dst, src)
	mergeDetect(dst, src)

	// SkipPluginDiff: OR semantics
	if src.SkipPluginDiff {
		dst.SkipPluginDiff = true
	}

	// PostApply: last-wins
	if src.PostApply != nil {
		dst.PostApply = src.PostApply
	}
}

// mergeMarketplaces unions marketplaces, deduplicating by key.
func mergeMarketplaces(dst, src *Profile) {
	if len(src.Marketplaces) == 0 {
		return
	}

	seen := make(map[string]bool)
	for _, m := range dst.Marketplaces {
		seen[marketplaceKey(m)] = true
	}

	for _, m := range src.Marketplaces {
		key := marketplaceKey(m)
		if !seen[key] {
			dst.Marketplaces = append(dst.Marketplaces, m)
			seen[key] = true
		}
	}
}

// mergePerScope merges per-scope settings from src into dst.
func mergePerScope(dst, src *Profile) {
	if src.PerScope == nil {
		return
	}

	if dst.PerScope == nil {
		dst.PerScope = &PerScopeSettings{}
	}

	if src.PerScope.User != nil {
		if dst.PerScope.User == nil {
			dst.PerScope.User = &ScopeSettings{}
		}
		mergeScopeSettings(dst.PerScope.User, src.PerScope.User)
	}

	if src.PerScope.Project != nil {
		if dst.PerScope.Project == nil {
			dst.PerScope.Project = &ScopeSettings{}
		}
		mergeScopeSettings(dst.PerScope.Project, src.PerScope.Project)
	}

	if src.PerScope.Local != nil {
		if dst.PerScope.Local == nil {
			dst.PerScope.Local = &ScopeSettings{}
		}
		mergeScopeSettings(dst.PerScope.Local, src.PerScope.Local)
	}
}

// mergeScopeSettings merges plugins (union, dedup), MCP servers (last-wins by name),
// and extensions (union, dedup per category).
func mergeScopeSettings(dst, src *ScopeSettings) {
	dst.Plugins = mergeStringSlice(dst.Plugins, src.Plugins)

	if len(src.MCPServers) > 0 {
		serverMap := make(map[string]int) // name -> index in dst
		for i, s := range dst.MCPServers {
			serverMap[s.Name] = i
		}
		for _, s := range src.MCPServers {
			if idx, ok := serverMap[s.Name]; ok {
				dst.MCPServers[idx] = s // last-wins
			} else {
				serverMap[s.Name] = len(dst.MCPServers)
				dst.MCPServers = append(dst.MCPServers, s)
			}
		}
	}

	if src.Extensions != nil {
		if dst.Extensions == nil {
			dst.Extensions = &ExtensionSettings{}
		}
		dst.Extensions.Agents = mergeStringSlice(dst.Extensions.Agents, src.Extensions.Agents)
		dst.Extensions.Commands = mergeStringSlice(dst.Extensions.Commands, src.Extensions.Commands)
		dst.Extensions.Skills = mergeStringSlice(dst.Extensions.Skills, src.Extensions.Skills)
		dst.Extensions.Hooks = mergeStringSlice(dst.Extensions.Hooks, src.Extensions.Hooks)
		dst.Extensions.Rules = mergeStringSlice(dst.Extensions.Rules, src.Extensions.Rules)
		dst.Extensions.OutputStyles = mergeStringSlice(dst.Extensions.OutputStyles, src.Extensions.OutputStyles)
	}
}

// mergeFlatPlugins unions legacy flat plugins with dedup.
func mergeFlatPlugins(dst, src *Profile) {
	dst.Plugins = mergeStringSlice(dst.Plugins, src.Plugins)
}

// mergeFlatMCPServers unions legacy flat MCP servers with last-wins by name.
func mergeFlatMCPServers(dst, src *Profile) {
	if len(src.MCPServers) == 0 {
		return
	}

	serverMap := make(map[string]int)
	for i, s := range dst.MCPServers {
		serverMap[s.Name] = i
	}
	for _, s := range src.MCPServers {
		if idx, ok := serverMap[s.Name]; ok {
			dst.MCPServers[idx] = s
		} else {
			serverMap[s.Name] = len(dst.MCPServers)
			dst.MCPServers = append(dst.MCPServers, s)
		}
	}
}

// mergeExtensions unions extensions per category with dedup.
func mergeExtensions(dst, src *Profile) {
	if src.Extensions == nil {
		return
	}

	if dst.Extensions == nil {
		dst.Extensions = &ExtensionSettings{}
	}

	dst.Extensions.Agents = mergeStringSlice(dst.Extensions.Agents, src.Extensions.Agents)
	dst.Extensions.Commands = mergeStringSlice(dst.Extensions.Commands, src.Extensions.Commands)
	dst.Extensions.Skills = mergeStringSlice(dst.Extensions.Skills, src.Extensions.Skills)
	dst.Extensions.Hooks = mergeStringSlice(dst.Extensions.Hooks, src.Extensions.Hooks)
	dst.Extensions.Rules = mergeStringSlice(dst.Extensions.Rules, src.Extensions.Rules)
	dst.Extensions.OutputStyles = mergeStringSlice(dst.Extensions.OutputStyles, src.Extensions.OutputStyles)
}

// mergeSettingsHooks unions hooks per event type, deduplicating by command.
func mergeSettingsHooks(dst, src *Profile) {
	if len(src.SettingsHooks) == 0 {
		return
	}

	if dst.SettingsHooks == nil {
		dst.SettingsHooks = make(map[string][]HookEntry)
	}

	for event, srcHooks := range src.SettingsHooks {
		existing := dst.SettingsHooks[event]
		seen := make(map[string]bool)
		for _, h := range existing {
			seen[h.Command] = true
		}
		for _, h := range srcHooks {
			if !seen[h.Command] {
				existing = append(existing, h)
				seen[h.Command] = true
			}
		}
		dst.SettingsHooks[event] = existing
	}
}

// mergeDetect unions detect files and merges contains map (later wins).
func mergeDetect(dst, src *Profile) {
	dst.Detect.Files = mergeStringSlice(dst.Detect.Files, src.Detect.Files)

	if len(src.Detect.Contains) > 0 {
		if dst.Detect.Contains == nil {
			dst.Detect.Contains = make(map[string]string)
		}
		for k, v := range src.Detect.Contains {
			dst.Detect.Contains[k] = v
		}
	}
}

// mergeStringSlice returns a union of two string slices, preserving order and deduplicating.
func mergeStringSlice(dst, src []string) []string {
	if len(src) == 0 {
		return dst
	}

	seen := make(map[string]bool, len(dst))
	for _, s := range dst {
		seen[s] = true
	}

	for _, s := range src {
		if !seen[s] {
			dst = append(dst, s)
			seen[s] = true
		}
	}

	return dst
}
