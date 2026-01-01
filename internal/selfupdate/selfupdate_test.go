// ABOUTME: Integration tests for self-update functionality
// ABOUTME: Uses mock HTTP server for GitHub API responses
package selfupdate

import (
	"net/http"
	"net/http/httptest"

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
