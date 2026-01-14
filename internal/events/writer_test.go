// ABOUTME: Tests for JSONL event writer that persists file operation events
// ABOUTME: to disk in a queryable format.
package events_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v2/internal/events"
)

var _ = Describe("JSONLWriter", func() {
	var (
		writer  *events.JSONLWriter
		tempDir string
		logPath string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "writer-test-*")
		Expect(err).NotTo(HaveOccurred())

		logPath = filepath.Join(tempDir, "events.log")
		writer, err = events.NewJSONLWriter(logPath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("Write", func() {
		It("writes events to JSONL file", func() {
			event := &events.FileOperation{
				Timestamp:  time.Now(),
				Operation:  "test-operation",
				File:       "/path/to/file.json",
				Scope:      "user",
				ChangeType: "create",
			}

			err := writer.Write(event)
			Expect(err).NotTo(HaveOccurred())

			// Verify file exists
			_, err = os.Stat(logPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("appends multiple events", func() {
			event1 := &events.FileOperation{
				Timestamp:  time.Now(),
				Operation:  "operation-1",
				File:       "/file1.json",
				Scope:      "user",
				ChangeType: "create",
			}

			event2 := &events.FileOperation{
				Timestamp:  time.Now(),
				Operation:  "operation-2",
				File:       "/file2.json",
				Scope:      "project",
				ChangeType: "update",
			}

			err := writer.Write(event1)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Write(event2)
			Expect(err).NotTo(HaveOccurred())

			// Query all events
			allEvents, err := writer.Query(events.EventFilters{})
			Expect(err).NotTo(HaveOccurred())
			Expect(allEvents).To(HaveLen(2))
		})

		It("creates parent directories if needed", func() {
			deepPath := filepath.Join(tempDir, "nested", "dir", "events.log")
			deepWriter, err := events.NewJSONLWriter(deepPath)
			Expect(err).NotTo(HaveOccurred())

			event := &events.FileOperation{
				Timestamp:  time.Now(),
				Operation:  "test",
				File:       "/file.json",
				Scope:      "user",
				ChangeType: "create",
			}

			err = deepWriter.Write(event)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(deepPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Query", func() {
		var (
			event1, event2, event3 *events.FileOperation
			timestamp1, timestamp2 time.Time
		)

		BeforeEach(func() {
			timestamp1 = time.Now().Add(-2 * time.Hour)
			timestamp2 = time.Now().Add(-1 * time.Hour)

			event1 = &events.FileOperation{
				Timestamp:  timestamp1,
				Operation:  "profile apply",
				File:       "/path/to/settings.json",
				Scope:      "user",
				ChangeType: "update",
			}

			event2 = &events.FileOperation{
				Timestamp:  timestamp2,
				Operation:  "plugin install",
				File:       "/path/to/plugins.json",
				Scope:      "project",
				ChangeType: "update",
			}

			event3 = &events.FileOperation{
				Timestamp:  time.Now(),
				Operation:  "profile apply",
				File:       "/path/to/settings.json",
				Scope:      "user",
				ChangeType: "update",
			}

			writer.Write(event1)
			writer.Write(event2)
			writer.Write(event3)
		})

		It("returns all events when no filters", func() {
			events, err := writer.Query(events.EventFilters{})
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(3))
		})

		It("filters by file path", func() {
			evts, err := writer.Query(events.EventFilters{
				File: "/path/to/settings.json",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(2))
			Expect(evts[0].File).To(Equal("/path/to/settings.json"))
			Expect(evts[1].File).To(Equal("/path/to/settings.json"))
		})

		It("filters by operation", func() {
			evts, err := writer.Query(events.EventFilters{
				Operation: "profile apply",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(2))
		})

		It("filters by scope", func() {
			evts, err := writer.Query(events.EventFilters{
				Scope: "project",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(1))
			Expect(evts[0].Scope).To(Equal("project"))
		})

		It("filters by time range", func() {
			evts, err := writer.Query(events.EventFilters{
				Since: timestamp2,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(2)) // event2 and event3
		})

		It("limits results", func() {
			evts, err := writer.Query(events.EventFilters{
				Limit: 2,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(2))
		})

		It("combines multiple filters", func() {
			evts, err := writer.Query(events.EventFilters{
				Operation: "profile apply",
				Scope:     "user",
				Limit:     1,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(1))
		})

		It("returns events in reverse chronological order", func() {
			evts, err := writer.Query(events.EventFilters{})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(3))

			// Most recent first
			Expect(evts[0].Timestamp.After(evts[1].Timestamp)).To(BeTrue())
			Expect(evts[1].Timestamp.After(evts[2].Timestamp)).To(BeTrue())
		})

		It("handles malformed JSONL gracefully", func() {
			// Create a fresh log file for this test
			malformedLogPath := filepath.Join(tempDir, "malformed-test.log")
			malformedWriter, err := events.NewJSONLWriter(malformedLogPath)
			Expect(err).NotTo(HaveOccurred())

			// Write a valid event
			event1 := &events.FileOperation{
				Timestamp:  time.Now(),
				Operation:  "test",
				File:       "/file1.json",
				Scope:      "user",
				ChangeType: "create",
			}
			malformedWriter.Write(event1)

			// Manually append malformed JSON to log
			f, err := os.OpenFile(malformedLogPath, os.O_APPEND|os.O_WRONLY, 0600)
			Expect(err).NotTo(HaveOccurred())
			f.WriteString("{invalid json}\n")
			f.WriteString("not json at all\n")
			f.Close()

			// Write another valid event
			event2 := &events.FileOperation{
				Timestamp:  time.Now(),
				Operation:  "test2",
				File:       "/file2.json",
				Scope:      "user",
				ChangeType: "update",
			}
			malformedWriter.Write(event2)

			// Query should skip malformed lines and return only valid events
			evts, err := malformedWriter.Query(events.EventFilters{})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(HaveLen(2))
			Expect(evts[0].Operation).To(Equal("test2"))
			Expect(evts[1].Operation).To(Equal("test"))
		})

		It("returns empty slice for non-existent log file", func() {
			emptyPath := filepath.Join(tempDir, "nonexistent.log")
			emptyWriter, err := events.NewJSONLWriter(emptyPath)
			Expect(err).NotTo(HaveOccurred())

			evts, err := emptyWriter.Query(events.EventFilters{})
			Expect(err).NotTo(HaveOccurred())
			Expect(evts).To(BeEmpty())
		})
	})
})
