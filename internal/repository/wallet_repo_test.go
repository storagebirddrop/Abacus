package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestWalletRepo_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := NewWalletRepo(db)
	_, err := repo.GetByID(context.Background(), "does-not-exist")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for a missing wallet, got %v", err)
	}
}

func TestWalletRepo_Delete_NonExistentWalletIsNotAnError(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := NewWalletRepo(db)
	if err := repo.Delete(context.Background(), "does-not-exist"); err != nil {
		t.Fatalf("Delete on a missing wallet should not error, got %v", err)
	}
}

func TestWalletRepo_UpdateDescriptor_NonExistentWalletIsNotAnError(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := NewWalletRepo(db)
	if err := repo.UpdateDescriptor(context.Background(), "does-not-exist", "wpkh(xpub...)", "aabbccdd"); err != nil {
		t.Fatalf("UpdateDescriptor on a missing wallet should not error, got %v", err)
	}
}
