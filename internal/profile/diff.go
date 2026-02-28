// ABOUTME: Diff logic for comparing a saved profile against live Claude Code state
// ABOUTME: Types and functions for scope-aware profile comparison
package profile

import "strings"

// DiffOp represents the type of difference
type DiffOp string

const (
	DiffAdded    DiffOp = "added"
	DiffRemoved  DiffOp = "removed"
	DiffModified DiffOp = "modified"
)

// DiffItemKind represents what type of config item differs
type DiffItemKind string

const (
	DiffPlugin      DiffItemKind = "plugin"
	DiffMCP         DiffItemKind = "mcp"
	DiffExtension   DiffItemKind = "extension"
	DiffMarketplace DiffItemKind = "marketplace"
)

// DiffItem represents a single difference
type DiffItem struct {
	Op     DiffOp
	Kind   DiffItemKind
	Name   string
	Detail string // optional context (e.g., extension category, changed MCP field)
}

// ScopeDiff contains all differences for a single scope
type ScopeDiff struct {
	Scope string // "user", "project", "local"
	Items []DiffItem
}

// ProfileDiff contains the full diff result
type ProfileDiff struct {
	ProfileName       string
	DescriptionChange *[2]string // [profile, live] if different
	Scopes            []ScopeDiff
}

// IsEmpty returns true if there are no differences
func (d *ProfileDiff) IsEmpty() bool {
	if d.DescriptionChange != nil {
		return false
	}
	for _, sd := range d.Scopes {
		if len(sd.Items) > 0 {
			return false
		}
	}
	return true
}

// Counts returns the number of additions, removals, and modifications across all scopes
func (d *ProfileDiff) Counts() (added, removed, modified int) {
	for _, sd := range d.Scopes {
		for _, item := range sd.Items {
			switch item.Op {
			case DiffAdded:
				added++
			case DiffRemoved:
				removed++
			case DiffModified:
				modified++
			}
		}
	}
	return
}

// AsPerScope returns a copy of the profile with flat fields lifted into PerScope.User.
// If PerScope is already set, returns a shallow copy as-is.
func (p *Profile) AsPerScope() *Profile {
	if p == nil {
		return &Profile{PerScope: &PerScopeSettings{}}
	}

	result := &Profile{
		Name:         p.Name,
		Description:  p.Description,
		Marketplaces: p.Marketplaces,
	}

	if p.PerScope != nil {
		result.PerScope = &PerScopeSettings{
			User:    p.PerScope.User,
			Project: p.PerScope.Project,
			Local:   p.PerScope.Local,
		}
		return result
	}

	// Lift flat fields into user scope
	result.PerScope = &PerScopeSettings{
		User: &ScopeSettings{
			Plugins:    p.Plugins,
			MCPServers: p.MCPServers,
			Extensions: p.Extensions,
		},
	}

	return result
}

// FilterToScopes returns a copy of the profile containing only the scopes
// present in the given map (keyed by scope name). Marketplaces are filtered
// to only those referenced by plugins in the retained scopes.
func FilterToScopes(p *Profile, scopes map[string]bool) *Profile {
	if p == nil {
		return nil
	}
	result := &Profile{
		Name:        p.Name,
		Description: p.Description,
		PerScope:    &PerScopeSettings{},
	}

	var allPlugins []string
	if scopes["user"] && p.PerScope != nil && p.PerScope.User != nil {
		result.PerScope.User = p.PerScope.User
		allPlugins = append(allPlugins, p.PerScope.User.Plugins...)
	}
	if scopes["project"] && p.PerScope != nil && p.PerScope.Project != nil {
		result.PerScope.Project = p.PerScope.Project
		allPlugins = append(allPlugins, p.PerScope.Project.Plugins...)
	}
	if scopes["local"] && p.PerScope != nil && p.PerScope.Local != nil {
		result.PerScope.Local = p.PerScope.Local
		allPlugins = append(allPlugins, p.PerScope.Local.Plugins...)
	}

	// Only keep marketplaces referenced by plugins in retained scopes
	pluginMarketplaces := make(map[string]bool)
	for _, plugin := range allPlugins {
		parts := strings.SplitN(plugin, "@", 2)
		if len(parts) == 2 && parts[1] != "" {
			pluginMarketplaces[parts[1]] = true
		}
	}
	for _, m := range p.Marketplaces {
		name := m.DisplayName()
		// Match by repo suffix (marketplace key in plugin is the repo name part)
		keep := false
		for key := range pluginMarketplaces {
			if key == name || strings.HasSuffix(name, "/"+key) {
				keep = true
				break
			}
		}
		if keep {
			result.Marketplaces = append(result.Marketplaces, m)
		}
	}

	return result
}

// ComputeProfileDiff compares a saved profile against a live snapshot.
// Both inputs should already be in PerScope form (caller normalizes via AsPerScope).
func ComputeProfileDiff(saved, live *Profile) *ProfileDiff {
	diff := &ProfileDiff{
		ProfileName: saved.Name,
	}

	// Compare description
	if saved.Description != live.Description {
		diff.DescriptionChange = &[2]string{saved.Description, live.Description}
	}

	// Compare each scope
	for _, scope := range []string{"user", "project", "local"} {
		savedScope := getScopeSettings(saved, scope)
		liveScope := getScopeSettings(live, scope)

		items := diffScope(savedScope, liveScope)

		// Marketplaces are always user-scoped
		if scope == "user" {
			items = append(items, diffMarketplaces(saved.Marketplaces, live.Marketplaces)...)
		}

		if len(items) > 0 {
			diff.Scopes = append(diff.Scopes, ScopeDiff{
				Scope: scope,
				Items: items,
			})
		}
	}

	return diff
}

// getScopeSettings returns the ScopeSettings for a given scope, or nil if not set
func getScopeSettings(p *Profile, scope string) *ScopeSettings {
	if p == nil || p.PerScope == nil {
		return nil
	}
	switch scope {
	case "user":
		return p.PerScope.User
	case "project":
		return p.PerScope.Project
	case "local":
		return p.PerScope.Local
	}
	return nil
}

// diffScope compares two scope settings and returns diff items
func diffScope(saved, live *ScopeSettings) []DiffItem {
	var items []DiffItem

	savedPlugins := scopePlugins(saved)
	livePlugins := scopePlugins(live)
	items = append(items, diffStringSet(savedPlugins, livePlugins, DiffPlugin)...)

	items = append(items, diffMCPServers(scopeMCPServers(saved), scopeMCPServers(live))...)

	items = append(items, diffExtensions(scopeExtensions(saved), scopeExtensions(live))...)

	return items
}

func scopePlugins(s *ScopeSettings) []string {
	if s == nil {
		return nil
	}
	return s.Plugins
}

func scopeMCPServers(s *ScopeSettings) []MCPServer {
	if s == nil {
		return nil
	}
	return s.MCPServers
}

func scopeExtensions(s *ScopeSettings) *ExtensionSettings {
	if s == nil {
		return nil
	}
	return s.Extensions
}

// diffStringSet computes added/removed items between two string slices
func diffStringSet(saved, live []string, kind DiffItemKind) []DiffItem {
	savedSet := make(map[string]bool, len(saved))
	for _, s := range saved {
		savedSet[s] = true
	}
	liveSet := make(map[string]bool, len(live))
	for _, s := range live {
		liveSet[s] = true
	}

	var items []DiffItem

	// Added: in live but not in saved
	for _, s := range live {
		if !savedSet[s] {
			items = append(items, DiffItem{Op: DiffAdded, Kind: kind, Name: s})
		}
	}

	// Removed: in saved but not in live
	for _, s := range saved {
		if !liveSet[s] {
			items = append(items, DiffItem{Op: DiffRemoved, Kind: kind, Name: s})
		}
	}

	return items
}

// diffMCPServers computes added/removed/modified MCP servers
func diffMCPServers(saved, live []MCPServer) []DiffItem {
	savedMap := make(map[string]MCPServer, len(saved))
	for _, s := range saved {
		savedMap[s.Name] = s
	}
	liveMap := make(map[string]MCPServer, len(live))
	for _, s := range live {
		liveMap[s.Name] = s
	}

	var items []DiffItem

	// Added or modified: in live
	for _, l := range live {
		s, exists := savedMap[l.Name]
		if !exists {
			items = append(items, DiffItem{Op: DiffAdded, Kind: DiffMCP, Name: l.Name})
		} else if !mcpServersEqual(s, l) {
			items = append(items, DiffItem{Op: DiffModified, Kind: DiffMCP, Name: l.Name, Detail: mcpDiffDetail(s, l)})
		}
	}

	// Removed: in saved but not in live
	for _, s := range saved {
		if _, exists := liveMap[s.Name]; !exists {
			items = append(items, DiffItem{Op: DiffRemoved, Kind: DiffMCP, Name: s.Name})
		}
	}

	return items
}

// mcpDiffDetail returns a summary of what changed between two MCP servers
func mcpDiffDetail(saved, live MCPServer) string {
	var changes []string
	if saved.Command != live.Command {
		changes = append(changes, "command")
	}
	if len(saved.Args) != len(live.Args) {
		changes = append(changes, "args")
	} else {
		for i := range saved.Args {
			if saved.Args[i] != live.Args[i] {
				changes = append(changes, "args")
				break
			}
		}
	}
	if saved.Scope != live.Scope {
		changes = append(changes, "scope")
	}
	if len(changes) == 0 {
		return "config changed"
	}
	return strings.Join(changes, ", ") + " changed"
}

// diffExtensions computes added/removed extensions per category
func diffExtensions(saved, live *ExtensionSettings) []DiffItem {
	var items []DiffItem

	categories := []struct {
		name       string
		savedItems []string
		liveItems  []string
	}{
		{"agents", extSlice(saved, "agents"), extSlice(live, "agents")},
		{"commands", extSlice(saved, "commands"), extSlice(live, "commands")},
		{"skills", extSlice(saved, "skills"), extSlice(live, "skills")},
		{"hooks", extSlice(saved, "hooks"), extSlice(live, "hooks")},
		{"rules", extSlice(saved, "rules"), extSlice(live, "rules")},
		{"output-styles", extSlice(saved, "output-styles"), extSlice(live, "output-styles")},
	}

	for _, cat := range categories {
		for _, d := range diffStringSet(cat.savedItems, cat.liveItems, DiffExtension) {
			d.Detail = cat.name
			items = append(items, d)
		}
	}

	return items
}

// extSlice returns the extension items for a given category
func extSlice(e *ExtensionSettings, category string) []string {
	if e == nil {
		return nil
	}
	switch category {
	case "agents":
		return e.Agents
	case "commands":
		return e.Commands
	case "skills":
		return e.Skills
	case "hooks":
		return e.Hooks
	case "rules":
		return e.Rules
	case "output-styles":
		return e.OutputStyles
	}
	return nil
}

// diffMarketplaces computes added/removed marketplaces
func diffMarketplaces(saved, live []Marketplace) []DiffItem {
	savedSet := make(map[string]bool, len(saved))
	for _, m := range saved {
		savedSet[m.DisplayName()] = true
	}
	liveSet := make(map[string]bool, len(live))
	for _, m := range live {
		liveSet[m.DisplayName()] = true
	}

	var items []DiffItem

	for _, m := range live {
		name := m.DisplayName()
		if !savedSet[name] {
			items = append(items, DiffItem{Op: DiffAdded, Kind: DiffMarketplace, Name: name})
		}
	}

	for _, m := range saved {
		name := m.DisplayName()
		if !liveSet[name] {
			items = append(items, DiffItem{Op: DiffRemoved, Kind: DiffMarketplace, Name: name})
		}
	}

	return items
}
