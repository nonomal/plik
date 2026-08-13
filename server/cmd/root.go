package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/server"
)

var configPath string
var config *common.Configuration
var port int

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "plikd",
	Short:   "Plik server",
	Long:    `Plik server`,
	Version: common.GetBuildInfo().String(),
	Run:     startPlikServer,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file (default: ./plikd.cfg or /etc/plikd.cfg, env: PLIKD_CONFIG)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().IntVar(&port, "port", 0, "Overrides plik listen port")
}

// initConfig load configuration
func initConfig() {
	var err error

	if configPath != "" {
		// Use config file from the flag.
		info, err := os.Stat(configPath)
		if err != nil {
			fmt.Printf("Unable to read config file %s : %s\n", configPath, err)
			os.Exit(1)
		}
		if info.IsDir() {
			fmt.Printf("Unable to read config file %s\n", configPath)
			os.Exit(1)
		}
	} else {
		configPath = os.Getenv("PLIKD_CONFIG")
		if configPath != "" {
			// Use config file from env.
			info, err := os.Stat(configPath)
			if err != nil {
				fmt.Printf("Unable to read config file %s : %s\n", configPath, err)
				os.Exit(1)
			}
			if info.IsDir() {
				fmt.Printf("Unable to read config file %s\n", configPath)
				os.Exit(1)
			}
		} else {
			// Use config file from default locations.
			info, err := os.Stat("plikd.cfg")
			if err == nil && !info.IsDir() {
				configPath = "plikd.cfg"
			} else {
				info, err := os.Stat("/etc/plikd.cfg")
				if err == nil && !info.IsDir() {
					configPath = "/etc/plikd.cfg"
				}
			}
		}
	}

	config, err = common.LoadConfiguration(configPath)
	if err != nil {
		fmt.Printf("Unable to load config : %s\n", err)
		os.Exit(1)
	}
}

// Initialize metadata backend
var initializeMetadataBackendOnce sync.Once
var metadataBackend *metadata.Backend
var initMetadataBackendErr error

func initializeMetadataBackend() {
	initializeMetadataBackendOnce.Do(func() {
		metadataBackend, initMetadataBackendErr = server.NewMetadataBackend(config.MetadataBackendConfig, config.NewLogger())
	})
	if initMetadataBackendErr != nil {
		fmt.Printf("unable to initialize metadata backend : %s\n", initMetadataBackendErr)
		os.Exit(1)
	}
}

// Initialize data backend
var initializeDataBackendOnce sync.Once
var dataBackend data.Backend
var initDataBackendErr error

func initializeDataBackend() {
	initializeDataBackendOnce.Do(func() {
		dataBackend, initDataBackendErr = server.NewDataBackend(config.DataBackend, config.DataBackendConfig)
	})
	if initDataBackendErr != nil {
		fmt.Printf("unable to initialize data backend : %s\n", initDataBackendErr)
		os.Exit(1)
	}
}

func startPlikServer(cmd *cobra.Command, args []string) {
	// Overrides port if provided in command line
	if port != 0 {
		config.ListenPort = port
	}

	plik := server.NewPlikServer(config)

	err := plik.Start()
	if err != nil {
		fmt.Printf("unable to start Plik server : %s\n", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	err = plik.Shutdown(time.Minute)
	if err != nil {
		fmt.Printf("error during shutdown : %s\n", err)
		os.Exit(1)
	}
}
