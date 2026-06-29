package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/reports"
)

type reportWalletRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
}

type reportTxRepo interface {
	List(ctx context.Context, walletID string, limit, offset int) ([]*domain.Transaction, int, error)
}

type reportUTXORepo interface {
	ListByWallet(ctx context.Context, walletID string, unspentOnly bool) ([]*domain.UTXO, error)
}

type reportCBRepo interface {
	ListByWallet(ctx context.Context, walletID string) ([]*domain.CostBasisRecord, error)
}

// ReportHandler serves CSV / PDF / Excel reports.
type ReportHandler struct {
	walletRepo reportWalletRepo
	txRepo     reportTxRepo
	utxoRepo   reportUTXORepo
	cbRepo     reportCBRepo
}

func NewReportHandler(
	walletRepo reportWalletRepo,
	txRepo reportTxRepo,
	utxoRepo reportUTXORepo,
	cbRepo reportCBRepo,
) *ReportHandler {
	return &ReportHandler{walletRepo: walletRepo, txRepo: txRepo, utxoRepo: utxoRepo, cbRepo: cbRepo}
}

func (h *ReportHandler) walletName(ctx context.Context, id string) string {
	w, err := h.walletRepo.GetByID(ctx, id)
	if err != nil || w == nil {
		return id
	}
	return w.Name
}

// Transactions handles GET /wallets/{walletID}/reports/transactions
func (h *ReportHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	format := reportFormat(r)
	currency := reportCurrency(r)

	// Load all transactions (batch 500)
	var rows []reports.TransactionRow
	offset := 0
	for {
		txs, _, err := h.txRepo.List(r.Context(), walletID, 500, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		for _, tx := range txs {
			rows = append(rows, reports.TransactionRow{
				Date:      tx.BlockTime,
				Txid:      tx.Txid,
				FeeSats:   tx.FeeSats,
				Confirmed: tx.Confirmed,
			})
		}
		if len(txs) < 500 {
			break
		}
		offset += 500
	}

	name := h.walletName(r.Context(), walletID)
	_ = currency

	var buf bytes.Buffer
	var ext, mime string

	switch format {
	case "pdf":
		ext, mime = "pdf", "application/pdf"
		if err := reports.WriteTransactionsPDF(&buf, rows, name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	case "xlsx":
		ext, mime = "xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		if err := reports.WriteTransactionsExcel(&buf, rows, name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	default:
		ext, mime = "csv", "text/csv"
		if err := reports.WriteTransactionsCSV(&buf, rows); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	filename := fmt.Sprintf("transactions-%s.%s", time.Now().Format("2006-01-02"), ext)
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Del("Content-Type") // set after to avoid double-write
	w.Header().Set("Content-Type", mime)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// PnL handles GET /wallets/{walletID}/reports/pnl
func (h *ReportHandler) PnL(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	format := reportFormat(r)
	currency := reportCurrency(r)

	cbs, err := h.cbRepo.ListByWallet(r.Context(), walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Filter to disposed records only and apply date range
	from, to := parseTimeRange(r)
	var rows []reports.PnLRow
	for _, cb := range cbs {
		if cb.DisposedAt == nil || cb.ProceedsFiat == nil || cb.GainFiat == nil {
			continue
		}
		if cb.DisposedAt.Before(from) || cb.DisposedAt.After(to) {
			continue
		}
		rows = append(rows, reports.PnLRow{
			Txid:         cb.Txid,
			Vout:         cb.Vout,
			AcquiredAt:   cb.AcquiredAt,
			DisposedAt:   *cb.DisposedAt,
			CostSats:     cb.CostSats,
			CostFiat:     cb.CostFiat,
			ProceedsFiat: *cb.ProceedsFiat,
			GainFiat:     *cb.GainFiat,
			Method:       string(cb.Method),
			Currency:     cb.FiatCurrency,
		})
	}

	if len(rows) > 0 {
		currency = rows[0].Currency
	}

	name := h.walletName(r.Context(), walletID)

	var buf bytes.Buffer
	var ext, mime string

	switch format {
	case "pdf":
		ext, mime = "pdf", "application/pdf"
		if err := reports.WritePnLPDF(&buf, rows, currency, name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	case "xlsx":
		ext, mime = "xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		if err := reports.WritePnLExcel(&buf, rows, currency, name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	default:
		ext, mime = "csv", "text/csv"
		if err := reports.WritePnLCSV(&buf, rows, currency); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	filename := fmt.Sprintf("pnl-%s.%s", time.Now().Format("2006-01-02"), ext)
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// BalanceSheet handles GET /wallets/{walletID}/reports/balance-sheet
func (h *ReportHandler) BalanceSheet(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	format := reportFormat(r)
	currency := reportCurrency(r)

	utxos, err := h.utxoRepo.ListByWallet(r.Context(), walletID, true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Build cost basis map from accounting run results
	cbs, _ := h.cbRepo.ListByWallet(r.Context(), walletID)
	cbMap := map[string]int64{} // "txid:vout" → cost_fiat cents
	cbCurrency := currency
	for _, cb := range cbs {
		cbMap[fmt.Sprintf("%s:%d", cb.Txid, cb.Vout)] = cb.CostFiat
		if cb.FiatCurrency != "" {
			cbCurrency = cb.FiatCurrency
		}
	}
	currency = cbCurrency

	rows := make([]reports.BalanceRow, 0, len(utxos))
	for _, u := range utxos {
		key := fmt.Sprintf("%s:%d", u.Txid, u.Vout)
		rows = append(rows, reports.BalanceRow{
			Txid:       u.Txid,
			Vout:       u.Vout,
			Sats:       u.Sats,
			Address:    u.Address,
			AcquiredAt: u.BlockTime,
			CostFiat:   cbMap[key],
			Currency:   currency,
		})
	}

	name := h.walletName(r.Context(), walletID)

	var buf bytes.Buffer
	var ext, mime string

	switch format {
	case "pdf":
		ext, mime = "pdf", "application/pdf"
		if err := reports.WriteBalanceSheetPDF(&buf, rows, currency, name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	case "xlsx":
		ext, mime = "xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		if err := reports.WriteBalanceSheetExcel(&buf, rows, currency, name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	default:
		ext, mime = "csv", "text/csv"
		if err := reports.WriteBalanceSheetCSV(&buf, rows, currency); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	filename := fmt.Sprintf("balance-sheet-%s.%s", time.Now().Format("2006-01-02"), ext)
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func reportFormat(r *http.Request) string {
	f := strings.ToLower(r.URL.Query().Get("format"))
	switch f {
	case "pdf", "xlsx":
		return f
	default:
		return "csv"
	}
}

func reportCurrency(r *http.Request) string {
	c := r.URL.Query().Get("currency")
	if c == "" {
		return "EUR"
	}
	return strings.ToUpper(c)
}
