package middleware

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestLimitBody(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	body := []byte(`{"key":"value"}`)
	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(body))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LimitBody(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
}

// NeverEndingReader is an io.Reader that never returns io.EOF.
type NeverEndingReader struct{}

func (r *NeverEndingReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte('x')
	}
	return len(p), nil
}

func TestLimitBodyTooBig(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/upload", &NeverEndingReader{})
	require.NoError(t, err, "unable to create new request")

	// The handler reads the body after the middleware has wrapped it
	handler := LimitBody(ctx, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		_, err := io.ReadAll(req.Body)
		require.Error(t, err, "expected error reading oversized body")
		require.Contains(t, err.Error(), "http: request body too large")
	}))

	rr := ctx.NewRecorder(req)
	handler.ServeHTTP(rr, req)
}

func TestLimitBodySkipGet(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/upload", &NeverEndingReader{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LimitBody(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
}

func TestLimitBodySkipDelete(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/upload/abc123", &NeverEndingReader{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LimitBody(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
}

func TestLimitBodySkipFileUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/file/abc123", &NeverEndingReader{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LimitBody(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
}

func TestLimitBodySkipStreamUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/stream/abc123/file1/test.txt", &NeverEndingReader{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LimitBody(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
}

func TestLimitBodySkipQuickUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/", &NeverEndingReader{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LimitBody(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
}
