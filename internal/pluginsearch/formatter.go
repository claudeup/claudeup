// ABOUTME: Renders plugin search results to terminal output
// ABOUTME: Supports default, table, JSON formats and by-component grouping

package pluginsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/claudeup/claudeup/v2/internal/ui"
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
	fmt.Fprintln(f.w, ui.RenderSection(fmt.Sprintf("Search results for %q", query), totalMatches))
	fmt.Fprintf(f.w, "%s\n\n", ui.Muted(fmt.Sprintf("%d plugins", len(results))))

	for _, result := range results {
		f.renderPluginResult(result)
		fmt.Fprintln(f.w)
	}
}

// renderPluginResult outputs a single plugin's results.
func (f *Formatter) renderPluginResult(result SearchResult) {
	// Plugin header: name@marketplace (version) - styled like plugin show
	fullName := fmt.Sprintf("%s@%s", result.Plugin.Name, result.Plugin.Marketplace)
	fmt.Fprintf(f.w, "%s %s\n",
		ui.Bold(fullName),
		ui.Muted(fmt.Sprintf("(v%s)", result.Plugin.Version)))

	// Group matches by type for display
	skills := f.filterMatchesByType(result.Matches, "skill")
	commands := f.filterMatchesByType(result.Matches, "command")
	agents := f.filterMatchesByType(result.Matches, "agent")

	// Display component summaries with styling
	if len(skills) > 0 {
		names := f.extractMatchNames(skills)
		fmt.Fprintf(f.w, "  %s %s\n", ui.Info("Skills:"), joinNames(names))
	}
	if len(commands) > 0 {
		names := f.extractMatchNames(commands)
		fmt.Fprintf(f.w, "  %s %s\n", ui.Info("Commands:"), joinNames(names))
	}
	if len(agents) > 0 {
		names := f.extractMatchNames(agents)
		fmt.Fprintf(f.w, "  %s %s\n", ui.Info("Agents:"), joinNames(names))
	}

	// Show first match context with description and path
	for _, match := range result.Matches {
		if match.Description != "" {
			fmt.Fprintf(f.w, "  %s %s\n", ui.Muted(ui.SymbolArrow), ui.Muted(match.Description))
		}
		if match.Path != "" {
			fmt.Fprintf(f.w, "  %s %s\n", ui.Muted("Path:"), ui.Muted(shortenPath(match.Path)))
		}
		// Only show first match's details
		if match.Description != "" || match.Path != "" {
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
	fmt.Fprintln(f.w, ui.RenderSection(fmt.Sprintf("Search results for %q", query), totalMatches))
	fmt.Fprintf(f.w, "%s\n\n", ui.Muted(fmt.Sprintf("%d plugins", len(results))))

	// Collect all components by type
	type componentEntry struct {
		name        string
		description string
		path        string
		plugin      string
		marketplace string
	}

	var skills, commands, agents []componentEntry

	for _, result := range results {
		for _, match := range result.Matches {
			entry := componentEntry{
				name:        match.Name,
				description: match.Description,
				path:        match.Path,
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
		fmt.Fprintln(f.w, ui.RenderSection("Skills", len(skills)))
		for _, s := range skills {
			fmt.Fprintf(f.w, "  %s %s\n", ui.Bold(s.name), ui.Muted(fmt.Sprintf("(%s@%s)", s.plugin, s.marketplace)))
			if s.description != "" {
				fmt.Fprintf(f.w, "    %s\n", ui.Muted(s.description))
			}
			if s.path != "" {
				fmt.Fprintf(f.w, "    %s %s\n", ui.Muted("Path:"), ui.Muted(shortenPath(s.path)))
			}
		}
		fmt.Fprintln(f.w)
	}

	// Output commands section
	if len(commands) > 0 {
		fmt.Fprintln(f.w, ui.RenderSection("Commands", len(commands)))
		for _, c := range commands {
			fmt.Fprintf(f.w, "  %s %s\n", ui.Bold(c.name), ui.Muted(fmt.Sprintf("(%s@%s)", c.plugin, c.marketplace)))
			if c.description != "" {
				fmt.Fprintf(f.w, "    %s\n", ui.Muted(c.description))
			}
			if c.path != "" {
				fmt.Fprintf(f.w, "    %s %s\n", ui.Muted("Path:"), ui.Muted(shortenPath(c.path)))
			}
		}
		fmt.Fprintln(f.w)
	}

	// Output agents section
	if len(agents) > 0 {
		fmt.Fprintln(f.w, ui.RenderSection("Agents", len(agents)))
		for _, a := range agents {
			fmt.Fprintf(f.w, "  %s %s\n", ui.Bold(a.name), ui.Muted(fmt.Sprintf("(%s@%s)", a.plugin, a.marketplace)))
			if a.description != "" {
				fmt.Fprintf(f.w, "    %s\n", ui.Muted(a.description))
			}
			if a.path != "" {
				fmt.Fprintf(f.w, "    %s %s\n", ui.Muted("Path:"), ui.Muted(shortenPath(a.path)))
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
	fmt.Fprintln(f.w, ui.RenderSection(fmt.Sprintf("Search results for %q", query), totalMatches))
	fmt.Fprintf(f.w, "%s\n\n", ui.Muted(fmt.Sprintf("%d plugins", len(results))))

	// Table header with styling
	header := fmt.Sprintf("%-30s %-10s %-30s %s", "PLUGIN", "TYPE", "COMPONENT", "DESCRIPTION")
	fmt.Fprintln(f.w, ui.Bold(header))
	fmt.Fprintln(f.w, ui.Muted(strings.Repeat("─", 90)))

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
			fmt.Fprintf(f.w, "%s %s %s %s\n",
				ui.Bold(fmt.Sprintf("%-30s", truncate(pluginID, 30))),
				ui.Info(fmt.Sprintf("%-10s", match.Type)),
				fmt.Sprintf("%-30s", truncate(name, 30)),
				ui.Muted(desc))
		}
	}
}

// renderJSON outputs JSON format.
func (f *Formatter) renderJSON(results []SearchResult, query string) {
	type jsonMatch struct {
		Type        string `json:"type"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		Path        string `json:"path,omitempty"`
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
				Path:        match.Path,
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
	fmt.Fprintf(f.w, "%s %s\n\n", ui.Warning(ui.SymbolWarning), fmt.Sprintf("No results for %q", query))
	fmt.Fprintln(f.w, ui.Muted("Try:"))
	fmt.Fprintln(f.w, ui.Muted("  - Broaden your search term"))
	fmt.Fprintln(f.w, ui.Muted("  - Use --all to search all cached plugins"))
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

// shortenPath replaces the home directory with ~ for display.
func shortenPath(path string) string {
	if path == "" {
		return ""
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}
	return path
}
