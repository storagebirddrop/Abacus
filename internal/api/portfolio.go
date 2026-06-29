package api

import (
	"context"
	"net/http"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
)

type portfolioWalletLister interface {
	List(ctx context.Context) ([]*domain.Wallet, error)
}

type portfolioCBRepo interface {
	ListByWallet(ctx context.Context, walletID string) ([]*domain.CostBasisRecord, error)
}

type portfolioUTXORepo interface {
	ListByWallet(ctx context.Context, walletID string, unspentOnly bool) ([]*domain.UTXO, error)
}

// WalletSummary is one wallet's contribution to the portfolio.
type WalletSummary struct {
	WalletID           string                 `json:"wallet_id"`
	WalletName         string                 `json:"wallet_name"`
	Method             domain.CostBasisMethod `json:"method,omitempty"`
	FiatCurrency       string                 `json:"fiat_currency,omitempty"`
	TotalSats          int64                  `json:"total_sats"`
	TotalCostFiat      int64                  `json:"total_cost_fiat"`
	UnrealisedGainFiat int64                  `json:"unrealised_gain_fiat"`
	RealisedGainFiat   int64                  `json:"realised_gain_fiat"`
}

// PortfolioSummary is the cross-wallet aggregate.
type PortfolioSummary struct {
	WalletCount        int             `json:"wallet_count"`
	TotalSats          int64           `json:"total_sats"`
	TotalCostFiat      int64           `json:"total_cost_fiat"`
	UnrealisedGainFiat int64           `json:"unrealised_gain_fiat"`
	RealisedGainFiat   int64           `json:"realised_gain_fiat"`
	Wallets            []WalletSummary `json:"wallets"`
	ComputedAt         time.Time       `json:"computed_at"`
}

// PortfolioHandler handles cross-wallet portfolio endpoints.
type PortfolioHandler struct {
	wallets portfolioWalletLister
	cbRepo  portfolioCBRepo
	utxos   portfolioUTXORepo
}

func NewPortfolioHandler(wallets portfolioWalletLister, cbRepo portfolioCBRepo, utxos portfolioUTXORepo) *PortfolioHandler {
	return &PortfolioHandler{wallets: wallets, cbRepo: cbRepo, utxos: utxos}
}

// GetPortfolioSummary handles GET /api/v1/portfolio/summary
func (h *PortfolioHandler) GetPortfolioSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wallets, err := h.wallets.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	summary := PortfolioSummary{
		WalletCount: len(wallets),
		Wallets:     make([]WalletSummary, 0, len(wallets)),
		ComputedAt:  time.Now().UTC(),
	}

	for _, wallet := range wallets {
		ws := WalletSummary{
			WalletID:   wallet.ID,
			WalletName: wallet.Name,
		}

		// Unspent sats from UTXOs (ground truth for holdings).
		utxos, err := h.utxos.ListByWallet(ctx, wallet.ID, true)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		for _, u := range utxos {
			ws.TotalSats += u.Sats
		}

		// Cost basis and gains from accounting records (if a run has been done).
		records, err := h.cbRepo.ListByWallet(ctx, wallet.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		for _, rec := range records {
			if ws.Method == "" {
				ws.Method = rec.Method
				ws.FiatCurrency = rec.FiatCurrency
			}
			ws.TotalCostFiat += rec.CostFiat
			if rec.DisposedAt != nil && rec.GainFiat != nil {
				ws.RealisedGainFiat += *rec.GainFiat
			} else if rec.GainFiat != nil {
				ws.UnrealisedGainFiat += *rec.GainFiat
			}
		}

		summary.Wallets = append(summary.Wallets, ws)
		summary.TotalSats += ws.TotalSats
		summary.TotalCostFiat += ws.TotalCostFiat
		summary.UnrealisedGainFiat += ws.UnrealisedGainFiat
		summary.RealisedGainFiat += ws.RealisedGainFiat
	}

	writeJSON(w, http.StatusOK, summary)
}
