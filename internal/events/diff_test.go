// ABOUTME: Tests for the diff functionality that compares file snapshots
// ABOUTME: and generates human-readable output showing what changed.
package events_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v4/internal/events"
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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

			Expect(result.HasChanges).To(BeFalse())
		})
	})

	Context("malformed JSON", func() {
		It("handles malformed JSON gracefully with fallback to line-based diff", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"valid": "json"}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    120,
				Content: `{invalid json syntax`,
			}

			result := events.DiffSnapshots(before, after, false)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.ContentAvailable).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("Content changed"))
		})

		It("handles both before and after malformed JSON", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{not valid json}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    120,
				Content: `{also not valid`,
			}

			result := events.DiffSnapshots(before, after, false)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.ContentAvailable).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("Content changed"))
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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

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

			result := events.DiffSnapshots(before, after, false)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ items"))
			Expect(result.Summary).To(ContainSubstring("[]"))
		})

		It("truncates nested objects by default", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"plugins": {"name": "plugin1", "version": "1.0"}}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    150,
				Content: `{"plugins": {"name": "plugin2", "version": "2.0"}}`,
			}

			result := events.DiffSnapshots(before, after, false)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ plugins"))
			Expect(result.Summary).To(ContainSubstring("{...}"))
		})
	})

	Context("full diff mode", func() {
		It("shows deep diff for nested objects when full=true", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"plugins": {"name": "plugin1", "version": "1.0"}}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    150,
				Content: `{"plugins": {"name": "plugin2", "version": "2.0"}}`,
			}

			result := events.DiffSnapshots(before, after, true)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ plugins:"))
			Expect(result.Summary).NotTo(ContainSubstring("{...}"))
			Expect(result.Summary).To(ContainSubstring("~ name:"))
			Expect(result.Summary).To(ContainSubstring("~ version:"))
			Expect(result.Summary).To(ContainSubstring("\"plugin1\" → \"plugin2\""))
			Expect(result.Summary).To(ContainSubstring("\"1.0\" → \"2.0\""))
		})

		It("shows added and removed keys in nested objects", func() {
			before := &events.Snapshot{
				Hash:    "abc123",
				Size:    100,
				Content: `{"config": {"oldKey": "value1", "shared": "same"}}`,
			}
			after := &events.Snapshot{
				Hash:    "def456",
				Size:    150,
				Content: `{"config": {"newKey": "value2", "shared": "same"}}`,
			}

			result := events.DiffSnapshots(before, after, true)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ config:"))
			Expect(result.Summary).To(ContainSubstring("- oldKey:"))
			Expect(result.Summary).To(ContainSubstring("+ newKey:"))
			Expect(result.Summary).NotTo(ContainSubstring("~ shared:"))
		})

		It("shows all array items when full=true", func() {
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

			result := events.DiffSnapshots(before, after, true)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).NotTo(ContainSubstring("...7 more"))
			Expect(result.Summary).To(ContainSubstring("10"))
		})

		It("handles deeply nested object changes", func() {
			before := &events.Snapshot{
				Hash: "abc123",
				Size: 200,
				Content: `{
					"settings": {
						"user": {
							"preferences": {
								"theme": "light"
							}
						}
					}
				}`,
			}
			after := &events.Snapshot{
				Hash: "def456",
				Size: 200,
				Content: `{
					"settings": {
						"user": {
							"preferences": {
								"theme": "dark"
							}
						}
					}
				}`,
			}

			result := events.DiffSnapshots(before, after, true)

			Expect(result.HasChanges).To(BeTrue())
			Expect(result.Summary).To(ContainSubstring("~ settings:"))
			Expect(result.Summary).To(ContainSubstring("~ user:"))
			Expect(result.Summary).To(ContainSubstring("~ preferences:"))
			Expect(result.Summary).To(ContainSubstring("~ theme:"))
			Expect(result.Summary).To(ContainSubstring("\"light\" → \"dark\""))
		})
	})
})
