package sync

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/ledger"
)

const gapLimit = 20

// Interfaces for the repos the sync service needs.

type Storer interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type walletRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
}

type txRepo interface {
	UpsertWithTx(ctx context.Context, tx *sql.Tx, t *domain.Transaction) error
	UpsertInputWithTx(ctx context.Context, tx *sql.Tx, in *domain.TransactionInput) error
	UpsertOutputWithTx(ctx context.Context, tx *sql.Tx, out *domain.TransactionOutput) error
}

type ledgerRepo interface {
	InsertWithTx(ctx context.Context, tx *sql.Tx, e *domain.LedgerEntry) error
}

type utxoRepo interface {
	UpsertWithTx(ctx context.Context, tx *sql.Tx, u *domain.UTXO) error
	MarkSpentWithTx(ctx context.Context, tx *sql.Tx, prevTxid string, prevVout int, spentByTxid string) error
}

type syncJobRepo interface {
	Create(ctx context.Context, j *domain.SyncJob) error
	Update(ctx context.Context, j *domain.SyncJob) error
}

type syncStateRepo interface {
	Upsert(ctx context.Context, s *domain.SyncState) error
}

// Service orchestrates blockchain sync for a wallet.
type Service struct {
	db            Storer
	walletRepo    walletRepo
	txRepo        txRepo
	ledgerRepo    ledgerRepo
	utxoRepo      utxoRepo
	syncJobRepo   syncJobRepo
	syncStateRepo syncStateRepo
	backend       BlockchainBackend
}

func NewService(
	db Storer,
	walletRepo walletRepo,
	txRepo txRepo,
	ledgerRepo ledgerRepo,
	utxoRepo utxoRepo,
	syncJobRepo syncJobRepo,
	syncStateRepo syncStateRepo,
	backend BlockchainBackend,
) *Service {
	return &Service{
		db:            db,
		walletRepo:    walletRepo,
		txRepo:        txRepo,
		ledgerRepo:    ledgerRepo,
		utxoRepo:      utxoRepo,
		syncJobRepo:   syncJobRepo,
		syncStateRepo: syncStateRepo,
		backend:       backend,
	}
}

// StartSync creates a sync job and starts the sync in a goroutine.
// Returns the job ID immediately so the caller can poll for status.
func (s *Service) StartSync(ctx context.Context, walletID string) (string, error) {
	wallet, err := s.walletRepo.GetByID(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("get wallet: %w", err)
	}
	if wallet.Descriptor == "" {
		return "", fmt.Errorf("wallet has no descriptor — cannot derive addresses")
	}

	now := time.Now().UTC()
	job := &domain.SyncJob{
		ID:        uuid.New().String(),
		WalletID:  walletID,
		Backend:   s.backend.Name(),
		Status:    "pending",
		StartedAt: now,
	}
	if err := s.syncJobRepo.Create(ctx, job); err != nil {
		return "", fmt.Errorf("create sync job: %w", err)
	}

	// Run sync asynchronously.
	go func() {
		bgCtx := context.Background()
		if err := s.runSync(bgCtx, wallet, job); err != nil {
			fin := time.Now().UTC()
			job.Status = "failed"
			job.ErrorMsg = err.Error()
			job.FinishedAt = &fin
			_ = s.syncJobRepo.Update(bgCtx, job)
		}
	}()

	return job.ID, nil
}

func (s *Service) runSync(ctx context.Context, wallet *domain.Wallet, job *domain.SyncJob) error {
	job.Status = "running"
	if err := s.syncJobRepo.Update(ctx, job); err != nil {
		return err
	}

	net := networkToParams(wallet.Network)

	// Derive enough addresses to scan.
	batchSize := gapLimit * 5 // derive 100 at a time; we may fetch multiple rounds
	receiving, change, err := DeriveAddresses(wallet.Descriptor, net, batchSize)
	if err != nil {
		return fmt.Errorf("derive addresses: %w", err)
	}

	// Collect all unique TxRecords across all addresses.
	txByID := map[string]TxRecord{}

	// addressSet is used to mark outputs as mine.
	addressSet := map[string]bool{}
	for _, a := range receiving {
		addressSet[a] = true
	}
	for _, a := range change {
		addressSet[a] = true
	}

	scan := func(addrs []string) (int, error) {
		totalScanned := 0
		consecutiveEmpty := 0
		for _, addr := range addrs {
			txs, err := s.backend.GetTransactions(ctx, addr)
			if err != nil {
				return totalScanned, fmt.Errorf("fetch tx for %s: %w", addr, err)
			}
			totalScanned++
			if len(txs) == 0 {
				consecutiveEmpty++
				if consecutiveEmpty >= gapLimit {
					break
				}
			} else {
				consecutiveEmpty = 0
				for _, tx := range txs {
					txByID[tx.Txid] = tx
				}
			}
		}
		return totalScanned, nil
	}

	rScanned, err := scan(receiving)
	if err != nil {
		return err
	}
	cScanned, err := scan(change)
	if err != nil {
		return err
	}
	job.AddressesScanned = rScanned + cScanned

	// Get current block height.
	height, _ := s.backend.BlockHeight(ctx)

	// Convert TxRecords to domain types.
	txSlice := make([]domain.Transaction, 0, len(txByID))
	inputsByTxid := map[string][]domain.TransactionInput{}
	outputsByTxid := map[string][]domain.TransactionOutput{}

	for _, rec := range txByID {
		txID := uuid.New().String()
		var blockTime time.Time
		if rec.BlockTime > 0 {
			blockTime = time.Unix(rec.BlockTime, 0).UTC()
		}
		tx := domain.Transaction{
			ID:          txID,
			WalletID:    wallet.ID,
			Txid:        rec.Txid,
			BlockHeight: rec.BlockHeight,
			BlockTime:   blockTime,
			FeeSats:     rec.FeeSats,
			Confirmed:   rec.Confirmed,
			CreatedAt:   time.Now().UTC(),
		}
		txSlice = append(txSlice, tx)

		for _, in := range rec.Inputs {
			inputsByTxid[rec.Txid] = append(inputsByTxid[rec.Txid], domain.TransactionInput{
				ID:         uuid.New().String(),
				ParentTxid: rec.Txid,
				PrevTxid:   in.PrevTxid,
				PrevVout:   in.PrevVout,
				Sats:       in.Sats,
				Address:    in.Address,
				IsMine:     addressSet[in.Address],
			})
		}
		for _, out := range rec.Outputs {
			outputsByTxid[rec.Txid] = append(outputsByTxid[rec.Txid], domain.TransactionOutput{
				ID:         uuid.New().String(),
				ParentTxid: rec.Txid,
				Vout:       out.Vout,
				Sats:       out.Sats,
				Address:    out.Address,
				IsMine:     addressSet[out.Address],
			})
		}
	}

	job.TxFound = len(txSlice)

	// Persist everything in one DB transaction.
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txMap := map[string]string{} // txid → domain.Transaction.ID
	for i := range txSlice {
		t := &txSlice[i]
		if err := s.txRepo.UpsertWithTx(ctx, dbTx, t); err != nil {
			_ = dbTx.Rollback()
			return fmt.Errorf("upsert tx %s: %w", t.Txid, err)
		}
		txMap[t.Txid] = t.ID
	}

	for txid, ins := range inputsByTxid {
		txID, ok := txMap[txid]
		if !ok {
			continue
		}
		for i := range ins {
			ins[i].TransactionID = txID
			if err := s.txRepo.UpsertInputWithTx(ctx, dbTx, &ins[i]); err != nil {
				_ = dbTx.Rollback()
				return fmt.Errorf("upsert input: %w", err)
			}
		}
		inputsByTxid[txid] = ins
	}

	for txid, outs := range outputsByTxid {
		txID, ok := txMap[txid]
		if !ok {
			continue
		}
		for i := range outs {
			outs[i].TransactionID = txID
			if err := s.txRepo.UpsertOutputWithTx(ctx, dbTx, &outs[i]); err != nil {
				_ = dbTx.Rollback()
				return fmt.Errorf("upsert output: %w", err)
			}
		}
		outputsByTxid[txid] = outs
	}

	// Build ledger entries and UTXOs.
	for i := range txSlice {
		t := &txSlice[i]
		ins := inputsByTxid[t.Txid]
		outs := outputsByTxid[t.Txid]

		entries, utxos, spentKeys := ledger.Build(t, ins, outs)

		for j := range entries {
			if err := s.ledgerRepo.InsertWithTx(ctx, dbTx, &entries[j]); err != nil {
				_ = dbTx.Rollback()
				return fmt.Errorf("insert ledger entry: %w", err)
			}
		}
		for j := range utxos {
			if err := s.utxoRepo.UpsertWithTx(ctx, dbTx, &utxos[j]); err != nil {
				_ = dbTx.Rollback()
				return fmt.Errorf("upsert utxo: %w", err)
			}
		}
		for _, sk := range spentKeys {
			if err := s.utxoRepo.MarkSpentWithTx(ctx, dbTx, sk.Txid, sk.Vout, t.Txid); err != nil {
				_ = dbTx.Rollback()
				return fmt.Errorf("mark utxo spent: %w", err)
			}
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Update sync state.
	_ = s.syncStateRepo.Upsert(ctx, &domain.SyncState{
		WalletID:        wallet.ID,
		LastSyncedAt:    time.Now().UTC(),
		ReceiveGapStart: rScanned,
		ChangeGapStart:  cScanned,
		BlockHeight:     height,
	})

	fin := time.Now().UTC()
	job.Status = "done"
	job.FinishedAt = &fin
	return s.syncJobRepo.Update(ctx, job)
}

func networkToParams(n domain.Network) *chaincfg.Params {
	switch n {
	case domain.NetworkTestnet:
		return &chaincfg.TestNet3Params
	case domain.NetworkSignet:
		return &chaincfg.SigNetParams
	default:
		return &chaincfg.MainNetParams
	}
}
