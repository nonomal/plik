package openssl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	Config *Config
	Stderr io.Writer // Diagnostic output writer (default: os.Stderr)
}

// NewOpenSSLBackend instantiate a new OpenSSL Crypto Backend
// and configure it from config map
func NewOpenSSLBackend(config map[string]any) (ob *Backend, err error) {
	ob = new(Backend)
	ob.Config = NewOpenSSLBackendConfig(config)
	ob.Stderr = os.Stderr
	ob.Config.Openssl, err = common.LookupBinary(ob.Config.Openssl, "openssl")
	return
}

// SetStderr sets the writer for diagnostic output.
func (ob *Backend) SetStderr(w io.Writer) {
	ob.Stderr = w
}

// Configure implementation for OpenSSL Crypto Backend
func (ob *Backend) Configure(arguments map[string]any) (err error) {
	if arguments["--openssl"] != nil && arguments["--openssl"].(string) != "" {
		ob.Config.Openssl = arguments["--openssl"].(string)
	}
	if arguments["--cipher"] != nil && arguments["--cipher"].(string) != "" {
		ob.Config.Cipher = arguments["--cipher"].(string)
	}
	if arguments["--passphrase"] != nil && arguments["--passphrase"].(string) != "" {
		ob.Config.Passphrase = arguments["--passphrase"].(string)
		if ob.Config.Passphrase == "-" {
			fmt.Fprintf(ob.Stderr, "Please enter a passphrase : ")
			_, err = fmt.Scanln(&ob.Config.Passphrase)
			if err != nil {
				return err
			}
		}
	} else {
		ob.Config.Passphrase = common.GenerateRandomID(25)
		fmt.Fprintln(ob.Stderr, "Passphrase : "+ob.Config.Passphrase)
	}
	if arguments["--secure-options"] != nil && arguments["--secure-options"].(string) != "" {
		ob.Config.Options = arguments["--secure-options"].(string)
	}

	return
}

// Encrypt implementation for OpenSSL Crypto Backend
func (ob *Backend) Encrypt(in io.Reader) (out io.Reader, err error) {
	passReader, passWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("unable to make pipe: %w", err)
	}
	_, err = passWriter.Write([]byte(ob.Config.Passphrase))
	if err != nil {
		return nil, fmt.Errorf("unable to write to pipe: %w", err)
	}
	err = passWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("unable to close pipe: %w", err)
	}

	out, writer := io.Pipe()

	var args []string
	args = append(args, ob.Config.Cipher)
	args = append(args, "-pass", "fd:3")
	args = append(args, strings.Fields(ob.Config.Options)...)

	go func() {
		var stderr bytes.Buffer
		cmd := exec.Command(ob.Config.Openssl, args...)
		cmd.Stdin = in                                      // fd:0
		cmd.Stdout = writer                                 // fd:1
		cmd.Stderr = &stderr                                // fd:2
		cmd.ExtraFiles = append(cmd.ExtraFiles, passReader) // fd:3
		err := cmd.Start()
		if err != nil {
			_ = writer.CloseWithError(fmt.Errorf("unable to start openssl cmd: %w", err))
			return
		}
		err = cmd.Wait()
		if err != nil {
			if stderr.Len() > 0 {
				_ = writer.CloseWithError(fmt.Errorf("openssl cmd failed: %w: %s", err, stderr.String()))
			} else {
				_ = writer.CloseWithError(fmt.Errorf("openssl cmd failed: %w", err))
			}
			return
		}

		_ = writer.Close()
	}()

	return out, nil
}

// Comments implementation for OpenSSL Crypto Backend
func (ob *Backend) Comments() string {
	return fmt.Sprintf("openssl %s -d -pass pass:%s %s", ob.Config.Cipher, ob.Config.Passphrase, ob.Config.Options)
}

// GetConfiguration implementation for OpenSSL Crypto Backend
func (ob *Backend) GetConfiguration() any {
	return ob.Config
}
