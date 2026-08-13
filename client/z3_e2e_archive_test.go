package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------- Tar tests ----------

func TestCLI_TarSingleFile(t *testing.T) {
	requireBinary(t, "tar")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
		"-a":   true,
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractTar(t, fileURL, dlDir, "archive.tar.gz")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted))
}

func TestCLI_TarMultiFile(t *testing.T) {
	requireBinary(t, "tar")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)
	createTestFile(t, dir, "FILE2", testContent+"extra")

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1", dir + "/FILE2"},
		"-a":   true,
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractTar(t, fileURL, dlDir, "archive.tar.gz")

	extracted1Path := findExtractedFile(t, dlDir, "FILE1")
	extracted1, err := os.ReadFile(extracted1Path)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted1))

	extracted2Path := findExtractedFile(t, dlDir, "FILE2")
	extracted2, err := os.ReadFile(extracted2Path)
	require.NoError(t, err)
	require.Equal(t, testContent+"extra", string(extracted2))
}

func TestCLI_TarDirectory(t *testing.T) {
	requireBinary(t, "tar")

	dir := t.TempDir()
	subDir := createTestDir(t, dir, "DIR")
	createTestFile(t, subDir, "FILE1", testContent)
	createTestFile(t, subDir, "FILE2", testContent+"dir")

	config := newTestConfig()

	// Directory automatically enables archive mode
	result := runCLI(t, config, map[string]any{
		"FILE": []string{subDir},
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractTar(t, fileURL, dlDir, "archive.tar.gz")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted1, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted1))
}

func TestCLI_TarCustomCompression(t *testing.T) {
	requireBinary(t, "tar")
	requireBinary(t, "bzip2")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":       []string{dir + "/FILE1"},
		"-a":         true,
		"--compress": "bzip2",
	})

	// Verify the output mentions .tar.bz2
	require.Contains(t, result.Stdout, ".tar.bz2", "output should contain .tar.bz2 filename")

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractTarBz2(t, fileURL, dlDir, "archive.tar.bz2")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted))
}

func TestCLI_TarCustomOptions(t *testing.T) {
	requireBinary(t, "tar")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)
	createTestFile(t, dir, "EXCLUDE", "should be excluded")

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":              []string{dir + "/FILE1", dir + "/EXCLUDE"},
		"-a":                true,
		"--archive-options": "--exclude=*/EXCLUDE",
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractTar(t, fileURL, dlDir, "archive.tar.gz")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted))

	requireFileNotExtracted(t, dlDir, "EXCLUDE")
}

func TestCLI_TarCustomName(t *testing.T) {
	requireBinary(t, "tar")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":   []string{dir + "/FILE1"},
		"-a":     true,
		"--name": "foobar.tar.gz",
	})

	require.Contains(t, result.Stdout, "foobar.tar.gz", "output should contain custom name")

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractTar(t, fileURL, dlDir, "foobar.tar.gz")
}

func TestCLI_TarDirCustomName(t *testing.T) {
	requireBinary(t, "tar")

	dir := t.TempDir()
	subDir := createTestDir(t, dir, "DIR")
	createTestFile(t, subDir, "FILE1", testContent)
	createTestFile(t, subDir, "FILE2", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":   []string{subDir},
		"--name": "foobar.tar.gz",
	})

	require.Contains(t, result.Stdout, "foobar.tar.gz", "output should contain custom name")
}

// ---------- Zip tests ----------

func TestCLI_ZipSingleFile(t *testing.T) {
	requireBinary(t, "zip")
	requireBinary(t, "unzip")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":      []string{dir + "/FILE1"},
		"--archive": "zip",
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractZip(t, fileURL, dlDir, "FILE1.zip")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted))
}

func TestCLI_ZipDirectory(t *testing.T) {
	requireBinary(t, "zip")
	requireBinary(t, "unzip")

	dir := t.TempDir()
	subDir := createTestDir(t, dir, "DIR")
	createTestFile(t, subDir, "FILE1", testContent)
	createTestFile(t, subDir, "FILE2", testContent+"zip")

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":      []string{subDir},
		"--archive": "zip",
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractZip(t, fileURL, dlDir, "DIR.zip")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted))
}

func TestCLI_ZipCustomOptions(t *testing.T) {
	requireBinary(t, "zip")
	requireBinary(t, "unzip")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)
	createTestFile(t, dir, "EXCLUDE", "should be excluded")

	config := newTestConfig()

	result := runCLI(t, config, map[string]any{
		"FILE":              []string{dir + "/FILE1", dir + "/EXCLUDE"},
		"--archive":         "zip",
		"--archive-options": "--exclude */EXCLUDE",
	})

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractZip(t, fileURL, dlDir, "archive.zip")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted))

	requireFileNotExtracted(t, dlDir, "EXCLUDE")
}

func TestCLI_ZipCustomName(t *testing.T) {
	requireBinary(t, "zip")
	requireBinary(t, "unzip")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":      []string{dir + "/FILE1"},
		"--archive": "zip",
		"--name":    "foobar.zip",
	})

	require.Contains(t, result.Stdout, "foobar.zip", "output should contain custom name")
}

func TestCLI_ZipDirCustomName(t *testing.T) {
	requireBinary(t, "zip")
	requireBinary(t, "unzip")

	dir := t.TempDir()
	subDir := createTestDir(t, dir, "DIR")
	createTestFile(t, subDir, "FILE1", testContent)
	createTestFile(t, subDir, "FILE2", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":      []string{subDir},
		"--archive": "zip",
		"--name":    "foobar.zip",
	})

	require.Contains(t, result.Stdout, "foobar.zip", "output should contain custom name")

	fileURL := extractFileURLFromOutput(t, result.Stdout)
	dlDir := t.TempDir()
	downloadAndExtractZip(t, fileURL, dlDir, "foobar.zip")

	extractedPath := findExtractedFile(t, dlDir, "FILE1")
	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	require.Equal(t, testContent, string(extracted))
}
