// ABOUTME: Credential type definitions and resolution for sandbox containers.
// ABOUTME: Maps credential names (git, ssh, gh) to host/container paths.
package sandbox

import (
	"os"
	"path/filepath"
)

// CredentialType defines a mountable credential
type CredentialType struct {
	Name         string // "git", "ssh", "gh"
	SourceSuffix string // Path suffix from home dir (e.g., ".gitconfig")
	TargetPath   string // Container path
}

var credentialTypes = map[string]*CredentialType{
	"git": {
		Name:         "git",
		SourceSuffix: ".gitconfig",
		TargetPath:   "/root/.gitconfig",
	},
	"ssh": {
		Name:         "ssh",
		SourceSuffix: ".ssh",
		TargetPath:   "/root/.ssh",
	},
	"gh": {
		Name:         "gh",
		SourceSuffix: ".config/gh",
		TargetPath:   "/root/.config/gh",
	},
}

// GetCredentialType returns the credential type definition, or nil if unknown
func GetCredentialType(name string) *CredentialType {
	return credentialTypes[name]
}

// ValidCredentialTypes returns all valid credential type names
func ValidCredentialTypes() []string {
	return []string{"git", "ssh", "gh"}
}

// MergeCredentials combines profile credentials with CLI overrides.
// Order: start with profile, add additions, remove exclusions.
// Unknown credential types are silently ignored.
func MergeCredentials(profile, add, exclude []string) []string {
	// Build set from profile
	set := make(map[string]bool)
	for _, c := range profile {
		if GetCredentialType(c) != nil {
			set[c] = true
		}
	}

	// Add CLI additions
	for _, c := range add {
		if GetCredentialType(c) != nil {
			set[c] = true
		}
	}

	// Remove CLI exclusions
	for _, c := range exclude {
		delete(set, c)
	}

	// Convert to sorted slice for deterministic output
	result := make([]string, 0, len(set))
	for _, validType := range ValidCredentialTypes() {
		if set[validType] {
			result = append(result, validType)
		}
	}

	return result
}

// ResolveCredentialMounts converts credential names to Docker mounts.
// Returns mounts and any warnings for missing credentials.
// stateDir is used for credentials that need extraction (gh on macOS).
func ResolveCredentialMounts(credentials []string, homeDir, stateDir string) ([]Mount, []string) {
	var mounts []Mount
	var warnings []string

	for _, name := range credentials {
		credType := GetCredentialType(name)
		if credType == nil {
			continue // Unknown type, already filtered by MergeCredentials
		}

		sourcePath := filepath.Join(homeDir, credType.SourceSuffix)

		// Check if source exists
		info, err := os.Stat(sourcePath)
		if os.IsNotExist(err) {
			warnings = append(warnings, "credential "+name+" not found at "+sourcePath)
			continue
		}

		// Warn about SSH directory permissions
		if credType.Name == "ssh" && info.IsDir() {
			perm := info.Mode().Perm()
			if perm != 0700 {
				warnings = append(warnings, "ssh directory has permissions "+perm.String()+", should be 0700; SSH may fail in container")
			}
		}

		mounts = append(mounts, Mount{
			Host:      sourcePath,
			Container: credType.TargetPath,
			ReadOnly:  true,
		})
	}

	return mounts, warnings
}
