# Plan: Profile Includes (Composable Stacks)

## Context

Profiles in `~/.claudeup/profiles/` are organized by category (languages/, platforms/, workflow/, tools/). Category profiles are small and focused -- each defines plugins, marketplaces, and settings for a single concern (e.g., `languages/go.json` has gopls and Go detect rules). Stack profiles in `stacks/` compose them via an `includes` field:

```json
{
  "name": "fullstack-go",
  "description": "Go fullstack: essentials + Go, backend, frontend, testing",
  "includes": ["essentials", "go", "backend", "frontend", "testing"]
}
```

Stacks can include other stacks. `essentials` itself is a stack:

```json
{
  "name": "essentials",
  "includes": ["memory", "superpowers", "git", "code-quality"]
}
```

The CLI doesn't support `includes` yet. This plan adds resolution and merge logic so `cu profile apply fullstack-go` resolves the include tree, merges all fields, and applies the result.

## Design

### Core principles

- `Load()` stays unchanged -- returns raw profile with `Includes` intact
- New `ResolveIncludes(profile, loader)` flattens the include tree into a single profile
- Resolution is explicit -- callers opt in before Apply
- **Stacks are pure** -- a profile with `includes` may only have `name` and `description` alongside it. No own plugins, marketplaces, MCP servers, or other config fields. Validation enforces this.
- Nested includes work (A->B->C) with cycle detection
- Diamond patterns allowed (A->[B,C], both include D) -- D's fields appear once via dedup
- Includes support both short names (`"go"`) and path-qualified names (`"languages/go"`)

### Scope handling

- Resolved stacks always apply via `ApplyAllScopes()` since included profiles define their own scopes via `PerScope`
- Passing `--scope` with a stack profile is an error: "Stack profiles define their own scopes; --scope is not supported with stacks"

### Merge strategy

Included profiles merge left-to-right. Later includes take precedence on conflicts.

| Field                      | Strategy                                                       |
| -------------------------- | -------------------------------------------------------------- |
| `Marketplaces`             | Union, dedup by `marketplaceKey()` (reuse from `apply.go:865`) |
| `PerScope.*.Plugins`       | Union per scope, dedup                                         |
| `PerScope.*.MCPServers`    | Union per scope, last-wins by name                             |
| `Plugins` (legacy flat)    | Union, dedup                                                   |
| `MCPServers` (legacy flat) | Union, last-wins by name                                       |
| `LocalItems.*`             | Union per category, dedup                                      |
| `SettingsHooks`            | Union per event type, dedup by command                         |
| `Detect.Files`             | Union, dedup                                                   |
| `Detect.Contains`          | Merge map, later wins                                          |
| `SkipPluginDiff`           | OR (any true -> true)                                          |
| `PostApply`                | Last-wins                                                      |
| `Name`, `Description`      | Always from root stack profile                                 |
| `Includes`                 | Cleared in resolved output                                     |

**PostApply note:** When multiple included profiles define `PostApply` hooks, only the rightmost (last) include's hook is used. If your stack needs hooks from multiple profiles, combine them into a single hook script in the stack or in a dedicated leaf profile listed last.

**Depth limit:** Include chains are limited to 50 levels deep (`MaxIncludeDepth`). This prevents resource exhaustion from pathological nesting.

## Files to Modify

### 1. `internal/profile/profile.go` -- Add Includes field and helpers

Add `Includes` field to Profile struct (after `Description`, line 19):

```go
Includes []string `json:"includes,omitempty"`
```

Add methods:

```go
func (p *Profile) IsStack() bool {
    return p != nil && len(p.Includes) > 0
}

// HasConfigFields returns true if the profile has any configuration
// fields beyond name, description, and includes.
func (p *Profile) HasConfigFields() bool
```

`HasConfigFields` checks: Marketplaces, Plugins, MCPServers, PerScope, LocalItems, SettingsHooks, Detect, PostApply, SkipPluginDiff.

Update `Clone()` (line 524) -- deep copy Includes slice.

Update `Equal()` (line 588) -- compare Includes with `strSlicesEqual()`.

Update `PreserveFrom()` (line 267) -- preserve LocalItems only (Includes are not preserved since stacks are resolved at apply time, not stored in saved profiles):

```go
func (p *Profile) PreserveFrom(existing *Profile) {
    p.LocalItems = existing.LocalItems
}
```

Update `GenerateDescription()` (line 768) -- for stacks, return "stack: N includes" instead of counting plugins/marketplaces.

### 2. `internal/profile/resolve.go` -- New file: resolution engine

```go
// ProfileLoader loads a profile by name or path-qualified name.
type ProfileLoader interface {
    LoadProfile(name string) (*Profile, error)
}

// DirLoader loads profiles from disk via Load() with embedded fallback.
type DirLoader struct {
    ProfilesDir string
}

func (l *DirLoader) LoadProfile(name string) (*Profile, error)

// ResolveIncludes recursively resolves includes and returns a merged profile.
// Returns an error if:
//   - the profile has includes alongside config fields (stacks must be pure)
//   - a cycle is detected
//   - an included profile cannot be loaded
func ResolveIncludes(p *Profile, loader ProfileLoader) (*Profile, error)
```

Internal functions:

- `validatePureStack(p *Profile) error` -- errors if `IsStack()` and `HasConfigFields()`
- `resolveRecursive(name string, loader, visited map[string]bool, resolved map[string]*Profile)` -- cycle detection via visited set; caches resolved profiles for diamond dedup
- `mergeProfiles(profiles []*Profile) *Profile` -- merges a flat list of resolved profiles left-to-right
- `mergeProfile(dst, src *Profile)` -- orchestrates all field merges
- `mergeMarketplaces(dst, src)` -- union by `marketplaceKey()`
- `mergePerScope(dst, src)` -- delegates to `mergeScopeSettings` per scope level
- `mergeScopeSettings(dst, src *ScopeSettings)` -- union plugins, last-wins MCP
- `mergePlugins(dst, src)` -- union legacy flat plugins
- `mergeMCPServers(dst, src)` -- union legacy flat MCP, last-wins by name
- `mergeLocalItems(dst, src)` -- union per category via `mergeStringSlice`
- `mergeSettingsHooks(dst, src)` -- union per event type, dedup by command
- `mergeDetect(dst, src)` -- union files, merge contains map
- `mergeStringSlice(dst, src []string) []string` -- dedup preserving order

### 3. `internal/profile/resolve_test.go` -- New file: TDD tests

Test cases using a `mockLoader`:

1. No includes -- returns profile unchanged
2. Nil profile -- returns error
3. Single include -- fields merge correctly
4. Multiple includes -- left-to-right merge order verified
5. Nested includes (A->B->C) -- three levels deep
6. Cycle detection (A->B->A) -- error
7. Self-cycle (A->A) -- error
8. Transitive cycle (A->B->C->A) -- error
9. Diamond pattern (A->[B,C], both include D) -- no error, D's fields appear once
10. Missing include -- clear error with profile name
11. Stack with config fields alongside includes -- validation error
12. Marketplace dedup by key
13. PerScope plugins dedup per scope
14. MCP server last-wins by name within scope
15. LocalItems union per category
16. SettingsHooks dedup by command
17. Detect.Files union
18. Detect.Contains merge with later-wins
19. PostApply last-wins
20. Resolved profile has nil Includes
21. Preserves root stack's name and description
22. Path-qualified include name (e.g., "languages/go")
23. Mixed short and path-qualified names in same includes list

### 4. `internal/commands/profile_cmd.go` -- Wire into apply and display

**`applyProfileWithScope()`** (line 944): After loading profile (line 965), before security check (line 973):

```go
if p.IsStack() {
    if scope != "" {
        return fmt.Errorf("stack profiles define their own scopes; --scope is not supported with stacks")
    }
    loader := &profile.DirLoader{ProfilesDir: profilesDir}
    resolved, err := profile.ResolveIncludes(p, loader)
    if err != nil {
        return fmt.Errorf("failed to resolve includes: %w", err)
    }
    p = resolved
}
```

**`runProfileList()`** (line 659): In display loop, append `[stack]` indicator to description for profiles where `IsStack()` is true.

**`runProfileShow()`**: Display resolved stack information:

```
Name:     fullstack-go
Type:     stack
Includes: essentials -> [memory, superpowers, git, code-quality]
          go, backend, frontend, testing

Resolved: 5 marketplaces, 18 plugins (11 user, 7 project)
```

Show the include tree (expanding nested stacks one level) and a summary of the resolved contents (marketplace count, plugin count by scope, MCP server count, local item count -- only non-zero categories).

### 5. `internal/profile/profile_test.go` -- Update existing tests

- `TestClone` -- verify Includes deep copy
- `TestEqual` -- verify Includes comparison
- `TestIsStack` -- true when Includes non-empty, false otherwise
- `TestHasConfigFields` -- true/false for various field combinations
- `TestPreserveFrom` -- verify Includes are not preserved
- `TestGenerateDescription` -- verify stack description format
- JSON round-trip test for profile with Includes

## Implementation Order (TDD)

1. Write `resolve_test.go` with all test cases (red)
2. Add `Includes` field + `IsStack()` + `HasConfigFields()` to Profile struct
3. Update `Clone()`, `Equal()`, `PreserveFrom()`
4. Create `resolve.go` with `ProfileLoader`, `DirLoader`, `ResolveIncludes`, validation, merge functions
5. Run tests until green
6. Update `GenerateDescription()`
7. Wire into `applyProfileWithScope()` in profile_cmd.go (including --scope error)
8. Update `runProfileList()` and `runProfileShow()`
9. Update `profile_test.go` with new test cases
10. Run full test suite: `go test ./...`

## Verification

1. `go test ./internal/profile/...` -- unit tests pass
2. `go test ./...` -- full suite passes
3. Manual: `cu profile apply fullstack-go` resolves the stack and applies all included plugins at correct scopes
4. Manual: `cu profile list` shows `[stack]` for stack profiles
5. Manual: `cu profile show fullstack-go` displays include tree and resolved summary
6. Manual: create a cycle (A includes B, B includes A) and verify clear error
7. Manual: apply a stack and verify plugins install at correct scopes (user vs project)
8. Manual: `cu profile apply fullstack-go --scope project` and verify error message

## What Doesn't Change

- `apply.go` -- operates on resolved profiles, no includes awareness needed
- `snapshot.go` -- captures current state, not profile definitions
- `install.go`, `worker.go`, `scope.go` -- unchanged
- Embedded profiles -- no includes support needed (could add later)
- `Load()` / `LoadFromPath()` -- returns raw profile, resolution is opt-in
