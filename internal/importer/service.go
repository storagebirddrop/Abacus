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
	"github.com/storagebirddrop/abacus/internal/ledger"
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

type LedgerRepo interface {
	InsertWithTx(ctx context.Context, tx *sql.Tx, e *domain.LedgerEntry) error
}

type UTXORepo interface {
	UpsertWithTx(ctx context.Context, tx *sql.Tx, u *domain.UTXO) error
	MarkSpentWithTx(ctx context.Context, tx *sql.Tx, prevTxid string, prevVout int, spentByTxid string) error
}

type JobRepo interface {
	CreateWithTx(ctx context.Context, tx *sql.Tx, job *domain.ImportJob) error
	UpdateWithTx(ctx context.Context, tx *sql.Tx, job *domain.ImportJob) error
	Update(ctx context.Context, job *domain.ImportJob) error
}

type Service struct {
	db         Storer
	txRepo     TxRepo
	lblRepo    LabelRepo
	ledgerRepo LedgerRepo
	utxoRepo   UTXORepo
	jobRepo    JobRepo
}

func NewService(db Storer, txRepo TxRepo, lblRepo LabelRepo, ledgerRepo LedgerRepo, utxoRepo UTXORepo, jobRepo JobRepo) *Service {
	return &Service{
		db:         db,
		txRepo:     txRepo,
		lblRepo:    lblRepo,
		ledgerRepo: ledgerRepo,
		utxoRepo:   utxoRepo,
		jobRepo:    jobRepo,
	}
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
		ID:        uuid.New().String(),
		WalletID:  walletID,
		Source:    imp.Name(),
		Filename:  filename,
		Status:    "running",
		StartedAt: &now,
	}

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

	// Persist transactions and build txid → UUID map for linking.
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

	// Group inputs/outputs by parent txid for ledger building.
	inputsByTxid := map[string][]domain.TransactionInput{}
	for i := range result.Inputs {
		in := &result.Inputs[i]
		txID, ok := txMap[in.ParentTxid]
		if !ok {
			continue
		}
		in.TransactionID = txID
		if err := s.txRepo.UpsertInputWithTx(ctx, dbTx, in); err != nil {
			_ = dbTx.Rollback()
			return nil, fmt.Errorf("upsert input: %w", err)
		}
		inputsByTxid[in.ParentTxid] = append(inputsByTxid[in.ParentTxid], *in)
	}

	outputsByTxid := map[string][]domain.TransactionOutput{}
	for i := range result.Outputs {
		out := &result.Outputs[i]
		txID, ok := txMap[out.ParentTxid]
		if !ok {
			continue
		}
		out.TransactionID = txID
		if err := s.txRepo.UpsertOutputWithTx(ctx, dbTx, out); err != nil {
			_ = dbTx.Rollback()
			return nil, fmt.Errorf("upsert output: %w", err)
		}
		outputsByTxid[out.ParentTxid] = append(outputsByTxid[out.ParentTxid], *out)
	}

	// Build ledger entries and UTXOs for each transaction.
	for i := range result.Transactions {
		t := &result.Transactions[i]
		ins := inputsByTxid[t.Txid]
		outs := outputsByTxid[t.Txid]

		entries, utxos, spentKeys := ledger.Build(t, ins, outs)

		for j := range entries {
			if err := s.ledgerRepo.InsertWithTx(ctx, dbTx, &entries[j]); err != nil {
				_ = dbTx.Rollback()
				return nil, fmt.Errorf("insert ledger entry: %w", err)
			}
		}
		for j := range utxos {
			if err := s.utxoRepo.UpsertWithTx(ctx, dbTx, &utxos[j]); err != nil {
				_ = dbTx.Rollback()
				return nil, fmt.Errorf("upsert utxo: %w", err)
			}
		}
		for _, sk := range spentKeys {
			if err := s.utxoRepo.MarkSpentWithTx(ctx, dbTx, sk.Txid, sk.Vout, t.Txid); err != nil {
				_ = dbTx.Rollback()
				return nil, fmt.Errorf("mark utxo spent: %w", err)
			}
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

	_ = io.Discard
	return job, nil
}
