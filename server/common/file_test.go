package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFile(t *testing.T) {
	file := NewFile()
	require.NotNil(t, file, "invalid file")
	require.NotZero(t, file.ID, "invalid file id")
}

func TestFileSanitize(t *testing.T) {
	file := &File{}
	file.BackendDetails = "value"
	file.Sanitize()
	require.Zero(t, file.BackendDetails, "invalid backend details")
}

func TestDetectMIME_PlainText(t *testing.T) {
	mimeType, isText := DetectMIME([]byte("Hello, world!\n"))
	require.Equal(t, "text/plain; charset=utf-8", mimeType)
	require.True(t, isText)
}

func TestDetectMIME_JSON(t *testing.T) {
	// JSON inherits from text/plain in the MIME hierarchy
	mimeType, isText := DetectMIME([]byte(`{"key": "value"}`))
	require.Equal(t, "application/json", mimeType)
	require.True(t, isText, "JSON should be detected as text via MIME hierarchy")
}

func TestDetectMIME_Binary(t *testing.T) {
	// PNG magic bytes
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	mimeType, isText := DetectMIME(data)
	require.Equal(t, "image/png", mimeType)
	require.False(t, isText)
}

func TestDetectMIME_EmptyData(t *testing.T) {
	mimeType, isText := DetectMIME([]byte{})
	// mimetype library returns text/plain for empty input
	require.Equal(t, "text/plain", mimeType)
	require.True(t, isText)
}
