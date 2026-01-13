# Plugin List Filtering - Status Filters

**Date:** 2026-01-13
**Status:** Approved

## Problem

With many marketplaces installed, `claudeup plugin list` shows 60+ plugins, making it hard to find enabled ones.

## Solution

Add `--enabled` and `--disabled` flags to filter the plugin list by status.

## Interface

```bash
claudeup plugin list --enabled    # Only enabled plugins
claudeup plugin list --disabled   # Only disabled plugins
claudeup plugin list              # All plugins (unchanged)
```

## Behavior

- Flags are mutually exclusive (using both is an error)
- Output format unchanged, just filtered
- Summary footer shows filtered context: `Showing: 11 enabled (of 63 total)`

## Implementation

**Files:**
1. `internal/commands/plugin.go` - Add flags, filter in `runPluginList`
2. `internal/commands/plugin_stats.go` - Update `printPluginListFooter` signature
3. `test/acceptance/plugins_test.go` - Add acceptance tests

**Changes to `plugin.go`:**
- Add `pluginFilterEnabled` and `pluginFilterDisabled` bool flags
- Filter `names` slice after sorting based on `analysis[name].IsEnabled()`
- Pass filter state to footer function

**Changes to `plugin_stats.go`:**
- Update `printPluginListFooter` to accept shown vs total counts
- Display "Showing: X enabled (of Y total)" when filtered
