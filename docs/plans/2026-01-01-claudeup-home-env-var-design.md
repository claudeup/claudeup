# CLAUDEUP_HOME Environment Variable Support

**Issue**: [#75](https://github.com/claudeup/claudeup/issues/75)
**Date**: 2026-01-01
**Status**: Approved

## Problem

claudeup ignores the `CLAUDEUP_HOME` environment variable. The `configPath()` function and 15+ other locations hardcode `~/.claudeup`:

```go
homeDir, _ := os.UserHomeDir()
return filepath.Join(homeDir, ".claudeup", "config.json")
```

This forces scripts to override `HOME` entirely to achieve isolation, which affects other tools and is a heavy-handed workaround.

## Solution

Create a centralized `MustClaudeupHome()` helper that checks `CLAUDEUP_HOME` first, falling back to `~/.claudeup`. Update all call sites to use this helper.

### Helper Function

**New file: `internal/config/paths.go`**

```go
// ABOUTME: Centralized path resolution for claudeup directories
// ABOUTME: Respects CLAUDEUP_HOME environment variable for isolation

package config

import (
    "os"
    "path/filepath"
)

// MustClaudeupHome returns the claudeup home directory.
// Checks CLAUDEUP_HOME env var first, falls back to ~/.claudeup.
// Panics if home directory cannot be determined.
func MustClaudeupHome() string {
    if home := os.Getenv("CLAUDEUP_HOME"); home != "" {
        return home
    }
    homeDir, err := os.UserHomeDir()
    if err != nil {
        panic("cannot determine home directory: " + err.Error())
    }
    return filepath.Join(homeDir, ".claudeup")
}
```

### Call Sites to Update

**Config package** (same package, no import needed):
| File | Line(s) | Current Pattern |
|------|---------|-----------------|
| `global.go` | 50, 82-83 | `DefaultConfig()` ClaudeDir, `configPath()` |
| `projects.go` | 97-98 | `projectsPath()` |

**Backup package**:
| File | Line(s) | Current Pattern |
|------|---------|-----------------|
| `backup.go` | 34, 142, 163, 197, 239 | backup directory paths |

**Events package**:
| File | Line(s) | Current Pattern |
|------|---------|-----------------|
| `global.go` | 35-36 | events directory |

**Commands package**:
| File | Line(s) | Current Pattern |
|------|---------|-----------------|
| `events.go` | 53, 57 | events directory |
| `events_diff.go` | 55, 59 | events directory |
| `events_audit.go` | 58, 62 | events directory |
| `plugin.go` | 218 | profiles directory |
| `setup.go` | 323 | profiles directory |
| `status.go` | 88, 310 | profiles directory |
| `profile_cmd.go` | 702 | profiles directory |
| `sandbox.go` | 70 | claudeup directory |

**Sandbox package**:
| File | Line(s) | Current Pattern |
|------|---------|-----------------|
| `docker.go` | 183, 190 | Note: Uses `~` for docker mounts, may need different handling |

### Script Cleanup

**`examples/lib/common.sh`** - Remove HOME override:

```bash
# Before:
export HOME="$EXAMPLE_TEMP_DIR"
export CLAUDE_CONFIG_DIR="$EXAMPLE_TEMP_DIR/.claude"
export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/.claudeup"

# After:
export CLAUDE_CONFIG_DIR="$EXAMPLE_TEMP_DIR/.claude"
export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/.claudeup"
```

**`~/workspace/claudeup-test-repos/scripts/bob-syncs-profile.sh`** - Same change.

## Testing

**Unit test** (`internal/config/paths_test.go`):

```go
func TestMustClaudeupHome(t *testing.T) {
    t.Run("uses CLAUDEUP_HOME when set", func(t *testing.T) {
        t.Setenv("CLAUDEUP_HOME", "/custom/path")
        got := MustClaudeupHome()
        if got != "/custom/path" {
            t.Errorf("got %q, want /custom/path", got)
        }
    })

    t.Run("falls back to ~/.claudeup when not set", func(t *testing.T) {
        t.Setenv("CLAUDEUP_HOME", "")
        got := MustClaudeupHome()
        home, _ := os.UserHomeDir()
        want := filepath.Join(home, ".claudeup")
        if got != want {
            t.Errorf("got %q, want %q", got, want)
        }
    })
}
```

**Integration verification**:
1. Build claudeup with the fix
2. Run `bob-syncs-profile.sh` (now without `HOME` override)
3. If it passes, the fix works end-to-end

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Single helper function vs dependency injection | Simpler change, matches existing patterns |
| Location in `internal/config/paths.go` | Config package owns `~/.claudeup/config.json` |
| `Must` prefix with panic | Matches `MustHomeDir()` pattern, home resolution failing is exceptional |
| Fix all locations, not just `configPath()` | Consistent behavior, fully eliminates HOME workaround |
| Include script cleanup in same change | Logically connected, verification requires updated scripts |
