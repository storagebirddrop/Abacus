package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/accounting"
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

type reportPriceRepo interface {
	GetClosest(ctx context.Context, currency string, t time.Time) (*domain.PriceSnapshot, error)
}

// ReportHandler serves CSV / PDF / Excel reports.
type ReportHandler struct {
	walletRepo reportWalletRepo
	txRepo     reportTxRepo
	utxoRepo   reportUTXORepo
	cbRepo     reportCBRepo
	priceRepo  reportPriceRepo
}

func NewReportHandler(
	walletRepo reportWalletRepo,
	txRepo reportTxRepo,
	utxoRepo reportUTXORepo,
	cbRepo reportCBRepo,
	priceRepo reportPriceRepo,
) *ReportHandler {
	return &ReportHandler{
		walletRepo: walletRepo,
		txRepo:     txRepo,
		utxoRepo:   utxoRepo,
		cbRepo:     cbRepo,
		priceRepo:  priceRepo,
	}
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

// TaxReport handles GET /wallets/{walletID}/reports/tax
func (h *ReportHandler) TaxReport(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	jurisdiction := strings.ToLower(r.URL.Query().Get("jurisdiction"))
	format := reportFormat(r)
	year := parseYear(r)

	switch jurisdiction {
	case "nl":
		h.taxNL(w, r, walletID, year, format)
	case "de":
		h.taxDE(w, r, walletID, year, format)
	case "uk":
		h.taxUK(w, r, walletID, year, format)
	case "us":
		h.taxUS(w, r, walletID, year, format)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "jurisdiction must be nl, de, uk, or us"})
	}
}

// taxNL generates the Netherlands Box 3 tax report.
func (h *ReportHandler) taxNL(w http.ResponseWriter, r *http.Request, walletID string, year int, format string) {
	ctx := r.Context()
	peildatum := reports.NLPeildatum(year)

	cbs, err := h.cbRepo.ListByWallet(ctx, walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Collect UTXOs held on 1 januari: acquired_at < peildatum AND (disposed_at IS NULL OR disposed_at > peildatum).
	var holdings []reports.NLHoldingRow
	var totalSats int64
	for _, cb := range cbs {
		if cb.AcquiredAt.After(peildatum) || cb.AcquiredAt.Equal(peildatum) {
			continue
		}
		if cb.DisposedAt != nil && !cb.DisposedAt.After(peildatum) {
			continue
		}
		holdings = append(holdings, reports.NLHoldingRow{Txid: cb.Txid, Vout: cb.Vout, Sats: cb.CostSats})
		totalSats += cb.CostSats
	}

	// Fetch BTC/EUR price on 1 januari.
	var priceEUR int64
	if h.priceRepo != nil {
		snap, _ := h.priceRepo.GetClosest(ctx, "EUR", peildatum)
		if snap != nil {
			priceEUR = snap.PriceFiat
		}
	}

	data := reports.NLTaxData{
		Year:         year,
		PeildatumBTC: totalSats,
		PriceEUR:     priceEUR,
		ValueEUR:     totalSats * priceEUR / 100_000_000,
		Holdings:     holdings,
	}

	var buf bytes.Buffer
	var ext, mime string
	switch format {
	case "pdf":
		ext, mime = "pdf", "application/pdf"
		err = reports.WriteTaxNLPDF(&buf, data)
	default:
		ext, mime = "csv", "text/csv"
		err = reports.WriteTaxNLCSV(&buf, data)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	serveFile(w, fmt.Sprintf("tax-nl-%d.%s", year, ext), mime, buf.Bytes())
}

// taxDE generates the Germany §23 EStG tax report.
func (h *ReportHandler) taxDE(w http.ResponseWriter, r *http.Request, walletID string, year int, format string) {
	ctx := r.Context()
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	cbs, err := h.cbRepo.ListByWallet(ctx, walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var rows []reports.DETaxRow
	for _, cb := range cbs {
		if cb.DisposedAt == nil || cb.ProceedsFiat == nil || cb.GainFiat == nil {
			continue
		}
		if cb.DisposedAt.Before(from) || cb.DisposedAt.After(to) {
			continue
		}
		holdingDays := int(cb.DisposedAt.Sub(cb.AcquiredAt).Hours() / 24)
		rows = append(rows, reports.DETaxRow{
			Txid:         cb.Txid,
			Vout:         cb.Vout,
			AcquiredAt:   cb.AcquiredAt,
			DisposedAt:   *cb.DisposedAt,
			HoldingDays:  holdingDays,
			CostFiat:     cb.CostFiat,
			ProceedsFiat: *cb.ProceedsFiat,
			GainFiat:     *cb.GainFiat,
			TaxFree:      holdingDays >= 365,
		})
	}

	summary := reports.BuildDESummary(year, rows)

	var buf bytes.Buffer
	var ext, mime string
	switch format {
	case "pdf":
		ext, mime = "pdf", "application/pdf"
		err = reports.WriteTaxDEPDF(&buf, summary)
	default:
		ext, mime = "csv", "text/csv"
		err = reports.WriteTaxDECSV(&buf, summary)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	serveFile(w, fmt.Sprintf("tax-de-%d.%s", year, ext), mime, buf.Bytes())
}

// taxUK generates the UK HMRC CGT Section 104 report.
func (h *ReportHandler) taxUK(w http.ResponseWriter, r *http.Request, walletID string, year int, format string) {
	ctx := r.Context()
	taxYearStr, startYear := reports.UKTaxYear(year)
	tyStart, tyEnd := reports.UKTaxYearBounds(startYear)

	// Load all UTXOs (spent + unspent) for Section 104 computation.
	allUTXOs, err := h.utxoRepo.ListByWallet(ctx, walletID, false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Build spend-time map from transactions.
	spendTimes := map[string]time.Time{}
	offset := 0
	for {
		txs, _, err := h.txRepo.List(ctx, walletID, 500, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		for _, tx := range txs {
			spendTimes[tx.Txid] = tx.BlockTime
		}
		if len(txs) < 500 {
			break
		}
		offset += 500
	}

	// Price lookup in GBP.
	priceFn := func(currency string, t time.Time) int64 {
		if t.IsZero() || h.priceRepo == nil {
			return 0
		}
		snap, err := h.priceRepo.GetClosest(ctx, currency, t)
		if err != nil || snap == nil {
			return 0
		}
		return snap.PriceFiat
	}

	utxos := make([]domain.UTXO, len(allUTXOs))
	for i, u := range allUTXOs {
		utxos[i] = *u
	}

	// Run Section 104 in-memory (GBP).
	cbRecords := accounting.RunSection104(walletID, utxos, spendTimes, priceFn, "GBP")

	// Build pool movement log and disposals from the computed records.
	// The pool log is derived from the records: acquisitions add to pool, disposals draw from it.
	var poolRows []reports.UKPoolRow
	var disposals []reports.UKDisposalRow
	var poolSats, poolCost int64
	var totalGains, totalLosses int64

	for _, cb := range cbRecords {
		if cb.DisposedAt == nil {
			// Acquisition: add to pool if within or before tax year end.
			if cb.AcquiredAt.Before(tyEnd) {
				poolSats += cb.CostSats
				poolCost += cb.CostFiat
				poolRows = append(poolRows, reports.UKPoolRow{
					Date:        cb.AcquiredAt,
					Event:       "Acquisition",
					Sats:        cb.CostSats,
					CostGBP:     cb.CostFiat,
					PoolSats:    poolSats,
					PoolCostGBP: poolCost,
				})
			}
		} else if cb.DisposedAt.After(tyStart) && !cb.DisposedAt.After(tyEnd) &&
			cb.GainFiat != nil && cb.ProceedsFiat != nil {
			// Disposal in this tax year.
			gainGBP := *cb.GainFiat
			if gainGBP > 0 {
				totalGains += gainGBP
			} else {
				totalLosses += gainGBP
			}
			poolSats -= cb.CostSats
			if poolSats < 0 {
				poolSats = 0
			}
			poolCost -= cb.CostFiat
			if poolCost < 0 {
				poolCost = 0
			}
			poolRows = append(poolRows, reports.UKPoolRow{
				Date:        *cb.DisposedAt,
				Event:       "Disposal (pool)",
				Sats:        cb.CostSats,
				CostGBP:     cb.CostFiat,
				PoolSats:    poolSats,
				PoolCostGBP: poolCost,
			})
			disposals = append(disposals, reports.UKDisposalRow{
				Date:             *cb.DisposedAt,
				Txid:             cb.Txid,
				Vout:             cb.Vout,
				ProceedsGBP:      *cb.ProceedsFiat,
				AllowableCostGBP: cb.CostFiat,
				GainGBP:          gainGBP,
				MatchingRule:     "pool",
			})
		}
	}

	// Annual exempt amount: £3,000 from 2024/25 onward; £6,000 for 2023/24; £12,300 before.
	annualExempt := int64(300_000) // £3,000 in cents; default post-2024
	switch {
	case year < 2023:
		annualExempt = 1_230_000 // £12,300
	case year == 2023:
		annualExempt = 600_000 // £6,000
	}

	netGain := totalGains + totalLosses
	data := reports.UKTaxData{
		TaxYear:         taxYearStr,
		YearStart:       startYear,
		PoolRows:        poolRows,
		Disposals:       disposals,
		TotalGainsGBP:   totalGains,
		TotalLossesGBP:  totalLosses,
		NetGainGBP:      netGain,
		AnnualExemptGBP: annualExempt,
	}

	var buf bytes.Buffer
	var ext, mime string
	switch format {
	case "pdf":
		ext, mime = "pdf", "application/pdf"
		err = reports.WriteTaxUKPDF(&buf, data)
	default:
		ext, mime = "csv", "text/csv"
		err = reports.WriteTaxUKCSV(&buf, data)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	serveFile(w, fmt.Sprintf("tax-uk-%d-%d.%s", year, year+1, ext), mime, buf.Bytes())
}

// taxUS generates the US IRS Form 8949 report.
func (h *ReportHandler) taxUS(w http.ResponseWriter, r *http.Request, walletID string, year int, format string) {
	ctx := r.Context()
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	cbs, err := h.cbRepo.ListByWallet(ctx, walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	currency := "USD"
	var allRows []reports.USTaxRow
	for _, cb := range cbs {
		if cb.DisposedAt == nil || cb.ProceedsFiat == nil || cb.GainFiat == nil {
			continue
		}
		if cb.DisposedAt.Before(from) || cb.DisposedAt.After(to) {
			continue
		}
		if cb.FiatCurrency != "" {
			currency = cb.FiatCurrency
		}
		holdingDays, longTerm := reports.UsHoldingDays(cb.AcquiredAt, *cb.DisposedAt)
		allRows = append(allRows, reports.USTaxRow{
			Description: fmt.Sprintf("BTC %s:%d", cb.Txid, cb.Vout),
			AcquiredAt:  cb.AcquiredAt,
			DisposedAt:  *cb.DisposedAt,
			HoldingDays: holdingDays,
			ProceedsUSD: *cb.ProceedsFiat,
			CostUSD:     cb.CostFiat,
			GainUSD:     *cb.GainFiat,
			LongTerm:    longTerm,
		})
	}

	shortTerm, longTerm, stNet, ltNet := reports.BuildUSRows(allRows)
	data := reports.USTaxData{
		Year:         year,
		Currency:     currency,
		ShortTerm:    shortTerm,
		LongTerm:     longTerm,
		STNetGain:    stNet,
		LTNetGain:    ltNet,
		TotalNetGain: stNet + ltNet,
	}

	var buf bytes.Buffer
	var ext, mime string
	switch format {
	case "pdf":
		ext, mime = "pdf", "application/pdf"
		err = reports.WriteTaxUSPDF(&buf, data)
	default:
		ext, mime = "csv", "text/csv"
		err = reports.WriteTaxUSCSV(&buf, data)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	serveFile(w, fmt.Sprintf("tax-us-%d.%s", year, ext), mime, buf.Bytes())
}

func serveFile(w http.ResponseWriter, filename, mime string, data []byte) {
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = w.Write(data)
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

func parseYear(r *http.Request) int {
	if s := r.URL.Query().Get("year"); s != "" {
		if y, err := strconv.Atoi(s); err == nil && y >= 2009 && y <= 2100 {
			return y
		}
	}
	return time.Now().UTC().Year() - 1
}
