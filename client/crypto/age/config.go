package age

import (
	"filippo.io/age"
	"github.com/root-gg/utils"
)

// Config for the age crypto backend.
//
// User-facing fields (configurable via [SecureOptions] in .plikrc or CLI flags):
//   - Passphrase: symmetric encryption passphrase (auto-generated if omitted)
//   - Recipient:  asymmetric recipient (@github_user, ssh://host, URL, ssh key, or age1…)
//
// Passphrase and Recipient are mutually exclusive.
type Config struct {
	// --- User-configurable (via [SecureOptions] or CLI flags) ---
	Passphrase string // Encryption passphrase; auto-generated if both Passphrase and Recipient are empty
	Recipient  string // Recipient: @github_user, ssh://host, URL, ssh public key, or native age1… key

	// --- Runtime state (not configurable via .plikrc) ---
	Yes         bool            // Auto-accept confirmation prompts (from --yes flag)
	Recipients  []age.Recipient `json:"-"` // Resolved age.Recipient objects; set during Configure()
	DecryptHint string          `json:"-"` // Decrypt command hint for Comments(); set during Configure()
}

// NewAgeBackendConfig instantiate a new Backend Configuration
// from config map passed as argument
func NewAgeBackendConfig(params map[string]any) (config *Config) {
	config = new(Config)
	utils.Assign(config, params)
	return
}
