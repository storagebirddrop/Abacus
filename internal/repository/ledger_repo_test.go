package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestLedgerRepo_ListByWallet_NonExistentWalletReturnsEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := NewLedgerRepo(db)
	entries, total, err := repo.ListByWallet(context.Background(), "does-not-exist", 50, 0)
	if err != nil {
		t.Fatalf("ListByWallet on a missing wallet should not error, got %v", err)
	}
	if total != 0 || len(entries) != 0 {
		t.Fatalf("expected no entries for a missing wallet, got total=%d len=%d", total, len(entries))
	}
}

func TestLedgerRepo_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := NewLedgerRepo(db)
	_, err := repo.GetByID(context.Background(), "does-not-exist")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for a missing ledger entry, got %v", err)
	}
}

func TestLedgerRepo_ListByTransaction_NonExistentWalletReturnsEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := NewLedgerRepo(db)
	entries, err := repo.ListByTransaction(context.Background(), "does-not-exist", "also-missing")
	if err != nil {
		t.Fatalf("ListByTransaction on a missing wallet should not error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
}
