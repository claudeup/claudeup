# File Monitoring System Design

**Date:** 2024-12-26
**Status:** Validated Design
**Author:** Brainstorming session with Mark

## Problem Statement

Users need visibility into when and why `claudeup` modifies Claude CLI configuration files. When Claude Code breaks or behaves unexpectedly, it's difficult to answer:
- "Why did my plugin settings change?"
- "What operation modified this file?"
- "Is claudeup working correctly?"

The most critical files are Claude CLI's own configuration under `~/.claude/*`. When these files are corrupted or modified unexpectedly, Claude Code stops working entirely.

## Requirements

**Primary Use Cases:**
1. **Troubleshooting** - Understand why settings changed unexpectedly
2. **Development/Debugging** - Verify claudeup operations work correctly

**Priorities:**
- Focus on `~/.claude/*` file modifications first (highest breakage risk)
- Start with operation tracking, add external change detection later
- Provide query tool as core feature with optional notifications
- Semantic summaries by default, full diffs on-demand

## Architecture

The monitoring system consists of three layers:

### Layer 1: Event Tracker

A lightweight wrapper that intercepts file write operations. When any code tries to modify a `~/.claude/*` file, it goes through the tracker first. The tracker:

1. Captures a "before" snapshot (file hash)
2. Executes the write operation
3. Captures an "after" snapshot
4. Computes semantic changes (e.g., "+3 plugins, -1 plugin")
5. Writes event to log file

This approach preserves semantic context—we know it was "profile apply" that added those plugins, not just "file changed."

### Layer 2: Event Storage

Events are written to `~/.claudeup/events/operations.log` in JSONL format (one JSON object per line). This makes the log:
- Human-readable for debugging
- Machine-parseable for queries
- Appendable without locking the entire file

Each event includes:
- Timestamp
- Operation name (e.g., "profile apply")
- File path
- Scope (user/project/local)
- Before/after hashes
- Semantic context (plugin changes, marketplace changes)

### Layer 3: Query Interface

A new `claudeup events` command reads the log file, applies filters (by file, operation, timeframe), and displays results.

For semantic summaries, we re-parse the before/after states to show "+3 plugins, -1 marketplace" instead of raw JSON diffs.

For full diffs, we can reconstruct complete file contents on-demand using the stored hashes and current file state.

## Components & Data Structures

### Package: `internal/events`

**Tracker** - Event recording wrapper
```go
type Tracker struct {
    enabled bool           // Can be disabled via config
    logPath string         // ~/.claudeup/events/operations.log
    mu      sync.Mutex     // Thread-safe writes
}

func (t *Tracker) RecordFileWrite(
    operation string,      // "profile apply"
    file string,          // "/Users/mark/.claude/settings.json"
    scope string,         // "user"
    fn func() error       // The actual write operation
) error
```

**FileOperation** - Event data structure
```go
type FileOperation struct {
    Timestamp   time.Time
    Operation   string                 // "profile apply", "plugin install"
    File        string                 // Absolute path
    Scope       string                 // user/project/local
    BeforeHash  string                 // SHA256 of file before
    AfterHash   string                 // SHA256 of file after
    Changes     map[string]interface{} // Semantic changes
    Error       error
}
```

**EventReader** - Query and display events
```go
type EventReader struct {
    logPath string
}

func (r *EventReader) Query(filters EventFilters) ([]*FileOperation, error)
func (r *EventReader) CalculateDiff(event *FileOperation) (string, error)
```

The tracker is initialized once at startup and shared across all commands. It's opt-in via config so power users can disable it if needed.

## Data Flow

### Capturing Events (Write Path)

1. User runs `claudeup profile apply default`
2. Apply code calls `claude.SaveSettings(claudeDir, settings)`
3. **Wrapper intercepts**: `tracker.RecordFileWrite("profile apply", settingsPath, "user", fn)`
4. Tracker snapshots file before (reads `settings.json`, calculates SHA256 hash)
5. Tracker executes the actual write: `claude.SaveSettings()`
6. Tracker snapshots file after (re-reads file, calculates new hash)
7. Tracker computes semantic changes by comparing before/after plugin lists
8. Tracker appends JSONL event to `~/.claudeup/events/operations.log`
9. If notifications enabled, sends desktop notification: "Profile applied: +3 plugins"

### Querying Events (Read Path)

1. User runs `claudeup events --file ~/.claude/settings.json --since 24h`
2. EventReader opens `operations.log`
3. Reads file line-by-line, parsing JSONL
4. Applies filters (file path matches, timestamp within 24h)
5. For matching events, computes semantic summary if not already stored
6. Displays results sorted by timestamp (newest first)
7. If user runs `claudeup events diff`, re-reads original file at before/after hashes to show full diff

**Performance:** JSONL format means we can `tail` the file for recent events without parsing the entire log. For queries spanning long timeframes, we read the full file but it's just JSON parsing—fast enough for typical usage.

## Integration Points

### Priority 1: `~/.claude/*` File Writes

These are the critical files that break Claude Code when corrupted:

1. **`internal/claude/settings.go:SaveSettings()`**
   - Wraps writes to `~/.claude/settings.json`
   - Triggered by: profile apply (user scope), plugin enable/disable
   - Semantic changes: "+3 enabled plugins, -1 disabled plugin"

2. **`internal/claude/settings.go:SaveSettingsForScope()`**
   - Wraps writes to `./.claude/settings.json` (project) and `./.claude/settings.local.json` (local)
   - Triggered by: profile apply (project/local scope)
   - Semantic changes: same as above, but for different scopes

3. **`internal/claude/plugins.go:SavePlugins()`**
   - Wraps writes to `~/.claude/plugins/installed_plugins.json`
   - Triggered by: profile apply (when removing plugins from registry)
   - Semantic changes: "-1 plugin registration"

4. **`internal/claudemarketplaces.go:SaveMarketplaces()`**
   - Wraps writes to `~/.claude/plugins/known_marketplaces.json`
   - Triggered by: marketplace operations
   - Semantic changes: "+1 marketplace" or "-1 marketplace"

### Priority 2: MCP Server Files (optional in Phase 1)

5. **`internal/profile/apply.go:` MCP add operations**
   - Track when we modify `~/.claude.json` (user MCP servers)
   - Note: We call `claude mcp add` so we don't directly write this file, but we can log the operation

### Implementation Pattern

Each of these 4-5 locations gets a 3-line change:
```go
// Before
return SaveSettings(claudeDir, settings)

// After
return tracker.RecordFileWrite("profile apply", settingsPath, "user", func() error {
    return SaveSettings(claudeDir, settings)
})
```

## Error Handling

**Principle: Never break the operation if tracking fails**

The monitoring system is observability infrastructure—it should never cause the actual operation to fail.

### Error Scenarios

1. **Log file write failure** (disk full, permissions)
   - Tracker prints warning to stderr: "Warning: could not record event to log"
   - Operation proceeds successfully
   - User can still use claudeup, just loses that event in history

2. **Before snapshot fails** (file doesn't exist yet)
   - Common case: creating new file for first time
   - Record event with `BeforeHash: ""` (empty means "file didn't exist")
   - Operation proceeds normally

3. **After snapshot fails** (file deleted immediately after write)
   - Record event with `AfterHash: ""` and note in error field
   - Operation already succeeded, so this is just logged

4. **Semantic change calculation fails** (malformed JSON)
   - Store raw before/after hashes but skip semantic summary
   - Event still recorded with basic metadata
   - Query tool can still show "file changed" even if it can't parse the diff

5. **Tracker disabled via config**
   - All RecordFileWrite calls become no-ops
   - Zero overhead when monitoring is turned off

**Graceful degradation:** Even if tracking is partially broken (can't parse semantic changes), we still record basic events (timestamp, file, operation). This ensures visibility is never completely lost.

## CLI Interface

### Primary Command: `claudeup events`

Show recent file changes with smart defaults:

```bash
# Show last 20 events
claudeup events

# Show events for specific file
claudeup events --file ~/.claude/settings.json

# Show events from last 24 hours
claudeup events --since 24h

# Show only profile apply operations
claudeup events --operation "profile apply"

# Combine filters
claudeup events --file ~/.claude/settings.json --since 1h --operation "profile apply"
```

**Output format:**
```text
Recent File Changes:
────────────────────────────────────────────────

2024-12-26 10:15:30  PROFILE APPLY (user scope)
  File: ~/.claude/settings.json
  Changes: +3 plugins, -1 plugin

2024-12-26 09:45:12  PLUGIN INSTALL
  File: ~/.claude/plugins/installed_plugins.json
  Changes: Added backend-development@claude-code-workflows
```

### Diff Command: `claudeup events diff`

Show detailed changes:

```bash
# Diff most recent change to settings.json
claudeup events diff --file ~/.claude/settings.json

# Diff specific event by timestamp or index
claudeup events diff --at "2024-12-26 10:15:30"
```

**Output shows actual plugin names added/removed:**
```text
Changes to enabledPlugins:
  + backend-development@claude-code-workflows: true
  + python-development@claude-code-workflows: true
  - old-plugin@some-marketplace: true
```

### Configuration

Add to `~/.claudeup/config.json`:
```json
{
  "monitoring": {
    "enabled": true,
    "maxEvents": 1000,
    "notifications": {
      "enabled": false,
      "desktop": true
    }
  }
}
```

## Testing Strategy

### Unit Tests

1. **Tracker behavior**
   - Test that RecordFileWrite captures before/after snapshots correctly
   - Test that tracking failures don't break the wrapped operation
   - Test that disabled tracker is a true no-op (zero overhead)

2. **Event storage**
   - Test JSONL serialization/deserialization
   - Test that concurrent writes don't corrupt the log file (mutex works)
   - Test event parsing with malformed JSON (graceful degradation)

3. **Semantic change calculation**
   - Test plugin diff: "+3 plugins, -1 plugin"
   - Test marketplace diff: "+1 marketplace"
   - Test handling of malformed settings.json (doesn't crash)

### Integration Tests

1. **Profile apply tracking**
   - Run `profile apply`, verify event is logged with correct operation name
   - Verify semantic changes match actual file modifications
   - Verify hashes allow reconstruction of before/after states

2. **Query filtering**
   - Create events with different files/operations/timestamps
   - Verify filters return correct subset
   - Verify sorting (newest first)

### Acceptance Tests

1. **Troubleshooting workflow**
   - Apply profile that changes plugins
   - Run `claudeup events` - verify recent change shown
   - Run `claudeup events diff` - verify exact plugins listed
   - Validates end-to-end troubleshooting experience

2. **Error resilience**
   - Fill disk to 100%, verify profile apply still succeeds (just warns about logging)
   - Delete log file mid-operation, verify graceful recovery

### Manual Testing Checklist

- Apply profile with mix of plugin additions/removals
- Check events show semantic summary correctly
- Verify diff reconstructs actual changes
- Test on fresh install (no existing log file)

## Implementation Plan

### Phase 1: Core Monitoring (MVP)
**Goal:** Track claudeup's modifications to `~/.claude/*` files

**Tasks:**
1. Create `internal/events` package with Tracker, FileOperation, EventReader
2. Add monitoring config to `~/.claudeup/config.json` structure
3. Wrap the 4 critical file write operations (SaveSettings, SaveSettingsForScope, SavePlugins, SaveMarketplaces)
4. Implement JSONL log writer
5. Add `claudeup events` command (basic listing)
6. Add `claudeup events diff` command
7. Add unit tests for tracker and event storage
8. Add integration tests for profile apply tracking

**Value:** Users can troubleshoot `~/.claude/*` file changes

### Phase 2: Enhanced Querying
**Goal:** Better troubleshooting tools

**Tasks:**
1. Add time-based filtering (`--since`, `--before`)
2. Add operation filtering (`--operation`)
3. Add semantic summary calculation
4. Improve output formatting
5. Add acceptance tests

**Value:** Users can pinpoint specific changes quickly

### Phase 3: Notifications (Optional)
**Goal:** Proactive awareness of changes

**Tasks:**
1. Desktop notification support (macOS/Linux)
2. Notification preferences in config
3. Smart notification filtering (don't spam)

**Value:** Users know immediately when something changes

### Phase 4: External Change Detection (Future)
**Goal:** Detect modifications outside claudeup

**Tasks:**
1. Add fsnotify dependency
2. Implement file watcher for `~/.claude/*`
3. Add `claudeup events watch` command
4. Distinguish claudeup changes from external changes

**Value:** Complete visibility into all modifications

## Success Criteria

- Users can answer "why did this change?" in < 30 seconds
- 95% of `~/.claude/*` file modifications are tracked
- Zero performance degradation for instrumented operations
- Event logs consume < 1MB per month of typical usage
- Tracking failures never break profile apply operations

## Security Considerations

1. **Log File Permissions** - Event logs should be user-readable only (0600)
2. **Sensitive Data** - Never log secret values, only references
3. **File Path Sanitization** - Prevent log injection via malicious paths
4. **Storage Limits** - Enforce maxEvents to prevent disk exhaustion

## Future Enhancements

- **Git Integration** - Auto-commit changes with descriptive messages
- **Rollback System** - `claudeup undo` to revert recent changes
- **Audit Reports** - Generate markdown reports for compliance
- **SQLite Storage** - Optional structured storage for complex queries
- **Webhook Notifications** - Send events to external systems
