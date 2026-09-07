// ABOUTME: Tests for the pre-flight missing-extension check
// ABOUTME: Verifies per-scope matching rules and that nothing is written
package profile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeExtFile creates a file under <claudeupHome>/ext/<category>/<name>.
func writeExtFile(t *testing.T, claudeupHome, category, name string) {
	t.Helper()
	dir := filepath.Join(claudeupHome, "ext", category)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("# "+name), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func TestMissingExtensionsLegacyProfile(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	writeExtFile(t, claudeupHome, "rules", "present.md")
	writeExtFile(t, claudeupHome, "agents", "reviewer.md")

	p := &Profile{
		Name: "legacy",
		Extensions: &ExtensionSettings{
			Agents: []string{"reviewer"},
			Rules:  []string{"present.md", "missing.md"},
			Hooks:  []string{"no-such-hook"},
		},
	}

	missing, err := MissingExtensions(p, claudeDir, claudeupHome)
	if err != nil {
		t.Fatalf("MissingExtensions() error = %v", err)
	}

	want := []string{"hooks/no-such-hook", "rules/missing.md"}
	if !reflect.DeepEqual(missing, want) {
		t.Errorf("MissingExtensions() = %v, want %v", missing, want)
	}

	// The check must not enable anything.
	if _, err := os.Lstat(filepath.Join(claudeDir, "rules", "present.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("MissingExtensions() created a symlink; Lstat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(claudeupHome, "enabled.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("MissingExtensions() wrote enabled.json; Stat error = %v", err)
	}
}

func TestMissingExtensionsMultiScopeProfile(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	writeExtFile(t, claudeupHome, "rules", "present.md")

	p := &Profile{
		Name: "multi",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				// Symlink rules infer the extension, so "present" resolves.
				Extensions: &ExtensionSettings{Rules: []string{"present", "user-missing.md"}},
			},
			Project: &ScopeSettings{
				// Copy rules match literally, so "present" does not resolve.
				Extensions: &ExtensionSettings{Rules: []string{"present", "present.md"}},
			},
			Local: &ScopeSettings{
				Extensions: &ExtensionSettings{Rules: []string{"local-missing.md"}},
			},
		},
	}

	missing, err := MissingExtensions(p, claudeDir, claudeupHome)
	if err != nil {
		t.Fatalf("MissingExtensions() error = %v", err)
	}

	want := []string{
		"user: rules/user-missing.md",
		"project: rules/present",
		"local: rules/local-missing.md",
	}
	if !reflect.DeepEqual(missing, want) {
		t.Errorf("MissingExtensions() = %v, want %v", missing, want)
	}
}

func TestMissingExtensionsNothingMissing(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	writeExtFile(t, claudeupHome, "rules", "present.md")

	cases := map[string]*Profile{
		"nil profile":      nil,
		"no extensions":    {Name: "plain"},
		"legacy present":   {Name: "ok", Extensions: &ExtensionSettings{Rules: []string{"present.md"}}},
		"multi nil scopes": {Name: "multi", PerScope: &PerScopeSettings{}},
	}

	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			missing, err := MissingExtensions(p, claudeDir, claudeupHome)
			if err != nil {
				t.Fatalf("MissingExtensions() error = %v", err)
			}
			if len(missing) != 0 {
				t.Errorf("MissingExtensions() = %v, want []", missing)
			}
		})
	}
}
