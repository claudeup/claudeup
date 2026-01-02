// ABOUTME: Credential type definitions and resolution for sandbox containers.
// ABOUTME: Maps credential names (git, ssh, gh) to host/container paths.
package sandbox

// CredentialType defines a mountable credential
type CredentialType struct {
	Name         string // "git", "ssh", "gh"
	SourceSuffix string // Path suffix from home dir (e.g., ".gitconfig")
	TargetPath   string // Container path
	NeedsExtract bool   // True if credential needs Keychain extraction (macOS)
}

var credentialTypes = map[string]*CredentialType{
	"git": {
		Name:         "git",
		SourceSuffix: ".gitconfig",
		TargetPath:   "/root/.gitconfig",
		NeedsExtract: false,
	},
	"ssh": {
		Name:         "ssh",
		SourceSuffix: ".ssh",
		TargetPath:   "/root/.ssh",
		NeedsExtract: false,
	},
	"gh": {
		Name:         "gh",
		SourceSuffix: ".config/gh",
		TargetPath:   "/root/.config/gh",
		NeedsExtract: true, // macOS stores in Keychain
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
