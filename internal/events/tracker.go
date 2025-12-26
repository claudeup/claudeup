// ABOUTME: Event tracking system that records file operations made by claudeup
// ABOUTME: commands, enabling audit trails and troubleshooting capabilities.
package events

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ChangeType constants for file operations
const (
	ChangeTypeCreate   = "create"
	ChangeTypeUpdate   = "update"
	ChangeTypeDelete   = "delete"
	ChangeTypeNoChange = "no-change"
	ChangeTypeUnknown  = "unknown"
)

// FileOperation represents a single file modification event
type FileOperation struct {
	Timestamp  time.Time              `json:"timestamp"`
	Operation  string                 `json:"operation"`  // "profile apply", "plugin install", etc.
	File       string                 `json:"file"`       // Absolute path
	Scope      string                 `json:"scope"`      // user/project/local
	ChangeType string                 `json:"changeType"` // create/update/delete
	Before     *Snapshot              `json:"before,omitempty"`
	After      *Snapshot              `json:"after,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// Snapshot represents the state of a file at a point in time
type Snapshot struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// EventWriter writes and queries file operation events
type EventWriter interface {
	Write(event *FileOperation) error
	Query(filters EventFilters) ([]*FileOperation, error)
}

// EventFilters for querying events
type EventFilters struct {
	File      string
	Operation string
	Since     time.Time
	Scope     string
	Limit     int
}

// Tracker records file operations
type Tracker struct {
	enabled bool // Exported via Enable/Disable methods
	writer  EventWriter
}

// SetEnabled enables or disables the tracker
func (t *Tracker) SetEnabled(enabled bool) {
	t.enabled = enabled
}

// IsEnabled returns whether the tracker is enabled
func (t *Tracker) IsEnabled() bool {
	return t.enabled
}

// NewTracker creates a new event tracker
func NewTracker(writer EventWriter, enabled bool) *Tracker {
	return &Tracker{
		enabled: enabled,
		writer:  writer,
	}
}

// errToString converts an error to a string, returning empty string for nil
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// RecordFileWrite wraps a file write operation with event tracking
func (t *Tracker) RecordFileWrite(operation string, file string, scope string, fn func() error) error {
	if !t.enabled {
		return fn()
	}

	// Sanitize file path to prevent injection attacks
	file = filepath.Clean(file)

	// Snapshot before
	before := t.snapshot(file)

	// Execute operation
	err := fn()

	// Snapshot after
	after := t.snapshot(file)

	// Determine change type
	changeType := inferChangeType(before, after)

	// Record event
	event := &FileOperation{
		Timestamp:  time.Now(),
		Operation:  operation,
		File:       file,
		Scope:      scope,
		ChangeType: changeType,
		Before:     before,
		After:      after,
		Error:      errToString(err),
	}

	// Write event (don't fail the operation if event writing fails)
	if t.writer != nil {
		_ = t.writer.Write(event)
	}

	return err
}

// snapshot creates a snapshot of a file's current state
func (t *Tracker) snapshot(path string) *Snapshot {
	info, err := os.Stat(path)
	if err != nil {
		// File doesn't exist or can't be read
		return nil
	}

	hash, err := hashFile(path)
	if err != nil {
		// Can't hash file, return basic info
		return &Snapshot{
			Hash: "",
			Size: info.Size(),
		}
	}

	return &Snapshot{
		Hash: hash,
		Size: info.Size(),
	}
}

// hashFile computes SHA-256 hash of a file
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// inferChangeType determines the type of change based on before/after snapshots
func inferChangeType(before, after *Snapshot) string {
	if before == nil && after != nil {
		return ChangeTypeCreate
	}
	if before != nil && after == nil {
		return ChangeTypeDelete
	}
	if before != nil && after != nil {
		// Detect changes via hash or size difference
		if before.Hash != after.Hash || before.Size != after.Size {
			return ChangeTypeUpdate
		}
		return ChangeTypeNoChange
	}
	return ChangeTypeUnknown
}
