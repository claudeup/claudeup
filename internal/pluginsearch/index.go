// ABOUTME: Data types for plugin search indexing
// ABOUTME: Defines PluginSearchIndex and ComponentInfo structures

package pluginsearch

// ComponentInfo represents a skill, command, or agent within a plugin.
type ComponentInfo struct {
	Name        string
	Description string
	Path        string
	Content     string // SKILL.md body content (after frontmatter)
	Truncated   bool   // true when content was truncated at the byte limit
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
