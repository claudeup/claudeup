// ABOUTME: Global event tracker instance for application-wide event tracking
// ABOUTME: Provides centralized access to file operation monitoring.
package events

import (
	"path/filepath"
	"sync"

	"github.com/claudeup/claudeup/v4/internal/config"
)

var (
	globalTracker     *Tracker
	globalTrackerOnce sync.Once
)

// GlobalTracker returns the global event tracker instance
// Creates and initializes it on first access
func GlobalTracker() *Tracker {
	globalTrackerOnce.Do(func() {
		globalTracker = initializeGlobalTracker()
	})
	return globalTracker // sync.Once provides sufficient synchronization
}

// initializeGlobalTracker creates the default global tracker
func initializeGlobalTracker() *Tracker {
	eventsDir := filepath.Join(config.MustClaudeupHome(), "events")
	logPath := filepath.Join(eventsDir, "operations.log")

	writer, err := NewJSONLWriter(logPath)
	if err != nil {
		// If we can't create the writer, return a disabled tracker
		return NewTracker(nil, false)
	}

	// Enabled by default - can be disabled via config later
	return NewTracker(writer, true)
}
