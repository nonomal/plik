package common

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCLIAuthSession(t *testing.T) {
	session := NewCLIAuthSession()
	require.NotNil(t, session, "nil session")
	require.NotEmpty(t, session.Code, "missing code")
	require.NotEmpty(t, session.Secret, "missing secret")
	require.Equal(t, "pending", session.Status, "invalid default status")
	require.Empty(t, session.Token, "token should be empty")
	require.False(t, session.CreatedAt.IsZero(), "missing created at")
	require.False(t, session.ExpiresAt.IsZero(), "missing expires at")
	require.True(t, session.ExpiresAt.After(session.CreatedAt), "expires at should be after created at")
}

func TestNewCLIAuthSession_UniqueCodesAndSecrets(t *testing.T) {
	s1 := NewCLIAuthSession()
	s2 := NewCLIAuthSession()
	require.NotEqual(t, s1.Code, s2.Code, "codes should be unique")
	require.NotEqual(t, s1.Secret, s2.Secret, "secrets should be unique")
}

func TestGenerateCode_Format(t *testing.T) {
	code := generateCode()
	// Should be XXXX-XXXX format (9 chars total)
	require.Len(t, code, 9, "invalid code length")
	require.Equal(t, byte('-'), code[4], "missing dash separator")

	// Should only contain allowed characters
	matched, err := regexp.MatchString(`^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{4}-[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{4}$`, code)
	require.NoError(t, err)
	require.True(t, matched, "code contains invalid characters: %s", code)
}

func TestGenerateCode_NoConfusingCharacters(t *testing.T) {
	// Generate many codes and make sure none contain 0, O, 1, or I
	for range 100 {
		code := generateCode()
		require.NotContains(t, code, "0", "code should not contain 0")
		require.NotContains(t, code, "O", "code should not contain O")
		require.NotContains(t, code, "1", "code should not contain 1")
		require.NotContains(t, code, "I", "code should not contain I")
	}
}

func TestGenerateSecret(t *testing.T) {
	secret := generateSecret()
	require.NotEmpty(t, secret, "missing secret")

	// Should be a UUID format
	matched, err := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, secret)
	require.NoError(t, err)
	require.True(t, matched, "secret is not a valid UUID: %s", secret)
}

func TestCLIAuthSession_IsExpired(t *testing.T) {
	session := NewCLIAuthSession()
	require.False(t, session.IsExpired(), "new session should not be expired")

	session.ExpiresAt = time.Now().Add(-1 * time.Second)
	require.True(t, session.IsExpired(), "session with past expiry should be expired")
}
