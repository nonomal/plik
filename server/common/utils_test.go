package common

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStripPrefixNoPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/prefix", req.URL.Path, "invalid request url")
}

func TestStripPrefixNoExactPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, 301, rr.Code, "invalid handler response status code")
	require.Equal(t, "/prefix/", rr.Result().Header.Get("Location"), "invalid location header")
}

func TestStripPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/path", req.URL.Path, "invalid location header")
}

func TestStripPrefixNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/invalid/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code, "invalid handler response status code")
}

func TestStripPrefixRootSlash(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix/", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/path", req.URL.Path, "invalid location header")
}

func TestNewHTTPError(t *testing.T) {
	e := NewHTTPError("msg", fmt.Errorf("error"), http.StatusInternalServerError)
	require.Equal(t, "msg : error", e.Error())
}

func TestEncodeAuthBasicHeader(t *testing.T) {
	b64 := EncodeAuthBasicHeader("login", "password")
	out := make([]byte, 14)
	_, err := base64.StdEncoding.Decode(out, []byte(b64))
	require.NoError(t, err)
	require.Equal(t, "login:password", string(out))
}

func TestWriteJSONResponse(t *testing.T) {
	obj := &struct{ Foo string }{"Bar"}

	rr := httptest.NewRecorder()
	WriteJSONResponse(rr, obj)

	body, err := io.ReadAll(rr.Body)
	require.NoError(t, err)
	require.NotNil(t, body)

	obj2 := &struct{ Foo string }{}
	err = json.Unmarshal(body, obj2)
	require.NoError(t, err)

	require.Equal(t, obj.Foo, obj2.Foo)
}

func TestHumanDuration(t *testing.T) {
	require.Equal(t, "0s", HumanDuration(time.Duration(0)))
	require.Equal(t, "10ms", HumanDuration(10*time.Millisecond))
	require.Equal(t, "1s10ms", HumanDuration(time.Second+10*time.Millisecond))
	require.Equal(t, "30s", HumanDuration(30*time.Second))
	require.Equal(t, "30m", HumanDuration(30*time.Minute))
	require.Equal(t, "30m3s", HumanDuration(30*time.Minute+3*time.Second))
	require.Equal(t, "1h", HumanDuration(time.Hour))
	require.Equal(t, "1h1s", HumanDuration(time.Hour+time.Second))
	require.Equal(t, "1h1m", HumanDuration(time.Hour+time.Minute))
	require.Equal(t, "1h1m1s", HumanDuration(time.Hour+time.Minute+time.Second))
	require.Equal(t, "1d", HumanDuration(24*time.Hour))
	require.Equal(t, "1d1m1s", HumanDuration(24*time.Hour+time.Minute+time.Second))
	require.Equal(t, "1d1h1m1s", HumanDuration(24*time.Hour+time.Hour+time.Minute+time.Second))
	require.Equal(t, "30d", HumanDuration(30*24*time.Hour))
	require.Equal(t, "1y", HumanDuration(365*24*time.Hour))
	require.Equal(t, "1y1d", HumanDuration(366*24*time.Hour))
	require.Equal(t, "1y1d1s", HumanDuration(366*24*time.Hour+time.Second))
	require.Equal(t, "1y1d1h1m1s", HumanDuration(366*24*time.Hour+3661*time.Second))
	require.Equal(t, "-10s", HumanDuration(-10*time.Second))
}

func TestSanitizeFilenameForDisposition(t *testing.T) {
	// Normal filename passes through unchanged
	require.Equal(t, "file.txt", SanitizeFilenameForDisposition("file.txt"))

	// Double quotes are stripped
	require.Equal(t, "file.txt", SanitizeFilenameForDisposition(`fi"le.txt`))

	// CRLF characters are stripped (header injection vector)
	require.Equal(t, "file.txt", SanitizeFilenameForDisposition("file\r\n.txt"))

	// Null bytes are stripped
	require.Equal(t, "file.txt", SanitizeFilenameForDisposition("file\x00.txt"))

	// Empty string
	require.Equal(t, "", SanitizeFilenameForDisposition(""))

	// Multiple dangerous characters at once
	require.Equal(t, "malicious.txt", SanitizeFilenameForDisposition("mal\"\rici\nous\x00.txt"))

	// Filename truncated at 1024 characters
	longName := strings.Repeat("a", 1025)
	require.Len(t, SanitizeFilenameForDisposition(longName), 1024)

	// Exactly 1024 passes through
	exactName := strings.Repeat("b", 1024)
	require.Len(t, SanitizeFilenameForDisposition(exactName), 1024)

	// Unicode BiDi override characters are stripped (RLO can spoof file extensions)
	require.Equal(t, "evilfdp.exe", SanitizeFilenameForDisposition("evil\u202Efdp.exe"))

	// Multiple BiDi overrides are stripped
	require.Equal(t, "safe.txt", SanitizeFilenameForDisposition("\u202A\u202Bsafe\u2066.\u2069txt\u202C"))
}

func TestGenerateRandomID(t *testing.T) {
	// Correct length
	for _, length := range []int{0, 1, 8, 16, 32, 64, 128} {
		id := GenerateRandomID(length)
		require.Equal(t, length, len(id), "unexpected length for GenerateRandomID(%d)", length)
	}

	// All characters are within Base62Charset
	id := GenerateRandomID(1000)
	for i, c := range id {
		require.Contains(t, Base62Charset, string(c), "char %d (%q) not in Base62Charset", i, c)
	}

	// Uniqueness (no collisions in 1000 IDs of length 16)
	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		s := GenerateRandomID(16)
		_, exists := seen[s]
		require.False(t, exists, "duplicate ID: %s", s)
		seen[s] = struct{}{}
	}
}

func BenchmarkGenerateRandomID(b *testing.B) {
	for b.Loop() {
		GenerateRandomID(32)
	}
}

func TestLookupBinary(t *testing.T) {
	// Valid path → returned as-is
	path, err := LookupBinary("/bin/sh", "sh")
	require.NoError(t, err)
	require.Equal(t, "/bin/sh", path)

	// Invalid configured path → fallback to $PATH
	path, err = LookupBinary("/nonexistent/bin/sh", "sh")
	require.NoError(t, err)
	require.NotEmpty(t, path) // resolved via exec.LookPath

	// Both invalid → descriptive error
	_, err = LookupBinary("/nonexistent/nope", "nonexistent_binary_xyz")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent_binary_xyz not found at /nonexistent/nope")
	require.Contains(t, err.Error(), "not in $PATH")
}
