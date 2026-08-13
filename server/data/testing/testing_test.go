package file

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure Testing Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

func TestGetFiles(t *testing.T) {
	backend := NewBackend()
	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	files := backend.GetFiles()
	require.NotNil(t, files, "missing file map")
	require.Lenf(t, files, 1, "empty file map")
	require.NotNil(t, files[file.ID], "missing file")
}

func TestAddFileError(t *testing.T) {
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestAddFileReaderError(t *testing.T) {
	backend := NewBackend()

	upload := &common.Upload{}
	file := upload.NewFile()
	reader := common.NewErrorReader(errors.New("io error"))

	err := backend.AddFile(file, reader)
	require.Error(t, err, "missing error")
	require.Equal(t, "io error", err.Error(), "invalid error message")
}

func TestAddFile(t *testing.T) {
	backend := NewBackend()
	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")
}

func TestGetFileError(t *testing.T) {
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := &common.Upload{}
	file := upload.NewFile()

	_, err := backend.GetFile(file)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestGetFile(t *testing.T) {
	backend := NewBackend()
	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.GetFile(file)
	require.NoError(t, err, "unable to get file")
}

func TestRemoveFileError(t *testing.T) {
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.RemoveFile(file)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestRemoveFile(t *testing.T) {
	backend := NewBackend()

	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.GetFile(file)
	require.NoError(t, err, "unable to get file")

	err = backend.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	_, err = backend.GetFile(file)
	require.Error(t, err, "unable to get file")
	require.Equal(t, "file not found", err.Error(), "invalid error message")
}

func TestBufferRead(t *testing.T) {
	buffer := &Buffer{buf: []byte("data data data"), off: 0}
	dst := make([]byte, 2)

	n, err := buffer.Read(dst)
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, 2, n, "unexpected read size")
	require.Equal(t, []byte("da"), dst, "unexpected read data")

	n, err = buffer.Read(dst)
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, 2, n, "unexpected read size")
	require.Equal(t, []byte("ta"), dst, "unexpected read data")

	n, err = buffer.Read(dst[:1])
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, 1, n, "unexpected read size")
	require.Equal(t, []byte(" "), dst[:1], "unexpected read data")

	// Drain the remaining data to reach EOF.
	remaining, err := io.ReadAll(buffer)
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, []byte("data data"), remaining, "unexpected remaining data")

	n, err = buffer.Read(dst)
	require.ErrorIs(t, err, io.EOF, "expected EOF")
	require.Equal(t, 0, n, "unexpected read size")

	// After EOF, the buffer resets and empty reads are allowed.
	n, err = buffer.Read(dst[:0])
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, 0, n, "unexpected read size")
}

func TestBufferSeek(t *testing.T) {
	buffer := &Buffer{buf: []byte("data data data"), off: 0}
	dst := make([]byte, 2)

	offset, err := buffer.Seek(2, io.SeekStart)
	require.NoError(t, err, "unexpected seek error")
	require.Equal(t, int64(2), offset, "unexpected offset")

	n, err := buffer.Read(dst)
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, 2, n, "unexpected read size")
	require.Equal(t, []byte("ta"), dst, "unexpected read data")

	offset, err = buffer.Seek(-1, io.SeekCurrent)
	require.NoError(t, err, "unexpected seek error")
	require.Equal(t, int64(3), offset, "unexpected offset")

	n, err = buffer.Read(dst[:1])
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, 1, n, "unexpected read size")
	require.Equal(t, []byte("a"), dst[:1], "unexpected read data")

	offset, err = buffer.Seek(-2, io.SeekEnd)
	require.NoError(t, err, "unexpected seek error")
	require.Equal(t, int64(12), offset, "unexpected offset")

	n, err = buffer.Read(dst)
	require.NoError(t, err, "unexpected read error")
	require.Equal(t, 2, n, "unexpected read size")
	require.Equal(t, []byte("ta"), dst, "unexpected read data")
}

func TestBufferClose(t *testing.T) {
	buffer := &Buffer{}
	require.NoError(t, buffer.Close(), "unexpected close error")
}
