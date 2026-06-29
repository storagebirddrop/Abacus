package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
)

type SyncJobRepo struct {
	db *sql.DB
}

func NewSyncJobRepo(db *sql.DB) *SyncJobRepo {
	return &SyncJobRepo{db: db}
}

func (r *SyncJobRepo) Create(ctx context.Context, j *domain.SyncJob) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_jobs (id, wallet_id, backend, status, addresses_scanned, tx_found, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		j.ID, j.WalletID, j.Backend, j.Status, j.AddressesScanned, j.TxFound,
		j.StartedAt.UTC().Unix(),
	)
	return err
}

func (r *SyncJobRepo) Update(ctx context.Context, j *domain.SyncJob) error {
	var finishedAt *int64
	if j.FinishedAt != nil {
		v := j.FinishedAt.UTC().Unix()
		finishedAt = &v
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE sync_jobs
		SET status=?, addresses_scanned=?, tx_found=?, error_message=?, finished_at=?
		WHERE id=?`,
		j.Status, j.AddressesScanned, j.TxFound, nilIfEmpty(j.ErrorMsg), finishedAt, j.ID,
	)
	return err
}

func (r *SyncJobRepo) Get(ctx context.Context, id string) (*domain.SyncJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, wallet_id, backend, status, addresses_scanned, tx_found,
		       COALESCE(error_message,''), started_at, finished_at
		FROM sync_jobs WHERE id=?`, id)
	return scanSyncJob(row)
}

func (r *SyncJobRepo) ListByWallet(ctx context.Context, walletID string) ([]*domain.SyncJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, wallet_id, backend, status, addresses_scanned, tx_found,
		       COALESCE(error_message,''), started_at, finished_at
		FROM sync_jobs WHERE wallet_id=? ORDER BY started_at DESC`, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []*domain.SyncJob
	for rows.Next() {
		j, err := scanSyncJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSyncJob(s scannable) (*domain.SyncJob, error) {
	j := &domain.SyncJob{}
	var startedAt int64
	var finishedAt *int64
	if err := s.Scan(
		&j.ID, &j.WalletID, &j.Backend, &j.Status,
		&j.AddressesScanned, &j.TxFound, &j.ErrorMsg,
		&startedAt, &finishedAt,
	); err != nil {
		return nil, err
	}
	j.StartedAt = time.Unix(startedAt, 0).UTC()
	if finishedAt != nil {
		t := time.Unix(*finishedAt, 0).UTC()
		j.FinishedAt = &t
	}
	return j, nil
}

// SyncStateRepo manages the sync state per wallet.
type SyncStateRepo struct {
	db *sql.DB
}

func NewSyncStateRepo(db *sql.DB) *SyncStateRepo {
	return &SyncStateRepo{db: db}
}

func (r *SyncStateRepo) Upsert(ctx context.Context, s *domain.SyncState) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_state (wallet_id, last_synced_at, receive_gap_start, change_gap_start, block_height)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(wallet_id) DO UPDATE SET
		  last_synced_at=excluded.last_synced_at,
		  receive_gap_start=excluded.receive_gap_start,
		  change_gap_start=excluded.change_gap_start,
		  block_height=excluded.block_height`,
		s.WalletID, s.LastSyncedAt.UTC().Unix(),
		s.ReceiveGapStart, s.ChangeGapStart, s.BlockHeight,
	)
	return err
}

func (r *SyncStateRepo) Get(ctx context.Context, walletID string) (*domain.SyncState, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT wallet_id, last_synced_at, receive_gap_start, change_gap_start, block_height
		FROM sync_state WHERE wallet_id=?`, walletID)
	s := &domain.SyncState{}
	var lastSynced int64
	if err := row.Scan(&s.WalletID, &lastSynced, &s.ReceiveGapStart, &s.ChangeGapStart, &s.BlockHeight); err != nil {
		return nil, err
	}
	s.LastSyncedAt = time.Unix(lastSynced, 0).UTC()
	return s, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
