package pgp

import (
	"os"

	"github.com/root-gg/utils"
	"golang.org/x/crypto/openpgp"
)

// Config for the pgp crypto backend.
//
// User-facing fields (configurable via [SecureOptions] in .plikrc or CLI flags):
//   - Recipient: name or email to search for in the keyring (required)
//   - Keyring:   path to the GnuPG public keyring (default: $GNUPGHOME/pubring.gpg or ~/.gnupg/pubring.gpg)
type Config struct {
	// --- User-configurable (via [SecureOptions] or CLI flags) ---
	Keyring   string // Path to GnuPG public keyring (default: $GNUPGHOME/pubring.gpg or ~/.gnupg/pubring.gpg)
	Recipient string // Name or email to search in keyring (required)

	// --- Runtime state (not configurable via .plikrc) ---
	Email  string          // Resolved email of the matched key; set during Configure()
	Entity *openpgp.Entity // Resolved PGP entity for encryption; set during Configure()
}

// NewPgpBackendConfig instantiate a new Backend Configuration
// from config map passed as argument
func NewPgpBackendConfig(params map[string]any) (config *Config) {
	config = new(Config)
	gnupgHome := os.Getenv("GNUPGHOME")
	if gnupgHome == "" {
		gnupgHome = os.Getenv("HOME") + "/.gnupg"
	}
	config.Keyring = gnupgHome + "/pubring.gpg"
	utils.Assign(config, params)
	return
}
