package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type ImportJobRepo struct {
	db *sql.DB
}

func NewImportJobRepo(db *sql.DB) *ImportJobRepo {
	return &ImportJobRepo{db: db}
}

func (r *ImportJobRepo) Create(ctx context.Context, job *domain.ImportJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO import_jobs (id, wallet_id, source, filename, status, records_imported, error_message, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.WalletID, job.Source, job.Filename, job.Status,
		job.RecordsImported, job.ErrorMessage,
		toUnixPtr(job.StartedAt), toUnixPtr(job.FinishedAt),
	)
	return err
}

func (r *ImportJobRepo) Update(ctx context.Context, job *domain.ImportJob) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE import_jobs SET status=?, records_imported=?, error_message=?, started_at=?, finished_at=?
		 WHERE id=?`,
		job.Status, job.RecordsImported, job.ErrorMessage,
		toUnixPtr(job.StartedAt), toUnixPtr(job.FinishedAt),
		job.ID,
	)
	return err
}

func (r *ImportJobRepo) GetByID(ctx context.Context, id string) (*domain.ImportJob, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, wallet_id, source, filename, status, records_imported, error_message, started_at, finished_at
		 FROM import_jobs WHERE id=?`, id)
	return scanImportJob(row)
}

func (r *ImportJobRepo) ListByWallet(ctx context.Context, walletID string) ([]*domain.ImportJob, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, wallet_id, source, filename, status, records_imported, error_message, started_at, finished_at
		 FROM import_jobs WHERE wallet_id=? ORDER BY rowid DESC`, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []*domain.ImportJob
	for rows.Next() {
		j, err := scanImportJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func scanImportJob(s scanner) (*domain.ImportJob, error) {
	var job domain.ImportJob
	var startedUnix, finishedUnix sql.NullInt64
	err := s.Scan(
		&job.ID, &job.WalletID, &job.Source, &job.Filename,
		&job.Status, &job.RecordsImported, &job.ErrorMessage,
		&startedUnix, &finishedUnix,
	)
	if err != nil {
		return nil, err
	}
	if startedUnix.Valid {
		t := time.Unix(startedUnix.Int64, 0).UTC()
		job.StartedAt = &t
	}
	if finishedUnix.Valid {
		t := time.Unix(finishedUnix.Int64, 0).UTC()
		job.FinishedAt = &t
	}
	return &job, nil
}

func toUnixPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	v := t.Unix()
	return &v
}

func (r *ImportJobRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, job *domain.ImportJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO import_jobs (id, wallet_id, source, filename, status, records_imported, error_message, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.WalletID, job.Source, job.Filename, job.Status,
		job.RecordsImported, job.ErrorMessage,
		toUnixPtr(job.StartedAt), toUnixPtr(job.FinishedAt),
	)
	return err
}

func (r *ImportJobRepo) UpdateWithTx(ctx context.Context, tx *sql.Tx, job *domain.ImportJob) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE import_jobs SET status=?, records_imported=?, error_message=?, started_at=?, finished_at=?
		 WHERE id=?`,
		job.Status, job.RecordsImported, job.ErrorMessage,
		toUnixPtr(job.StartedAt), toUnixPtr(job.FinishedAt),
		job.ID,
	)
	return err
}

// Ensure fmt is used.
var _ = fmt.Sprintf
