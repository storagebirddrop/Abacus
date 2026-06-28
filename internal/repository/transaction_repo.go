package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type TransactionRepo struct {
	db *sql.DB
}

func NewTransactionRepo(db *sql.DB) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) UpsertWithTx(ctx context.Context, tx *sql.Tx, t *domain.Transaction) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	confirmed := 0
	if t.Confirmed {
		confirmed = 1
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO transactions (id, wallet_id, txid, block_height, block_hash, block_time, fee_sats, confirmed, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(wallet_id, txid) DO UPDATE SET
		   block_height=excluded.block_height,
		   block_hash=excluded.block_hash,
		   block_time=excluded.block_time,
		   fee_sats=excluded.fee_sats,
		   confirmed=excluded.confirmed`,
		t.ID, t.WalletID, t.Txid, t.BlockHeight, t.BlockHash,
		t.BlockTime.Unix(), t.FeeSats, confirmed, t.CreatedAt.Unix(),
	)
	return err
}

func (r *TransactionRepo) UpsertInputWithTx(ctx context.Context, tx *sql.Tx, in *domain.TransactionInput) error {
	if in.ID == "" {
		in.ID = uuid.New().String()
	}
	isMine := 0
	if in.IsMine {
		isMine = 1
	}
	_, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO transaction_inputs (id, transaction_id, prev_txid, prev_vout, sats, address, sequence, is_mine)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ID, in.TransactionID, in.PrevTxid, in.PrevVout, in.Sats, in.Address, in.Sequence, isMine,
	)
	return err
}

func (r *TransactionRepo) UpsertOutputWithTx(ctx context.Context, tx *sql.Tx, out *domain.TransactionOutput) error {
	if out.ID == "" {
		out.ID = uuid.New().String()
	}
	isMine := 0
	if out.IsMine {
		isMine = 1
	}
	_, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO transaction_outputs (id, transaction_id, vout, sats, address, script_pubkey, is_mine)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		out.ID, out.TransactionID, out.Vout, out.Sats, out.Address, out.ScriptPubkey, isMine,
	)
	return err
}

func (r *TransactionRepo) List(ctx context.Context, walletID string, limit, offset int) ([]*domain.Transaction, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM transactions WHERE wallet_id=?`, walletID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, wallet_id, txid, block_height, block_hash, block_time, fee_sats, confirmed, created_at
		 FROM transactions WHERE wallet_id=? ORDER BY block_time DESC, created_at DESC LIMIT ? OFFSET ?`,
		walletID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var txs []*domain.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, 0, err
		}
		txs = append(txs, t)
	}
	return txs, total, rows.Err()
}

func (r *TransactionRepo) GetByTxid(ctx context.Context, walletID, txid string) (*domain.Transaction, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, wallet_id, txid, block_height, block_hash, block_time, fee_sats, confirmed, created_at
		 FROM transactions WHERE wallet_id=? AND txid=?`, walletID, txid)
	return scanTransaction(row)
}

func (r *TransactionRepo) GetInputsByTransactionID(ctx context.Context, txID string) ([]*domain.TransactionInput, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, transaction_id, prev_txid, prev_vout, sats, address, sequence, is_mine
		 FROM transaction_inputs WHERE transaction_id=?`, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ins []*domain.TransactionInput
	for rows.Next() {
		var in domain.TransactionInput
		var isMine int
		if err := rows.Scan(&in.ID, &in.TransactionID, &in.PrevTxid, &in.PrevVout, &in.Sats, &in.Address, &in.Sequence, &isMine); err != nil {
			return nil, err
		}
		in.IsMine = isMine == 1
		ins = append(ins, &in)
	}
	return ins, rows.Err()
}

func (r *TransactionRepo) GetOutputsByTransactionID(ctx context.Context, txID string) ([]*domain.TransactionOutput, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, transaction_id, vout, sats, address, script_pubkey, is_mine
		 FROM transaction_outputs WHERE transaction_id=?`, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var outs []*domain.TransactionOutput
	for rows.Next() {
		var out domain.TransactionOutput
		var isMine int
		if err := rows.Scan(&out.ID, &out.TransactionID, &out.Vout, &out.Sats, &out.Address, &out.ScriptPubkey, &isMine); err != nil {
			return nil, err
		}
		out.IsMine = isMine == 1
		outs = append(outs, &out)
	}
	return outs, rows.Err()
}

func scanTransaction(s scanner) (*domain.Transaction, error) {
	var t domain.Transaction
	var blockTimeUnix, createdUnix int64
	var confirmed int
	err := s.Scan(
		&t.ID, &t.WalletID, &t.Txid, &t.BlockHeight, &t.BlockHash,
		&blockTimeUnix, &t.FeeSats, &confirmed, &createdUnix,
	)
	if err != nil {
		return nil, err
	}
	t.BlockTime = time.Unix(blockTimeUnix, 0).UTC()
	t.CreatedAt = time.Unix(createdUnix, 0).UTC()
	t.Confirmed = confirmed == 1
	return &t, nil
}

func (r *TransactionRepo) DB() *sql.DB { return r.db }
