package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/accounting"
	"github.com/storagebirddrop/abacus/internal/domain"
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

// AccountingHandler handles all Phase 3 accounting and price endpoints.
type AccountingHandler struct {
	svc       accountingSvc
	priceRepo priceSnapRepo
	cbRepo    costBasisRepo
}

func NewAccountingHandler(svc accountingSvc, priceRepo priceSnapRepo, cbRepo costBasisRepo) *AccountingHandler {
	return &AccountingHandler{svc: svc, priceRepo: priceRepo, cbRepo: cbRepo}
}

// RunAccounting handles POST /wallets/{walletID}/accounting/run
func (h *AccountingHandler) RunAccounting(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")

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
	case domain.MethodFIFO, domain.MethodAvgCost, domain.MethodLIFO, domain.MethodHIFO, domain.MethodSpecificID:
		// valid
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "method must be fifo, avgcost, lifo, hifo, or specificid"})
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
