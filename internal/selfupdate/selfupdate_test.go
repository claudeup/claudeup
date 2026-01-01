// ABOUTME: Integration tests for self-update functionality
// ABOUTME: Uses mock HTTP server for GitHub API responses
package selfupdate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

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
