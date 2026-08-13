# CLI

Plik ships with a powerful cross-platform CLI client written in Go.

## Installation

### From GitHub Releases

Download the latest client binary for your platform directly from the [GitHub releases page](https://github.com/root-gg/plik/releases):

```bash
# Linux (amd64)
wget https://github.com/root-gg/plik/releases/download/__VERSION__/plik-__VERSION__-linux-amd64
chmod +x plik-__VERSION__-linux-amd64
sudo mv plik-__VERSION__-linux-amd64 /usr/local/bin/plik

# macOS (amd64)
curl -L -o plik https://github.com/root-gg/plik/releases/download/__VERSION__/plik-__VERSION__-darwin-amd64
chmod +x plik
sudo mv plik /usr/local/bin/plik

# Windows (amd64)
# Download plik-__VERSION__-windows-amd64.exe from the release page
```

Available platforms: `linux-amd64`, `linux-386`, `linux-arm`, `linux-arm64`, `darwin-amd64`, `freebsd-amd64`, `freebsd-386`, `openbsd-amd64`, `openbsd-386`, `windows-amd64`, `windows-386`

### From Plik Web UI

Any running Plik instance serves its client binaries through the web interface. Navigate to your Plik server and download the client for your platform.


## Usage

<!-- BEGIN:HELP -->
```
plik — temporary file sharing

Usage:
  plik [options] [FILE] ...

Profile Options:
  -P, --profile PROFILES    Use named profiles from ~/.plikrc (comma-separated for composition)

Upload Options:
  -o, --oneshot             Delete each file after first download
  -r, --removable           Allow anyone to delete uploaded files
  -S, --stream              Stream upload (blocks until receiver downloads)
  -t, --ttl TTL             Time before expiration (e.g. 30m, 24h, 7d)
  --extend-ttl              Extend expiration on each download
  -p                        Prompt for upload login and password
  --password PASSWD         Protect upload with login:password (default login: "plik")
  --comments COMMENT        Set upload comments (Markdown)
  -n, --name NAME           Set filename when piping from STDIN

Server Options:
  --server SERVER           Override server URL
  --token TOKEN             Set upload token (use '-' to prompt)
  --insecure                Skip TLS certificate verification

Archive Options:
  -a                        Archive files using default settings from ~/.plikrc
  --archive MODE            Archive files with specified backend (tar | zip)
  --compress MODE           [tar] Compression codec (gzip|bzip2|xz|lzip|lzma|lzop|no)
  --archive-options OPTIONS Additional command line options passed to archiver

Encryption Options:
  -s                        Encrypt files using default settings from ~/.plikrc
  --not-secure              Disable encryption even if enabled in ~/.plikrc
  --secure MODE             Encrypt files with backend (age | openssl | pgp, default: age)
  --passphrase PASSPHRASE   [age|openssl] Encryption passphrase (use '-' to prompt)
  --recipient RECIPIENT     [age] @github_user, ssh://host, URL, key, or age1...
                            [pgp] Recipient name or email
  --cipher CIPHER           [openssl] Cipher algorithm (default: aes-256-cbc)
  --secure-options OPTIONS  [openssl|pgp] Additional command line options

Output Options:
  -q, --quiet               Suppress progress and non-essential output
  -j, --json                Output upload metadata as JSON (implies --quiet)
  -d, --debug               Enable debug mode

General Options:
  --login                   Authenticate with server (opens browser)
  --update                  Update client binary from server
  --update-plikrc           Rewrite ~/.plikrc in canonical format
  --mcp                     Start MCP (Model Context Protocol) server over stdio
  --stdin                   Read from STDIN even when DisableStdin is set
  -y, --yes                 Auto-accept confirmation prompts
  -v, --version             Show client version
  -i, --info                Show client and server information
  -h, --help                Show this help

Examples:
  plik file.txt                       Upload a single file
  plik -o file1.txt file2.txt         Upload files, delete after first download
  plik -t 1h *.log                    Upload with 1 hour expiration
  plik -s secret.pdf                  Encrypt with age (passphrase auto-generated)
  plik -a src/                        Archive and upload a directory
  plik -P work report.pdf             Upload using the "work" profile
  plik -P work,zip report.pdf         Compose profiles (work server + zip archive)
  cat data.csv | plik -n data.csv     Pipe from STDIN
```
<!-- END:HELP -->

### Examples

Upload a file:
```bash
🪂 ➜  plik git:(master) ✗ plik README.md
Upload successfully created at Sat, 21 Feb 2026 09:02:54 CET :
    http://127.0.0.1:8080/#/?id=vDPmPEUqc5oCt31T

README.md :  2.56 KiB / 2.56 KiB [=========================================] 100.00% 719.15 KiB/s 0s

Commands :
curl -s "http:/127.0.0.1:8080/file/vDPmPEUqc5oCt31T/UZzSdZ7zPgfRiTem/README.md" > 'README.md'
```

Create an encrypted archive:
```bash
plik -a -s mydirectory/
```

Upload with expiration:
```bash
plik --ttl 24h document.pdf
```

## Quick Upload with curl

No CLI needed — upload with a single curl command:

```bash
curl --form 'file=@/path/to/file' http://127.0.0.1:8080
```

With authentication token:
```bash
curl --form 'file=@/path/to/file' \
     --header 'X-PlikToken: xxxx-xxx-xxxx-xxxxx-xxxxxxxx' \
     http://127.0.0.1:8080
```

::: tip
The `DownloadDomain` configuration option must be set for quick upload to work properly.
:::

## CLI Authentication

When authentication is enabled on the server, you can authenticate the CLI client using `--login`:

```bash
plik --login
```

This starts a device authorization flow:
1. The CLI displays a **verification code** and opens a URL in your browser
2. In the browser, log in (if needed) and **approve** the CLI login by confirming the code
3. The CLI automatically receives a token and saves it to `~/.plikrc`

::: tip
The token created via `--login` is identical to tokens created in the web UI. It appears in your token list and can be revoked from the web UI at any time.
:::

### First-run experience

When running `plik` for the first time and the server has authentication enabled, the CLI will prompt you to authenticate via browser:
- If authentication is **forced**: you'll be prompted with a default of **Yes**
- If authentication is **enabled**: you'll be prompted with a default of **No**

You can always authenticate later with `plik --login`.

::: tip Non-interactive mode
Use `plik --yes` to auto-accept all confirmation prompts (first-run wizard, updates, HTTP key fetch warnings). This is useful for scripting and CI/CD pipelines.
:::

### Manual token configuration

Alternatively, you can create a token manually in the web UI and add it to your configuration:

```toml
# ~/.plikrc
Token = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

Or pass it on the command line:

```bash
plik --token xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx myfile.txt
```

## Configuration (.plikrc)

### Maintaining your config file

If your `.plikrc` has accumulated drift or you want to normalize it after hand-editing, you can rewrite it in canonical format:

```bash
plik --update-plikrc
```

This rewrites your config file using the same format as the first-run wizard — well-commented, consistently ordered, and with all sections labeled. **All values and profiles are preserved.** Custom comments are replaced with standard inline comments.

Use `--yes` to skip the confirmation prompt:

```bash
plik --update-plikrc --yes
```

::: tip
The `--login` flag uses surgical patching to update only your token — it preserves all comments and ordering. `--update-plikrc` is the intentional "reformat everything" command when you want a clean file.
:::

The client configuration is a TOML file loaded from:
1. `PLIKRC` environment variable
2. `~/.plikrc`
3. `/etc/plik/plikrc`

Key settings:

<!-- BEGIN:PLIKRC -->
```toml
# --- Server ---
URL = "https://plik.root.gg"    # URL of the plik server
Token = ""                      # Authentication token (created via web UI or --login)
Insecure = false                # Skip TLS certificate validation

# --- Upload defaults ---
OneShot = false                 # Delete file after first download (if available server side)
Removable = false               # Allow anyone to delete the file (if available server side)
Stream = false                  # Stream upload, blocks until download starts (if available server side)
TTL = 0                         # Upload time-to-live in seconds (0 = server default)
ExtendTTL = false               # Extend expiration on access (if available server side)
Comments = ""                   # Default upload comments (Markdown)

# --- Authentication ---
Login = ""                      # HTTP basic auth login
Password = ""                   # HTTP basic auth password

# --- Archive ---
Archive = false                 # Archive files before upload
ArchiveMethod = "tar"           # Archive backend (tar | zip)

# --- Encryption ---
Secure = false                  # Encrypt files before upload
SecureMethod = "age"            # Crypto backend (age | openssl | pgp)

# --- Output ---
Debug = false                   # Verbose debug output
Quiet = false                   # Suppress non-essential output
JSON = false                    # Output upload metadata as JSON (implies Quiet)
DownloadBinary = "curl"         # Download command for output (curl | wget)

# --- Behavior ---
AutoUpdate = true               # Auto-update client binary from server
DisableStdin = false            # Disable STDIN pipe input by default
Yes = false                     # Auto-accept confirmation prompts (non-interactive)
DefaultProfile = ""             # Default profile to use (can also be set via PLIK_PROFILE env var)

[ArchiveOptions]
  Compress = "gzip"
  Options = ""
  Tar = "/bin/tar"

# [SecureOptions]
#   Passphrase = ""             # [age|openssl] Encryption passphrase (auto-generated if omitted)
#   Recipient = ""              # [age] @github_user, ssh://host, URL, ssh key, or age1...
#                               # [pgp] Name or email to search in keyring
#   Cipher = ""                 # [openssl] Cipher (default: aes-256-cbc)
#   Options = "-md sha512 -pbkdf2 -iter 120000"  # [openssl] Additional command line options
#   Openssl = ""                # [openssl] Path to openssl binary (default: /usr/bin/openssl)
#   Keyring = ""                # [pgp] Path to GnuPG keyring (default: $GNUPGHOME/pubring.gpg or ~/.gnupg/pubring.gpg)

# --- Profiles ---
# Named profiles let you maintain different configurations
# for multiple servers or use-cases. Use with: plik -P <name> file.txt
# Profiles inherit all top-level settings and can override any field.
# If a profile sets URL, it *must* also set Token (use Token = "" for anonymous).
#
# [Profiles.local]
# URL = "http://127.0.0.1:8080"
# Token = ""
# AutoUpdate = false
#
# [Profiles.work]
# URL = "https://plik.work.corp"
# Token = "your-token-here"
# AutoUpdate = false
#
# # Create a .zip archive instead of the default .tar.gz
# [Profiles.zip]
# Archive = true
# ArchiveMethod = "zip"
#
# # Encrypt files for a specific GitHub user using age
# [Profiles.bob]
# Secure = true
# SecureMethod = "age"
# [Profiles.bob.SecureOptions]
# Recipient = "@bob"   # github username
```
<!-- END:PLIKRC -->

### SecureOptions

The `[SecureOptions]` table configures encryption backend-specific settings. Available keys depend on the `SecureMethod`:

| Key | Backend | Description | Default |
|-----|---------|-------------|---------|
| `Passphrase` | age, openssl | Encryption passphrase (auto-generated if omitted) | — |
| `Recipient` | age | `@github_user`, `ssh://host`, URL, ssh key, or `age1…` | — |
| `Recipient` | pgp | Name or email to search in keyring | — |
| `Cipher` | openssl | Cipher algorithm | `aes-256-cbc` |
| `Options` | openssl | Additional command line options | `-md sha512 -pbkdf2 -iter 120000` |
| `Openssl` | openssl | Path to openssl binary | `/usr/bin/openssl` |
| `Keyring` | pgp | Path to GnuPG public keyring | `$GNUPGHOME/pubring.gpg` or `~/.gnupg/pubring.gpg` |

::: tip Passphrase vs Recipient
For age, `Passphrase` and `Recipient` are mutually exclusive. If neither is set, a random passphrase is auto-generated.
:::

See the [full .plikrc template](https://github.com/root-gg/plik/blob/master/client/.plikrc) for all available options.

## Profiles

Profiles let you maintain configurations for multiple servers (or different defaults for the same server) and switch between them with a single flag.

### Defining Profiles

Add `[Profiles.<name>]` sections to your `.plikrc`. Each profile inherits all top-level settings and can override any field. If a profile sets `URL`, it *must* also set `Token` (use `Token = ""` for anonymous) to prevent credential leakage:

```toml
# ~/.plikrc — shared defaults
URL = "https://plik.root.gg"
Token = "your-default-token"
AutoUpdate = true
DefaultProfile = "local"        # Optional (can also be set via PLIK_PROFILE env var)

[Profiles.local]
URL = "http://127.0.0.1:8080"
Token = ""                      # no auth for local dev
AutoUpdate = false              # don't auto-update from local dev server

[Profiles.work]
URL = "https://plik.work.corp"
Token = "your-token-here"
AutoUpdate = false              # don't auto-update from work server

# Create a .zip archive instead of the default .tar.gz
[Profiles.zip]
Archive = true
ArchiveMethod = "zip"

# Encrypt files for a specific GitHub user using age
[Profiles.bob]
Secure = true
SecureMethod = "age"
[Profiles.bob.SecureOptions]
Recipient = "@bob"   # github username
```

### Using Profiles

```bash
# Use the "local" profile
plik -P local file.txt

# Use the long form
plik --profile work file.txt

# Set a default via environment variable
export PLIK_PROFILE=work
plik file.txt     # uses "work" profile

# CLI flags still override profile settings
plik -P work --server https://other.example.com file.txt

# Login to a specific profile
plik --login -P work
```

### Profile Composition

Combine profiles with commas — they merge **left-to-right** (last wins on conflicts):

```bash
# Use work server settings, then add zip archive override
plik -P work,zip file.txt

# Compose three profiles
plik -P local,openssl,oneshot file.txt
```

`DefaultProfile` and `PLIK_PROFILE` also support composition:

```toml
DefaultProfile = "work,zip"
```

```bash
PLIK_PROFILE=local,openssl plik file.txt
```

::: tip Last wins
Profiles are applied left-to-right. If `work` sets `URL = "https://work.corp"` and `zip` also sets `URL`, `zip`'s value wins. Fields only set by one profile are always preserved.
:::

::: warning --login requires a single profile
`plik -P work,zip --login` will error — the login flow saves a token to exactly one profile section and can't know which to use with multiple profiles.
:::

### Profile Selection Precedence

When multiple sources specify a profile, the following precedence applies (highest to lowest):

1. `--profile` / `-P` CLI flag
2. `PLIK_PROFILE` environment variable
3. `DefaultProfile` field in the config file

::: tip Backward Compatible
Existing flat `.plikrc` files (without any `[Profiles]` sections) continue to work exactly as before. Profiles are entirely opt-in.
:::

::: warning Nested Sections
`[ArchiveOptions]` and `[SecureOptions]` are **fully replaced** when overridden in a profile — individual keys are not merged. If a profile defines `[Profiles.local.ArchiveOptions]`, it must include all desired keys.
:::

## Tips & Tricks

### Screenshot Upload (Linux)

Upload screenshots directly to clipboard (requires `scrot` and `xclip`):

```bash
alias pshot="scrot -s -e 'plik -q \$f | xclip ; xclip -o ; rm \$f'"
```

### Windows "Send to Plik"

Upload files to Plik directly from the Windows Explorer right-click menu. See the [dedicated guide](/guide/windows-send-to) for step-by-step instructions.

## Bash Client (Lightweight Alternative)

A minimal bash client (`plik.sh`) is available for environments where installing a Go binary is not practical. It requires only `bash`, `curl`, and optionally `openssl`.

```bash
# From a running Plik server
curl -o plik.sh https://your-plik-server/clients/bash/plik.sh
chmod +x plik.sh

# Or from GitHub releases
wget https://github.com/root-gg/plik/releases/download/__VERSION__/plik-__VERSION__.sh
```

Run `plik.sh -h` for the full list of supported options. The bash client supports most upload features (oneshot, removable, stream, TTL, comments, basic auth, encryption) but does not support STDIN piping, JSON output, auto-update, browser login, or age/PGP encryption.
