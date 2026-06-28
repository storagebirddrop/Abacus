package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
)

type UTXORepo struct {
	db *sql.DB
}

func NewUTXORepo(db *sql.DB) *UTXORepo {
	return &UTXORepo{db: db}
}

func (r *UTXORepo) UpsertWithTx(ctx context.Context, tx *sql.Tx, u *domain.UTXO) error {
	spent := 0
	if u.Spent {
		spent = 1
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO utxos (id, wallet_id, txid, vout, sats, address, block_height, block_time, spent, spent_txid, label)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(txid, vout) DO UPDATE SET
		   sats=excluded.sats,
		   block_height=excluded.block_height,
		   block_time=excluded.block_time,
		   spent=excluded.spent,
		   spent_txid=excluded.spent_txid`,
		u.ID, u.WalletID, u.Txid, u.Vout, u.Sats, u.Address,
		u.BlockHeight, u.BlockTime.Unix(), spent, u.SpentTxid, u.Label,
	)
	return err
}

// MarkSpentWithTx marks a UTXO (identified by prev_txid + prev_vout) as spent
// by spentByTxid. No-op if the UTXO is not found (it may belong to a different wallet).
func (r *UTXORepo) MarkSpentWithTx(ctx context.Context, tx *sql.Tx, prevTxid string, prevVout int, spentByTxid string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE utxos SET spent=1, spent_txid=? WHERE txid=? AND vout=? AND spent=0`,
		spentByTxid, prevTxid, prevVout,
	)
	return err
}

func (r *UTXORepo) ListByWallet(ctx context.Context, walletID string, unspentOnly bool) ([]*domain.UTXO, error) {
	q := `SELECT id, wallet_id, txid, vout, sats, address, block_height, block_time, spent, spent_txid, label
	      FROM utxos WHERE wallet_id=?`
	args := []interface{}{walletID}
	if unspentOnly {
		q += ` AND spent=0`
	}
	q += ` ORDER BY block_time DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var utxos []*domain.UTXO
	for rows.Next() {
		u, err := scanUTXO(rows)
		if err != nil {
			return nil, err
		}
		utxos = append(utxos, u)
	}
	return utxos, rows.Err()
}

func scanUTXO(s scanner) (*domain.UTXO, error) {
	var u domain.UTXO
	var blockTimeUnix int64
	var spent int
	err := s.Scan(
		&u.ID, &u.WalletID, &u.Txid, &u.Vout, &u.Sats, &u.Address,
		&u.BlockHeight, &blockTimeUnix, &spent, &u.SpentTxid, &u.Label,
	)
	if err != nil {
		return nil, err
	}
	u.BlockTime = time.Unix(blockTimeUnix, 0).UTC()
	u.Spent = spent == 1
	return &u, nil
}
