// ABOUTME: Profile bootstrap functionality for sandbox containers.
// ABOUTME: Applies profile's Claude configuration on first sandbox run.
package sandbox

import (
	"os"
	"path/filepath"
	"time"
)

const sentinelFile = ".bootstrapped"

// IsFirstRun returns true if the sandbox state directory has not been bootstrapped.
func IsFirstRun(stateDir string) bool {
	sentinel := filepath.Join(stateDir, sentinelFile)
	_, err := os.Stat(sentinel)
	return os.IsNotExist(err)
}

// WriteSentinel marks the sandbox as bootstrapped.
func WriteSentinel(stateDir string) error {
	sentinel := filepath.Join(stateDir, sentinelFile)
	timestamp := time.Now().Format(time.RFC3339)
	return os.WriteFile(sentinel, []byte(timestamp), 0644)
}
