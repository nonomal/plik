package gcs

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

// newTestBackend creates a real GCS backend from PLIKD_CONFIG.
// Skips the test if the config is absent or not a GCS backend.
func newTestBackend(t *testing.T) *Backend {
	t.Helper()

	configPath := os.Getenv("PLIKD_CONFIG")
	if configPath == "" {
		t.Skip("PLIKD_CONFIG not set, skipping GCS integration test")
	}

	config, err := common.LoadConfiguration(configPath)
	require.NoError(t, err, "unable to load config")

	if config.DataBackend != "gcs" {
		t.Skip("data backend is not gcs, skipping GCS integration test")
	}

	backend, err := NewBackend(NewConfig(config.DataBackendConfig))
	require.NoError(t, err, "unable to create GCS backend")

	return backend
}

// objectExists checks whether an object exists in GCS.
func objectExists(t *testing.T, b *Backend, objectName string) bool {
	t.Helper()
	_, err := b.client.Bucket(b.Config.Bucket).Object(objectName).Attrs(context.Background())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false
		}
		t.Fatalf("unexpected GCS Attrs error: %s", err)
	}
	return true
}

func TestRemoveFileNewFormat(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	content := "test data for removal"
	err := b.AddFile(file, bytes.NewBufferString(content))
	require.NoError(t, err, "unable to add file")

	// Verify the object exists at the storage level
	objectName := b.getObjectName(file.UploadID, file.ID)
	require.True(t, objectExists(t, b, objectName), "object should exist after AddFile")

	// Verify we can read it back
	reader, err := b.GetFile(file)
	require.NoError(t, err, "unable to get file")
	data, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, content, string(data), "file content mismatch")
	_ = reader.Close()

	// Remove the file
	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	// Verify the object is gone at the storage level
	require.False(t, objectExists(t, b, objectName), "object should not exist after RemoveFile")
}

func TestRemoveFileTwice(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	err := b.AddFile(file, bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to add file")

	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	// Removing again should not error (interface contract)
	err = b.RemoveFile(file)
	require.NoError(t, err, "error removing already-removed file")
}

func TestRemoveFileNotFound(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	// File was never added — RemoveFile should not error
	err := b.RemoveFile(file)
	require.NoError(t, err, "error removing non-existent file")
}
