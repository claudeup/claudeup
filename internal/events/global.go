// ABOUTME: Global event tracker instance for application-wide event tracking
// ABOUTME: Provides centralized access to file operation monitoring.
package events

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	globalTracker     *Tracker
	globalTrackerOnce sync.Once
	globalTrackerMu   sync.RWMutex
)

// GlobalTracker returns the global event tracker instance
// Creates and initializes it on first access
func GlobalTracker() *Tracker {
	globalTrackerOnce.Do(func() {
		globalTracker = initializeGlobalTracker()
	})

	globalTrackerMu.RLock()
	defer globalTrackerMu.RUnlock()
	return globalTracker
}

// SetGlobalTracker sets a custom global tracker (useful for testing)
func SetGlobalTracker(tracker *Tracker) {
	globalTrackerMu.Lock()
	defer globalTrackerMu.Unlock()
	globalTracker = tracker
}

// initializeGlobalTracker creates the default global tracker
func initializeGlobalTracker() *Tracker {
	homeDir, _ := os.UserHomeDir()
	eventsDir := filepath.Join(homeDir, ".claudeup", "events")
	logPath := filepath.Join(eventsDir, "operations.log")

	writer, err := NewJSONLWriter(logPath)
	if err != nil {
		// If we can't create the writer, return a disabled tracker
		return NewTracker(nil, false)
	}

	// Enabled by default - can be disabled via config later
	return NewTracker(writer, true)
}

// DisableGlobalTracking disables the global event tracker
func DisableGlobalTracking() {
	globalTrackerMu.Lock()
	defer globalTrackerMu.Unlock()
	if globalTracker != nil {
		globalTracker.SetEnabled(false)
	}
}

// EnableGlobalTracking enables the global event tracker
func EnableGlobalTracking() {
	globalTrackerMu.Lock()
	defer globalTrackerMu.Unlock()
	if globalTracker != nil {
		globalTracker.SetEnabled(true)
	}
}
