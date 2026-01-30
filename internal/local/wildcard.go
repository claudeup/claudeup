// ABOUTME: Wildcard pattern matching for item selection
// ABOUTME: Supports prefix (gsd-*), directory (gsd/*), and global (*) wildcards
package local

import (
	"sort"
	"strings"
)

// MatchWildcard returns items matching the given pattern.
// Patterns:
//   - "gsd-*" matches items starting with "gsd-"
//   - "gsd/*" matches items in the "gsd/" directory
//   - "*" matches all items
//   - exact string matches that specific item
func MatchWildcard(pattern string, items []string) []string {
	var matched []string

	// Global wildcard
	if pattern == "*" {
		result := make([]string, len(items))
		copy(result, items)
		sort.Strings(result)
		return result
	}

	// Directory wildcard: "group/*"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "*")
		for _, item := range items {
			if strings.HasPrefix(item, prefix) {
				matched = append(matched, item)
			}
		}
		sort.Strings(matched)
		return matched
	}

	// Prefix wildcard: "prefix*"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		for _, item := range items {
			// For prefix matching, only match the base name (not path)
			baseName := item
			if idx := strings.LastIndex(item, "/"); idx >= 0 {
				baseName = item[idx+1:]
			}
			if strings.HasPrefix(baseName, prefix) {
				matched = append(matched, item)
			}
		}
		sort.Strings(matched)
		return matched
	}

	// Exact match
	for _, item := range items {
		if item == pattern {
			return []string{item}
		}
	}

	return []string{}
}
