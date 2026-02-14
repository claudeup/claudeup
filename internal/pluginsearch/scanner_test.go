// ABOUTME: Unit tests for plugin cache scanner
// ABOUTME: Tests scanning filesystem to build plugin index

package pluginsearch

import (
	"os"
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
	if len(p.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(p.Skills))
	}

	// Find my-skill (has frontmatter)
	var mySkill *ComponentInfo
	for i := range p.Skills {
		if p.Skills[i].Name == "my-skill" {
			mySkill = &p.Skills[i]
		}
	}
	if mySkill == nil {
		t.Fatal("expected to find skill 'my-skill'")
	}
	if mySkill.Description != "A skill for testing purposes" {
		t.Errorf("expected skill description 'A skill for testing purposes', got '%s'", mySkill.Description)
	}
}

func TestScanner_ParsesSkillWithoutFrontmatter(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	p := plugins[0]
	var skill *ComponentInfo
	for i := range p.Skills {
		if p.Skills[i].Name == "no-frontmatter-skill" {
			skill = &p.Skills[i]
		}
	}
	if skill == nil {
		t.Fatal("expected to find skill 'no-frontmatter-skill'")
	}
	// Without frontmatter, entire file should be treated as body content
	if skill.Content == "" {
		t.Fatal("expected Content to be populated even without frontmatter")
	}
	if !strings.Contains(skill.Content, "No Frontmatter Skill") {
		t.Errorf("expected Content to contain file body, got: %q", skill.Content)
	}
	if !strings.Contains(skill.Content, "searchable by content") {
		t.Errorf("expected Content to contain body text, got: %q", skill.Content)
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

func TestScanner_ContentTruncatedByByteLimit(t *testing.T) {
	// Create a temp directory structure mimicking the cache layout
	cacheDir := t.TempDir()
	pluginMetaDir := filepath.Join(cacheDir, "test-mp", "big-plugin", "1.0.0", ".claude-plugin")
	skillDir := filepath.Join(cacheDir, "test-mp", "big-plugin", "1.0.0", "skills", "big-skill")
	if err := os.MkdirAll(pluginMetaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(pluginMetaDir, "plugin.json"), []byte(`{
		"name": "big-plugin",
		"description": "Plugin with large skill content",
		"version": "1.0.0"
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a SKILL.md that exceeds the byte limit (maxContentBytes = 64KB)
	var content strings.Builder
	content.WriteString("---\nname: big-skill\ndescription: A big skill\n---\n")
	line := strings.Repeat("x", 100) + "\n" // 101 bytes per line
	for i := 0; i < 1000; i++ {             // ~101KB of body content
		content.WriteString(line)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content.String()), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner()
	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	skill := plugins[0].Skills[0]

	// Content should be truncated
	if !skill.Truncated {
		t.Error("expected Truncated to be true for large content")
	}

	// Content byte length should not greatly exceed the limit
	if len(skill.Content) > maxContentBytes+200 {
		t.Errorf("content should be near byte limit, got %d bytes (limit %d)", len(skill.Content), maxContentBytes)
	}

	// Content should still have text
	if skill.Content == "" {
		t.Error("expected truncated content to still have text")
	}
}

func TestScanner_ContentNotTruncatedWhenSmall(t *testing.T) {
	scanner := NewScanner()
	cacheDir := filepath.Join(testdataDir(), "cache")

	plugins, err := scanner.Scan(cacheDir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, skill := range plugins[0].Skills {
		if skill.Truncated {
			t.Errorf("expected Truncated=false for small skill %q", skill.Name)
		}
	}
}
