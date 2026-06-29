package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// --- stub implementations ---

type stubWalletReader struct {
	wallet *domain.Wallet
	err    error
}

func (s *stubWalletReader) GetByID(_ context.Context, _ string) (*domain.Wallet, error) {
	return s.wallet, s.err
}

type stubLedgerReader struct {
	entries []*domain.LedgerEntry
	total   int
	single  *domain.LedgerEntry
	err     error
}

func (s *stubLedgerReader) ListByWallet(_ context.Context, _ string, _, _ int) ([]*domain.LedgerEntry, int, error) {
	return s.entries, s.total, s.err
}

func (s *stubLedgerReader) GetByID(_ context.Context, _ string) (*domain.LedgerEntry, error) {
	return s.single, s.err
}

type stubJournalReader struct {
	entries []*domain.JournalEntry
	err     error
}

func (s *stubJournalReader) ListByLedgerEntry(_ context.Context, _ string) ([]*domain.JournalEntry, error) {
	return s.entries, s.err
}

type stubUTXOReader struct {
	utxos []*domain.UTXO
	err   error
}

func (s *stubUTXOReader) ListByWallet(_ context.Context, _ string, unspentOnly bool) ([]*domain.UTXO, error) {
	if unspentOnly {
		var out []*domain.UTXO
		for _, u := range s.utxos {
			if !u.Spent {
				out = append(out, u)
			}
		}
		return out, s.err
	}
	return s.utxos, s.err
}

// --- helpers ---

func newLedgerRouter(lh *LedgerHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/wallets/{walletID}/ledger", lh.ListLedger)
	r.Get("/wallets/{walletID}/ledger/{entryID}", lh.GetLedgerEntry)
	r.Get("/wallets/{walletID}/utxos", lh.ListUTXOs)
	return r
}

func testWallet() *domain.Wallet {
	return &domain.Wallet{ID: "w1", Name: "test"}
}

// --- tests ---

func TestListLedger_Empty(t *testing.T) {
	lh := NewLedgerHandler(
		&stubWalletReader{wallet: testWallet()},
		&stubLedgerReader{entries: nil, total: 0},
		&stubJournalReader{},
		&stubUTXOReader{},
	)
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/ledger", nil)
	rec := httptest.NewRecorder()
	newLedgerRouter(lh).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	data := body["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data array, got %v", data)
	}
}

func TestListLedger_WalletNotFound(t *testing.T) {
	lh := NewLedgerHandler(
		&stubWalletReader{err: sql.ErrNoRows},
		&stubLedgerReader{},
		&stubJournalReader{},
		&stubUTXOReader{},
	)
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/ledger", nil)
	rec := httptest.NewRecorder()
	newLedgerRouter(lh).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetLedgerEntry_WithJournal(t *testing.T) {
	now := time.Now().UTC()
	entry := &domain.LedgerEntry{
		ID: "e1", WalletID: "w1", TransactionID: "t1",
		Type: domain.EntryTypeCredit, Sats: 5000, CreatedAt: now,
	}
	journal := []*domain.JournalEntry{
		{ID: "j1", LedgerEntryID: "e1", FieldChanged: "note", OldValue: "", NewValue: "hello", Reason: "edit", CreatedAt: now},
		{ID: "j2", LedgerEntryID: "e1", FieldChanged: "category", OldValue: "unknown", NewValue: "income", Reason: "fix", CreatedAt: now.Add(time.Second)},
	}
	lh := NewLedgerHandler(
		&stubWalletReader{wallet: testWallet()},
		&stubLedgerReader{single: entry},
		&stubJournalReader{entries: journal},
		&stubUTXOReader{},
	)
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/ledger/e1", nil)
	rec := httptest.NewRecorder()
	newLedgerRouter(lh).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	j := body["journal"].([]any)
	if len(j) != 2 {
		t.Errorf("expected 2 journal entries, got %d", len(j))
	}
}

func TestGetLedgerEntry_CrossWalletBlocked(t *testing.T) {
	entry := &domain.LedgerEntry{ID: "e1", WalletID: "other-wallet"}
	lh := NewLedgerHandler(
		&stubWalletReader{wallet: testWallet()},
		&stubLedgerReader{single: entry},
		&stubJournalReader{},
		&stubUTXOReader{},
	)
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/ledger/e1", nil)
	rec := httptest.NewRecorder()
	newLedgerRouter(lh).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-wallet access, got %d", rec.Code)
	}
}

func TestListUTXOs_All(t *testing.T) {
	utxos := []*domain.UTXO{
		{ID: "u1", WalletID: "w1", Txid: "tx1", Sats: 1000, Spent: false},
		{ID: "u2", WalletID: "w1", Txid: "tx2", Sats: 500, Spent: true},
	}
	lh := NewLedgerHandler(
		&stubWalletReader{wallet: testWallet()},
		&stubLedgerReader{},
		&stubJournalReader{},
		&stubUTXOReader{utxos: utxos},
	)
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/utxos", nil)
	rec := httptest.NewRecorder()
	newLedgerRouter(lh).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body []any
	json.NewDecoder(rec.Body).Decode(&body)
	if len(body) != 2 {
		t.Errorf("expected 2 UTXOs, got %d", len(body))
	}
}

func TestListUTXOs_UnspentFilter(t *testing.T) {
	utxos := []*domain.UTXO{
		{ID: "u1", WalletID: "w1", Txid: "tx1", Sats: 1000, Spent: false},
		{ID: "u2", WalletID: "w1", Txid: "tx2", Sats: 500, Spent: true},
	}
	lh := NewLedgerHandler(
		&stubWalletReader{wallet: testWallet()},
		&stubLedgerReader{},
		&stubJournalReader{},
		&stubUTXOReader{utxos: utxos},
	)
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/utxos?unspent=true", nil)
	rec := httptest.NewRecorder()
	newLedgerRouter(lh).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body []any
	json.NewDecoder(rec.Body).Decode(&body)
	if len(body) != 1 {
		t.Errorf("expected 1 unspent UTXO, got %d", len(body))
	}
}
