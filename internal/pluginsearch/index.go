// ABOUTME: Data types for plugin search indexing
// ABOUTME: Defines PluginSearchIndex and ComponentInfo structures

package pluginsearch

// ComponentInfo represents a skill, command, or agent within a plugin.
type ComponentInfo struct {
	Name        string
	Description string
	Path        string
}

// PluginSearchIndex holds searchable metadata for a single plugin.
type PluginSearchIndex struct {
	Name        string
	Description string
	Keywords    []string
	Marketplace string
	Version     string
	Path        string

	Skills   []ComponentInfo
	Commands []ComponentInfo
	Agents   []ComponentInfo
}

// HasSkills returns true if the plugin has any skills.
func (p *PluginSearchIndex) HasSkills() bool {
	return len(p.Skills) > 0
}

// HasCommands returns true if the plugin has any commands.
func (p *PluginSearchIndex) HasCommands() bool {
	return len(p.Commands) > 0
}

// HasAgents returns true if the plugin has any agents.
func (p *PluginSearchIndex) HasAgents() bool {
	return len(p.Agents) > 0
}
