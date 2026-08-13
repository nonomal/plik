package stream

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure Stream Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Backend object
type Backend struct {
	store   map[string]io.ReadSeekCloser
	mu      sync.Mutex
	timeout time.Duration
}

// PipeReadSeeker fills the gap so that stream looks like a regular backend
// and to not implement branched code pathes. Main code in get_file will ensure
// that seek is never called
type PipeReadSeeker struct {
	pipe *io.PipeReader
}

func (r *PipeReadSeeker) Read(data []byte) (n int, err error) {
	return r.pipe.Read(data)
}

func (r *PipeReadSeeker) Close() error {
	return r.pipe.Close()
}

func (r *PipeReadSeeker) Seek(int64, int) (int64, error) {
	return 0, fmt.Errorf("seek not supported on stream backend")
}

// NewBackend instantiate a new Stream Data Backend.
// timeout is the maximum time to wait for a download to start (0 = no timeout).
func NewBackend(timeout time.Duration) (b *Backend) {
	b = new(Backend)
	b.store = make(map[string]io.ReadSeekCloser)
	b.timeout = timeout
	return
}

// GetFile implementation for steam data backend will search
// on filesystem the requested steam and return its reading filehandle
func (b *Backend) GetFile(file *common.File) (stream io.ReadSeekCloser, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	storeID := file.UploadID + "/" + file.ID
	stream, ok := b.store[storeID]
	if !ok {
		return nil, fmt.Errorf("missing reader")
	}

	delete(b.store, storeID)

	return stream, err
}

// AddFile implementation for stream data backend will create a pipe and block until download begins.
// The store entry is consumed (deleted) by GetFile on first retrieval.
// The deferred delete here is a safety net for the case where GetFile is
// never called — it cleans up the entry after io.Copy completes.
func (b *Backend) AddFile(file *common.File, stream io.Reader) (err error) {
	storeID := file.UploadID + "/" + file.ID

	pipeReader, pipeWriter := io.Pipe()
	pipeReaderSeeker := &PipeReadSeeker{pipe: pipeReader}

	b.mu.Lock()

	b.store[storeID] = pipeReaderSeeker
	// Safety-net cleanup: if neither GetFile nor RemoveFile consumed the entry,
	// delete it after io.Copy returns. Both GetFile and RemoveFile also delete
	// under the mutex, so this may be a no-op — delete on a missing key is safe.
	defer func() {
		b.mu.Lock()
		delete(b.store, storeID)
		b.mu.Unlock()
	}()

	b.mu.Unlock()

	// Timeout: if no download starts within the configured duration, close the pipe
	// to unblock io.Copy and release the handler goroutine. Disabled when timeout is 0.
	if b.timeout > 0 {
		timer := time.AfterFunc(b.timeout, func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if _, ok := b.store[storeID]; ok {
				_ = pipeReaderSeeker.Close()
				delete(b.store, storeID)
			}
		})
		defer timer.Stop()
	}

	// This will block until download begins
	_, err = io.Copy(pipeWriter, stream)
	_ = pipeWriter.CloseWithError(err)

	return err
}

// RemoveFile closes the pipe reader (if still in the store) so that AddFile's
// blocked io.Copy returns ErrClosedPipe and the handler goroutine can exit.
func (b *Backend) RemoveFile(file *common.File) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	storeID := file.UploadID + "/" + file.ID
	if reader, ok := b.store[storeID]; ok {
		_ = reader.Close()
		delete(b.store, storeID)
	}
	return nil
}
