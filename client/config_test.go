package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docopt/docopt-go"
	"github.com/stretchr/testify/require"
)

// makeOpts returns a docopt.Opts map with sensible zero-value defaults
// for every flag/option the CLI declares. Callers can override specific
// keys before passing to UnmarshalArgs.
func makeOpts() docopt.Opts {
	return docopt.Opts{
		"FILE":              []string{},
		"--debug":           false,
		"--quiet":           false,
		"--json":            false,
		"--server":          nil,
		"--name":            nil,
		"--oneshot":         false,
		"--removable":       false,
		"--stream":          false,
		"--ttl":             nil,
		"--extend-ttl":      false,
		"--comments":        nil,
		"-p":                false,
		"--password":        nil,
		"-a":                false,
		"--archive":         nil,
		"--compress":        nil,
		"--archive-options": nil,
		"-s":                false,
		"--not-secure":      false,
		"--secure":          nil,
		"--cipher":          nil,
		"--passphrase":      nil,
		"--recipient":       nil,
		"--secure-options":  nil,
		"-P":                nil,
		"--profile":         nil,
		"--insecure":        false,
		"--update":          false,
		"--login":           false,
		"--update-plikrc":   false,
		"--mcp":             false,
		"--version":         false,
		"--info":            false,
		"--help":            false,
		"--token":           nil,
		"--stdin":           false,
		"--yes":             false,
	}
}

// --- TTL parsing ---

func TestUnmarshalArgs_TTL_Minutes(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--ttl"] = "5m"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, 300, config.TTL) // 5 * 60
}

func TestUnmarshalArgs_TTL_Hours(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--ttl"] = "2h"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, 7200, config.TTL) // 2 * 3600
}

func TestUnmarshalArgs_TTL_Days(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--ttl"] = "1d"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, 86400, config.TTL) // 1 * 86400
}

func TestUnmarshalArgs_TTL_Seconds(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--ttl"] = "3600"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, 3600, config.TTL) // raw seconds
}

func TestUnmarshalArgs_TTL_Invalid(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--ttl"] = "abc"

	err := config.UnmarshalArgs(opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid TTL")
}

// --- Password parsing ---

func TestUnmarshalArgs_Password_LoginPassword(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--password"] = "admin:secret"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "admin", config.Login)
	require.Equal(t, "secret", config.Password)
}

func TestUnmarshalArgs_Password_DefaultLogin(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--password"] = "mysecret"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "plik", config.Login)
	require.Equal(t, "mysecret", config.Password)
}

func TestUnmarshalArgs_Password_ColonInPassword(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--password"] = "user:pass:word"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "user", config.Login)
	require.Equal(t, "pass:word", config.Password)
}

// --- Boolean flags ---

func TestUnmarshalArgs_Flags(t *testing.T) {
	tests := []struct {
		flag  string
		field func(c *CliConfig) bool
		name  string
	}{
		{"--oneshot", func(c *CliConfig) bool { return c.OneShot }, "OneShot"},
		{"--removable", func(c *CliConfig) bool { return c.Removable }, "Removable"},
		{"--stream", func(c *CliConfig) bool { return c.Stream }, "Stream"},
		{"--quiet", func(c *CliConfig) bool { return c.Quiet }, "Quiet"},
		{"--debug", func(c *CliConfig) bool { return c.Debug }, "Debug"},
		{"--extend-ttl", func(c *CliConfig) bool { return c.ExtendTTL }, "ExtendTTL"},
		{"--yes", func(c *CliConfig) bool { return c.Yes }, "Yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewUploadConfig()
			opts := makeOpts()
			opts[tt.flag] = true

			err := config.UnmarshalArgs(opts)
			require.NoError(t, err)
			require.True(t, tt.field(config))
		})
	}
}

func TestUnmarshalArgs_JSON_ImpliesQuiet(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--json"] = true

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.True(t, config.JSON)
	require.True(t, config.Quiet)
}

// --- Server override ---

func TestUnmarshalArgs_ServerOverride(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--server"] = "https://plik.example.com"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "https://plik.example.com", config.URL)
}

func TestUnmarshalArgs_ServerClearsToken(t *testing.T) {
	config := NewUploadConfig()
	config.Token = "should-be-cleared"
	opts := makeOpts()
	opts["--server"] = "https://other.example.com"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "https://other.example.com", config.URL)
	require.Equal(t, "", config.Token, "--server should clear token to prevent leakage")
}

func TestUnmarshalArgs_ServerWithToken(t *testing.T) {
	config := NewUploadConfig()
	config.Token = "should-be-cleared"
	opts := makeOpts()
	opts["--server"] = "https://other.example.com"
	opts["--token"] = "explicit-token"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "https://other.example.com", config.URL)
	require.Equal(t, "explicit-token", config.Token, "--token should override the cleared value")
}

// --- Secure mode ---

func TestUnmarshalArgs_SecureEnabled(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["-s"] = true

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.True(t, config.Secure)
}

func TestUnmarshalArgs_SecureExplicitBackend(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--secure"] = "age"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.True(t, config.Secure)
	require.Equal(t, "age", config.SecureMethod)
}

func TestUnmarshalArgs_NotSecure(t *testing.T) {
	config := NewUploadConfig()
	config.Secure = true // pre-set from config file
	opts := makeOpts()
	opts["--not-secure"] = true

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.False(t, config.Secure)
}

// --- Archive mode ---

func TestUnmarshalArgs_ArchiveShortFlag(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["-a"] = true

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.True(t, config.Archive)
}

func TestUnmarshalArgs_ArchiveExplicitBackend(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--archive"] = "zip"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.True(t, config.Archive)
	require.Equal(t, "zip", config.ArchiveMethod)
}

// --- Token handling ---

func TestUnmarshalArgs_Token(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--token"] = "my-upload-token"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "my-upload-token", config.Token)
}

func TestUnmarshalArgs_StdinOverride(t *testing.T) {
	config := NewUploadConfig()
	config.DisableStdin = true
	opts := makeOpts()
	opts["--stdin"] = true

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.False(t, config.DisableStdin)
}

// --- Config file loading ---

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://upload.example.com"
OneShot = true
TTL = 3600
DownloadBinary = "wget"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "https://upload.example.com", config.URL)
	require.True(t, config.OneShot)
	require.Equal(t, 3600, config.TTL)
	require.Equal(t, "wget", config.DownloadBinary)
}

func TestLoadConfigFromFile_MissingFile(t *testing.T) {
	_, err := LoadConfigFromFile("/nonexistent/plikrc", "")
	require.Error(t, err)
}

func TestLoadConfigFromFile_URLTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://upload.example.com/"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "https://upload.example.com", config.URL, "trailing slash should be stripped")
	require.Equal(t, path, config.ConfigPath, "ConfigPath should be set to the loaded file")
}

// --- NewUploadConfig defaults ---

func TestNewUploadConfig_Defaults(t *testing.T) {
	config := NewUploadConfig()
	require.Equal(t, "http://127.0.0.1:8080", config.URL)
	require.Equal(t, "tar", config.ArchiveMethod)
	require.Equal(t, "age", config.SecureMethod)
	require.Equal(t, "curl", config.DownloadBinary)
	require.False(t, config.Debug)
	require.False(t, config.Quiet)
	require.False(t, config.OneShot)
}

// --- Comments ---

func TestUnmarshalArgs_Comments(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--comments"] = "This is a test upload"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "This is a test upload", config.Comments)
}

// --- Filename override ---

func TestUnmarshalArgs_FilenameOverride(t *testing.T) {
	config := NewUploadConfig()
	opts := makeOpts()
	opts["--name"] = "custom-name.txt"

	err := config.UnmarshalArgs(opts)
	require.NoError(t, err)
	require.Equal(t, "custom-name.txt", config.filenameOverride)
}

// --- Profile tests ---

func TestLoadConfigFromFile_WithProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
Token = "base-token"
OneShot = false

[Profiles.local]
URL = "http://127.0.0.1:8080"
Token = ""

[Profiles.staging]
URL = "https://staging.example.com"
Token = "staging-token"
OneShot = true
`), 0600)
	require.NoError(t, err)

	// Load with "local" profile
	config, err := LoadConfigFromFile(path, "local")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:8080", config.URL)
	require.Equal(t, "", config.Token, "Token should be cleared — profile defined URL with Token = empty")
	require.False(t, config.OneShot, "OneShot should be inherited from base")
	require.Equal(t, []string{"local"}, config.ActiveProfiles)

	// Load with "staging" profile
	config, err = LoadConfigFromFile(path, "staging")
	require.NoError(t, err)
	require.Equal(t, "https://staging.example.com", config.URL)
	require.Equal(t, "staging-token", config.Token)
	require.True(t, config.OneShot)
	require.Equal(t, []string{"staging"}, config.ActiveProfiles)
}

func TestLoadConfigFromFile_ProfileExplicitEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
Token = "base-token"

[Profiles.local]
URL = "http://127.0.0.1:8080"
Token = ""
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "local")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:8080", config.URL)
	require.Equal(t, "", config.Token, "Explicit empty Token in profile should clear base")
}

func TestLoadConfigFromFile_ProfileInheritsBase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
Token = "base-token"
DownloadBinary = "wget"
TTL = 3600
SecureMethod = "pgp"

[Profiles.minimal]
URL = "http://localhost:9090"
Token = "minimal-token"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "minimal")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:9090", config.URL)
	require.Equal(t, "minimal-token", config.Token, "Token should come from profile")
	require.Equal(t, "wget", config.DownloadBinary, "DownloadBinary should be inherited")
	require.Equal(t, 3600, config.TTL, "TTL should be inherited")
	require.Equal(t, "pgp", config.SecureMethod, "SecureMethod should be inherited")
}

func TestLoadConfigFromFile_DefaultProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://default.example.com"
DefaultProfile = "prod"

[Profiles.prod]
URL = "https://prod.example.com"
Token = "prod-token"

[Profiles.dev]
URL = "http://localhost:8080"
Token = ""
`), 0600)
	require.NoError(t, err)

	// No profile specified — should use DefaultProfile
	config, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "https://prod.example.com", config.URL)
	require.Equal(t, "prod-token", config.Token)
	require.Equal(t, []string{"prod"}, config.ActiveProfiles)
}

func TestLoadConfigFromFile_MissingProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"

[Profiles.local]
URL = "http://127.0.0.1:8080"
Token = ""
`), 0600)
	require.NoError(t, err)

	_, err = LoadConfigFromFile(path, "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), `Profile "nonexistent" not found`)
	require.Contains(t, err.Error(), "local")
}

func TestLoadConfigFromFile_MissingProfile_NoProfilesDefined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
`), 0600)
	require.NoError(t, err)

	_, err = LoadConfigFromFile(path, "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), `Profile "nonexistent" not found`)
	require.Contains(t, err.Error(), "no profiles defined")
}

func TestLoadConfigFromFile_NoProfiles_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://upload.example.com"
OneShot = true
TTL = 3600
DownloadBinary = "wget"
`), 0600)
	require.NoError(t, err)

	// No profile — should work exactly as before
	config, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "https://upload.example.com", config.URL)
	require.True(t, config.OneShot)
	require.Equal(t, 3600, config.TTL)
	require.Equal(t, "wget", config.DownloadBinary)
	require.Empty(t, config.ActiveProfiles)
	require.Empty(t, config.AvailableProfiles)
}

func TestValidateProfile_URLWithoutToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"
Token = "base-secret"

[Profiles.leak]
URL = "https://evil.example.com"
`), 0600)
	require.NoError(t, err)

	_, err = LoadConfigFromFile(path, "leak")
	require.Error(t, err)
	require.Contains(t, err.Error(), `"leak"`)
	require.Contains(t, err.Error(), "defines URL but not Token")
}

func TestValidateProfile_URLWithEmptyToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"
Token = "base-secret"

[Profiles.anon]
URL = "https://public.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "anon")
	require.NoError(t, err)
	require.Equal(t, "https://public.example.com", config.URL)
	require.Equal(t, "", config.Token, "empty token should not inherit base token")
}

func TestValidateProfile_NoURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"
Token = "base-secret"

[Profiles.zip]
Archive = true
ArchiveMethod = "zip"
`), 0600)
	require.NoError(t, err)

	// Profile without URL should pass validation — no risk of token leakage
	config, err := LoadConfigFromFile(path, "zip")
	require.NoError(t, err)
	require.Equal(t, "https://base.example.com", config.URL)
	require.Equal(t, "base-secret", config.Token, "token should be inherited when URL is not changed")
}

func TestLoadConfig_EnvProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
Token = "base-token"

[Profiles.envtest]
URL = "http://env-test.local:8080"
Token = "env-token"
`), 0600)
	require.NoError(t, err)

	// Set PLIK_PROFILE env var
	t.Setenv("PLIK_PROFILE", "envtest")
	t.Setenv("PLIKRC", path)

	opts := makeOpts()
	opts["--quiet"] = true
	config, err := LoadConfig(opts)
	require.NoError(t, err)
	require.Equal(t, "http://env-test.local:8080", config.URL)
	require.Equal(t, "env-token", config.Token)
	require.Equal(t, []string{"envtest"}, config.ActiveProfiles)
	require.Equal(t, "env", config.ProfileSource)
}

func TestProfileSource_Flag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
[Profiles.work]
URL = "https://work.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	t.Setenv("PLIKRC", path)
	opts := makeOpts()
	opts["--quiet"] = true
	opts["--profile"] = "work"
	config, err := LoadConfig(opts)
	require.NoError(t, err)
	require.Equal(t, "flag", config.ProfileSource)
	require.Equal(t, []string{"work"}, config.ActiveProfiles)
}

func TestProfileSource_Default(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
DefaultProfile = "work"
[Profiles.work]
URL = "https://work.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "default", config.ProfileSource)
	require.Equal(t, []string{"work"}, config.ActiveProfiles)
}

func TestProfileSource_None(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "", config.ProfileSource)
	require.Empty(t, config.ActiveProfiles)
}

func TestLoadConfigFromFile_AvailableProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"

[Profiles.alpha]
URL = "http://alpha.local"

[Profiles.beta]
URL = "http://beta.local"
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Len(t, config.AvailableProfiles, 2)
	require.Contains(t, config.AvailableProfiles, "alpha")
	require.Contains(t, config.AvailableProfiles, "beta")
}

func TestLoadConfigFromFile_ProfileBooleanOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"
OneShot = false
Stream = false

[Profiles.ogg]
OneShot = true
Stream = true
`), 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(path, "ogg")
	require.NoError(t, err)
	require.True(t, config.OneShot, "Profile should enable OneShot")
	require.True(t, config.Stream, "Profile should enable Stream")
}

func TestLoadConfigFromFile_ProfileArchiveOptionsFullOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"

[ArchiveOptions]
  Compress = "gzip"
  Tar = "/bin/tar"

[Profiles.custom]
URL = "http://localhost:8080"
Token = ""

[Profiles.custom.ArchiveOptions]
  Compress = "xz"
`), 0600)
	require.NoError(t, err)

	// Profile overrides ArchiveOptions — full replacement, NOT merge
	config, err := LoadConfigFromFile(path, "custom")
	require.NoError(t, err)
	require.Equal(t, "xz", config.ArchiveOptions["Compress"], "Profile should override Compress")
	require.Nil(t, config.ArchiveOptions["Tar"], "Tar should NOT be inherited (full override)")
}

func TestLoadConfigFromFile_ProfileArchiveOptionsInherited(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"

[ArchiveOptions]
  Compress = "gzip"
  Tar = "/bin/tar"

[Profiles.minimal]
URL = "http://localhost:8080"
Token = ""
`), 0600)
	require.NoError(t, err)

	// Profile does NOT define ArchiveOptions — should inherit from base
	config, err := LoadConfigFromFile(path, "minimal")
	require.NoError(t, err)
	require.Equal(t, "gzip", config.ArchiveOptions["Compress"], "Compress should be inherited")
	require.Equal(t, "/bin/tar", config.ArchiveOptions["Tar"], "Tar should be inherited")
}

func TestLoadConfigFromFile_ProfileSecureOptionsFullOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://plik.root.gg"

[SecureOptions]
  Cipher = "aes-256-cbc"
  Passphrase = "secret"

[Profiles.custom]
URL = "http://localhost:8080"
Token = ""

[Profiles.custom.SecureOptions]
  Cipher = "chacha20"
`), 0600)
	require.NoError(t, err)

	// Profile overrides SecureOptions — full replacement, NOT merge
	config, err := LoadConfigFromFile(path, "custom")
	require.NoError(t, err)
	require.Equal(t, "chacha20", config.SecureOptions["Cipher"], "Profile should override Cipher")
	require.Nil(t, config.SecureOptions["Passphrase"], "Passphrase should NOT be inherited (full override)")
}

// ---------- writeConfig tests ----------

func TestWriteConfig_Structure(t *testing.T) {
	config := NewUploadConfig()
	config.URL = "https://plik.example.com"
	config.Token = "test-token"
	config.AutoUpdate = true

	plikrc := &PlikrcFile{CliConfig: *config}
	buf := new(bytes.Buffer)
	err := writeConfig(buf, plikrc)
	require.NoError(t, err)

	output := buf.String()

	// Verify section headers are present and in order
	sections := []string{
		"# --- Server ---",
		"# --- Upload defaults ---",
		"# --- Authentication ---",
		"# --- Archive ---",
		"# --- Encryption ---",
		"# --- Output ---",
		"# --- Behavior ---",
	}
	lastIdx := -1
	for _, section := range sections {
		idx := strings.Index(output, section)
		require.NotEqual(t, -1, idx, "missing section: %s", section)
		require.Greater(t, idx, lastIdx, "section %q should be after previous section", section)
		lastIdx = idx
	}

	// Verify inline comments are present
	require.Contains(t, output, "# URL of the plik server")
	require.Contains(t, output, "# Authentication token")
	require.Contains(t, output, "# Auto-update client binary")

	// Verify values are written correctly
	require.Contains(t, output, `URL = "https://plik.example.com"`)
	require.Contains(t, output, `Token = "test-token"`)
	require.Contains(t, output, `AutoUpdate = true`)
}

func TestWriteConfig_EmptySecureOptionsOmitted(t *testing.T) {
	config := NewUploadConfig()
	plikrc := &PlikrcFile{CliConfig: *config}

	buf := new(bytes.Buffer)
	err := writeConfig(buf, plikrc)
	require.NoError(t, err)

	output := buf.String()

	// Empty SecureOptions should NOT produce an active [SecureOptions] block
	// but SHOULD produce a commented-out reference section with all backend keys
	require.Contains(t, output, "# [SecureOptions]")
	require.Contains(t, output, "#   Passphrase")
	require.Contains(t, output, "#   Recipient")
	require.Contains(t, output, "#   Cipher")
	require.Contains(t, output, "#   Options")
	require.Contains(t, output, "#   Openssl")
	require.Contains(t, output, "#   Keyring")

	// ArchiveOptions with values should produce an active block
	require.Contains(t, output, "[ArchiveOptions]")
	require.Contains(t, output, `Compress = "gzip"`)

	// No active profiles → should produce commented-out profile examples
	require.Contains(t, output, "# [Profiles.local]")
	require.Contains(t, output, "# [Profiles.work]")
	require.Contains(t, output, "# [Profiles.zip]")
	require.Contains(t, output, "# [Profiles.bob]")
	require.Contains(t, output, `# Recipient = "@bob"`)
}

func TestWriteConfig_RoundTrip(t *testing.T) {
	// Write a config, then load it back and verify values are preserved
	config := NewUploadConfig()
	config.URL = "https://plik.example.com"
	config.Token = "round-trip-token"
	config.OneShot = true
	config.TTL = 7200
	config.AutoUpdate = false
	config.Quiet = true
	config.JSON = true
	config.Yes = true

	plikrc := &PlikrcFile{CliConfig: *config}

	buf := new(bytes.Buffer)
	err := writeConfig(buf, plikrc)
	require.NoError(t, err)

	// Write to a temp file and load it back
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err = os.WriteFile(path, buf.Bytes(), 0600)
	require.NoError(t, err)

	loaded, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)

	require.Equal(t, "https://plik.example.com", loaded.URL)
	require.Equal(t, "round-trip-token", loaded.Token)
	require.True(t, loaded.OneShot)
	require.Equal(t, 7200, loaded.TTL)
	require.False(t, loaded.AutoUpdate)
	require.True(t, loaded.Quiet)
	require.True(t, loaded.JSON)
	require.True(t, loaded.Yes)
	require.Equal(t, "gzip", loaded.ArchiveOptions["Compress"])
	require.Equal(t, "/bin/tar", loaded.ArchiveOptions["Tar"])
}

func TestUpdatePlikrc_RoundTrip(t *testing.T) {
	// Write a hand-crafted config with user comments and a non-alphabetical
	// profile order, then call updatePlikrc and verify it comes out in canonical
	// format with all values preserved.
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	original := `# hand-crafted config with my own comments
URL = "https://plik.example.com"  # my server
Token = "my-token"
OneShot = true

[Profiles.work]
URL = "https://work.example.com"
Token = "work-token"

[Profiles.alpha]
URL = "http://alpha.local"
Token = ""
`
	err := os.WriteFile(path, []byte(original), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{
		ConfigPath: path,
		Yes:        true, // skip confirmation prompt
	}
	err = updatePlikrc(cfg)
	require.NoError(t, err)

	// Values must be preserved
	loaded, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "https://plik.example.com", loaded.URL)
	require.Equal(t, "my-token", loaded.Token)
	require.True(t, loaded.OneShot)

	// Profiles must be preserved
	work, err := LoadConfigFromFile(path, "work")
	require.NoError(t, err)
	require.Equal(t, "https://work.example.com", work.URL)
	require.Equal(t, "work-token", work.Token)

	alpha, err := LoadConfigFromFile(path, "alpha")
	require.NoError(t, err)
	require.Equal(t, "http://alpha.local", alpha.URL)
	require.Equal(t, "", alpha.Token)

	// Output is now in canonical format (has standard section headers)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	output := string(content)
	require.Contains(t, output, "# --- Server ---")
	require.Contains(t, output, "# --- Profiles ---")
	// Original custom comment is gone (replaced by canonical format)
	require.NotContains(t, output, "hand-crafted config")
}

func TestUpdatePlikrc_FileNotFound(t *testing.T) {
	cfg := &CliConfig{
		ConfigPath: "/nonexistent/path/.plikrc",
		Yes:        true,
	}
	err := updatePlikrc(cfg)
	require.Error(t, err)
}

func TestWriteConfig_WithProfiles(t *testing.T) {
	config := NewUploadConfig()
	config.URL = "https://plik.example.com"
	config.DefaultProfile = "local"

	profile := CliConfig{
		URL:   "http://localhost:8080",
		Token: "local-token",
	}

	plikrc := &PlikrcFile{
		CliConfig: *config,
		Profiles: map[string]CliConfig{
			"local": profile,
		},
	}

	buf := new(bytes.Buffer)
	err := writeConfig(buf, plikrc)
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, `DefaultProfile = "local"`)
	require.Contains(t, output, "[Profiles.local]")
	require.Contains(t, output, `URL = "http://localhost:8080"`)

	// Round-trip: write then load
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err = os.WriteFile(path, buf.Bytes(), 0600)
	require.NoError(t, err)

	loaded, err := LoadConfigFromFile(path, "local")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080", loaded.URL)
	require.Equal(t, "local-token", loaded.Token)
}

func TestPlikrcTemplate_UpToDate(t *testing.T) {
	// Generate the canonical .plikrc template from code
	buf := new(bytes.Buffer)
	err := WritePlikrcTemplate(buf)
	require.NoError(t, err)

	// Write the generated template to client/.plikrc.
	// CI catches drift via `git diff --exit-code` after running tests.
	committedPath := filepath.Join(".", ".plikrc")
	err = os.WriteFile(committedPath, buf.Bytes(), 0644)
	require.NoError(t, err, "unable to write client/.plikrc")

	// Read back and verify the file was written correctly
	written, err := os.ReadFile(committedPath)
	require.NoError(t, err)
	require.Equal(t, buf.String(), string(written), "client/.plikrc content mismatch after write")
}

func TestWriteConfig_SparseProfiles(t *testing.T) {
	// A profile with just Archive=true and ArchiveMethod="zip" should
	// produce only those fields — not all 23 CliConfig zero-value fields.
	config := NewUploadConfig()
	config.URL = "https://plik.example.com"

	zipProfile := CliConfig{
		Archive:       true,
		ArchiveMethod: "zip",
	}

	plikrc := &PlikrcFile{
		CliConfig: *config,
		Profiles: map[string]CliConfig{
			"zip": zipProfile,
		},
	}

	buf := new(bytes.Buffer)
	err := writeConfig(buf, plikrc)
	require.NoError(t, err)

	output := buf.String()

	// Verify the profile header is present
	require.Contains(t, output, "[Profiles.zip]")

	// Verify only the non-zero fields are written
	require.Contains(t, output, "Archive = true")
	require.Contains(t, output, `ArchiveMethod = "zip"`)

	// Verify zero-value fields are NOT written in the profile section
	// Extract just the profile section for focused assertions
	profileStart := strings.Index(output, "[Profiles.zip]")
	require.NotEqual(t, -1, profileStart)
	profileSection := output[profileStart:]

	// These zero-value fields should NOT appear in the profile
	require.NotContains(t, profileSection, "URL =")
	require.NotContains(t, profileSection, "Token =")
	require.NotContains(t, profileSection, "OneShot =")
	require.NotContains(t, profileSection, "Debug =")
	require.NotContains(t, profileSection, "AutoUpdate =")

	// Round-trip: verify it still loads and works
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err = os.WriteFile(path, buf.Bytes(), 0600)
	require.NoError(t, err)

	loaded, err := LoadConfigFromFile(path, "zip")
	require.NoError(t, err)
	require.True(t, loaded.Archive)
	require.Equal(t, "zip", loaded.ArchiveMethod)
	// Inherited from base config
	require.Equal(t, "https://plik.example.com", loaded.URL)
}

func TestSaveToken_PreservesProfiles(t *testing.T) {
	// Write a config with a sparse "zip" profile, then call saveToken
	// with a different profile ("zip") and verify the profile stays sparse.
	config := NewUploadConfig()
	config.URL = "https://plik.example.com"
	config.Token = "original-token"

	zipProfile := CliConfig{
		Archive:       true,
		ArchiveMethod: "zip",
	}

	plikrc := &PlikrcFile{
		CliConfig: *config,
		Profiles: map[string]CliConfig{
			"zip": zipProfile,
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := saveConfig(path, plikrc)
	require.NoError(t, err)

	// Now saveToken with profile "zip"
	cfg := &CliConfig{
		ConfigPath:     path,
		ActiveProfiles: []string{"zip"},
	}
	err = saveToken(cfg, "new-zip-token")
	require.NoError(t, err)

	// Read back the file and verify the profile is still sparse
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	output := string(content)

	// Profile should have the new token and the archive fields
	profileStart := strings.Index(output, "[Profiles.zip]")
	require.NotEqual(t, -1, profileStart)
	profileSection := output[profileStart:]

	require.Contains(t, profileSection, `Token = "new-zip-token"`)
	require.Contains(t, profileSection, "Archive = true")
	require.Contains(t, profileSection, `ArchiveMethod = "zip"`)

	// Zero-value fields should NOT appear
	require.NotContains(t, profileSection, "OneShot =")
	require.NotContains(t, profileSection, "Debug =")
	require.NotContains(t, profileSection, "AutoUpdate =")

	// Top-level token should be preserved
	require.Contains(t, output, `Token = "original-token"`)
}

func TestSaveToken_ProfileNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	err := os.WriteFile(path, []byte(`
URL = "https://plik.example.com"

[Profiles.local]
URL = "http://127.0.0.1:8080"
`), 0600)
	require.NoError(t, err)

	before, err := os.ReadFile(path)
	require.NoError(t, err)

	cfg := &CliConfig{
		ConfigPath:     path,
		ActiveProfiles: []string{"nonexistent"},
	}
	err = saveToken(cfg, "some-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent")

	// File must be unchanged
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after), "file should not be modified on error")
}

func TestSaveToken_NoProfilesDefined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	err := os.WriteFile(path, []byte(`URL = "https://plik.example.com"`), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{
		ConfigPath:     path,
		ActiveProfiles: []string{"local"},
	}
	err = saveToken(cfg, "some-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), `"local"`)
	require.Contains(t, err.Error(), "not found")
}

func TestSaveToken_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	original := `# My custom plik config
URL = "https://plik.root.gg"    # production server
Token = "old-token"             # will be replaced
Insecure = false                # keep TLS on

# Upload settings I like
OneShot = true

[Profiles.work]
# Work server with special config
URL = "https://plik.work.corp"
Token = "work-token"   # auto-generated
`
	err := os.WriteFile(path, []byte(original), 0600)
	require.NoError(t, err)

	// Patch top-level token
	cfg := &CliConfig{ConfigPath: path}
	err = saveToken(cfg, "new-top-level-token")
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	output := string(content)

	// All user comments must survive
	require.Contains(t, output, "# My custom plik config")
	require.Contains(t, output, "# production server")
	require.Contains(t, output, "# keep TLS on")
	require.Contains(t, output, "# Upload settings I like")
	require.Contains(t, output, "# Work server with special config")
	require.Contains(t, output, "# auto-generated")

	// Token value must be updated
	require.Contains(t, output, `Token = "new-top-level-token"`)

	// Work token must be untouched
	require.Contains(t, output, `Token = "work-token"`)

	// Inline comment on the Token line is preserved (trailing content after the value)
	// The inline comment changes because the value length changed, but trailing
	// content after the quoted value is preserved by replaceTokenValue.
}

func TestSaveToken_PreservesProfileOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	// Deliberately non-alphabetical order: work > alpha > local
	original := `URL = "https://plik.root.gg"
Token = "base-token"

[Profiles.work]
URL = "https://plik.work.corp"
Token = "work-token"

[Profiles.alpha]
URL = "https://alpha.example.com"
Token = ""

[Profiles.local]
URL = "http://127.0.0.1:8080"
Token = "local-token"
`
	err := os.WriteFile(path, []byte(original), 0600)
	require.NoError(t, err)

	// Patch the alpha profile
	cfg := &CliConfig{
		ConfigPath:     path,
		ActiveProfiles: []string{"alpha"},
	}
	err = saveToken(cfg, "new-alpha-token")
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	output := string(content)

	// Profile order must be preserved: work > alpha > local
	workIdx := strings.Index(output, "[Profiles.work]")
	alphaIdx := strings.Index(output, "[Profiles.alpha]")
	localIdx := strings.Index(output, "[Profiles.local]")

	require.NotEqual(t, -1, workIdx)
	require.NotEqual(t, -1, alphaIdx)
	require.NotEqual(t, -1, localIdx)
	require.Less(t, workIdx, alphaIdx, "work should come before alpha")
	require.Less(t, alphaIdx, localIdx, "alpha should come before local")

	// Token updated in the right place
	require.Contains(t, output, `Token = "new-alpha-token"`)
	// Other tokens untouched
	require.Contains(t, output, `Token = "work-token"`)
	require.Contains(t, output, `Token = "local-token"`)
}

func TestSaveToken_InsertsTokenWhenMissing_Profile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	original := `URL = "https://plik.root.gg"

[Profiles.zip]
Archive = true
ArchiveMethod = "zip"
`
	err := os.WriteFile(path, []byte(original), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{
		ConfigPath:     path,
		ActiveProfiles: []string{"zip"},
	}
	err = saveToken(cfg, "zip-token")
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	output := string(content)

	// Token should be inserted and the file should be valid TOML
	require.Contains(t, output, `Token = "zip-token"`)

	// Verify the file round-trips through TOML
	loaded, err := LoadConfigFromFile(path, "zip")
	require.NoError(t, err)
	require.Equal(t, "zip-token", loaded.Token)
	require.True(t, loaded.Archive)
	require.Equal(t, "zip", loaded.ArchiveMethod)
}

func TestSaveToken_InsertsTokenAfterURL_Profile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	// Profile with URL but no Token line
	original := `URL = "https://plik.root.gg"

[Profiles.work]
URL = "https://work.example.com"
OneShot = true
`
	err := os.WriteFile(path, []byte(original), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{
		ConfigPath:     path,
		ActiveProfiles: []string{"work"},
	}
	err = saveToken(cfg, "work-token")
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(string(content), "\n")

	// Find positions of URL and Token within the profile section
	urlIdx, tokenIdx := -1, -1
	inProfile := false
	for i, line := range lines {
		if strings.TrimSpace(line) == "[Profiles.work]" {
			inProfile = true
			continue
		}
		if inProfile && strings.HasPrefix(strings.TrimSpace(line), "[") {
			break // next section
		}
		if inProfile && strings.Contains(line, "URL =") {
			urlIdx = i
		}
		if inProfile && strings.Contains(line, "Token =") {
			tokenIdx = i
		}
	}

	require.NotEqual(t, -1, urlIdx, "URL line should exist in profile")
	require.NotEqual(t, -1, tokenIdx, "Token line should exist in profile")
	require.Equal(t, urlIdx+1, tokenIdx, "Token should be inserted immediately after URL")

	// Verify the file round-trips through TOML
	loaded, err := LoadConfigFromFile(path, "work")
	require.NoError(t, err)
	require.Equal(t, "work-token", loaded.Token)
	require.Equal(t, "https://work.example.com", loaded.URL)
	require.True(t, loaded.OneShot)
}

func TestSaveToken_InsertsTokenWhenMissing_TopLevel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	// Config with no Token line at all
	original := `URL = "https://plik.root.gg"
OneShot = true
`
	err := os.WriteFile(path, []byte(original), 0600)
	require.NoError(t, err)

	cfg := &CliConfig{ConfigPath: path}
	err = saveToken(cfg, "brand-new-token")
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	output := string(content)

	require.Contains(t, output, `Token = "brand-new-token"`)

	// Verify the file is valid TOML
	loaded, err := LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, "brand-new-token", loaded.Token)
	require.Equal(t, "https://plik.root.gg", loaded.URL)
	require.True(t, loaded.OneShot)
}

func TestDefaultProfile_ResolutionPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	err := os.WriteFile(path, []byte(`
URL = "https://plik.example.com"
DefaultProfile = "local"

[Profiles.local]
OneShot = true

[Profiles.work]
OneShot = false
`), 0600)
	require.NoError(t, err)

	// (a) Explicit profile wins over DefaultProfile
	config, err := LoadConfigFromFile(path, "work")
	require.NoError(t, err)
	require.Equal(t, []string{"work"}, config.ActiveProfiles)
	require.False(t, config.OneShot, "explicit -P work should win")

	// (b) PLIK_PROFILE env var wins over DefaultProfile (when no explicit flag)
	t.Setenv("PLIK_PROFILE", "work")
	config, err = LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, []string{"work"}, config.ActiveProfiles)
	require.False(t, config.OneShot, "PLIK_PROFILE env var should win over DefaultProfile")
	t.Setenv("PLIK_PROFILE", "") // unset for step (c) — t.Setenv restores at test end, not mid-test

	// (c) DefaultProfile used when no flag and no env var
	config, err = LoadConfigFromFile(path, "")
	require.NoError(t, err)
	require.Equal(t, []string{"local"}, config.ActiveProfiles)
	require.True(t, config.OneShot, "DefaultProfile should be used when nothing else set")
}

func TestLoadConfigFromFile_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")

	err := os.WriteFile(path, []byte(`
URL = "https://plik.example.com"
this is not valid toml !!!
`), 0600)
	require.NoError(t, err)

	_, err = LoadConfigFromFile(path, "")
	require.Error(t, err, "invalid TOML should return error")
}

// ─── Profile composition tests ──────────────────────────────────────────────

func TestParseProfiles_Basic(t *testing.T) {
	require.Equal(t, []string{"work", "zip"}, parseProfiles("work,zip"))
}

func TestParseProfiles_Single(t *testing.T) {
	require.Equal(t, []string{"work"}, parseProfiles("work"))
}

func TestParseProfiles_Dedup(t *testing.T) {
	require.Equal(t, []string{"work", "zip"}, parseProfiles("work,work,zip"))
}

func TestParseProfiles_EmptySegments(t *testing.T) {
	require.Equal(t, []string{"work", "zip"}, parseProfiles(",work,,zip,"))
}

func TestParseProfiles_Whitespace(t *testing.T) {
	require.Equal(t, []string{"work", "zip"}, parseProfiles("work, zip"))
}

func TestComposition_OverrideOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"
Token = "base-token"

[Profiles.work]
URL = "https://work.example.com"
Token = "work-token"
OneShot = true

[Profiles.zip]
URL = "https://zip.example.com"
Token = "zip-token"
Archive = true
ArchiveMethod = "zip"
`), 0600)
	require.NoError(t, err)

	// work, then zip — zip.URL wins, but work.OneShot survives
	config, err := LoadConfigFromFile(path, "work,zip")
	require.NoError(t, err)
	require.Equal(t, "https://zip.example.com", config.URL, "zip URL should win (last)")
	require.Equal(t, "zip-token", config.Token, "zip Token should win (last)")
	require.True(t, config.OneShot, "work OneShot should survive")
	require.True(t, config.Archive, "zip Archive should be set")
	require.Equal(t, "zip", config.ArchiveMethod)
	require.Equal(t, []string{"work", "zip"}, config.ActiveProfiles)
}

func TestComposition_ReverseOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = "work-token"

[Profiles.zip]
URL = "https://zip.example.com"
Token = "zip-token"
`), 0600)
	require.NoError(t, err)

	// zip first, then work — work.URL wins
	config, err := LoadConfigFromFile(path, "zip,work")
	require.NoError(t, err)
	require.Equal(t, "https://work.example.com", config.URL)
	require.Equal(t, []string{"zip", "work"}, config.ActiveProfiles)
}

func TestComposition_InvalidProfileInChain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	_, err = LoadConfigFromFile(path, "work,typo,zip")
	require.Error(t, err)
	require.Contains(t, err.Error(), "typo")
}

func TestSaveToken_RejectsMultipleProfiles(t *testing.T) {
	cfg := &CliConfig{
		ActiveProfiles: []string{"work", "zip"},
	}
	err := saveToken(cfg, "some-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "single profile")
}

// TestComposition_CommaInProfileNameRejected verifies that a profile name
// containing a comma is treated as composition: "us,east" becomes ["us", "east"].
// TOML itself also rejects commas in unquoted table headers, so a profile named
// "us,east" cannot exist in a .plikrc file.
func TestComposition_CommaInProfileNameRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".plikrc")
	err := os.WriteFile(path, []byte(`
URL = "https://base.example.com"

[Profiles.work]
URL = "https://work.example.com"
Token = ""
`), 0600)
	require.NoError(t, err)

	// "us,east" is treated as composition "us" + "east", neither of which exist
	_, err = LoadConfigFromFile(path, "us,east")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
