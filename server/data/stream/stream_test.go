package stream

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestAddGetFile(t *testing.T) {
	backend := NewBackend(0)

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	var wg sync.WaitGroup
	wg.Go(func() {
		time.Sleep(10 * time.Millisecond)
		err := backend.AddFile(file, bytes.NewBufferString("data"))
		require.NoError(t, err, "unable to add file")
		require.NotNil(t, file.BackendDetails, "invalid nil details")
	})

	f := func() {
		for {
			reader, err := backend.GetFile(file)
			if err != nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			data, err := io.ReadAll(reader)
			require.NoError(t, err, "unable to read reader")

			err = reader.Close()
			require.NoError(t, err, "unable to close reader")

			require.Equal(t, "data", string(data), "invalid reader content")
			break
		}
		wg.Wait()
	}

	err := common.TestTimeout(f, 1*time.Second)
	require.NoError(t, err, "timeout")
}

func TestRemoveFileNotInStore(t *testing.T) {
	backend := NewBackend(0)

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	// RemoveFile on a file not in the store should be a no-op
	err := backend.RemoveFile(file)
	require.NoError(t, err)
}

func TestRemoveFileUnblocksAddFile(t *testing.T) {
	backend := NewBackend(0)

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	addFileDone := make(chan error, 1)

	// Start AddFile in a goroutine — it will block in io.Copy waiting for
	// a downloader that never comes.
	go func() {
		addFileDone <- backend.AddFile(file, bytes.NewBufferString("data"))
	}()

	// Wait for the pipe to appear in the store
	f := func() {
		for {
			backend.mu.Lock()
			storeID := file.UploadID + "/" + file.ID
			_, ok := backend.store[storeID]
			backend.mu.Unlock()
			if ok {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Now call RemoveFile — this should close the pipe and unblock AddFile
		err := backend.RemoveFile(file)
		require.NoError(t, err)

		// AddFile should return (with an error from the closed pipe)
		select {
		case err := <-addFileDone:
			// ErrClosedPipe is expected — the pipe reader was closed
			require.Error(t, err)
		case <-time.After(1 * time.Second):
			t.Fatal("AddFile did not unblock after RemoveFile")
		}
	}

	err := common.TestTimeout(f, 2*time.Second)
	require.NoError(t, err, "timeout")
}
