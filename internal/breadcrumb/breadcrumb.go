// ABOUTME: Manages per-scope breadcrumbs recording which profile was last applied
// ABOUTME: Enables profile diff and save to default to the last-applied profile
package breadcrumb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const filename = "last-applied.json"

// Entry records when a profile was applied at a scope.
type Entry struct {
	Profile   string    `json:"profile"`
	AppliedAt time.Time `json:"appliedAt"`
}

// File holds per-scope breadcrumb entries.
type File map[string]Entry

// Load reads the breadcrumb file from claudeupHome.
// Returns an empty File if the file does not exist.
func Load(claudeupHome string) (File, error) {
	path := filepath.Join(claudeupHome, filename)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return File{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return f, nil
}

// Save writes the breadcrumb file atomically.
func Save(claudeupHome string, f File) error {
	path := filepath.Join(claudeupHome, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating breadcrumb directory: %w", err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("writing breadcrumb: %w", err)
	}
	return nil
}

// Record writes a breadcrumb entry for the given scopes.
// Preserves existing entries for other scopes. If the existing
// breadcrumb file cannot be read, returns the error rather than
// silently discarding existing entries.
func Record(claudeupHome, profileName string, scopes []string) error {
	if profileName == "" {
		return fmt.Errorf("breadcrumb: profile name must not be empty")
	}
	f, err := Load(claudeupHome)
	if err != nil {
		return fmt.Errorf("loading existing breadcrumb: %w", err)
	}
	now := time.Now().UTC()
	for _, scope := range scopes {
		f[scope] = Entry{
			Profile:   profileName,
			AppliedAt: now,
		}
	}
	return Save(claudeupHome, f)
}

// Remove deletes breadcrumb entries referencing the given profile name.
// Deletes the breadcrumb file entirely when no entries remain.
func Remove(claudeupHome, profileName string) error {
	f, err := Load(claudeupHome)
	if err != nil {
		return fmt.Errorf("loading breadcrumb for removal: %w", err)
	}
	if len(f) == 0 {
		return nil
	}
	changed := false
	for scope, entry := range f {
		if entry.Profile == profileName {
			delete(f, scope)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if len(f) == 0 {
		err := os.Remove(filepath.Join(claudeupHome, filename))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing empty breadcrumb file: %w", err)
		}
		return nil
	}
	return Save(claudeupHome, f)
}

// Rename updates breadcrumb entries from oldName to newName.
func Rename(claudeupHome, oldName, newName string) error {
	f, err := Load(claudeupHome)
	if err != nil {
		return fmt.Errorf("loading breadcrumb for rename: %w", err)
	}
	if len(f) == 0 {
		return nil
	}
	changed := false
	for scope, entry := range f {
		if entry.Profile == oldName {
			entry.Profile = newName
			f[scope] = entry
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return Save(claudeupHome, f)
}

// HighestPrecedence returns the profile name and scope for the highest-precedence
// breadcrumb entry (local > project > user). Returns empty strings if no entries exist.
func HighestPrecedence(f File) (profileName, scope string) {
	for _, s := range []string{"local", "project", "user"} {
		if entry, ok := f[s]; ok {
			return entry.Profile, s
		}
	}
	return "", ""
}

// ForScope returns the breadcrumb entry for a specific scope.
func ForScope(f File, scope string) (profileName string, appliedAt time.Time, ok bool) {
	entry, exists := f[scope]
	if !exists {
		return "", time.Time{}, false
	}
	return entry.Profile, entry.AppliedAt, true
}
