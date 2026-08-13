package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/root-gg/utils"

	"github.com/root-gg/plik/client/archive"
	"github.com/root-gg/plik/client/crypto"
	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

// PlikCLI holds the CLI runtime state, encapsulating what was
// previously scattered across package-level variables.
type PlikCLI struct {
	Config         *CliConfig
	Arguments      map[string]any
	ArchiveBackend archive.Backend
	CryptoBackend  crypto.Backend
	Stdout         io.Writer // Output writer (default: os.Stdout)
	Stderr         io.Writer // Error/diagnostic writer (default: os.Stderr)
}

// NewPlikCLI creates a new CLI instance from parsed arguments and config.
func NewPlikCLI(config *CliConfig, arguments map[string]any) *PlikCLI {
	return &PlikCLI{
		Config:    config,
		Arguments: arguments,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}
}

// askConfirmation prompts the user for confirmation.
// When --yes is set, it returns defaultValue immediately without prompting.
func (cli *PlikCLI) askConfirmation(defaultValue bool) (bool, error) {
	if cli.Config.Yes {
		return defaultValue, nil
	}
	return common.AskConfirmation(defaultValue)
}

// Run executes the main upload flow.
func (cli *PlikCLI) Run(client *plik.Client) error {

	if cli.Config.Debug {
		cli.errorf("Arguments : \n")
		cli.errorf("%s\n", utils.Sdump(cli.Arguments))
		cli.errorf("Configuration : \n")
		cli.errorf("%s\n", utils.Sdump(cli.Config))
	}

	upload := client.NewUpload()

	if len(cli.Config.filePaths) == 0 {
		if cli.Config.DisableStdin {
			return fmt.Errorf("stdin is disabled by default, use the --stdin flag to override")
		}
		upload.AddFileFromReader("STDIN", bufio.NewReader(os.Stdin))
	} else {
		if cli.Config.Archive {
			var err error
			cli.ArchiveBackend, err = archive.NewArchiveBackend(cli.Config.ArchiveMethod, cli.Config.ArchiveOptions)
			if err != nil {
				return fmt.Errorf("unable to initialize archive backend: %w", err)
			}

			err = cli.ArchiveBackend.Configure(cli.Arguments)
			if err != nil {
				return fmt.Errorf("unable to configure archive backend: %w", err)
			}

			reader, err := cli.ArchiveBackend.Archive(cli.Config.filePaths)
			if err != nil {
				return fmt.Errorf("unable to create archive: %w", err)
			}

			filename := cli.ArchiveBackend.GetFileName(cli.Config.filePaths)
			upload.AddFileFromReader(filename, reader)
		} else {
			for _, path := range cli.Config.filePaths {
				_, err := upload.AddFileFromPath(path)
				if err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
			}
		}
	}

	if cli.Config.filenameOverride != "" {
		if len(upload.Files()) != 1 {
			return fmt.Errorf("can't override filename if more than one file to upload")
		}
		upload.Files()[0].Name = cli.Config.filenameOverride
	}

	// Initialize crypto backend
	if cli.Config.Secure {
		var err error
		cli.CryptoBackend, err = crypto.NewCryptoBackend(cli.Config.SecureMethod, cli.Config.SecureOptions)
		if err != nil {
			return fmt.Errorf("unable to initialize crypto backend: %w", err)
		}
		cli.CryptoBackend.SetStderr(cli.Stderr)

		// Emit deprecation warnings for legacy backends
		if cli.Config.SecureMethod == "openssl" || cli.Config.SecureMethod == "pgp" {
			configHint := "~/.plikrc"
			if cli.Config.ConfigPath != "" {
				configHint = cli.Config.ConfigPath
			}
			cli.errorf("\nWARNING: The %q encryption backend is deprecated.\n", cli.Config.SecureMethod)
			cli.errorf("You can switch to \"age\" by setting SecureMethod = \"age\" in %s\n\n", configHint)
		}

		err = cli.CryptoBackend.Configure(cli.Arguments)
		if err != nil {
			return fmt.Errorf("unable to configure crypto backend: %w", err)
		}

		// Set E2EE metadata on the upload when using age backend with passphrase mode.
		// Recipient mode (-r <pubkey>) is not webapp-compatible — the webapp can't
		// ask for the private key, so we don't flag the upload as E2EE.
		if cli.Config.SecureMethod == "age" && cli.Arguments["--recipient"] == nil {
			upload.E2EE = "age"
		}
	}

	// Initialize progress bar display
	var progress *Progress
	if !cli.Config.Quiet && !cli.Config.Debug {
		progress = NewProgress(upload.Files())
	}

	// Wrap file readers with crypto and progress
	for _, file := range upload.Files() {
		if cli.Config.Secure {
			file.WrapReader(func(fileReader io.ReadCloser) io.ReadCloser {
				reader, err := cli.CryptoBackend.Encrypt(fileReader)
				if err != nil {
					// TODO: WrapReader's callback signature (func(io.ReadCloser) io.ReadCloser)
					// does not allow returning an error. Refactor the plik library's WrapReader
					// API to support error propagation and remove this os.Exit.
					cli.errorf("Unable to encrypt file: %s", err)
					os.Exit(1)
				}
				return io.NopCloser(reader)
			})
		}

		if !cli.Config.Quiet && !cli.Config.Debug {
			progress.register(file)
		}
	}

	// Create upload on server
	err := upload.Create()
	if err != nil {
		return fmt.Errorf("unable to create upload: %w", err)
	}

	// Mon, 02 Jan 2006 15:04:05 MST
	creationDate := upload.Metadata().CreatedAt.Format(time.RFC1123)

	// Display upload url
	cli.printf("Upload successfully created at %s : \n", creationDate)

	uploadURL, err := upload.GetURL()
	if err != nil {
		return fmt.Errorf("unable to get upload url: %w", err)
	}

	cli.printf("    %s\n\n", uploadURL)

	if cli.Config.Stream && !cli.Config.Debug {
		for _, file := range upload.Files() {
			cmd, err := cli.getFileCommand(file)
			if err != nil {
				cli.errorf("Unable to get download command for file %s : %s\n", file.Name, err)
			}
			cli.errorf("%s\n", cmd)
		}
		cli.printf("\n")
	}

	if !cli.Config.Quiet && !cli.Config.Debug {
		// Nothing should be printed between this and progress.Stop()
		progress.start()
	}

	// Upload files
	_ = upload.Upload()

	if !cli.Config.Quiet && !cli.Config.Debug {
		// Finalize the progress bar display
		progress.stop()
	}

	// JSON output mode
	if cli.Config.JSON {
		data, err := json.MarshalIndent(upload.WithURL(), "", "  ")
		if err != nil {
			return fmt.Errorf("unable to marshal upload metadata: %w", err)
		}
		cli.printAlways("%s\n", string(data))
		return nil
	}

	// Display download commands
	if !cli.Config.Stream {
		var commands []string
		for _, file := range upload.Files() {
			// Skip files that failed to upload
			if file.Error() != nil {
				continue
			}
			if cli.Config.Quiet {
				URL, err := file.GetURL()
				if err != nil {
					cli.errorf("Unable to get download command for file %s : %s\n", file.Name, err)
				}
				commands = append(commands, URL.String())
			} else {
				cmd, err := cli.getFileCommand(file)
				if err != nil {
					cli.errorf("Unable to get download command for file %s : %s\n", file.Name, err)
				}
				commands = append(commands, cmd)
			}
		}
		if len(commands) > 0 {
			cli.printf("\nCommands : \n")
			for _, cmd := range commands {
				cli.printAlways("%s\n", cmd)
			}
		}
	} else {
		cli.printf("\n")
	}

	return nil
}

func (cli *PlikCLI) info(client *plik.Client) error {
	cli.printAlways("Plik client version : %s\n\n", common.GetBuildInfo())

	cli.printAlways("Plik server url : %s\n", cli.Config.URL)

	serverBuildInfo, err := client.GetServerVersion()
	if err != nil {
		return fmt.Errorf("Plik server unreachable : %s", err)
	}

	cli.printAlways("Plik server version : %s\n", serverBuildInfo)

	serverConfig, err := client.GetServerConfig()
	if err != nil {
		return fmt.Errorf("Plik server unreachable : %s", err)
	}

	cli.printAlways("\nPlik server configuration :\n")
	cli.printAlways("%s", serverConfig.String())

	// Profile information (at the end for readability)
	if len(cli.Config.ActiveProfiles) > 0 || len(cli.Config.AvailableProfiles) > 0 {
		cli.printAlways("\n")
	}
	if len(cli.Config.ActiveProfiles) > 0 {
		label := "Active profile"
		if len(cli.Config.ActiveProfiles) > 1 {
			label = "Active profiles"
		}
		cli.printAlways("%s : %s\n", label, strings.Join(cli.Config.ActiveProfiles, ", "))
	}
	if len(cli.Config.AvailableProfiles) > 0 {
		cli.printAlways("Available profiles : %s\n", strings.Join(cli.Config.AvailableProfiles, ", "))
	}

	return nil
}

func (cli *PlikCLI) getFileCommand(file *plik.File) (command string, err error) {
	// Step one - Downloading file
	switch cli.Config.DownloadBinary {
	case "wget":
		command += "wget -q -O-"
	case "curl":
		command += "curl -s"
	default:
		command += cli.Config.DownloadBinary
	}

	URL, err := file.GetURL()
	if err != nil {
		return "", err
	}
	command += fmt.Sprintf(` "%s"`, URL)

	// If Ssl
	if cli.Config.Secure {
		command += fmt.Sprintf(" | %s", cli.CryptoBackend.Comments())
	}

	// If archive
	if cli.Config.Archive {
		if cli.Config.ArchiveMethod == "zip" {
			command += fmt.Sprintf(` > '%s'`, file.Name)
		} else {
			command += fmt.Sprintf(" | %s", cli.ArchiveBackend.Comments())
		}
	} else {
		command += fmt.Sprintf(` > '%s'`, file.Name)
	}

	return
}

func (cli *PlikCLI) printf(format string, args ...any) {
	if !cli.Config.Quiet {
		fmt.Fprintf(cli.Stdout, format, args...)
	}
}

func (cli *PlikCLI) errorf(format string, args ...any) {
	fmt.Fprintf(cli.Stderr, format, args...)
}

// printAlways writes to stdout regardless of quiet mode.
// Use for interactive prompts and status messages that must always be visible.
func (cli *PlikCLI) printAlways(format string, args ...any) {
	fmt.Fprintf(cli.Stdout, format, args...)
}
