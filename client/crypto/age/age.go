package age

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"

	"github.com/root-gg/plik/server/common"
)

var randRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// Backend object
type Backend struct {
	Config *Config
	Stderr io.Writer // Diagnostic output writer (default: os.Stderr)
}

// NewAgeBackend instantiate a new Age Backend
// from config map passed as argument
func NewAgeBackend(config map[string]any) (backend *Backend) {
	backend = new(Backend)
	backend.Config = NewAgeBackendConfig(config)
	backend.Stderr = os.Stderr
	return
}

// SetStderr sets the writer for diagnostic output.
func (ab *Backend) SetStderr(w io.Writer) {
	ab.Stderr = w
}

// Configure implementation for Age Backend
func (ab *Backend) Configure(arguments map[string]any) (err error) {
	// Check for passphrase from arguments (--passphrase flag)
	if passphrase, ok := arguments["--passphrase"]; ok {
		if passphrase != nil {
			ab.Config.Passphrase = passphrase.(string)
		}
	}

	// Check for recipient from arguments (--recipient flag)
	if recipient, ok := arguments["--recipient"]; ok {
		if recipient != nil {
			ab.Config.Recipient = recipient.(string)
		}
	}

	// Check for --yes flag (auto-accept confirmation prompts)
	if yes, ok := arguments["--yes"]; ok && yes.(bool) {
		ab.Config.Yes = true
	}

	// Cannot use both passphrase and recipient
	if ab.Config.Passphrase != "" && ab.Config.Recipient != "" {
		return fmt.Errorf("cannot use both --passphrase and --recipient with age backend")
	}

	// If no passphrase or recipient specified, generate a random passphrase
	if ab.Config.Passphrase == "" && ab.Config.Recipient == "" {
		ab.Config.Passphrase = generatePassphrase(32)
		fmt.Fprintf(ab.Stderr, "Passphrase : %s\n\n", ab.Config.Passphrase)
	}

	// Resolve recipient to age.Recipient objects
	if ab.Config.Recipient != "" {
		recipients, decryptHint, err := resolveRecipients(ab.Stderr, ab.Config.Recipient, ab.Config.Yes)
		if err != nil {
			return fmt.Errorf("unable to resolve recipient: %s", err)
		}
		ab.Config.Recipients = recipients
		ab.Config.DecryptHint = decryptHint
	}

	return nil
}

// resolveRecipients resolves a recipient string to one or more age.Recipient objects.
//
// Supported formats:
//   - "@username"       → fetch SSH keys from https://github.com/{username}.keys
//   - "ssh://host"      → scan SSH host key from server
//   - "https://..."     → fetch SSH keys from arbitrary URL
//   - "http://..."      → fetch SSH keys from arbitrary URL
//   - "ssh-rsa ..."     → parse as SSH public key
//   - "ssh-ed25519 ..." → parse as SSH public key
//   - "age1..."         → parse as native age X25519 recipient
func resolveRecipients(stderr io.Writer, recipient string, yes bool) ([]age.Recipient, string, error) {
	switch {
	case strings.HasPrefix(recipient, "@"):
		// GitHub shorthand: @username → https://github.com/{username}.keys
		username := strings.TrimPrefix(recipient, "@")
		if username == "" {
			return nil, "", fmt.Errorf("empty GitHub username")
		}
		url := fmt.Sprintf("https://github.com/%s.keys", username)
		fmt.Fprintf(stderr, "Fetching SSH keys for @%s from %s\n", username, url)
		recipients, err := fetchKeys(stderr, url)
		return recipients, "", err

	case strings.HasPrefix(recipient, "ssh://"):
		// SSH host key scanning: ssh://hostname[:port]
		host := strings.TrimPrefix(recipient, "ssh://")
		if host == "" {
			return nil, "", fmt.Errorf("empty SSH hostname")
		}
		if !strings.Contains(host, ":") {
			host = host + ":22"
		}
		fmt.Fprintf(stderr, "Scanning SSH host key from %s\n", host)
		key, keyType, err := fetchHostKey(host)
		if err != nil {
			return nil, "", err
		}
		var keyPath string
		switch keyType {
		case ssh.KeyAlgoED25519:
			keyPath = "/etc/ssh/ssh_host_ed25519_key"
		case ssh.KeyAlgoRSA:
			keyPath = "/etc/ssh/ssh_host_rsa_key"
		}
		fmt.Fprintf(stderr, "Found %s host key\n", keyType)
		r, err := agessh.ParseRecipient(key)
		if err != nil {
			return nil, "", fmt.Errorf("invalid SSH host key: %s", err)
		}
		return []age.Recipient{r}, fmt.Sprintf("age --decrypt -i %s", keyPath), nil

	case strings.HasPrefix(recipient, "https://") || strings.HasPrefix(recipient, "http://"):
		// Arbitrary URL containing recipient public keys
		if strings.HasPrefix(recipient, "http://") {
			fmt.Fprintf(stderr, "\nWARNING: Fetching keys over plain HTTP (%s)\n", recipient)
			fmt.Fprintf(stderr, "This is vulnerable to man-in-the-middle attacks.\n")
			fmt.Fprintf(stderr, "An attacker could substitute their own key and decrypt your files.\n")
			if yes {
				fmt.Fprintf(stderr, "Continuing (--yes)\n")
			} else {
				fmt.Fprintf(stderr, "Continue? [y/N] : ")
				ok, err := common.AskConfirmation(false)
				if err != nil {
					return nil, "", fmt.Errorf("unable to ask for confirmation: %s", err)
				}
				if !ok {
					return nil, "", fmt.Errorf("aborted: user declined insecure HTTP key fetch")
				}
			}
		}
		fmt.Fprintf(stderr, "Fetching keys from %s\n", recipient)
		recipients, err := fetchKeys(stderr, recipient)
		return recipients, "", err

	case strings.HasPrefix(recipient, "ssh-"):
		// Raw SSH public key
		r, err := agessh.ParseRecipient(recipient)
		if err != nil {
			return nil, "", fmt.Errorf("invalid SSH key: %s", err)
		}
		return []age.Recipient{r}, "", nil

	default:
		// Native age X25519 recipient
		r, err := age.ParseX25519Recipient(recipient)
		if err != nil {
			return nil, "", fmt.Errorf("invalid age recipient: %s", err)
		}
		return []age.Recipient{r}, "", nil
	}
}

// fetchKeys fetches public keys from a URL and parses them as age recipients.
// Each line is expected to be either an SSH public key (ssh-rsa, ssh-ed25519)
// in authorized_keys format, or a native age recipient (age1...).
func fetchKeys(stderr io.Writer, url string) ([]age.Recipient, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch keys from %s: %s", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no keys found at %s (404 Not Found)", url)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch keys from %s: HTTP %d", url, resp.StatusCode)
	}

	var recipients []age.Recipient
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try SSH key first (ssh-rsa, ssh-ed25519)
		if strings.HasPrefix(line, "ssh-") {
			r, err := agessh.ParseRecipient(line)
			if err == nil {
				recipients = append(recipients, r)
				continue
			}
		}

		// Try native age recipient (age1...)
		if strings.HasPrefix(line, "age1") {
			r, err := age.ParseX25519Recipient(line)
			if err == nil {
				recipients = append(recipients, r)
				continue
			}
		}

		// Skip unsupported key types
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading keys from %s: %s", url, err)
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no supported keys found at %s (supported: ssh-rsa, ssh-ed25519, age1...)", url)
	}

	fmt.Fprintf(stderr, "Found %d supported key(s)\n", len(recipients))

	return recipients, nil
}

// fetchHostKey connects to an SSH server and captures its host key.
// It prefers ed25519 over RSA and aborts the connection immediately after capturing the key.
// Returns the key in authorized_keys format and the key algorithm name.
func fetchHostKey(target string) (string, string, error) {
	var capturedKey ssh.PublicKey

	config := &ssh.ClientConfig{
		User: "keyscan",
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoRSA,
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			keyType := key.Type()
			if keyType == ssh.KeyAlgoED25519 || keyType == ssh.KeyAlgoRSA {
				capturedKey = key
			}
			return fmt.Errorf("key captured, aborting connection")
		},
	}

	// Dial will always fail (we abort in HostKeyCallback), so we ignore the error
	_, _ = ssh.Dial("tcp", target, config)

	if capturedKey == nil {
		return "", "", fmt.Errorf("no compatible SSH host key found on %s (age supports ssh-ed25519 and ssh-rsa)", target)
	}

	formattedKey := string(bytes.TrimSpace(ssh.MarshalAuthorizedKey(capturedKey)))
	return formattedKey, capturedKey.Type(), nil
}

// Encrypt implementation for Age Backend
func (ab *Backend) Encrypt(in io.Reader) (out io.Reader, err error) {
	out, writer := io.Pipe()

	go func() {
		var w io.WriteCloser
		var encErr error

		if len(ab.Config.Recipients) > 0 {
			// SSH or X25519 recipient(s) resolved during Configure
			w, encErr = age.Encrypt(writer, ab.Config.Recipients...)
		} else {
			// Passphrase (scrypt) mode
			recipient, err := age.NewScryptRecipient(ab.Config.Passphrase)
			if err != nil {
				_ = writer.CloseWithError(fmt.Errorf("failed to create scrypt recipient: %s", err))
				return
			}
			w, encErr = age.Encrypt(writer, recipient)
		}

		if encErr != nil {
			_ = writer.CloseWithError(fmt.Errorf("failed to initialize age encryption: %s", encErr))
			return
		}

		_, copyErr := io.Copy(w, in)
		closeErr := w.Close()
		if copyErr != nil {
			_ = writer.CloseWithError(copyErr)
		} else if closeErr != nil {
			_ = writer.CloseWithError(closeErr)
		} else {
			_ = writer.Close()
		}
	}()

	return out, nil
}

// Comments implementation for Age Backend
func (ab *Backend) Comments() string {
	if ab.Config.DecryptHint != "" {
		return ab.Config.DecryptHint
	}
	if len(ab.Config.Recipients) > 0 {
		return "age --decrypt -i <private_key>"
	}
	return "age --decrypt"
}

// GetConfiguration implementation for Age Backend
func (ab *Backend) GetConfiguration() any {
	return ab.Config
}

// generatePassphrase generates a cryptographically secure random passphrase
func generatePassphrase(length int) string {
	max := big.NewInt(int64(len(randRunes)))
	b := make([]rune, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(fmt.Sprintf("failed to generate random passphrase: %s", err))
		}
		b[i] = randRunes[n.Int64()]
	}
	return string(b)
}
