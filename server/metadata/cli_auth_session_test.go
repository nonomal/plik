package metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestBackend_CreateCLIAuthSession(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	session := common.NewCLIAuthSession()
	err := b.CreateCLIAuthSession(session)
	require.NoError(t, err, "create session error")

	// Duplicate should fail
	err = b.CreateCLIAuthSession(session)
	require.Error(t, err, "create duplicate session error expected")
}

func TestBackend_GetCLIAuthSession(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// Not found
	session, err := b.GetCLIAuthSession("XXXX-YYYY")
	require.NoError(t, err, "get session error")
	require.Nil(t, session, "non nil session")

	// Create and retrieve
	created := common.NewCLIAuthSession()
	err = b.CreateCLIAuthSession(created)
	require.NoError(t, err, "create session error")

	session, err = b.GetCLIAuthSession(created.Code)
	require.NoError(t, err, "get session error")
	require.NotNil(t, session, "nil session")
	require.Equal(t, created.Code, session.Code, "invalid code")
	require.Equal(t, created.Secret, session.Secret, "invalid secret")
	require.Equal(t, "pending", session.Status, "invalid status")
}

func TestBackend_GetCLIAuthSession_Expired(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	session := common.NewCLIAuthSession()
	session.ExpiresAt = time.Now().Add(-1 * time.Second)
	err := b.CreateCLIAuthSession(session)
	require.NoError(t, err, "create session error")

	// Should not return expired session
	result, err := b.GetCLIAuthSession(session.Code)
	require.NoError(t, err, "get session error")
	require.Nil(t, result, "expired session should not be returned")
}

func TestBackend_UpdateCLIAuthSession(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	session := common.NewCLIAuthSession()
	err := b.CreateCLIAuthSession(session)
	require.NoError(t, err, "create session error")

	session.Status = "approved"
	session.Token = "test-token-value"
	err = b.UpdateCLIAuthSession(session)
	require.NoError(t, err, "update session error")

	updated, err := b.GetCLIAuthSession(session.Code)
	require.NoError(t, err, "get session error")
	require.NotNil(t, updated, "nil session")
	require.Equal(t, "approved", updated.Status, "invalid status")
	require.Equal(t, "test-token-value", updated.Token, "invalid token")
}

func TestBackend_DeleteCLIAuthSession(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	session := common.NewCLIAuthSession()
	err := b.CreateCLIAuthSession(session)
	require.NoError(t, err, "create session error")

	err = b.DeleteCLIAuthSession(session.Code)
	require.NoError(t, err, "delete session error")

	result, err := b.GetCLIAuthSession(session.Code)
	require.NoError(t, err, "get session error")
	require.Nil(t, result, "session should be deleted")
}

func TestBackend_DeleteExpiredCLIAuthSessions(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// Create an expired session
	expired := common.NewCLIAuthSession()
	expired.ExpiresAt = time.Now().Add(-1 * time.Minute)
	err := b.CreateCLIAuthSession(expired)
	require.NoError(t, err, "create expired session error")

	// Create a valid session
	valid := common.NewCLIAuthSession()
	err = b.CreateCLIAuthSession(valid)
	require.NoError(t, err, "create valid session error")

	// Clean up expired
	count, err := b.DeleteExpiredCLIAuthSessions()
	require.NoError(t, err, "delete expired sessions error")
	require.Equal(t, 1, count, "should have deleted 1 expired session")

	// Valid session should still exist
	result, err := b.GetCLIAuthSession(valid.Code)
	require.NoError(t, err, "get session error")
	require.NotNil(t, result, "valid session should still exist")
}
