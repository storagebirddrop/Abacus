// Package strike implements a WalletImporter for Strike "Transaction History" CSV exports.
//
// Strike is a Lightning-first Bitcoin app. Trades are BTC/USD only.
// To export: go to Activity → Download CSV.
//
// Expected CSV columns:
//
//	Date,Type,BTC Amount,USD Amount,Fee,Balance (BTC),Balance (USD),Description
//
// Lightning payments map to lightning_receive/lightning_send; exchange trades
// map to buy/sell.
package strike

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

func (imp *Importer) Name() string              { return "strike" }
func (imp *Importer) SupportedFormats() []string { return []string{".csv"} }

func (imp *Importer) Detect(filename string, r io.ReadSeeker) bool {
	if strings.Contains(strings.ToLower(filename), "strike") {
		return true
	}
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	h := strings.ToLower(string(buf[:n]))
	return strings.Contains(h, "balance (btc)") && strings.Contains(h, "balance (usd)")
}

func (imp *Importer) Import(_ context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		return nil, err
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

		rawType := strings.ToLower(strings.TrimSpace(get(row, "type")))
		tt, ok := strikeTradeType(rawType)
		if !ok {
			// Unknown/unrecognised row type — skip silently.
			continue
		}

		tradedAt, err := parseStrikeTime(get(row, "date"))
		if err != nil {
			result.Errors = append(result.Errors, importer.ImportError{Message: "invalid date: " + get(row, "date")})
			continue
		}

		sats := common.ParseBTCSats(get(row, "btc amount"))
		if sats < 0 {
			sats = -sats
		}

		fiatCents := common.ParseFiatCents(get(row, "usd amount"))
		if fiatCents < 0 {
			fiatCents = -fiatCents
		}

		feeFiat := common.ParseFiatCents(get(row, "fee"))
		if feeFiat < 0 {
			feeFiat = -feeFiat
		}

		// Strike's CSV export has no per-row transaction ID, so derive a
		// stable dedup key from the row's own fields (see
		// common.SyntheticExternalID) -- otherwise re-importing an export
		// that overlaps previous history would insert full duplicate trades
		// with no error.
		externalID := common.SyntheticExternalID(
			get(row, "date"), rawType, get(row, "btc amount"),
			get(row, "usd amount"), get(row, "fee"),
		)

		result.Trades = append(result.Trades, domain.ExchangeTrade{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			TradeType:    tt,
			Exchange:     "strike",
			ExternalID:   externalID,
			TradedAt:     tradedAt,
			Sats:         sats,
			FiatAmount:   fiatCents,
			FiatCurrency: "USD",
			FeeFiat:      feeFiat,
			Note:         get(row, "description"),
			CreatedAt:    now,
		})
	}

	return result, nil
}

func strikeTradeType(s string) (domain.TradeType, bool) {
	switch s {
	case "payment received", "receive":
		return domain.TradeTypeLightningReceive, true
	case "payment sent", "send":
		return domain.TradeTypeLightningSend, true
	case "trade buy", "buy":
		return domain.TradeTypeBuy, true
	case "trade sell", "sell":
		return domain.TradeTypeSell, true
	case "deposit":
		return domain.TradeTypeDeposit, true
	case "withdrawal", "withdraw":
		return domain.TradeTypeWithdrawal, true
	default:
		return "", false
	}
}

func parseStrikeTime(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, &strikeParseErr{s}
}

type strikeParseErr struct{ s string }

func (e *strikeParseErr) Error() string { return "cannot parse time: " + e.s }
