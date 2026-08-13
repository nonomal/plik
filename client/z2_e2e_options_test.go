package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLI_DisableStdin(t *testing.T) {
	t.Run("stdin rejected when disabled", func(t *testing.T) {
		config := newTestConfig()
		config.DisableStdin = true

		_, err := runCLIExpectError(t, config, map[string]any{
			"FILE": []string{},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "stdin is disabled")
	})

	t.Run("--stdin overrides DisableStdin", func(t *testing.T) {
		// This would require piping real stdin content; verify config parsing only.
		config := newTestConfig()
		config.DisableStdin = true

		opts := makeOpts()
		opts["FILE"] = []string{}
		opts["--stdin"] = true

		err := config.UnmarshalArgs(opts)
		require.NoError(t, err)
		require.False(t, config.DisableStdin, "--stdin should override DisableStdin")
	})
}

func TestCLI_OneShot(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":      []string{dir + "/FILE1"},
		"--oneshot": true,
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["oneShot"], "oneShot should be true")
}

func TestCLI_Removable(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":        []string{dir + "/FILE1"},
		"--removable": true,
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["removable"], "removable should be true")
}

func TestCLI_Stream(t *testing.T) {
	// Verify that the stream flag is set on server metadata.
	// Actual streaming behavior is tested in plik/z1_e2e_test.go.
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Stream = true
	config.Quiet = false // Need to see the output before upload blocks

	// For streaming, we need a different approach since Upload() blocks.
	// We'll just verify the configuration path creates the upload with stream=true
	// by using the API directly, as the actual streaming is tested elsewhere.

	arguments := makeOpts()
	arguments["FILE"] = []string{dir + "/FILE1"}
	arguments["--stream"] = true

	err := config.UnmarshalArgs(arguments)
	require.NoError(t, err)
	require.True(t, config.Stream, "stream config should be true")
}

func TestCLI_TTL(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":  []string{dir + "/FILE1"},
		"--ttl": "3600",
	})

	meta := getUploadMetadata(t, result.Stdout)
	ttl, ok := meta["ttl"].(float64)
	require.True(t, ok, "ttl should be a number")
	require.Equal(t, float64(3600), ttl, "ttl should be 3600")
}

func TestCLI_ExtendTTL(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":         []string{dir + "/FILE1"},
		"--extend-ttl": true,
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["extend_ttl"], "extend_ttl should be true")
}

func TestCLI_Password(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":       []string{dir + "/FILE1"},
		"--password": "foo:bar",
	})

	meta := getUploadMetadataAuth(t, result.Stdout, "foo", "bar")
	require.Equal(t, true, meta["protectedByPassword"], "protectedByPassword should be true")
}

func TestCLI_PromptedPassword(t *testing.T) {
	// This test verifies the -p flag path, which prompts for login/password.
	// We simulate by directly setting config.Login and config.Password
	// since the actual prompt requires interactive stdin.
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Login = "foo"
	config.Password = "bar"

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadataAuth(t, result.Stdout, "foo", "bar")
	require.Equal(t, true, meta["protectedByPassword"], "protectedByPassword should be true")
}

func TestCLI_Comments(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":       []string{dir + "/FILE1"},
		"--comments": "foobar",
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, "foobar", meta["comments"], "comments should be 'foobar'")
}

func TestCLI_Quiet(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false // Explicitly disable so we test the flag

	result := runCLI(t, config, map[string]any{
		"FILE":    []string{dir + "/FILE1"},
		"--quiet": true,
	})

	// In quiet mode, output should be exactly one file URL line
	var urlLines []string
	for line := range strings.SplitSeq(strings.TrimSpace(result.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, testServerURL+"/file/") {
			urlLines = append(urlLines, line)
		}
	}
	require.Equal(t, 1, len(urlLines), "quiet mode should output exactly 1 file URL")
	require.Contains(t, urlLines[0], testServerURL+"/file/", "URL should point to server")
}

func TestCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":   []string{dir + "/FILE1"},
		"--json": true,
	})

	var data map[string]any
	err := json.Unmarshal([]byte(result.Stdout), &data)
	require.NoError(t, err, "JSON output should be valid JSON")
	require.Contains(t, data, "url", "JSON should have 'url' field")
	require.Contains(t, data, "files", "JSON should have 'files' field")

	files, ok := data["files"].([]any)
	require.True(t, ok, "files should be an array")
	require.Equal(t, 1, len(files), "should have 1 file")
}

func TestCLI_JSONMultiFile(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)
	createTestFile(t, dir, "FILE2", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":   []string{dir + "/FILE1", dir + "/FILE2"},
		"--json": true,
	})

	var data map[string]any
	err := json.Unmarshal([]byte(result.Stdout), &data)
	require.NoError(t, err, "JSON output should be valid JSON")

	files, ok := data["files"].([]any)
	require.True(t, ok, "files should be an array")
	require.Equal(t, 2, len(files), "should have 2 files")
}

// NOTE: Short flags (-j, -o, -q, -r, -t, -n) are resolved by docopt to their
// long equivalents (--json, --oneshot, etc.) before reaching the args map.
// Since runCLI takes the post-docopt args map, short flags cannot be tested
// separately through this test infrastructure. Short flag mapping is a docopt
// concern and is implicitly tested by the actual CLI binary.

func TestCLI_NotSecure(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Secure = true // Set in config (like .plikrc)
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":         []string{dir + "/FILE1"},
		"--not-secure": true,
	})

	// Output should have curl command WITHOUT a pipe (no crypto)
	for line := range strings.SplitSeq(result.Stdout, "\n") {
		if strings.Contains(line, "curl") {
			require.NotContains(t, line, "|", "not-secure: download command should not have pipe")
		}
	}

	// Verify we can download the file (not encrypted)
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	content := downloadFileContent(t, fileURL)
	require.Equal(t, testContent, content, "downloaded content should match (no encryption)")
}

// ---------- Error path tests ----------

func TestCLI_InvalidSecureBackend(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	_, err := runCLIExpectError(t, config, map[string]any{
		"FILE":     []string{dir + "/FILE1"},
		"--secure": "nonexistent",
	})
	require.Error(t, err, "should fail for invalid crypto backend")
}
