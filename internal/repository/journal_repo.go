package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
)

type JournalRepo struct {
	db *sql.DB
}

func NewJournalRepo(db *sql.DB) *JournalRepo {
	return &JournalRepo{db: db}
}

// ListByLedgerEntry returns all journal entries for a ledger entry, ordered by created_at ASC.
func (r *JournalRepo) ListByLedgerEntry(ctx context.Context, ledgerEntryID string) ([]*domain.JournalEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, ledger_entry_id, field_changed, old_value, new_value, reason, created_at
		 FROM journal_entries WHERE ledger_entry_id=? ORDER BY created_at ASC`,
		ledgerEntryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*domain.JournalEntry
	for rows.Next() {
		var e domain.JournalEntry
		var createdUnix int64
		if err := rows.Scan(&e.ID, &e.LedgerEntryID, &e.FieldChanged, &e.OldValue, &e.NewValue, &e.Reason, &createdUnix); err != nil {
			return nil, err
		}
		e.CreatedAt = time.Unix(createdUnix, 0).UTC()
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}
