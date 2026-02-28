# Last-Applied Breadcrumb Implementation Plan

**Goal:** Record which profile was last applied at each scope so `profile diff` and `profile save` can default to it.

**Architecture:** A new `internal/breadcrumb` package handles read/write of a per-scope JSON file at `~/.claudeup/last-applied.json`. Commands in `profile_cmd.go` write the breadcrumb on apply and read it when diff/save are invoked without arguments. Delete and rename commands maintain breadcrumb consistency.

**Tech Stack:** Go, Cobra CLI framework, Ginkgo/Gomega test framework

**Design:** See `docs/plans/2026-02-27-last-applied-breadcrumb-design.md`

---

### Task 1: Breadcrumb Package

**Files:**

- Create: `internal/breadcrumb/breadcrumb.go`
- Test: `internal/breadcrumb/breadcrumb_test.go`

**Step 1: Write the failing tests**

Create `internal/breadcrumb/breadcrumb_test.go`:

```go
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
		name          string
		file          File
		wantProfile   string
		wantScope     string
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/breadcrumb/... -v`
Expected: Compilation failure (package does not exist)

**Step 3: Write the implementation**

Create `internal/breadcrumb/breadcrumb.go`:

```go
// ABOUTME: Manages per-scope breadcrumbs recording which profile was last applied
// ABOUTME: Enables profile diff and save to default to the last-applied profile
package breadcrumb

import (
	"encoding/json"
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
	data, err := os.ReadFile(filepath.Join(claudeupHome, filename))
	if os.IsNotExist(err) {
		return File{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f, nil
}

// Save writes the breadcrumb file atomically.
func Save(claudeupHome string, f File) error {
	path := filepath.Join(claudeupHome, filename)
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Record writes a breadcrumb entry for the given scopes.
// Preserves existing entries for other scopes.
func Record(claudeupHome, profileName string, scopes []string) error {
	f, err := Load(claudeupHome)
	if err != nil {
		f = File{}
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
func Remove(claudeupHome, profileName string) error {
	f, err := Load(claudeupHome)
	if err != nil || len(f) == 0 {
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
		return os.Remove(filepath.Join(claudeupHome, filename))
	}
	return Save(claudeupHome, f)
}

// Rename updates breadcrumb entries from oldName to newName.
func Rename(claudeupHome, oldName, newName string) error {
	f, err := Load(claudeupHome)
	if err != nil || len(f) == 0 {
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/breadcrumb/... -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/breadcrumb/breadcrumb.go internal/breadcrumb/breadcrumb_test.go
git commit -m "feat: add breadcrumb package for last-applied profile tracking"
```

---

### Task 2: Write Breadcrumb on Profile Apply

**Files:**

- Modify: `internal/commands/profile_cmd.go`
- Test: `test/acceptance/profile_breadcrumb_test.go` (new)
- Modify: `test/helpers/testenv.go` (add breadcrumb helpers)

**Step 1: Add test helpers for breadcrumbs**

Add to `test/helpers/testenv.go`:

```go
// WriteBreadcrumb creates a breadcrumb entry in the test environment
func (e *TestEnv) WriteBreadcrumb(scope, profileName string) {
	bc, _ := breadcrumb.Load(e.ClaudeupDir)
	bc[scope] = breadcrumb.Entry{
		Profile:   profileName,
		AppliedAt: time.Now().UTC(),
	}
	Expect(breadcrumb.Save(e.ClaudeupDir, bc)).To(Succeed())
}

// ReadBreadcrumb loads the breadcrumb file from the test environment
func (e *TestEnv) ReadBreadcrumb() breadcrumb.File {
	f, err := breadcrumb.Load(e.ClaudeupDir)
	Expect(err).NotTo(HaveOccurred())
	return f
}

// BreadcrumbExists checks if the breadcrumb file exists
func (e *TestEnv) BreadcrumbExists() bool {
	_, err := os.Stat(filepath.Join(e.ClaudeupDir, "last-applied.json"))
	return err == nil
}
```

Add imports for `breadcrumb` package and `time` to testenv.go.

**Step 2: Write the failing acceptance test**

Create `test/acceptance/profile_breadcrumb_test.go`:

```go
package acceptance

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
)

var _ = Describe("Profile breadcrumb", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("apply writes breadcrumb", func() {
		BeforeEach(func() {
			// Create a minimal profile (no plugins, so apply succeeds without claude CLI)
			env.CreateProfile(&profile.Profile{
				Name:        "test-profile",
				Description: "test",
			})
		})

		It("writes breadcrumb at user scope by default", func() {
			result := env.RunWithInput("y\n", "profile", "apply", "test-profile", "-y")

			// Apply may fail on plugin install, but breadcrumb is written on the
			// "already matches" path since profile is empty and live is empty
			if result.ExitCode == 0 {
				bc := env.ReadBreadcrumb()
				Expect(bc).To(HaveKey("user"))
				Expect(bc["user"].Profile).To(Equal("test-profile"))
			}
		})
	})
})
```

**Step 3: Run test to verify it fails**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "writes breadcrumb" ./test/acceptance/...`
Expected: FAIL (breadcrumb not written yet)

**Step 4: Write the implementation**

Modify `internal/commands/profile_cmd.go`:

Add import: `"github.com/claudeup/claudeup/v5/internal/breadcrumb"`

Add a helper function after `applyProfileWithScope`:

```go
// recordBreadcrumb writes a breadcrumb entry recording which profile was applied.
// Errors are logged but do not fail the operation.
func recordBreadcrumb(name string, scopes []string) {
	if err := breadcrumb.Record(claudeupHome, name, scopes); err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not save breadcrumb: %v", err))
	}
}
```

In `applyProfileWithScope` (around line 918-939), add breadcrumb write for the early-return "already matches" path. Insert before the `if p.SkipPluginDiff {` line:

```go
if !needsApply {
	// Record breadcrumb even when no changes needed -- user applied this profile
	recordBreadcrumb(name, scopesForBreadcrumb(scope, p, wasStack))

	if p.SkipPluginDiff {
		// ... existing code ...
```

After line 1024 (`ui.PrintSuccess("Profile applied!")`), add:

```go
	recordBreadcrumb(name, scopesForBreadcrumb(scope, p, wasStack))
```

Add the scope-determination helper:

```go
// scopesForBreadcrumb determines which scopes a profile apply touched.
func scopesForBreadcrumb(scope profile.Scope, p *profile.Profile, wasStack bool) []string {
	if p.IsMultiScope() || wasStack {
		var scopes []string
		if p.PerScope != nil {
			if p.PerScope.User != nil {
				scopes = append(scopes, "user")
			}
			if p.PerScope.Project != nil {
				scopes = append(scopes, "project")
			}
			if p.PerScope.Local != nil {
				scopes = append(scopes, "local")
			}
		}
		return scopes
	}
	return []string{string(scope)}
}
```

**Step 5: Run test to verify it passes**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "writes breadcrumb" ./test/acceptance/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_breadcrumb_test.go test/helpers/testenv.go
git commit -m "feat: write breadcrumb on profile apply"
```

---

### Task 3: Profile Diff Defaults to Breadcrumb

**Files:**

- Modify: `internal/commands/profile_cmd.go`
- Modify: `test/acceptance/profile_breadcrumb_test.go`

**Step 1: Write the failing acceptance tests**

Add to the `Describe("Profile breadcrumb")` block in `profile_breadcrumb_test.go`:

```go
Describe("diff with no args", func() {
	BeforeEach(func() {
		// Create a profile with a plugin at user scope
		env.CreateProfile(&profile.Profile{
			Name: "my-setup",
			PerScope: &profile.PerScopeSettings{
				User: &profile.ScopeSettings{
					Plugins: []string{"extra-plugin@marketplace"},
				},
			},
		})
	})

	It("uses highest-precedence breadcrumb", func() {
		env.WriteBreadcrumb("user", "my-setup")

		result := env.Run("profile", "diff")

		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("my-setup"))
	})

	It("prefers project breadcrumb over user", func() {
		env.WriteBreadcrumb("user", "some-other")
		env.WriteBreadcrumb("project", "my-setup")

		// Create the other profile too
		env.CreateProfile(&profile.Profile{Name: "some-other"})

		result := env.Run("profile", "diff")

		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("my-setup"))
	})

	It("errors when no breadcrumb exists", func() {
		result := env.Run("profile", "diff")

		Expect(result.ExitCode).To(Equal(1))
		Expect(result.Stderr).To(ContainSubstring("No profile has been applied"))
	})

	It("errors when breadcrumbed profile is deleted", func() {
		env.WriteBreadcrumb("user", "deleted-profile")

		result := env.Run("profile", "diff")

		Expect(result.ExitCode).To(Equal(1))
		Expect(result.Stderr).To(ContainSubstring("no longer exists"))
	})

	It("explicit name still works", func() {
		env.WriteBreadcrumb("user", "my-setup")

		result := env.Run("profile", "diff", "my-setup")

		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("my-setup"))
	})
})
```

**Step 2: Run tests to verify they fail**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "diff with no args" ./test/acceptance/...`
Expected: FAIL (diff requires exactly 1 arg)

**Step 3: Write the implementation**

In the `profileDiffCmd` definition (around line 185-203), change:

- `Use:` from `"diff <name>"` to `"diff [name]"`
- `Args:` from `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)`
- Update `Long:` to mention the breadcrumb default behavior

In `runProfileDiff` (line 1688), add breadcrumb resolution before the existing logic:

```go
func runProfileDiff(cmd *cobra.Command, args []string) error {
	if profileDiffOriginal {
		return runProfileDiffOriginal(cmd, args)
	}

	var name string
	var breadcrumbScope string

	if len(args) > 0 {
		name = args[0]
	} else {
		// Load breadcrumb for default profile
		bc, err := breadcrumb.Load(claudeupHome)
		if err != nil {
			return fmt.Errorf("failed to read breadcrumb: %w", err)
		}

		// Resolve scope: explicit --scope flag or highest precedence
		resolvedScope, err := resolveScopeFlags(profileDiffScope, profileDiffUser, profileDiffProject, profileDiffLocal)
		if err != nil {
			return err
		}

		if resolvedScope != "" {
			profileName, appliedAt, ok := breadcrumb.ForScope(bc, resolvedScope)
			if !ok {
				return fmt.Errorf("no profile has been applied at %s scope. Run: claudeup profile diff <name>", resolvedScope)
			}
			name = profileName
			breadcrumbScope = fmt.Sprintf("applied %s, %s scope", appliedAt.Format("Jan 2"), resolvedScope)
		} else {
			profileName, scope := breadcrumb.HighestPrecedence(bc)
			if profileName == "" {
				return fmt.Errorf("No profile has been applied yet. Run: claudeup profile diff <name>")
			}
			name = profileName
			entry := bc[scope]
			breadcrumbScope = fmt.Sprintf("applied %s, %s scope", entry.AppliedAt.Format("Jan 2"), scope)
		}
	}

	profilesDir := getProfilesDir()

	// Load saved profile (disk first, fallback to embedded)
	saved, err := loadProfileWithFallback(profilesDir, name)
	if err != nil {
		var ambigErr *profile.AmbiguousProfileError
		if errors.As(err, &ambigErr) {
			return ambigErr
		}
		if breadcrumbScope != "" {
			return fmt.Errorf("profile %q no longer exists. Run: claudeup profile diff <name>", name)
		}
		return fmt.Errorf("profile '%s' not found", name)
	}

	if breadcrumbScope != "" {
		fmt.Printf("Comparing against %q (%s)\n\n", name, breadcrumbScope)
	}

	// ... rest of existing diff logic (snapshot, normalize, compute diff, display) ...
```

Also add scope flag variables and register them. Add to the flags block (around line 24-29):

```go
var (
	profileDiffOriginal bool
	profileDiffScope    string
	profileDiffUser     bool
	profileDiffProject  bool
	profileDiffLocal    bool
)
```

In `init()`, add scope flags to profileDiffCmd (after line 554):

```go
profileDiffCmd.Flags().StringVar(&profileDiffScope, "scope", "", "Diff against breadcrumb for specified scope: user, project, local")
profileDiffCmd.Flags().BoolVar(&profileDiffUser, "user", false, "Diff against breadcrumb for user scope")
profileDiffCmd.Flags().BoolVar(&profileDiffProject, "project", false, "Diff against breadcrumb for project scope")
profileDiffCmd.Flags().BoolVar(&profileDiffLocal, "local", false, "Diff against breadcrumb for local scope")
```

**Step 4: Run tests to verify they pass**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "diff with no args" ./test/acceptance/...`
Expected: All PASS

**Step 5: Also run existing diff tests to verify no regressions**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "Profile diff" ./test/acceptance/...`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_breadcrumb_test.go
git commit -m "feat: profile diff defaults to last-applied breadcrumb"
```

---

### Task 4: Profile Save Defaults to Breadcrumb

**Files:**

- Modify: `internal/commands/profile_cmd.go`
- Modify: `test/acceptance/profile_breadcrumb_test.go`

**Step 1: Write the failing acceptance tests**

Add to the `Describe("Profile breadcrumb")` block:

```go
Describe("save with no args", func() {
	It("saves to breadcrumbed profile name", func() {
		// Create original profile
		env.CreateProfile(&profile.Profile{
			Name:        "my-setup",
			Description: "original",
		})
		env.WriteBreadcrumb("user", "my-setup")

		// Add a plugin to live state so there's something to save
		env.CreateInstalledPlugins(map[string]interface{}{
			"new-plugin@marketplace": map[string]interface{}{
				"scope": "user",
			},
		})

		result := env.RunWithInput("y\n", "profile", "save")

		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("my-setup"))
	})

	It("errors when no breadcrumb exists", func() {
		result := env.Run("profile", "save")

		Expect(result.ExitCode).To(Equal(1))
		Expect(result.Stderr).To(ContainSubstring("No profile has been applied"))
	})

	It("explicit name still works", func() {
		result := env.RunWithInput("y\n", "profile", "save", "explicit-name")

		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("explicit-name"))
	})
})
```

**Step 2: Run tests to verify they fail**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "save with no args" ./test/acceptance/...`
Expected: FAIL (save requires exactly 1 arg)

**Step 3: Write the implementation**

In the `profileSaveCmd` definition (around line 102-124), change:

- `Use:` from `"save <name>"` to `"save [name]"`
- `Args:` from `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)`
- Update `Long:` and `Example:` to mention breadcrumb default

In `runProfileSave` (line 1086), add breadcrumb resolution:

```go
func runProfileSave(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Profiles are always saved to user profiles directory
	profilesDir := getProfilesDir()

	// Resolve scope flags
	resolvedScope, err := resolveScopeFlags(profileSaveScope, profileSaveUser, profileSaveProject, profileSaveLocal)
	if err != nil {
		return err
	}
	if resolvedScope != "" {
		if err := claude.ValidateScope(resolvedScope); err != nil {
			return err
		}
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Load breadcrumb for default profile name
		bc, err := breadcrumb.Load(claudeupHome)
		if err != nil {
			return fmt.Errorf("failed to read breadcrumb: %w", err)
		}

		var bcScope string
		if resolvedScope != "" {
			profileName, appliedAt, ok := breadcrumb.ForScope(bc, resolvedScope)
			if !ok {
				return fmt.Errorf("no profile has been applied at %s scope. Run: claudeup profile save <name>", resolvedScope)
			}
			name = profileName
			bcScope = fmt.Sprintf("applied %s, %s scope", appliedAt.Format("Jan 2"), resolvedScope)
		} else {
			profileName, scope := breadcrumb.HighestPrecedence(bc)
			if profileName == "" {
				return fmt.Errorf("No profile has been applied yet. Run: claudeup profile save <name>")
			}
			name = profileName
			entry := bc[scope]
			bcScope = fmt.Sprintf("applied %s, %s scope", entry.AppliedAt.Format("Jan 2"), scope)
		}
		fmt.Printf("Saving to %q (%s)\n\n", name, bcScope)
	}

	// "current" is reserved as a keyword for live status view
	if name == "current" {
		return fmt.Errorf("'current' is a reserved name. Use a different profile name")
	}

	// ... rest of existing save logic ...
```

**Step 4: Run tests to verify they pass**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "save with no args" ./test/acceptance/...`
Expected: All PASS

**Step 5: Run existing save tests for regressions**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "Profile save" ./test/acceptance/...`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_breadcrumb_test.go
git commit -m "feat: profile save defaults to last-applied breadcrumb"
```

---

### Task 5: Profile Delete Cleans Breadcrumb

**Files:**

- Modify: `internal/commands/profile_cmd.go`
- Modify: `test/acceptance/profile_breadcrumb_test.go`

**Step 1: Write the failing acceptance test**

Add to `Describe("Profile breadcrumb")`:

```go
Describe("delete cleans breadcrumb", func() {
	It("removes breadcrumb entry for deleted profile", func() {
		env.CreateProfile(&profile.Profile{
			Name: "to-delete",
		})
		env.WriteBreadcrumb("user", "to-delete")
		env.WriteBreadcrumb("project", "keep-this")

		result := env.RunWithInput("y\n", "profile", "delete", "to-delete")

		Expect(result.ExitCode).To(Equal(0))

		bc := env.ReadBreadcrumb()
		Expect(bc).NotTo(HaveKey("user"))
		Expect(bc).To(HaveKey("project"))
		Expect(bc["project"].Profile).To(Equal("keep-this"))
	})
})
```

**Step 2: Run test to verify it fails**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "delete cleans breadcrumb" ./test/acceptance/...`
Expected: FAIL (breadcrumb entry not removed)

**Step 3: Write the implementation**

In `runProfileDelete` (around line 2549, after `os.Remove` succeeds), add:

```go
	// Clean breadcrumb entries referencing the deleted profile
	if err := breadcrumb.Remove(claudeupHome, name); err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not clean breadcrumb: %v", err))
	}

	ui.PrintSuccess(fmt.Sprintf("Deleted profile %q", name))
```

**Step 4: Run test to verify it passes**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "delete cleans breadcrumb" ./test/acceptance/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_breadcrumb_test.go
git commit -m "feat: profile delete removes breadcrumb entries"
```

---

### Task 6: Profile Rename Updates Breadcrumb

**Files:**

- Modify: `internal/commands/profile_cmd.go`
- Modify: `test/acceptance/profile_breadcrumb_test.go`

**Step 1: Write the failing acceptance test**

Add to `Describe("Profile breadcrumb")`:

```go
Describe("rename updates breadcrumb", func() {
	It("updates breadcrumb entry to new name", func() {
		env.CreateProfile(&profile.Profile{
			Name: "old-name",
		})
		env.WriteBreadcrumb("user", "old-name")

		result := env.Run("profile", "rename", "old-name", "new-name")

		Expect(result.ExitCode).To(Equal(0))

		bc := env.ReadBreadcrumb()
		Expect(bc).To(HaveKey("user"))
		Expect(bc["user"].Profile).To(Equal("new-name"))
	})
})
```

**Step 2: Run test to verify it fails**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "rename updates breadcrumb" ./test/acceptance/...`
Expected: FAIL (breadcrumb still has old-name)

**Step 3: Write the implementation**

In `runProfileRename` (around line 2649, after the success message), add:

```go
	// Update breadcrumb entries referencing the old name
	if err := breadcrumb.Rename(claudeupHome, oldName, newName); err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not update breadcrumb: %v", err))
	}

	ui.PrintSuccess(fmt.Sprintf("Renamed profile %q to %q", oldName, newName))
```

Note: Move the existing `ui.PrintSuccess` line and add the breadcrumb call before it, or add the breadcrumb call just before the existing `ui.PrintSuccess`.

**Step 4: Run test to verify it passes**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "rename updates breadcrumb" ./test/acceptance/...`
Expected: PASS

**Step 5: Run all breadcrumb tests**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -focus "breadcrumb" ./test/acceptance/...`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/commands/profile_cmd.go test/acceptance/profile_breadcrumb_test.go
git commit -m "feat: profile rename updates breadcrumb entries"
```

---

### Task 7: Full Test Suite and Design Doc Update

**Files:**

- Modify: `docs/plans/2026-02-27-last-applied-breadcrumb-design.md`

**Step 1: Run the full test suite**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Run acceptance tests specifically**

Run: `go run github.com/onsi/ginkgo/v2/ginkgo -v ./test/acceptance/...`
Expected: All PASS

**Step 3: Update design doc to note rename exists**

In `docs/plans/2026-02-27-last-applied-breadcrumb-design.md`, update the line about profile rename:

Change:

```
**Profile rename** -- No rename command exists. If added later, it should update
breadcrumbs.
```

To:

```
**Profile rename** -- The rename command updates breadcrumb entries.
```

**Step 4: Commit**

```bash
git add docs/plans/2026-02-27-last-applied-breadcrumb-design.md
git commit -m "docs: update design doc with rename breadcrumb behavior"
```
