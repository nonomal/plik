package metadata

import (
	"time"

	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

// CreateCLIAuthSession creates a new CLI auth session in the database
func (b *Backend) CreateCLIAuthSession(session *common.CLIAuthSession) error {
	return b.db.Create(session).Error
}

// GetCLIAuthSession returns a CLI auth session by its code (returns nil if not found)
func (b *Backend) GetCLIAuthSession(code string) (*common.CLIAuthSession, error) {
	session := &common.CLIAuthSession{}
	err := b.db.Where(&common.CLIAuthSession{Code: code}).Where("expires_at > ?", time.Now()).Take(session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return session, nil
}

// UpdateCLIAuthSession updates a CLI auth session in the database
func (b *Backend) UpdateCLIAuthSession(session *common.CLIAuthSession) error {
	return b.db.Save(session).Error
}

// DeleteCLIAuthSession removes a CLI auth session from the database
func (b *Backend) DeleteCLIAuthSession(code string) error {
	return b.db.Delete(&common.CLIAuthSession{Code: code}).Error
}

// DeleteExpiredCLIAuthSessions removes all expired CLI auth sessions from the database
func (b *Backend) DeleteExpiredCLIAuthSessions() (int, error) {
	result := b.db.Where("expires_at < ?", time.Now()).Delete(&common.CLIAuthSession{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}
