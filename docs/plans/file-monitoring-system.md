# File Monitoring & Notification System Design

## Problem Statement

Users need visibility into when and why claudeup modifies Claude configuration files. Currently, there's no audit trail or notification system to answer questions like:
- "Why did my plugin settings change?"
- "What operation modified this file?"
- "When was this file last changed by claudeup?"
- "What's the complete history for this project?"

This lack of transparency makes troubleshooting difficult, especially in team environments where multiple people apply profiles.

## Design Goals

1. **Transparent Operations** - Every file modification should be logged with context
2. **Audit Trail** - Queryable history of all changes
3. **Real-time Notifications** - Optional alerts when files change
4. **External Change Detection** - Detect when files change outside claudeup
5. **Troubleshooting Tools** - CLI commands to investigate issues
6. **Minimal Overhead** - No performance impact on normal operations

## Architecture

### Three-Layer Monitoring System

```text
┌─────────────────────────────────────────────┐
│  Layer 1: Operation Instrumentation         │
│  (Track changes made by claudeup commands)  │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  Layer 2: File Change Detection             │
│  (Detect external modifications)            │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  Layer 3: Event Processing & Notification   │
│  (Log, alert, and provide query interface)  │
└─────────────────────────────────────────────┘
```

### Layer 1: Operation Instrumentation

**Approach:** Wrap all file write operations with event tracking.

**Implementation:**
```go
// internal/events/tracker.go
package events

type FileOperation struct {
    Timestamp   time.Time
    Operation   string // "profile apply", "plugin install", etc.
    File        string // Absolute path
    Scope       string // user/project/local
    ChangeType  string // create/update/delete
    Before      *Snapshot // Hash + size before change
    After       *Snapshot // Hash + size after change
    Context     map[string]interface{} // Additional metadata
    Error       error
}

type Tracker struct {
    enabled bool
    events  chan *FileOperation
    writer  EventWriter
}

func (t *Tracker) RecordFileWrite(op string, file string, scope string, fn func() error) error {
    if !t.enabled {
        return fn()
    }

    // Snapshot before
    before := t.snapshot(file)

    // Execute operation
    err := fn()

    // Snapshot after
    after := t.snapshot(file)

    // Record event
    t.events <- &FileOperation{
        Timestamp: time.Now(),
        Operation: op,
        File: file,
        Scope: scope,
        ChangeType: inferChangeType(before, after),
        Before: before,
        After: after,
        Error: err,
    }

    return err
}
```

**Integration Points:**
- `internal/claude/plugins.go:SavePlugins()` - Wrap with tracker
- `internal/claude/settings.go:SaveSettings()` - Wrap with tracker
- `internal/profile/apply.go` - Track each apply operation
- `internal/backup/backup.go` - Track backup creation
- All other file write operations

### Layer 2: File Change Detection

**Approach:** Optional filesystem watcher to detect external changes.

**Implementation:**
```go
// internal/events/watcher.go
package events

import "github.com/fsnotify/fsnotify"

type FileWatcher struct {
    watcher *fsnotify.Watcher
    tracked map[string]WatchConfig
    events  chan *FileOperation
}

type WatchConfig struct {
    Path        string
    Description string // "Claude user settings", "Profile definition", etc.
    Owner       string // "claude-cli" or "claudeup"
}

func (w *FileWatcher) Watch(paths ...WatchConfig) error {
    for _, cfg := range paths {
        if err := w.watcher.Add(cfg.Path); err != nil {
            return err
        }
        w.tracked[cfg.Path] = cfg
    }
    return nil
}

func (w *FileWatcher) Run(ctx context.Context) {
    for {
        select {
        case event := <-w.watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                w.events <- &FileOperation{
                    Timestamp: time.Now(),
                    Operation: "external-change",
                    File: event.Name,
                    ChangeType: "update",
                    Context: map[string]interface{}{
                        "description": w.tracked[event.Name].Description,
                        "owner": w.tracked[event.Name].Owner,
                    },
                }
            }
        case <-ctx.Done():
            return
        }
    }
}
```

**Watched Files:**
- All files documented in `docs/file-operations.md`
- User can configure which files to watch via `~/.claudeup/config.json`

**Default Watch List:**
```json
{
  "monitoring": {
    "enabled": true,
    "watchFiles": [
      "~/.claude/settings.json",
      "~/.claude/plugins/installed_plugins.json",
      "./.claude/settings.json",
      "./.claudeup.json"
    ]
  }
}
```

### Layer 3: Event Processing & Storage

**Approach:** Write events to structured log files with queryable format.

**Event Storage:**
```bash
~/.claudeup/events/
├── operations.log       # Operation-level events (claudeup commands)
├── file-changes.log     # All file changes (claudeup + external)
└── audit-trail.db       # SQLite for queries (optional)
```

**Log Format (JSONL):**
```json
{"timestamp":"2024-12-26T10:15:30Z","operation":"profile apply","scope":"user","profile":"default","files":[{"path":"~/.claude/settings.json","change":"update","before_hash":"abc123","after_hash":"def456"}],"user":"markalston"}
{"timestamp":"2024-12-26T10:16:45Z","operation":"external-change","file":"~/.claude/settings.json","change":"update","source":"unknown"}
```

**Event Writer:**
```go
// internal/events/writer.go
package events

type EventWriter interface {
    Write(event *FileOperation) error
    Query(filters EventFilters) ([]*FileOperation, error)
}

type JSONLWriter struct {
    logPath string
    mu      sync.Mutex
}

func (w *JSONLWriter) Write(event *FileOperation) error {
    w.mu.Lock()
    defer w.mu.Unlock()

    f, err := os.OpenFile(w.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    _, err = f.Write(append(data, '\n'))
    return err
}

type EventFilters struct {
    File      string
    Operation string
    Since     time.Time
    Scope     string
    Limit     int
}

func (w *JSONLWriter) Query(filters EventFilters) ([]*FileOperation, error) {
    // Read log file, parse JSONL, apply filters
    // Return matching events sorted by timestamp desc
}
```

## CLI Commands

### `claudeup events`

Show recent file change events.

```bash
# Show all recent events
claudeup events

# Show events for a specific file
claudeup events --file ~/.claude/settings.json

# Show events from last 24 hours
claudeup events --since 24h

# Show events for a specific operation
claudeup events --operation "profile apply"

# Show events in a project
claudeup events --project .

# Export events as JSON
claudeup events --json > events.json
```

**Output Format:**
```text
Recent File Changes:
────────────────────────────────────────────────────────────────────

2024-12-26 10:15:30  PROFILE APPLY (user scope)
  Profile: default
  Modified: ~/.claude/settings.json
  Changes: +3 plugins, -1 plugin
  User: markalston

2024-12-26 10:16:45  EXTERNAL CHANGE
  File: ~/.claude/settings.json
  Warning: File modified outside claudeup

2024-12-26 09:30:12  BACKUP CREATED
  Scope: user
  File: ~/.claudeup/backups/user-scope.json
  Reason: Before profile apply
```

### `claudeup events diff`

Show detailed diff of a file change.

```bash
# Show what changed in the most recent modification
claudeup events diff --file ~/.claude/settings.json

# Show diff for a specific timestamp
claudeup events diff --file ~/.claude/settings.json --at "2024-12-26 10:15:30"
```

**Output:**
```text
File: ~/.claude/settings.json
Operation: profile apply (user scope)
Timestamp: 2024-12-26 10:15:30

Changes to enabledPlugins:
  + backend-development@claude-code-workflows: true
  + python-development@claude-code-workflows: true
  + systems-programming@claude-code-workflows: true
  - old-plugin@some-marketplace: true
```

### `claudeup events audit`

Generate audit trail for a project or file.

```bash
# Audit trail for current project
claudeup events audit

# Audit trail for a specific file
claudeup events audit --file ./.claudeup.json

# Export audit trail as markdown report
claudeup events audit --format markdown > audit-report.md
```

**Output:**
```text
Audit Trail: /Users/markalston/workspace/myproject
════════════════════════════════════════════════

2024-12-26 10:15:30  markalston
  Action: Applied profile 'python-dev' (project scope)
  Files Modified:
    - ./.claude/settings.json (enabledPlugins updated)
    - ./.mcp.json (created with 2 MCP servers)
    - ./.claudeup.json (profile metadata recorded)

2024-12-25 15:20:00  teammate@example.com
  Action: Applied profile 'default' (project scope)
  Files Modified:
    - ./.claude/settings.json (enabledPlugins updated)
```

### `claudeup events watch`

Real-time file change monitoring (foreground process).

```bash
# Watch all configured files
claudeup events watch

# Watch specific files
claudeup events watch --file ~/.claude/settings.json --file ./.claudeup.json

# Watch with notifications
claudeup events watch --notify
```

**Output (live updating):**
```text
Watching file changes... (Press Ctrl+C to stop)

[10:15:30] ~/.claude/settings.json
           Operation: profile apply (user scope)
           Changes: +3 plugins, -1 plugin

[10:16:45] ~/.claude/settings.json
           ⚠️  EXTERNAL CHANGE DETECTED
           File modified outside claudeup
```

### `claudeup events clear`

Clear event history.

```bash
# Clear all events
claudeup events clear

# Clear events older than 30 days
claudeup events clear --older-than 30d

# Clear events for a specific file
claudeup events clear --file ~/.claude/settings.json
```

## Configuration

Add monitoring configuration to `~/.claudeup/config.json`:

```json
{
  "monitoring": {
    "enabled": true,
    "watchFiles": [
      "~/.claude/settings.json",
      "~/.claude/plugins/installed_plugins.json",
      "~/.claude/plugins/known_marketplaces.json",
      "./.claude/settings.json",
      "./.claudeup.json",
      "./.mcp.json"
    ],
    "notifications": {
      "enabled": false,
      "method": "desktop",  // "desktop", "log", "webhook"
      "webhookUrl": ""
    },
    "retention": {
      "maxEvents": 10000,
      "maxAge": "90d"
    }
  }
}
```

## Notification Methods

### Desktop Notifications (macOS/Linux)

```go
// internal/events/notify.go
package events

import "github.com/gen2brain/beeep"

func (n *Notifier) SendDesktop(event *FileOperation) error {
    title := fmt.Sprintf("Claude File Changed: %s", filepath.Base(event.File))
    message := fmt.Sprintf("Operation: %s\nScope: %s", event.Operation, event.Scope)
    return beeep.Notify(title, message, "")
}
```

### Webhook Notifications

```go
func (n *Notifier) SendWebhook(event *FileOperation) error {
    payload := map[string]interface{}{
        "timestamp": event.Timestamp,
        "operation": event.Operation,
        "file": event.File,
        "scope": event.Scope,
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    resp, err := http.Post(n.webhookURL, "application/json", bytes.NewReader(data))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("webhook failed: %s", resp.Status)
    }

    return nil
}
```

## Implementation Phases

### Phase 1: Operation Instrumentation (MVP)
**Goal:** Track claudeup's own operations
- [x] Create `internal/events` package with Tracker
- [x] Wrap all file write operations
- [x] Implement JSONL log writer
- [x] Add `claudeup events` command (read-only)
- [x] Add monitoring config to global config
- [ ] **Retention policy enforcement** - Config exists but not enforced yet (Phase 2)

**Value:** Users can see what claudeup did and when

**Phase 1 Limitations:**
- Log files will grow unbounded (no automatic cleanup)
- Users must manually clear old events using `claudeup events clear` (coming in Phase 2)
- For Phase 1, monitor log file size and delete manually if needed
- Operation names are generic (e.g., "plugin update" doesn't distinguish between profile apply vs plugin cleanup)
- Events are written synchronously (no async queue)
- Double file read for snapshots (before/after hashing)
- In-memory query processing (works well for <3MB logs)

### Phase 2: Query & Diff Tools
**Goal:** Enable troubleshooting
- [ ] Implement `events diff` command
- [ ] Implement `events audit` command
- [ ] Add filtering options to `events` command
- [ ] Calculate and display meaningful diffs
- [ ] **Add operation context** - Distinguish between "profile apply: plugin update" vs "cleanup: plugin update"
- [ ] **Performance: Async event queue** - Background goroutine for non-blocking event writes
- [ ] **Performance: Snapshot optimization** - Skip "before" snapshot for create operations
- [ ] **Performance: Query indexing** - Add SQLite index for frequently queried fields

**Value:** Users can understand *what* changed, not just that it changed

### Phase 3: External Change Detection
**Goal:** Detect modifications outside claudeup
- [ ] Implement FileWatcher with fsnotify
- [ ] Add background watcher process
- [ ] Implement `events watch` command
- [ ] Add external change warnings to status output

**Value:** Users know when tools or teammates modify files directly

### Phase 4: Notifications
**Goal:** Real-time alerting
- [ ] Implement desktop notifications
- [ ] Implement webhook notifications
- [ ] Add notification preferences to config
- [ ] Add notification silencing/filtering

**Value:** Users get proactive alerts instead of discovering issues later

### Phase 5: Advanced Features
**Goal:** Team collaboration & compliance
- [ ] SQLite storage for complex queries
- [ ] User attribution (track who made changes)
- [ ] Change approval workflows
- [ ] Export audit reports (PDF, HTML)
- [ ] Integration with git commit hooks

**Value:** Enterprise teams can enforce policy and maintain compliance

## Testing Strategy

### Unit Tests
- Event tracker records correct metadata
- JSONL writer formats events correctly
- Query filters work as expected
- Snapshot hashing is consistent

### Integration Tests
- File operations trigger events
- Events are written to log files
- Watch mode detects external changes
- Notifications are sent correctly

### Acceptance Tests
- `claudeup events` shows recent changes
- `claudeup events diff` displays meaningful diffs
- `claudeup events audit` generates complete trail
- External changes are detected and reported

## Security Considerations

1. **Log File Permissions** - Events logs should be user-readable only (0600)
2. **Sensitive Data** - Never log secret values, only references
3. **File Path Sanitization** - Prevent log injection via malicious paths
4. **Webhook Security** - Support authentication for webhook endpoints
5. **Storage Limits** - Enforce retention policies to prevent disk exhaustion

## Performance Considerations

1. **Async Event Writing** - Don't block operations waiting for logs
2. **Bounded Event Queue** - Prevent memory exhaustion
3. **Log Rotation** - Automatically rotate large log files
4. **Efficient Queries** - Index frequently queried fields (SQLite)
5. **Watcher Efficiency** - Use OS-native file watching (inotify/FSEvents)

## Example Use Cases

### Use Case 1: Profile Application Debugging

**Scenario:** User applies a profile but plugins aren't enabled as expected.

**Workflow:**
```bash
# See what happened during profile apply
$ claudeup events --operation "profile apply" --since 1h

# Check the diff to see what actually changed
$ claudeup events diff --file ~/.claude/settings.json

# Verify backup was created
$ claudeup events --file ~/.claudeup/backups/user-scope.json
```

### Use Case 2: Team Conflict Detection

**Scenario:** Multiple team members apply different profiles to the same project.

**Workflow:**
```bash
# Generate audit trail for the project
$ claudeup events audit --format markdown > project-audit.md

# See who made the last change
$ claudeup events --project . --limit 1

# Restore to known good state
$ claudeup scope restore --scope project
```

### Use Case 3: External Modification Detection

**Scenario:** IDE extension modifies Claude settings directly.

**Workflow:**
```bash
# Watch files in real-time
$ claudeup events watch --notify

# [Notification appears]: "External change detected in settings.json"

# Investigate what changed
$ claudeup events diff --file ~/.claude/settings.json

# Reapply profile to restore desired state
$ claudeup profile apply default
```

## Open Questions

1. **Event Retention:** Default to 90 days or keep forever with rotation?
2. **Watch Mode:** Run as background daemon or foreground process?
3. **Change Detection:** Calculate semantic diffs (plugin changes) or just file hashes?
4. **Concurrency:** Handle multiple claudeup processes modifying files simultaneously?
5. **Backwards Compatibility:** Migrate existing users to new event system?

## Success Metrics

- Users can answer "why did this change?" in < 30 seconds
- 90% of file modifications are attributed to a claudeup operation
- External changes are detected within 1 second
- Event logs consume < 10MB per month of typical usage
- Zero performance degradation for instrumented operations

## Future Enhancements

- **Git Integration** - Auto-commit changes with descriptive messages
- **Rollback System** - `claudeup undo` to revert recent changes
- **Change Approvals** - Require confirmation before applying destructive changes
- **Compliance Reports** - Generate SOC2/audit-friendly reports
- **Dashboard UI** - Web UI for visualizing change history
- **ML-based Anomaly Detection** - Flag suspicious change patterns
