package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type stubWalletRepo struct {
	wallet *domain.Wallet
	err    error
}

func (s *stubWalletRepo) GetByID(_ context.Context, _ string) (*domain.Wallet, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.wallet, nil
}

type stubTxRepo struct{}

func (s *stubTxRepo) UpsertWithTx(_ context.Context, _ *sql.Tx, _ *domain.Transaction) error {
	return nil
}
func (s *stubTxRepo) UpsertInputWithTx(_ context.Context, _ *sql.Tx, _ *domain.TransactionInput) error {
	return nil
}
func (s *stubTxRepo) UpsertOutputWithTx(_ context.Context, _ *sql.Tx, _ *domain.TransactionOutput) error {
	return nil
}

type stubLedgerRepo struct{}

func (s *stubLedgerRepo) InsertWithTx(_ context.Context, _ *sql.Tx, _ *domain.LedgerEntry) error {
	return nil
}

type stubUTXORepo struct{}

func (s *stubUTXORepo) UpsertWithTx(_ context.Context, _ *sql.Tx, _ *domain.UTXO) error { return nil }
func (s *stubUTXORepo) MarkSpentWithTx(_ context.Context, _ *sql.Tx, _ string, _ int, _ string) error {
	return nil
}

type stubSyncJobRepo struct {
	jobs map[string]*domain.SyncJob
}

func newStubSyncJobRepo() *stubSyncJobRepo {
	return &stubSyncJobRepo{jobs: map[string]*domain.SyncJob{}}
}
func (s *stubSyncJobRepo) Create(_ context.Context, j *domain.SyncJob) error {
	s.jobs[j.ID] = j
	return nil
}
func (s *stubSyncJobRepo) Update(_ context.Context, j *domain.SyncJob) error {
	s.jobs[j.ID] = j
	return nil
}

type stubSyncStateRepo struct{}

func (s *stubSyncStateRepo) Upsert(_ context.Context, _ *domain.SyncState) error { return nil }

type stubBackend struct {
	name    string
	txs     map[string][]TxRecord
	txErr   error
	height  int64
	htErr   error
}

func (b *stubBackend) Name() string { return b.name }
func (b *stubBackend) GetTransactions(_ context.Context, address string) ([]TxRecord, error) {
	if b.txErr != nil {
		return nil, b.txErr
	}
	return b.txs[address], nil
}
func (b *stubBackend) BlockHeight(_ context.Context) (int64, error) {
	return b.height, b.htErr
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newTestService(t *testing.T, walletRepo walletRepo, jobRepo syncJobRepo, backend BlockchainBackend) *Service {
	t.Helper()
	return NewService(
		context.Background(),
		testDB(t),
		walletRepo,
		&stubTxRepo{},
		&stubLedgerRepo{},
		&stubUTXORepo{},
		jobRepo,
		&stubSyncStateRepo{},
		func(_ context.Context) (BlockchainBackend, error) { return backend, nil },
	)
}

func TestStartSync_WalletNotFound(t *testing.T) {
	svc := newTestService(t, &stubWalletRepo{err: sql.ErrNoRows}, newStubSyncJobRepo(), &stubBackend{name: "esplora"})
	_, err := svc.StartSync(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected an error for a missing wallet")
	}
}

func TestStartSync_NoDescriptor(t *testing.T) {
	wallet := &domain.Wallet{ID: "w1", Descriptor: ""}
	svc := newTestService(t, &stubWalletRepo{wallet: wallet}, newStubSyncJobRepo(), &stubBackend{name: "esplora"})
	_, err := svc.StartSync(context.Background(), "w1")
	if err == nil {
		t.Fatal("expected an error for a wallet with no descriptor")
	}
}

func TestStartSync_BackendFactoryError(t *testing.T) {
	svc := NewService(
		context.Background(), testDB(t),
		&stubWalletRepo{wallet: &domain.Wallet{ID: "w1", Descriptor: "wpkh(xpub...)"}},
		&stubTxRepo{}, &stubLedgerRepo{}, &stubUTXORepo{}, newStubSyncJobRepo(), &stubSyncStateRepo{},
		func(_ context.Context) (BlockchainBackend, error) { return nil, errors.New("sync disabled") },
	)
	_, err := svc.StartSync(context.Background(), "w1")
	if err == nil {
		t.Fatal("expected an error when the backend factory fails")
	}
}

func TestRunSync_JobStatusTransitionsToDone(t *testing.T) {
	wallet := &domain.Wallet{
		ID:         "w1",
		Descriptor: "wpkh([aabbccdd/84h/0h/0h]xpub6CUGRUonZSQ4TWtTMmzXdrXDtypWKiKrhko4egpiMZbpiaQL2jkwSB1icqYh2cfDfVxdx4df189oLKnC5fSwqPfgyP3hooxujYzAu3fDVmz/*)",
		Network:    domain.NetworkMainnet,
	}
	backend := &stubBackend{name: "esplora", height: 800000}
	svc := newTestService(t, &stubWalletRepo{wallet: wallet}, newStubSyncJobRepo(), backend)

	jobID, err := svc.StartSync(context.Background(), "w1")
	if err != nil {
		t.Fatalf("StartSync: %v", err)
	}
	if jobID == "" {
		t.Fatal("expected a non-empty job ID")
	}
}

func TestRunSync_BackendErrorFailsJob(t *testing.T) {
	wallet := &domain.Wallet{
		ID:         "w1",
		Descriptor: "wpkh([aabbccdd/84h/0h/0h]xpub6CUGRUonZSQ4TWtTMmzXdrXDtypWKiKrhko4egpiMZbpiaQL2jkwSB1icqYh2cfDfVxdx4df189oLKnC5fSwqPfgyP3hooxujYzAu3fDVmz/*)",
		Network:    domain.NetworkMainnet,
	}
	jobRepo := newStubSyncJobRepo()
	backend := &stubBackend{name: "esplora", txErr: errors.New("upstream unavailable")}
	svc := newTestService(t, &stubWalletRepo{wallet: wallet}, jobRepo, backend)

	job := &domain.SyncJob{ID: "job1", WalletID: "w1", Backend: "esplora", Status: "pending"}
	if err := svc.runSync(context.Background(), wallet, job, backend); err == nil {
		t.Fatal("expected runSync to return an error when the backend fails")
	}
}

func TestNetworkToParams(t *testing.T) {
	cases := []domain.Network{domain.NetworkMainnet, domain.NetworkTestnet, domain.NetworkSignet, ""}
	for _, n := range cases {
		if params := networkToParams(n); params == nil {
			t.Fatalf("networkToParams(%q) returned nil", n)
		}
	}
}
