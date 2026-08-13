package gcs

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// gcsReadSeekCloser Implements io.ReadSeekCloser for GCS objects
// Courtesy of github.com/bobg/gcsobj for original code
type gcsReadSeekCloser struct {
	// Embedding a context in a data structure is an antipattern,
	// except when needed to satisfy interfaces (like io.ReadSeekCloser) that don't permit passing a context.
	// See https://go.dev/wiki/CodeReviewComments#contexts
	ctx context.Context

	obj       *storage.ObjectHandle
	r         *storage.Reader
	pos, size int64
}

func newReader(ctx context.Context, obj *storage.ObjectHandle) (*gcsReadSeekCloser, error) {
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return newReaderWithSize(ctx, obj, attrs.Size), nil
}

func newReaderWithSize(ctx context.Context, obj *storage.ObjectHandle, size int64) *gcsReadSeekCloser {
	return &gcsReadSeekCloser{
		ctx:  ctx,
		obj:  obj,
		size: size,
	}
}

func (r *gcsReadSeekCloser) Read(dest []byte) (int, error) {
	if r.r == nil && r.pos < r.size {
		var err error
		r.r, err = r.obj.NewRangeReader(r.ctx, r.pos, -1)
		if err != nil {
			return 0, err
		}
	}
	if r.r == nil {
		return 0, io.EOF
	}
	n, err := r.r.Read(dest)
	r.pos += int64(n)
	return n, err
}

func (r *gcsReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	var newPos int64

	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.pos + offset
	case io.SeekEnd:
		newPos = r.size + offset
	default:
		return 0, fmt.Errorf("illegal whence value %d", whence)
	}

	if r.r != nil && r.pos == newPos {
		// Optimization: don't close and reopen the reader if we're already at the desired position.
		return r.pos, nil
	}

	r.pos = newPos

	err := r.Close()
	if err != nil {
		return 0, err
	}

	return r.pos, nil
}

func (r *gcsReadSeekCloser) Close() error {
	if r.r == nil {
		return nil
	}
	err := r.r.Close()
	r.r = nil
	return err
}

// Ensure File Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for Google Cloud Storage data backend
type Config struct {
	Bucket string
	Folder string
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]any) (config *Config) {
	config = new(Config)
	utils.Assign(config, params)
	return config
}

// Backend object
type Backend struct {
	Config *Config
	client *storage.Client
}

// NewBackend instantiate a new GCS Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend, err error) {
	b = new(Backend)
	b.Config = config

	// Initialize GCS client
	b.client, err = storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to create GCS client : %s", err)
	}

	return b, nil
}

// GetFile implementation for Google Cloud Storage Data Backend
func (b *Backend) GetFile(file *common.File) (reader io.ReadSeekCloser, err error) {
	// Get object name
	objectName := b.getObjectName(file.UploadID, file.ID)

	// Get the object
	reader, err = newReader(context.Background(), b.client.Bucket(b.Config.Bucket).Object(objectName))
	if err != nil {
		return nil, fmt.Errorf("Unable to get GCS object %s : %s", objectName, err)
	}

	return reader, nil
}

// AddFile implementation for Google Cloud Storage Data Backend
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {
	// Get object name
	objectName := b.getObjectName(file.UploadID, file.ID)

	// Get a writer
	wc := b.client.Bucket(b.Config.Bucket).Object(objectName).NewWriter(context.Background())

	_, err = io.Copy(wc, fileReader)
	if err != nil {
		_ = wc.Close()
		return fmt.Errorf("Unable to write GCS object %s : %s", objectName, err)
	}

	err = wc.Close()
	if err != nil {
		return fmt.Errorf("Unable to finalize GCS object %s : %s", objectName, err)
	}

	return nil
}

// RemoveFile implementation for Google Cloud Storage Data Backend
func (b *Backend) RemoveFile(file *common.File) (err error) {
	// Get object name
	objectName := b.getObjectName(file.UploadID, file.ID)

	// Delete the object
	err = b.client.Bucket(b.Config.Bucket).Object(objectName).Delete(context.Background())
	if err != nil {
		// Ignore "file not found" errors
		if err == storage.ErrObjectNotExist {
			return nil
		}

		return fmt.Errorf("Unable to remove gcs object %s : %s", objectName, err)
	}

	return nil
}

func (b *Backend) getObjectName(uploadID string, fileID string) string {
	if b.Config.Folder != "" {
		return fmt.Sprintf("%s/%s.%s", b.Config.Folder, uploadID, fileID)
	}
	return fmt.Sprintf("%s.%s", uploadID, fileID)
}
