// ABOUTME: Acceptance tests for scope alias flags (--user, --project, --local)
// ABOUTME: Tests the shorthand aliases work correctly and conflict detection

package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v3/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scope Alias Flags", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("Conflict Detection", func() {
		It("rejects --scope with --user", func() {
			result := env.Run("profile", "apply", "default", "--scope", "user", "--user")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})

		It("rejects --scope with --project", func() {
			result := env.Run("profile", "apply", "default", "--scope", "project", "--project")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})

		It("rejects --user with --project", func() {
			result := env.Run("profile", "apply", "default", "--user", "--project")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})

		It("rejects --project with --local", func() {
			result := env.Run("profile", "apply", "default", "--project", "--local")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})

		It("rejects all three aliases together", func() {
			result := env.Run("profile", "apply", "default", "--user", "--project", "--local")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})
	})

	Describe("profile apply", func() {
		It("accepts --user flag", func() {
			result := env.Run("profile", "apply", "default", "--user", "--dry-run")

			// Should succeed (or show dry-run output)
			// The flag is accepted - that's what we're testing
			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --project flag", func() {
			result := env.Run("profile", "apply", "default", "--project", "--dry-run")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --local flag", func() {
			result := env.Run("profile", "apply", "default", "--local", "--dry-run")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})
	})

	Describe("profile list", func() {
		It("accepts --user flag", func() {
			result := env.Run("profile", "list", "--user")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --project flag in project directory", func() {
			// Create the .claude directory and project settings file in TempDir
			projectClaudeDir := filepath.Join(env.TempDir, ".claude")
			Expect(os.MkdirAll(projectClaudeDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(projectClaudeDir, "settings.json"), []byte(`{"enabledPlugins":{}}`), 0644)).To(Succeed())

			result := env.RunInDir(env.TempDir, "profile", "list", "--project")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --local flag in project directory", func() {
			// Create the .claude directory and local settings file in TempDir
			projectClaudeDir := filepath.Join(env.TempDir, ".claude")
			Expect(os.MkdirAll(projectClaudeDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(projectClaudeDir, "settings.local.json"), []byte(`{"enabledPlugins":{}}`), 0644)).To(Succeed())

			result := env.RunInDir(env.TempDir, "profile", "list", "--local")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})
	})

	Describe("profile clean", func() {
		It("accepts --project flag", func() {
			result := env.Run("profile", "clean", "--project", "test-plugin@marketplace")

			// Will fail because plugin doesn't exist, but flag is accepted
			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --local flag", func() {
			result := env.Run("profile", "clean", "--local", "test-plugin@marketplace")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("requires scope with new error message", func() {
			result := env.Run("profile", "clean", "test-plugin@marketplace")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("scope required: use --project or --local"))
		})
	})

	Describe("scope list", func() {
		It("accepts --user flag", func() {
			result := env.Run("scope", "list", "--user")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --project flag", func() {
			result := env.Run("scope", "list", "--project")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --local flag", func() {
			result := env.Run("scope", "list", "--local")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("rejects conflicting scope flags", func() {
			result := env.Run("scope", "list", "--user", "--project")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})
	})

	Describe("status", func() {
		It("accepts --user flag", func() {
			result := env.Run("status", "--user")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --project flag", func() {
			result := env.Run("status", "--project")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --local flag", func() {
			result := env.Run("status", "--local")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("rejects conflicting scope flags", func() {
			result := env.Run("status", "--user", "--local")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})
	})

	Describe("events", func() {
		It("accepts --user flag", func() {
			result := env.Run("events", "--user")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --project flag", func() {
			result := env.Run("events", "--project")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --local flag", func() {
			result := env.Run("events", "--local")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("rejects conflicting scope flags", func() {
			result := env.Run("events", "--scope", "user", "--project")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})
	})

	Describe("events audit", func() {
		It("accepts --user flag", func() {
			result := env.Run("events", "audit", "--user")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --project flag", func() {
			result := env.Run("events", "audit", "--project")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("accepts --local flag", func() {
			result := env.Run("events", "audit", "--local")

			Expect(result.Stderr).NotTo(ContainSubstring("unknown flag"))
		})

		It("rejects conflicting scope flags", func() {
			result := env.Run("events", "audit", "--user", "--local")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
		})
	})
})
