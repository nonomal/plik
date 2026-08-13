package openssl

import (
	"github.com/root-gg/utils"
)

// Config for the openssl crypto backend.
//
// All fields are configurable via [SecureOptions] in .plikrc or CLI flags.
type Config struct {
	Openssl    string // Path to the openssl binary (default: /usr/bin/openssl)
	Cipher     string // Cipher algorithm (default: aes-256-cbc)
	Passphrase string // Encryption passphrase; auto-generated if empty
	Options    string // Additional openssl command line options (default: -md sha512 -pbkdf2 -iter 120000)
}

// NewOpenSSLBackendConfig instantiate a new Backend Configuration
// from config map passed as argument
func NewOpenSSLBackendConfig(params map[string]any) (config *Config) {
	config = new(Config)
	config.Openssl = "/usr/bin/openssl"
	config.Cipher = "aes-256-cbc"
	config.Options = "-md sha512 -pbkdf2 -iter 120000"
	utils.Assign(config, params)
	return
}
