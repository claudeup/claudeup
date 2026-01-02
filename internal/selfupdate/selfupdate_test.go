// ABOUTME: Integration tests for self-update functionality
// ABOUTME: Uses mock HTTP server for GitHub API responses
package selfupdate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckLatestVersion", func() {
	var server *httptest.Server

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	It("parses version from GitHub releases API", func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"tag_name": "v2.1.0"}`))
		}))

		version, err := CheckLatestVersion(server.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(version).To(Equal("v2.1.0"))
	})

	It("returns error on network failure", func() {
		_, err := CheckLatestVersion("http://localhost:99999")
		Expect(err).To(HaveOccurred())
	})

	It("returns error on invalid JSON", func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`not json`))
		}))

		_, err := CheckLatestVersion(server.URL)
		Expect(err).To(HaveOccurred())
	})

	It("returns error on non-200 status", func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

		_, err := CheckLatestVersion(server.URL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("status 404"))
	})
})

var _ = Describe("DownloadBinary", func() {
	var server *httptest.Server
	var tempDir string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	It("downloads binary to temp file", func() {
		binaryContent := []byte("fake binary content")
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(binaryContent)
		}))

		path, err := DownloadBinary(server.URL, tempDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(BeARegularFile())

		content, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(Equal(binaryContent))
	})
})

var _ = Describe("VerifyChecksum", func() {
	var tempDir string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
	})

	It("returns nil for valid checksum", func() {
		content := []byte("test content")
		filePath := filepath.Join(tempDir, "testfile")
		Expect(os.WriteFile(filePath, content, 0644)).To(Succeed())

		// Known SHA256 of "test content"
		expectedHash := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
		err := VerifyChecksum(filePath, expectedHash)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error for invalid checksum", func() {
		content := []byte("test content")
		filePath := filepath.Join(tempDir, "testfile")
		Expect(os.WriteFile(filePath, content, 0644)).To(Succeed())

		err := VerifyChecksum(filePath, "badhash")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("checksum mismatch"))
	})
})

var _ = Describe("ReplaceBinary", func() {
	var tempDir string
	var currentBinary string
	var newBinary string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
		currentBinary = filepath.Join(tempDir, "claudeup")
		Expect(os.WriteFile(currentBinary, []byte("old version"), 0755)).To(Succeed())
		newBinary = filepath.Join(tempDir, "claudeup-new")
		Expect(os.WriteFile(newBinary, []byte("new version"), 0755)).To(Succeed())
	})

	It("replaces current binary with new one", func() {
		err := ReplaceBinary(currentBinary, newBinary)
		Expect(err).NotTo(HaveOccurred())
		content, err := os.ReadFile(currentBinary)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("new version"))
	})

	It("cleans up backup file on success", func() {
		err := ReplaceBinary(currentBinary, newBinary)
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(currentBinary + ".old")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})
})

var _ = Describe("Update", func() {
	It("returns AlreadyUpToDate when versions match", func() {
		result := Update("v1.0.0", "v1.0.0", "")
		Expect(result.AlreadyUpToDate).To(BeTrue())
		Expect(result.Error).NotTo(HaveOccurred())
	})
})

var _ = Describe("ValidateVersion", func() {
	It("accepts valid semver versions", func() {
		Expect(ValidateVersion("v1.0.0")).To(Succeed())
		Expect(ValidateVersion("v1.2.3")).To(Succeed())
		Expect(ValidateVersion("v10.20.30")).To(Succeed())
	})

	It("accepts pre-release versions", func() {
		Expect(ValidateVersion("v1.0.0-beta")).To(Succeed())
		Expect(ValidateVersion("v1.0.0-rc1")).To(Succeed())
	})

	It("rejects empty version", func() {
		err := ValidateVersion("")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot be empty"))
	})

	It("rejects versions with invalid characters", func() {
		err := ValidateVersion("v1.0.0; rm -rf /")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid character"))
	})

	It("rejects versions with path traversal", func() {
		err := ValidateVersion("../../../etc/passwd")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("fetchExpectedChecksum", func() {
	var server *httptest.Server

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	It("parses checksum from valid checksums.txt", func() {
		// Use valid 64-character hex checksums
		// Build the binary name dynamically to match the test platform
		binaryName := fmt.Sprintf("claudeup-%s-%s", runtime.GOOS, runtime.GOARCH)
		// Valid SHA256 hash is exactly 64 hex characters
		validHash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
		checksumContent := fmt.Sprintf("%s  %s\n", validHash, binaryName)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(checksumContent))
		}))

		hash, err := fetchExpectedChecksum(server.URL, "v1.0.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(hash).To(Equal(validHash))
	})

	It("returns error when platform checksum not found", func() {
		// Checksums file exists but doesn't have entry for current platform
		checksumContent := "abc123def456abc123def456abc123def456abc123def456abc123def456abcd1234  claudeup-unknown-platform\n"

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(checksumContent))
		}))

		_, err := fetchExpectedChecksum(server.URL, "v1.0.0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("checksum not found for"))
	})

	It("returns error for invalid checksum format (wrong field count)", func() {
		// Malformed line with only one field
		binaryName := "claudeup-" + "darwin-arm64"
		checksumContent := "abc123" + binaryName + "\n" // Missing space separator

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(checksumContent))
		}))

		_, err := fetchExpectedChecksum(server.URL, "v1.0.0")
		Expect(err).To(HaveOccurred())
		// Either "not found" because the suffix doesn't match, or "invalid format"
	})

	It("returns error for invalid checksum length", func() {
		// Checksum with wrong length (not 64 chars)
		checksumContent := "short  claudeup-darwin-arm64\n"

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(checksumContent))
		}))

		_, err := fetchExpectedChecksum(server.URL, "v1.0.0")
		Expect(err).To(HaveOccurred())
		// Will return "not found" because line parsing fails validation
	})

	It("returns error when checksums file download fails", func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

		_, err := fetchExpectedChecksum(server.URL, "v1.0.0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("status 404"))
	})
})
