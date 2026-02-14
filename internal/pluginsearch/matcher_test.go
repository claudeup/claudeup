// ABOUTME: Unit tests for plugin search matcher
// ABOUTME: Tests search logic including substring, regex, and filtering

package pluginsearch

import (
	"testing"
)

func TestMatcher_SubstringMatch(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin for testing",
			Marketplace: "test-market",
			Skills: []ComponentInfo{
				{Name: "tdd-skill", Description: "Test driven development helper"},
			},
		},
		{
			Name:        "other-plugin",
			Description: "Something else",
			Marketplace: "test-market",
			Commands: []ComponentInfo{
				{Name: "build-cmd"},
			},
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "tdd", SearchOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Plugin.Name != "test-plugin" {
		t.Errorf("expected plugin 'test-plugin', got '%s'", results[0].Plugin.Name)
	}
	if len(results[0].Matches) == 0 {
		t.Error("expected at least one match")
	}
}

func TestMatcher_CaseInsensitive(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "TDD-Plugin",
			Description: "Test Driven Development",
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "tdd", SearchOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result for case-insensitive match, got %d", len(results))
	}
	if results[0].Plugin.Name != "TDD-Plugin" {
		t.Errorf("expected plugin 'TDD-Plugin', got '%s'", results[0].Plugin.Name)
	}
}

func TestMatcher_FilterByType(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "multi-plugin",
			Description: "Has skills and commands",
			Marketplace: "test-market",
			Skills: []ComponentInfo{
				{Name: "my-skill", Description: "A skill"},
			},
			Commands: []ComponentInfo{
				{Name: "my-command"},
			},
			Agents: []ComponentInfo{
				{Name: "my-agent"},
			},
		},
	}

	m := NewMatcher()

	// Filter by skills only - search for "my" which appears in all
	skillResults := m.Search(plugins, "my", SearchOptions{FilterType: "skills"})
	if len(skillResults) != 1 {
		t.Fatalf("expected 1 result for skills filter, got %d", len(skillResults))
	}
	// Should only have skill matches, not command or agent matches
	for _, match := range skillResults[0].Matches {
		if match.Type == "command" || match.Type == "agent" {
			t.Errorf("expected only skill matches with FilterType=skills, got match type '%s'", match.Type)
		}
	}

	// Filter by commands only
	cmdResults := m.Search(plugins, "my", SearchOptions{FilterType: "commands"})
	if len(cmdResults) != 1 {
		t.Fatalf("expected 1 result for commands filter, got %d", len(cmdResults))
	}
	for _, match := range cmdResults[0].Matches {
		if match.Type == "skill" || match.Type == "agent" {
			t.Errorf("expected only command matches with FilterType=commands, got match type '%s'", match.Type)
		}
	}

	// Filter by agents only
	agentResults := m.Search(plugins, "my", SearchOptions{FilterType: "agents"})
	if len(agentResults) != 1 {
		t.Fatalf("expected 1 result for agents filter, got %d", len(agentResults))
	}
	for _, match := range agentResults[0].Matches {
		if match.Type == "skill" || match.Type == "command" {
			t.Errorf("expected only agent matches with FilterType=agents, got match type '%s'", match.Type)
		}
	}
}

func TestMatcher_FilterByMarketplace(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "plugin-one",
			Description: "First plugin",
			Marketplace: "market-a",
		},
		{
			Name:        "plugin-two",
			Description: "Second plugin",
			Marketplace: "market-b",
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "plugin", SearchOptions{FilterMarket: "market-a"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result for marketplace filter, got %d", len(results))
	}
	if results[0].Plugin.Name != "plugin-one" {
		t.Errorf("expected plugin 'plugin-one', got '%s'", results[0].Plugin.Name)
	}
}

func TestMatcher_RegexMatch(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "Helps with testing",
			Marketplace: "test-market",
		},
		{
			Name:        "other-plugin",
			Description: "Something else",
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	// Regex to match words ending in "ing"
	results := m.Search(plugins, "test.*ing", SearchOptions{UseRegex: true})

	if len(results) != 1 {
		t.Fatalf("expected 1 result for regex match, got %d", len(results))
	}
	if results[0].Plugin.Name != "test-plugin" {
		t.Errorf("expected plugin 'test-plugin', got '%s'", results[0].Plugin.Name)
	}
}

func TestMatcher_MatchesPluginName(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "awesome-plugin",
			Description: "Does things",
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "awesome", SearchOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	foundNameMatch := false
	for _, match := range results[0].Matches {
		if match.Type == "name" {
			foundNameMatch = true
			break
		}
	}
	if !foundNameMatch {
		t.Error("expected a match with Type='name'")
	}
}

func TestMatcher_MatchesDescription(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "Handles complex workflows",
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "workflows", SearchOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	foundDescMatch := false
	for _, match := range results[0].Matches {
		if match.Type == "description" {
			foundDescMatch = true
			break
		}
	}
	if !foundDescMatch {
		t.Error("expected a match with Type='description'")
	}
}

func TestMatcher_MatchesKeywords(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin",
			Keywords:    []string{"automation", "testing", "ci-cd"},
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "automation", SearchOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	foundKeywordMatch := false
	for _, match := range results[0].Matches {
		if match.Type == "keyword" {
			foundKeywordMatch = true
			break
		}
	}
	if !foundKeywordMatch {
		t.Error("expected a match with Type='keyword'")
	}
}

func TestMatcher_NoMatches(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin",
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "nonexistent", SearchOptions{})

	if len(results) != 0 {
		t.Errorf("expected 0 results for non-matching query, got %d", len(results))
	}
}

func TestMatcher_InvalidRegex(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin",
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	// Invalid regex should not panic and return empty results
	results := m.Search(plugins, "[invalid", SearchOptions{UseRegex: true})

	if len(results) != 0 {
		t.Errorf("expected 0 results for invalid regex, got %d", len(results))
	}
}

func TestMatcher_SkillDescription(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin",
			Marketplace: "test-market",
			Skills: []ComponentInfo{
				{Name: "some-skill", Description: "Helps with debugging"},
			},
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "debugging", SearchOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result for skill description match, got %d", len(results))
	}

	foundSkillMatch := false
	for _, match := range results[0].Matches {
		if match.Type == "skill" && match.Name == "some-skill" {
			foundSkillMatch = true
			break
		}
	}
	if !foundSkillMatch {
		t.Error("expected a skill match")
	}
}

func TestMatcher_EmptyQuery(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin",
			Marketplace: "test-market",
		},
	}

	m := NewMatcher()
	results := m.Search(plugins, "", SearchOptions{})

	// Empty query should return no results (not match everything)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestMatcher_EmptyPluginList(t *testing.T) {
	m := NewMatcher()
	results := m.Search([]PluginSearchIndex{}, "test", SearchOptions{})

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty plugin list, got %d", len(results))
	}
}

func TestMatcher_ContentSearch(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin",
			Marketplace: "test-market",
			Skills: []ComponentInfo{
				{
					Name:        "my-skill",
					Description: "A skill",
					Content:     "This skill helps with debugging and profiling Go code",
				},
			},
		},
		{
			Name:        "other-plugin",
			Description: "Another plugin",
			Marketplace: "test-market",
			Skills: []ComponentInfo{
				{
					Name:        "other-skill",
					Description: "Another skill",
					Content:     "This skill handles frontend React components",
				},
			},
		},
	}

	m := NewMatcher()

	// Without --content, "profiling" should not match (not in name or description)
	results := m.Search(plugins, "profiling", SearchOptions{})
	if len(results) != 0 {
		t.Errorf("expected 0 results without content search, got %d", len(results))
	}

	// With --content, "profiling" should match the skill body
	results = m.Search(plugins, "profiling", SearchOptions{SearchContent: true})
	if len(results) != 1 {
		t.Fatalf("expected 1 result with content search, got %d", len(results))
	}
	if results[0].Plugin.Name != "test-plugin" {
		t.Errorf("expected test-plugin, got %s", results[0].Plugin.Name)
	}

	// Verify match type is "content"
	foundContentMatch := false
	for _, match := range results[0].Matches {
		if match.Type == "content" {
			foundContentMatch = true
			if match.Name != "my-skill" {
				t.Errorf("expected content match for 'my-skill', got '%s'", match.Name)
			}
			break
		}
	}
	if !foundContentMatch {
		t.Error("expected a match with Type='content'")
	}
}

func TestMatcher_ContentSearchWithTypeFilter(t *testing.T) {
	plugins := []PluginSearchIndex{
		{
			Name:        "test-plugin",
			Description: "A plugin",
			Marketplace: "test-market",
			Skills: []ComponentInfo{
				{
					Name:    "my-skill",
					Content: "Contains unique-search-term in body",
				},
			},
		},
	}

	m := NewMatcher()

	// Content search with type filter for skills should still match
	results := m.Search(plugins, "unique-search-term", SearchOptions{
		SearchContent: true,
		FilterType:    "skills",
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result with content+type filter, got %d", len(results))
	}

	// Content search with type filter for commands should NOT match
	results = m.Search(plugins, "unique-search-term", SearchOptions{
		SearchContent: true,
		FilterType:    "commands",
	})
	if len(results) != 0 {
		t.Errorf("expected 0 results with wrong type filter, got %d", len(results))
	}
}
