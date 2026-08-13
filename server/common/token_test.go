package common

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewToken(t *testing.T) {
	token := NewToken()
	require.NotNil(t, token, "invalid token")
	require.NotZero(t, token.Token, "missing token")
	require.True(t, len(token.Token) == TokenTotalLength, "token should be %d chars, got %d", TokenTotalLength, len(token.Token))
	require.Equal(t, TokenPrefix, token.Token[:len(TokenPrefix)], "token should start with prefix %q", TokenPrefix)
}

func TestGeneratePrefixedToken(t *testing.T) {
	token := GeneratePrefixedToken()

	// Check prefix
	require.Equal(t, TokenPrefix, token[:len(TokenPrefix)], "missing prefix")

	// Check total length
	require.Equal(t, TokenTotalLength, len(token), "unexpected token length")

	// Check charset (after prefix, should be all Base62)
	body := token[len(TokenPrefix):]
	matched, err := regexp.MatchString(`^[a-zA-Z0-9]+$`, body)
	require.NoError(t, err)
	require.True(t, matched, "token body should only contain Base62 chars, got %q", body)

	// Check checksum validity
	require.True(t, ValidateTokenChecksum(token), "checksum validation should pass for a fresh token")
}

func TestValidateTokenChecksum(t *testing.T) {
	// Valid token
	token := GeneratePrefixedToken()
	require.True(t, ValidateTokenChecksum(token), "valid token should pass checksum")

	// Corrupted token (flip last char)
	corrupted := token[:len(token)-1] + "!"
	require.False(t, ValidateTokenChecksum(corrupted), "corrupted token should fail checksum")

	// Truncated token
	truncated := token[:20]
	require.False(t, ValidateTokenChecksum(truncated), "truncated token should fail checksum")

	// Legacy UUIDv4 token (no prefix)
	require.False(t, ValidateTokenChecksum("550e8400-e29b-41d4-a716-446655440000"), "legacy UUID should return false")

	// Empty string
	require.False(t, ValidateTokenChecksum(""), "empty string should return false")

	// Just the prefix
	require.False(t, ValidateTokenChecksum(TokenPrefix), "prefix-only should return false")

	// Wrong prefix, right length
	wrongPrefix := "xlik_" + token[len(TokenPrefix):]
	require.False(t, ValidateTokenChecksum(wrongPrefix), "wrong prefix should fail")
}

func TestTokenUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		token := GeneratePrefixedToken()
		_, exists := seen[token]
		require.False(t, exists, "duplicate token generated: %s", token)
		seen[token] = struct{}{}
	}
}

func TestEncodeBase62(t *testing.T) {
	// Zero value
	result := encodeBase62(0, 6)
	require.Equal(t, 6, len(result), "should be padded to 6 chars")
	require.Equal(t, "aaaaaa", result, "zero should encode to all first chars")

	// Max uint32
	result = encodeBase62(0xFFFFFFFF, 6)
	require.Equal(t, 6, len(result), "should be 6 chars for max uint32")

	// Deterministic
	result1 := encodeBase62(42, 6)
	result2 := encodeBase62(42, 6)
	require.Equal(t, result1, result2, "same input should produce same output")

	// Different inputs produce different outputs
	result3 := encodeBase62(43, 6)
	require.NotEqual(t, result1, result3, "different inputs should produce different outputs")
}
