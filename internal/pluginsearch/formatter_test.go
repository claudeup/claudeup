// ABOUTME: Unit tests for plugin search formatter
// ABOUTME: Tests output rendering in default, by-component, JSON, and no-results formats

package pluginsearch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatter_DefaultOutput(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "superpowers",
				Marketplace: "superpowers-marketplace",
				Version:     "4.0.3",
				Skills: []ComponentInfo{
					{Name: "test-driven-development", Description: "Use when implementing any feature or bugfix"},
					{Name: "systematic-debugging", Description: "Use when debugging issues"},
				},
			},
			Matches: []Match{
				{Type: "skill", Name: "test-driven-development", Description: "Use when implementing any feature or bugfix", Context: "test-driven-development"},
			},
		},
		{
			Plugin: PluginSearchIndex{
				Name:        "backend-development",
				Marketplace: "claude-code-workflows",
				Version:     "2.1.0",
				Skills: []ComponentInfo{
					{Name: "tdd-orchestrator", Description: "Master TDD orchestrator"},
				},
			},
			Matches: []Match{
				{Type: "skill", Name: "tdd-orchestrator", Description: "Master TDD orchestrator", Context: "tdd-orchestrator"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "tdd", FormatOptions{Format: "default"})

	output := buf.String()

	// Should show header with query and counts
	if !strings.Contains(output, "tdd") {
		t.Error("expected output to contain query 'tdd'")
	}
	if !strings.Contains(output, "2 plugins") {
		t.Errorf("expected output to show '2 plugins', got:\n%s", output)
	}

	// Should show plugin names with marketplace and version
	if !strings.Contains(output, "superpowers@superpowers-marketplace") {
		t.Errorf("expected output to contain 'superpowers@superpowers-marketplace', got:\n%s", output)
	}
	if !strings.Contains(output, "v4.0.3") {
		t.Errorf("expected output to contain version 'v4.0.3', got:\n%s", output)
	}

	// Should show skill names
	if !strings.Contains(output, "test-driven-development") {
		t.Errorf("expected output to contain skill 'test-driven-development', got:\n%s", output)
	}

	// Should show match context/description
	if !strings.Contains(output, "Use when implementing any feature or bugfix") {
		t.Errorf("expected output to contain match description, got:\n%s", output)
	}
}

func TestFormatter_ByComponent(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "superpowers",
				Marketplace: "superpowers-marketplace",
				Version:     "4.0.3",
				Skills: []ComponentInfo{
					{Name: "test-driven-development", Description: "TDD methodology"},
				},
				Commands: []ComponentInfo{
					{Name: "/tdd", Description: "Run TDD workflow"},
				},
			},
			Matches: []Match{
				{Type: "skill", Name: "test-driven-development", Description: "TDD methodology", Context: "test-driven-development"},
				{Type: "command", Name: "/tdd", Description: "Run TDD workflow", Context: "/tdd"},
			},
		},
		{
			Plugin: PluginSearchIndex{
				Name:        "backend-development",
				Marketplace: "claude-code-workflows",
				Version:     "2.1.0",
				Skills: []ComponentInfo{
					{Name: "tdd-orchestrator", Description: "Master TDD orchestrator"},
				},
			},
			Matches: []Match{
				{Type: "skill", Name: "tdd-orchestrator", Description: "Master TDD orchestrator", Context: "tdd-orchestrator"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "tdd", FormatOptions{Format: "default", ByComponent: true})

	output := buf.String()

	// Should group by component type with "Skills" section header
	if !strings.Contains(output, "Skills") {
		t.Errorf("expected output to contain 'Skills' header, got:\n%s", output)
	}

	// Should show components grouped together
	if !strings.Contains(output, "test-driven-development") {
		t.Errorf("expected output to contain 'test-driven-development', got:\n%s", output)
	}
	if !strings.Contains(output, "tdd-orchestrator") {
		t.Errorf("expected output to contain 'tdd-orchestrator', got:\n%s", output)
	}

	// Should show plugin info in parentheses
	if !strings.Contains(output, "(superpowers@superpowers-marketplace)") {
		t.Errorf("expected output to contain plugin reference in parentheses, got:\n%s", output)
	}

	// Should have Commands section if commands matched
	if !strings.Contains(output, "Commands") {
		t.Errorf("expected output to contain 'Commands' header, got:\n%s", output)
	}
}

func TestFormatter_JSONOutput(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "superpowers",
				Marketplace: "superpowers-marketplace",
				Version:     "4.0.3",
				Skills: []ComponentInfo{
					{Name: "test-driven-development", Description: "TDD methodology"},
				},
			},
			Matches: []Match{
				{Type: "skill", Name: "test-driven-development", Description: "TDD methodology", Context: "test-driven-development"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "tdd", FormatOptions{Format: "json"})

	output := buf.String()

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput:\n%s", err, output)
	}

	// Should have expected fields
	if parsed["query"] != "tdd" {
		t.Errorf("expected query 'tdd', got %v", parsed["query"])
	}

	totalPlugins, ok := parsed["totalPlugins"].(float64)
	if !ok || totalPlugins != 1 {
		t.Errorf("expected totalPlugins 1, got %v", parsed["totalPlugins"])
	}

	results_arr, ok := parsed["results"].([]interface{})
	if !ok || len(results_arr) != 1 {
		t.Errorf("expected results array with 1 item, got %v", parsed["results"])
	}

	// Check first result structure
	if len(results_arr) > 0 {
		result := results_arr[0].(map[string]interface{})
		if result["plugin"] != "superpowers" {
			t.Errorf("expected plugin 'superpowers', got %v", result["plugin"])
		}
		if result["marketplace"] != "superpowers-marketplace" {
			t.Errorf("expected marketplace 'superpowers-marketplace', got %v", result["marketplace"])
		}
		if result["version"] != "4.0.3" {
			t.Errorf("expected version '4.0.3', got %v", result["version"])
		}
	}
}

func TestFormatter_NoResults(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render([]SearchResult{}, "nonexistent", FormatOptions{Format: "default"})

	output := buf.String()

	// Should show "no results" message
	if !strings.Contains(strings.ToLower(output), "no results") {
		t.Errorf("expected 'no results' message, got:\n%s", output)
	}

	// Should include the query
	if !strings.Contains(output, "nonexistent") {
		t.Errorf("expected output to contain query 'nonexistent', got:\n%s", output)
	}

	// Should show helpful tips
	if !strings.Contains(output, "Try:") {
		t.Errorf("expected helpful tips starting with 'Try:', got:\n%s", output)
	}
}

func TestFormatter_TableFormat(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "superpowers",
				Marketplace: "superpowers-marketplace",
				Version:     "4.0.3",
				Skills: []ComponentInfo{
					{Name: "test-driven-development", Description: "TDD methodology"},
				},
			},
			Matches: []Match{
				{Type: "skill", Name: "test-driven-development", Description: "TDD methodology", Context: "test-driven-development"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "tdd", FormatOptions{Format: "table"})

	output := buf.String()

	// Table format should have some tabular structure
	// At minimum, plugin name and component should appear
	if !strings.Contains(output, "superpowers") {
		t.Errorf("expected output to contain 'superpowers', got:\n%s", output)
	}
	if !strings.Contains(output, "test-driven-development") {
		t.Errorf("expected output to contain skill name, got:\n%s", output)
	}
}

func TestFormatter_MatchCountsInHeader(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "plugin-a",
				Marketplace: "market",
				Version:     "1.0.0",
			},
			Matches: []Match{
				{Type: "skill", Name: "skill-1"},
				{Type: "skill", Name: "skill-2"},
			},
		},
		{
			Plugin: PluginSearchIndex{
				Name:        "plugin-b",
				Marketplace: "market",
				Version:     "1.0.0",
			},
			Matches: []Match{
				{Type: "command", Name: "cmd-1"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "test", FormatOptions{Format: "default"})

	output := buf.String()

	// Header should show total matches count (3) in section header
	if !strings.Contains(output, "(3)") {
		t.Errorf("expected header to show match count '(3)', got:\n%s", output)
	}
}

func TestFormatter_MultipleMatchTypes(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "test-plugin",
				Description: "Plugin for testing",
				Marketplace: "test-market",
				Version:     "1.0.0",
				Skills: []ComponentInfo{
					{Name: "test-skill", Description: "A skill for tests"},
				},
				Commands: []ComponentInfo{
					{Name: "/test", Description: "Run tests"},
				},
				Agents: []ComponentInfo{
					{Name: "test-agent", Description: "An agent for tests"},
				},
			},
			Matches: []Match{
				{Type: "name", Context: "test-plugin"},
				{Type: "skill", Name: "test-skill", Description: "A skill for tests"},
				{Type: "command", Name: "/test", Description: "Run tests"},
				{Type: "agent", Name: "test-agent", Description: "An agent for tests"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "test", FormatOptions{Format: "default"})

	output := buf.String()

	// Should include different match types in output
	if !strings.Contains(output, "test-skill") {
		t.Errorf("expected skill name in output, got:\n%s", output)
	}
	if !strings.Contains(output, "/test") {
		t.Errorf("expected command name in output, got:\n%s", output)
	}
	if !strings.Contains(output, "test-agent") {
		t.Errorf("expected agent name in output, got:\n%s", output)
	}
}

func TestFormatter_ByComponentShowsContentMatches(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "test-plugin",
				Marketplace: "test-market",
				Version:     "1.0.0",
			},
			Matches: []Match{
				{Type: "content", Name: "my-skill", Description: "A skill", Context: "body text"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "body", FormatOptions{Format: "default", ByComponent: true})

	output := buf.String()

	// Content matches should appear under Skills section
	if !strings.Contains(output, "Skills") {
		t.Errorf("expected content matches to appear under Skills section, got:\n%s", output)
	}
	if !strings.Contains(output, "my-skill") {
		t.Errorf("expected content match skill name in output, got:\n%s", output)
	}
}

func TestFormatter_DefaultShowsContentMatches(t *testing.T) {
	results := []SearchResult{
		{
			Plugin: PluginSearchIndex{
				Name:        "test-plugin",
				Marketplace: "test-market",
				Version:     "1.0.0",
			},
			Matches: []Match{
				{Type: "content", Name: "my-skill", Description: "A skill"},
			},
		},
	}

	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render(results, "test", FormatOptions{})

	output := buf.String()

	// Content matches should appear under Skills in default view
	if !strings.Contains(output, "Skills:") {
		t.Errorf("expected content matches to show under Skills in default view, got:\n%s", output)
	}
}

func TestFormatter_JSONNoResults(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.Render([]SearchResult{}, "nonexistent", FormatOptions{Format: "json"})

	output := buf.String()

	// Should still be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("expected valid JSON output even with no results, got error: %v", err)
	}

	// Should have empty results array
	results_arr, ok := parsed["results"].([]interface{})
	if !ok {
		t.Errorf("expected results array, got %v", parsed["results"])
	}
	if len(results_arr) != 0 {
		t.Errorf("expected empty results array, got %d items", len(results_arr))
	}

	// totalPlugins should be 0
	totalPlugins, _ := parsed["totalPlugins"].(float64)
	if totalPlugins != 0 {
		t.Errorf("expected totalPlugins 0, got %v", parsed["totalPlugins"])
	}
}
