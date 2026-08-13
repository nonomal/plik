package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/docopt/docopt-go"

	"github.com/root-gg/plik/server/common"
)

// Main
func main() {

	// Usage /!\ INDENT THIS WITH SPACES NOT TABS /!\
	usage := `plik — temporary file sharing

Usage:
  plik [options] [FILE] ...

Profile Options:
  -P, --profile PROFILES    Use named profiles from ~/.plikrc (comma-separated for composition)

Upload Options:
  -o, --oneshot             Delete each file after first download
  -r, --removable           Allow anyone to delete uploaded files
  -S, --stream              Stream upload (blocks until receiver downloads)
  -t, --ttl TTL             Time before expiration (e.g. 30m, 24h, 7d)
  --extend-ttl              Extend expiration on each download
  -p                        Prompt for upload login and password
  --password PASSWD         Protect upload with login:password (default login: "plik")
  --comments COMMENT        Set upload comments (Markdown)
  -n, --name NAME           Set filename when piping from STDIN

Server Options:
  --server SERVER           Override server URL
  --token TOKEN             Set upload token (use '-' to prompt)
  --insecure                Skip TLS certificate verification

Archive Options:
  -a                        Archive files using default settings from ~/.plikrc
  --archive MODE            Archive files with specified backend (tar | zip)
  --compress MODE           [tar] Compression codec (gzip|bzip2|xz|lzip|lzma|lzop|no)
  --archive-options OPTIONS Additional command line options passed to archiver

Encryption Options:
  -s                        Encrypt files using default settings from ~/.plikrc
  --not-secure              Disable encryption even if enabled in ~/.plikrc
  --secure MODE             Encrypt files with backend (age | openssl | pgp, default: age)
  --passphrase PASSPHRASE   [age|openssl] Encryption passphrase (use '-' to prompt)
  --recipient RECIPIENT     [age] @github_user, ssh://host, URL, key, or age1...
                            [pgp] Recipient name or email
  --cipher CIPHER           [openssl] Cipher algorithm (default: aes-256-cbc)
  --secure-options OPTIONS  [openssl|pgp] Additional command line options

Output Options:
  -q, --quiet               Suppress progress and non-essential output
  -j, --json                Output upload metadata as JSON (implies --quiet)
  -d, --debug               Enable debug mode

General Options:
  --login                   Authenticate with server (opens browser)
  --update                  Update client binary from server
  --update-plikrc           Rewrite ~/.plikrc in canonical format
  --mcp                     Start MCP (Model Context Protocol) server over stdio
  --stdin                   Read from STDIN even when DisableStdin is set
  -y, --yes                 Auto-accept confirmation prompts
  -v, --version             Show client version
  -i, --info                Show client and server information
  -h, --help                Show this help

Examples:
  plik file.txt                       Upload a single file
  plik -o file1.txt file2.txt         Upload files, delete after first download
  plik -t 1h *.log                    Upload with 1 hour expiration
  plik -s secret.pdf                  Encrypt with age (passphrase auto-generated)
  plik -a src/                        Archive and upload a directory
  plik -P work report.pdf             Upload using the "work" profile
  plik -P work,zip report.pdf         Compose profiles (work server + zip archive)
  cat data.csv | plik -n data.csv     Pipe from STDIN
`
	// Parse command line arguments
	arguments, _ := docopt.ParseDoc(usage)

	if arguments["--version"].(bool) {
		fmt.Printf("Plik client %s\n", common.GetBuildInfo())
		os.Exit(0)
	}

	// Load config
	config, err := LoadConfig(arguments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load configuration : %s\n", err)
		os.Exit(1)
	}

	// Load arguments
	err = config.UnmarshalArgs(arguments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// MCP server mode
	if arguments["--mcp"].(bool) {
		err = RunMCPServer(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	cli := NewPlikCLI(config, arguments)

	client := config.NewClient("plik_cli")

	// --insecure CLI flag (additive to config.Insecure handled in NewClient)
	if arguments["--insecure"].(bool) {
		client.Insecure()
	}

	// Display info
	if arguments["--info"].(bool) {
		err = cli.info(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Update
	updateFlag := arguments["--update"].(bool)
	err = cli.update(client, updateFlag)
	if err == nil {
		if updateFlag {
			os.Exit(0)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Unable to update Plik client : \n")
		fmt.Fprintf(os.Stderr, "%s\n", err)
		if updateFlag {
			os.Exit(1)
		}
	}

	// Login
	if arguments["--login"].(bool) {
		if arguments["--server"] != nil && arguments["--server"].(string) != "" {
			fmt.Fprintf(os.Stderr, "Cannot use --login with --server: the login flow saves the token to ~/.plikrc and must use the server URL configured there.\n")
			os.Exit(1)
		}
		// --login requires a single profile: it saves a token to one profile section
		if _, err := config.SingleProfile(); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot use --login with multiple profiles: %s\n", err)
			os.Exit(1)
		}
		if len(config.ActiveProfiles) == 1 {
			fmt.Fprintf(os.Stderr, "Authenticating profile %q (%s)...\n", config.ActiveProfiles[0], config.URL)
		}
		err = login(config, client)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Login failed: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Rewrite ~/.plikrc in canonical format
	if arguments["--update-plikrc"].(bool) {
		err = updatePlikrc(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "--update-plikrc: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Detect STDIN type
	// --> If from pipe : ok, doing nothing
	// --> If not from pipe, and no files in arguments : printing help
	fi, _ := os.Stdin.Stat()

	if runtime.GOOS != "windows" {
		if (fi.Mode()&os.ModeCharDevice) != 0 && len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(1)
		}
	} else {
		if len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(1)
		}
	}

	// Run the main upload flow
	err = cli.Run(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
