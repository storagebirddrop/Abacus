// Package coinbase implements a WalletImporter for Coinbase "Transaction History" CSV exports.
//
// To export from Coinbase: go to Assets → BTC → Statements → Generate report → All time.
// Or: Profile → Taxes → Download → Transaction History CSV.
//
// Expected CSV columns:
//
//	Timestamp,Transaction Type,Asset,Quantity Transacted,Spot Price Currency,
//	Spot Price at Transaction,Subtotal,Total (inclusive of fees and/or spread),
//	Fees and/or Spread,Notes
//
// Only BTC rows are imported.
package coinbase

import (
	"context"
	"encoding/csv"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/importer"
	"github.com/storagebirddrop/abacus/internal/importer/common"
)

type Importer struct{}

func New() *Importer { return &Importer{} }

func (imp *Importer) Name() string              { return "coinbase" }
func (imp *Importer) SupportedFormats() []string { return []string{".csv"} }

func (imp *Importer) Detect(filename string, r io.ReadSeeker) bool {
	if strings.Contains(strings.ToLower(filename), "coinbase") {
		return true
	}
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	h := strings.ToLower(string(buf[:n]))
	return strings.Contains(h, "spot price currency") && strings.Contains(h, "quantity transacted")
}

func (imp *Importer) Import(_ context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.LazyQuotes = true

	// Coinbase exports sometimes include a title row or blank lines before the header.
	var header []string
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(row) > 0 && strings.EqualFold(strings.TrimSpace(row[0]), "timestamp") {
			header = row
			break
		}
	}
	if header == nil {
		return &importer.ImportResult{}, nil
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(row []string, key string) string {
		if i, ok := idx[key]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	result := &importer.ImportResult{}
	now := time.Now().UTC()

	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, importer.ImportError{Message: err.Error()})
			continue
		}

		if get(row, "asset") != "BTC" {
			continue
		}

		rawType := strings.ToLower(get(row, "transaction type"))
		tt, ok := coinbaseTradeType(rawType)
		if !ok {
			// Skip non-BTC transaction types like "Convert" from altcoins.
			continue
		}

		tradedAt, err := parseCoinbaseTime(get(row, "timestamp"))
		if err != nil {
			result.Errors = append(result.Errors, importer.ImportError{Message: "invalid timestamp: " + get(row, "timestamp")})
			continue
		}

		sats := common.ParseBTCSats(get(row, "quantity transacted"))
		if sats < 0 {
			sats = -sats
		}

		priceCurrency := get(row, "spot price currency")
		if priceCurrency == "" {
			priceCurrency = "USD"
		}

		// "Total (inclusive of fees and/or spread)" is the all-in cost/proceeds.
		totalFiatStr := get(row, "total (inclusive of fees and/or spread)")
		fiatCents := common.ParseFiatCents(totalFiatStr)
		if fiatCents < 0 {
			fiatCents = -fiatCents
		}

		feeFiat := common.ParseFiatCents(get(row, "fees and/or spread"))
		if feeFiat < 0 {
			feeFiat = -feeFiat
		}

		// Coinbase's CSV export has no order/trade ID, so derive a stable dedup
		// key from the row's own fields (see common.SyntheticExternalID) --
		// otherwise re-importing an export that overlaps previous history
		// would insert full duplicate trades with no error.
		externalID := common.SyntheticExternalID(
			get(row, "timestamp"), rawType, get(row, "quantity transacted"),
			totalFiatStr, get(row, "fees and/or spread"),
		)

		result.Trades = append(result.Trades, domain.ExchangeTrade{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			TradeType:    tt,
			Exchange:     "coinbase",
			ExternalID:   externalID,
			TradedAt:     tradedAt,
			Sats:         sats,
			FiatAmount:   fiatCents,
			FiatCurrency: priceCurrency,
			FeeFiat:      feeFiat,
			Note:         get(row, "notes"),
			CreatedAt:    now,
		})
	}

	return result, nil
}

func coinbaseTradeType(s string) (domain.TradeType, bool) {
	switch s {
	case "buy":
		return domain.TradeTypeBuy, true
	case "sell":
		return domain.TradeTypeSell, true
	case "send":
		return domain.TradeTypeWithdrawal, true
	case "receive":
		return domain.TradeTypeDeposit, true
	default:
		return "", false
	}
}

// parseCoinbaseTime handles ISO 8601 timestamps like "2024-01-15T14:30:00Z"
// and "2024-01-15T14:30:00.000Z".
func parseCoinbaseTime(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, &parseError{s}
}

type parseError struct{ s string }

func (e *parseError) Error() string { return "cannot parse time: " + e.s }
