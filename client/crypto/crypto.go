package crypto

import (
	"errors"
	"io"

	agebackend "github.com/root-gg/plik/client/crypto/age"
	"github.com/root-gg/plik/client/crypto/openssl"
	"github.com/root-gg/plik/client/crypto/pgp"
)

// Backend interface describe methods that the different
// types of crypto backend must implement to work.
type Backend interface {
	Configure(arguments map[string]any) (err error)
	Encrypt(in io.Reader) (out io.Reader, err error)
	Comments() string
	GetConfiguration() any
	SetStderr(w io.Writer)
}

// NewCryptoBackend instantiate the wanted archive backend with the name provided in configuration file
// We are passing its configuration found in .plikrc file or arguments
func NewCryptoBackend(name string, config map[string]any) (backend Backend, err error) {
	switch name {
	case "openssl":
		backend, err = openssl.NewOpenSSLBackend(config)
	case "pgp":
		backend = pgp.NewPgpBackend(config)
	case "age":
		backend = agebackend.NewAgeBackend(config)
	default:
		err = errors.New("Invalid crypto backend")
	}
	return
}
