// ABOUTME: Tests for the event tracking system that records file operations
// ABOUTME: and modifications made by claudeup commands.
package events_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/internal/events"
)

func TestEvents(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Events Suite")
}

var _ = Describe("Tracker", func() {
	var (
		tracker   *events.Tracker
		writer    *fakeEventWriter
		tempDir   string
		testFile  string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "events-test-*")
		Expect(err).NotTo(HaveOccurred())

		testFile = filepath.Join(tempDir, "test.json")

		writer = &fakeEventWriter{events: make([]*events.FileOperation, 0)}
		tracker = events.NewTracker(writer, true)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("RecordFileWrite", func() {
		Context("when enabled", func() {
			It("records successful file creation", func() {
				err := tracker.RecordFileWrite("test-operation", testFile, "user", func() error {
					return os.WriteFile(testFile, []byte("test content"), 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(writer.events).To(HaveLen(1))

				event := writer.events[0]
				Expect(event.Operation).To(Equal("test-operation"))
				Expect(event.File).To(Equal(testFile))
				Expect(event.Scope).To(Equal("user"))
				Expect(event.ChangeType).To(Equal("create"))
				Expect(event.Error).To(BeNil())
			})

			It("records successful file update", func() {
				// Create initial file
				os.WriteFile(testFile, []byte("initial"), 0644)

				err := tracker.RecordFileWrite("test-operation", testFile, "project", func() error {
					return os.WriteFile(testFile, []byte("updated content"), 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(writer.events).To(HaveLen(1))

				event := writer.events[0]
				Expect(event.ChangeType).To(Equal("update"))
				Expect(event.Before).NotTo(BeNil())
				Expect(event.After).NotTo(BeNil())
				Expect(event.Before.Hash).NotTo(Equal(event.After.Hash))
			})

			It("records file deletion", func() {
				// Create initial file
				os.WriteFile(testFile, []byte("content"), 0644)

				err := tracker.RecordFileWrite("delete-operation", testFile, "user", func() error {
					return os.Remove(testFile)
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(writer.events).To(HaveLen(1))

				event := writer.events[0]
				Expect(event.ChangeType).To(Equal("delete"))
				Expect(event.Before).NotTo(BeNil())
				Expect(event.After).To(BeNil())
			})

			It("records operation errors", func() {
				expectedErr := errors.New("operation failed")

				err := tracker.RecordFileWrite("failing-operation", testFile, "user", func() error {
					return expectedErr
				})

				Expect(err).To(Equal(expectedErr))
				Expect(writer.events).To(HaveLen(1))

				event := writer.events[0]
				Expect(event.Error).To(Equal(expectedErr))
			})

			It("includes snapshots with file hash and size", func() {
				content := []byte("test content for hashing")

				err := tracker.RecordFileWrite("test-operation", testFile, "user", func() error {
					return os.WriteFile(testFile, content, 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				event := writer.events[0]

				Expect(event.After).NotTo(BeNil())
				Expect(event.After.Hash).NotTo(BeEmpty())
				Expect(event.After.Size).To(Equal(int64(len(content))))
			})
		})

		Context("when disabled", func() {
			BeforeEach(func() {
				tracker = events.NewTracker(writer, false)
			})

			It("executes operation without recording", func() {
				executed := false
				err := tracker.RecordFileWrite("test-operation", testFile, "user", func() error {
					executed = true
					return os.WriteFile(testFile, []byte("test"), 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(executed).To(BeTrue())
				Expect(writer.events).To(HaveLen(0))
			})
		})
	})
})

// fakeEventWriter for testing
type fakeEventWriter struct {
	events []*events.FileOperation
}

func (w *fakeEventWriter) Write(event *events.FileOperation) error {
	w.events = append(w.events, event)
	return nil
}

func (w *fakeEventWriter) Query(filters events.EventFilters) ([]*events.FileOperation, error) {
	return nil, nil
}
