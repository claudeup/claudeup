// ABOUTME: Project detection logic for profile suggestions
// ABOUTME: Matches project files and content against profile detect rules
package profile

import (
	"os"
	"path/filepath"
	"strings"
)

// Detect checks if a profile's detect rules match the given directory
// Files: ANY file existing is a match (OR-based within files)
// Contains: ANY pattern matching is a match (OR-based within contains)
// Overall: If both are specified, BOTH categories must have at least one match
func Detect(dir string, p *Profile) (bool, error) {
	rules := p.Detect

	// No rules means no match
	if len(rules.Files) == 0 && len(rules.Contains) == 0 {
		return false, nil
	}

	// Check if ANY of the files exist (OR-based)
	fileMatch := len(rules.Files) == 0 // If no files specified, consider it satisfied
	for _, file := range rules.Files {
		filePath := filepath.Join(dir, file)
		if _, err := os.Stat(filePath); err == nil {
			fileMatch = true
			break
		}
	}

	// Check if ANY file content pattern matches (OR-based)
	containsMatch := len(rules.Contains) == 0 // If no patterns specified, consider it satisfied
	for file, pattern := range rules.Contains {
		filePath := filepath.Join(dir, file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // File doesn't exist, try next pattern
		}

		if strings.Contains(string(content), pattern) {
			containsMatch = true
			break
		}
	}

	// Both conditions must be satisfied
	return fileMatch && containsMatch, nil
}

// FindMatchingProfiles returns all profiles that match the given directory
func FindMatchingProfiles(dir string, profiles []*Profile) []*Profile {
	var matches []*Profile

	for _, p := range profiles {
		match, err := Detect(dir, p)
		if err == nil && match {
			matches = append(matches, p)
		}
	}

	return matches
}

// SuggestProfile finds the best matching profile for a directory
// Returns nil if no profiles match
func SuggestProfile(dir string, profiles []*Profile) *Profile {
	matches := FindMatchingProfiles(dir, profiles)

	if len(matches) == 0 {
		return nil
	}

	// Return the first match (profiles should be ordered by priority)
	return matches[0]
}
