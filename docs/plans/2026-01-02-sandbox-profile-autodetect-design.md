# Sandbox Profile Auto-Detection Design

**Date:** 2026-01-02
**Status:** Approved
**Issue:** Enhancement to `claudeup sandbox` command

## Summary

Enable `claudeup sandbox` to automatically detect and use the profile referenced in a project's `.claudeup.json` file, eliminating the need for explicit `--profile` flags when working in configured projects.

## Profile Selection Precedence

When `claudeup sandbox` runs, profile selection follows this order:

1. `--ephemeral` flag → No profile, clean sandbox
2. `--profile <name>` → Use explicit profile
3. `.claudeup.json` in working directory → Auto-detect profile
4. None of the above → Ephemeral mode

## Scope Handling

### Container Directory Layout

```text
/root/.claude/                    # User scope (from sandbox state)
    settings.json                 # Plugins enabled by profile
    plugins/
        installed_plugins.json    # Plugin registry
        marketplaces.json         # Marketplace sources

/workspace/                       # Project root (mounted from host)
    .claude/
        settings.json             # Project scope (from host)
        settings.local.json       # Local scope (from host)
    .claudeup.json                # Profile reference
```

### What Goes Where

**User scope (`/root/.claude/settings.json`)** - From profile:
- Plugins (enabled in `enabledPlugins`)
- Marketplaces (written to `plugins/marketplaces.json`)

**Project scope (`/workspace/.claude/`)** - From host project:
- MCP servers (project-specific)
- Project-scoped plugin overrides
- Local-scoped settings

MCP servers remain in project scope because they often have project-specific paths or configurations.

## Plugin Installation

Writing config files (`settings.json`, `marketplaces.json`) declares which plugins should be enabled, but doesn't actually install them. Plugins are git repositories that must be cloned from marketplaces.

### Bootstrap Sequence

When `claudeup sandbox` runs with a profile (auto-detected or explicit):

1. **Write config files** - Bootstrap writes `settings.json` and `marketplaces.json` to sandbox state directory
2. **Run `claudeup sync`** - Clones plugin repositories from configured marketplaces
3. **Show progress bar** - Provide visual feedback during clone operations (can take time)
4. **Launch Claude or shell** - Once plugins are installed, start the requested entrypoint

### Progress Bar

Plugin installation involves cloning git repositories, which can be slow. Display a progress bar:

```text
Using profile 'diego-cap-analyzer' from .claudeup.json
Syncing plugins...
  [████████████░░░░░░░░] 3/5 plugins installed
```

### Skip Sync When Bootstrapped

If the sandbox has already been bootstrapped (`.bootstrapped` sentinel exists) and `--sync` is not specified, skip the sync step. This makes subsequent runs fast.

| Condition | Sync Behavior |
|-----------|---------------|
| First run (no `.bootstrapped`) | Run sync with progress |
| Subsequent runs | Skip sync |
| `--sync` flag | Force sync with progress |
| `--ephemeral` | No sync (no profile) |

## CLI Flag Interactions

| Flags Present | Behavior |
|--------------|----------|
| `--ephemeral` | Clean sandbox, no profile, ignore `.claudeup.json` |
| `--profile foo` | Use "foo", ignore `.claudeup.json` |
| (neither) | Auto-detect from `.claudeup.json`, else ephemeral |

Credential and secret merging remains unchanged. The auto-detected profile's `SandboxConfig` provides defaults, then CLI flags modify:

```text
Profile defines: credentials: ["git", "ssh"]
CLI adds:        --creds gh
CLI excludes:    --no-creds ssh
Result:          ["git", "gh"]
```

The `--sync` flag works the same regardless of how the profile was selected.

## User Feedback

When auto-detecting a profile, print:
```text
Using profile 'diego-cap-analyzer' from .claudeup.json
```

## Example Commands

```bash
# Auto-detect profile, launch Claude
claudeup sandbox

# Auto-detect profile, drop to shell
claudeup sandbox --shell

# Auto-detect profile, add GitHub CLI creds
claudeup sandbox --creds gh

# Override auto-detection with explicit profile
claudeup sandbox --profile different-profile

# Ignore .claudeup.json, run clean
claudeup sandbox --ephemeral
```

## Implementation

### Files to Modify

1. **`internal/commands/sandbox.go`**
   - Add auto-detection logic before profile loading
   - Check for `.claudeup.json` in working directory
   - Print feedback message when auto-detecting
   - Skip detection if `--ephemeral` or `--profile` is set
   - After bootstrap, call sync with progress bar on first run or `--sync`

2. **`internal/profile/project_config.go`**
   - Add helper: `DetectProfileFromProject(dir string) (string, error)`

3. **`internal/sandbox/bootstrap.go`** (or new `sync.go`)
   - Add `SyncPlugins(stateDir string, progress ProgressWriter) error`
   - Calls existing sync logic with progress reporting
   - Progress bar shows plugin clone status

4. **Progress bar support**
   - Use a library like `github.com/schollz/progressbar/v3` or simple terminal output
   - Report: plugin name being installed, N/M progress

### Pseudocode

```go
func runSandbox(cmd *cobra.Command, args []string) error {
    // Existing: check Docker availability

    profileName := ""

    if !ephemeral {
        if explicitProfile != "" {
            profileName = explicitProfile
        } else if cfg, err := profile.LoadProjectConfig("."); err == nil {
            profileName = cfg.Profile
            fmt.Printf("Using profile '%s' from .claudeup.json\n", profileName)
        }
    }

    // Bootstrap writes config files (existing logic)
    if profileName != "" {
        stateDir := sandbox.StateDir(profileName)
        firstRun := sandbox.IsFirstRun(stateDir)

        if firstRun || syncFlag {
            // Write settings.json, marketplaces.json
            sandbox.Bootstrap(stateDir, profile)

            // Sync plugins with progress bar
            fmt.Println("Syncing plugins...")
            sandbox.SyncPlugins(stateDir, progressBar)
        }
    }

    // Launch container with Claude or shell
    return sandbox.Run(opts)
}
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `.claudeup.json` exists but malformed | Error: "Invalid .claudeup.json: [parse error]" |
| `.claudeup.json` references missing profile | Error: "Profile 'X' from .claudeup.json not found in ~/.claudeup/profiles/ or ./.claudeup/profiles/" |
| `.claudeup.json` has empty profile field | Treat as no profile (ephemeral mode) |

## Testing

### Acceptance Tests

1. **Auto-detection happy path** - Create project with `.claudeup.json`, run `claudeup sandbox --shell`, verify profile message appears and settings applied
2. **Explicit override** - Project has `.claudeup.json`, run with `--profile other`, verify "other" is used
3. **Ephemeral skips detection** - Project has `.claudeup.json`, run with `--ephemeral`, verify no profile loaded
4. **Missing profile error** - `.claudeup.json` references non-existent profile, verify helpful error
5. **No config file** - No `.claudeup.json`, verify ephemeral mode silently
6. **Plugin sync on first run** - First sandbox run shows "Syncing plugins..." with progress, plugins installed
7. **Skip sync on subsequent runs** - Second run skips sync (fast startup)
8. **Force sync with --sync** - Even after bootstrap, `--sync` re-runs plugin installation

Tests go in `test/acceptance/sandbox_profile_detection_test.go`.
