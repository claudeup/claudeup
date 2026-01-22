// ABOUTME: Renders plugin search results to terminal output
// ABOUTME: Supports default, table, JSON formats and by-component grouping

package pluginsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// FormatOptions configures output rendering.
type FormatOptions struct {
	Format      string // "default", "table", "json"
	ByComponent bool   // Group by component type instead of plugin
}

// Formatter renders search results to an io.Writer.
type Formatter struct {
	w io.Writer
}

// NewFormatter creates a new Formatter that writes to w.
func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{w: w}
}

// Render outputs search results according to the specified options.
func (f *Formatter) Render(results []SearchResult, query string, opts FormatOptions) {
	switch opts.Format {
	case "json":
		f.renderJSON(results, query)
	case "table":
		f.renderTable(results, query, opts)
	default:
		if opts.ByComponent {
			f.renderByComponent(results, query)
		} else {
			f.renderDefault(results, query)
		}
	}
}

// renderDefault outputs plugin-centric format.
func (f *Formatter) renderDefault(results []SearchResult, query string) {
	if len(results) == 0 {
		f.renderNoResults(query)
		return
	}

	totalMatches := f.countMatches(results)
	fmt.Fprintf(f.w, "Search results for \"%s\" (%d plugins, %d matches)\n\n",
		query, len(results), totalMatches)

	for _, result := range results {
		f.renderPluginResult(result)
		fmt.Fprintln(f.w)
	}
}

// renderPluginResult outputs a single plugin's results.
func (f *Formatter) renderPluginResult(result SearchResult) {
	// Plugin header: name@marketplace (version)
	fmt.Fprintf(f.w, "%s@%s (v%s)\n",
		result.Plugin.Name,
		result.Plugin.Marketplace,
		result.Plugin.Version)

	// Group matches by type for display
	skills := f.filterMatchesByType(result.Matches, "skill")
	commands := f.filterMatchesByType(result.Matches, "command")
	agents := f.filterMatchesByType(result.Matches, "agent")

	// Display component summaries
	if len(skills) > 0 {
		names := f.extractMatchNames(skills)
		fmt.Fprintf(f.w, "  Skills: %s\n", joinNames(names))
	}
	if len(commands) > 0 {
		names := f.extractMatchNames(commands)
		fmt.Fprintf(f.w, "  Commands: %s\n", joinNames(names))
	}
	if len(agents) > 0 {
		names := f.extractMatchNames(agents)
		fmt.Fprintf(f.w, "  Agents: %s\n", joinNames(names))
	}

	// Show first match context
	for _, match := range result.Matches {
		if match.Description != "" {
			fmt.Fprintf(f.w, "  Match: \"%s\" - %s\n", match.Name, match.Description)
			break
		}
	}
}

// renderByComponent outputs component-centric format grouped by type.
func (f *Formatter) renderByComponent(results []SearchResult, query string) {
	if len(results) == 0 {
		f.renderNoResults(query)
		return
	}

	totalMatches := f.countMatches(results)
	fmt.Fprintf(f.w, "Search results for \"%s\" (%d components across %d plugins)\n\n",
		query, totalMatches, len(results))

	// Collect all components by type
	type componentEntry struct {
		name        string
		description string
		plugin      string
		marketplace string
	}

	var skills, commands, agents []componentEntry

	for _, result := range results {
		for _, match := range result.Matches {
			entry := componentEntry{
				name:        match.Name,
				description: match.Description,
				plugin:      result.Plugin.Name,
				marketplace: result.Plugin.Marketplace,
			}
			switch match.Type {
			case "skill":
				skills = append(skills, entry)
			case "command":
				commands = append(commands, entry)
			case "agent":
				agents = append(agents, entry)
			}
		}
	}

	// Output skills section
	if len(skills) > 0 {
		fmt.Fprintln(f.w, "Skills:")
		for _, s := range skills {
			fmt.Fprintf(f.w, "  %s (%s@%s)\n", s.name, s.plugin, s.marketplace)
			if s.description != "" {
				fmt.Fprintf(f.w, "    %s\n", s.description)
			}
		}
		fmt.Fprintln(f.w)
	}

	// Output commands section
	if len(commands) > 0 {
		fmt.Fprintln(f.w, "Commands:")
		for _, c := range commands {
			fmt.Fprintf(f.w, "  %s (%s@%s)\n", c.name, c.plugin, c.marketplace)
			if c.description != "" {
				fmt.Fprintf(f.w, "    %s\n", c.description)
			}
		}
		fmt.Fprintln(f.w)
	}

	// Output agents section
	if len(agents) > 0 {
		fmt.Fprintln(f.w, "Agents:")
		for _, a := range agents {
			fmt.Fprintf(f.w, "  %s (%s@%s)\n", a.name, a.plugin, a.marketplace)
			if a.description != "" {
				fmt.Fprintf(f.w, "    %s\n", a.description)
			}
		}
		fmt.Fprintln(f.w)
	}
}

// renderTable outputs tabular format.
func (f *Formatter) renderTable(results []SearchResult, query string, opts FormatOptions) {
	if len(results) == 0 {
		f.renderNoResults(query)
		return
	}

	totalMatches := f.countMatches(results)
	fmt.Fprintf(f.w, "Search results for \"%s\" (%d plugins, %d matches)\n\n",
		query, len(results), totalMatches)

	// Simple table: PLUGIN | TYPE | COMPONENT | DESCRIPTION
	fmt.Fprintf(f.w, "%-30s %-10s %-30s %s\n", "PLUGIN", "TYPE", "COMPONENT", "DESCRIPTION")
	fmt.Fprintf(f.w, "%-30s %-10s %-30s %s\n", "------", "----", "---------", "-----------")

	for _, result := range results {
		pluginID := fmt.Sprintf("%s@%s", result.Plugin.Name, result.Plugin.Marketplace)
		for _, match := range result.Matches {
			name := match.Name
			if name == "" {
				name = match.Context
			}
			desc := match.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			fmt.Fprintf(f.w, "%-30s %-10s %-30s %s\n",
				truncate(pluginID, 30),
				match.Type,
				truncate(name, 30),
				desc)
		}
	}
}

// renderJSON outputs JSON format.
func (f *Formatter) renderJSON(results []SearchResult, query string) {
	type jsonMatch struct {
		Type        string `json:"type"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}

	type jsonResult struct {
		Plugin      string      `json:"plugin"`
		Marketplace string      `json:"marketplace"`
		Version     string      `json:"version"`
		Matches     []jsonMatch `json:"matches"`
	}

	type jsonOutput struct {
		Query        string       `json:"query"`
		TotalPlugins int          `json:"totalPlugins"`
		TotalMatches int          `json:"totalMatches"`
		Results      []jsonResult `json:"results"`
	}

	output := jsonOutput{
		Query:        query,
		TotalPlugins: len(results),
		TotalMatches: f.countMatches(results),
		Results:      make([]jsonResult, 0, len(results)),
	}

	for _, result := range results {
		jr := jsonResult{
			Plugin:      result.Plugin.Name,
			Marketplace: result.Plugin.Marketplace,
			Version:     result.Plugin.Version,
			Matches:     make([]jsonMatch, 0, len(result.Matches)),
		}
		for _, match := range result.Matches {
			jr.Matches = append(jr.Matches, jsonMatch{
				Type:        match.Type,
				Name:        match.Name,
				Description: match.Description,
			})
		}
		output.Results = append(output.Results, jr)
	}

	enc := json.NewEncoder(f.w)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}

// renderNoResults outputs a helpful message when no results found.
func (f *Formatter) renderNoResults(query string) {
	fmt.Fprintf(f.w, "No results for \"%s\"\n\n", query)
	fmt.Fprintln(f.w, "Try:")
	fmt.Fprintln(f.w, "  - Broaden your search term")
	fmt.Fprintln(f.w, "  - Use --all to search all cached plugins")
}

// Helper methods

func (f *Formatter) countMatches(results []SearchResult) int {
	total := 0
	for _, r := range results {
		total += len(r.Matches)
	}
	return total
}

func (f *Formatter) filterMatchesByType(matches []Match, matchType string) []Match {
	var filtered []Match
	for _, m := range matches {
		if m.Type == matchType {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (f *Formatter) extractMatchNames(matches []Match) []string {
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		if m.Name != "" {
			names = append(names, m.Name)
		}
	}
	return names
}

func joinNames(names []string) string {
	return strings.Join(names, ", ")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
