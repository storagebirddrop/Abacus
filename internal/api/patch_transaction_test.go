package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// --- stubs ---

type stubWalletStore struct{ wallet *domain.Wallet }

func (s *stubWalletStore) Create(_ context.Context, _ *domain.Wallet) error { return nil }
func (s *stubWalletStore) GetByID(_ context.Context, _ string) (*domain.Wallet, error) {
	if s.wallet == nil {
		return nil, sql.ErrNoRows
	}
	return s.wallet, nil
}
func (s *stubWalletStore) List(_ context.Context) ([]*domain.Wallet, error) { return nil, nil }
func (s *stubWalletStore) Delete(_ context.Context, _ string) error         { return nil }

type stubTxStore struct {
	tx  *domain.Transaction
	err error
}

func (s *stubTxStore) List(_ context.Context, _ string, _, _ int) ([]*domain.Transaction, int, error) {
	return nil, 0, nil
}
func (s *stubTxStore) ListFiltered(_ context.Context, _ string, _ domain.TxFilter) ([]*domain.Transaction, int, error) {
	return nil, 0, nil
}
func (s *stubTxStore) GetByTxid(_ context.Context, _, _ string) (*domain.Transaction, error) {
	return s.tx, s.err
}
func (s *stubTxStore) GetInputsByTransactionID(_ context.Context, _ string) ([]*domain.TransactionInput, error) {
	return nil, nil
}
func (s *stubTxStore) GetOutputsByTransactionID(_ context.Context, _ string) ([]*domain.TransactionOutput, error) {
	return nil, nil
}

type stubTxLedger struct {
	entries []*domain.LedgerEntry
	updated []string // IDs updated
}

func (s *stubTxLedger) ListByTransaction(_ context.Context, _, _ string) ([]*domain.LedgerEntry, error) {
	return s.entries, nil
}
func (s *stubTxLedger) UpdateMetadata(_ context.Context, _ *sql.Tx, id string, _ domain.Category, _, _ string) error {
	s.updated = append(s.updated, id)
	return nil
}

type stubTxJournal struct {
	inserted []*domain.JournalEntry
}

func (s *stubTxJournal) Insert(_ context.Context, _ *sql.Tx, e *domain.JournalEntry) error {
	s.inserted = append(s.inserted, e)
	return nil
}

type stubDB struct{}

func (s *stubDB) BeginTx(_ context.Context, _ *sql.TxOptions) (*sql.Tx, error) {
	// Return a real *sql.Tx by opening an in-memory SQLite DB.
	// We use a no-op approach: return nil and let the handler call Rollback/Commit on nil.
	// Instead, open a real in-memory DB for the test.
	return nil, nil
}

// realStubDB uses an in-memory SQLite for actual tx support.
type realStubDB struct {
	db *sql.DB
}

func (s *realStubDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}

// patchRouter builds a minimal chi router for PatchTransaction.
func patchRouter(h *WalletHandler) http.Handler {
	r := chi.NewRouter()
	r.Patch("/wallets/{walletID}/transactions/{txid}", h.PatchTransaction)
	return r
}

func ptr(s string) *string { return &s }

func TestPatchTransaction_NotFound(t *testing.T) {
	h := NewWalletHandler(
		&stubWalletStore{wallet: nil},
		&stubTxStore{err: sql.ErrNoRows},
		&stubTxLedger{},
		&stubTxJournal{},
		&stubDB{},
		nil, nil, nil,
	)
	body, _ := json.Marshal(map[string]string{"note": "hello"})
	req := httptest.NewRequest(http.MethodPatch, "/wallets/w1/transactions/abc", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	patchRouter(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestPatchTransaction_NoFields(t *testing.T) {
	tx := &domain.Transaction{ID: "t1", WalletID: "w1", Txid: "abc"}
	h := NewWalletHandler(
		&stubWalletStore{wallet: testWallet()},
		&stubTxStore{tx: tx},
		&stubTxLedger{},
		&stubTxJournal{},
		&stubDB{},
		nil, nil, nil,
	)
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPatch, "/wallets/w1/transactions/abc", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	patchRouter(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPatchTransaction_UpdatesLedgerAndJournal(t *testing.T) {
	// Use a real in-memory SQLite for transaction support.
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _dummy (id TEXT)`); err != nil {
		t.Fatal(err)
	}

	domainTx := &domain.Transaction{ID: "t1", WalletID: "w1", Txid: "abc"}
	entries := []*domain.LedgerEntry{
		{ID: "e1", WalletID: "w1", TransactionID: "t1", Category: domain.CategoryUnknown, Note: "", CreatedAt: time.Now()},
		{ID: "e2", WalletID: "w1", TransactionID: "t1", Category: domain.CategoryUnknown, Note: "", CreatedAt: time.Now()},
	}
	ledger := &stubTxLedger{entries: entries}
	journal := &stubTxJournal{}

	h := NewWalletHandler(
		&stubWalletStore{wallet: testWallet()},
		&stubTxStore{tx: domainTx},
		ledger,
		journal,
		&realStubDB{db: db},
		nil, nil, nil,
	)

	body, _ := json.Marshal(map[string]string{"category": "income", "note": "salary"})
	req := httptest.NewRequest(http.MethodPatch, "/wallets/w1/transactions/abc", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	patchRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Both entries updated.
	if len(ledger.updated) != 2 {
		t.Errorf("expected 2 ledger updates, got %d", len(ledger.updated))
	}
	// 2 fields × 2 entries = 4 journal entries.
	if len(journal.inserted) != 4 {
		t.Errorf("expected 4 journal entries, got %d", len(journal.inserted))
	}
	// Verify field names.
	fields := map[string]int{}
	for _, j := range journal.inserted {
		fields[j.FieldChanged]++
	}
	if fields["category"] != 2 || fields["note"] != 2 {
		t.Errorf("unexpected journal field counts: %v", fields)
	}
}

func TestPatchTransaction_NoChangeSkipsJournal(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS _dummy (id TEXT)`) //nolint:errcheck

	domainTx := &domain.Transaction{ID: "t1", WalletID: "w1", Txid: "abc"}
	// Entry already has category=income and note=salary.
	entries := []*domain.LedgerEntry{
		{ID: "e1", WalletID: "w1", Category: domain.CategoryIncome, Note: "salary", CreatedAt: time.Now()},
	}
	ledger := &stubTxLedger{entries: entries}
	journal := &stubTxJournal{}

	h := NewWalletHandler(
		&stubWalletStore{wallet: testWallet()},
		&stubTxStore{tx: domainTx},
		ledger, journal,
		&realStubDB{db: db},
		nil, nil, nil,
	)

	body, _ := json.Marshal(map[string]string{"category": "income", "note": "salary"})
	req := httptest.NewRequest(http.MethodPatch, "/wallets/w1/transactions/abc", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	patchRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(journal.inserted) != 0 {
		t.Errorf("expected no journal entries when nothing changed, got %d", len(journal.inserted))
	}
}
