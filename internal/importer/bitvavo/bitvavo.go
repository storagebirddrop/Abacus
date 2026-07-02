// Package bitvavo implements a WalletImporter for Bitvavo exchange CSV exports.
//
// To export from Bitvavo: Account → Transaction history → Export → CSV.
// The file includes all assets; only BTC rows are imported.
//
// Expected CSV columns:
//
//	Date,Time,Type,Currency,Amount,EUR equivalent,Balance after,Fee currency,Fee amount,Status,Transaction ID,Address
package bitvavo

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

func (imp *Importer) Name() string              { return "bitvavo" }
func (imp *Importer) SupportedFormats() []string { return []string{".csv"} }

func (imp *Importer) Detect(filename string, r io.ReadSeeker) bool {
	if strings.Contains(strings.ToLower(filename), "bitvavo") {
		return true
	}
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	h := strings.ToLower(string(buf[:n]))
	return strings.Contains(h, "eur equivalent") && strings.Contains(h, "balance after")
}

func (imp *Importer) Import(_ context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		return nil, err
	}
	idx := headerIndex(header)
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

		currency := get(row, "currency")
		if currency != "BTC" && currency != "XBT" {
			continue
		}

		rawType := strings.ToLower(get(row, "type"))
		tt, ok := bitvavoTradeType(rawType)
		if !ok {
			result.Errors = append(result.Errors, importer.ImportError{Message: "unknown trade type: " + rawType})
			continue
		}

		tradedAt, err := parseDateTime(get(row, "date"), get(row, "time"))
		if err != nil {
			result.Errors = append(result.Errors, importer.ImportError{Message: "invalid date: " + get(row, "date")})
			continue
		}

		sats := common.ParseBTCSats(get(row, "amount"))
		fiatCents := common.ParseFiatCents(get(row, "eur equivalent"))
		if fiatCents < 0 {
			fiatCents = -fiatCents
		}

		var feeSats, feeFiat int64
		feeCur := get(row, "fee currency")
		rawFee := get(row, "fee amount")
		switch feeCur {
		case "BTC", "XBT":
			feeSats = common.ParseBTCSats(rawFee)
			if feeSats < 0 {
				feeSats = -feeSats
			}
		case "EUR":
			feeFiat = common.ParseFiatCents(rawFee)
			if feeFiat < 0 {
				feeFiat = -feeFiat
			}
		}

		result.Trades = append(result.Trades, domain.ExchangeTrade{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			TradeType:    tt,
			Exchange:     "bitvavo",
			ExternalID:   get(row, "transaction id"),
			TradedAt:     tradedAt,
			Sats:         sats,
			FiatAmount:   fiatCents,
			FiatCurrency: "EUR",
			FeeSats:      feeSats,
			FeeFiat:      feeFiat,
			CreatedAt:    now,
		})
	}

	return result, nil
}

func headerIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return idx
}

func bitvavoTradeType(s string) (domain.TradeType, bool) {
	switch s {
	case "buy":
		return domain.TradeTypeBuy, true
	case "sell":
		return domain.TradeTypeSell, true
	case "deposit":
		return domain.TradeTypeDeposit, true
	case "withdrawal", "withdraw":
		return domain.TradeTypeWithdrawal, true
	case "fee":
		return domain.TradeTypeFee, true
	default:
		return "", false
	}
}

// parseDateTime parses Bitvavo date "2024-01-15" and optional time "14:30:00".
func parseDateTime(date, t string) (time.Time, error) {
	if t == "" {
		return time.Parse("2006-01-02", date)
	}
	return time.Parse("2006-01-02 15:04:05", date+" "+t)
}
