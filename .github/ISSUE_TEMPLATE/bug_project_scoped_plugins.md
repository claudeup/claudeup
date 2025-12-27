# Bug: Profile Comparison Ignores Claude Code's Project-Scoped Plugin Installations

## Summary
claudeup does not respect Claude Code's project-scoped plugin installations when comparing profiles, leading to incorrect "(modified)" indicators and unreliable profile status across different projects.

## Current Behavior

When running `claudeup profile list` from different directories:
- The active profile changes based on working directory
- The "(modified)" status is unreliable
- All plugins from `installed_plugins.json` are included in comparison, regardless of which project they belong to

### Example

```bash
# From ~/.claude directory
$ claudeup profile list
* base-tools           Base tools for everyday claude-ing (modified)
  claudeup             My claudeup setup with all my favorite tools

# From ~/workspace/claudeup directory
$ claudeup profile list
  base-tools           Base tools for everyday claude-ing
* claudeup             My claudeup setup with all my favorite tools
```

## Root Cause

**File:** `internal/claude/plugins.go`

The `GetPlugin()` and `GetAllPlugins()` methods:
1. Return plugins with "user" scope OR the first instance
2. **Completely ignore the `projectPath` field**
3. Do not filter based on current working directory

```go
// GetPlugin retrieves a plugin by name, defaulting to "user" scope
func (r *PluginRegistry) GetPlugin(pluginName string) (PluginMetadata, bool) {
    instances, exists := r.Plugins[pluginName]
    if !exists || len(instances) == 0 {
        return PluginMetadata{}, false
    }
    // Return first instance with "user" scope, or first instance if no user scope
    for _, inst := range instances {
        if inst.Scope == "user" || inst.Scope == "" {
            return inst, true
        }
    }
    return instances[0], true  // ← Returns project-scoped plugin without checking projectPath!
}
```

## Expected Behavior

claudeup should:
1. **Detect current project context** from working directory
2. **Filter plugins** to only those matching:
   - Current project's `projectPath`, OR
   - Global/user-scoped plugins (no projectPath)
3. **Compare profiles** against this filtered set
4. **Show consistent status** when run from the same project directory

### Example (Expected)

```bash
# From ~/.claude directory (no project context)
$ claudeup profile list
* base-tools           Base tools for everyday claude-ing
  # Shows only global/user-scoped plugins

# From ~/workspace/claudeup directory
$ claudeup profile list
* claudeup             My claudeup setup with all my favorite tools
  # Shows plugins for THIS project + global plugins
```

## Impact

**Medium-High Priority**
- Affects users working in multiple projects
- Makes profile management unreliable
- "(modified)" indicator becomes meaningless
- Could lead to unintended profile applications

## Claude Code Plugin Format (v2)

Claude Code's `installed_plugins.json` uses project-scoped installations:

```json
{
  "version": 2,
  "plugins": {
    "backend-development@claude-code-workflows": [
      {
        "scope": "project",
        "projectPath": "/Users/markalston/workspace/claudeup",
        "installPath": "/Users/markalston/.claude/plugins/cache/...",
        "version": "1.2.4",
        "installedAt": "2025-12-26T05:14:20.184Z",
        "lastUpdated": "2025-12-26T05:14:20.184Z",
        "gitCommitSha": "e4dade12847a99d277d81192c2966e9b61c0d3f1",
        "isLocal": true
      }
    ]
  }
}
```

## Proposed Fix

### 1. Add PluginMetadata.ProjectPath field
```go
type PluginMetadata struct {
    Scope        string  `json:"scope"`
    ProjectPath  *string `json:"projectPath,omitempty"` // ← Add this
    Version      string  `json:"version"`
    // ... other fields
}
```

### 2. Create project context detection
```go
// GetCurrentProjectPath returns the absolute path of the current working directory
// Used to filter project-scoped plugins
func GetCurrentProjectPath() (string, error) {
    cwd, err := os.Getwd()
    if err != nil {
        return "", err
    }
    return filepath.Abs(cwd)
}
```

### 3. Add project-aware plugin filtering
```go
// GetPluginsForProject returns plugins relevant to the current project context
// Includes:
// - Plugins scoped to the specified projectPath
// - Plugins with no projectPath (global/user-scoped)
func (r *PluginRegistry) GetPluginsForProject(projectPath string) map[string]PluginMetadata {
    result := make(map[string]PluginMetadata)
    absProjectPath, _ := filepath.Abs(projectPath)

    for name, instances := range r.Plugins {
        for _, inst := range instances {
            // Include if project matches OR no project specified (global)
            if inst.ProjectPath == nil || *inst.ProjectPath == absProjectPath {
                result[name] = inst
                break
            }
        }
    }
    return result
}
```

### 4. Update profile comparison to use project context
```go
// In profile comparison logic:
projectPath, err := GetCurrentProjectPath()
if err != nil {
    // Fall back to current behavior
    projectPath = ""
}
currentPlugins := registry.GetPluginsForProject(projectPath)
```

## Testing Requirements

- [ ] Test plugin filtering with project-scoped plugins
- [ ] Test plugin filtering with global/user-scoped plugins
- [ ] Test plugin filtering with mixed scope scenarios
- [ ] Test behavior when run from different directories
- [ ] Test backward compatibility with v1 format (no projectPath)
- [ ] Test profile comparison across different project contexts

## Related Files

- `internal/claude/plugins.go` - Core plugin loading/filtering
- `internal/claude/plugins_test.go` - Tests for plugin logic
- `internal/profile/compare.go` - Profile comparison logic
- `internal/commands/profile_list.go` - Profile list command

## Workaround

Until fixed, always run `claudeup profile list` from the project directory where you applied the profile.

## Discovery

Discovered through user investigation when noticing different profile status from different directories. Event tracking system confirmed that profiles are global but plugins are project-scoped.
