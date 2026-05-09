package models

import (
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
)

// Pagination holds pagination parameters.
type Pagination struct {
	Page    uint64
	PerPage uint64
	Count   uint64
}

// Offset returns the record offset for the current page.
func (p *Pagination) Offset() uint64 {
	return (p.Page - 1) * p.PerPage
}

// SortDirection is the sort direction.
type SortDirection string

const Ascending SortDirection = "ASC"
const Descending SortDirection = "DESC"
const CreatedAt = "created_at"

// SortParams holds sorting parameters.
type SortParams struct {
	Fields []SortField
}

// SortField is a single sort field with direction.
type SortField struct {
	Name string
	Dir  SortDirection
}

// TruncateAll truncates all tables within a transaction.
func TruncateAll(conn *storage.Connection) error {
	return conn.Transaction(func(tx *storage.Connection) error {
		if err := tx.Exec("TRUNCATE users"); err != nil {
			return errors.Wrap(err, "error truncating users")
		}
		if err := tx.Exec("TRUNCATE refresh_tokens"); err != nil {
			return errors.Wrap(err, "error truncating refresh_tokens")
		}
		if err := tx.Exec("TRUNCATE audit_log_entries"); err != nil {
			return errors.Wrap(err, "error truncating audit_log_entries")
		}
		return errors.Wrap(tx.Exec("TRUNCATE instances"), "error truncating instances")
	})
}
