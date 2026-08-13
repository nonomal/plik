# Architecture — CLI Client (`client/`)

> The Plik command-line client. For system-wide overview, see the root [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Structure

```
client/
├── plik.go          ← entry point: arg parsing, config loading, dispatch
├── app.go           ← PlikCLI struct: upload flow, helpers (Run, info, getFileCommand, printf)
├── mcp.go           ← MCP (Model Context Protocol) server over stdio for AI assistants
├── config.go        ← configuration loading (.plikrc)
├── config_test.go   ← unit tests for config parsing (TTL, password, flags, file loading)
├── login.go         ← CLI device auth flow (--login)
├── progress.go      ← upload progress bar
├── update.go        ← self-update mechanism (PlikCLI method)
├── update_test.go   ← unit tests for update flow (early exits, error handling)
├── archive/         ← archive backends (tar, zip) — errors via CloseWithError
├── crypto/          ← crypto backends (openssl, pgp, age) — errors via CloseWithError
├── setup_test.go    ← e2e test infrastructure (TestMain, server lifecycle, helpers)
├── z1_e2e_basics_test.go   ← basic CLI tests (info, debug, single/multi file, stdin)
├── z2_e2e_options_test.go  ← upload option tests (oneshot, ttl, quiet, JSON, etc.)
├── z3_e2e_archive_test.go  ← archive backend tests (tar, zip)
├── z4_e2e_crypto_test.go   ← crypto backend tests (openssl, pgp, age)
├── z5_e2e_profiles_test.go ← profile e2e tests (upload, inheritance, info)
├── .plikrc          ← example client configuration
└── plik.sh          ← bash upload wrapper
```

---

## Key Components

### CLI Entry Point (`plik.go`) and Runtime State (`app.go`)

`plik.go` is a slim `main()` using [docopt-go](https://github.com/docopt/docopt-go) for argument parsing. It delegates all upload logic to the `PlikCLI` struct defined in `app.go`.

**`PlikCLI` struct** encapsulates all mutable runtime state:
- `Config`, `Arguments` — parsed configuration and CLI args
- `ArchiveBackend`, `CryptoBackend` — initialized lazily during `Run()`
- `Stdout`, `Stderr` — injectable `io.Writer` for output (default: `os.Stdout`/`os.Stderr`); enables test output capture without global state mutation

**`main()` flow** (in `plik.go`):
1. Parse CLI args → early exits: `--version`, `--mcp`, `--info`, `--login`
2. Load config from `.plikrc` → `NewPlikCLI(config, args)`
   - First-run wizard is skipped when `--quiet`, `--yes`, or `--server` is set
3. Dispatch to `cli.Run(client)` for the upload flow

**Early-exit flags** (handled before `cli.Run`):
- `--version`, `--help` — print and exit
- `--mcp` — start MCP server over stdio
- `--info` — print client/server info
- `--update` — self-update binary
- `--login` — device auth flow, saves token via surgical patch
- `--update-plikrc` — rewrite config in canonical format (`updatePlikrc()` in `config.go`)

**`PlikCLI.Run()` flow** (in `app.go`):
1. Create upload via the Go library (`plik/`)
2. Add files (with optional archive/encrypt preprocessing)
3. Upload files with progress bars
4. Output results:
   - Default: print download URLs/commands to stdout
   - `--quiet`: print only file URLs to stdout
   - `--json`: print `UploadWithURL` as pretty-printed JSON to stdout (implies `--quiet`)

### Configuration (`config.go`)

Config is a TOML file loaded from (in order):
1. `PLIKRC` environment variable
2. `~/.plikrc`
3. `/etc/plik/plikrc`

`CliConfig` fields are grouped logically: Server, Upload defaults, Authentication, Archive, Encryption, Output, Behavior, Runtime. This order determines both the struct layout and the TOML serialization order produced by `writeConfig()`.

**`writeConfig()`** produces human-readable, commented TOML matching the `.plikrc` template format. It writes all scalar fields first, then `[Table]` sections (`[ArchiveOptions]`, `[SecureOptions]`, `[Profiles.*]`) — this ordering is required by TOML spec. The `configLine()` helper handles column-aligned inline comments. Used by the first-run wizard and `saveConfig()` for creating new config files.

**`WritePlikrcTemplate()`** generates the canonical `client/.plikrc` reference template. It calls `writeConfig()` with showcase defaults (DRY — same code path, different values). The `TestPlikrcTemplate_UpToDate` test compares the generated output against the committed file and rewrites it if stale. CI catches drift via `git diff --exit-code`.

#### Multi-Profile Support

The config file supports named profiles via `[Profiles.<name>]` TOML sections. Each profile can override any subset of the top-level fields. An on-disk config file is represented by `PlikrcFile`, which embeds `CliConfig` (base fields) plus `Profiles map[string]CliConfig` and `DefaultProfile string`.

**Profile selection precedence** (highest to lowest):
1. `--profile` / `-P` CLI flag (supports comma-separated names for composition)
2. `PLIK_PROFILE` environment variable (also supports comma-separated names)
3. `DefaultProfile` field in config file (also supports comma-separated names)

**Profile composition**: `plik -P work,zip` applies profiles left-to-right over the base config. Last one wins on conflicts; non-overlapping fields from all profiles survive. Implemented via `parseProfiles()` (split + trim + dedup) and the composition loop in `LoadConfigFromFile`.

**Config layering** (highest to lowest):
1. CLI flags (`--server`, `--token`, etc.)
2. Selected profile(s) fields (composed left-to-right)
3. Top-level config fields
4. Built-in defaults (`NewUploadConfig()`)

**Merge semantics**: `mergeProfile()` uses `toml.MetaData.IsDefined()` to apply only fields explicitly set in the profile section. This distinguishes "not present" from "set to zero value" (e.g., `Token = ""` in a profile clears the base token). `validateProfile()` enforces that any profile defining `URL` must also define `Token` to prevent credential leakage to a different server. Similarly, `--server` in `UnmarshalArgs` clears the token (re-settable with `--token`).

**Key helpers**:
- `parseProfiles(input string) []string` — splits a comma-separated profile string into a deduplicated ordered list. Trims whitespace, drops empty segments.
- `SingleProfile() (string, error)` — returns the single active profile name, or errors if multiple profiles are active. Used as the DRY gate by `--login` (in `plik.go`) and `saveToken` (in `login.go`) which require exactly one profile to know where to write the token.
- `NewClient(name string) *plik.Client` — creates a `plik.Client` from the config with all upload defaults (`Token`, `Stream`, `OneShot`, `Removable`, `TTL`, `ExtendTTL`, `Comments`, `Login`, `Password`) set on the client. `plik.Client.NewUpload()` inherits these automatically via the embedded `*UploadParams` copy. Used by `plik.go` (CLI), `mcp.go` (MCP server), and test helpers.

The runtime `CliConfig` carries `ActiveProfiles []string` (the resolved profile name(s)), `ProfileSource string` (`"flag"` from `-P`, `"env"` from `PLIK_PROFILE`, `"default"` from `DefaultProfile`, or `""` for none), and `AvailableProfiles []string` (list of all profiles defined in the config) — all are `toml:"-"` and not serialized. `DefaultProfile string` (the file-level default) stays a plain string in the config struct. The MCP safety gate only locks profile switching when `ProfileSource == "flag"` (explicit `-P`); `DefaultProfile` does not lock.

Existing flat configs (no `[Profiles]` sections) are 100% backward compatible.

### CLI Login (`login.go`)

Implements a device authorization flow for CLI authentication:
1. POST `/auth/cli/init` with hostname → receives a code, secret, and verification URL
2. Opens verification URL in user's browser (best-effort)
3. Polls POST `/auth/cli/poll` with code + secret every 2s
4. On approval, saves the token to `~/.plikrc` via surgical text patching (`patchToken`) that preserves user comments and profile ordering, then exits

`saveToken()` performs an in-place edit of the raw config file bytes — it finds the correct `Token = "..."` line (top-level or inside `[Profiles.<name>]`) and replaces only its value. If no Token line exists, one is inserted after the URL line (if present), otherwise right after the section header. This avoids the full-file rewrite that `writeConfig()` would produce.

**`--update-plikrc`** is the intentional counterpart: it calls `loadPlikrc()` → `saveConfig()` to rewrite the entire file in canonical format, preserving all values and profile definitions but replacing custom comments with standard inline comments. A `[y/N]` confirmation prompt is shown unless `--yes` is set.

Triggered by `--login` flag or interactively during first-run when auth is enabled/forced. When `--login` is set, the first-run wizard skips its own interactive login to avoid triggering the flow twice.

### Archive Backends (`archive/`)

| Backend | Description |
|---------|-------------|
| `tar` | Create tar archives with compression (gzip, bzip2, xz, lzip, lzma, lzop) |
| `zip` | Create zip archives |

Archives wrap multiple files/directories into a single upload file. Errors are propagated via `io.PipeWriter.CloseWithError()` from the archiving goroutine.

**Binary discovery**: All backends that shell out to external commands (tar, zip, openssl) use `binutil.LookupBinary(configured, name)` which first checks `os.Stat(configured)` and falls back to `exec.LookPath(name)`. This handles cross-platform binary locations (e.g. `/bin/tar` vs `/usr/bin/tar`, macOS Homebrew paths).

### Crypto Backends (`crypto/`)

| Backend | Description |
|---------|-------------|
| `openssl` | Symmetric encryption via OpenSSL CLI (configurable cipher). Uses `binutil.LookupBinary` for portable binary discovery. **Deprecated** — use `age` instead |
| `pgp` | Asymmetric encryption using pure Go `openpgp` (no external gpg binary needed). Keyring path respects `$GNUPGHOME` (default: `~/.gnupg/pubring.gpg`). **Deprecated** — use `age` instead |
| `age` | Modern encryption via [age](https://age-encryption.org/). Supports passphrase, X25519, SSH recipients (`@github_user`, URL, raw key), and SSH host key scanning (`ssh://hostname`). URLs can serve SSH keys **and** native `age1…` recipients. Plain HTTP URLs trigger a MITM security prompt (default: decline). **Default backend.** Sets `upload.E2EE = "age"` for webapp interop (passphrase mode only) |

Encryption wraps the file data stream before upload. Errors are propagated via `io.PipeWriter.CloseWithError()` from the encryption goroutine. All backends expose a `Stderr io.Writer` field (default: `os.Stderr`) and a `SetStderr(w io.Writer)` method so that `PlikCLI` can redirect diagnostic output (passphrase display, recipient resolution progress, warnings) through its injectable writer for test capture.

When the `age` backend is used, the upload is flagged as E2EE (`upload.E2EE = "age"`). This tells the webapp to prompt for a passphrase on download and decrypt client-side. A cryptographically-secure passphrase is auto-generated when none is provided.

### Self-Update (`update.go`)

The client can update itself by downloading the latest matching binary from the configured Plik server. It compares versions and replaces the current binary in-place. Between the current and target versions, the client displays changelogs from the releases list. If the client's current version is not found in the list (e.g. RC upgrading to stable), only the target version's changelog is shown.

### MCP Server (`mcp.go`)

Implements a local [Model Context Protocol](https://modelcontextprotocol.io/) server over stdio, enabling AI coding assistants (Cursor, VS Code Copilot, etc.) to upload files via Plik. Activated by `plik --mcp`.

Uses the official [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) (`mcp.StdioTransport`) and the `plik/` Go library for uploads.

**Tools:**
| Tool | Description |
|------|-------------|
| `upload_text` | Upload inline text content as a named file |
| `upload_file` | Upload a single file by path |
| `upload_files` | Upload multiple files by paths in a single upload |
| `server_info` | Get server version, config, capabilities, and profile info |
| `list_profiles` | List available profiles from `~/.plikrc` with their URLs |

**Prompts:** `upload_guide`

**Profile awareness:** All upload tools accept an optional `profile` parameter to target a different server. `clientForProfile()` always re-reads `~/.plikrc` from disk on every tool call, so edits (token rotation, URL changes, `--login`) take effect immediately without restarting the MCP server. It builds a `plik.Client` via `cfg.NewClient()`, which carries over all upload defaults (OneShot, TTL, Token, etc.) from the resolved config. `Stream` is cleared before creating the client to prevent indefinite blocking.

**Safety gate:** If the MCP server is started with `-P <profile>` (`ProfileSource == "flag"`), the `profile` parameter on tools is rejected — the server is locked to the startup profile(s). `DefaultProfile` and `PLIK_PROFILE` env do not lock.

**`loadPlikrc()`** (in `config.go`): Factored out of `LoadConfigFromFile` to allow `list_profiles` to read profile definitions without triggering the full resolution/merge logic.

---

## Tests

### Unit Tests
- `config_test.go` — TTL parsing, password splitting, boolean flags, config file loading, defaults
- `update_test.go` — auto-update disabled, quiet mode, unreachable server, missing platform binary
- `crypto/age/age_test.go` — recipient resolution, encryption round-trips

### Integration Tests (e2e)

End-to-end tests run against an ephemeral `plikd` server (started in `TestMain`):

| File | Coverage |
|------|---------|
| `setup_test.go` | Server lifecycle, helpers |
| `z1_e2e_basics_test.go` | Info, debug, single/multi file, custom name, stdin |
| `z2_e2e_options_test.go` | Oneshot, removable, stream, TTL, password, comments, quiet, JSON, not-secure, error paths |
| `z3_e2e_archive_test.go` | Tar (single, multi, dir, compression, options, name), zip (single, dir, options, name, dir+name) |
| `z4_e2e_crypto_test.go` | OpenSSL (auto/custom/prompted passphrase + decrypt round-trip, cipher, options), PGP (encrypt+decrypt), Age (passphrase + decrypt round-trip, recipient + decrypt) |
| `z5_e2e_profiles_test.go` | Profile upload (settings flow through), base config inheritance, info output with/without profiles |

Tests requiring external binaries (`tar`, `zip`, `gpg`, `age`, `openssl`) use `requireBinary()` to fail immediately if unavailable.

---

## Conventions

### Stderr for all non-data output

Because `--quiet` and `--json` modes reserve stdout exclusively for machine-readable data (file URLs or JSON), **all** informational, diagnostic, and error messages in the CLI must be written to **stderr** (`fmt.Fprintf(os.Stderr, ...)`). This includes:

- Passphrase display (crypto backends)
- Recipient resolution progress (age backend)
- Debug output
- Streaming download commands
- Archive/crypto error messages
- Progress bars (already write to stderr via the `pb` library)

Never use `fmt.Printf` / `fmt.Println` for non-data output in the CLI.
