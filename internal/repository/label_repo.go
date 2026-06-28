package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type LabelRepo struct {
	db *sql.DB
}

func NewLabelRepo(db *sql.DB) *LabelRepo {
	return &LabelRepo{db: db}
}

func (r *LabelRepo) UpsertWithTx(ctx context.Context, tx *sql.Tx, l *domain.Label) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now().UTC()
	}
	var spendable *int
	if l.Spendable != nil {
		v := 0
		if *l.Spendable {
			v = 1
		}
		spendable = &v
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO labels (id, wallet_id, type, ref, label, origin, spendable, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(wallet_id, type, ref) DO UPDATE SET
		   label=excluded.label,
		   origin=excluded.origin,
		   spendable=excluded.spendable`,
		l.ID, l.WalletID, l.Type, l.Ref, l.Label, l.Origin, spendable, l.CreatedAt.Unix(),
	)
	return err
}

func (r *LabelRepo) ListByWallet(ctx context.Context, walletID string) ([]*domain.Label, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, wallet_id, type, ref, label, origin, spendable, created_at
		 FROM labels WHERE wallet_id=? ORDER BY created_at DESC`, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var labels []*domain.Label
	for rows.Next() {
		var l domain.Label
		var spendable sql.NullInt64
		var createdUnix int64
		if err := rows.Scan(&l.ID, &l.WalletID, &l.Type, &l.Ref, &l.Label, &l.Origin, &spendable, &createdUnix); err != nil {
			return nil, err
		}
		l.CreatedAt = time.Unix(createdUnix, 0).UTC()
		if spendable.Valid {
			v := spendable.Int64 == 1
			l.Spendable = &v
		}
		labels = append(labels, &l)
	}
	return labels, rows.Err()
}
