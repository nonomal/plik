package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

// ---------- OpenSSL tests ----------

func TestCLI_OpenSSL_AutoPassphrase(t *testing.T) {
	requireBinary(t, "openssl")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":     []string{dir + "/FILE1"},
		"--secure": "openssl",
	})

	// Stderr should contain the auto-generated passphrase
	require.Contains(t, result.Stderr, "Passphrase : ",
		"stderr should display the auto-generated passphrase")

	// Stdout should contain the download command with the passphrase
	require.Contains(t, result.Stdout, "openssl")
	require.Contains(t, result.Stdout, "pass:",
		"download command should contain passphrase")

	// Extract passphrase from stderr and verify it matches the download command
	for line := range strings.SplitSeq(result.Stderr, "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "Passphrase : "); ok {
			require.Contains(t, result.Stdout, after,
				"download command passphrase should match displayed passphrase")
			break
		}
	}
}

func TestCLI_OpenSSL_CustomPassphrase(t *testing.T) {
	requireBinary(t, "openssl")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":         []string{dir + "/FILE1"},
		"--secure":     "openssl",
		"--passphrase": "foobar",
	})

	// Custom passphrase should NOT trigger "Passphrase : " display
	require.NotContains(t, result.Stderr, "Passphrase : ",
		"custom passphrase should not be displayed to stderr")

	// But the download command should contain it
	require.Contains(t, result.Stdout, "pass:foobar",
		"download command should contain custom passphrase")

	// Full decrypt round-trip
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	encBytes := downloadFileBytes(t, fileURL)

	decrypted := opensslDecrypt(t, encBytes, "foobar")
	require.Equal(t, testContent, string(decrypted))
}

func TestCLI_OpenSSL_PromptedPassphrase(t *testing.T) {
	requireBinary(t, "openssl")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	// Simulate prompted passphrase by piping "foobar" to stdin
	// --passphrase - reads from stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		w.Write([]byte("foobar\n"))
		w.Close()
	}()

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":         []string{dir + "/FILE1"},
		"--secure":     "openssl",
		"--passphrase": "-",
	})

	// Should prompt for passphrase on stderr
	require.Contains(t, result.Stderr, "Please enter a passphrase",
		"stderr should contain stdin passphrase prompt")

	// Download command should contain the entered passphrase
	require.Contains(t, result.Stdout, "pass:foobar",
		"download command should contain the prompted passphrase")
}

func TestCLI_OpenSSL_CustomCipher(t *testing.T) {
	requireBinary(t, "openssl")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":         []string{dir + "/FILE1"},
		"--secure":     "openssl",
		"--cipher":     "aes-128-cbc",
		"--passphrase": "test",
	})

	require.Contains(t, result.Stdout, "aes-128-cbc",
		"download command should reference the custom cipher")
}

func TestCLI_OpenSSL_CustomOptions(t *testing.T) {
	requireBinary(t, "openssl")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":             []string{dir + "/FILE1"},
		"--secure":         "openssl",
		"--secure-options": "-a",
	})

	// Download the encrypted file and verify it's base64 encoded (ASCII text)
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	encrypted := downloadFileContent(t, fileURL)

	// Base64 encoded content should be printable ASCII
	for _, c := range encrypted {
		if c > 127 {
			// Binary content found — the -a flag should have made it base64
			t.Fatalf("expected base64 (ASCII) content with -a option, got binary at char %d", c)
		}
	}
}

// ---------- PGP tests ----------

// testPGPKey is the test PGP private key (same as test.sh)
const testPGPKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----
Version: GnuPG v1

lQHYBFYLA7cBBACgMFkOEqqWop6bQYp4LGq0A79XakKj1vYVYom7Jg+V9utPQsK3
29rKzSBAiq2yWAQLNyJ6dpyFctabSU1pJ/OAsvssuMxio/M6Kf+mvMHmAjyW9s5F
fqeZROKjyOswFwkPKS36ifOKm2CigfJlMavV74h3p4f9JznWYn0MBrCDMQARAQAB
AAP+I5ZKGonEEx4CjWxUllkLxX01o3ZsYpitZ9fR0F1mxgKqiRvERXNW2ooSmbQV
XZMXJuSzSLCUGkOGcM4qn+truXhE3vbxEtyNpKQP1ae/m2zLcUJk3JJWWmnt/BPD
JSRsOtOGovoasKVxK58BQ1UfkV0Mred31NxYVvU3LxAhCAkCAMRt1aWgfWgAA007
j+wDtXz9qeAuBQ8jb5OZ/O0WU9mNfGauFVeby0Mld0m7ofUyAVdIGn26woY/hPCL
zCMcDZ0CANDE7vyYHRWcNclETzqkDuCKB/MG62jPRerF9QLKozDcP8fu8xm//iYd
K8v3UTHbhZ5X2wvcyMIQxm7Iov18oaUB/iJH5jUlhyHZOWIZXJ/xpV/fFZr+DIAi
kL7KP+nknyFrP4czcfzhSLkQKyU67ODkPfxqltf7SVZlnqk5GKpXB4WfBbQrcGxp
ay5yb290LmdnIChwbGlrIHRlc3Qga2V5KSA8cGxpa0Byb290LmdnPoi4BBMBAgAi
BQJWCwO3AhsDBgsJCAcDAgYVCAIJCgsEFgIDAQIeAQIXgAAKCRBqKML4FUjUXW18
A/9Rutp5SWnk+Vi4nbFh2QAyl5rwDdF45mzZBY1DQsBzpkREg8URvBLN0lpWDr4k
4Mm7ONIqmAta23NvPe4yR1f68Q2SGsheUbL27vGcbQ/bY1pkzTRSGZFnWu3Q6Oo2
0gOo8b0HsbJMF4VJvwmAhXk+IiIbUpQ0Zep27BwQWagmDp0B2ARWCwO3AQQAp1GU
ZsAPOUgtm/gLA6fYf9OuUvaUprVL7GBpdhjIA1r5syJrCxRtWocvxH+EMHgF6CCq
Qe03PODI8NhjK1zCCZr1CQRontD1a59CHdSFk+2eTa40CNsJ17f16eiDwE8GvhNT
T/ZGEztS9b5uCp0higrAqtKTvx0NsX/V3juJBtMAEQEAAQAD/1OARJn0voQ9T7m3
U7Pa15KPjz+LHIuIDeBlCyyrWGJITDZIdnhclOhpb/7WDp/rvjLm3mExY/BHVDDS
JMe2roSyraBj86SnejJDuA1JWhCLvBF8bQKrXNMdKH76gAdcT++tEuYMRmlur22z
PcW+FDZspr4lRn33AZPtHn21mrdNAgDCeasfTHlgXrOQ9o/iNVC9tfxFjVEe3JCC
nOaoBNNpDrOe8xpJ2amYbW0I3KkoYx2Q2hZKuIgj88WyoGdiirRnAgDcQId88AcH
vKPKunU4oFfePtqLjX5s5TKffcqmTtQRW0sqcoo8tNECXq8lsk9PUoihs15Ux2X4
r6LexhGrMHa1AfwKZJBWXwoxzUKYVQW3qRh1MokzHS+ZLT25w/7Co/IF+CjLeSiU
IzKv2YBRXHV9YeTpqUxFSGOyIiAgC6kap0HVnbCInwQYAQIACQUCVgsDtwIbDAAK
CRBqKML4FUjUXRfBA/4pLcWcBOJ8suh7kTgmicZA55bAbY+CTnNlHma7pzW1rcqD
TojG/RllyilI8QHfR9+da/iEGoAcY8eTgpAYZfNnd8tCy1bQQM+YkjAgh7lFEUdV
Wslu8jCqJpbcKUL7k2mfTKwJ97h1Go5LMurSR9W2psZrmyHXbCccu0CghK/Y7g==
=/xyA
-----END PGP PRIVATE KEY BLOCK-----`

func TestCLI_PGP(t *testing.T) {
	requireBinary(t, "gpg")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	// Set up a temporary GPG home
	gpgHome := t.TempDir()

	// Import the test key
	keyPath := filepath.Join(gpgHome, "pgp.key")
	require.NoError(t, os.WriteFile(keyPath, []byte(testPGPKey), 0600))

	cmd := exec.Command("gpg", "--homedir", gpgHome, "--batch", "--import", keyPath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "gpg import failed: %s", string(out))

	// Export the public key in old OpenPGP binary format (pubring.gpg)
	// GPG v2 stores keys in kbx format which Go's openpgp library cannot read
	keyring := filepath.Join(gpgHome, "pubring.gpg")
	exportOut, err := exec.Command("gpg", "--homedir", gpgHome, "--batch", "--export").Output()
	require.NoError(t, err, "gpg export failed")
	require.NotEmpty(t, exportOut, "gpg export produced no output")
	require.NoError(t, os.WriteFile(keyring, exportOut, 0644))

	// Set GNUPGHOME so gpg uses our temp keyring
	t.Setenv("GNUPGHOME", gpgHome)

	config := newTestConfig()
	config.Quiet = false
	config.SecureOptions["Keyring"] = keyring

	result := runCLI(t, config, map[string]any{
		"FILE":        []string{dir + "/FILE1"},
		"--secure":    "pgp",
		"--recipient": "plik.root.gg",
	})

	// Download encrypted file
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	encrypted := downloadFileContent(t, fileURL)

	// Decrypt with gpg
	encFile := filepath.Join(dir, "encrypted")
	require.NoError(t, os.WriteFile(encFile, []byte(encrypted), 0644))

	decryptCmd := exec.Command("gpg", "--homedir", gpgHome, "--batch", "--yes", "--decrypt", encFile)
	decrypted, err := decryptCmd.Output()
	require.NoError(t, err, "gpg decrypt failed")
	require.Equal(t, testContent, string(decrypted))
}

// ---------- Age tests ----------

func TestCLI_Age_AutoPassphrase(t *testing.T) {
	requireBinary(t, "age")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":     []string{dir + "/FILE1"},
		"--secure": "age",
	})

	// Stderr should contain the auto-generated passphrase
	require.Contains(t, result.Stderr, "Passphrase : ",
		"stderr should display the auto-generated passphrase")

	// Stdout should contain the age decrypt command
	require.Contains(t, result.Stdout, "age --decrypt",
		"output should contain age decrypt command")

	// Verify the uploaded file has age encryption header
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	encBytes := downloadFileBytes(t, fileURL)
	require.True(t, strings.HasPrefix(string(encBytes), "age-encryption.org"),
		"encrypted file should start with age-encryption.org header")

	// Full round-trip: extract passphrase from stderr and decrypt
	var passphrase string
	for line := range strings.SplitSeq(result.Stderr, "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "Passphrase : "); ok {
			passphrase = strings.TrimSpace(after)
			break
		}
	}
	require.NotEmpty(t, passphrase, "should have extracted passphrase from stderr")

	decrypted := agePassphraseDecrypt(t, encBytes, passphrase)
	require.Equal(t, testContent, string(decrypted),
		"decrypted content should match original")
}

func TestCLI_Age_CustomPassphrase(t *testing.T) {
	requireBinary(t, "age")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":         []string{dir + "/FILE1"},
		"--secure":     "age",
		"--passphrase": "foobar",
	})

	// Custom passphrase should NOT trigger "Passphrase : " display
	require.NotContains(t, result.Stderr, "Passphrase : ",
		"custom passphrase should not be displayed to stderr")

	require.Contains(t, result.Stdout, "age --decrypt",
		"output should contain age decrypt command")

	// Full decrypt round-trip
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	encBytes := downloadFileBytes(t, fileURL)
	require.True(t, strings.HasPrefix(string(encBytes), "age-encryption.org"),
		"encrypted file should start with age-encryption.org header")

	decrypted := agePassphraseDecrypt(t, encBytes, "foobar")
	require.Equal(t, testContent, string(decrypted))
}

func TestCLI_Age_Recipient(t *testing.T) {
	requireBinary(t, "age")
	requireBinary(t, "age-keygen")

	dir := t.TempDir()
	createTestFile(t, dir, "FILE1", testContent)

	// Generate age keypair
	identityPath := filepath.Join(dir, "age_identity.txt")
	cmd := exec.Command("age-keygen", "-o", identityPath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "age-keygen failed: %s", string(out))

	// Extract public key from identity file
	identityData, err := os.ReadFile(identityPath)
	require.NoError(t, err)

	var recipient string
	for line := range strings.SplitSeq(string(identityData), "\n") {
		if after, ok := strings.CutPrefix(line, "# public key: "); ok {
			recipient = after
			break
		}
	}
	require.NotEmpty(t, recipient, "should have extracted age public key")

	config := newTestConfig()
	config.Quiet = false

	result := runCLI(t, config, map[string]any{
		"FILE":        []string{dir + "/FILE1"},
		"--secure":    "age",
		"--recipient": recipient,
	})

	// Recipient mode should NOT display any passphrase
	require.NotContains(t, result.Stderr, "Passphrase : ",
		"recipient mode should not display a passphrase")

	// Download and decrypt with identity file
	fileURL := extractFileURLFromOutput(t, result.Stdout)
	encrypted := downloadFileContent(t, fileURL)

	encFile := filepath.Join(dir, "encrypted")
	require.NoError(t, os.WriteFile(encFile, []byte(encrypted), 0644))

	decryptCmd := exec.Command("age", "--decrypt", "-i", identityPath, encFile)
	decrypted, err := decryptCmd.Output()
	require.NoError(t, err, "age decrypt failed")
	require.Equal(t, testContent, string(decrypted))
}

// agePassphraseDecrypt decrypts age scrypt-encrypted data using the Go library.
// The age CLI prompts on tty and can't be scripted, so we use the library directly.
func agePassphraseDecrypt(t *testing.T, encBytes []byte, passphrase string) []byte {
	t.Helper()

	identity, err := age.NewScryptIdentity(passphrase)
	require.NoError(t, err)

	reader, err := age.Decrypt(bytes.NewReader(encBytes), identity)
	require.NoError(t, err, "age decrypt failed")

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	return decrypted
}

// opensslDecrypt decrypts data using the openssl CLI with the default
// cipher (aes-256-cbc) and options (-md sha512 -pbkdf2 -iter 120000)
// matching the openssl backend's defaults.
func opensslDecrypt(t *testing.T, encBytes []byte, passphrase string) []byte {
	t.Helper()

	decryptCmd := exec.Command("openssl", "aes-256-cbc", "-d",
		"-pass", "pass:"+passphrase,
		"-md", "sha512", "-pbkdf2", "-iter", "120000")
	decryptCmd.Stdin = bytes.NewReader(encBytes)
	var stdout, stderr bytes.Buffer
	decryptCmd.Stdout = &stdout
	decryptCmd.Stderr = &stderr
	err := decryptCmd.Run()
	require.NoError(t, err, "openssl decrypt failed: %s", stderr.String())
	return stdout.Bytes()
}
