// ABOUTME: Tests for the event tracking system that records file operations
// ABOUTME: and modifications made by claudeup commands.
package events_test

import (
	"encoding/json"
	"errors"
	"fmt"
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
				Expect(event.ChangeType).To(Equal(events.ChangeTypeCreate))
				Expect(event.Error).To(BeEmpty())
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
				Expect(event.ChangeType).To(Equal(events.ChangeTypeUpdate))
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
				Expect(event.ChangeType).To(Equal(events.ChangeTypeDelete))
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
				Expect(event.Error).To(Equal("operation failed"))
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

			It("marshals error messages to JSON correctly", func() {
				expectedErr := errors.New("permission denied")

				err := tracker.RecordFileWrite("failing-operation", testFile, "user", func() error {
					return expectedErr
				})

				Expect(err).To(Equal(expectedErr))
				Expect(writer.events).To(HaveLen(1))

				// Marshal event to JSON and verify error is preserved
				event := writer.events[0]
				jsonData, err := json.Marshal(event)
				Expect(err).NotTo(HaveOccurred())

				// Unmarshal and verify error message is present
				var unmarshaled map[string]interface{}
				err = json.Unmarshal(jsonData, &unmarshaled)
				Expect(err).NotTo(HaveOccurred())
				Expect(unmarshaled["error"]).To(Equal("permission denied"))
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

		Context("concurrent operations", func() {
			It("handles concurrent RecordFileWrite calls safely", func() {
				const numGoroutines = 10
				done := make(chan bool, numGoroutines)

				for i := 0; i < numGoroutines; i++ {
					go func(id int) {
						defer GinkgoRecover()
						testFile := filepath.Join(tempDir, fmt.Sprintf("concurrent-%d.txt", id))
						err := tracker.RecordFileWrite(
							"concurrent-test",
							testFile,
							"user",
							func() error {
								return os.WriteFile(testFile, []byte(fmt.Sprintf("test-%d", id)), 0644)
							},
						)
						Expect(err).NotTo(HaveOccurred())
						done <- true
					}(i)
				}

				// Wait for all goroutines to complete
				for i := 0; i < numGoroutines; i++ {
					<-done
				}

				// Verify all events were recorded
				Expect(writer.events).To(HaveLen(numGoroutines))
			})
		})

		Context("edge cases", func() {
			It("handles path injection attempts safely", func() {
				maliciousPath := filepath.Join(tempDir, "../../../etc/passwd")
				err := tracker.RecordFileWrite(
					"injection-test",
					maliciousPath,
					"user",
					func() error {
						// Path should be cleaned before snapshot
						return nil
					},
				)

				Expect(err).NotTo(HaveOccurred())
				Expect(writer.events).To(HaveLen(1))
				// Event should have cleaned path
				Expect(writer.events[0].File).NotTo(ContainSubstring(".."))
			})

			It("handles deleted files gracefully", func() {
				// Create file first
				os.WriteFile(testFile, []byte("content"), 0644)

				err := tracker.RecordFileWrite(
					"delete-test",
					testFile,
					"user",
					func() error {
						// Delete file during operation
						return os.Remove(testFile)
					},
				)

				Expect(err).NotTo(HaveOccurred())
				Expect(writer.events).To(HaveLen(1))
				Expect(writer.events[0].ChangeType).To(Equal(events.ChangeTypeDelete))
			})
		})

		Context("content capture for diffing", func() {
			It("captures content for JSON files under 1MB", func() {
				jsonContent := `{"key": "value", "number": 42}`

				err := tracker.RecordFileWrite("test-operation", testFile, "user", func() error {
					return os.WriteFile(testFile, []byte(jsonContent), 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				event := writer.events[0]

				Expect(event.After).NotTo(BeNil())
				Expect(event.After.Content).To(Equal(jsonContent))
			})

			It("does not capture content for JSON files over 1MB", func() {
				// Create a JSON file > 1MB
				largeJSON := make([]byte, 1024*1024+1) // 1MB + 1 byte
				for i := range largeJSON {
					largeJSON[i] = 'a'
				}

				err := tracker.RecordFileWrite("test-operation", testFile, "user", func() error {
					return os.WriteFile(testFile, largeJSON, 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				event := writer.events[0]

				Expect(event.After).NotTo(BeNil())
				Expect(event.After.Hash).NotTo(BeEmpty())
				Expect(event.After.Content).To(BeEmpty())
			})

			It("does not capture content for non-JSON files", func() {
				txtFile := filepath.Join(tempDir, "test.txt")
				content := "plain text content"

				err := tracker.RecordFileWrite("test-operation", txtFile, "user", func() error {
					return os.WriteFile(txtFile, []byte(content), 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				event := writer.events[0]

				Expect(event.After).NotTo(BeNil())
				Expect(event.After.Hash).NotTo(BeEmpty())
				Expect(event.After.Content).To(BeEmpty())
			})

			It("captures before and after content for updates", func() {
				beforeJSON := `{"version": 1}`
				afterJSON := `{"version": 2}`

				// Create initial file
				os.WriteFile(testFile, []byte(beforeJSON), 0644)

				err := tracker.RecordFileWrite("test-operation", testFile, "user", func() error {
					return os.WriteFile(testFile, []byte(afterJSON), 0644)
				})

				Expect(err).NotTo(HaveOccurred())
				event := writer.events[0]

				Expect(event.Before).NotTo(BeNil())
				Expect(event.Before.Content).To(Equal(beforeJSON))
				Expect(event.After).NotTo(BeNil())
				Expect(event.After.Content).To(Equal(afterJSON))
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
