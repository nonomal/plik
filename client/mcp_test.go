package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestClientForProfile_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"
`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path}

	// Empty profile should reload from disk and return a working client
	client, err := clientForProfile(cfg, "")
	require.NoError(t, err)
	require.Equal(t, "https://base.example.com", client.URL)
}

func TestClientForProfile_ValidProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = "work-token-123"
`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path, URL: "https://base.example.com"}

	client, err := clientForProfile(cfg, "work")
	require.NoError(t, err)
	require.Equal(t, "https://work.example.com", client.URL)
	require.Equal(t, "work-token-123", client.Token)
}

func TestClientForProfile_InvalidProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path}

	_, err = clientForProfile(cfg, "typo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "typo")
	require.Contains(t, err.Error(), "not found")
}

func TestClientForProfile_LockedByFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = ""

[Profiles.local]
URL = "http://localhost:8080"
Token = ""
`), 0600)
	require.NoError(t, err)

	// Simulate MCP started with -P work
	cfg := &CliConfig{ConfigPath: path, ActiveProfiles: []string{"work"}, ProfileSource: "flag"}

	// Switching to a different profile should be rejected
	_, err = clientForProfile(cfg, "local")
	require.Error(t, err)
	require.Contains(t, err.Error(), "profile switching is locked by -P work")

	// Empty profile (reload same profile) should still work
	client, err := clientForProfile(cfg, "")
	require.NoError(t, err)
	require.Equal(t, "https://work.example.com", client.URL)
}

func TestClientForProfile_DefaultProfileAllowsSwitch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = ""

[Profiles.local]
URL = "http://localhost:8080"
Token = ""
`), 0600)
	require.NoError(t, err)

	// Simulate MCP started with DefaultProfile (not -P)
	cfg := &CliConfig{ConfigPath: path, ActiveProfiles: []string{"work"}, ProfileSource: "default"}

	// Switching should be allowed when source is "default"
	client, err := clientForProfile(cfg, "local")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080", client.URL)
}

func TestClientForProfile_Composition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = "work-token"

[Profiles.zip]
OneShot = true
`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path}

	// "work,zip" should merge: work URL + work token, zip OneShot
	client, err := clientForProfile(cfg, "work,zip")
	require.NoError(t, err)
	require.Equal(t, "https://work.example.com", client.URL)
	require.Equal(t, "work-token", client.Token)
}

func TestListProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"
DefaultProfile = "work"

[Profiles.work]
URL = "https://work.example.com"
Token = ""

[Profiles.local]
URL = "http://localhost:8080"
Token = ""

[Profiles.zip]
Archive = true
ArchiveMethod = "zip"
`), 0600)
	require.NoError(t, err)

	t.Run("unlocked", func(t *testing.T) {
		// No ActiveProfiles → all profiles visible
		plikrc, _, err := loadPlikrc(path)
		require.NoError(t, err)
		require.Equal(t, "work", plikrc.DefaultProfile)
		require.Len(t, plikrc.Profiles, 3)

		// Verify profile URLs (zip has no URL, inherits base)
		require.Equal(t, "https://work.example.com", plikrc.Profiles["work"].URL)
		require.Equal(t, "http://localhost:8080", plikrc.Profiles["local"].URL)
		require.Equal(t, "", plikrc.Profiles["zip"].URL, "zip has no URL override")
	})

	t.Run("locked_by_flag", func(t *testing.T) {
		// When ProfileSource is "flag", the handler returns empty (no profiles listed)
		cfg := &CliConfig{ConfigPath: path, ActiveProfiles: []string{"work"}, ProfileSource: "flag"}
		handler := makeListProfilesHandler(cfg)
		result, _, err := handler(context.Background(), nil, ListProfilesInput{})
		require.NoError(t, err)
		require.False(t, result.IsError)
		// Should contain empty JSON with null profiles
		text := result.Content[0].(*mcp.TextContent).Text
		require.Contains(t, text, `"profiles": null`)
		require.NotContains(t, text, `"default_profile"`)
	})
}
