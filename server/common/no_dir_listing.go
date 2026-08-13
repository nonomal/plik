package common

import (
	"net/http"
	"strings"
)

// NoDirListing wraps an http.Handler (typically http.FileServer) to return
// 404 for directory requests instead of generating a directory listing.
// The root path "/" is allowed through so that index.html is served for SPAs.
// All other paths ending in "/" are blocked.
func NoDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
