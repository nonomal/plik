package swift

import (
	"bytes"
	"io"
	"os"
	"testing"

	goswift "github.com/ncw/swift"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

// newTestBackend creates a real Swift backend from PLIKD_CONFIG.
// Skips the test if the config is absent or not a Swift backend.
func newTestBackend(t *testing.T) *Backend {
	t.Helper()

	configPath := os.Getenv("PLIKD_CONFIG")
	if configPath == "" {
		t.Skip("PLIKD_CONFIG not set, skipping Swift integration test")
	}

	config, err := common.LoadConfiguration(configPath)
	require.NoError(t, err, "unable to load config")

	if config.DataBackend != "swift" {
		t.Skip("data backend is not swift, skipping Swift integration test")
	}

	backend := NewBackend(NewConfig(config.DataBackendConfig))

	// Authenticate and ensure the container exists
	err = backend.auth()
	require.NoError(t, err, "unable to authenticate to Swift")

	return backend
}

// objectExistsInSwift checks whether an object exists in the Swift container.
func objectExistsInSwift(t *testing.T, b *Backend, objectName string) bool {
	t.Helper()
	_, _, err := b.connection.Object(b.config.Container, objectName)
	if err != nil {
		if err == goswift.ObjectNotFound {
			return false
		}
		t.Fatalf("unexpected Swift Object error: %s", err)
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
	objName := objectID(file)
	require.True(t, objectExistsInSwift(t, b, objName), "object should exist after AddFile")

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
	require.False(t, objectExistsInSwift(t, b, objName), "object should not exist after RemoveFile")
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
