# Non-Interactive Profile Create Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add non-interactive modes to `claudeup profile create` supporting flags and file/stdin input.

**Architecture:** Mode detection in `runProfileCreate` dispatches to three paths: flags mode (builds profile from CLI args), file mode (parses JSON from file/stdin), or existing wizard (no flags). Core creation logic lives in new `internal/profile/create.go`.

**Tech Stack:** Go, Cobra CLI, Ginkgo/Gomega tests

---

## Task 1: Add Flag Variables

**Files:**
- Modify: `internal/commands/profile_cmd.go` (around line 148)

**Step 1: Add flag variables at package level**

Find the existing flag variables (around line 50-80) and add:

```go
var (
	profileCreateDescription  string
	profileCreateMarketplaces []string
	profileCreatePlugins      []string
	profileCreateFromFile     string
	profileCreateFromStdin    bool
)
```

**Step 2: Register flags on profileCreateCmd**

Find where `profileCreateCmd` flags are registered (in `init()` around line 503) and add:

```go
profileCreateCmd.Flags().StringVar(&profileCreateDescription, "description", "", "Profile description")
profileCreateCmd.Flags().StringSliceVar(&profileCreateMarketplaces, "marketplace", nil, "Marketplace in owner/repo format (can be repeated)")
profileCreateCmd.Flags().StringSliceVar(&profileCreatePlugins, "plugin", nil, "Plugin in name@marketplace-ref format (can be repeated)")
profileCreateCmd.Flags().StringVar(&profileCreateFromFile, "from-file", "", "Create profile from JSON file")
profileCreateCmd.Flags().BoolVar(&profileCreateFromStdin, "from-stdin", false, "Create profile from JSON on stdin")
```

**Step 3: Build and verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "feat(profile): add flag variables for non-interactive create"
```

---

## Task 2: Create CreateSpec and Validation Types

**Files:**
- Create: `internal/profile/create.go`

**Step 1: Write the failing test**

Create `internal/profile/create_test.go`:

```go
// ABOUTME: Unit tests for non-interactive profile creation
// ABOUTME: Tests validation and profile construction from specs
package profile

import (
	"testing"
)

func TestParseMarketplaceArg(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    Marketplace
		wantErr bool
	}{
		{
			name: "shorthand format",
			arg:  "anthropics/claude-code",
			want: Marketplace{Source: "github", Repo: "anthropics/claude-code"},
		},
		{
			name:    "empty string",
			arg:     "",
			wantErr: true,
		},
		{
			name:    "no slash",
			arg:     "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMarketplaceArg(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMarketplaceArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (got.Source != tt.want.Source || got.Repo != tt.want.Repo) {
				t.Errorf("ParseMarketplaceArg() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/profile -run TestParseMarketplaceArg -v`
Expected: FAIL with "undefined: ParseMarketplaceArg"

**Step 3: Write minimal implementation**

Create `internal/profile/create.go`:

```go
// ABOUTME: Non-interactive profile creation from flags or file input
// ABOUTME: Provides CreateSpec validation and profile construction
package profile

import (
	"fmt"
	"strings"
)

// ParseMarketplaceArg parses a marketplace argument in "owner/repo" format
func ParseMarketplaceArg(arg string) (Marketplace, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return Marketplace{}, fmt.Errorf("marketplace cannot be empty")
	}

	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Marketplace{}, fmt.Errorf("invalid marketplace format %q: expected owner/repo", arg)
	}

	return Marketplace{
		Source: "github",
		Repo:   arg,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/profile -run TestParseMarketplaceArg -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/profile/create.go internal/profile/create_test.go
git commit -m "feat(profile): add ParseMarketplaceArg for shorthand marketplace format"
```

---

## Task 3: Add Plugin Format Validation

**Files:**
- Modify: `internal/profile/create.go`
- Modify: `internal/profile/create_test.go`

**Step 1: Write the failing test**

Add to `internal/profile/create_test.go`:

```go
func TestValidatePluginFormat(t *testing.T) {
	tests := []struct {
		name    string
		plugin  string
		wantErr bool
	}{
		{
			name:   "valid format",
			plugin: "plugin-dev@claude-code-plugins",
		},
		{
			name:   "valid with colons",
			plugin: "backend:api-design@claude-workflows",
		},
		{
			name:    "empty string",
			plugin:  "",
			wantErr: true,
		},
		{
			name:    "no at sign",
			plugin:  "invalid-plugin",
			wantErr: true,
		},
		{
			name:    "empty marketplace ref",
			plugin:  "plugin@",
			wantErr: true,
		},
		{
			name:    "empty plugin name",
			plugin:  "@marketplace",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePluginFormat(tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePluginFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/profile -run TestValidatePluginFormat -v`
Expected: FAIL with "undefined: ValidatePluginFormat"

**Step 3: Write minimal implementation**

Add to `internal/profile/create.go`:

```go
// ValidatePluginFormat validates a plugin string is in "name@marketplace-ref" format
func ValidatePluginFormat(plugin string) error {
	plugin = strings.TrimSpace(plugin)
	if plugin == "" {
		return fmt.Errorf("plugin cannot be empty")
	}

	atIdx := strings.LastIndex(plugin, "@")
	if atIdx == -1 {
		return fmt.Errorf("invalid plugin format %q: expected name@marketplace-ref", plugin)
	}

	name := plugin[:atIdx]
	ref := plugin[atIdx+1:]

	if name == "" {
		return fmt.Errorf("invalid plugin format %q: plugin name cannot be empty", plugin)
	}
	if ref == "" {
		return fmt.Errorf("invalid plugin format %q: marketplace ref cannot be empty", plugin)
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/profile -run TestValidatePluginFormat -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/profile/create.go internal/profile/create_test.go
git commit -m "feat(profile): add ValidatePluginFormat for plugin string validation"
```

---

## Task 4: Add CreateSpec Validation

**Files:**
- Modify: `internal/profile/create.go`
- Modify: `internal/profile/create_test.go`

**Step 1: Write the failing test**

Add to `internal/profile/create_test.go`:

```go
func TestValidateCreateSpec(t *testing.T) {
	tests := []struct {
		name        string
		description string
		markets     []string
		plugins     []string
		wantErr     string
	}{
		{
			name:        "valid minimal",
			description: "Test profile",
			markets:     []string{"owner/repo"},
			plugins:     []string{},
			wantErr:     "",
		},
		{
			name:        "valid with plugins",
			description: "Test profile",
			markets:     []string{"owner/repo"},
			plugins:     []string{"plugin@ref"},
			wantErr:     "",
		},
		{
			name:        "missing description",
			description: "",
			markets:     []string{"owner/repo"},
			wantErr:     "description is required",
		},
		{
			name:        "missing marketplaces",
			description: "Test",
			markets:     []string{},
			wantErr:     "at least one marketplace is required",
		},
		{
			name:        "invalid marketplace",
			description: "Test",
			markets:     []string{"invalid"},
			wantErr:     "invalid marketplace format",
		},
		{
			name:        "invalid plugin",
			description: "Test",
			markets:     []string{"owner/repo"},
			plugins:     []string{"no-at-sign"},
			wantErr:     "invalid plugin format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateSpec(tt.description, tt.markets, tt.plugins)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateCreateSpec() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateCreateSpec() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateCreateSpec() error = %v, want containing %q", err, tt.wantErr)
				}
			}
		})
	}
}
```

Add import at top: `"strings"` (if not already present in test file)

**Step 2: Run test to verify it fails**

Run: `go test ./internal/profile -run TestValidateCreateSpec -v`
Expected: FAIL with "undefined: ValidateCreateSpec"

**Step 3: Write minimal implementation**

Add to `internal/profile/create.go`:

```go
// ValidateCreateSpec validates input for non-interactive profile creation
func ValidateCreateSpec(description string, marketplaces []string, plugins []string) error {
	if strings.TrimSpace(description) == "" {
		return fmt.Errorf("description is required")
	}

	if len(marketplaces) == 0 {
		return fmt.Errorf("at least one marketplace is required")
	}

	for _, m := range marketplaces {
		if _, err := ParseMarketplaceArg(m); err != nil {
			return err
		}
	}

	for _, p := range plugins {
		if err := ValidatePluginFormat(p); err != nil {
			return err
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/profile -run TestValidateCreateSpec -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/profile/create.go internal/profile/create_test.go
git commit -m "feat(profile): add ValidateCreateSpec for input validation"
```

---

## Task 5: Add CreateFromFlags Function

**Files:**
- Modify: `internal/profile/create.go`
- Modify: `internal/profile/create_test.go`

**Step 1: Write the failing test**

Add to `internal/profile/create_test.go`:

```go
func TestCreateFromFlags(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		description string
		markets     []string
		plugins     []string
		wantErr     bool
	}{
		{
			name:        "creates valid profile",
			profileName: "test-profile",
			description: "Test description",
			markets:     []string{"anthropics/claude-code", "obra/superpowers"},
			plugins:     []string{"plugin-dev@claude-code-plugins"},
			wantErr:     false,
		},
		{
			name:        "fails on validation error",
			profileName: "test",
			description: "",
			markets:     []string{"owner/repo"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := CreateFromFlags(tt.profileName, tt.description, tt.markets, tt.plugins)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFromFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if p.Name != tt.profileName {
					t.Errorf("CreateFromFlags() name = %v, want %v", p.Name, tt.profileName)
				}
				if p.Description != tt.description {
					t.Errorf("CreateFromFlags() description = %v, want %v", p.Description, tt.description)
				}
				if len(p.Marketplaces) != len(tt.markets) {
					t.Errorf("CreateFromFlags() marketplaces = %v, want %v", len(p.Marketplaces), len(tt.markets))
				}
				if len(p.Plugins) != len(tt.plugins) {
					t.Errorf("CreateFromFlags() plugins = %v, want %v", len(p.Plugins), len(tt.plugins))
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/profile -run TestCreateFromFlags -v`
Expected: FAIL with "undefined: CreateFromFlags"

**Step 3: Write minimal implementation**

Add to `internal/profile/create.go`:

```go
// CreateFromFlags creates a profile from CLI flag values
func CreateFromFlags(name, description string, marketplaceArgs, plugins []string) (*Profile, error) {
	if err := ValidateCreateSpec(description, marketplaceArgs, plugins); err != nil {
		return nil, err
	}

	marketplaces := make([]Marketplace, 0, len(marketplaceArgs))
	for _, arg := range marketplaceArgs {
		m, _ := ParseMarketplaceArg(arg) // Already validated
		marketplaces = append(marketplaces, m)
	}

	return &Profile{
		Name:         name,
		Description:  description,
		Marketplaces: marketplaces,
		Plugins:      plugins,
		MCPServers:   []MCPServer{},
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/profile -run TestCreateFromFlags -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/profile/create.go internal/profile/create_test.go
git commit -m "feat(profile): add CreateFromFlags for building profiles from CLI args"
```

---

## Task 6: Add CreateFromReader Function

**Files:**
- Modify: `internal/profile/create.go`
- Modify: `internal/profile/create_test.go`

**Step 1: Write the failing test**

Add to `internal/profile/create_test.go`:

```go
func TestCreateFromReader(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		json        string
		descOverride string
		wantErr     string
	}{
		{
			name:        "valid JSON with object marketplaces",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": [{"source": "github", "repo": "owner/repo"}],
				"plugins": ["plugin@ref"]
			}`,
			wantErr: "",
		},
		{
			name:        "valid JSON with shorthand marketplaces",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": ["owner/repo"],
				"plugins": []
			}`,
			wantErr: "",
		},
		{
			name:        "description override",
			profileName: "my-profile",
			json: `{
				"description": "Original",
				"marketplaces": ["owner/repo"]
			}`,
			descOverride: "Overridden",
			wantErr: "",
		},
		{
			name:        "invalid JSON",
			profileName: "my-profile",
			json:        `{invalid`,
			wantErr:     "invalid JSON",
		},
		{
			name:        "missing description",
			profileName: "my-profile",
			json: `{
				"marketplaces": ["owner/repo"]
			}`,
			wantErr: "description is required",
		},
		{
			name:        "missing marketplaces",
			profileName: "my-profile",
			json: `{
				"description": "Test"
			}`,
			wantErr: "at least one marketplace is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.json)
			p, err := CreateFromReader(tt.profileName, r, tt.descOverride)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("CreateFromReader() unexpected error = %v", err)
					return
				}
				if p.Name != tt.profileName {
					t.Errorf("CreateFromReader() name = %v, want %v", p.Name, tt.profileName)
				}
				if tt.descOverride != "" && p.Description != tt.descOverride {
					t.Errorf("CreateFromReader() description = %v, want %v", p.Description, tt.descOverride)
				}
			} else {
				if err == nil {
					t.Errorf("CreateFromReader() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CreateFromReader() error = %v, want containing %q", err, tt.wantErr)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/profile -run TestCreateFromReader -v`
Expected: FAIL with "undefined: CreateFromReader"

**Step 3: Write minimal implementation**

Add imports at top of `internal/profile/create.go`:

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)
```

Add to `internal/profile/create.go`:

```go
// CreateSpec is the input format for file/stdin profile creation
type CreateSpec struct {
	Description  string          `json:"description"`
	Marketplaces json.RawMessage `json:"marketplaces"`
	Plugins      []string        `json:"plugins"`
	MCPServers   []MCPServer     `json:"mcpServers,omitempty"`
	Detect       DetectRules     `json:"detect,omitempty"`
	Sandbox      SandboxConfig   `json:"sandbox,omitempty"`
}

// CreateFromReader creates a profile from JSON input
func CreateFromReader(name string, r io.Reader, descOverride string) (*Profile, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	var spec CreateSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Parse marketplaces (can be strings or objects)
	marketplaces, err := parseMarketplacesJSON(spec.Marketplaces)
	if err != nil {
		return nil, err
	}

	// Apply description override
	description := spec.Description
	if descOverride != "" {
		description = descOverride
	}

	// Convert to string args for validation
	marketArgs := make([]string, len(marketplaces))
	for i, m := range marketplaces {
		marketArgs[i] = m.Repo
	}

	if err := ValidateCreateSpec(description, marketArgs, spec.Plugins); err != nil {
		return nil, err
	}

	return &Profile{
		Name:         name,
		Description:  description,
		Marketplaces: marketplaces,
		Plugins:      spec.Plugins,
		MCPServers:   spec.MCPServers,
		Detect:       spec.Detect,
		Sandbox:      spec.Sandbox,
	}, nil
}

// parseMarketplacesJSON handles both string and object marketplace formats
func parseMarketplacesJSON(raw json.RawMessage) ([]Marketplace, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Try array of strings first
	var stringMarkets []string
	if err := json.Unmarshal(raw, &stringMarkets); err == nil {
		markets := make([]Marketplace, 0, len(stringMarkets))
		for _, s := range stringMarkets {
			m, err := ParseMarketplaceArg(s)
			if err != nil {
				return nil, err
			}
			markets = append(markets, m)
		}
		return markets, nil
	}

	// Try array of objects
	var objMarkets []Marketplace
	if err := json.Unmarshal(raw, &objMarkets); err != nil {
		return nil, fmt.Errorf("invalid marketplace format: expected array of strings or objects")
	}

	return objMarkets, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/profile -run TestCreateFromReader -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/profile/create.go internal/profile/create_test.go
git commit -m "feat(profile): add CreateFromReader for JSON input parsing"
```

---

## Task 7: Write Acceptance Test for Flags Mode

**Files:**
- Create: `test/acceptance/profile_create_noninteractive_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Acceptance tests for non-interactive profile create
// ABOUTME: Tests flags mode, file mode, and validation errors
package acceptance

import (
	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile create non-interactive", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("flags mode", func() {
		It("creates profile with all flags", func() {
			result := env.Run("profile", "create", "test-profile",
				"--description", "Test description",
				"--marketplace", "anthropics/claude-code",
				"--plugin", "plugin-dev@claude-code-plugins",
			)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)
			Expect(result.Stdout).To(ContainSubstring("created successfully"))
			Expect(env.ProfileExists("test-profile")).To(BeTrue())

			// Verify profile contents
			p := env.LoadProfile("test-profile")
			Expect(p.Description).To(Equal("Test description"))
			Expect(p.Marketplaces).To(HaveLen(1))
			Expect(p.Marketplaces[0].Repo).To(Equal("anthropics/claude-code"))
			Expect(p.Plugins).To(HaveLen(1))
			Expect(p.Plugins[0]).To(Equal("plugin-dev@claude-code-plugins"))
		})

		It("creates profile with multiple marketplaces", func() {
			result := env.Run("profile", "create", "multi-market",
				"--description", "Multi marketplace",
				"--marketplace", "anthropics/claude-code",
				"--marketplace", "obra/superpowers-marketplace",
			)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)

			p := env.LoadProfile("multi-market")
			Expect(p.Marketplaces).To(HaveLen(2))
		})

		It("fails without description in flags mode", func() {
			result := env.Run("profile", "create", "no-desc",
				"--marketplace", "owner/repo",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("description is required"))
		})

		It("fails without marketplaces in flags mode", func() {
			result := env.Run("profile", "create", "no-market",
				"--description", "Test",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("at least one marketplace is required"))
		})

		It("fails with invalid marketplace format", func() {
			result := env.Run("profile", "create", "bad-market",
				"--description", "Test",
				"--marketplace", "invalid",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid marketplace format"))
		})

		It("fails with invalid plugin format", func() {
			result := env.Run("profile", "create", "bad-plugin",
				"--description", "Test",
				"--marketplace", "owner/repo",
				"--plugin", "no-at-sign",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid plugin format"))
		})
	})
})
```

**Step 2: Run test to verify it fails**

Run: `go test ./test/acceptance -run "profile create non-interactive/flags mode" -v`
Expected: FAIL (command doesn't implement flags mode yet)

**Step 3: Commit the test**

```bash
git add test/acceptance/profile_create_noninteractive_test.go
git commit -m "test(profile): add acceptance tests for non-interactive flags mode"
```

---

## Task 8: Implement Flags Mode in runProfileCreate

**Files:**
- Modify: `internal/commands/profile_cmd.go`

**Step 1: Update runProfileCreate to detect and handle flags mode**

Find `func runProfileCreate` (line 1803) and replace the beginning with:

```go
func runProfileCreate(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()

	// Detect mode: file, flags, or wizard
	hasFileInput := profileCreateFromFile != "" || profileCreateFromStdin
	hasFlagsInput := len(profileCreateMarketplaces) > 0 || len(profileCreatePlugins) > 0 || profileCreateDescription != ""

	// Mutual exclusivity check
	if hasFileInput && hasFlagsInput && (len(profileCreateMarketplaces) > 0 || len(profileCreatePlugins) > 0) {
		return fmt.Errorf("cannot combine --from-file/--from-stdin with --marketplace/--plugin flags")
	}

	// Name is required for non-interactive modes
	if (hasFileInput || hasFlagsInput) && len(args) == 0 {
		return fmt.Errorf("profile name is required for non-interactive mode")
	}

	// Flags mode
	if hasFlagsInput && !hasFileInput {
		name := args[0]
		if err := profile.ValidateName(name); err != nil {
			return err
		}

		existingPath := filepath.Join(profilesDir, name+".json")
		if _, err := os.Stat(existingPath); err == nil {
			return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
		}

		newProfile, err := profile.CreateFromFlags(name, profileCreateDescription, profileCreateMarketplaces, profileCreatePlugins)
		if err != nil {
			return err
		}

		if err := profile.Save(profilesDir, newProfile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Printf("Profile %q created successfully.\n\n", name)
		fmt.Printf("  Marketplaces: %d\n", len(newProfile.Marketplaces))
		fmt.Printf("  Plugins: %d\n", len(newProfile.Plugins))
		fmt.Printf("\nRun 'claudeup profile apply %s' to use it.\n", name)
		return nil
	}

	// File/stdin mode
	if hasFileInput {
		name := args[0]
		if err := profile.ValidateName(name); err != nil {
			return err
		}

		existingPath := filepath.Join(profilesDir, name+".json")
		if _, err := os.Stat(existingPath); err == nil {
			return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
		}

		var reader io.Reader
		if profileCreateFromStdin {
			reader = os.Stdin
		} else {
			f, err := os.Open(profileCreateFromFile)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()
			reader = f
		}

		newProfile, err := profile.CreateFromReader(name, reader, profileCreateDescription)
		if err != nil {
			return err
		}

		if err := profile.Save(profilesDir, newProfile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Printf("Profile %q created successfully.\n\n", name)
		fmt.Printf("  Marketplaces: %d\n", len(newProfile.Marketplaces))
		fmt.Printf("  Plugins: %d\n", len(newProfile.Plugins))
		fmt.Printf("\nRun 'claudeup profile apply %s' to use it.\n", name)
		return nil
	}

	// Wizard mode (existing code continues from here...)
```

Also add `"io"` to the imports at the top of the file.

**Step 2: Run test to verify it passes**

Run: `go test ./test/acceptance -run "profile create non-interactive/flags mode" -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/commands/profile_cmd.go
git commit -m "feat(profile): implement flags mode for non-interactive create"
```

---

## Task 9: Write Acceptance Test for File Mode

**Files:**
- Modify: `test/acceptance/profile_create_noninteractive_test.go`

**Step 1: Add file mode tests**

Add to the test file after the flags mode context:

```go
	Context("file mode", func() {
		It("creates profile from file", func() {
			specPath := filepath.Join(env.TempDir, "spec.json")
			spec := `{
				"description": "From file",
				"marketplaces": ["anthropics/claude-code"],
				"plugins": ["plugin@ref"]
			}`
			Expect(os.WriteFile(specPath, []byte(spec), 0644)).To(Succeed())

			result := env.Run("profile", "create", "from-file-profile", "--from-file", specPath)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)
			Expect(env.ProfileExists("from-file-profile")).To(BeTrue())

			p := env.LoadProfile("from-file-profile")
			Expect(p.Description).To(Equal("From file"))
		})

		It("creates profile from stdin", func() {
			spec := `{
				"description": "From stdin",
				"marketplaces": ["anthropics/claude-code"],
				"plugins": []
			}`

			result := env.RunWithInput(spec, "profile", "create", "from-stdin-profile", "--from-stdin")
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)
			Expect(env.ProfileExists("from-stdin-profile")).To(BeTrue())

			p := env.LoadProfile("from-stdin-profile")
			Expect(p.Description).To(Equal("From stdin"))
		})

		It("allows description override with --from-file", func() {
			specPath := filepath.Join(env.TempDir, "spec.json")
			spec := `{
				"description": "Original",
				"marketplaces": ["owner/repo"]
			}`
			Expect(os.WriteFile(specPath, []byte(spec), 0644)).To(Succeed())

			result := env.Run("profile", "create", "override-profile",
				"--from-file", specPath,
				"--description", "Overridden",
			)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)

			p := env.LoadProfile("override-profile")
			Expect(p.Description).To(Equal("Overridden"))
		})

		It("fails with invalid JSON", func() {
			specPath := filepath.Join(env.TempDir, "bad.json")
			Expect(os.WriteFile(specPath, []byte(`{invalid`), 0644)).To(Succeed())

			result := env.Run("profile", "create", "bad-json", "--from-file", specPath)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid JSON"))
		})

		It("rejects combining --from-file with --marketplace", func() {
			specPath := filepath.Join(env.TempDir, "spec.json")
			spec := `{"description": "Test", "marketplaces": ["owner/repo"]}`
			Expect(os.WriteFile(specPath, []byte(spec), 0644)).To(Succeed())

			result := env.Run("profile", "create", "conflict",
				"--from-file", specPath,
				"--marketplace", "other/repo",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("cannot combine"))
		})
	})
```

Add imports at top:

```go
import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)
```

**Step 2: Run tests to verify they pass**

Run: `go test ./test/acceptance -run "profile create non-interactive/file mode" -v`
Expected: PASS

**Step 3: Commit**

```bash
git add test/acceptance/profile_create_noninteractive_test.go
git commit -m "test(profile): add acceptance tests for non-interactive file mode"
```

---

## Task 10: Run Full Test Suite and Final Verification

**Files:** None (verification only)

**Step 1: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 2: Manual verification**

Test the commands manually:

```bash
# Build
go build -o bin/claudeup ./cmd/claudeup

# Flags mode
./bin/claudeup profile create test-flags \
  --description "Test flags" \
  --marketplace "anthropics/claude-code" \
  --plugin "code-reviewer@claude-code-plugins"

# File mode
echo '{"description":"From file","marketplaces":["obra/superpowers"]}' > /tmp/spec.json
./bin/claudeup profile create test-file --from-file /tmp/spec.json

# Stdin mode
echo '{"description":"From stdin","marketplaces":["anthropics/claude-code"]}' | \
  ./bin/claudeup profile create test-stdin --from-stdin

# Verify
./bin/claudeup profile show test-flags
./bin/claudeup profile show test-file
./bin/claudeup profile show test-stdin

# Cleanup
./bin/claudeup profile delete test-flags -y
./bin/claudeup profile delete test-file -y
./bin/claudeup profile delete test-stdin -y
```

**Step 3: Commit any final fixes**

If any issues found, fix and commit individually.

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat(profile): complete non-interactive profile create (closes #100)"
```

---

## Task 11: Update Documentation Examples

**Files:**
- Search codebase for examples writing JSON directly to profiles

**Step 1: Search for direct JSON writes**

```bash
grep -r "~/.claudeup/profiles" docs/ README.md CLAUDE.md 2>/dev/null || true
grep -r "profiles.*json" docs/ 2>/dev/null || true
```

**Step 2: Update any found examples**

Replace direct file writes with CLI commands:

Before:
```bash
cat > ~/.claudeup/profiles/my-profile.json <<'EOF'
{"name": "my-profile", ...}
EOF
```

After:
```bash
claudeup profile create my-profile \
  --description "My profile" \
  --marketplace "owner/repo" \
  --plugin "plugin@ref"
```

**Step 3: Commit documentation updates**

```bash
git add docs/ README.md CLAUDE.md
git commit -m "docs: update profile examples to use CLI instead of direct JSON"
```

---

## Summary

| Task | Description | Estimated Time |
|------|-------------|----------------|
| 1 | Add flag variables | 5 min |
| 2 | ParseMarketplaceArg | 10 min |
| 3 | ValidatePluginFormat | 10 min |
| 4 | ValidateCreateSpec | 10 min |
| 5 | CreateFromFlags | 10 min |
| 6 | CreateFromReader | 15 min |
| 7 | Acceptance tests (flags) | 10 min |
| 8 | Implement flags mode | 15 min |
| 9 | Acceptance tests (file) | 10 min |
| 10 | Full test suite | 10 min |
| 11 | Documentation | 10 min |

**Total: ~2 hours**
