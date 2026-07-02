// Package bitonic implements a WalletImporter for Bitonic exchange CSV exports.
//
// Bitonic is a Dutch BTC-only instant buy/sell exchange.
// To export: log in → History → Export CSV.
//
// Expected CSV columns (semicolon-separated):
//
//	Datum;Type;Bedrag BTC;Prijs (EUR);Kosten (EUR);Totaal (EUR);Referentie
//
// Types: "Aankoop" (buy), "Verkoop" (sell).
package bitonic

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

func (imp *Importer) Name() string              { return "bitonic" }
func (imp *Importer) SupportedFormats() []string { return []string{".csv"} }

func (imp *Importer) Detect(filename string, r io.ReadSeeker) bool {
	if strings.Contains(strings.ToLower(filename), "bitonic") {
		return true
	}
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	h := string(buf[:n])
	return strings.Contains(h, "Bedrag BTC") || strings.Contains(h, "bedrag btc")
}

func (imp *Importer) Import(_ context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	cr := csv.NewReader(r)
	cr.Comma = ';'
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

		rawType := get(row, "type")
		tt, ok := bitonicTradeType(rawType)
		if !ok {
			result.Errors = append(result.Errors, importer.ImportError{Message: "unknown trade type: " + rawType})
			continue
		}

		tradedAt, err := time.Parse("02-01-2006 15:04:05", get(row, "datum"))
		if err != nil {
			// Try date-only fallback.
			tradedAt, err = time.Parse("02-01-2006", get(row, "datum"))
			if err != nil {
				result.Errors = append(result.Errors, importer.ImportError{Message: "invalid date: " + get(row, "datum")})
				continue
			}
		}

		sats := common.ParseBTCSats(get(row, "bedrag btc"))
		if sats < 0 {
			sats = -sats
		}

		// "Totaal (EUR)" is the total cost/proceeds including fees.
		fiatCents := common.ParseFiatCents(get(row, "totaal (eur)"))
		if fiatCents < 0 {
			fiatCents = -fiatCents
		}
		feeFiat := common.ParseFiatCents(get(row, "kosten (eur)"))
		if feeFiat < 0 {
			feeFiat = -feeFiat
		}

		result.Trades = append(result.Trades, domain.ExchangeTrade{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			TradeType:    tt,
			Exchange:     "bitonic",
			ExternalID:   get(row, "referentie"),
			TradedAt:     tradedAt.UTC(),
			Sats:         sats,
			FiatAmount:   fiatCents,
			FiatCurrency: "EUR",
			FeeFiat:      feeFiat,
			CreatedAt:    now,
		})
	}

	return result, nil
}

func bitonicTradeType(s string) (domain.TradeType, bool) {
	switch strings.ToLower(s) {
	case "aankoop", "buy":
		return domain.TradeTypeBuy, true
	case "verkoop", "sell":
		return domain.TradeTypeSell, true
	default:
		return "", false
	}
}
