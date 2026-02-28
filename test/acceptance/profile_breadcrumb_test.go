package acceptance

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
)

var _ = Describe("Profile breadcrumb", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("apply writes breadcrumb", func() {
		BeforeEach(func() {
			// Create a minimal profile (no plugins, so apply succeeds without claude CLI)
			env.CreateProfile(&profile.Profile{
				Name:        "test-profile",
				Description: "test",
			})
		})

		It("writes breadcrumb at user scope by default", func() {
			result := env.RunWithInput("y\n", "profile", "apply", "test-profile", "-y")

			Expect(result.ExitCode).To(Equal(0))
			bc := env.ReadBreadcrumb()
			Expect(bc).To(HaveKey("user"))
			Expect(bc["user"].Profile).To(Equal("test-profile"))
		})

		It("does not write breadcrumb in dry-run mode", func() {
			result := env.Run("profile", "apply", "test-profile", "--dry-run")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.BreadcrumbExists()).To(BeFalse())
		})

		It("warns but succeeds when breadcrumb file is corrupt", func() {
			// Write garbage to the breadcrumb file
			bcPath := filepath.Join(env.ClaudeupDir, "last-applied.json")
			Expect(os.WriteFile(bcPath, []byte("{invalid json"), 0600)).To(Succeed())

			result := env.RunWithInput("y\n", "profile", "apply", "test-profile", "-y")

			Expect(result.ExitCode).To(Equal(0))
			combined := result.Stdout + result.Stderr
			Expect(combined).To(ContainSubstring("breadcrumb"))
		})
	})

	Describe("save with no args", func() {
		It("saves to breadcrumbed profile name", func() {
			// Create original profile
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "original",
			})
			env.WriteBreadcrumb("user", "my-setup")

			// Add a plugin to live state so there's something to save
			env.CreateInstalledPlugins(map[string]interface{}{
				"new-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})

			result := env.RunWithInput("y\n", "profile", "save")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("errors when no breadcrumb exists", func() {
			result := env.Run("profile", "save")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no profile has been applied"))
		})

		It("explicit name still works", func() {
			result := env.RunWithInput("y\n", "profile", "save", "explicit-name")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("explicit-name"))
		})
	})

	Describe("delete cleans breadcrumb", func() {
		It("removes breadcrumb entry for deleted profile", func() {
			env.CreateProfile(&profile.Profile{
				Name: "to-delete",
			})
			env.WriteBreadcrumb("user", "to-delete")
			env.WriteBreadcrumb("project", "keep-this")

			result := env.RunWithInput("y\n", "profile", "delete", "to-delete")

			Expect(result.ExitCode).To(Equal(0))

			bc := env.ReadBreadcrumb()
			Expect(bc).NotTo(HaveKey("user"))
			Expect(bc).To(HaveKey("project"))
			Expect(bc["project"].Profile).To(Equal("keep-this"))
		})
	})

	Describe("rename updates breadcrumb", func() {
		It("updates breadcrumb entry to new name", func() {
			env.CreateProfile(&profile.Profile{
				Name: "old-name",
			})
			env.WriteBreadcrumb("user", "old-name")

			result := env.Run("profile", "rename", "old-name", "new-name")

			Expect(result.ExitCode).To(Equal(0))

			bc := env.ReadBreadcrumb()
			Expect(bc).To(HaveKey("user"))
			Expect(bc["user"].Profile).To(Equal("new-name"))
		})
	})

	Describe("diff with no args", func() {
		BeforeEach(func() {
			// Create a profile with a plugin at user scope
			env.CreateProfile(&profile.Profile{
				Name: "my-setup",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"extra-plugin@marketplace"},
					},
				},
			})
		})

		It("uses highest-precedence breadcrumb", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("prefers project breadcrumb over user", func() {
			projectDir := env.ProjectDir("my-project")

			env.WriteBreadcrumb("user", "some-other")
			env.WriteBreadcrumbWithDir("project", "my-setup", projectDir)

			// Create the other profile too
			env.CreateProfile(&profile.Profile{Name: "some-other"})

			result := env.RunInDir(projectDir, "profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("errors when no breadcrumb exists", func() {
			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no profile has been applied"))
		})

		It("errors when breadcrumbed profile is deleted", func() {
			env.WriteBreadcrumb("user", "deleted-profile")

			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})

		It("explicit name still works", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff", "my-setup")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("uses --user flag to select user scope breadcrumb", func() {
			env.WriteBreadcrumb("user", "my-setup")
			env.WriteBreadcrumb("project", "other-profile")
			env.CreateProfile(&profile.Profile{Name: "other-profile"})

			result := env.Run("profile", "diff", "--user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("errors when scope flag has no breadcrumb", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff", "--project")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no profile has been applied at project scope"))
		})

		It("errors when scope flag used with explicit name", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff", "my-setup", "--user")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot use --scope"))
		})

		It("errors on invalid scope value", func() {
			result := env.Run("profile", "diff", "--scope", "banana")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("invalid scope"))
		})
	})

	Describe("status shows last-applied info", func() {
		It("shows last-applied line when breadcrumb exists and profile matches", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test profile",
			})
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("user scope"))
		})

		It("shows (modified) when live state differs from saved profile", func() {
			// Create profile with a plugin
			env.CreateProfile(&profile.Profile{
				Name: "my-setup",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"saved-plugin@marketplace"},
					},
				},
			})
			env.WriteBreadcrumb("user", "my-setup")

			// Live state has a different plugin
			env.CreateInstalledPlugins(map[string]interface{}{
				"different-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("modified"))
		})

		It("does not show (modified) when live matches saved profile", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test",
			})
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).NotTo(ContainSubstring("modified"))
		})

		It("does not show (modified) when unrelated config exists at other scopes", func() {
			// Profile only defines user-scope settings
			env.CreateProfile(&profile.Profile{
				Name: "user-only",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"my-plugin@marketplace"},
					},
				},
			})
			env.WriteBreadcrumb("user", "user-only")

			// Enable the same plugin at user scope (matches profile)
			env.CreateSettings(map[string]bool{
				"my-plugin@marketplace": true,
			})
			// Use a separate project directory so project-scope settings.json
			// doesn't overwrite user-scope settings.json (they'd be the same
			// path if projectDir == env.TempDir since ClaudeDir = TempDir/.claude).
			projectDir := GinkgoT().TempDir()
			env.CreateProjectScopeSettings(projectDir, map[string]bool{
				"project-plugin@marketplace": true,
			})

			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("user-only"))
			// Should NOT show modified -- the user-scope config matches
			Expect(result.Stdout).NotTo(ContainSubstring("modified"))
		})

		It("skips last-applied section when no breadcrumb exists", func() {
			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("Last applied"))
		})

		It("skips last-applied section when breadcrumbed profile is missing", func() {
			env.WriteBreadcrumb("user", "deleted-profile")

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("Last applied"))
		})
	})

	Describe("list shows applied marker", func() {
		It("shows (applied) on the matching profile", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test profile",
			})
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("(applied)"))
		})

		It("shows (applied, modified) when profile has drifted", func() {
			env.CreateProfile(&profile.Profile{
				Name: "my-setup",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"saved-plugin@marketplace"},
					},
				},
			})
			env.WriteBreadcrumb("user", "my-setup")

			// Live state has a different plugin
			env.CreateInstalledPlugins(map[string]interface{}{
				"different-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("(applied, modified)"))
		})

		It("does not show (modified) when unrelated config exists at other scopes", func() {
			env.CreateProfile(&profile.Profile{
				Name: "user-only",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"my-plugin@marketplace"},
					},
				},
			})
			env.WriteBreadcrumb("user", "user-only")

			// Enable the same plugin at user scope (matches profile)
			env.CreateSettings(map[string]bool{
				"my-plugin@marketplace": true,
			})
			// Use a separate project directory so project-scope settings.json
			// doesn't overwrite user-scope settings.json.
			projectDir := GinkgoT().TempDir()
			env.CreateProjectScopeSettings(projectDir, map[string]bool{
				"project-plugin@marketplace": true,
			})

			result := env.RunInDir(projectDir, "profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "user-only") {
					Expect(line).To(ContainSubstring("(applied)"))
					Expect(line).NotTo(ContainSubstring("modified"))
				}
			}
		})

		It("shows no marker when no breadcrumb exists", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test profile",
			})

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).NotTo(ContainSubstring("(applied"))
		})

		It("shows marker on built-in profile when breadcrumbed", func() {
			env.WriteBreadcrumb("user", "frontend")

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("frontend"))
			// Built-in frontend has plugins; empty test env means it's always modified
			Expect(result.Stdout).To(ContainSubstring("(applied, modified)"))
		})

		It("does not show marker on non-applied profiles", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "applied-one",
				Description: "applied",
			})
			env.CreateProfile(&profile.Profile{
				Name:        "not-applied",
				Description: "should have no marker",
			})
			env.WriteBreadcrumb("user", "applied-one")

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			// applied-one should have marker
			Expect(result.Stdout).To(ContainSubstring("(applied)"))
			// Verify "not-applied" line does not contain the marker by checking
			// that "(applied)" only appears once (on applied-one's line)
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "not-applied") {
					Expect(line).NotTo(ContainSubstring("(applied"))
				}
			}
		})

		It("shows markers on multiple profiles at different scopes", func() {
			projectDir := env.ProjectDir("multi-scope")

			env.CreateProfile(&profile.Profile{
				Name:        "user-profile",
				Description: "at user scope",
			})
			env.CreateProfile(&profile.Profile{
				Name:        "project-profile",
				Description: "at project scope",
			})
			env.WriteBreadcrumb("user", "user-profile")
			env.WriteBreadcrumbWithDir("project", "project-profile", projectDir)

			result := env.RunInDir(projectDir, "profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			// Both profiles should have applied markers
			lines := strings.Split(result.Stdout, "\n")
			userLine := ""
			projectLine := ""
			for _, line := range lines {
				if strings.Contains(line, "user-profile") {
					userLine = line
				}
				if strings.Contains(line, "project-profile") {
					projectLine = line
				}
			}
			Expect(userLine).To(ContainSubstring("(applied"))
			Expect(projectLine).To(ContainSubstring("(applied"))
		})

		It("shows marker on nested profile when breadcrumbed by path", func() {
			env.CreateNestedProfile("team", &profile.Profile{
				Name:        "backend",
				Description: "team backend profile",
			})
			env.WriteBreadcrumb("user", "team/backend")

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("backend"))
			Expect(result.Stdout).To(ContainSubstring("(applied"))
		})

		It("shows marker on nested profile applied by leaf name", func() {
			env.CreateNestedProfile("team", &profile.Profile{
				Name:        "backend",
				Description: "team backend profile",
			})

			// Apply by leaf name (not path) -- the breadcrumb should
			// record the display name so it matches on listing
			result := env.Run("profile", "apply", "backend")
			Expect(result.ExitCode).To(Equal(0))

			result = env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "backend") {
					Expect(line).To(ContainSubstring("(applied"))
				}
			}
		})

		It("shows applied marker alongside stack marker", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-stack",
				Description: "a stack profile",
				Includes:    []string{"base", "extras"},
			})
			env.WriteBreadcrumb("user", "my-stack")

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "my-stack") {
					Expect(line).To(ContainSubstring("[stack]"))
					Expect(line).To(ContainSubstring("(applied"))
				}
			}
		})

		It("shows marker at project scope breadcrumb", func() {
			projectDir := env.ProjectDir("proj-scope")

			env.CreateProfile(&profile.Profile{
				Name:        "proj-setup",
				Description: "project profile",
			})
			env.WriteBreadcrumbWithDir("project", "proj-setup", projectDir)

			result := env.RunInDir(projectDir, "profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("proj-setup"))
			Expect(result.Stdout).To(ContainSubstring("(applied"))
		})
	})

	Describe("status scope display", func() {
		It("shows project scope when breadcrumbed at project scope", func() {
			projectDir := env.ProjectDir("proj-scope-status")

			env.CreateProfile(&profile.Profile{
				Name:        "proj-setup",
				Description: "project profile",
			})
			env.WriteBreadcrumbWithDir("project", "proj-setup", projectDir)

			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("proj-setup"))
			Expect(result.Stdout).To(ContainSubstring("project scope"))
		})

		It("shows local scope when breadcrumbed at local scope", func() {
			projectDir := env.ProjectDir("local-scope-status")

			env.CreateProfile(&profile.Profile{
				Name:        "local-setup",
				Description: "local profile",
			})
			env.WriteBreadcrumbWithDir("local", "local-setup", projectDir)

			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("local-setup"))
			Expect(result.Stdout).To(ContainSubstring("local scope"))
		})

		It("shows highest-precedence scope when multiple breadcrumbs exist", func() {
			projectDir := env.ProjectDir("multi-scope-status")

			env.CreateProfile(&profile.Profile{
				Name:        "user-profile",
				Description: "user",
			})
			env.CreateProfile(&profile.Profile{
				Name:        "local-profile",
				Description: "local",
			})
			env.WriteBreadcrumb("user", "user-profile")
			env.WriteBreadcrumbWithDir("local", "local-profile", projectDir)

			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("local-profile"))
			Expect(result.Stdout).To(ContainSubstring("local scope"))
		})
	})

	Describe("directory-specific breadcrumbs", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "proj-profile",
				Description: "project-scoped profile",
			})
		})

		It("does not show project breadcrumb as applied from a different directory", func() {
			origDir := env.ProjectDir("original")
			otherDir := env.ProjectDir("other")

			env.WriteBreadcrumbWithDir("project", "proj-profile", origDir)

			result := env.RunInDir(otherDir, "profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "proj-profile") {
					Expect(line).NotTo(ContainSubstring("(applied"))
				}
			}
		})

		It("shows project breadcrumb as applied from the same directory", func() {
			projectDir := env.ProjectDir("matching")

			env.WriteBreadcrumbWithDir("project", "proj-profile", projectDir)

			result := env.RunInDir(projectDir, "profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "proj-profile") {
					Expect(line).To(ContainSubstring("(applied"))
				}
			}
		})

		It("shows user breadcrumb from any directory", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "user-profile",
				Description: "user-scoped profile",
			})
			env.WriteBreadcrumb("user", "user-profile")

			otherDir := env.ProjectDir("random-dir")
			result := env.RunInDir(otherDir, "profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "user-profile") {
					Expect(line).To(ContainSubstring("(applied"))
				}
			}
		})

		It("does not show last-applied in status when project breadcrumb is for different directory", func() {
			origDir := env.ProjectDir("original")
			otherDir := env.ProjectDir("other")

			env.WriteBreadcrumbWithDir("project", "proj-profile", origDir)

			result := env.RunInDir(otherDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("Last applied"))
		})

		It("falls back to user breadcrumb when project breadcrumb is for different directory", func() {
			origDir := env.ProjectDir("original")
			otherDir := env.ProjectDir("other")

			env.CreateProfile(&profile.Profile{
				Name:        "user-base",
				Description: "user-scoped",
			})
			env.WriteBreadcrumb("user", "user-base")
			env.WriteBreadcrumbWithDir("project", "proj-profile", origDir)

			result := env.RunInDir(otherDir, "profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			// Should use user breadcrumb, not the project one from a different dir
			Expect(result.Stdout).To(ContainSubstring("user-base"))
		})

		It("does not show multi-scope profile as modified from a different directory", func() {
			origDir := env.ProjectDir("original")
			otherDir := env.ProjectDir("other")

			// Create a multi-scope profile with both user and project settings
			env.CreateProfile(&profile.Profile{
				Name:        "multi-scope",
				Description: "has user and project settings",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"some-plugin@some-marketplace"},
					},
					Project: &profile.ScopeSettings{
						Plugins: []string{"project-plugin@some-marketplace"},
					},
				},
			})

			// Apply user settings so user-scope matches
			env.CreateSettings(map[string]bool{
				"some-plugin@some-marketplace": true,
			})

			// Breadcrumbs: user is global, project is directory-specific
			env.WriteBreadcrumb("user", "multi-scope")
			env.WriteBreadcrumbWithDir("project", "multi-scope", origDir)

			// From a different directory, only user breadcrumb is active.
			// Should not show modified due to missing project-scope settings.
			result := env.RunInDir(otherDir, "profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			lines := strings.Split(result.Stdout, "\n")
			for _, line := range lines {
				if strings.Contains(line, "multi-scope") {
					Expect(line).To(ContainSubstring("(applied)"))
					Expect(line).NotTo(ContainSubstring("modified"))
				}
			}
		})
	})
})
