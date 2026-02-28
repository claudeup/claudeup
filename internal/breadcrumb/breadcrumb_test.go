package breadcrumb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFile(t *testing.T) {
	f, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f) != 0 {
		t.Fatalf("expected empty file, got %d entries", len(f))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	f := File{
		"user": {Profile: "base-tools", AppliedAt: time.Date(2026, 2, 27, 21, 0, 0, 0, time.UTC)},
	}
	if err := Save(dir, f); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	entry, ok := loaded["user"]
	if !ok {
		t.Fatal("missing user entry")
	}
	if entry.Profile != "base-tools" {
		t.Fatalf("expected base-tools, got %s", entry.Profile)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Save should create the parent directory if it doesn't exist
	dir := filepath.Join(t.TempDir(), "nonexistent", "subdir")
	f := File{
		"user": {Profile: "test", AppliedAt: time.Date(2026, 2, 27, 21, 0, 0, 0, time.UTC)},
	}
	if err := Save(dir, f); err != nil {
		t.Fatalf("save to nonexistent dir failed: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load after save failed: %v", err)
	}
	if loaded["user"].Profile != "test" {
		t.Fatalf("expected test, got %s", loaded["user"].Profile)
	}
}

func TestRecord(t *testing.T) {
	dir := t.TempDir()

	// Record user scope
	if err := Record(dir, "base-tools", []string{"user"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Record project scope (preserves user)
	if err := Record(dir, "my-project", []string{"project"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	f, _ := Load(dir)
	if f["user"].Profile != "base-tools" {
		t.Fatalf("user entry lost: got %s", f["user"].Profile)
	}
	if f["project"].Profile != "my-project" {
		t.Fatalf("expected my-project, got %s", f["project"].Profile)
	}
}

func TestRecordMultiScope(t *testing.T) {
	dir := t.TempDir()

	if err := Record(dir, "my-stack", []string{"user", "project"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	f, _ := Load(dir)
	if f["user"].Profile != "my-stack" {
		t.Fatalf("expected my-stack at user, got %s", f["user"].Profile)
	}
	if f["project"].Profile != "my-stack" {
		t.Fatalf("expected my-stack at project, got %s", f["project"].Profile)
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()

	// Set up two entries
	Record(dir, "base-tools", []string{"user"})
	Record(dir, "my-project", []string{"project"})

	// Remove base-tools
	if err := Remove(dir, "base-tools"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	f, _ := Load(dir)
	if _, ok := f["user"]; ok {
		t.Fatal("user entry should be removed")
	}
	if f["project"].Profile != "my-project" {
		t.Fatal("project entry should be preserved")
	}
}

func TestRemoveLastEntry(t *testing.T) {
	dir := t.TempDir()

	Record(dir, "only-one", []string{"user"})
	Remove(dir, "only-one")

	// File should be deleted when empty
	_, err := os.Stat(filepath.Join(dir, "last-applied.json"))
	if !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted when empty")
	}
}

func TestRemoveNonexistent(t *testing.T) {
	dir := t.TempDir()

	// Should not error when no file exists
	if err := Remove(dir, "nope"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRename(t *testing.T) {
	dir := t.TempDir()

	Record(dir, "old-name", []string{"user", "project"})

	if err := Rename(dir, "old-name", "new-name"); err != nil {
		t.Fatalf("rename failed: %v", err)
	}

	f, _ := Load(dir)
	if f["user"].Profile != "new-name" {
		t.Fatalf("expected new-name at user, got %s", f["user"].Profile)
	}
	if f["project"].Profile != "new-name" {
		t.Fatalf("expected new-name at project, got %s", f["project"].Profile)
	}
}

func TestRenamePreservesOtherEntries(t *testing.T) {
	dir := t.TempDir()

	Record(dir, "rename-me", []string{"user"})
	Record(dir, "keep-me", []string{"project"})

	Rename(dir, "rename-me", "renamed")

	f, _ := Load(dir)
	if f["user"].Profile != "renamed" {
		t.Fatalf("expected renamed, got %s", f["user"].Profile)
	}
	if f["project"].Profile != "keep-me" {
		t.Fatalf("expected keep-me preserved, got %s", f["project"].Profile)
	}
}

func TestHighestPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		file        File
		wantProfile string
		wantScope   string
	}{
		{"empty", File{}, "", ""},
		{"user only", File{"user": {Profile: "a"}}, "a", "user"},
		{"project wins over user", File{
			"user":    {Profile: "a"},
			"project": {Profile: "b"},
		}, "b", "project"},
		{"local wins over all", File{
			"user":    {Profile: "a"},
			"project": {Profile: "b"},
			"local":   {Profile: "c"},
		}, "c", "local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, scope := HighestPrecedence(tt.file)
			if name != tt.wantProfile || scope != tt.wantScope {
				t.Fatalf("got (%s, %s), want (%s, %s)", name, scope, tt.wantProfile, tt.wantScope)
			}
		})
	}
}

func TestLoadCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last-applied.json")
	os.WriteFile(path, []byte("{invalid json"), 0600)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}

func TestRecordWithCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last-applied.json")
	os.WriteFile(path, []byte("{invalid json"), 0600)

	// Record should propagate the load error, not silently overwrite
	err := Record(dir, "my-profile", []string{"user"})
	if err == nil {
		t.Fatal("expected error when existing breadcrumb is corrupt")
	}
}

func TestRecordEmptyScopes(t *testing.T) {
	dir := t.TempDir()

	// Record with empty scopes should still succeed (writes file with no entries)
	if err := Record(dir, "my-profile", []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f, _ := Load(dir)
	if len(f) != 0 {
		t.Fatalf("expected empty file, got %d entries", len(f))
	}
}

func TestRecordEmptyProfileName(t *testing.T) {
	dir := t.TempDir()

	err := Record(dir, "", []string{"user"})
	if err == nil {
		t.Fatal("expected error for empty profile name")
	}
}

func TestRemoveWithCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last-applied.json")
	os.WriteFile(path, []byte("{invalid json"), 0600)

	// Remove should propagate the load error
	err := Remove(dir, "my-profile")
	if err == nil {
		t.Fatal("expected error when breadcrumb file is corrupt")
	}
}

func TestRenameNonexistent(t *testing.T) {
	dir := t.TempDir()

	// Should not error when no file exists
	if err := Rename(dir, "old", "new"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenameWithCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last-applied.json")
	os.WriteFile(path, []byte("{invalid json"), 0600)

	// Rename should propagate the load error
	err := Rename(dir, "old", "new")
	if err == nil {
		t.Fatal("expected error when breadcrumb file is corrupt")
	}
}

func TestHighestPrecedenceIgnoresUnknownKeys(t *testing.T) {
	f := File{
		"banana":  {Profile: "fruit"},
		"unknown": {Profile: "mystery"},
	}
	name, scope := HighestPrecedence(f)
	if name != "" || scope != "" {
		t.Fatalf("expected empty result for unknown keys, got (%s, %s)", name, scope)
	}
}

func TestForScope(t *testing.T) {
	f := File{
		"user": {Profile: "base-tools", AppliedAt: time.Date(2026, 2, 27, 21, 0, 0, 0, time.UTC)},
	}

	name, at, ok := ForScope(f, "user")
	if !ok || name != "base-tools" || at.IsZero() {
		t.Fatalf("expected base-tools entry, got (%s, %v, %v)", name, at, ok)
	}

	_, _, ok = ForScope(f, "project")
	if ok {
		t.Fatal("expected no project entry")
	}
}
