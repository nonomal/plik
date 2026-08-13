package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/metadata"
)

type importFlagParams struct {
	ignoreErrors bool
}

var importParams = importFlagParams{}

// importCmd to import metadata
var importCmd = &cobra.Command{
	Use:   "import [input-file]",
	Short: "Import metadata",
	Run:   importMetadata,
}

func init() {
	importCmd.Flags().BoolVar(&importParams.ignoreErrors, "ignore-errors", false, "ignore and logs errors")
	rootCmd.AddCommand(importCmd)
}

func importMetadata(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Println("Missing metadata import file")
		os.Exit(1)
	}

	initializeMetadataBackend()

	fmt.Printf("Importing metadata from %s to %s\n", args[0], metadataBackend.Config.Driver)

	importOptions := &metadata.ImportOptions{
		IgnoreErrors: importParams.ignoreErrors,
	}

	err := metadataBackend.Import(args[0], importOptions)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
