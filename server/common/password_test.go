package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashUploadPassword(t *testing.T) {
	hash, err := HashUploadPassword("login", "password")
	require.NoError(t, err)
	require.True(t, len(hash) > 0)
	require.Equal(t, "$2", hash[:2], "hash should be bcrypt format")
}

func TestCheckUploadPassword(t *testing.T) {
	hash, err := HashUploadPassword("login", "password")
	require.NoError(t, err)

	b64 := EncodeAuthBasicHeader("login", "password")

	// Correct credentials
	require.True(t, CheckUploadPassword(b64, hash))

	// Wrong credentials
	wrongB64 := EncodeAuthBasicHeader("login", "wrong")
	require.False(t, CheckUploadPassword(wrongB64, hash))
}

func TestHashUploadPasswordLongCredentials(t *testing.T) {
	// 128-char login and password should work (bcrypt's 72-byte limit is removed by SHA-256 pre-hash)
	longLogin := strings.Repeat("a", 128)
	longPass := strings.Repeat("b", 128)

	hash, err := HashUploadPassword(longLogin, longPass)
	require.NoError(t, err)

	b64 := EncodeAuthBasicHeader(longLogin, longPass)
	require.True(t, CheckUploadPassword(b64, hash))
}
