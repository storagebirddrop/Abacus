package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type CostBasisRepo struct {
	db *sql.DB
}

func NewCostBasisRepo(db *sql.DB) *CostBasisRepo {
	return &CostBasisRepo{db: db}
}

func (r *CostBasisRepo) UpsertWithTx(ctx context.Context, tx *sql.Tx, cb *domain.CostBasisRecord) error {
	if cb.ID == "" {
		cb.ID = uuid.New().String()
	}
	var disposedAt interface{}
	if cb.DisposedAt != nil {
		disposedAt = cb.DisposedAt.Unix()
	}
	var proceedsFiat, gainFiat interface{}
	if cb.ProceedsFiat != nil {
		proceedsFiat = *cb.ProceedsFiat
	}
	if cb.GainFiat != nil {
		gainFiat = *cb.GainFiat
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO cost_basis_records
		 (id, wallet_id, txid, vout, acquired_at, cost_sats, cost_fiat, fiat_currency, method,
		  disposed_at, proceeds_fiat, gain_fiat)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   cost_fiat=excluded.cost_fiat,
		   disposed_at=excluded.disposed_at,
		   proceeds_fiat=excluded.proceeds_fiat,
		   gain_fiat=excluded.gain_fiat`,
		cb.ID, cb.WalletID, cb.Txid, cb.Vout,
		cb.AcquiredAt.Unix(), cb.CostSats, cb.CostFiat, cb.FiatCurrency, string(cb.Method),
		disposedAt, proceedsFiat, gainFiat,
	)
	return err
}

func (r *CostBasisRepo) DeleteByWallet(ctx context.Context, tx *sql.Tx, walletID string) error {
	_, err := tx.ExecContext(ctx,
		`DELETE FROM cost_basis_records WHERE wallet_id=?`, walletID,
	)
	return err
}

func (r *CostBasisRepo) ListByWallet(ctx context.Context, walletID string) ([]*domain.CostBasisRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, wallet_id, txid, vout, acquired_at, cost_sats, cost_fiat, fiat_currency, method,
		        disposed_at, proceeds_fiat, gain_fiat
		 FROM cost_basis_records WHERE wallet_id=?
		 ORDER BY acquired_at ASC`,
		walletID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []*domain.CostBasisRecord
	for rows.Next() {
		cb, err := scanCostBasis(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, cb)
	}
	return records, rows.Err()
}

func scanCostBasis(s scanner) (*domain.CostBasisRecord, error) {
	var cb domain.CostBasisRecord
	var acquiredUnix int64
	var disposedUnix sql.NullInt64
	var proceedsFiat, gainFiat sql.NullInt64
	var method string
	err := s.Scan(
		&cb.ID, &cb.WalletID, &cb.Txid, &cb.Vout,
		&acquiredUnix, &cb.CostSats, &cb.CostFiat, &cb.FiatCurrency, &method,
		&disposedUnix, &proceedsFiat, &gainFiat,
	)
	if err != nil {
		return nil, err
	}
	cb.Method = domain.CostBasisMethod(method)
	cb.AcquiredAt = time.Unix(acquiredUnix, 0).UTC()
	if disposedUnix.Valid {
		t := time.Unix(disposedUnix.Int64, 0).UTC()
		cb.DisposedAt = &t
	}
	if proceedsFiat.Valid {
		cb.ProceedsFiat = &proceedsFiat.Int64
	}
	if gainFiat.Valid {
		cb.GainFiat = &gainFiat.Int64
	}
	return &cb, nil
}
