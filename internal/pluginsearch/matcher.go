// ABOUTME: Performs search operations across plugin indices
// ABOUTME: Supports substring matching, regex, and filtering by type/marketplace

package pluginsearch

import (
	"regexp"
	"strings"
)

// SearchOptions configures search behavior.
type SearchOptions struct {
	UseRegex      bool
	FilterType    string // "skills", "commands", "agents", or "" for all
	FilterMarket  string // Filter to specific marketplace
	SearchContent bool   // Also search SKILL.md body content (future feature)
}

// Match represents a single match within a plugin.
type Match struct {
	Type        string // "name", "description", "keyword", "skill", "command", "agent"
	Name        string // Component name if applicable
	Description string // Component description if applicable
	Context     string // The matched text
	Path        string // Component path if applicable
}

// SearchResult represents a plugin with its matches.
type SearchResult struct {
	Plugin  PluginSearchIndex
	Matches []Match
}

// Matcher performs searches across plugin indices.
type Matcher struct{}

// NewMatcher creates a new Matcher.
func NewMatcher() *Matcher {
	return &Matcher{}
}

// Search searches across all plugins and returns matching results.
func (m *Matcher) Search(plugins []PluginSearchIndex, query string, opts SearchOptions) []SearchResult {
	if query == "" {
		return nil
	}

	var matchFunc func(text string) bool

	if opts.UseRegex {
		// Compile regex with case-insensitive flag
		re, err := regexp.Compile("(?i)" + query)
		if err != nil {
			return nil // Invalid regex returns no results
		}
		matchFunc = re.MatchString
	} else {
		// Default: case-insensitive substring match
		lowerQuery := strings.ToLower(query)
		matchFunc = func(text string) bool {
			return strings.Contains(strings.ToLower(text), lowerQuery)
		}
	}

	var results []SearchResult

	for _, plugin := range plugins {
		// Filter by marketplace if specified
		if opts.FilterMarket != "" && plugin.Marketplace != opts.FilterMarket {
			continue
		}

		matches := m.matchPlugin(plugin, matchFunc, opts)
		if len(matches) > 0 {
			results = append(results, SearchResult{
				Plugin:  plugin,
				Matches: matches,
			})
		}
	}

	return results
}

func (m *Matcher) matchPlugin(plugin PluginSearchIndex, matchFunc func(string) bool, opts SearchOptions) []Match {
	var matches []Match

	// Match plugin-level fields (only when not filtering by specific type)
	if opts.FilterType == "" {
		if matchFunc(plugin.Name) {
			matches = append(matches, Match{
				Type:    "name",
				Context: plugin.Name,
			})
		}
		if matchFunc(plugin.Description) {
			matches = append(matches, Match{
				Type:    "description",
				Context: plugin.Description,
			})
		}
		for _, keyword := range plugin.Keywords {
			if matchFunc(keyword) {
				matches = append(matches, Match{
					Type:    "keyword",
					Context: keyword,
				})
			}
		}
	}

	// Match skills (if not filtering or filtering by skills)
	if opts.FilterType == "" || opts.FilterType == "skills" {
		for _, skill := range plugin.Skills {
			if matchFunc(skill.Name) || matchFunc(skill.Description) {
				matches = append(matches, Match{
					Type:        "skill",
					Name:        skill.Name,
					Description: skill.Description,
					Context:     skill.Name,
					Path:        skill.Path,
				})
			} else if opts.SearchContent && skill.Content != "" && matchFunc(skill.Content) {
				matches = append(matches, Match{
					Type:        "content",
					Name:        skill.Name,
					Description: skill.Description,
					Context:     skill.Content,
					Path:        skill.Path,
				})
			}
		}
	}

	// Match commands (if not filtering or filtering by commands)
	if opts.FilterType == "" || opts.FilterType == "commands" {
		for _, cmd := range plugin.Commands {
			if matchFunc(cmd.Name) {
				matches = append(matches, Match{
					Type:    "command",
					Name:    cmd.Name,
					Context: cmd.Name,
					Path:    cmd.Path,
				})
			}
		}
	}

	// Match agents (if not filtering or filtering by agents)
	if opts.FilterType == "" || opts.FilterType == "agents" {
		for _, agent := range plugin.Agents {
			if matchFunc(agent.Name) {
				matches = append(matches, Match{
					Type:    "agent",
					Name:    agent.Name,
					Context: agent.Name,
					Path:    agent.Path,
				})
			}
		}
	}

	return matches
}
