package common

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashUploadPassword hashes upload credentials using bcrypt(sha256(base64(login:password))).
// The SHA-256 pre-hash produces a fixed 32-byte (64-hex) digest, removing bcrypt's
// 72-byte input limit and allowing arbitrarily long credentials.
func HashUploadPassword(login string, password string) (string, error) {
	b64 := EncodeAuthBasicHeader(login, password)
	digest := sha256.Sum256([]byte(b64))
	hash, err := bcrypt.GenerateFromPassword(fmt.Appendf(nil, "%x", digest), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckUploadPassword verifies upload credentials against a stored bcrypt(sha256) hash.
func CheckUploadPassword(b64Creds string, storedHash string) bool {
	digest := sha256.Sum256([]byte(b64Creds))
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), fmt.Appendf(nil, "%x", digest))
	return err == nil
}
