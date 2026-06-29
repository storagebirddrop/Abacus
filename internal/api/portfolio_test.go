package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// --- stubs for portfolio handler ---

type stubPortfolioWallets struct {
	wallets []*domain.Wallet
}

func (s *stubPortfolioWallets) List(_ context.Context) ([]*domain.Wallet, error) {
	return s.wallets, nil
}

type stubPortfolioCB struct {
	records map[string][]*domain.CostBasisRecord
}

func (s *stubPortfolioCB) ListByWallet(_ context.Context, walletID string) ([]*domain.CostBasisRecord, error) {
	return s.records[walletID], nil
}

type stubPortfolioUTXOs struct {
	utxos map[string][]*domain.UTXO
}

func (s *stubPortfolioUTXOs) ListByWallet(_ context.Context, walletID string, _ bool) ([]*domain.UTXO, error) {
	return s.utxos[walletID], nil
}

func portfolioRouter(h *PortfolioHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/portfolio/summary", h.GetPortfolioSummary)
	return r
}

func TestPortfolioSummary_Empty(t *testing.T) {
	h := NewPortfolioHandler(
		&stubPortfolioWallets{wallets: []*domain.Wallet{}},
		&stubPortfolioCB{records: map[string][]*domain.CostBasisRecord{}},
		&stubPortfolioUTXOs{utxos: map[string][]*domain.UTXO{}},
	)
	req := httptest.NewRequest(http.MethodGet, "/portfolio/summary", nil)
	rec := httptest.NewRecorder()
	portfolioRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body PortfolioSummary
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.WalletCount != 0 {
		t.Errorf("expected 0 wallets, got %d", body.WalletCount)
	}
	if body.TotalSats != 0 {
		t.Errorf("expected 0 total sats, got %d", body.TotalSats)
	}
}

func TestPortfolioSummary_TwoWallets(t *testing.T) {
	w1 := &domain.Wallet{ID: "w1", Name: "Hot Wallet", Network: domain.NetworkMainnet}
	w2 := &domain.Wallet{ID: "w2", Name: "Cold Storage", Network: domain.NetworkMainnet}

	gain1 := int64(10_000)  // 100 EUR unrealised
	gain2 := int64(5_000)   // 50 EUR realised
	disposed := time.Now()

	h := NewPortfolioHandler(
		&stubPortfolioWallets{wallets: []*domain.Wallet{w1, w2}},
		&stubPortfolioCB{records: map[string][]*domain.CostBasisRecord{
			"w1": {
				{ID: "cb1", WalletID: "w1", CostFiat: 20_000, FiatCurrency: "EUR", Method: domain.MethodFIFO, GainFiat: &gain1},
			},
			"w2": {
				{ID: "cb2", WalletID: "w2", CostFiat: 30_000, FiatCurrency: "EUR", Method: domain.MethodFIFO, DisposedAt: &disposed, ProceedsFiat: &gain2, GainFiat: &gain2},
			},
		}},
		&stubPortfolioUTXOs{utxos: map[string][]*domain.UTXO{
			"w1": {{WalletID: "w1", Sats: 1_000_000}},
			"w2": {{WalletID: "w2", Sats: 500_000}},
		}},
	)

	req := httptest.NewRequest(http.MethodGet, "/portfolio/summary", nil)
	rec := httptest.NewRecorder()
	portfolioRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body PortfolioSummary
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.WalletCount != 2 {
		t.Errorf("expected 2 wallets, got %d", body.WalletCount)
	}
	if body.TotalSats != 1_500_000 {
		t.Errorf("TotalSats = %d, want 1500000", body.TotalSats)
	}
	if body.TotalCostFiat != 50_000 {
		t.Errorf("TotalCostFiat = %d, want 50000", body.TotalCostFiat)
	}
	if body.UnrealisedGainFiat != 10_000 {
		t.Errorf("UnrealisedGainFiat = %d, want 10000", body.UnrealisedGainFiat)
	}
	if body.RealisedGainFiat != 5_000 {
		t.Errorf("RealisedGainFiat = %d, want 5000", body.RealisedGainFiat)
	}
	if len(body.Wallets) != 2 {
		t.Errorf("expected 2 wallet entries, got %d", len(body.Wallets))
	}
}

func TestPortfolioSummary_NoAccountingRun(t *testing.T) {
	// Wallet with UTXOs but no cost basis records yet.
	w1 := &domain.Wallet{ID: "w1", Name: "Wallet", Network: domain.NetworkMainnet}
	h := NewPortfolioHandler(
		&stubPortfolioWallets{wallets: []*domain.Wallet{w1}},
		&stubPortfolioCB{records: map[string][]*domain.CostBasisRecord{"w1": nil}},
		&stubPortfolioUTXOs{utxos: map[string][]*domain.UTXO{
			"w1": {{WalletID: "w1", Sats: 2_000_000}},
		}},
	)
	req := httptest.NewRequest(http.MethodGet, "/portfolio/summary", nil)
	rec := httptest.NewRecorder()
	portfolioRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body PortfolioSummary
	json.NewDecoder(rec.Body).Decode(&body) //nolint:errcheck
	if body.TotalSats != 2_000_000 {
		t.Errorf("TotalSats = %d, want 2000000", body.TotalSats)
	}
	if body.TotalCostFiat != 0 {
		t.Errorf("TotalCostFiat should be 0 before accounting run, got %d", body.TotalCostFiat)
	}
}
