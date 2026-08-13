package metadata

import (
	"fmt"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

// CreateUser create a new user in DB
func (b *Backend) CreateUser(user *common.User) (err error) {
	return b.db.Create(user).Error
}

// UpdateUser update user info in DB
func (b *Backend) UpdateUser(user *common.User) (err error) {
	result := b.db.Save(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != int64(1) {
		return fmt.Errorf("no user updated")
	}

	return nil
}

// GetUser return a user from DB ( return nil and no error if not found )
func (b *Backend) GetUser(ID string) (user *common.User, err error) {
	user = &common.User{}
	err = b.db.Where(&common.User{ID: ID}).Take(user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return user, err
}

// GetUsers return all users
// provider is an optional filter
// admin is an optional filter ( nil = no filter, true = admins only, false = non-admins only )
func (b *Backend) GetUsers(provider string, admin *bool, withTokens bool, pagingQuery *common.PagingQuery) (users []*common.User, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "ID")

	stmt := b.db.Model(&common.User{})

	if withTokens {
		stmt = stmt.Preload("Tokens")
	}

	if provider != "" {
		stmt = stmt.Where(&common.User{Provider: provider})
	}

	if admin != nil {
		// Use raw SQL instead of struct-based Where because GORM ignores zero-value
		// fields in structs, and false is the zero value for bool. Using the struct
		// pattern would silently skip the filter when querying for non-admin users.
		stmt = stmt.Where("is_admin = ?", *admin)
	}

	result, c, err := p.Paginate(stmt, &users)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return users, &c, err
}

// SearchUsers returns users matching a LIKE query on id, login, name, and email.
// Results are always sorted by login and hard-capped at limit (max 20).
// provider and admin are optional filters, same as GetUsers.
func (b *Backend) SearchUsers(query string, provider string, admin *bool, limit int) (users []*common.User, err error) {
	if query == "" {
		return nil, fmt.Errorf("missing search query")
	}
	if limit <= 0 || limit > 20 {
		limit = 20
	}

	pattern := "%" + query + "%"
	stmt := b.db.Model(&common.User{}).
		Where("id LIKE ? OR login LIKE ? OR name LIKE ? OR email LIKE ?", pattern, pattern, pattern, pattern)

	if provider != "" {
		stmt = stmt.Where(&common.User{Provider: provider})
	}

	if admin != nil {
		stmt = stmt.Where("is_admin = ?", *admin)
	}

	stmt = stmt.Order("login ASC").Limit(limit)

	err = stmt.Find(&users).Error
	if err != nil {
		return nil, err
	}

	if users == nil {
		users = []*common.User{}
	}

	return users, nil
}

// ForEachUserUploads execute f for all upload matching the user and token filters
func (b *Backend) ForEachUserUploads(userID string, tokenStr string, f func(upload *common.Upload) error) (err error) {
	stmt := b.db.Model(&common.Upload{}).Where(&common.Upload{User: userID, Token: tokenStr})

	rows, err := stmt.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		if err != nil {
			return err
		}
		err = f(upload)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveUserUploads soft-deletes all uploads matching the user and token filters
// in a single transaction: marks files for cleanup and soft-deletes the uploads.
// The cleanup job will handle actual file removal from the data backend.
func (b *Backend) RemoveUserUploads(userID string, tokenStr string) (removed int, err error) {
	err = b.db.Transaction(func(tx *gorm.DB) (err error) {
		// Subquery: SELECT id FROM uploads WHERE user = ? [AND token = ?]
		// Used by the file updates below so the DB handles filtering internally
		// without materializing IDs in Go memory.
		uploadIDsSubquery := tx.Model(&common.Upload{}).
			Select("id").
			Where(&common.Upload{User: userID, Token: tokenStr})

		// Mark files with no data on disk (missing/empty) as deleted
		err = tx.Model(&common.File{}).
			Where("upload_id IN (?)", uploadIDsSubquery).
			Where(tx.Where(&common.File{Status: common.FileMissing}).Or(&common.File{Status: ""})).
			Update("status", common.FileDeleted).Error
		if err != nil {
			return fmt.Errorf("unable to mark missing files as deleted : %s", err)
		}

		// Mark files with data on disk (uploading/uploaded) as removed
		err = tx.Model(&common.File{}).
			Where("upload_id IN (?)", uploadIDsSubquery).
			Where(tx.Where(&common.File{Status: common.FileUploading}).Or(&common.File{Status: common.FileUploaded})).
			Update("status", common.FileRemoved).Error
		if err != nil {
			return fmt.Errorf("unable to mark uploaded files as removed : %s", err)
		}

		// Soft-delete all uploads
		result := tx.Where(&common.Upload{User: userID, Token: tokenStr}).Delete(&common.Upload{})
		if result.Error != nil {
			return fmt.Errorf("unable to soft-delete uploads : %s", result.Error)
		}
		removed = int(result.RowsAffected)

		return nil
	})

	return removed, err
}

// DeleteUser delete a user from the DB
func (b *Backend) DeleteUser(userID string) (deleted bool, err error) {
	_, err = b.RemoveUserUploads(userID, "")
	if err != nil {
		return false, err
	}

	err = b.db.Transaction(func(tx *gorm.DB) (err error) {
		// Delete user tokens
		err = tx.Where(&common.Token{UserID: userID}).Delete(&common.Token{}).Error
		if err != nil {
			return fmt.Errorf("unable to delete tokens metadata : %s", err)
		}

		// Delete user
		result := tx.Where(&common.User{ID: userID}).Delete(common.User{})
		if result.Error != nil {
			return fmt.Errorf("unable to delete user metadata : %s", result.Error)
		}

		if result.RowsAffected > 0 {
			deleted = true
		}

		return nil
	})

	return deleted, err
}

// CountUsers count the number of users matching the optional filters
func (b *Backend) CountUsers(provider string, admin *bool) (count int64, err error) {
	stmt := b.db.Model(&common.User{})

	if provider != "" {
		stmt = stmt.Where(&common.User{Provider: provider})
	}

	if admin != nil {
		stmt = stmt.Where("is_admin = ?", *admin)
	}

	err = stmt.Count(&count).Error
	return count, err
}

// ForEachUsers execute f for every user in the database
func (b *Backend) ForEachUsers(f func(user *common.User) error) (err error) {
	rows, err := b.db.Model(&common.User{}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		user := &common.User{}
		err = b.db.ScanRows(rows, user)
		if err != nil {
			return err
		}
		err = f(user)
		if err != nil {
			return err
		}
	}

	return nil
}
