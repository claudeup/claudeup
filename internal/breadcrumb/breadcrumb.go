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
	Profile    string    `json:"profile"`
	AppliedAt  time.Time `json:"appliedAt"`
	ProjectDir string    `json:"projectDir,omitempty"`
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

// Save writes the breadcrumb file atomically (write-tmp + rename).
// Concurrent writers produce a last-write-wins result, which is acceptable
// since the breadcrumb is a convenience hint, not a source of truth.
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
// projectDir is stored on project/local scope entries so breadcrumbs
// can be filtered by directory later. User-scope entries never store
// a project directory. projectDir is normalized via filepath.EvalSymlinks.
// Preserves existing entries for other scopes. If the existing
// breadcrumb file cannot be read, returns the error rather than
// silently discarding existing entries.
func Record(claudeupHome, profileName, projectDir string, scopes []string) error {
	if profileName == "" {
		return fmt.Errorf("breadcrumb: profile name must not be empty")
	}
	f, err := Load(claudeupHome)
	if err != nil {
		return fmt.Errorf("loading existing breadcrumb: %w", err)
	}

	resolved := projectDir
	if resolved != "" {
		if r, err := filepath.EvalSymlinks(resolved); err == nil {
			resolved = r
		}
	}

	now := time.Now().UTC()
	for _, scope := range scopes {
		entry := Entry{
			Profile:   profileName,
			AppliedAt: now,
		}
		if scope != "user" && resolved != "" {
			entry.ProjectDir = resolved
		}
		f[scope] = entry
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

// FilterByDir returns a new File containing only entries relevant to cwd.
// User-scope entries are always included. Project/local entries are included
// only when their ProjectDir matches cwd (both normalized via EvalSymlinks).
// Project/local entries with empty ProjectDir (pre-fix breadcrumbs) are excluded.
func FilterByDir(f File, cwd string) File {
	result := make(File, len(f))
	if f == nil {
		return result
	}

	resolved := cwd
	if r, err := filepath.EvalSymlinks(resolved); err == nil {
		resolved = r
	}

	for scope, entry := range f {
		if scope == "user" {
			result[scope] = entry
			continue
		}
		// Project/local: require matching directory
		if entry.ProjectDir == "" {
			continue
		}
		entryDir := entry.ProjectDir
		if r, err := filepath.EvalSymlinks(entryDir); err == nil {
			entryDir = r
		}
		if entryDir == resolved {
			result[scope] = entry
		}
	}
	return result
}

// ForScope returns the breadcrumb entry for a specific scope.
func ForScope(f File, scope string) (profileName string, appliedAt time.Time, ok bool) {
	entry, exists := f[scope]
	if !exists {
		return "", time.Time{}, false
	}
	return entry.Profile, entry.AppliedAt, true
}
