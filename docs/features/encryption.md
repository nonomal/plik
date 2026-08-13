# End-to-End Encryption

Plik supports optional end-to-end encryption using [age](https://age-encryption.org/). When enabled, files are encrypted **client-side** before upload — the server only stores encrypted bytes and never has access to the plaintext or passphrase.

## Web UI

Toggle **End-to-End Encryption** in the upload sidebar. A passphrase is auto-generated and can be customized. After upload, the passphrase is displayed in the share section with a toggle to include it in the shareable link.

## CLI

```bash
plik --secure                         # Auto-generated passphrase (default)
plik --secure -p "passphrase"         # Custom passphrase
plik --secure -r @camathieu           # Encrypt for a GitHub user's SSH keys
plik --secure -r https://gitlab.com/user.keys  # SSH keys from any URL
plik --secure -r "ssh-ed25519 AAAA..."         # Raw SSH public key
plik --secure -r ssh://myserver.example.com     # Encrypt for a server's SSH host key
plik --secure -r age1...              # Native age X25519 recipient
```

The `@username` shorthand fetches SSH keys from `https://github.com/{username}.keys`. You can also provide any URL that serves public keys (one key per line). Supported key types: `ssh-rsa`, `ssh-ed25519`, and native `age1…` X25519 recipients. The `ssh://hostname` format scans the server's SSH host key directly.

::: warning Plain HTTP
Fetching keys over plain `http://` triggers a security warning — an attacker could substitute their own key (MITM). Use `https://` whenever possible. The prompt defaults to **No**; use `--yes` to bypass it in non-interactive scripts.
:::

## Configuration

Encryption defaults can be configured in `~/.plikrc` to avoid repeating flags. Use `SecureOptions` for backend-specific settings and profiles for reusable encryption presets.

### Setting defaults

```toml
# Always encrypt by default
Secure = true
SecureMethod = "age"
```

### Encryption profile

Create a profile to encrypt for a specific recipient:

```toml
[Profiles.bob]
Secure = true
SecureMethod = "age"
[Profiles.bob.SecureOptions]
Recipient = "@bob"   # github username
```

Then use it:

```bash
plik -P bob secret.txt          # encrypt for @bob
plik -P work,bob secret.txt     # use work server + encrypt for @bob
```

See [CLI Client > SecureOptions](/features/cli-client#secureoptions) for the full reference of available keys per backend.

## Interoperability

Files encrypted via the web UI can be decrypted with the CLI and vice versa:

```bash
# Decrypt a passphrase-encrypted file
curl <file_url> | age --decrypt

# Decrypt a recipient-encrypted file (e.g. GitHub SSH key)
curl <file_url> | age --decrypt -i ~/.ssh/id_ed25519

# Decrypt a file encrypted for a server's SSH host key (requires root)
curl <file_url> | age --decrypt -i /etc/ssh/ssh_host_ed25519_key > secret_file
```

::: tip Use Case: Server-Only Secrets
Encrypt a file so that only a specific server can decrypt it — great for deploying secrets, config files, or credentials that only the target machine should be able to read.
:::

## How It Works

1. **Upload**: Files are encrypted client-side using `age` before being sent to the server
2. **Storage**: The server stores only encrypted bytes and sets `Content-Type: application/octet-stream`
3. **Download**: The web UI prompts for the passphrase and decrypts in the browser; CLI tools receive raw encrypted bytes
4. **Passphrase**: Never stored server-side — shared via the URL fragment, copy button, or out-of-band

## Feature Flag

Control E2EE availability via `FeatureE2EE` in `plikd.cfg`:

| Value | Behavior |
|-------|----------|
| `disabled` | E2EE toggle hidden in web UI |
| `enabled` | E2EE available, off by default (default) |
| `default` | E2EE available, on by default |
| `forced` | E2EE always on, cannot be disabled |

::: tip
E2EE is independent of server-side password protection. Password protection controls access to the upload metadata and download URLs; E2EE protects the file contents themselves.
:::
