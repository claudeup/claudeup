// ABOUTME: Diff functionality for comparing file snapshots and generating
// ABOUTME: human-readable summaries of what changed between before/after states.
package events

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Diff symbols for change visualization
const (
	SymbolAdded    = "+"
	SymbolRemoved  = "-"
	SymbolModified = "~"
)

// DiffResult contains the result of comparing two snapshots
type DiffResult struct {
	HasChanges       bool
	ContentAvailable bool
	Summary          string
	Details          []string
}

// DiffSnapshots compares before and after snapshots and returns a human-readable diff
func DiffSnapshots(before, after *Snapshot) *DiffResult {
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
		return diffContent(before.Content, after.Content, before.Size, after.Size)
	}

	// Hash-only diff
	return diffHashOnly(before, after)
}

// diffContent performs a content-based diff for JSON files
func diffContent(beforeContent, afterContent string, beforeSize, afterSize int64) *DiffResult {
	// Try to parse as JSON
	var beforeJSON, afterJSON map[string]interface{}
	beforeErr := json.Unmarshal([]byte(beforeContent), &beforeJSON)
	afterErr := json.Unmarshal([]byte(afterContent), &afterJSON)

	if beforeErr == nil && afterErr == nil {
		return diffJSON(beforeJSON, afterJSON, beforeSize, afterSize)
	}

	// Fallback to line-based diff for non-JSON content
	return diffLines(beforeContent, afterContent, beforeSize, afterSize)
}

// diffJSON compares two JSON objects and generates a summary of changes
func diffJSON(before, after map[string]interface{}, beforeSize, afterSize int64) *DiffResult {
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
			changes = append(changes, fmt.Sprintf("%s %s: %v", SymbolAdded, key, formatValue(afterVal)))
		} else if beforeExists && !afterExists {
			changes = append(changes, fmt.Sprintf("%s %s: %v", SymbolRemoved, key, formatValue(beforeVal)))
		} else if !reflect.DeepEqual(beforeVal, afterVal) {
			changes = append(changes, fmt.Sprintf("%s %s: %v → %v", SymbolModified, key, formatValue(beforeVal), formatValue(afterVal)))
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

// formatValue formats a JSON value for display
func formatValue(v interface{}) string {
	return formatValueWithDepth(v, 0)
}

// formatValueWithDepth formats a JSON value with depth and size limits
func formatValueWithDepth(v interface{}, depth int) string {
	const maxDepth = 3
	const maxArrayItems = 3

	if depth > maxDepth {
		return "..."
	}

	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		limit := len(val)
		if limit > maxArrayItems {
			limit = maxArrayItems
		}
		items := make([]string, limit)
		for i := 0; i < limit; i++ {
			items[i] = formatValueWithDepth(val[i], depth+1)
		}
		result := "[" + strings.Join(items, ", ")
		if len(val) > maxArrayItems {
			result += fmt.Sprintf(", ...%d more", len(val)-maxArrayItems)
		}
		return result + "]"
	case map[string]interface{}:
		return "{...}"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// truncateHash returns first 8 chars of a hash for display
func truncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}
