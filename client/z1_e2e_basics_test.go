package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLI_Info(t *testing.T) {
	config := newTestConfig()
	output := runInfo(t, config)

	require.Contains(t, output, "Plik client version :")
	require.Contains(t, output, "Plik server url :")
	require.Contains(t, output, "Plik server version :")
	require.Contains(t, output, "Plik server configuration :")
}

func TestCLI_Debug(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Debug = true
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":    []string{dir + "/FILE1"},
		"--debug": true,
	})

	// Debug output goes to stderr
	require.Contains(t, result.Stderr, "Arguments")
	require.Contains(t, result.Stderr, "Configuration")
}

func TestCLI_SingleFile(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	// Extract URL and download
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	content := downloadFileContent(t, fileURL)
	require.Equal(t, testContent, content, "downloaded content should match uploaded")
}

func TestCLI_SingleFileCustomName(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":   []string{dir + "/FILE1"},
		"--name": "CUSTOM",
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	require.Contains(t, fileURL, "CUSTOM", "URL should contain custom name")
	content := downloadFileContent(t, fileURL)
	require.Equal(t, testContent, content)
}

func TestCLI_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)
	createTestFile(t, dir, "FILE2", testContent+"extra")

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1", dir + "/FILE2"},
	})

	urls := extractAllFileURLsFromOutput(t, result.Stdout)
	require.Len(t, urls, 2, "should have 2 file URLs")

	for i, url := range urls {
		content := downloadFileContent(t, url)
		if strings.Contains(url, "FILE1") {
			require.Equal(t, testContent, content, "FILE1 content mismatch")
		} else if strings.Contains(url, "FILE2") {
			require.Equal(t, testContent+"extra", content, "FILE2 content mismatch")
		} else {
			t.Fatalf("unexpected URL at index %d: %s", i, url)
		}
	}
}

func TestCLI_Stdin(t *testing.T) {
	dumpServerLogs(t)

	config := newTestConfig()

	// Pipe content to os.Stdin (Run() reads from it directly)
	content := "stdin test content"
	r, w, err := os.Pipe()
	require.NoError(t, err)

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		w.Write([]byte(content))
		w.Close()
	}()

	config.filenameOverride = "STDIN_FILE"

	cli, client, outBuf, _, err := buildCLI(t, config, map[string]any{
		"FILE":   []string{},
		"--name": "STDIN_FILE",
	})
	require.NoError(t, err)
	cli.Stderr = io.Discard

	runErr := cli.Run(client)
	require.NoError(t, runErr, "Run() should succeed for stdin upload")

	fileURL := extractFileURLFromOutput(t, outBuf.String())
	downloaded := downloadFileContent(t, fileURL)
	require.Equal(t, content, downloaded, "downloaded content should match stdin input")
}
