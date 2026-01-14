// ABOUTME: Diff functionality for comparing file snapshots and generating
// ABOUTME: human-readable summaries of what changed between before/after states.
package events

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v2/internal/ui"
)

// Diff symbols for change visualization
const (
	SymbolAdded    = "+"
	SymbolRemoved  = "-"
	SymbolModified = "~"
)

// Formatting limits for diff output
const (
	maxHashDisplayLength = 8 // Number of characters to show from hash
	maxValueDepth        = 3 // Maximum nesting depth for JSON value formatting
	maxArrayDisplayItems = 3 // Maximum array items to show before truncation
)

// DiffResult contains the result of comparing two snapshots
type DiffResult struct {
	HasChanges       bool
	ContentAvailable bool
	Summary          string
	Details          []string
}

// DiffSnapshots compares before and after snapshots and returns a human-readable diff
func DiffSnapshots(before, after *Snapshot, full bool) *DiffResult {
	// Handle creation
	if before == nil && after != nil {
		return &DiffResult{
			HasChanges:       true,
			ContentAvailable: after.Content != "",
			Summary:          fmt.Sprintf("File created (%d bytes)", after.Size),
		}
	}

	// Handle deletion
	if before != nil && after == nil {
		return &DiffResult{
			HasChanges:       true,
			ContentAvailable: before.Content != "",
			Summary:          fmt.Sprintf("File deleted (was %d bytes)", before.Size),
		}
	}

	// Both nil - no change
	if before == nil && after == nil {
		return &DiffResult{
			HasChanges: false,
			Summary:    "No change (file did not exist)",
		}
	}

	// Check if content changed
	if before.Hash == after.Hash {
		return &DiffResult{
			HasChanges: false,
			Summary:    "No changes detected",
		}
	}

	// Content-based diff if available
	if before.Content != "" && after.Content != "" {
		return diffContent(before.Content, after.Content, before.Size, after.Size, full)
	}

	// Hash-only diff
	return diffHashOnly(before, after)
}

// diffContent performs a content-based diff for JSON files
func diffContent(beforeContent, afterContent string, beforeSize, afterSize int64, full bool) *DiffResult {
	// Try to parse as JSON
	var beforeJSON, afterJSON map[string]interface{}
	beforeErr := json.Unmarshal([]byte(beforeContent), &beforeJSON)
	afterErr := json.Unmarshal([]byte(afterContent), &afterJSON)

	if beforeErr == nil && afterErr == nil {
		return diffJSON(beforeJSON, afterJSON, beforeSize, afterSize, full)
	}

	// Fallback to line-based diff for non-JSON content
	return diffLines(beforeContent, afterContent, beforeSize, afterSize)
}

// diffJSON compares two JSON objects and generates a summary of changes
func diffJSON(before, after map[string]interface{}, beforeSize, afterSize int64, full bool) *DiffResult {
	// Defensive nil checks
	if before == nil && after == nil {
		return &DiffResult{
			HasChanges: false,
			Summary:    "Both JSON objects are nil",
		}
	}
	if before == nil {
		before = make(map[string]interface{})
	}
	if after == nil {
		after = make(map[string]interface{})
	}

	var changes []string
	allKeys := make(map[string]bool)

	// Collect all keys from both objects
	for k := range before {
		allKeys[k] = true
	}
	for k := range after {
		allKeys[k] = true
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Compare values
	for _, key := range keys {
		beforeVal, beforeExists := before[key]
		afterVal, afterExists := after[key]

		if !beforeExists && afterExists {
			changes = append(changes, fmt.Sprintf("%s %s: %v", SymbolAdded, key, formatValueFull(afterVal, full)))
		} else if beforeExists && !afterExists {
			changes = append(changes, fmt.Sprintf("%s %s: %v", SymbolRemoved, key, formatValueFull(beforeVal, full)))
		} else if !reflect.DeepEqual(beforeVal, afterVal) {
			// For full mode with nested objects, do deep diff
			if full {
				if beforeMap, beforeIsMap := beforeVal.(map[string]interface{}); beforeIsMap {
					if afterMap, afterIsMap := afterVal.(map[string]interface{}); afterIsMap {
						nestedDiff := diffNestedObjects(beforeMap, afterMap, 1)
						changes = append(changes, fmt.Sprintf("%s %s:\n%s", SymbolModified, key, nestedDiff))
						continue
					}
				}
			}
			changes = append(changes, fmt.Sprintf("%s %s: %v → %v", SymbolModified, key, formatValueFull(beforeVal, full), formatValueFull(afterVal, full)))
		}
	}

	sizeDiff := afterSize - beforeSize

	// Build summary with changes
	var summary string
	if len(changes) > 0 {
		summary = strings.Join(changes, "\n")
		if sizeDiff != 0 {
			summary += fmt.Sprintf("\nSize: %+d bytes", sizeDiff)
		}
	} else {
		summary = "No field changes detected"
	}

	return &DiffResult{
		HasChanges:       len(changes) > 0,
		ContentAvailable: true,
		Summary:          summary,
		Details:          changes,
	}
}

// diffLines performs a simple line-based diff
func diffLines(before, after string, beforeSize, afterSize int64) *DiffResult {
	sizeDiff := afterSize - beforeSize
	summary := fmt.Sprintf("Content changed, size: %+d bytes", sizeDiff)

	return &DiffResult{
		HasChanges:       true,
		ContentAvailable: true,
		Summary:          summary,
	}
}

// diffHashOnly generates a diff based on hash and size only
func diffHashOnly(before, after *Snapshot) *DiffResult {
	sizeDiff := after.Size - before.Size
	summary := fmt.Sprintf("Hash changed: %s → %s, size: %+d bytes",
		truncateHash(before.Hash), truncateHash(after.Hash), sizeDiff)

	return &DiffResult{
		HasChanges:       true,
		ContentAvailable: false,
		Summary:          summary,
	}
}

// formatValue formats a JSON value for display with default truncation
func formatValue(v interface{}) string {
	return formatValueWithDepth(v, 0, false)
}

// formatValueFull formats a JSON value with optional truncation based on full flag
func formatValueFull(v interface{}, full bool) string {
	return formatValueWithDepth(v, 0, full)
}

// formatValueWithDepth formats a JSON value with depth and size limits
func formatValueWithDepth(v interface{}, depth int, full bool) string {
	// Apply depth limit only when not in full mode
	if !full && depth > maxValueDepth {
		return "..."
	}

	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		// Determine array limit based on full flag
		limit := len(val)
		if !full && limit > maxArrayDisplayItems {
			limit = maxArrayDisplayItems
		}
		items := make([]string, limit)
		for i := 0; i < limit; i++ {
			items[i] = formatValueWithDepth(val[i], depth+1, full)
		}
		result := "[" + strings.Join(items, ", ")
		if !full && len(val) > maxArrayDisplayItems {
			result += fmt.Sprintf(", ...%d more", len(val)-maxArrayDisplayItems)
		}
		return result + "]"
	case map[string]interface{}:
		// Show full object structure in full mode
		if full {
			return formatMapFull(val, depth, full)
		}
		return "{...}"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// formatMapFull formats a map showing all keys and values
func formatMapFull(m map[string]interface{}, depth int, full bool) string {
	if len(m) == 0 {
		return "{}"
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Format key-value pairs
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = fmt.Sprintf("%q: %s", k, formatValueWithDepth(m[k], depth+1, full))
	}

	return "{" + strings.Join(pairs, ", ") + "}"
}

// diffNestedObjects recursively compares two objects and returns indented diff output
func diffNestedObjects(before, after map[string]interface{}, indentLevel int) string {
	var changes []string
	allKeys := make(map[string]bool)

	// Collect all keys
	for k := range before {
		allKeys[k] = true
	}
	for k := range after {
		allKeys[k] = true
	}

	// Sort keys
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	indent := strings.Repeat("  ", indentLevel)

	// Compare each key
	for _, key := range keys {
		beforeVal, beforeExists := before[key]
		afterVal, afterExists := after[key]

		if !beforeExists && afterExists {
			changes = append(changes, fmt.Sprintf("%s%s %s: %v %s",
				indent,
				ui.Success(SymbolAdded),
				ui.Bold(key),
				formatValueFull(afterVal, true),
				ui.Muted("(added)")))
		} else if beforeExists && !afterExists {
			changes = append(changes, fmt.Sprintf("%s%s %s: %v %s",
				indent,
				ui.Error(SymbolRemoved),
				ui.Bold(key),
				formatValueFull(beforeVal, true),
				ui.Muted("(removed)")))
		} else if !reflect.DeepEqual(beforeVal, afterVal) {
			// Check if both are maps for recursive diffing
			if beforeMap, beforeIsMap := beforeVal.(map[string]interface{}); beforeIsMap {
				if afterMap, afterIsMap := afterVal.(map[string]interface{}); afterIsMap {
					nestedDiff := diffNestedObjects(beforeMap, afterMap, indentLevel+1)
					changes = append(changes, fmt.Sprintf("%s%s %s:\n%s",
						indent,
						ui.Info(SymbolModified),
						ui.Bold(key),
						nestedDiff))
					continue
				}
			}
			// Check if both are arrays for element-wise diffing
			if beforeArr, beforeIsArr := beforeVal.([]interface{}); beforeIsArr {
				if afterArr, afterIsArr := afterVal.([]interface{}); afterIsArr {
					arrDiff := diffArrays(beforeArr, afterArr, indentLevel, key)
					if arrDiff != "" {
						changes = append(changes, arrDiff)
						continue
					}
				}
			}
			// Not nested structures, show simple before → after
			changes = append(changes, fmt.Sprintf("%s%s %s: %v → %v",
				indent,
				ui.Info(SymbolModified),
				ui.Bold(key),
				formatValueFull(beforeVal, true),
				formatValueFull(afterVal, true)))
		}
	}

	return strings.Join(changes, "\n")
}

// diffArrays compares two arrays and returns a formatted diff
func diffArrays(before, after []interface{}, indentLevel int, key string) string {
	indent := strings.Repeat("  ", indentLevel)

	// For arrays with single object elements (common in plugin metadata)
	if len(before) == 1 && len(after) == 1 {
		if beforeMap, beforeIsMap := before[0].(map[string]interface{}); beforeIsMap {
			if afterMap, afterIsMap := after[0].(map[string]interface{}); afterIsMap {
				// Both arrays contain single objects - diff the objects
				nestedDiff := diffNestedObjects(beforeMap, afterMap, indentLevel+1)
				return fmt.Sprintf("%s%s %s:\n%s",
					indent,
					ui.Info(SymbolModified),
					ui.Bold(key),
					nestedDiff)
			}
		}
	}

	// For different length arrays or non-object elements, show full arrays
	return ""
}

// truncateHash returns first N chars of a hash for display
func truncateHash(hash string) string {
	if len(hash) > maxHashDisplayLength {
		return hash[:maxHashDisplayLength]
	}
	return hash
}
