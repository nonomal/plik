package main

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/docopt/docopt-go"
	"github.com/mitchellh/go-homedir"

	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

// CliConfig holds all CLI client configuration fields.
// Fields are grouped logically; this order determines TOML serialization order.
type CliConfig struct {
	// --- Server ---
	URL      string // Plik server URL
	Token    string // Authentication token (created via web UI or --login)
	Insecure bool   // Skip TLS certificate validation

	// --- Upload defaults ---
	OneShot   bool   // Delete file after first download (if available server side)
	Removable bool   // Allow anyone to delete the file (if available server side)
	Stream    bool   // Block until remote user starts downloading (if available server side)
	TTL       int    // Upload time-to-live in seconds (0 = server default)
	ExtendTTL bool   // Extend expiration on access (if available server side)
	Comments  string // Default upload comments (Markdown)

	// --- Authentication ---
	Login    string // HTTP basic auth login
	Password string // HTTP basic auth password

	// --- Archive ---
	Archive        bool           // Archive files before upload
	ArchiveMethod  string         // Archive backend: tar | zip
	ArchiveOptions map[string]any // Backend-specific options (Tar, Compress, Options)

	// --- Encryption ---
	Secure        bool           // Encrypt files before upload
	SecureMethod  string         // Crypto backend: age | openssl | pgp
	SecureOptions map[string]any // Backend-specific options (Passphrase, Cipher, etc.)

	// --- Output ---
	Debug          bool   // Verbose debug output
	Quiet          bool   // Suppress non-essential output
	JSON           bool   // Output upload metadata as JSON (implies Quiet)
	DownloadBinary string // Download command for output: curl | wget

	// --- Behavior ---
	AutoUpdate     bool   // Auto-update client binary from server
	DisableStdin   bool   // Disable STDIN pipe input by default
	Yes            bool   // Auto-accept confirmation prompts (non-interactive)
	DefaultProfile string `profile:"-"` // Default profile name

	// --- Runtime (not serialized, not merged from profiles) ---
	ConfigPath        string   `toml:"-" profile:"-"` // Path to the loaded config file
	ActiveProfiles    []string `toml:"-" profile:"-"` // Resolved profile name(s) (set after loading)
	AvailableProfiles []string `toml:"-" profile:"-"` // All profile names from config
	ProfileSource     string   `toml:"-" profile:"-"` // How profile was selected: "flag", "env", "default", ""

	filePaths        []string // Upload file paths (from CLI args)
	filenameOverride string   // Filename override (--name flag)
}

// PlikrcFile is the on-disk representation of .plikrc.
// It embeds CliConfig for the top-level (default) fields and adds optional
// named profiles and a default profile selector.
type PlikrcFile struct {
	CliConfig
	Profiles map[string]CliConfig `toml:"Profiles,omitempty"`

	metadata toml.MetaData // unexported; set after DecodeFile, used by writeProfileSection
}

// NewUploadConfig construct a new configuration with default values
func NewUploadConfig() (config *CliConfig) {
	config = new(CliConfig)

	// Server
	config.URL = "http://127.0.0.1:8080"

	// Archive
	config.ArchiveMethod = "tar"
	config.ArchiveOptions = map[string]any{
		"Compress": "gzip",
		"Tar":      "/bin/tar",
		"Options":  "",
	}

	// Encryption
	config.SecureMethod = "age"
	config.SecureOptions = make(map[string]any)

	// Output
	config.DownloadBinary = "curl"

	return
}

// configLine writes a TOML key-value pair with an inline comment, padded to
// align comments across lines. Uses a minimum column width of 32 characters
// for the key-value part, but always ensures at least one space before the comment.
func configLine(w io.Writer, kv, comment string) {
	const minCol = 32
	pad := max(minCol-len(kv), 1)
	fmt.Fprintf(w, "%s%*s# %s\n", kv, pad, "", comment)
}

// writeConfig writes a PlikrcFile as human-readable, commented TOML.
// This produces the same format as the .plikrc template, with logical grouping
// and inline comments. Used by both the first-run wizard and saveToken.
//
// TOML requires all bare key-value pairs (scalars) before any [Table] sections.
// Once a [Table] header appears, all subsequent bare keys belong to that table
// until a new section starts. Therefore we write: scalars → tables → profiles.
func writeConfig(w io.Writer, plikrc *PlikrcFile) error {
	c := &plikrc.CliConfig

	// ── Scalar fields (must come before any [Table] section) ──

	// --- Server ---
	fmt.Fprintf(w, "# --- Server ---\n")
	configLine(w, fmt.Sprintf("URL = %q", c.URL), "URL of the plik server")
	configLine(w, fmt.Sprintf("Token = %q", c.Token), "Authentication token (created via web UI or --login)")
	configLine(w, fmt.Sprintf("Insecure = %t", c.Insecure), "Skip TLS certificate validation")
	fmt.Fprintln(w)

	// --- Upload defaults ---
	fmt.Fprintf(w, "# --- Upload defaults ---\n")
	configLine(w, fmt.Sprintf("OneShot = %t", c.OneShot), "Delete file after first download (if available server side)")
	configLine(w, fmt.Sprintf("Removable = %t", c.Removable), "Allow anyone to delete the file (if available server side)")
	configLine(w, fmt.Sprintf("Stream = %t", c.Stream), "Stream upload, blocks until download starts (if available server side)")
	configLine(w, fmt.Sprintf("TTL = %d", c.TTL), "Upload time-to-live in seconds (0 = server default)")
	configLine(w, fmt.Sprintf("ExtendTTL = %t", c.ExtendTTL), "Extend expiration on access (if available server side)")
	configLine(w, fmt.Sprintf("Comments = %q", c.Comments), "Default upload comments (Markdown)")
	fmt.Fprintln(w)

	// --- Authentication ---
	fmt.Fprintf(w, "# --- Authentication ---\n")
	configLine(w, fmt.Sprintf("Login = %q", c.Login), "HTTP basic auth login")
	configLine(w, fmt.Sprintf("Password = %q", c.Password), "HTTP basic auth password")
	fmt.Fprintln(w)

	// --- Archive (scalars only) ---
	fmt.Fprintf(w, "# --- Archive ---\n")
	configLine(w, fmt.Sprintf("Archive = %t", c.Archive), "Archive files before upload")
	configLine(w, fmt.Sprintf("ArchiveMethod = %q", c.ArchiveMethod), "Archive backend (tar | zip)")
	fmt.Fprintln(w)

	// --- Encryption (scalars only) ---
	fmt.Fprintf(w, "# --- Encryption ---\n")
	configLine(w, fmt.Sprintf("Secure = %t", c.Secure), "Encrypt files before upload")
	configLine(w, fmt.Sprintf("SecureMethod = %q", c.SecureMethod), "Crypto backend (age | openssl | pgp)")
	fmt.Fprintln(w)

	// --- Output ---
	fmt.Fprintf(w, "# --- Output ---\n")
	configLine(w, fmt.Sprintf("Debug = %t", c.Debug), "Verbose debug output")
	configLine(w, fmt.Sprintf("Quiet = %t", c.Quiet), "Suppress non-essential output")
	configLine(w, fmt.Sprintf("JSON = %t", c.JSON), "Output upload metadata as JSON (implies Quiet)")
	configLine(w, fmt.Sprintf("DownloadBinary = %q", c.DownloadBinary), "Download command for output (curl | wget)")
	fmt.Fprintln(w)

	// --- Behavior ---
	fmt.Fprintf(w, "# --- Behavior ---\n")
	configLine(w, fmt.Sprintf("AutoUpdate = %t", c.AutoUpdate), "Auto-update client binary from server")
	configLine(w, fmt.Sprintf("DisableStdin = %t", c.DisableStdin), "Disable STDIN pipe input by default")
	configLine(w, fmt.Sprintf("Yes = %t", c.Yes), "Auto-accept confirmation prompts (non-interactive)")
	configLine(w, fmt.Sprintf("DefaultProfile = %q", c.DefaultProfile),
		"Default profile to use (can also be set via PLIK_PROFILE env var)")
	fmt.Fprintln(w)

	// ── Table sections (must come after all scalars) ──

	// [ArchiveOptions]
	if len(c.ArchiveOptions) > 0 {
		fmt.Fprintf(w, "[ArchiveOptions]\n")
		writeMapSection(w, c.ArchiveOptions)
		fmt.Fprintln(w)
	}

	// [SecureOptions]
	if len(c.SecureOptions) > 0 {
		fmt.Fprintf(w, "[SecureOptions]\n")
		writeMapSection(w, c.SecureOptions)
		fmt.Fprintln(w)
	}

	// Commented-out [SecureOptions] reference (when no active SecureOptions)
	if len(c.SecureOptions) == 0 {
		fmt.Fprintf(w, "# [SecureOptions]\n")
		fmt.Fprintf(w, "#   Passphrase = \"\"             # [age|openssl] Encryption passphrase (auto-generated if omitted)\n")
		fmt.Fprintf(w, "#   Recipient = \"\"              # [age] @github_user, ssh://host, URL, ssh key, or age1...\n")
		fmt.Fprintf(w, "#                               # [pgp] Name or email to search in keyring\n")
		fmt.Fprintf(w, "#   Cipher = \"\"                 # [openssl] Cipher (default: aes-256-cbc)\n")
		fmt.Fprintf(w, "#   Options = \"-md sha512 -pbkdf2 -iter 120000\"  # [openssl] Additional command line options\n")
		fmt.Fprintf(w, "#   Openssl = \"\"                # [openssl] Path to openssl binary (default: /usr/bin/openssl)\n")
		fmt.Fprintf(w, "#   Keyring = \"\"                # [pgp] Path to GnuPG keyring (default: $GNUPGHOME/pubring.gpg or ~/.gnupg/pubring.gpg)\n")
		fmt.Fprintln(w)
	}

	// [Profiles.*]
	// Always write the Profiles header comment (consistent with other sections)
	fmt.Fprintf(w, "# --- Profiles ---\n")
	fmt.Fprintf(w, "# Named profiles let you maintain different configurations\n")
	fmt.Fprintf(w, "# for multiple servers or use-cases. Use with: plik -P <name> file.txt\n")
	fmt.Fprintf(w, "# Profiles inherit all top-level settings and can override any field.\n")
	fmt.Fprintf(w, "# If a profile sets URL, it *must* also set Token (use Token = \"\" for anonymous).\n")

	if len(plikrc.Profiles) > 0 {
		// Sort profile names for deterministic output
		names := make([]string, 0, len(plikrc.Profiles))
		for name := range plikrc.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			profile := plikrc.Profiles[name]
			fmt.Fprintf(w, "[Profiles.%s]\n", name)
			writeProfileSection(w, &profile, name, plikrc.metadata)
			fmt.Fprintln(w)
		}
	} else {
		// No active profiles — show commented-out examples
		fmt.Fprintf(w, "#\n")
		fmt.Fprintf(w, "# [Profiles.local]\n")
		fmt.Fprintf(w, "# URL = \"http://127.0.0.1:8080\"\n")
		fmt.Fprintf(w, "# Token = \"\"\n")
		fmt.Fprintf(w, "# AutoUpdate = false\n")
		fmt.Fprintf(w, "#\n")
		fmt.Fprintf(w, "# [Profiles.work]\n")
		fmt.Fprintf(w, "# URL = \"https://plik.work.corp\"\n")
		fmt.Fprintf(w, "# Token = \"your-token-here\"\n")
		fmt.Fprintf(w, "# AutoUpdate = false\n")
		fmt.Fprintf(w, "#\n")
		fmt.Fprintf(w, "# # Create a .zip archive instead of the default .tar.gz\n")
		fmt.Fprintf(w, "# [Profiles.zip]\n")
		fmt.Fprintf(w, "# Archive = true\n")
		fmt.Fprintf(w, "# ArchiveMethod = \"zip\"\n")
		fmt.Fprintf(w, "#\n")
		fmt.Fprintf(w, "# # Encrypt files for a specific GitHub user using age\n")
		fmt.Fprintf(w, "# [Profiles.bob]\n")
		fmt.Fprintf(w, "# Secure = true\n")
		fmt.Fprintf(w, "# SecureMethod = \"age\"\n")
		fmt.Fprintf(w, "# [Profiles.bob.SecureOptions]\n")
		fmt.Fprintf(w, "# Recipient = \"@bob\"   # github username\n")
	}

	return nil
}

// WritePlikrcTemplate writes the canonical .plikrc reference template.
// It uses writeConfig with showcase defaults so the template is always
// consistent with the serialization code (DRY).
func WritePlikrcTemplate(w io.Writer) error {
	config := NewUploadConfig()
	config.URL = "https://plik.root.gg"
	config.AutoUpdate = true

	plikrc := &PlikrcFile{CliConfig: *config}
	return writeConfig(w, plikrc)
}

// writeMapSection writes a map[string]any as indented TOML key-value pairs.
func writeMapSection(w io.Writer, m map[string]any) {
	// Sort keys for deterministic output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case string:
			fmt.Fprintf(w, "  %s = %q\n", k, val)
		case bool:
			fmt.Fprintf(w, "  %s = %t\n", k, val)
		case int, int64, float64:
			fmt.Fprintf(w, "  %s = %v\n", k, val)
		default:
			fmt.Fprintf(w, "  %s = %q\n", k, fmt.Sprintf("%v", val))
		}
	}
}

// writeProfileSection writes the explicitly-defined fields of a CliConfig profile.
// When TOML metadata is available (from DecodeFile), it uses IsDefined() to determine
// which fields were explicitly set — preserving intentional zero values like `false` or `""`.
// When metadata is not available (programmatic profiles), it falls back to writing
// only non-zero fields.
// Fields tagged `profile:"-"` or `toml:"-"` and unexported fields are skipped.
func writeProfileSection(w io.Writer, profile *CliConfig, profileName string, md toml.MetaData) {
	profVal := reflect.ValueOf(profile).Elem()
	profType := profVal.Type()
	hasMetadata := len(md.Keys()) > 0

	for i := 0; i < profType.NumField(); i++ {
		field := profType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields tagged profile:"-"
		if field.Tag.Get("profile") == "-" {
			continue
		}

		// Skip fields tagged toml:"-"
		if field.Tag.Get("toml") == "-" {
			continue
		}

		fieldVal := profVal.Field(i)
		zeroVal := reflect.Zero(field.Type)
		isZero := reflect.DeepEqual(fieldVal.Interface(), zeroVal.Interface())
		isDefined := hasMetadata && md.IsDefined("Profiles", profileName, field.Name)

		// Write field if it was explicitly defined in TOML (preserves intentional
		// zero values like `false`) OR if it has a non-zero value (catches
		// programmatic additions like Token set by saveToken).
		if !isDefined && isZero {
			continue
		}

		// Handle map fields as TOML sub-tables
		if field.Type.Kind() == reflect.Map {
			if fieldVal.Len() > 0 {
				fmt.Fprintf(w, "\n[Profiles.%s.%s]\n", profileName, field.Name)
				writeMapSection(w, fieldVal.Interface().(map[string]any))
			}
			continue
		}

		// Write scalar fields
		switch fieldVal.Kind() {
		case reflect.String:
			fmt.Fprintf(w, "%s = %q\n", field.Name, fieldVal.String())
		case reflect.Bool:
			fmt.Fprintf(w, "%s = %t\n", field.Name, fieldVal.Bool())
		case reflect.Int, reflect.Int64:
			fmt.Fprintf(w, "%s = %d\n", field.Name, fieldVal.Int())
		}
	}
}

// configFilePath returns the path to the .plikrc config file.
// It checks $PLIKRC first, then falls back to ~/.plikrc.
func configFilePath() string {
	path := os.Getenv("PLIKRC")
	if path != "" {
		return path
	}

	home, err := homedir.Dir()
	if err != nil {
		home = os.Getenv("HOME")
		if home == "" {
			home = "."
		}
	}

	return home + "/.plikrc"
}

// saveConfig writes a PlikrcFile to disk at the given path.
func saveConfig(path string, plikrc *PlikrcFile) error {
	buf := new(bytes.Buffer)
	if err := writeConfig(buf, plikrc); err != nil {
		return fmt.Errorf("unable to serialize config: %s", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to write config file %s: %s", path, err)
	}
	defer f.Close()

	_, err = f.Write(buf.Bytes())
	return err
}

// updatePlikrc rewrites the config file at cfg.ConfigPath in canonical format
// using writeConfig. All values and profiles are preserved; custom comments are
// replaced with standard inline comments. Prompts for confirmation unless
// cfg.Yes is set.
func updatePlikrc(cfg *CliConfig) error {
	path := cfg.ConfigPath
	if path == "" {
		path = configFilePath()
	}

	plikrc, _, err := loadPlikrc(path)
	if err != nil {
		return fmt.Errorf("unable to read %s: %s", path, err)
	}

	if !cfg.Yes {
		fmt.Printf("This will rewrite %s in canonical format.\n", path)
		fmt.Printf("All values and profiles are preserved; custom comments will be replaced.\n")
		fmt.Printf("Continue? [y/N] ")
		ok, err := common.AskConfirmation(false)
		if err != nil {
			return fmt.Errorf("unable to ask for confirmation: %s", err)
		}
		if !ok {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := saveConfig(path, plikrc); err != nil {
		return err
	}

	fmt.Printf("✓ %s updated\n", path)
	return nil
}

// loadPlikrc reads and decodes a PlikrcFile from the given path.
// It returns the parsed PlikrcFile and TOML metadata for field-presence checks.
func loadPlikrc(path string) (*PlikrcFile, toml.MetaData, error) {
	var plikrc PlikrcFile
	plikrc.CliConfig = *NewUploadConfig()

	md, err := toml.DecodeFile(path, &plikrc)
	if err != nil {
		return nil, md, fmt.Errorf("Failed to deserialize ~/.plikrc : %s", err)
	}

	plikrc.metadata = md
	return &plikrc, md, nil
}

// LoadConfigFromFile reads, parses, and resolves a Plik CLI configuration.
// It loads the base config, applies profile(s) if selected, and returns
// the fully-merged CliConfig ready for use.
func LoadConfigFromFile(path string, profileName string) (*CliConfig, error) {
	plikrc, md, err := loadPlikrc(path)
	if err != nil {
		return nil, err
	}

	config := &plikrc.CliConfig

	// Populate available profile names (sorted for stable output)
	for name := range plikrc.Profiles {
		config.AvailableProfiles = append(config.AvailableProfiles, name)
	}
	sort.Strings(config.AvailableProfiles)

	// Resolve profile name: CLI flag > env var > config DefaultProfile
	if profileName == "" {
		profileName = os.Getenv("PLIK_PROFILE")
		if profileName != "" {
			config.ProfileSource = "env"
		}
	}
	if profileName == "" {
		profileName = config.DefaultProfile
		if profileName != "" {
			config.ProfileSource = "default"
		}
	}

	// Apply profile(s) if selected — composition: merge left-to-right, last wins
	if profileName != "" {
		profileNames := parseProfiles(profileName)
		for _, name := range profileNames {
			profile, ok := plikrc.Profiles[name]
			if !ok {
				if len(config.AvailableProfiles) == 0 {
					return nil, fmt.Errorf("Profile %q not found (no profiles defined in config)", name)
				}
				return nil, fmt.Errorf("Profile %q not found (available: %s)", name, strings.Join(config.AvailableProfiles, ", "))
			}
			if err := validateProfile(md, name); err != nil {
				return nil, err
			}
			mergeProfile(config, &profile, md, name)
		}
		config.ActiveProfiles = profileNames
	}

	// Sanitize URL
	config.URL = strings.TrimSuffix(config.URL, "/")

	config.ConfigPath = path

	return config, nil
}

// parseProfiles splits a comma-separated profile string into a deduplicated,
// ordered list. Empty segments (from leading/trailing/double commas) are dropped.
func parseProfiles(input string) []string {
	parts := strings.Split(input, ",")
	seen := make(map[string]bool, len(parts))
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		result = append(result, p)
	}
	return result
}

// SingleProfile returns the single active profile name, or an error if
// multiple profiles are active. Used as a gate for operations that require
// exactly one profile (e.g. --login, saveToken).
func (c *CliConfig) SingleProfile() (string, error) {
	if len(c.ActiveProfiles) > 1 {
		return "", fmt.Errorf("this operation requires a single profile (got: %s)",
			strings.Join(c.ActiveProfiles, ", "))
	}
	if len(c.ActiveProfiles) == 1 {
		return c.ActiveProfiles[0], nil
	}
	return "", nil
}

// NewClient creates a plik.Client from this config with upload defaults
// set on the client. plik.Client.NewUpload() inherits them automatically
// via the embedded *UploadParams copy.
func (c *CliConfig) NewClient(name string) *plik.Client {
	client := plik.NewClient(c.URL)
	client.Debug = c.Debug
	client.ClientName = name

	// Upload defaults — NewUpload() copies these via embedded *UploadParams
	client.Token = c.Token
	client.Stream = c.Stream
	client.OneShot = c.OneShot
	client.Removable = c.Removable
	client.TTL = c.TTL
	client.ExtendTTL = c.ExtendTTL
	client.Comments = c.Comments
	client.Login = c.Login
	client.Password = c.Password

	if c.Insecure {
		client.Insecure()
	}

	return client
}

// validateProfile checks that a profile doesn't inherit sensitive fields by accident.
// If a profile changes the server URL, it *must* also define Token (even Token = "")
// to prevent the base config's token from leaking to a different server.
func validateProfile(md toml.MetaData, profileName string) error {
	if md.IsDefined("Profiles", profileName, "URL") && !md.IsDefined("Profiles", profileName, "Token") {
		return fmt.Errorf("profile %q defines URL but not Token — add Token = \"\" for anonymous or set your token to prevent credential leakage", profileName)
	}
	return nil
}

// mergeProfile overlays explicitly-set profile fields onto the base config.
// It uses TOML metadata to distinguish "field not present" from "field set to zero value".
// Fields tagged `profile:"-"` and unexported fields are skipped.
func mergeProfile(base *CliConfig, profile *CliConfig, md toml.MetaData, profileName string) {
	baseVal := reflect.ValueOf(base).Elem()
	profVal := reflect.ValueOf(profile).Elem()
	baseType := baseVal.Type()

	for i := 0; i < baseType.NumField(); i++ {
		field := baseType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields tagged profile:"-"
		if field.Tag.Get("profile") == "-" {
			continue
		}

		if md.IsDefined("Profiles", profileName, field.Name) {
			baseVal.Field(i).Set(profVal.Field(i))
		}
	}
}

// LoadConfig creates a new default configuration and override it with .plikrc file.
// If .plikrc does not exist, ask domain, and create a new one in user HOMEDIR
func LoadConfig(opts docopt.Opts) (config *CliConfig, err error) {
	// Resolve profile name from CLI flag (env var / config default handled in LoadConfigFromFile)
	var profileName string
	var fromFlag bool
	if opts["--profile"] != nil && opts["--profile"].(string) != "" {
		profileName = opts["--profile"].(string)
		fromFlag = true
	}

	loadAndTag := func(path string) (*CliConfig, error) {
		cfg, err := LoadConfigFromFile(path, profileName)
		if err != nil {
			return nil, err
		}
		if fromFlag {
			cfg.ProfileSource = "flag"
		}
		return cfg, nil
	}

	// Load config file from environment variable
	path := os.Getenv("PLIKRC")
	if path != "" {
		_, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("Plikrc file %s not found", path)
		}
		return loadAndTag(path)
	}

	// Detect home dir
	home, err := homedir.Dir()
	if err != nil {
		home = os.Getenv("HOME")
		if home == "" {
			home = "."
		}
	}

	// Load config file from ~/.plikrc
	path = home + "/.plikrc"
	_, err = os.Stat(path)
	if err == nil {
		// Config file found — return result or error directly
		return loadAndTag(path)
	}

	// Load global config file from /etc directory
	path = "/etc/plik/plikrc"
	_, err = os.Stat(path)
	if err == nil {
		return loadAndTag(path)
	}

	config = NewUploadConfig()

	// Bypass ~/.plikrc file creation if quiet mode, --yes mode, and/or --server flag
	if opts["--quiet"].(bool) || opts["--yes"].(bool) || (opts["--server"] != nil && opts["--server"].(string) != "") {
		return config, nil
	}

	// Config file not found. Create one.
	path = home + "/.plikrc"

	// Ask for domain
	var domain string
	fmt.Println("Please enter your plik domain [default:http://127.0.0.1:8080] : ")
	_, err = fmt.Scanf("%s", &domain)
	if err == nil {
		domain = strings.TrimRight(domain, "/")
		parsedDomain, err := url.Parse(domain)
		if err == nil {
			if parsedDomain.Scheme == "" {
				parsedDomain.Scheme = "http"
			}
			config.URL = parsedDomain.String()
		}
	}

	// Try to HEAD the site to see if we have a redirection
	client := plik.NewClient(config.URL)
	client.Insecure()
	resp, err := client.HTTPClient.Head(config.URL)
	if err != nil {
		return nil, err
	}

	finalURL := resp.Request.URL.String()
	if finalURL != "" && finalURL != config.URL {
		fmt.Printf("We have been redirected to : %s\n", finalURL)
		fmt.Printf("Replace current url (%s) with the new one ? [Y/n] ", config.URL)

		ok, err := common.AskConfirmation(true)
		if err != nil {
			return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
		}
		if ok {
			config.URL = strings.TrimSuffix(finalURL, "/")
		}
	}

	// Try to get server config to sync default values
	serverConfig, err := client.GetServerConfig()
	if err != nil {
		fmt.Printf("Unable to get server configuration : %s", err)
	} else {
		config.OneShot = common.IsFeatureDefault(serverConfig.FeatureOneShot)
		config.Removable = common.IsFeatureDefault(serverConfig.FeatureRemovable)
		config.Stream = common.IsFeatureDefault(serverConfig.FeatureStream)
		config.ExtendTTL = common.IsFeatureDefault(serverConfig.FeatureExtendTTL)

		// Skip interactive login during first-run setup when --login is set,
		// the --login handler in main() will perform the login flow.
		if !opts["--login"].(bool) {
			switch serverConfig.FeatureAuthentication {
			case common.FeatureForced:
				fmt.Printf("\nAuthentication is required on this server.\n")
				fmt.Printf("Would you like to authenticate with your browser? [Y/n] ")
				ok, err := common.AskConfirmation(true)
				if err != nil {
					return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
				}
				if ok {
					loginClient := plik.NewClient(config.URL)
					loginClient.Insecure()
					err = login(config, loginClient)
					if err != nil {
						fmt.Printf("Login failed: %s\n", err)
						fmt.Printf("You can provide a token manually instead.\n")
						fmt.Printf("Please enter a valid user token : \n")
						var token string
						_, err = fmt.Scanf("%s", &token)
						if err == nil {
							config.Token = token
						}
					}
				} else {
					fmt.Printf("Please enter a valid user token : \n")
					var token string
					_, err = fmt.Scanf("%s", &token)
					if err == nil {
						config.Token = token
					}
				}
			case common.FeatureEnabled:
				fmt.Printf("\nAuthentication is available on this server.\n")
				fmt.Printf("Would you like to authenticate with your browser? [y/N] ")
				ok, err := common.AskConfirmation(false)
				if err != nil {
					return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
				}
				if ok {
					loginClient := plik.NewClient(config.URL)
					loginClient.Insecure()
					err = login(config, loginClient)
					if err != nil {
						fmt.Printf("Login failed: %s\n", err)
					}
				}
			}
		}
	}

	// Enable client updates ?
	fmt.Println("Do you want to enable client auto update ? [Y/n] ")
	ok, err := common.AskConfirmation(true)
	if err != nil {
		return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
	}
	if ok {
		config.AutoUpdate = true
	}

	// Write config file
	plikrc := &PlikrcFile{CliConfig: *config}
	if err = saveConfig(path, plikrc); err != nil {
		return nil, fmt.Errorf("Failed to save ~/.plikrc : %s", err)
	}

	fmt.Println("Plik client settings successfully saved to " + path)
	return config, nil
}

// UnmarshalArgs turns command line arguments into upload settings
// Command line arguments override config file settings
func (config *CliConfig) UnmarshalArgs(opts docopt.Opts) (err error) {
	if opts["--debug"].(bool) {
		config.Debug = true
	}
	if opts["--yes"].(bool) {
		config.Yes = true
	}
	if opts["--quiet"].(bool) {
		config.Quiet = true
	}
	if opts["--json"].(bool) {
		config.JSON = true
		config.Quiet = true
	}

	// Plik server url
	if opts["--server"] != nil && opts["--server"].(string) != "" {
		config.URL = opts["--server"].(string)
		if config.Token != "" {
			config.Token = ""
			// Only warn if --token is not also provided (it will restore the value later)
			if opts["--token"] == nil || opts["--token"].(string) == "" {
				fmt.Fprintf(os.Stderr, "Warning: --server overrides URL — token cleared (use --token to set explicitly)\n")
			}
		}
	}

	// Paths
	if _, ok := opts["FILE"].([]string); ok {
		config.filePaths = opts["FILE"].([]string)
	} else {
		return fmt.Errorf("No files specified")
	}

	for _, path := range config.filePaths {
		// Test if file exists
		fileInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("File %s not found", path)
		}

		// Automatically enable archive mode is at least one file is a directory
		if fileInfo.IsDir() {
			config.Archive = true
		}
	}

	// Override file name if specified
	if opts["--name"] != nil && opts["--name"].(string) != "" {
		config.filenameOverride = opts["--name"].(string)
	}

	// Upload options
	if opts["--oneshot"].(bool) {
		config.OneShot = true
	}
	if opts["--removable"].(bool) {
		config.Removable = true
	}

	if opts["--stream"].(bool) {
		config.Stream = true
	}

	if opts["--comments"] != nil && opts["--comments"].(string) != "" {
		config.Comments = opts["--comments"].(string)
	}

	// Configure upload expire date
	if opts["--ttl"] != nil && opts["--ttl"].(string) != "" {
		ttlStr := opts["--ttl"].(string)
		mul := 1
		if string(ttlStr[len(ttlStr)-1]) == "m" {
			mul = 60
		} else if string(ttlStr[len(ttlStr)-1]) == "h" {
			mul = 3600
		} else if string(ttlStr[len(ttlStr)-1]) == "d" {
			mul = 86400
		}
		if mul != 1 {
			ttlStr = ttlStr[:len(ttlStr)-1]
		}
		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			return fmt.Errorf("Invalid TTL %s", opts["--ttl"].(string))
		}
		config.TTL = ttl * mul
	}

	if opts["--extend-ttl"].(bool) {
		config.ExtendTTL = true
	}

	// Enable archive mode ?
	if opts["-a"].(bool) || opts["--archive"] != nil || config.Archive {
		config.Archive = true

		if opts["--archive"] != nil && opts["--archive"] != "" {
			config.ArchiveMethod = opts["--archive"].(string)
		}
	}

	// Enable secure mode ?
	if opts["--not-secure"].(bool) {
		config.Secure = false
	} else if opts["-s"].(bool) || opts["--secure"] != nil || config.Secure {
		config.Secure = true
		if opts["--secure"] != nil && opts["--secure"].(string) != "" {
			config.SecureMethod = opts["--secure"].(string)
		}
	}

	// Enable password protection ?
	if opts["-p"].(bool) {
		fmt.Printf("Login [plik]: ")
		var err error
		_, err = fmt.Scanln(&config.Login)
		if err != nil && err.Error() != "unexpected newline" {
			return fmt.Errorf("Unable to get login : %s", err)
		}
		if config.Login == "" {
			config.Login = "plik"
		}
		fmt.Printf("Password: ")
		_, err = fmt.Scanln(&config.Password)
		if err != nil {
			return fmt.Errorf("Unable to get password : %s", err)
		}
	} else if opts["--password"] != nil && opts["--password"].(string) != "" {
		credentials := opts["--password"].(string)
		sepIndex := strings.Index(credentials, ":")
		var login, password string
		if sepIndex > 0 {
			login = credentials[:sepIndex]
			password = credentials[sepIndex+1:]
		} else {
			login = "plik"
			password = credentials
		}
		config.Login = login
		config.Password = password
	}

	// Override upload token ?
	if opts["--token"] != nil && opts["--token"].(string) != "" {
		config.Token = opts["--token"].(string)
	}

	// Ask for token
	if config.Token == "-" {
		fmt.Printf("Token: ")
		var err error
		_, err = fmt.Scanln(&config.Token)
		if err != nil {
			return fmt.Errorf("Unable to get token : %s", err)
		}
	}

	if opts["--stdin"].(bool) {
		config.DisableStdin = false
	}

	return
}
