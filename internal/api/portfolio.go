package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
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

type portfolioPriceRepo interface {
	GetClosest(ctx context.Context, currency string, t time.Time) (*domain.PriceSnapshot, error)
	List(ctx context.Context, currency string, from, to time.Time) ([]*domain.PriceSnapshot, error)
}

type portfolioLedgerRepo interface {
	ListAllOrdered(ctx context.Context) ([]*domain.LedgerEntry, error)
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
	wallets   portfolioWalletLister
	cbRepo    portfolioCBRepo
	utxos     portfolioUTXORepo
	priceRepo portfolioPriceRepo
	ledger    portfolioLedgerRepo
}

func NewPortfolioHandler(wallets portfolioWalletLister, cbRepo portfolioCBRepo, utxos portfolioUTXORepo, priceRepo portfolioPriceRepo, ledger portfolioLedgerRepo) *PortfolioHandler {
	return &PortfolioHandler{wallets: wallets, cbRepo: cbRepo, utxos: utxos, priceRepo: priceRepo, ledger: ledger}
}

// currentPrice returns the latest known price for currency, or 0 if unknown.
func (h *PortfolioHandler) currentPrice(ctx context.Context, currency string, now time.Time) int64 {
	if currency == "" {
		return 0
	}
	snap, err := h.priceRepo.GetClosest(ctx, currency, now)
	if err != nil || snap == nil {
		return 0
	}
	return snap.PriceFiat
}

// satsToFiat converts satoshis to fiat cents given a BTC price in cents.
func satsToFiat(sats, pricePerBTCCents int64) int64 {
	if pricePerBTCCents == 0 {
		return 0
	}
	return sats * pricePerBTCCents / 100_000_000
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
		var price int64
		for _, rec := range records {
			if ws.Method == "" {
				ws.Method = rec.Method
				ws.FiatCurrency = rec.FiatCurrency
				price = h.currentPrice(ctx, ws.FiatCurrency, summary.ComputedAt)
			}
			if rec.DisposedAt != nil && rec.GainFiat != nil {
				ws.RealisedGainFiat += *rec.GainFiat
				continue
			}
			// Held (undisposed) lots have no GainFiat — only disposals set it.
			// Mark to market so "unrealised gain" isn't always zero, and only
			// count cost of coins still held (matches TotalSats).
			ws.TotalCostFiat += rec.CostFiat
			if price > 0 {
				ws.UnrealisedGainFiat += satsToFiat(rec.CostSats, price) - rec.CostFiat
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

// PortfolioHistoryPoint is one day's cumulative BTC balance and its fiat
// value at the closest known price, across all wallets.
type PortfolioHistoryPoint struct {
	Date      string `json:"date"` // YYYY-MM-DD (UTC)
	TotalSats int64  `json:"total_sats"`
	FiatValue int64  `json:"fiat_value"` // cents; 0 if no price known for that date
}

// GetPortfolioHistory handles GET /api/v1/portfolio/history?currency=EUR&days=180
// It reconstructs the cumulative sats balance across every wallet's ledger day
// by day, and marks each day to the closest known price snapshot.
func (h *PortfolioHandler) GetPortfolioHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = "EUR"
	}
	days := 365
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			days = n
		}
	}

	entries, err := h.ledger.ListAllOrdered(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(entries) == 0 {
		writeJSON(w, http.StatusOK, []PortfolioHistoryPoint{})
		return
	}

	today := truncateToDay(time.Now().UTC())
	start := truncateToDay(entries[0].CreatedAt)
	if earliest := today.AddDate(0, 0, -days+1); earliest.After(start) {
		start = earliest
	}

	prices, err := h.priceRepo.List(ctx, currency, start, today.AddDate(0, 0, 1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	sort.Slice(prices, func(i, j int) bool { return prices[i].Timestamp.Before(prices[j].Timestamp) })

	points := make([]PortfolioHistoryPoint, 0, days)
	var runningSats int64
	entryIdx := 0
	priceIdx := 0
	for day := start; !day.After(today); day = day.AddDate(0, 0, 1) {
		dayEnd := day.AddDate(0, 0, 1)
		for entryIdx < len(entries) && entries[entryIdx].CreatedAt.Before(dayEnd) {
			e := entries[entryIdx]
			if e.Type == domain.EntryTypeCredit {
				runningSats += e.Sats
			} else {
				runningSats -= e.Sats
			}
			entryIdx++
		}
		// Advance to the last price snapshot at or before this day; hold the
		// most recent known price forward across days with no snapshot.
		for priceIdx+1 < len(prices) && !prices[priceIdx+1].Timestamp.After(dayEnd) {
			priceIdx++
		}
		var fiatValue int64
		if len(prices) > 0 && !prices[priceIdx].Timestamp.After(dayEnd) {
			fiatValue = satsToFiat(runningSats, prices[priceIdx].PriceFiat)
		}
		points = append(points, PortfolioHistoryPoint{
			Date:      day.Format("2006-01-02"),
			TotalSats: runningSats,
			FiatValue: fiatValue,
		})
	}

	writeJSON(w, http.StatusOK, points)
}
