package importer

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// Storer is the minimal DB interface the import service needs.
type Storer interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type TxRepo interface {
	UpsertWithTx(ctx context.Context, tx *sql.Tx, t *domain.Transaction) error
	UpsertInputWithTx(ctx context.Context, tx *sql.Tx, in *domain.TransactionInput) error
	UpsertOutputWithTx(ctx context.Context, tx *sql.Tx, out *domain.TransactionOutput) error
}

type LabelRepo interface {
	UpsertWithTx(ctx context.Context, tx *sql.Tx, l *domain.Label) error
}

type JobRepo interface {
	CreateWithTx(ctx context.Context, tx *sql.Tx, job *domain.ImportJob) error
	UpdateWithTx(ctx context.Context, tx *sql.Tx, job *domain.ImportJob) error
	Update(ctx context.Context, job *domain.ImportJob) error
}

type Service struct {
	db       Storer
	txRepo   TxRepo
	lblRepo  LabelRepo
	jobRepo  JobRepo
}

func NewService(db Storer, txRepo TxRepo, lblRepo LabelRepo, jobRepo JobRepo) *Service {
	return &Service{db: db, txRepo: txRepo, lblRepo: lblRepo, jobRepo: jobRepo}
}

// Run detects format, creates an ImportJob, and processes the file.
func (s *Service) Run(ctx context.Context, walletID, filename string, data []byte) (*domain.ImportJob, error) {
	rs := bytes.NewReader(data)
	imp := Detect(filename, rs)
	if imp == nil {
		return nil, fmt.Errorf("unrecognized file format: %s", filename)
	}

	now := time.Now().UTC()
	job := &domain.ImportJob{
		ID:       uuid.New().String(),
		WalletID: walletID,
		Source:   imp.Name(),
		Filename: filename,
		Status:   "running",
		StartedAt: &now,
	}

	// Persist job in a transaction
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if err := s.jobRepo.CreateWithTx(ctx, dbTx, job); err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}

	result, importErr := imp.Import(ctx, walletID, bytes.NewReader(data))
	if importErr != nil {
		_ = dbTx.Rollback()
		fin := time.Now().UTC()
		job.Status = "failed"
		job.ErrorMessage = importErr.Error()
		job.FinishedAt = &fin
		_ = s.jobRepo.Update(ctx, job)
		return job, importErr
	}

	// Persist all imported data in the same transaction
	txMap := map[string]string{} // txid → transaction.ID
	for i := range result.Transactions {
		t := &result.Transactions[i]
		t.WalletID = walletID
		if err := s.txRepo.UpsertWithTx(ctx, dbTx, t); err != nil {
			_ = dbTx.Rollback()
			return nil, fmt.Errorf("upsert tx %s: %w", t.Txid, err)
		}
		txMap[t.Txid] = t.ID
	}

	for i := range result.Inputs {
		in := &result.Inputs[i]
		if in.TransactionID == "" {
			continue // will be linked in phase 2
		}
		if err := s.txRepo.UpsertInputWithTx(ctx, dbTx, in); err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
	}

	for i := range result.Outputs {
		out := &result.Outputs[i]
		if out.TransactionID == "" {
			continue
		}
		if err := s.txRepo.UpsertOutputWithTx(ctx, dbTx, out); err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
	}

	for i := range result.Labels {
		l := &result.Labels[i]
		if err := s.lblRepo.UpsertWithTx(ctx, dbTx, l); err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
	}

	fin := time.Now().UTC()
	job.Status = "done"
	job.RecordsImported = len(result.Transactions) + len(result.Labels)
	job.FinishedAt = &fin
	if err := s.jobRepo.UpdateWithTx(ctx, dbTx, job); err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}

	if err := dbTx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	_ = io.Discard // keep io imported
	return job, nil
}
