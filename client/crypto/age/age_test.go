package age

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveRecipients_X25519(t *testing.T) {
	// Valid native age recipient
	recipients, _, err := resolveRecipients(io.Discard, "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p", false)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestResolveRecipients_InvalidX25519(t *testing.T) {
	_, _, err := resolveRecipients(io.Discard, "not-a-valid-recipient", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid age recipient")
}

func TestResolveRecipients_SSHKey(t *testing.T) {
	// Valid SSH ed25519 key
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev test@example"
	recipients, _, err := resolveRecipients(io.Discard, key, false)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestResolveRecipients_InvalidSSHKey(t *testing.T) {
	_, _, err := resolveRecipients(io.Discard, "ssh-ed25519 invalid-key-data", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid SSH key")
}

func TestFetchKeys_SSHKey(t *testing.T) {
	// Test fetchKeys directly — parses SSH key from URL
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev test@example"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(key + "\n"))
	}))
	defer server.Close()

	recipients, err := fetchKeys(os.Stderr, server.URL)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestFetchKeys_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchKeys(io.Discard, server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "404 Not Found")
}

func TestFetchKeys_NoSupportedKeys(t *testing.T) {
	// Server returns only unsupported key types
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ecdsa-sha2-nistp256 AAAA...\n"))
	}))
	defer server.Close()

	_, err := fetchKeys(io.Discard, server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no supported keys")
}

func TestFetchKeys_MultipleSSHKeys(t *testing.T) {
	// Server returns multiple keys, some supported, some not
	keys := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev key1\n" +
		"ecdsa-sha2-nistp256 not-supported\n" +
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev key2\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(keys))
	}))
	defer server.Close()

	recipients, err := fetchKeys(os.Stderr, server.URL)
	require.NoError(t, err)
	require.Len(t, recipients, 2)
}

func TestFetchKeys_AgeKey(t *testing.T) {
	// Server returning a native age X25519 recipient
	ageKey := "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(ageKey + "\n"))
	}))
	defer server.Close()

	recipients, err := fetchKeys(io.Discard, server.URL)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestFetchKeys_MixedKeys(t *testing.T) {
	// Server returns both SSH and native age keys
	keys := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev key1\n" +
		"age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p\n" +
		"ecdsa-sha2-nistp256 not-supported\n" +
		"# comment line\n" +
		"\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(keys))
	}))
	defer server.Close()

	recipients, err := fetchKeys(os.Stderr, server.URL)
	require.NoError(t, err)
	require.Len(t, recipients, 2) // SSH key + age key, ecdsa skipped
}

func TestResolveRecipients_HTTPPromptDeclined(t *testing.T) {
	// Plain HTTP URL should trigger prompt; with no stdin input the default (false) is used
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev test\n"))
	}))
	defer server.Close()

	_, _, err := resolveRecipients(io.Discard, server.URL, false)
	require.Error(t, err)
	// AskConfirmation returns EOF from empty stdin in test → treated as error
	require.Contains(t, err.Error(), "unable to ask for confirmation")
}

func TestResolveRecipients_HTTPYesFlag(t *testing.T) {
	// Plain HTTP URL with yes=true should bypass the prompt
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(key + "\n"))
	}))
	defer server.Close()

	recipients, _, err := resolveRecipients(os.Stderr, server.URL, true)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestResolveRecipients_HTTPS(t *testing.T) {
	// HTTPS URL should not trigger the prompt
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev test@example"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(key + "\n"))
	}))
	defer server.Close()

	// httptest.NewTLSServer uses a self-signed cert; override the default HTTP client
	origTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = origTransport }()

	recipients, _, err := resolveRecipients(os.Stderr, server.URL, false)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
}

func TestResolveRecipients_SSHHostEmpty(t *testing.T) {
	_, _, err := resolveRecipients(io.Discard, "ssh://", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty SSH hostname")
}

func TestResolveRecipients_SSHHostUnreachable(t *testing.T) {
	// Port 1 should not have an SSH server
	_, _, err := resolveRecipients(os.Stderr, "ssh://127.0.0.1:1", false)
	require.Error(t, err)
}

func TestResolveRecipients_GitHubShorthand(t *testing.T) {
	_, _, err := resolveRecipients(io.Discard, "@", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty GitHub username")
}

func TestEncryptWithSSHRecipient(t *testing.T) {
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN6urz7ksxlezuqXw40WL7AK++XKSGAFZG95NZFgpzev test"
	recipients, _, err := resolveRecipients(io.Discard, key, false)
	require.NoError(t, err)

	backend := &Backend{Config: &Config{Recipients: recipients}}
	plaintext := "hello world"
	reader, err := backend.Encrypt(bytes.NewBufferString(plaintext))
	require.NoError(t, err)

	encrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)
	require.NotEqual(t, plaintext, string(encrypted))
	// Encrypted data should start with the age header
	require.Contains(t, string(encrypted), "age-encryption.org/v1")
}

func TestEncryptWithPassphrase(t *testing.T) {
	backend := &Backend{Config: &Config{Passphrase: "test-passphrase"}}
	plaintext := "hello world"
	reader, err := backend.Encrypt(bytes.NewBufferString(plaintext))
	require.NoError(t, err)

	encrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)
	require.Contains(t, string(encrypted), "age-encryption.org/v1")
}
