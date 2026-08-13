package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCLI_Profile_Upload verifies that profile settings flow through the full
// upload path. A profile that sets OneShot = true should produce server metadata
// with oneShot: true.
func TestCLI_Profile_Upload(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"

[Profiles.ogg]
OneShot = true
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "ogg")
	require.NoError(t, err)

	// Override URL to point at the test server
	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["oneShot"], "profile-set OneShot should be reflected in upload metadata")
}

// TestCLI_Profile_Upload_InheritsBase verifies that profile inheritance works
// through the full upload path. A profile that only overrides URL should
// inherit TTL from the base config.
func TestCLI_Profile_Upload_InheritsBase(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"
TTL = 3600

[Profiles.minimal]
URL = "http://also-overridden.local"
Token = ""
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "minimal")
	require.NoError(t, err)

	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	ttl, ok := meta["ttl"].(float64)
	require.True(t, ok, "ttl should be a number")
	require.Equal(t, float64(3600), ttl, "TTL should be inherited from base config")
}

// TestCLI_Profile_Info verifies that info output includes profile information
// when profiles are configured.
func TestCLI_Profile_Info(t *testing.T) {
	dir := t.TempDir()

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"

[Profiles.local]
URL = "http://localhost:8080"
Token = ""

[Profiles.work]
URL = "https://plik.work.corp"
Token = ""
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "local")
	require.NoError(t, err)

	config.URL = testServerURL

	output := runInfo(t, config)

	require.Contains(t, output, "Active profile")
	require.Contains(t, output, "local")
	require.Contains(t, output, "Available profiles")
	require.Contains(t, output, "work")
}

// TestCLI_Profile_Info_NoProfile verifies that info output does NOT contain
// profile lines when no profiles are defined.
func TestCLI_Profile_Info_NoProfile(t *testing.T) {
	dir := t.TempDir()

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "")
	require.NoError(t, err)

	config.URL = testServerURL

	output := runInfo(t, config)

	require.NotContains(t, output, "Active profile")
	require.NotContains(t, output, "Available profiles")
}

// TestCLI_Profile_DefaultProfile verifies that DefaultProfile is automatically
// selected when no -P flag is given. The default profile sets OneShot = true,
// which should propagate to the upload metadata.
func TestCLI_Profile_DefaultProfile(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"
DefaultProfile = "oneshot"

[Profiles.oneshot]
OneShot = true
`), 0600)
	require.NoError(t, err)

	// Load without explicit profile — DefaultProfile should kick in
	config, err := LoadConfigFromFile(plikrc, "")
	require.NoError(t, err)
	require.Equal(t, []string{"oneshot"}, config.ActiveProfiles, "DefaultProfile should set ActiveProfiles")
	require.True(t, config.OneShot, "profile OneShot should be merged")

	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["oneShot"], "DefaultProfile-set OneShot should reach server")
}

// TestCLI_Profile_DefaultProfile_OverriddenByCLI verifies that an explicit
// profile name overrides DefaultProfile. DefaultProfile sets OneShot = true,
// but loading with profile "plain" (no OneShot) should produce oneShot: false.
func TestCLI_Profile_DefaultProfile_OverriddenByCLI(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"
DefaultProfile = "oneshot"

[Profiles.oneshot]
OneShot = true

[Profiles.plain]
URL = "https://also-overridden.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	// Load with explicit profile — should override DefaultProfile
	config, err := LoadConfigFromFile(plikrc, "plain")
	require.NoError(t, err)
	require.Equal(t, []string{"plain"}, config.ActiveProfiles, "explicit profile should win over DefaultProfile")
	require.False(t, config.OneShot, "plain profile should not inherit OneShot from oneshot profile")

	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, false, meta["oneShot"], "explicit -P should override DefaultProfile")
}

// TestCLI_Profile_SaveToken_RoundTrip writes a sparse config with 3 profiles,
// calls saveToken on one, then re-reads and verifies:
// - token landed in the correct profile
// - other profiles are untouched
// - the file stays sparse (no zero-value bloat)
func TestCLI_Profile_SaveToken_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	plikrc := filepath.Join(dir, ".plikrc")

	original := `URL = "https://plik.example.com"
Token = "base-token"
DefaultProfile = "local"

[Profiles.local]
URL = "http://127.0.0.1:8080"
AutoUpdate = false

[Profiles.work]
URL = "https://plik.work.corp"
Token = "work-token"

[Profiles.zip]
Archive = true
ArchiveMethod = "zip"
`
	err := os.WriteFile(plikrc, []byte(original), 0600)
	require.NoError(t, err)

	// Simulate plik -P local --login completing auth
	cfg := &CliConfig{
		ConfigPath:     plikrc,
		ActiveProfiles: []string{"local"},
	}
	err = saveToken(cfg, "new-local-token")
	require.NoError(t, err)

	// Re-read and verify
	config, err := LoadConfigFromFile(plikrc, "local")
	require.NoError(t, err)
	require.Equal(t, "new-local-token", config.Token, "token should be saved to local profile")

	// Other profiles untouched
	config2, err := LoadConfigFromFile(plikrc, "work")
	require.NoError(t, err)
	require.Equal(t, "work-token", config2.Token, "work profile token should be untouched")

	config3, err := LoadConfigFromFile(plikrc, "zip")
	require.NoError(t, err)
	require.True(t, config3.Archive, "zip profile archive setting should be untouched")

	// Base token untouched
	configBase, err := LoadConfigFromFile(plikrc, "")
	require.NoError(t, err)
	require.Equal(t, "new-local-token", configBase.Token, "DefaultProfile local token should show when loaded via default")

	// File is still sparse — zip profile should not have URL/Token/etc.
	content, err := os.ReadFile(plikrc)
	require.NoError(t, err)
	output := string(content)

	zipStart := findProfileSection(output, "zip")
	require.NotEqual(t, -1, zipStart)
	zipSection := output[zipStart:]
	require.NotContains(t, zipSection, "URL =", "zip profile should not have URL field")
	require.NotContains(t, zipSection, "Token =", "zip profile should not have Token field")
	require.Contains(t, zipSection, "Archive = true")
}

// findProfileSection returns the byte offset of [Profiles.<name>] in output, or -1.
func findProfileSection(output, name string) int {
	return strings.Index(output, "[Profiles."+name+"]")
}

// TestCLI_Profile_Composition_Upload verifies that composing two profiles
// produces the correct merged config (last-wins for overlapping fields,
// non-overlapping fields from each profile survive).
func TestCLI_Profile_Composition_Upload(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"

[Profiles.oneshot]
OneShot = true

[Profiles.removable]
Removable = true
`), 0600)
	require.NoError(t, err)

	// Compose: oneshot first, then removable — both flags should be set
	config, err := LoadConfigFromFile(plikrc, "oneshot,removable")
	require.NoError(t, err)
	require.True(t, config.OneShot, "OneShot from first profile should survive")
	require.True(t, config.Removable, "Removable from second profile should be set")
	require.Equal(t, []string{"oneshot", "removable"}, config.ActiveProfiles)

	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["oneShot"], "composed oneShot should reach server")
	require.Equal(t, true, meta["removable"], "composed removable should reach server")
}

// TestCLI_Profile_Composition_DefaultProfile verifies that DefaultProfile
// supports comma-separated names for composition.
func TestCLI_Profile_Composition_DefaultProfile(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	plikrc := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(plikrc, []byte(`
URL = "https://should-be-overridden.example.com"
DefaultProfile = "oneshot,removable"

[Profiles.oneshot]
OneShot = true

[Profiles.removable]
Removable = true
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(plikrc, "")
	require.NoError(t, err)
	require.True(t, config.OneShot, "OneShot from composed DefaultProfile")
	require.True(t, config.Removable, "Removable from composed DefaultProfile")
	require.Equal(t, []string{"oneshot", "removable"}, config.ActiveProfiles)

	config.URL = testServerURL
	config.Quiet = true

	result := runCLI(t, config, map[string]any{
		"FILE": []string{dir + "/FILE1"},
	})

	meta := getUploadMetadata(t, result.Stdout)
	require.Equal(t, true, meta["oneShot"], "composed oneShot from DefaultProfile should reach server")
	require.Equal(t, true, meta["removable"], "composed removable from DefaultProfile should reach server")
}
