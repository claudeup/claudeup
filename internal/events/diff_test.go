// ABOUTME: Tests for the diff functionality that compares file snapshots
// ABOUTME: and generates human-readable output showing what changed.
package events_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/internal/events"
)

var _ = Describe("DiffSnapshots", func() {
	Context("JSON content diffs", func() {
		It("shows added fields", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"version": 1}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    150,
				Content: `{"version": 1, "newField": "value"}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.ContentAvailable).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("+ newField"))
		})

		It("shows removed fields", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    150,
				Content: `{"version": 1, "oldField": "value"}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    100,
				Content: `{"version": 1}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.ContentAvailable).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("- oldField"))
		})

		It("shows modified values", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"version": 1}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    100,
				Content: `{"version": 2}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.ContentAvailable).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ version"))
		})

		It("handles complex nested changes", func() {
			before := &events.Snapshot{
				Hash: "abc123",
				Size: 200,
				Content: `{
					"plugins": {
						"enabled": ["plugin1"]
					}
				}`,
			}
			after := &events.Snapshot{
				Hash: "def456",
				Size: 250,
				Content: `{
					"plugins": {
						"enabled": ["plugin1", "plugin2"]
					}
				}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.ContentAvailable).To(BeTrue())
			Expect(result.Summary).NotTo(BeEmpty())
		})
	})

	Context("hash-only diffs", func() {
		It("shows hash change when content not available", func() {
			before := &events.Snapshot{
				Hash: "abc123",
				Size: 1000,
			}
			after := &events.Snapshot{
				Hash: "def456",
				Size: 1500,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.ContentAvailable).To(BeFalse())
			Expect(result.Summary).To(ContainSubstring("abc123"))
			Expect(result.Summary).To(ContainSubstring("def456"))
		})

		It("shows size change in hash-only mode", func() {
			before := &events.Snapshot{
				Hash: "abc123",
				Size: 1000,
			}
			after := &events.Snapshot{
				Hash: "def456",
				Size: 1500,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.Summary).To(ContainSubstring("+500"))
		})
	})

	Context("file creation", func() {
		It("shows file creation", func() {
			before := (*events.Snapshot)(nil)
			after := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"version": 1}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("created"))
		})
	})

	Context("file deletion", func() {
		It("shows file deletion", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"version": 1}`,
			}
			after := (*events.Snapshot)(nil)

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("deleted"))
		})
	})

	Context("no changes", func() {
		It("reports no changes when hashes match", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"version": 1}`,
			}
			after := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"version": 1}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeFalse())
		})
	})

	Context("array truncation and depth limits", func() {
		It("truncates large arrays to prevent overflow", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"items": [1, 2, 3]}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    200,
				Content: `{"items": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ items"))
			Expect(result.Summary).To(ContainSubstring("...7 more"))
		})

		It("limits depth for deeply nested arrays", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"deep": []}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    200,
				Content: `{"deep": [[[[["too deep"]]]]]}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ deep"))
			Expect(result.Summary).To(ContainSubstring("..."))
		})

		It("handles empty arrays correctly", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"items": [1, 2, 3]}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    50,
				Content: `{"items": []}`,
			}

			result := events.DiffSnapshots(before, after)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ items"))
			Expect(result.Summary).To(ContainSubstring("[]"))
		})
	})
})
