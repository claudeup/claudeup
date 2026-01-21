// ABOUTME: Unit tests for plugin cache scanner
// ABOUTME: Tests scanning filesystem to build plugin index

package pluginsearch

import (
	"path/filepath"
	"runtime"
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
