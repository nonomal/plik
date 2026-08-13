package common

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoDirListing_BlocksSubdirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.Mkdir(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("hello"), 0644))

	fs := NoDirListing(http.FileServer(http.Dir(dir)))

	// Subdirectory listing should return 404
	req, err := http.NewRequest("GET", "/subdir/", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	fs.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestNoDirListing_AllowsRootIndex(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>ok</html>"), 0644))

	fs := NoDirListing(http.FileServer(http.Dir(dir)))

	// Root "/" should pass through (serves index.html)
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	fs.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "<html>ok</html>")
}

func TestNoDirListing_AllowsFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644))

	fs := NoDirListing(http.FileServer(http.Dir(dir)))

	// File request should succeed
	req, err := http.NewRequest("GET", "/test.txt", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	fs.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "hello", rr.Body.String())
}
