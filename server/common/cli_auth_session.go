package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	uuid "github.com/nu7hatch/gouuid"
)

const cliAuthSessionTTL = 5 * time.Minute

// CLIAuthSession is an ephemeral session used for the CLI device authorization flow.
// The CLI initiates a session, the user approves it in the browser, and the CLI polls for the resulting token.
type CLIAuthSession struct {
	Code      string    `json:"code" gorm:"primaryKey;size:16"`
	Secret    string    `json:"-" gorm:"size:64"`
	Status    string    `json:"status" gorm:"size:16;default:pending"` // "pending" or "approved"
	Token     string    `json:"token,omitempty" gorm:"size:64"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"-" gorm:"index"`
}

// NewCLIAuthSession creates a new CLIAuthSession with a random code and secret
func NewCLIAuthSession() *CLIAuthSession {
	return &CLIAuthSession{
		Code:      generateCode(),
		Secret:    generateSecret(),
		Status:    "pending",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(cliAuthSessionTTL),
	}
}

// IsExpired returns true if the session has passed its TTL
func (s *CLIAuthSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// generateCode creates a human-readable 8-character code in the format XXXX-XXXX
func generateCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // No 0/O/1/I to avoid confusion
	var b strings.Builder
	for i := range 8 {
		if i == 4 {
			b.WriteByte('-')
		}
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(fmt.Errorf("unable to generate random code: %s", err))
		}
		b.WriteByte(charset[idx.Int64()])
	}
	return b.String()
}

// generateSecret creates a UUID secret for CLI polling
func generateSecret() string {
	secret, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Errorf("unable to generate secret uuid: %s", err))
	}
	return secret.String()
}
