package tar

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	Config *BackendConfig
}

// NewTarBackend instantiate a new Tar Archive Backend
// and configure it from config map
func NewTarBackend(config map[string]any) (tb *Backend, err error) {
	tb = new(Backend)
	tb.Config = NewTarBackendConfig(config)
	tb.Config.Tar, err = common.LookupBinary(tb.Config.Tar, "tar")
	return
}

// Configure implementation for TAR Archive Backend
func (tb *Backend) Configure(arguments map[string]any) (err error) {
	if arguments["--compress"] != nil && arguments["--compress"].(string) != "" {
		tb.Config.Compress = arguments["--compress"].(string)
	}
	if arguments["--archive-options"] != nil && arguments["--archive-options"].(string) != "" {
		tb.Config.Options = arguments["--archive-options"].(string)
	}
	return
}

// Archive implementation for TAR Archive Backend
func (tb *Backend) Archive(files []string) (reader io.Reader, err error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("unable to make a tar archive from STDIN")
	}

	var args []string
	args = append(args, "--create")
	if tb.Config.Compress != "no" {
		args = append(args, "--"+tb.Config.Compress)
	}
	args = append(args, strings.Fields(tb.Config.Options)...)
	args = append(args, files...)

	reader, writer := io.Pipe()

	var stderr bytes.Buffer
	cmd := exec.Command(tb.Config.Tar, args...)
	cmd.Stdout = writer
	cmd.Stderr = &stderr

	go func() {
		err := cmd.Start()
		if err != nil {
			_ = writer.CloseWithError(fmt.Errorf("unable to start tar cmd: %w", err))
			return
		}
		err = cmd.Wait()
		if err != nil {
			if stderr.Len() > 0 {
				_ = writer.CloseWithError(fmt.Errorf("tar cmd failed: %w: %s", err, stderr.String()))
			} else {
				_ = writer.CloseWithError(fmt.Errorf("tar cmd failed: %w", err))
			}
			return
		}
		_ = writer.Close()
	}()

	return reader, nil
}

// Comments implementation for TAR Archive Backend
func (tb *Backend) Comments() string {
	comment := "tar xvf -"
	if tb.Config.Compress != "no" {
		comment += " --" + tb.Config.Compress
	}

	return comment
}

// GetConfiguration implementation for TAR Archive Backend
func (tb *Backend) GetConfiguration() any {
	return tb.Config
}

// GetFileName returns the final archive file name
func (tb *Backend) GetFileName(files []string) (name string) {
	name = "archive"
	if len(files) == 1 {
		name = filepath.Base(files[0])
	}
	name += ".tar" + getCompressExtension(tb.Config.Compress)
	return
}

func getCompressExtension(mode string) string {
	switch mode {
	case "gzip":
		return ".gz"
	case "bzip2":
		return ".bz2"
	case "xz":
		return ".xz"
	case "lzip":
		return ".lz"
	case "lzop":
		return ".lzo"
	case "lzma":
		return ".lzma"
	case "compress":
		return ".Z"
	default:
		return ""
	}
}
