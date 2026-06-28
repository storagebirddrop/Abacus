package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
)

type LedgerRepo struct {
	db *sql.DB
}

func NewLedgerRepo(db *sql.DB) *LedgerRepo {
	return &LedgerRepo{db: db}
}

func (r *LedgerRepo) InsertWithTx(ctx context.Context, tx *sql.Tx, e *domain.LedgerEntry) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	_, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO ledger_entries
		 (id, wallet_id, transaction_id, type, sats, fiat_amount, fiat_currency,
		  price_snapshot_id, category, counterparty_id, note, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.WalletID, e.TransactionID, string(e.Type),
		e.Sats, e.FiatAmount, e.FiatCurrency,
		nullString(e.PriceSnapshotID), string(e.Category),
		nullString(e.CounterpartyID), e.Note, e.CreatedAt.Unix(),
	)
	return err
}

func (r *LedgerRepo) ListByWallet(ctx context.Context, walletID string, limit, offset int) ([]*domain.LedgerEntry, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_entries WHERE wallet_id=?`, walletID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, wallet_id, transaction_id, type, sats, fiat_amount, fiat_currency,
		        COALESCE(price_snapshot_id,''), category, COALESCE(counterparty_id,''), note, created_at
		 FROM ledger_entries WHERE wallet_id=?
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		walletID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var entries []*domain.LedgerEntry
	for rows.Next() {
		e, err := scanLedgerEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func (r *LedgerRepo) GetByID(ctx context.Context, id string) (*domain.LedgerEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, wallet_id, transaction_id, type, sats, fiat_amount, fiat_currency,
		        COALESCE(price_snapshot_id,''), category, COALESCE(counterparty_id,''), note, created_at
		 FROM ledger_entries WHERE id=?`, id)
	return scanLedgerEntry(row)
}

func scanLedgerEntry(s scanner) (*domain.LedgerEntry, error) {
	var e domain.LedgerEntry
	var createdUnix int64
	var entryType, category string
	err := s.Scan(
		&e.ID, &e.WalletID, &e.TransactionID, &entryType,
		&e.Sats, &e.FiatAmount, &e.FiatCurrency,
		&e.PriceSnapshotID, &category, &e.CounterpartyID, &e.Note, &createdUnix,
	)
	if err != nil {
		return nil, err
	}
	e.Type = domain.EntryType(entryType)
	e.Category = domain.Category(category)
	e.CreatedAt = time.Unix(createdUnix, 0).UTC()
	return &e, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
