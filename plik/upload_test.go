package plik

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/root-gg/plik/server/common"

	"github.com/stretchr/testify/require"
)

func TestGetUploadURL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()

	_, err = upload.GetURL()
	common.RequireError(t, err, "upload has not been created yet")

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")

	uploadURL, err := upload.GetURL()
	require.NoError(t, err, "unable to get upload URL")
	require.Equal(t, pc.URL+"/#/?id="+upload.ID(), uploadURL.String(), "invalid upload URL")
}

func TestGetUploadAdminURL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()

	_, err = upload.GetAdminURL()
	common.RequireError(t, err, "upload has not been created yet")

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")

	uploadURL, err := upload.GetAdminURL()
	require.NoError(t, err, "unable to get upload URL")
	require.Equal(t, fmt.Sprintf("%s/#/?id=%s&uploadToken=%s", pc.URL, upload.ID(), upload.Metadata().UploadToken), uploadURL.String(), "invalid upload URL")
}

func TestApplyOverridesNonZeroFields(t *testing.T) {
	pc := NewClient("http://127.0.0.1:8080")
	pc.Token = "existing-token"
	pc.Comments = "existing-comments"

	upload := pc.NewUpload()

	// Verify defaults were copied from client
	require.Equal(t, "existing-token", upload.Token, "token should be copied from client")
	require.Equal(t, "existing-comments", upload.Comments, "comments should be copied from client")

	// Apply params that override some fields
	params := UploadParams{
		OneShot:  true,
		TTL:      3600,
		Comments: "new-comments",
	}
	params.Apply(upload)

	// Overridden fields
	require.True(t, upload.OneShot, "one_shot should be set")
	require.Equal(t, 3600, upload.TTL, "TTL should be overridden")
	require.Equal(t, "new-comments", upload.Comments, "comments should be overridden")

	// Preserved fields (zero values in params should NOT clear existing values)
	require.Equal(t, "existing-token", upload.Token, "token should be preserved")
	require.False(t, upload.Stream, "stream should remain false")
	require.False(t, upload.Removable, "removable should remain false")
}

func TestApplyPreservesTokenWhenEmpty(t *testing.T) {
	pc := NewClient("http://127.0.0.1:8080")
	pc.Token = "user-auth-token"

	upload := pc.NewUpload()

	// Apply with empty token — should NOT clear the existing token
	params := UploadParams{OneShot: true}
	params.Apply(upload)

	require.Equal(t, "user-auth-token", upload.Token, "token should be preserved when Apply token is empty")
	require.True(t, upload.OneShot, "one_shot should be set")
}

func TestApplyOverridesToken(t *testing.T) {
	pc := NewClient("http://127.0.0.1:8080")
	pc.Token = "original-token"

	upload := pc.NewUpload()

	params := UploadParams{Token: "override-token"}
	params.Apply(upload)

	require.Equal(t, "override-token", upload.Token, "token should be overridden when explicitly set")
}

func TestUploadWithURL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	data := "test data"
	upload, file, err := pc.UploadReader("testfile.txt", bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")

	result := upload.WithURL()
	require.NotNil(t, result, "WithURL should not return nil")
	require.NotEmpty(t, result.URL, "upload URL should not be empty")
	require.Contains(t, result.URL, upload.ID(), "upload URL should contain upload ID")
	require.Len(t, result.Files, 1, "should have one file")
	require.Equal(t, file.Name, result.Files[0].Name, "file name should match")
	require.NotEmpty(t, result.Files[0].URL, "file URL should not be empty")
}

func TestUploadWithURLBeforeCreate(t *testing.T) {
	pc := NewClient("http://127.0.0.1:8080")
	upload := pc.NewUpload()
	upload.AddFileFromReader("test.txt", bytes.NewBufferString("data"))

	result := upload.WithURL()
	require.NotNil(t, result, "WithURL should not return nil even before create")
	require.Empty(t, result.URL, "URL should be empty before create")
}
