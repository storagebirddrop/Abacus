package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/accounting"
	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/prices"
)

type accountingSvc interface {
	Run(ctx context.Context, walletID string, method domain.CostBasisMethod, currency string) error
	Summary(ctx context.Context, walletID string) (*accounting.AccountingSummary, error)
}

type priceSnapRepo interface {
	Insert(ctx context.Context, p *domain.PriceSnapshot) error
	List(ctx context.Context, currency string, from, to time.Time) ([]*domain.PriceSnapshot, error)
}

type costBasisRepo interface {
	ListByWallet(ctx context.Context, walletID string) ([]*domain.CostBasisRecord, error)
}

type accountingWalletRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
}

type txBlockTimeRepo interface {
	ListBlockTimes(ctx context.Context, walletID string) ([]time.Time, error)
}

// AccountingHandler handles all Phase 3 accounting and price endpoints.
type AccountingHandler struct {
	svc        accountingSvc
	priceRepo  priceSnapRepo
	cbRepo     costBasisRepo
	walletRepo accountingWalletRepo
	txRepo     txBlockTimeRepo
}

func NewAccountingHandler(svc accountingSvc, priceRepo priceSnapRepo, cbRepo costBasisRepo, walletRepo accountingWalletRepo, txRepo txBlockTimeRepo) *AccountingHandler {
	return &AccountingHandler{svc: svc, priceRepo: priceRepo, cbRepo: cbRepo, walletRepo: walletRepo, txRepo: txRepo}
}

func (h *AccountingHandler) requireWallet(w http.ResponseWriter, r *http.Request, walletID string) bool {
	_, err := h.walletRepo.GetByID(r.Context(), walletID)
	if err == nil {
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "wallet not found"})
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return false
}

// RunAccounting handles POST /wallets/{walletID}/accounting/run
func (h *AccountingHandler) RunAccounting(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	if !h.requireWallet(w, r, walletID) {
		return
	}

	var req struct {
		Method   string `json:"method"`   // "fifo" | "avgcost"
		Currency string `json:"currency"` // e.g. "EUR"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Method == "" {
		req.Method = "fifo"
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	method := domain.CostBasisMethod(req.Method)
	switch method {
	case domain.MethodFIFO, domain.MethodAvgCost, domain.MethodLIFO, domain.MethodHIFO, domain.MethodSpecificID, domain.MethodSection104:
		// valid
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "method must be fifo, avgcost, lifo, hifo, specificid, or section104"})
		return
	}

	if err := h.svc.Run(r.Context(), walletID, method, req.Currency); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	sum, err := h.svc.Summary(r.Context(), walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// GetSummary handles GET /wallets/{walletID}/accounting/summary
func (h *AccountingHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	if !h.requireWallet(w, r, walletID) {
		return
	}
	sum, err := h.svc.Summary(r.Context(), walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// ListCostBasis handles GET /wallets/{walletID}/accounting/cost-basis
func (h *AccountingHandler) ListCostBasis(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	if !h.requireWallet(w, r, walletID) {
		return
	}
	records, err := h.cbRepo.ListByWallet(r.Context(), walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if records == nil {
		records = []*domain.CostBasisRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}

// ListPrices handles GET /prices?currency=EUR&from=2024-01-01&to=2024-12-31
func (h *AccountingHandler) ListPrices(w http.ResponseWriter, r *http.Request) {
	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = "EUR"
	}
	from, to := parseTimeRange(r)
	snaps, err := h.priceRepo.List(r.Context(), currency, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if snaps == nil {
		snaps = []*domain.PriceSnapshot{}
	}
	writeJSON(w, http.StatusOK, snaps)
}

// CreatePrice handles POST /prices
func (h *AccountingHandler) CreatePrice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Currency  string `json:"currency"`
		PriceFiat int64  `json:"price_fiat"` // cents per BTC
		Source    string `json:"source"`
		Timestamp int64  `json:"timestamp"` // unix epoch
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Currency == "" || req.PriceFiat <= 0 || req.Timestamp == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "currency, price_fiat, and timestamp are required"})
		return
	}
	if req.Source == "" {
		req.Source = "manual"
	}
	snap := &domain.PriceSnapshot{
		Currency:  req.Currency,
		PriceFiat: req.PriceFiat,
		Source:    req.Source,
		Timestamp: time.Unix(req.Timestamp, 0).UTC(),
	}
	if err := h.priceRepo.Insert(r.Context(), snap); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

// FetchPrices handles POST /prices/fetch.
// It reads all confirmed transaction dates for the requested wallet, fetches
// missing BTC/fiat prices from CoinGecko, and stores them with source="coingecko".
// Dates that already have any price entry (manual or automated) are skipped so
// that manual overrides are never clobbered.
func (h *AccountingHandler) FetchPrices(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WalletID string `json:"wallet_id"`
		Currency string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.WalletID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "wallet_id is required"})
		return
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	if !h.requireWallet(w, r, req.WalletID) {
		return
	}

	// All confirmed block times for this wallet.
	blockTimes, err := h.txRepo.ListBlockTimes(r.Context(), req.WalletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(blockTimes) == 0 {
		writeJSON(w, http.StatusOK, map[string]int{"fetched": 0, "skipped": 0})
		return
	}

	// Collect unique UTC calendar dates from transaction times.
	dateSet := make(map[time.Time]struct{}, len(blockTimes))
	for _, t := range blockTimes {
		day := truncateToDay(t)
		dateSet[day] = struct{}{}
	}

	// Date range for existing-price lookup and CoinGecko request.
	minDay := truncateToDay(blockTimes[0])
	maxDay := truncateToDay(blockTimes[len(blockTimes)-1]).Add(48 * time.Hour)

	// Existing prices — we skip any date that already has at least one entry.
	existing, err := h.priceRepo.List(r.Context(), req.Currency, minDay, maxDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	coveredDays := make(map[time.Time]struct{}, len(existing))
	for _, p := range existing {
		coveredDays[truncateToDay(p.Timestamp)] = struct{}{}
	}

	// Which dates are still missing?
	var missing []time.Time
	for day := range dateSet {
		if _, ok := coveredDays[day]; !ok {
			missing = append(missing, day)
		}
	}
	skipped := len(dateSet) - len(missing)
	if len(missing) == 0 {
		writeJSON(w, http.StatusOK, map[string]int{"fetched": 0, "skipped": skipped})
		return
	}

	// Single CoinGecko call for the full range.
	cgPrices, err := prices.FetchRange(r.Context(), req.Currency, minDay, maxDay)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	// Insert a price snapshot for each missing date.
	fetched := 0
	for _, day := range missing {
		priceCents, ok := cgPrices[day]
		if !ok || priceCents <= 0 {
			// CoinGecko may not have data for this exact date (e.g. very old or future).
			continue
		}
		snap := &domain.PriceSnapshot{
			Currency:  req.Currency,
			PriceFiat: priceCents,
			Source:    "coingecko",
			Timestamp: day,
		}
		if err := h.priceRepo.Insert(r.Context(), snap); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		fetched++
	}

	writeJSON(w, http.StatusOK, map[string]int{"fetched": fetched, "skipped": skipped})
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func parseTimeRange(r *http.Request) (from, to time.Time) {
	from = time.Unix(0, 0).UTC()
	to = time.Now().UTC().Add(24 * time.Hour)

	if s := r.URL.Query().Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = t.UTC()
		}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			to = t.UTC().Add(24 * time.Hour)
		}
	}
	return from, to
}
