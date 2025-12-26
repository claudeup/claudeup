// ABOUTME: JSONL event writer that persists file operation events to disk
// ABOUTME: in a queryable format for audit trails and troubleshooting.
package events

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// JSONLWriter writes events to a JSONL (JSON Lines) file
type JSONLWriter struct {
	logPath string
	mu      sync.Mutex
}

// NewJSONLWriter creates a new JSONL event writer
func NewJSONLWriter(logPath string) (*JSONLWriter, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &JSONLWriter{
		logPath: logPath,
	}, nil
}

// Write appends an event to the log file
func (w *JSONLWriter) Write(event *FileOperation) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.OpenFile(w.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

// Query reads events from the log file and applies filters
func (w *JSONLWriter) Query(filters EventFilters) ([]*FileOperation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if log file exists
	if _, err := os.Stat(w.logPath); os.IsNotExist(err) {
		return []*FileOperation{}, nil
	}

	f, err := os.Open(w.logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []*FileOperation
	scanner := bufio.NewScanner(f)

	// Note: For large log files, this loads all matching events into memory
	// before sorting and limiting. A future optimization could use a bounded
	// priority queue to keep only the top N events during scanning.
	for scanner.Scan() {
		var event FileOperation
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			// Skip malformed lines
			continue
		}

		// Apply filters
		if !matchesFilters(&event, filters) {
			continue
		}

		events = append(events, &event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	// Apply limit after sorting to ensure we return the most recent events
	if filters.Limit > 0 && len(events) > filters.Limit {
		events = events[:filters.Limit]
	}

	return events, nil
}

// matchesFilters checks if an event matches the given filters
func matchesFilters(event *FileOperation, filters EventFilters) bool {
	// Filter by file
	if filters.File != "" && event.File != filters.File {
		return false
	}

	// Filter by operation
	if filters.Operation != "" && !strings.Contains(event.Operation, filters.Operation) {
		return false
	}

	// Filter by scope
	if filters.Scope != "" && event.Scope != filters.Scope {
		return false
	}

	// Filter by time
	if !filters.Since.IsZero() && event.Timestamp.Before(filters.Since) {
		return false
	}

	return true
}
