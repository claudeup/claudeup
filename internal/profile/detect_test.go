// ABOUTME: Tests for project detection logic
// ABOUTME: Validates matching projects to profiles based on detect rules
package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMatchesFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	profile := &Profile{
		Name: "nodejs",
		Detect: DetectRules{
			Files: []string{"package.json"},
		},
	}

	match, err := Detect(tmpDir, profile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !match {
		t.Error("Expected profile to match, but it didn't")
	}
}

func TestDetectMatchesContains(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json with react
	content := `{"name": "test", "dependencies": {"react": "^18.0.0"}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	profile := &Profile{
		Name: "react",
		Detect: DetectRules{
			Files:    []string{"package.json"},
			Contains: map[string]string{"package.json": "react"},
		},
	}

	match, err := Detect(tmpDir, profile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !match {
		t.Error("Expected profile to match react dependency")
	}
}

func TestDetectNoMatchMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	profile := &Profile{
		Name: "go",
		Detect: DetectRules{
			Files: []string{"go.mod"},
		},
	}

	match, err := Detect(tmpDir, profile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if match {
		t.Error("Expected no match for missing go.mod")
	}
}

func TestDetectNoMatchContentMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json WITHOUT react
	content := `{"name": "test", "dependencies": {"vue": "^3.0.0"}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	profile := &Profile{
		Name: "react",
		Detect: DetectRules{
			Files:    []string{"package.json"},
			Contains: map[string]string{"package.json": "react"},
		},
	}

	match, err := Detect(tmpDir, profile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if match {
		t.Error("Expected no match - vue project should not match react profile")
	}
}

func TestDetectEmptyRulesNoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	profile := &Profile{
		Name:   "empty",
		Detect: DetectRules{},
	}

	match, err := Detect(tmpDir, profile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if match {
		t.Error("Expected no match for profile with no detect rules")
	}
}

func TestFindMatchingProfiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	profiles := []*Profile{
		{Name: "nodejs", Detect: DetectRules{Files: []string{"package.json"}}},
		{Name: "go", Detect: DetectRules{Files: []string{"go.mod"}}},
		{Name: "rust", Detect: DetectRules{Files: []string{"Cargo.toml"}}},
	}

	matches := FindMatchingProfiles(tmpDir, profiles)

	if len(matches) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(matches))
	}

	if matches[0].Name != "go" {
		t.Errorf("Expected 'go' profile to match, got %q", matches[0].Name)
	}
}

func TestFrontendProfileDetectsNextJS(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a typical Next.js project structure
	packageJSON := `{
  "name": "my-nextjs-app",
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "next.config.js"), []byte("module.exports = {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Load the embedded frontend profile
	frontendProfile, err := GetEmbeddedProfile("frontend")
	if err != nil {
		t.Fatalf("Failed to load embedded frontend profile: %v", err)
	}

	match, err := Detect(tmpDir, frontendProfile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !match {
		t.Error("Expected frontend profile to match Next.js project")
	}
}

func TestFrontendProfileDetectsTailwind(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Tailwind project (without Next.js)
	if err := os.WriteFile(filepath.Join(tmpDir, "tailwind.config.js"), []byte("module.exports = {}"), 0644); err != nil {
		t.Fatal(err)
	}

	frontendProfile, err := GetEmbeddedProfile("frontend")
	if err != nil {
		t.Fatalf("Failed to load embedded frontend profile: %v", err)
	}

	match, err := Detect(tmpDir, frontendProfile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !match {
		t.Error("Expected frontend profile to match Tailwind project")
	}
}

func TestFrontendProfileDetectsShadcn(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a shadcn project (components.json is shadcn's config file)
	if err := os.WriteFile(filepath.Join(tmpDir, "components.json"), []byte(`{"style": "default"}`), 0644); err != nil {
		t.Fatal(err)
	}

	frontendProfile, err := GetEmbeddedProfile("frontend")
	if err != nil {
		t.Fatalf("Failed to load embedded frontend profile: %v", err)
	}

	match, err := Detect(tmpDir, frontendProfile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !match {
		t.Error("Expected frontend profile to match shadcn project (components.json)")
	}
}

func TestFrontendFullProfileDetectsPlaywright(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a project with Playwright
	packageJSON := `{
  "name": "my-app",
  "devDependencies": {
    "@playwright/test": "^1.40.0"
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "playwright.config.ts"), []byte("export default {}"), 0644); err != nil {
		t.Fatal(err)
	}

	frontendFullProfile, err := GetEmbeddedProfile("frontend-full")
	if err != nil {
		t.Fatalf("Failed to load embedded frontend-full profile: %v", err)
	}

	match, err := Detect(tmpDir, frontendFullProfile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !match {
		t.Error("Expected frontend-full profile to match Playwright project")
	}
}

func TestFrontendProfileDoesNotMatchGo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go project
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	frontendProfile, err := GetEmbeddedProfile("frontend")
	if err != nil {
		t.Fatalf("Failed to load embedded frontend profile: %v", err)
	}

	match, err := Detect(tmpDir, frontendProfile)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if match {
		t.Error("Expected frontend profile NOT to match Go project")
	}
}

func TestSuggestProfileSelectsFrontendForNextJS(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Next.js + Tailwind + shadcn project
	packageJSON := `{
  "name": "my-nextjs-app",
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "next.config.mjs"), []byte("export default {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "tailwind.config.ts"), []byte("export default {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "components.json"), []byte(`{"style": "default"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Load all embedded profiles
	profiles, err := ListEmbeddedProfiles()
	if err != nil {
		t.Fatalf("Failed to list embedded profiles: %v", err)
	}

	// Find matching profiles
	matches := FindMatchingProfiles(tmpDir, profiles)

	if len(matches) == 0 {
		t.Fatal("Expected at least one profile to match Next.js project")
	}

	// Both frontend and frontend-full should match
	foundFrontend := false
	foundFrontendFull := false
	for _, p := range matches {
		if p.Name == "frontend" {
			foundFrontend = true
		}
		if p.Name == "frontend-full" {
			foundFrontendFull = true
		}
	}

	if !foundFrontend {
		t.Error("Expected 'frontend' profile to match Next.js project")
	}
	if !foundFrontendFull {
		t.Error("Expected 'frontend-full' profile to match Next.js project")
	}
}
