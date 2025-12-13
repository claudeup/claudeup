// ABOUTME: Embedded default profiles for claudeup
// ABOUTME: Uses Go embed to bundle profiles in the binary
package profile

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed profiles/*.json
var embeddedProfiles embed.FS

//go:embed all:profiles/scripts
var embeddedScripts embed.FS

// EnsureDefaultProfiles extracts embedded profiles to the profiles directory
// if they don't already exist
func EnsureDefaultProfiles(profilesDir string) error {
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return err
	}

	entries, err := embeddedProfiles.ReadDir("profiles")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		destPath := filepath.Join(profilesDir, name)

		// Skip if file already exists
		if _, err := os.Stat(destPath); err == nil {
			continue
		}

		// Read embedded file
		data, err := embeddedProfiles.ReadFile("profiles/" + name)
		if err != nil {
			return err
		}

		// Write to profiles directory
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

// GetEmbeddedProfile returns an embedded profile by name
func GetEmbeddedProfile(name string) (*Profile, error) {
	data, err := embeddedProfiles.ReadFile("profiles/" + name + ".json")
	if err != nil {
		return nil, err
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// ListEmbeddedProfiles returns all embedded profiles
func ListEmbeddedProfiles() ([]*Profile, error) {
	entries, err := embeddedProfiles.ReadDir("profiles")
	if err != nil {
		return nil, err
	}

	var profiles []*Profile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		profileName := strings.TrimSuffix(name, ".json")
		p, err := GetEmbeddedProfile(profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping invalid embedded profile %q: %v\n", profileName, err)
			continue
		}
		profiles = append(profiles, p)
	}

	return profiles, nil
}

// IsEmbeddedProfile checks if a profile with the given name exists in embedded profiles
func IsEmbeddedProfile(name string) bool {
	_, err := embeddedProfiles.ReadFile("profiles/" + name + ".json")
	return err == nil
}

// GetEmbeddedProfileScriptDir extracts embedded scripts for a profile to a temp directory
// Returns empty string if no scripts exist or extraction fails
func GetEmbeddedProfileScriptDir(profileName string) string {
	// Check if scripts directory exists for this profile
	scriptDir := "profiles/scripts/" + profileName
	entries, err := embeddedScripts.ReadDir(scriptDir)
	if err != nil || len(entries) == 0 {
		return ""
	}

	// Create temp directory for scripts
	tempDir, err := os.MkdirTemp("", "claudeup-scripts-"+profileName+"-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create temp directory for profile scripts: %v\n", err)
		return ""
	}

	// Extract scripts, tracking success
	extractedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := embeddedScripts.ReadFile(scriptDir + "/" + entry.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read embedded script %s: %v\n", entry.Name(), err)
			continue
		}

		destPath := filepath.Join(tempDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to extract script %s: %v\n", entry.Name(), err)
			continue
		}
		extractedCount++
	}

	// If no scripts were extracted, clean up and return empty
	if extractedCount == 0 {
		os.RemoveAll(tempDir)
		return ""
	}

	return tempDir
}
