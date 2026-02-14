// ABOUTME: Unit tests for plugin cache scanner
// ABOUTME: Tests scanning filesystem to build plugin index

package pluginsearch

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func TestScanner_ScanPlugin(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	p := plugins[0]

	if p.Name != "test-plugin" {
		t.Errorf("expected Name 'test-plugin', got '%s'", p.Name)
	}
	if p.Description != "A test plugin for unit tests" {
		t.Errorf("expected Description 'A test plugin for unit tests', got '%s'", p.Description)
	}
	if p.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got '%s'", p.Version)
	}
	if p.Marketplace != "test-marketplace" {
		t.Errorf("expected Marketplace 'test-marketplace', got '%s'", p.Marketplace)
	}

	expectedPath := filepath.Join(cacheDir, "test-marketplace", "test-plugin", "1.0.0")
	if p.Path != expectedPath {
		t.Errorf("expected Path '%s', got '%s'", expectedPath, p.Path)
	}

	// Check keywords
	hasKeyword := func(keywords []string, keyword string) bool {
		for _, k := range keywords {
			if k == keyword {
				return true
			}
		}
		return false
	}

	if !hasKeyword(p.Keywords, "testing") {
		t.Error("expected Keywords to contain 'testing'")
	}
	if !hasKeyword(p.Keywords, "example") {
		t.Error("expected Keywords to contain 'example'")
	}
}

func TestScanner_EmptyCache(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "empty-cache")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("Scan() returned error for empty cache: %v", err)
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins for empty cache, got %d", len(plugins))
	}
}

func TestScanner_NonExistentCache(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "does-not-exist")

	_, err := scanner.Scan(cacheDir)
	if err == nil {
		t.Error("expected error for non-existent cache directory, got nil")
	}
}

func TestScanner_DeduplicatesVersions(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache-multi-version")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	// Should return 2 plugins (test-plugin and no-version-plugin), not 5
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins (deduplicated), got %d", len(plugins))
	}

	// Find the test-plugin entry
	var testPlugin *PluginSearchIndex
	for i := range plugins {
		if plugins[i].Name == "test-plugin" {
			testPlugin = &plugins[i]
			break
		}
	}

	if testPlugin == nil {
		t.Fatal("expected to find test-plugin in results")
	}

	// Should keep the latest version (2.0.0)
	if testPlugin.Version != "2.0.0" {
		t.Errorf("expected latest version '2.0.0', got '%s'", testPlugin.Version)
	}

	expectedPath := filepath.Join(cacheDir, "test-marketplace", "test-plugin", "2.0.0")
	if testPlugin.Path != expectedPath {
		t.Errorf("expected path for latest version, got '%s'", testPlugin.Path)
	}
}

func TestScanner_DeduplicatesNonSemverVersions(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache-multi-version")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	// Find the no-version-plugin entry
	var found *PluginSearchIndex
	var count int
	for i := range plugins {
		if plugins[i].Name == "no-version-plugin" {
			found = &plugins[i]
			count++
		}
	}

	if count != 1 {
		t.Fatalf("expected 1 entry for no-version-plugin (deduplicated), got %d", count)
	}

	// Lexicographic fallback should keep "nightly-2" over "nightly-1"
	if found.Version != "nightly-2" {
		t.Errorf("expected version 'nightly-2' (lexicographic winner), got '%s'", found.Version)
	}
}

func TestScanner_ParsesSkills(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	p := plugins[0]
	if len(p.Skills) == 0 {
		t.Fatal("expected plugin to have skills")
	}
	if len(p.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(p.Skills))
	}

	skill := p.Skills[0]
	if skill.Name != "my-skill" {
		t.Errorf("expected skill name 'my-skill', got '%s'", skill.Name)
	}
	if skill.Description != "A skill for testing purposes" {
		t.Errorf("expected skill description 'A skill for testing purposes', got '%s'", skill.Description)
	}
}

func TestScanner_ParsesSkillContent(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	skill := plugins[0].Skills[0]
	// SKILL.md has body content: "# My Skill\n\nThis is the skill content."
	if skill.Content == "" {
		t.Fatal("expected skill Content to be populated from SKILL.md body")
	}
	if !strings.Contains(skill.Content, "This is the skill content") {
		t.Errorf("expected Content to contain body text, got: %q", skill.Content)
	}
	// Content should NOT include frontmatter
	if strings.Contains(skill.Content, "name:") {
		t.Error("Content should not include frontmatter")
	}
}
