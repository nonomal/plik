package archive

import (
	"errors"
	"io"

	"github.com/root-gg/plik/client/archive/tar"
	"github.com/root-gg/plik/client/archive/zip"
)

// Backend interface describe methods that the different
// types of archive backend must implement to work.
type Backend interface {
	Configure(arguments map[string]any) (err error)
	Archive(files []string) (reader io.Reader, err error)
	Comments() (comments string)
	GetFileName(files []string) (name string)
	GetConfiguration() any
}

// NewArchiveBackend instantiate the wanted archive backend with the name provided in configuration file
// We are passing its configuration found in .plikrc file or arguments
func NewArchiveBackend(name string, config map[string]any) (backend Backend, err error) {
	switch name {
	case "tar":
		backend, err = tar.NewTarBackend(config)
	case "zip":
		backend, err = zip.NewZipBackend(config)
	default:
		err = errors.New("Invalid archive backend")
	}
	return
}
