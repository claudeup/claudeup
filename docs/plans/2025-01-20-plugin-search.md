# Plugin Search Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `claudeup plugin search <query>` command to search across marketplace plugins by capability.

**Architecture:** New `internal/pluginsearch/` package handles indexing, matching, and formatting. Command in `internal/commands/plugin_search.go` integrates with existing plugin infrastructure. Scanner walks `~/.claude/plugins/cache/`, parses plugin.json and SKILL.md frontmatter.

**Tech Stack:** Go, Cobra CLI, YAML parsing (gopkg.in/yaml.v3), regexp for regex mode

---

## Task 1: Create PluginSearchIndex Types

**Files:**
- Create: `internal/pluginsearch/index.go`
- Test: `internal/pluginsearch/index_test.go`

**Step 1: Write the failing test**

```go
// internal/pluginsearch/index_test.go
// ABOUTME: Unit tests for plugin search index types
// ABOUTME: Tests data structures used for indexing plugins

package pluginsearch_test

import (
	"testing"

	"github.com/claudeup/claudeup/v2/internal/pluginsearch"
	"github.com/stretchr/testify/assert"
)

func TestComponentInfo_String(t *testing.T) {
	c := pluginsearch.ComponentInfo{
		Name:        "test-skill",
		Description: "A test skill",
		Path:        "/path/to/skill",
	}
	assert.Equal(t, "test-skill", c.Name)
	assert.Equal(t, "A test skill", c.Description)
}

func TestPluginSearchIndex_HasComponents(t *testing.T) {
	p := pluginsearch.PluginSearchIndex{
		Name:        "test-plugin",
		Description: "A test plugin",
		Marketplace: "test-marketplace",
		Skills: []pluginsearch.ComponentInfo{
			{Name: "skill1"},
		},
	}
	assert.True(t, p.HasSkills())
	assert.False(t, p.HasCommands())
	assert.False(t, p.HasAgents())
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/pluginsearch/... -v`
Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

```go
// internal/pluginsearch/index.go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/pluginsearch/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/pluginsearch/
git commit -m "feat(pluginsearch): add index types for plugin search"
```

---

## Task 2: Create Scanner to Parse plugin.json

**Files:**
- Create: `internal/pluginsearch/scanner.go`
- Test: `internal/pluginsearch/scanner_test.go`
- Create: `internal/pluginsearch/testdata/cache/test-marketplace/test-plugin/1.0.0/.claude-plugin/plugin.json`

**Step 1: Create test fixture**

```bash
mkdir -p internal/pluginsearch/testdata/cache/test-marketplace/test-plugin/1.0.0/.claude-plugin
```

```json
// internal/pluginsearch/testdata/cache/test-marketplace/test-plugin/1.0.0/.claude-plugin/plugin.json
{
  "name": "test-plugin",
  "description": "A test plugin for unit tests",
  "version": "1.0.0",
  "keywords": ["testing", "example"]
}
```

**Step 2: Write the failing test**

```go
// internal/pluginsearch/scanner_test.go
// ABOUTME: Unit tests for plugin cache scanner
// ABOUTME: Tests scanning filesystem to build plugin index

package pluginsearch_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/claudeup/claudeup/v2/internal/pluginsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func TestScanner_ScanPlugin(t *testing.T) {
	scanner := pluginsearch.NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache")

	plugins, err := scanner.Scan(cacheDir)
	require.NoError(t, err)
	require.Len(t, plugins, 1)

	p := plugins[0]
	assert.Equal(t, "test-plugin", p.Name)
	assert.Equal(t, "A test plugin for unit tests", p.Description)
	assert.Equal(t, "1.0.0", p.Version)
	assert.Equal(t, "test-marketplace", p.Marketplace)
	assert.Contains(t, p.Keywords, "testing")
	assert.Contains(t, p.Keywords, "example")
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/pluginsearch/... -v -run TestScanner`
Expected: FAIL - NewScanner undefined

**Step 4: Write minimal implementation**

```go
// internal/pluginsearch/scanner.go
// ABOUTME: Scans plugin cache directory to build search index
// ABOUTME: Parses plugin.json and component directories

package pluginsearch

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Scanner scans the plugin cache to build a search index.
type Scanner struct{}

// NewScanner creates a new Scanner.
func NewScanner() *Scanner {
	return &Scanner{}
}

// pluginJSON represents the structure of .claude-plugin/plugin.json
type pluginJSON struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Keywords    []string `json:"keywords"`
}

// Scan walks the cache directory and builds an index of all plugins.
func (s *Scanner) Scan(cacheDir string) ([]PluginSearchIndex, error) {
	var plugins []PluginSearchIndex

	// Walk marketplace directories
	marketplaces, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	for _, marketplace := range marketplaces {
		if !marketplace.IsDir() {
			continue
		}
		marketplacePath := filepath.Join(cacheDir, marketplace.Name())

		// Walk plugin directories within marketplace
		pluginDirs, err := os.ReadDir(marketplacePath)
		if err != nil {
			continue
		}

		for _, pluginDir := range pluginDirs {
			if !pluginDir.IsDir() {
				continue
			}
			pluginPath := filepath.Join(marketplacePath, pluginDir.Name())

			// Walk version directories within plugin
			versions, err := os.ReadDir(pluginPath)
			if err != nil {
				continue
			}

			for _, version := range versions {
				if !version.IsDir() {
					continue
				}
				versionPath := filepath.Join(pluginPath, version.Name())

				// Parse plugin.json
				plugin, err := s.parsePlugin(versionPath, marketplace.Name())
				if err != nil {
					continue // Skip malformed plugins
				}

				plugins = append(plugins, *plugin)
			}
		}
	}

	return plugins, nil
}

func (s *Scanner) parsePlugin(pluginPath, marketplace string) (*PluginSearchIndex, error) {
	jsonPath := filepath.Join(pluginPath, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	var pj pluginJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return nil, err
	}

	return &PluginSearchIndex{
		Name:        pj.Name,
		Description: pj.Description,
		Version:     pj.Version,
		Keywords:    pj.Keywords,
		Marketplace: marketplace,
		Path:        pluginPath,
	}, nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/pluginsearch/... -v -run TestScanner`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/pluginsearch/
git commit -m "feat(pluginsearch): add scanner to parse plugin.json"
```

---

## Task 3: Add Skill Parsing to Scanner

**Files:**
- Modify: `internal/pluginsearch/scanner.go`
- Modify: `internal/pluginsearch/scanner_test.go`
- Create: `internal/pluginsearch/testdata/cache/test-marketplace/test-plugin/1.0.0/skills/my-skill/SKILL.md`

**Step 1: Create test fixture**

```bash
mkdir -p internal/pluginsearch/testdata/cache/test-marketplace/test-plugin/1.0.0/skills/my-skill
```

```markdown
// internal/pluginsearch/testdata/cache/test-marketplace/test-plugin/1.0.0/skills/my-skill/SKILL.md
---
name: my-skill
description: A skill for testing purposes
---

# My Skill

This is the skill content.
```

**Step 2: Write the failing test**

```go
// Add to scanner_test.go

func TestScanner_ParsesSkills(t *testing.T) {
	scanner := pluginsearch.NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache")

	plugins, err := scanner.Scan(cacheDir)
	require.NoError(t, err)
	require.Len(t, plugins, 1)

	p := plugins[0]
	require.True(t, p.HasSkills())
	require.Len(t, p.Skills, 1)

	skill := p.Skills[0]
	assert.Equal(t, "my-skill", skill.Name)
	assert.Equal(t, "A skill for testing purposes", skill.Description)
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/pluginsearch/... -v -run TestScanner_ParsesSkills`
Expected: FAIL - Skills slice is empty

**Step 4: Update implementation**

Add to scanner.go imports:
```go
import (
	"bufio"
	"strings"
	// ... existing imports
)
```

Add after parsePlugin call in Scan():
```go
// After creating plugin from parsePlugin, add:
plugin.Skills = s.scanSkills(versionPath)
plugin.Commands = s.scanComponents(versionPath, "commands")
plugin.Agents = s.scanComponents(versionPath, "agents")
```

Add new methods:
```go
func (s *Scanner) scanSkills(pluginPath string) []ComponentInfo {
	skillsDir := filepath.Join(pluginPath, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}

	var skills []ComponentInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsDir, entry.Name())
		skill := s.parseSkillFrontmatter(skillPath, entry.Name())
		skills = append(skills, skill)
	}
	return skills
}

func (s *Scanner) parseSkillFrontmatter(skillPath, dirName string) ComponentInfo {
	skillFile := filepath.Join(skillPath, "SKILL.md")
	file, err := os.Open(skillFile)
	if err != nil {
		return ComponentInfo{Name: dirName, Path: skillPath}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	name := dirName
	description := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if inFrontmatter {
				break // End of frontmatter
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if strings.HasPrefix(line, "name:") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				name = strings.Trim(name, "\"'")
			} else if strings.HasPrefix(line, "description:") {
				description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				description = strings.Trim(description, "\"'")
			}
		}
	}

	return ComponentInfo{
		Name:        name,
		Description: description,
		Path:        skillPath,
	}
}

func (s *Scanner) scanComponents(pluginPath, componentType string) []ComponentInfo {
	dir := filepath.Join(pluginPath, componentType)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var components []ComponentInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		components = append(components, ComponentInfo{
			Name: entry.Name(),
			Path: filepath.Join(dir, entry.Name()),
		})
	}
	return components
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/pluginsearch/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/pluginsearch/
git commit -m "feat(pluginsearch): add skill/command/agent parsing to scanner"
```

---

## Task 4: Create Matcher for Search Logic

**Files:**
- Create: `internal/pluginsearch/matcher.go`
- Modify: `internal/pluginsearch/matcher_test.go`

**Step 1: Write the failing test**

```go
// internal/pluginsearch/matcher_test.go
// ABOUTME: Unit tests for search matching logic
// ABOUTME: Tests substring and regex matching across plugin metadata

package pluginsearch_test

import (
	"testing"

	"github.com/claudeup/claudeup/v2/internal/pluginsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatcher_SubstringMatch(t *testing.T) {
	index := []pluginsearch.PluginSearchIndex{
		{
			Name:        "superpowers",
			Description: "TDD and debugging skills",
			Keywords:    []string{"tdd", "debugging"},
			Marketplace: "test",
			Skills: []pluginsearch.ComponentInfo{
				{Name: "test-driven-development", Description: "Use TDD for all features"},
			},
		},
		{
			Name:        "frontend-tools",
			Description: "React and Vue helpers",
			Marketplace: "test",
		},
	}

	matcher := pluginsearch.NewMatcher()
	results := matcher.Search(index, "tdd", pluginsearch.SearchOptions{})

	require.Len(t, results, 1)
	assert.Equal(t, "superpowers", results[0].Plugin.Name)
	assert.NotEmpty(t, results[0].Matches)
}

func TestMatcher_CaseInsensitive(t *testing.T) {
	index := []pluginsearch.PluginSearchIndex{
		{Name: "TDD-Plugin", Description: "uppercase", Marketplace: "test"},
	}

	matcher := pluginsearch.NewMatcher()
	results := matcher.Search(index, "tdd", pluginsearch.SearchOptions{})

	require.Len(t, results, 1)
}

func TestMatcher_FilterByType(t *testing.T) {
	index := []pluginsearch.PluginSearchIndex{
		{
			Name:        "mixed-plugin",
			Marketplace: "test",
			Skills:      []pluginsearch.ComponentInfo{{Name: "tdd-skill"}},
			Commands:    []pluginsearch.ComponentInfo{{Name: "tdd-command"}},
		},
	}

	matcher := pluginsearch.NewMatcher()
	results := matcher.Search(index, "tdd", pluginsearch.SearchOptions{
		FilterType: "skills",
	})

	require.Len(t, results, 1)
	require.Len(t, results[0].Matches, 1)
	assert.Equal(t, "skill", results[0].Matches[0].Type)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/pluginsearch/... -v -run TestMatcher`
Expected: FAIL - NewMatcher undefined

**Step 3: Write implementation**

```go
// internal/pluginsearch/matcher.go
// ABOUTME: Search matching logic for plugin search
// ABOUTME: Supports substring and regex matching with type filtering

package pluginsearch

import (
	"regexp"
	"strings"
)

// SearchOptions configures search behavior.
type SearchOptions struct {
	UseRegex        bool
	FilterType      string // "skills", "commands", "agents", or "" for all
	FilterMarket    string // Filter to specific marketplace
	SearchContent   bool   // Also search SKILL.md body content
}

// Match represents a single match within a plugin.
type Match struct {
	Type        string // "name", "description", "keyword", "skill", "command", "agent"
	Name        string // Component name if applicable
	Description string // Component description if applicable
	Context     string // The matched text
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

// Search finds plugins matching the query.
func (m *Matcher) Search(index []PluginSearchIndex, query string, opts SearchOptions) []SearchResult {
	var results []SearchResult

	matchFunc := m.substringMatch
	if opts.UseRegex {
		re, err := regexp.Compile("(?i)" + query)
		if err == nil {
			matchFunc = func(text, _ string) bool {
				return re.MatchString(text)
			}
		}
	}

	queryLower := strings.ToLower(query)

	for _, plugin := range index {
		// Filter by marketplace
		if opts.FilterMarket != "" && plugin.Marketplace != opts.FilterMarket {
			continue
		}

		var matches []Match

		// Match plugin name
		if matchFunc(plugin.Name, queryLower) {
			matches = append(matches, Match{
				Type:    "name",
				Context: plugin.Name,
			})
		}

		// Match plugin description
		if matchFunc(plugin.Description, queryLower) {
			matches = append(matches, Match{
				Type:    "description",
				Context: plugin.Description,
			})
		}

		// Match keywords
		for _, kw := range plugin.Keywords {
			if matchFunc(kw, queryLower) {
				matches = append(matches, Match{
					Type:    "keyword",
					Context: kw,
				})
			}
		}

		// Match skills (if not filtering or filtering for skills)
		if opts.FilterType == "" || opts.FilterType == "skills" {
			for _, skill := range plugin.Skills {
				if matchFunc(skill.Name, queryLower) || matchFunc(skill.Description, queryLower) {
					matches = append(matches, Match{
						Type:        "skill",
						Name:        skill.Name,
						Description: skill.Description,
						Context:     skill.Name,
					})
				}
			}
		}

		// Match commands
		if opts.FilterType == "" || opts.FilterType == "commands" {
			for _, cmd := range plugin.Commands {
				if matchFunc(cmd.Name, queryLower) {
					matches = append(matches, Match{
						Type:    "command",
						Name:    cmd.Name,
						Context: cmd.Name,
					})
				}
			}
		}

		// Match agents
		if opts.FilterType == "" || opts.FilterType == "agents" {
			for _, agent := range plugin.Agents {
				if matchFunc(agent.Name, queryLower) {
					matches = append(matches, Match{
						Type:    "agent",
						Name:    agent.Name,
						Context: agent.Name,
					})
				}
			}
		}

		if len(matches) > 0 {
			results = append(results, SearchResult{
				Plugin:  plugin,
				Matches: matches,
			})
		}
	}

	return results
}

func (m *Matcher) substringMatch(text, queryLower string) bool {
	return strings.Contains(strings.ToLower(text), queryLower)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/pluginsearch/... -v -run TestMatcher`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/pluginsearch/
git commit -m "feat(pluginsearch): add matcher for search logic"
```

---

## Task 5: Create Formatter for Output

**Files:**
- Create: `internal/pluginsearch/formatter.go`
- Create: `internal/pluginsearch/formatter_test.go`

**Step 1: Write the failing test**

```go
// internal/pluginsearch/formatter_test.go
// ABOUTME: Unit tests for search result formatting
// ABOUTME: Tests plugin-centric and component-centric output formats

package pluginsearch_test

import (
	"bytes"
	"testing"

	"github.com/claudeup/claudeup/v2/internal/pluginsearch"
	"github.com/stretchr/testify/assert"
)

func TestFormatter_DefaultOutput(t *testing.T) {
	results := []pluginsearch.SearchResult{
		{
			Plugin: pluginsearch.PluginSearchIndex{
				Name:        "test-plugin",
				Marketplace: "test-market",
				Version:     "1.0.0",
			},
			Matches: []pluginsearch.Match{
				{Type: "skill", Name: "tdd-skill", Description: "TDD workflow"},
			},
		},
	}

	var buf bytes.Buffer
	formatter := pluginsearch.NewFormatter(&buf)
	formatter.Render(results, "tdd", pluginsearch.FormatOptions{})

	output := buf.String()
	assert.Contains(t, output, "test-plugin@test-market")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "tdd-skill")
}

func TestFormatter_ByComponent(t *testing.T) {
	results := []pluginsearch.SearchResult{
		{
			Plugin: pluginsearch.PluginSearchIndex{
				Name:        "plugin-a",
				Marketplace: "market",
			},
			Matches: []pluginsearch.Match{
				{Type: "skill", Name: "skill-1", Description: "First skill"},
			},
		},
		{
			Plugin: pluginsearch.PluginSearchIndex{
				Name:        "plugin-b",
				Marketplace: "market",
			},
			Matches: []pluginsearch.Match{
				{Type: "skill", Name: "skill-2", Description: "Second skill"},
			},
		},
	}

	var buf bytes.Buffer
	formatter := pluginsearch.NewFormatter(&buf)
	formatter.Render(results, "skill", pluginsearch.FormatOptions{ByComponent: true})

	output := buf.String()
	assert.Contains(t, output, "Skills:")
	assert.Contains(t, output, "skill-1")
	assert.Contains(t, output, "skill-2")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/pluginsearch/... -v -run TestFormatter`
Expected: FAIL - NewFormatter undefined

**Step 3: Write implementation**

```go
// internal/pluginsearch/formatter.go
// ABOUTME: Formats search results for terminal output
// ABOUTME: Supports plugin-centric, component-centric, and JSON formats

package pluginsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// FormatOptions configures output format.
type FormatOptions struct {
	Format      string // "default", "table", "json"
	ByComponent bool   // Group by component type instead of plugin
}

// Formatter renders search results.
type Formatter struct {
	w io.Writer
}

// NewFormatter creates a new Formatter writing to w.
func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{w: w}
}

// Render outputs the search results.
func (f *Formatter) Render(results []SearchResult, query string, opts FormatOptions) {
	if len(results) == 0 {
		fmt.Fprintf(f.w, "No results for %q\n", query)
		fmt.Fprintln(f.w, "\nTry:")
		fmt.Fprintln(f.w, "  - Broaden your search term")
		fmt.Fprintln(f.w, "  - Use --all to search all cached plugins")
		return
	}

	switch opts.Format {
	case "json":
		f.renderJSON(results, query)
	default:
		if opts.ByComponent {
			f.renderByComponent(results, query)
		} else {
			f.renderDefault(results, query)
		}
	}
}

func (f *Formatter) renderDefault(results []SearchResult, query string) {
	totalMatches := 0
	for _, r := range results {
		totalMatches += len(r.Matches)
	}

	fmt.Fprintf(f.w, "Search results for %q (%d plugins, %d matches)\n\n",
		query, len(results), totalMatches)

	for _, r := range results {
		// Plugin header
		fullName := r.Plugin.Name + "@" + r.Plugin.Marketplace
		if r.Plugin.Version != "" {
			fmt.Fprintf(f.w, "%s (v%s)\n", fullName, r.Plugin.Version)
		} else {
			fmt.Fprintf(f.w, "%s\n", fullName)
		}

		// Group matches by type
		skills := filterMatches(r.Matches, "skill")
		commands := filterMatches(r.Matches, "command")
		agents := filterMatches(r.Matches, "agent")

		if len(skills) > 0 {
			names := matchNames(skills)
			fmt.Fprintf(f.w, "  Skills: %s\n", joinNames(names))
		}
		if len(commands) > 0 {
			names := matchNames(commands)
			fmt.Fprintf(f.w, "  Commands: %s\n", joinNames(names))
		}
		if len(agents) > 0 {
			names := matchNames(agents)
			fmt.Fprintf(f.w, "  Agents: %s\n", joinNames(names))
		}

		// Show first match context
		if len(r.Matches) > 0 {
			m := r.Matches[0]
			if m.Description != "" {
				fmt.Fprintf(f.w, "  Match: %q - %s\n", m.Name, m.Description)
			} else if m.Type == "keyword" {
				fmt.Fprintf(f.w, "  Match: keyword %q\n", m.Context)
			}
		}

		fmt.Fprintln(f.w)
	}
}

func (f *Formatter) renderByComponent(results []SearchResult, query string) {
	// Collect all components by type
	type componentEntry struct {
		Name        string
		Description string
		Plugin      string
		Marketplace string
	}

	skills := []componentEntry{}
	commands := []componentEntry{}
	agents := []componentEntry{}

	for _, r := range results {
		fullName := r.Plugin.Name + "@" + r.Plugin.Marketplace
		for _, m := range r.Matches {
			entry := componentEntry{
				Name:        m.Name,
				Description: m.Description,
				Plugin:      fullName,
			}
			switch m.Type {
			case "skill":
				skills = append(skills, entry)
			case "command":
				commands = append(commands, entry)
			case "agent":
				agents = append(agents, entry)
			}
		}
	}

	totalComponents := len(skills) + len(commands) + len(agents)
	fmt.Fprintf(f.w, "Search results for %q (%d components across %d plugins)\n\n",
		query, totalComponents, len(results))

	if len(skills) > 0 {
		fmt.Fprintln(f.w, "Skills:")
		for _, s := range skills {
			fmt.Fprintf(f.w, "  %s (%s)\n", s.Name, s.Plugin)
			if s.Description != "" {
				fmt.Fprintf(f.w, "    %s\n", s.Description)
			}
		}
		fmt.Fprintln(f.w)
	}

	if len(commands) > 0 {
		fmt.Fprintln(f.w, "Commands:")
		for _, c := range commands {
			fmt.Fprintf(f.w, "  %s (%s)\n", c.Name, c.Plugin)
		}
		fmt.Fprintln(f.w)
	}

	if len(agents) > 0 {
		fmt.Fprintln(f.w, "Agents:")
		for _, a := range agents {
			fmt.Fprintf(f.w, "  %s (%s)\n", a.Name, a.Plugin)
		}
		fmt.Fprintln(f.w)
	}
}

func (f *Formatter) renderJSON(results []SearchResult, query string) {
	type matchOutput struct {
		Type        string `json:"type"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}

	type pluginOutput struct {
		Plugin      string        `json:"plugin"`
		Marketplace string        `json:"marketplace"`
		Version     string        `json:"version,omitempty"`
		Matches     []matchOutput `json:"matches"`
	}

	output := struct {
		Query        string         `json:"query"`
		TotalPlugins int            `json:"totalPlugins"`
		TotalMatches int            `json:"totalMatches"`
		Results      []pluginOutput `json:"results"`
	}{
		Query:        query,
		TotalPlugins: len(results),
		Results:      make([]pluginOutput, len(results)),
	}

	for i, r := range results {
		matches := make([]matchOutput, len(r.Matches))
		for j, m := range r.Matches {
			matches[j] = matchOutput{
				Type:        m.Type,
				Name:        m.Name,
				Description: m.Description,
			}
		}
		output.Results[i] = pluginOutput{
			Plugin:      r.Plugin.Name,
			Marketplace: r.Plugin.Marketplace,
			Version:     r.Plugin.Version,
			Matches:     matches,
		}
		output.TotalMatches += len(r.Matches)
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(f.w, string(data))
}

func filterMatches(matches []Match, matchType string) []Match {
	var filtered []Match
	for _, m := range matches {
		if m.Type == matchType {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func matchNames(matches []Match) []string {
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m.Name
	}
	sort.Strings(names)
	return names
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		result += ", " + names[i]
	}
	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/pluginsearch/... -v -run TestFormatter`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/pluginsearch/
git commit -m "feat(pluginsearch): add formatter for output rendering"
```

---

## Task 6: Create CLI Command

**Files:**
- Create: `internal/commands/plugin_search.go`
- Test via acceptance test in Task 7

**Step 1: Write the command**

```go
// internal/commands/plugin_search.go
// ABOUTME: CLI command for searching plugins by capability
// ABOUTME: Integrates pluginsearch package with Cobra CLI

package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/internal/claude"
	"github.com/claudeup/claudeup/v2/internal/pluginsearch"
	"github.com/spf13/cobra"
)

var (
	searchAll         bool
	searchType        string
	searchMarketplace string
	searchByComponent bool
	searchContent     bool
	searchRegex       bool
	searchFormat      string
)

var pluginSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search plugins by capability",
	Long: `Search across installed plugins to find those with specific capabilities.

By default, searches only installed plugins. Use --all to search the entire
plugin cache (all synced marketplaces).

Searches plugin names, descriptions, keywords, and component names/descriptions.`,
	Example: `  # Find TDD-related plugins
  claudeup plugin search tdd

  # Search all cached plugins for skill-creation capabilities
  claudeup plugin search "skill" --all --type skills --by-component

  # Find commit commands in a specific marketplace
  claudeup plugin search commit --type commands --marketplace superpowers-marketplace

  # Regex search
  claudeup plugin search "front.?end|react" --regex --all`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginSearch,
}

func init() {
	pluginCmd.AddCommand(pluginSearchCmd)

	pluginSearchCmd.Flags().BoolVar(&searchAll, "all", false, "Search all cached plugins, not just installed")
	pluginSearchCmd.Flags().StringVar(&searchType, "type", "", "Filter by component type: skills, commands, agents")
	pluginSearchCmd.Flags().StringVar(&searchMarketplace, "marketplace", "", "Limit to specific marketplace")
	pluginSearchCmd.Flags().BoolVar(&searchByComponent, "by-component", false, "Group results by component type")
	pluginSearchCmd.Flags().BoolVar(&searchContent, "content", false, "Also search SKILL.md body content")
	pluginSearchCmd.Flags().BoolVar(&searchRegex, "regex", false, "Treat query as regular expression")
	pluginSearchCmd.Flags().StringVar(&searchFormat, "format", "", "Output format: json")
}

func runPluginSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Validate --type flag
	if searchType != "" && searchType != "skills" && searchType != "commands" && searchType != "agents" {
		return fmt.Errorf("invalid --type: must be skills, commands, or agents")
	}

	// Determine cache directory
	cacheDir := filepath.Join(claudeDir, "plugins", "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin cache not found at %s\n\nInstall some plugins first with 'claude plugin install'", cacheDir)
	}

	// Build index
	scanner := pluginsearch.NewScanner()
	allPlugins, err := scanner.Scan(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to scan plugin cache: %w", err)
	}

	// Filter to installed only (unless --all)
	plugins := allPlugins
	if !searchAll {
		installed, err := claude.LoadPlugins(claudeDir)
		if err != nil {
			return fmt.Errorf("failed to load installed plugins: %w", err)
		}

		var filtered []pluginsearch.PluginSearchIndex
		for _, p := range allPlugins {
			fullName := p.Name + "@" + p.Marketplace
			if installed.PluginExists(fullName) {
				filtered = append(filtered, p)
			}
		}
		plugins = filtered
	}

	if len(plugins) == 0 {
		if searchAll {
			return fmt.Errorf("no plugins found in cache\n\nSync a marketplace first with 'claude marketplace add'")
		}
		return fmt.Errorf("no plugins installed\n\nInstall plugins with 'claude plugin install' or use --all to search cached plugins")
	}

	// Search
	matcher := pluginsearch.NewMatcher()
	results := matcher.Search(plugins, query, pluginsearch.SearchOptions{
		UseRegex:      searchRegex,
		FilterType:    searchType,
		FilterMarket:  searchMarketplace,
		SearchContent: searchContent,
	})

	// Format output
	formatter := pluginsearch.NewFormatter(os.Stdout)
	formatter.Render(results, query, pluginsearch.FormatOptions{
		Format:      searchFormat,
		ByComponent: searchByComponent,
	})

	return nil
}
```

**Step 2: Run unit tests to ensure no regressions**

Run: `go test ./internal/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/plugin_search.go
git commit -m "feat(commands): add plugin search command"
```

---

## Task 7: Add Acceptance Tests

**Files:**
- Create: `test/acceptance/plugin_search_test.go`

**Step 1: Write acceptance test**

```go
// test/acceptance/plugin_search_test.go
// ABOUTME: Acceptance tests for plugin search command
// ABOUTME: Tests CLI behavior with real binary execution

package acceptance_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v2/test/helpers"
)

var _ = Describe("plugin search", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)

		// Create a fake plugin in the cache
		cacheDir := filepath.Join(env.ClaudeDir, "plugins", "cache", "test-marketplace", "test-plugin", "1.0.0")
		Expect(os.MkdirAll(filepath.Join(cacheDir, ".claude-plugin"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(cacheDir, "skills", "tdd-skill"), 0755)).To(Succeed())

		// Write plugin.json
		pluginJSON := `{
			"name": "test-plugin",
			"description": "A plugin for testing search",
			"version": "1.0.0",
			"keywords": ["testing", "tdd"]
		}`
		Expect(os.WriteFile(
			filepath.Join(cacheDir, ".claude-plugin", "plugin.json"),
			[]byte(pluginJSON),
			0644,
		)).To(Succeed())

		// Write SKILL.md
		skillMD := `---
name: tdd-skill
description: Test-driven development workflow
---

# TDD Skill

Use this for TDD.
`
		Expect(os.WriteFile(
			filepath.Join(cacheDir, "skills", "tdd-skill", "SKILL.md"),
			[]byte(skillMD),
			0644,
		)).To(Succeed())

		// Install the plugin (add to installed_plugins.json)
		pluginsDir := filepath.Join(env.ClaudeDir, "plugins")
		Expect(os.MkdirAll(pluginsDir, 0755)).To(Succeed())
		installedJSON := `{
			"version": "1.0",
			"plugins": {
				"test-plugin@test-marketplace": {
					"enabled": true,
					"version": "1.0.0",
					"installedAt": "2025-01-01T00:00:00Z"
				}
			}
		}`
		Expect(os.WriteFile(
			filepath.Join(pluginsDir, "installed_plugins.json"),
			[]byte(installedJSON),
			0644,
		)).To(Succeed())
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("finds plugins matching query", func() {
		result := env.Run("plugin", "search", "tdd")
		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("test-plugin@test-marketplace"))
		Expect(result.Stdout).To(ContainSubstring("tdd-skill"))
	})

	It("filters by component type", func() {
		result := env.Run("plugin", "search", "tdd", "--type", "skills")
		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("tdd-skill"))
	})

	It("supports --by-component flag", func() {
		result := env.Run("plugin", "search", "tdd", "--by-component")
		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("Skills:"))
	})

	It("supports JSON output", func() {
		result := env.Run("plugin", "search", "tdd", "--format", "json")
		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring(`"query": "tdd"`))
		Expect(result.Stdout).To(ContainSubstring(`"plugin": "test-plugin"`))
	})

	It("shows no results message for non-matching query", func() {
		result := env.Run("plugin", "search", "nonexistent")
		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("No results"))
	})

	It("validates --type flag values", func() {
		result := env.Run("plugin", "search", "tdd", "--type", "invalid")
		Expect(result.ExitCode).To(Equal(1))
		Expect(result.Stderr).To(ContainSubstring("invalid --type"))
	})
})
```

**Step 2: Build binary and run acceptance test**

Run: `go build -o bin/claudeup ./cmd/claudeup && go test ./test/acceptance/... -v -run "plugin search" --timeout 5m`
Expected: PASS

**Step 3: Commit**

```bash
git add test/acceptance/plugin_search_test.go
git commit -m "test(acceptance): add plugin search acceptance tests"
```

---

## Task 8: Run Full Test Suite and Final Commit

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: PASS

**Step 2: Create final summary commit**

```bash
git log --oneline feature/99-plugin-search ^main
```

Review commits look correct.

**Step 3: Push branch**

```bash
git push -u origin feature/99-plugin-search
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Index types | `internal/pluginsearch/index.go` |
| 2 | Scanner basics | `internal/pluginsearch/scanner.go` |
| 3 | Skill parsing | `internal/pluginsearch/scanner.go` |
| 4 | Matcher | `internal/pluginsearch/matcher.go` |
| 5 | Formatter | `internal/pluginsearch/formatter.go` |
| 6 | CLI command | `internal/commands/plugin_search.go` |
| 7 | Acceptance tests | `test/acceptance/plugin_search_test.go` |
| 8 | Final validation | - |

**Total estimated tasks:** 8 bite-sized implementation tasks following TDD.
