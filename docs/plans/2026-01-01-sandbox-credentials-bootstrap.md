# Sandbox Credentials and Bootstrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable credential mounting and profile bootstrapping in sandbox containers.

**Architecture:** Credentials resolve to docker mounts based on profile config and CLI flags. Profile bootstrap writes Claude config files to sandbox state directory on first run. Both integrate into the existing DockerRunner.

**Tech Stack:** Go, Docker CLI, macOS Keychain (security command)

---

## Task 1: Add Credentials Field to SandboxConfig

**Files:**
- Modify: `internal/profile/profile.go:48-57`
- Test: `internal/profile/profile_test.go` (if exists, otherwise skip)

**Step 1: Add Credentials field to SandboxConfig struct**

In `internal/profile/profile.go`, add `Credentials` to the `SandboxConfig` struct:

```go
// SandboxConfig defines sandbox-specific settings for a profile
type SandboxConfig struct {
	// Credentials are credential types to mount (git, ssh, gh)
	Credentials []string `json:"credentials,omitempty"`

	// Secrets are secret names to resolve and inject into the sandbox
	Secrets []string `json:"secrets,omitempty"`

	// Mounts are additional host:container path mappings
	Mounts []SandboxMount `json:"mounts,omitempty"`

	// Env are static environment variables to set
	Env map[string]string `json:"env,omitempty"`
}
```

**Step 2: Update Clone method to copy Credentials**

Find the Clone method's Sandbox section (~line 318) and add:

```go
	// Deep copy Sandbox
	if len(p.Sandbox.Credentials) > 0 {
		clone.Sandbox.Credentials = make([]string, len(p.Sandbox.Credentials))
		copy(clone.Sandbox.Credentials, p.Sandbox.Credentials)
	}
	if len(p.Sandbox.Secrets) > 0 {
```

**Step 3: Run tests to verify no regressions**

Run: `go test ./internal/profile/... -v`
Expected: All existing tests pass

**Step 4: Commit**

```bash
git add internal/profile/profile.go
git commit -m "feat(profile): add Credentials field to SandboxConfig"
```

---

## Task 2: Create Credential Types and Resolution

**Files:**
- Create: `internal/sandbox/credentials.go`
- Create: `internal/sandbox/credentials_test.go`

**Step 1: Write failing test for credential type definitions**

Create `internal/sandbox/credentials_test.go`:

```go
// ABOUTME: Unit tests for credential resolution.
// ABOUTME: Tests credential type mapping and mount generation.
package sandbox

import (
	"testing"
)

func TestCredentialTypes(t *testing.T) {
	t.Run("git credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("git")
		if cred == nil {
			t.Fatal("git credential type not found")
		}
		if cred.SourceSuffix != ".gitconfig" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".gitconfig")
		}
		if cred.TargetPath != "/root/.gitconfig" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.gitconfig")
		}
	})

	t.Run("ssh credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("ssh")
		if cred == nil {
			t.Fatal("ssh credential type not found")
		}
		if cred.SourceSuffix != ".ssh" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".ssh")
		}
		if cred.TargetPath != "/root/.ssh" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.ssh")
		}
	})

	t.Run("gh credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("gh")
		if cred == nil {
			t.Fatal("gh credential type not found")
		}
		if cred.SourceSuffix != ".config/gh" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".config/gh")
		}
		if cred.TargetPath != "/root/.config/gh" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.config/gh")
		}
	})

	t.Run("unknown credential returns nil", func(t *testing.T) {
		cred := GetCredentialType("unknown")
		if cred != nil {
			t.Error("expected nil for unknown credential type")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/sandbox/... -run TestCredentialTypes -v`
Expected: FAIL with "undefined: GetCredentialType"

**Step 3: Implement credential types**

Create `internal/sandbox/credentials.go`:

```go
// ABOUTME: Credential type definitions and resolution for sandbox containers.
// ABOUTME: Maps credential names (git, ssh, gh) to host/container paths.
package sandbox

// CredentialType defines a mountable credential
type CredentialType struct {
	Name         string // "git", "ssh", "gh"
	SourceSuffix string // Path suffix from home dir (e.g., ".gitconfig")
	TargetPath   string // Container path
	NeedsExtract bool   // True if credential needs Keychain extraction (macOS)
}

var credentialTypes = map[string]*CredentialType{
	"git": {
		Name:         "git",
		SourceSuffix: ".gitconfig",
		TargetPath:   "/root/.gitconfig",
		NeedsExtract: false,
	},
	"ssh": {
		Name:         "ssh",
		SourceSuffix: ".ssh",
		TargetPath:   "/root/.ssh",
		NeedsExtract: false,
	},
	"gh": {
		Name:         "gh",
		SourceSuffix: ".config/gh",
		TargetPath:   "/root/.config/gh",
		NeedsExtract: true, // macOS stores in Keychain
	},
}

// GetCredentialType returns the credential type definition, or nil if unknown
func GetCredentialType(name string) *CredentialType {
	return credentialTypes[name]
}

// ValidCredentialTypes returns all valid credential type names
func ValidCredentialTypes() []string {
	return []string{"git", "ssh", "gh"}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/sandbox/... -run TestCredentialTypes -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/sandbox/credentials.go internal/sandbox/credentials_test.go
git commit -m "feat(sandbox): add credential type definitions"
```

---

## Task 3: Implement Credential Merge Logic

**Files:**
- Modify: `internal/sandbox/credentials.go`
- Modify: `internal/sandbox/credentials_test.go`

**Step 1: Write failing test for merge logic**

Add to `internal/sandbox/credentials_test.go`:

```go
func TestMergeCredentials(t *testing.T) {
	t.Run("empty inputs returns empty", func(t *testing.T) {
		result := MergeCredentials(nil, nil, nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %v", result)
		}
	})

	t.Run("profile credentials returned when no overrides", func(t *testing.T) {
		result := MergeCredentials([]string{"git", "ssh"}, nil, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
		if result[0] != "git" || result[1] != "ssh" {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("add credentials extends list", func(t *testing.T) {
		result := MergeCredentials([]string{"git"}, []string{"ssh"}, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
	})

	t.Run("exclude credentials removes from list", func(t *testing.T) {
		result := MergeCredentials([]string{"git", "ssh", "gh"}, nil, []string{"ssh"})
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
		for _, c := range result {
			if c == "ssh" {
				t.Error("ssh should have been excluded")
			}
		}
	})

	t.Run("add and exclude together", func(t *testing.T) {
		result := MergeCredentials([]string{"git"}, []string{"ssh", "gh"}, []string{"gh"})
		// Start: [git], Add: [ssh, gh], Exclude: [gh] -> [git, ssh]
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
	})

	t.Run("ignores unknown credential types", func(t *testing.T) {
		result := MergeCredentials([]string{"git", "unknown"}, nil, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 credential, got %d", len(result))
		}
		if result[0] != "git" {
			t.Errorf("expected git, got %s", result[0])
		}
	})

	t.Run("deduplicates credentials", func(t *testing.T) {
		result := MergeCredentials([]string{"git"}, []string{"git", "ssh"}, nil)
		gitCount := 0
		for _, c := range result {
			if c == "git" {
				gitCount++
			}
		}
		if gitCount != 1 {
			t.Errorf("expected 1 git, got %d", gitCount)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/sandbox/... -run TestMergeCredentials -v`
Expected: FAIL with "undefined: MergeCredentials"

**Step 3: Implement merge logic**

Add to `internal/sandbox/credentials.go`:

```go
// MergeCredentials combines profile credentials with CLI overrides.
// Order: start with profile, add additions, remove exclusions.
// Unknown credential types are silently ignored.
func MergeCredentials(profile, add, exclude []string) []string {
	// Build set from profile
	set := make(map[string]bool)
	for _, c := range profile {
		if GetCredentialType(c) != nil {
			set[c] = true
		}
	}

	// Add CLI additions
	for _, c := range add {
		if GetCredentialType(c) != nil {
			set[c] = true
		}
	}

	// Remove CLI exclusions
	for _, c := range exclude {
		delete(set, c)
	}

	// Convert to sorted slice for deterministic output
	result := make([]string, 0, len(set))
	for _, validType := range ValidCredentialTypes() {
		if set[validType] {
			result = append(result, validType)
		}
	}

	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/sandbox/... -run TestMergeCredentials -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/sandbox/credentials.go internal/sandbox/credentials_test.go
git commit -m "feat(sandbox): add credential merge logic"
```

---

## Task 4: Implement Credential Mount Resolution

**Files:**
- Modify: `internal/sandbox/credentials.go`
- Modify: `internal/sandbox/credentials_test.go`

**Step 1: Write failing test for mount resolution**

Add to `internal/sandbox/credentials_test.go`:

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCredentialMounts(t *testing.T) {
	t.Run("resolves git credential to mount", func(t *testing.T) {
		homeDir := t.TempDir()
		gitconfig := filepath.Join(homeDir, ".gitconfig")
		if err := os.WriteFile(gitconfig, []byte("[user]\nname = Test"), 0644); err != nil {
			t.Fatalf("failed to create gitconfig: %v", err)
		}

		mounts, warnings := ResolveCredentialMounts([]string{"git"}, homeDir, "")
		if len(warnings) > 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		if len(mounts) != 1 {
			t.Fatalf("expected 1 mount, got %d", len(mounts))
		}
		if mounts[0].Host != gitconfig {
			t.Errorf("wrong host path: got %q, want %q", mounts[0].Host, gitconfig)
		}
		if mounts[0].Container != "/root/.gitconfig" {
			t.Errorf("wrong container path: got %q", mounts[0].Container)
		}
		if !mounts[0].ReadOnly {
			t.Error("mount should be read-only")
		}
	})

	t.Run("missing credential warns and skips", func(t *testing.T) {
		homeDir := t.TempDir()
		// No .gitconfig created

		mounts, warnings := ResolveCredentialMounts([]string{"git"}, homeDir, "")
		if len(mounts) != 0 {
			t.Errorf("expected no mounts, got %d", len(mounts))
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
	})

	t.Run("resolves multiple credentials", func(t *testing.T) {
		homeDir := t.TempDir()

		// Create git and ssh
		if err := os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(homeDir, ".ssh"), 0700); err != nil {
			t.Fatal(err)
		}

		mounts, warnings := ResolveCredentialMounts([]string{"git", "ssh"}, homeDir, "")
		if len(warnings) > 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		if len(mounts) != 2 {
			t.Fatalf("expected 2 mounts, got %d", len(mounts))
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/sandbox/... -run TestResolveCredentialMounts -v`
Expected: FAIL with "undefined: ResolveCredentialMounts"

**Step 3: Implement mount resolution**

Add to `internal/sandbox/credentials.go`:

```go
import (
	"os"
	"path/filepath"
)

// ResolveCredentialMounts converts credential names to Docker mounts.
// Returns mounts and any warnings for missing credentials.
// stateDir is used for credentials that need extraction (gh on macOS).
func ResolveCredentialMounts(credentials []string, homeDir, stateDir string) ([]Mount, []string) {
	var mounts []Mount
	var warnings []string

	for _, name := range credentials {
		credType := GetCredentialType(name)
		if credType == nil {
			continue // Unknown type, already filtered by MergeCredentials
		}

		sourcePath := filepath.Join(homeDir, credType.SourceSuffix)

		// Check if source exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			warnings = append(warnings, "credential "+name+" not found at "+sourcePath)
			continue
		}

		// For now, direct mount. macOS Keychain extraction handled in Task 5.
		mounts = append(mounts, Mount{
			Host:      sourcePath,
			Container: credType.TargetPath,
			ReadOnly:  true,
		})
	}

	return mounts, warnings
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/sandbox/... -run TestResolveCredentialMounts -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/sandbox/credentials.go internal/sandbox/credentials_test.go
git commit -m "feat(sandbox): add credential mount resolution"
```

---

## Task 5: Add CLI Flags for Credentials

**Files:**
- Modify: `internal/commands/sandbox.go`

**Step 1: Add flag variables**

In `internal/commands/sandbox.go`, add to the var block (~line 18):

```go
var (
	sandboxProfile    string
	sandboxMounts     []string
	sandboxNoMount    bool
	sandboxSecrets    []string
	sandboxNoSecrets  []string
	sandboxCreds      []string  // NEW
	sandboxNoCreds    []string  // NEW
	sandboxShell      bool
	sandboxClean      bool
	sandboxImage      string
	sandboxEphemeral  bool
	sandboxCopyAuth   bool
	sandboxSync       bool      // NEW
)
```

**Step 2: Register flags in init()**

Add to the init() function:

```go
	sandboxCmd.Flags().StringSliceVar(&sandboxCreds, "creds", nil, "Credentials to mount (git, ssh, gh)")
	sandboxCmd.Flags().StringSliceVar(&sandboxNoCreds, "no-creds", nil, "Credentials to exclude")
	sandboxCmd.Flags().BoolVar(&sandboxSync, "sync", false, "Re-apply profile settings to sandbox")
```

**Step 3: Add Credentials and Sync to Options struct**

In `internal/sandbox/sandbox.go`, add to the Options struct:

```go
type Options struct {
	// ... existing fields ...

	// Credentials are credential type names to mount (git, ssh, gh)
	Credentials []string

	// Sync forces re-application of profile settings
	Sync bool
}
```

**Step 4: Wire up credentials in runSandbox**

In `internal/commands/sandbox.go`, find `runSandbox` and add credential handling after profile loading (~line 112):

```go
	// Merge credentials: profile + CLI additions - CLI exclusions
	var profileCreds []string
	if sandboxProfile != "" && !sandboxEphemeral {
		// ... existing profile loading code ...
		profileCreds = p.Sandbox.Credentials
	}
	opts.Credentials = sandbox.MergeCredentials(profileCreds, sandboxCreds, sandboxNoCreds)
	opts.Sync = sandboxSync
```

**Step 5: Run tests**

Run: `go test ./internal/commands/... -v`
Expected: All existing tests pass

**Step 6: Commit**

```bash
git add internal/commands/sandbox.go internal/sandbox/sandbox.go
git commit -m "feat(sandbox): add --creds, --no-creds, --sync CLI flags"
```

---

## Task 6: Integrate Credential Mounts into Docker Runner

**Files:**
- Modify: `internal/sandbox/docker.go`

**Step 1: Add credential mount resolution to buildArgs**

In `internal/sandbox/docker.go`, find `buildArgs` method and add credential handling after additional mounts (~line 80):

```go
	// Credential mounts
	if len(opts.Credentials) > 0 {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			credMounts, warnings := ResolveCredentialMounts(opts.Credentials, homeDir, "")
			for _, w := range warnings {
				// Log warnings (could use ui.PrintWarning if imported)
				fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
			}
			for _, m := range credMounts {
				mountArg := fmt.Sprintf("%s:%s:ro", m.Host, m.Container)
				args = append(args, "-v", mountArg)
			}
		}
	}
```

**Step 2: Run tests**

Run: `go test ./internal/sandbox/... -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/sandbox/docker.go
git commit -m "feat(sandbox): integrate credential mounts into Docker runner"
```

---

## Task 7: Create Bootstrap Module

**Files:**
- Create: `internal/sandbox/bootstrap.go`
- Create: `internal/sandbox/bootstrap_test.go`

**Step 1: Write failing test for first-run detection**

Create `internal/sandbox/bootstrap_test.go`:

```go
// ABOUTME: Unit tests for profile bootstrap functionality.
// ABOUTME: Tests first-run detection, config writing, and sentinel management.
package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFirstRun(t *testing.T) {
	t.Run("empty directory is first run", func(t *testing.T) {
		stateDir := t.TempDir()
		if !IsFirstRun(stateDir) {
			t.Error("expected first run for empty directory")
		}
	})

	t.Run("directory with sentinel is not first run", func(t *testing.T) {
		stateDir := t.TempDir()
		sentinel := filepath.Join(stateDir, ".bootstrapped")
		if err := os.WriteFile(sentinel, []byte("2026-01-01"), 0644); err != nil {
			t.Fatal(err)
		}
		if IsFirstRun(stateDir) {
			t.Error("expected not first run when sentinel exists")
		}
	})

	t.Run("directory with other files but no sentinel is first run", func(t *testing.T) {
		stateDir := t.TempDir()
		otherFile := filepath.Join(stateDir, "settings.json")
		if err := os.WriteFile(otherFile, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if !IsFirstRun(stateDir) {
			t.Error("expected first run when no sentinel")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/sandbox/... -run TestIsFirstRun -v`
Expected: FAIL with "undefined: IsFirstRun"

**Step 3: Implement IsFirstRun**

Create `internal/sandbox/bootstrap.go`:

```go
// ABOUTME: Profile bootstrap functionality for sandbox containers.
// ABOUTME: Applies profile's Claude configuration on first sandbox run.
package sandbox

import (
	"os"
	"path/filepath"
	"time"
)

const sentinelFile = ".bootstrapped"

// IsFirstRun returns true if the sandbox state directory has not been bootstrapped.
func IsFirstRun(stateDir string) bool {
	sentinel := filepath.Join(stateDir, sentinelFile)
	_, err := os.Stat(sentinel)
	return os.IsNotExist(err)
}

// WriteSentinel marks the sandbox as bootstrapped.
func WriteSentinel(stateDir string) error {
	sentinel := filepath.Join(stateDir, sentinelFile)
	timestamp := time.Now().Format(time.RFC3339)
	return os.WriteFile(sentinel, []byte(timestamp), 0644)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/sandbox/... -run TestIsFirstRun -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/sandbox/bootstrap.go internal/sandbox/bootstrap_test.go
git commit -m "feat(sandbox): add first-run detection for bootstrap"
```

---

## Task 8: Implement Profile Config Bootstrap

**Files:**
- Modify: `internal/sandbox/bootstrap.go`
- Modify: `internal/sandbox/bootstrap_test.go`

**Step 1: Write failing test for bootstrap**

Add to `internal/sandbox/bootstrap_test.go`:

```go
import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/profile"
)

func TestBootstrapFromProfile(t *testing.T) {
	t.Run("writes marketplaces config", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{
			Name: "test",
			Marketplaces: []profile.Marketplace{
				{Source: "github", Repo: "obra/superpowers-marketplace"},
			},
		}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		// Check marketplaces.json was created
		data, err := os.ReadFile(filepath.Join(stateDir, "marketplaces.json"))
		if err != nil {
			t.Fatalf("failed to read marketplaces.json: %v", err)
		}

		var marketplaces []map[string]interface{}
		if err := json.Unmarshal(data, &marketplaces); err != nil {
			t.Fatalf("failed to parse marketplaces.json: %v", err)
		}

		if len(marketplaces) != 1 {
			t.Errorf("expected 1 marketplace, got %d", len(marketplaces))
		}
	})

	t.Run("writes plugins to settings.json", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{"superpowers@superpowers-marketplace"},
		}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(stateDir, "settings.json"))
		if err != nil {
			t.Fatalf("failed to read settings.json: %v", err)
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Fatalf("failed to parse settings.json: %v", err)
		}

		plugins, ok := settings["enabledPlugins"].([]interface{})
		if !ok {
			t.Fatal("enabledPlugins not found or wrong type")
		}
		if len(plugins) != 1 {
			t.Errorf("expected 1 plugin, got %d", len(plugins))
		}
	})

	t.Run("creates sentinel file", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{Name: "test"}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		if IsFirstRun(stateDir) {
			t.Error("should not be first run after bootstrap")
		}
	})

	t.Run("empty profile still creates sentinel", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{Name: "empty"}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		if IsFirstRun(stateDir) {
			t.Error("should not be first run after bootstrap")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/sandbox/... -run TestBootstrapFromProfile -v`
Expected: FAIL with "undefined: BootstrapFromProfile"

**Step 3: Implement BootstrapFromProfile**

Add to `internal/sandbox/bootstrap.go`:

```go
import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/claudeup/claudeup/internal/profile"
)

// BootstrapFromProfile writes Claude configuration files to the sandbox state directory.
// This applies the profile's marketplaces, plugins, and settings to the sandbox.
func BootstrapFromProfile(p *profile.Profile, stateDir string) error {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	// Write marketplaces.json if profile has marketplaces
	if len(p.Marketplaces) > 0 {
		if err := writeMarketplaces(p.Marketplaces, stateDir); err != nil {
			return err
		}
	}

	// Write settings.json with plugins if profile has plugins
	if len(p.Plugins) > 0 {
		if err := writeSettings(p.Plugins, stateDir); err != nil {
			return err
		}
	}

	// Mark as bootstrapped
	return WriteSentinel(stateDir)
}

func writeMarketplaces(marketplaces []profile.Marketplace, stateDir string) error {
	// Convert to Claude's marketplace format
	var data []map[string]interface{}
	for _, m := range marketplaces {
		entry := map[string]interface{}{
			"source": m.Source,
		}
		if m.Repo != "" {
			entry["repo"] = m.Repo
		}
		if m.URL != "" {
			entry["url"] = m.URL
		}
		data = append(data, entry)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(stateDir, "marketplaces.json"), jsonData, 0644)
}

func writeSettings(plugins []string, stateDir string) error {
	settings := map[string]interface{}{
		"enabledPlugins": plugins,
	}

	jsonData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(stateDir, "settings.json"), jsonData, 0644)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/sandbox/... -run TestBootstrapFromProfile -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/sandbox/bootstrap.go internal/sandbox/bootstrap_test.go
git commit -m "feat(sandbox): implement profile bootstrap to sandbox"
```

---

## Task 9: Integrate Bootstrap into Docker Runner

**Files:**
- Modify: `internal/sandbox/docker.go`
- Modify: `internal/commands/sandbox.go`

**Step 1: Add bootstrap call to Run method**

In `internal/sandbox/docker.go`, modify the `Run` method to call bootstrap:

```go
func (r *DockerRunner) Run(opts Options) error {
	if err := r.Available(); err != nil {
		return err
	}

	// Bootstrap profile settings on first run or sync
	if opts.Profile != "" {
		stateDir, err := StateDir(r.ClaudePMDir, opts.Profile)
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		if IsFirstRun(stateDir) || opts.Sync {
			// Load profile and bootstrap
			profilesDir := filepath.Join(r.ClaudePMDir, "profiles")
			p, err := profile.Load(profilesDir, opts.Profile)
			if err != nil {
				return fmt.Errorf("failed to load profile for bootstrap: %w", err)
			}
			if err := BootstrapFromProfile(p, stateDir); err != nil {
				return fmt.Errorf("failed to bootstrap sandbox: %w", err)
			}
		}
	}

	args := r.buildArgs(opts)
	// ... rest of method
}
```

**Step 2: Add import for profile package**

Add to imports in `internal/sandbox/docker.go`:

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/internal/profile"
)
```

**Step 3: Update printSandboxInfo to show sync status**

In `internal/commands/sandbox.go`, update `printSandboxInfo`:

```go
func printSandboxInfo(opts sandbox.Options) {
	fmt.Println(ui.RenderSection("Sandbox", -1))
	fmt.Println()

	if opts.Profile != "" {
		status := ui.Bold(opts.Profile) + " " + ui.Muted("(persistent)")
		if opts.Sync {
			status += " " + ui.Muted("[sync]")
		}
		fmt.Println(ui.RenderDetail("Profile", status))
	} else {
		fmt.Println(ui.RenderDetail("Mode", "ephemeral"))
	}

	// ... rest of function

	if len(opts.Credentials) > 0 {
		fmt.Println(ui.RenderDetail("Credentials", strings.Join(opts.Credentials, ", ")))
	}

	// ... rest of function
}
```

**Step 4: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 5: Commit**

```bash
git add internal/sandbox/docker.go internal/commands/sandbox.go
git commit -m "feat(sandbox): integrate bootstrap into Docker runner"
```

---

## Task 10: Add Acceptance Tests

**Files:**
- Create: `test/acceptance/sandbox_credentials_test.go`

**Step 1: Write acceptance test for credentials**

Create `test/acceptance/sandbox_credentials_test.go`:

```go
// ABOUTME: Acceptance tests for sandbox credential mounting.
// ABOUTME: Tests --creds and --no-creds CLI behavior with real Docker.
package acceptance_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sandbox credentials", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = NewTestEnv(binaryPath)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("--creds flag", func() {
		It("shows credentials in help output", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--creds"))
			Expect(result.Stdout).To(ContainSubstring("--no-creds"))
		})
	})

	Describe("--sync flag", func() {
		It("shows sync in help output", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--sync"))
		})
	})
})
```

**Step 2: Run acceptance tests**

Run: `go test ./test/acceptance/... -run "sandbox credentials" -v`
Expected: Tests pass (only testing help output, no Docker needed)

**Step 3: Commit**

```bash
git add test/acceptance/sandbox_credentials_test.go
git commit -m "test(sandbox): add acceptance tests for credential flags"
```

---

## Task 11: Run Full Test Suite and Final Commit

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Run acceptance tests with longer timeout**

Run: `go test ./test/acceptance/... -v -timeout 5m`
Expected: All tests pass

**Step 3: Final commit with feature summary**

```bash
git add -A
git status  # Verify only expected files
git commit -m "feat(sandbox): credential mounting and profile bootstrap

- Add --creds and --no-creds flags for git, ssh, gh credentials
- Add --sync flag to re-apply profile settings
- Bootstrap profile marketplaces/plugins on first sandbox run
- Credentials mount read-only for security

Fixes #61"
```

---

## Summary

This plan implements:

1. **Credential mounting** via profile config and CLI flags
2. **Profile bootstrap** on first sandbox run
3. **Sync command** to reset sandbox to profile state

Files created/modified:
- `internal/profile/profile.go` - Add Credentials to SandboxConfig
- `internal/sandbox/credentials.go` - Credential resolution
- `internal/sandbox/credentials_test.go` - Credential tests
- `internal/sandbox/bootstrap.go` - Profile bootstrap
- `internal/sandbox/bootstrap_test.go` - Bootstrap tests
- `internal/sandbox/docker.go` - Integration
- `internal/commands/sandbox.go` - CLI flags
- `test/acceptance/sandbox_credentials_test.go` - Acceptance tests
