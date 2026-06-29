package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type WalletRepo struct {
	db *sql.DB
}

func NewWalletRepo(db *sql.DB) *WalletRepo {
	return &WalletRepo{db: db}
}

func (r *WalletRepo) Create(ctx context.Context, w *domain.Wallet) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO wallets (id, name, descriptor, fingerprint, type, network, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.Name, w.Descriptor, w.Fingerprint,
		string(w.Type), string(w.Network), string(w.Source),
		now.Unix(), now.Unix(),
	)
	if err != nil {
		return fmt.Errorf("create wallet: %w", err)
	}
	return nil
}

func (r *WalletRepo) GetByID(ctx context.Context, id string) (*domain.Wallet, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, descriptor, fingerprint, type, network, source, created_at, updated_at
		 FROM wallets WHERE id = ?`, id)
	return scanWallet(row)
}

func (r *WalletRepo) List(ctx context.Context) ([]*domain.Wallet, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, descriptor, fingerprint, type, network, source, created_at, updated_at
		 FROM wallets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var wallets []*domain.Wallet
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, rows.Err()
}

func (r *WalletRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM wallets WHERE id = ?`, id)
	return err
}

// UpdateDescriptor sets the descriptor and fingerprint on a wallet only if the
// wallet currently has no descriptor. Existing user data is never overwritten.
func (r *WalletRepo) UpdateDescriptor(ctx context.Context, id, descriptor, fingerprint string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE wallets SET descriptor=?, fingerprint=?, updated_at=?
		 WHERE id=? AND (descriptor='' OR descriptor IS NULL)`,
		descriptor, fingerprint, time.Now().UTC().Unix(), id,
	)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanWallet(s scanner) (*domain.Wallet, error) {
	var w domain.Wallet
	var createdUnix, updatedUnix int64
	err := s.Scan(
		&w.ID, &w.Name, &w.Descriptor, &w.Fingerprint,
		&w.Type, &w.Network, &w.Source,
		&createdUnix, &updatedUnix,
	)
	if err != nil {
		return nil, err
	}
	w.CreatedAt = time.Unix(createdUnix, 0).UTC()
	w.UpdatedAt = time.Unix(updatedUnix, 0).UTC()
	return &w, nil
}
