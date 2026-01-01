// ABOUTME: Semantic version comparison utilities
// ABOUTME: Determines if a remote version is newer than local
package selfupdate

import (
	"strconv"
	"strings"
)

// IsNewer returns true if remoteVersion is newer than localVersion
func IsNewer(localVersion, remoteVersion string) bool {
	// Dev version is always outdated
	if localVersion == "dev" || localVersion == "(devel)" {
		return true
	}

	local := parseVersion(localVersion)
	remote := parseVersion(remoteVersion)

	// Compare major.minor.patch
	if remote[0] > local[0] {
		return true
	}
	if remote[0] < local[0] {
		return false
	}

	if remote[1] > local[1] {
		return true
	}
	if remote[1] < local[1] {
		return false
	}

	return remote[2] > local[2]
}

// parseVersion extracts major, minor, patch from version string
func parseVersion(v string) [3]int {
	// Strip 'v' prefix if present
	v = strings.TrimPrefix(v, "v")

	parts := strings.Split(v, ".")
	var result [3]int

	for i := 0; i < 3 && i < len(parts); i++ {
		// Remove any suffix after number (e.g., "1-beta")
		numStr := strings.Split(parts[i], "-")[0]
		num, _ := strconv.Atoi(numStr)
		result[i] = num
	}

	return result
}
