package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	_, file, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestJournalRepo_ListByLedgerEntry(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Insert a wallet + transaction + ledger entry directly.
	walletID := uuid.New().String()
	txID := uuid.New().String()
	ledgerID := uuid.New().String()

	now := time.Now().UTC().Unix()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO wallets (id, name, descriptor, fingerprint, type, network, source, created_at, updated_at)
		 VALUES (?, 'test', '', '', 'singlesig', 'mainnet', 'manual', ?, ?)`,
		walletID, now, now)
	if err != nil {
		t.Fatalf("insert wallet: %v", err)
	}
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO transactions (id, wallet_id, txid, block_height, block_time, fee_sats, confirmed, created_at)
		 VALUES (?, ?, 'abc123', 0, 0, 0, 0, ?)`,
		txID, walletID, now)
	if err != nil {
		t.Fatalf("insert tx: %v", err)
	}
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO ledger_entries (id, wallet_id, transaction_id, type, sats, fiat_amount, fiat_currency, category, note, created_at)
		 VALUES (?, ?, ?, 'credit', 1000, 0, 'EUR', 'unknown', '', ?)`,
		ledgerID, walletID, txID, now)
	if err != nil {
		t.Fatalf("insert ledger entry: %v", err)
	}

	// Insert two journal entries.
	j1 := uuid.New().String()
	j2 := uuid.New().String()
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO journal_entries (id, ledger_entry_id, field_changed, old_value, new_value, reason, created_at)
		 VALUES (?, ?, 'note', '', 'first note', 'user edit', ?)`,
		j1, ledgerID, now)
	if err != nil {
		t.Fatalf("insert journal 1: %v", err)
	}
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO journal_entries (id, ledger_entry_id, field_changed, old_value, new_value, reason, created_at)
		 VALUES (?, ?, 'category', 'unknown', 'income', 'correction', ?)`,
		j2, ledgerID, now+1)
	if err != nil {
		t.Fatalf("insert journal 2: %v", err)
	}

	repo := NewJournalRepo(db)
	entries, err := repo.ListByLedgerEntry(context.Background(), ledgerID)
	if err != nil {
		t.Fatalf("ListByLedgerEntry: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 journal entries, got %d", len(entries))
	}
	if entries[0].FieldChanged != "note" {
		t.Errorf("expected first entry field_changed=note, got %q", entries[0].FieldChanged)
	}
	if entries[1].FieldChanged != "category" {
		t.Errorf("expected second entry field_changed=category, got %q", entries[1].FieldChanged)
	}
}

func TestJournalRepo_ListByLedgerEntry_Empty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := NewJournalRepo(db)
	entries, err := repo.ListByLedgerEntry(context.Background(), "nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil slice for empty result, got %v", entries)
	}
}
