// ABOUTME: Acceptance tests for profile create command
// ABOUTME: Tests CLI behavior for interactive wizard to create new profiles
package acceptance

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// pathWithoutExecutable returns the current PATH with every directory that
// provides an executable named `name` removed. Acceptance tests use this to
// guarantee gum is never found on PATH, regardless of what happens to be
// installed on the host running the test -- see #292.
//
// Pass the result as an extraEnv["PATH"] override to RunWithEnv: Cmd.Env
// keeps only the last value for a duplicate key, so this override replaces
// baseEnv()'s unfiltered PATH for the child process rather than merging
// with it.
func pathWithoutExecutable(name string) string {
	dirs := filepath.SplitList(os.Getenv("PATH"))
	kept := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if dirContainsExecutable(dir, name) {
			continue
		}
		kept = append(kept, dir)
	}
	return strings.Join(kept, string(os.PathListSeparator))
}

func dirContainsExecutable(dir, name string) bool {
	for _, candidate := range executableCandidates(name) {
		info, err := os.Stat(filepath.Join(dir, candidate))
		if err != nil {
			continue
		}
		if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
			return true
		}
	}
	return false
}

func executableCandidates(name string) []string {
	candidates := []string{name}
	if runtime.GOOS != "windows" || filepath.Ext(name) != "" {
		return candidates
	}

	pathExt := os.Getenv("PATHEXT")
	if pathExt == "" {
		pathExt = ".COM;.EXE;.BAT;.CMD"
	}
	for _, ext := range filepath.SplitList(strings.ReplaceAll(pathExt, ";", string(os.PathListSeparator))) {
		if ext == "" {
			continue
		}
		candidates = append(candidates, name+ext)
	}
	return candidates
}

var _ = Describe("profile create", func() {
	// Note: Old tests that expected `create` to clone profiles have been removed.
	// The clone functionality now lives in `profile clone` command.
	// These tests now verify the interactive wizard behavior.

	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("wizard behavior", func() {
		It("starts wizard and fails gracefully in non-interactive mode", func() {
			// gum is deliberately excluded from PATH so this test's outcome
			// never depends on whether the host has gum installed, or on
			// gum's TTY-blocking behavior when it is. Without gum on PATH,
			// SelectMarketplaces always takes the stdin fallback, which
			// hits EOF immediately (no stdin is piped in) -- fast and
			// deterministic, never a candidate for commandTimeout. See #292.
			result := env.RunWithEnv(map[string]string{"PATH": pathWithoutExecutable("gum")}, "profile", "create", "new-profile")

			Expect(result.TimedOut).To(BeFalse())
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("failed to select marketplaces"))
		})
	})

	Context("when target profile already exists", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{Name: "existing"})
		})

		It("returns an error suggesting profile save", func() {
			result := env.Run("profile", "create", "existing")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("already exists"))
			Expect(result.Stderr).To(ContainSubstring("profile save"))
		})
	})

	Context("wizard mode", func() {
		It("shows help text", func() {
			result := env.Run("profile", "create", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Interactive wizard"))
		})
	})
})
