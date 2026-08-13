package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
)

func TestUpdate_AutoUpdateDisabled(t *testing.T) {
	config := NewUploadConfig()
	config.AutoUpdate = false

	cli := NewPlikCLI(config, nil)
	client := plik.NewClient(config.URL)

	// Should return nil immediately when AutoUpdate is false and updateFlag is false
	err := cli.update(client, false)
	require.NoError(t, err)
}

func TestUpdate_QuietMode(t *testing.T) {
	config := NewUploadConfig()
	config.AutoUpdate = true
	config.Quiet = true

	cli := NewPlikCLI(config, nil)
	client := plik.NewClient(config.URL)

	// Should return nil immediately when quiet mode and updateFlag is false
	err := cli.update(client, false)
	require.NoError(t, err)
}

func TestUpdate_ServerUnreachable(t *testing.T) {
	config := NewUploadConfig()
	config.URL = "http://localhost:1" // Unreachable port

	cli := NewPlikCLI(config, nil)
	client := plik.NewClient(config.URL)

	err := cli.update(client, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unable to get server version")
}

func TestUpdate_NoPlatformClient(t *testing.T) {
	// Mock server that returns a version with no matching client binary
	buildInfo := &common.BuildInfo{
		Version: "1.4.0",
		Clients: []*common.Client{
			{OS: "unsupported_os", ARCH: "unsupported_arch", Md5: "abc123", Path: "clients/fake"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildInfo)
	}))
	defer server.Close()

	config := NewUploadConfig()
	config.URL = server.URL

	cli := NewPlikCLI(config, nil)
	client := plik.NewClient(server.URL)

	err := cli.update(client, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Server does not offer a")
}
