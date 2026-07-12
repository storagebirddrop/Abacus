// Package kraken implements a WalletImporter for Kraken "Ledgers" CSV exports.
//
// To export from Kraken: go to History → Export → Ledgers → Export.
// The export is called "ledgers.csv".
//
// This importer reconstructs trades by pairing rows with the same "refid":
//   - A BTC buy generates a spend row (EUR, negative amount) + receive row (XBT, positive).
//   - A BTC sell generates a spend row (XBT, negative) + receive row (EUR, positive).
//
// Expected CSV columns:
//
//	txid,refid,time,type,subtype,aclass,asset,amount,fee,balance
//
// Only XBT/XXBT asset rows are retained; their EUR counterparts are found by refid.
package kraken

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

func (imp *Importer) Name() string              { return "kraken" }
func (imp *Importer) SupportedFormats() []string { return []string{".csv"} }

func (imp *Importer) Detect(filename string, r io.ReadSeeker) bool {
	if strings.Contains(strings.ToLower(filename), "kraken") {
		return true
	}
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	h := strings.ToLower(string(buf[:n]))
	return strings.Contains(h, "refid") && strings.Contains(h, "aclass")
}

// krakenRow holds one parsed ledger row.
type krakenRow struct {
	txid   string
	refid  string
	t      time.Time
	kind   string // "trade", "deposit", "withdrawal", "spend", "receive"
	asset  string // "XBT", "XXBT", "EUR", "ZEUR", etc.
	amount int64  // in asset's base unit (satoshis for BTC, cents for EUR)
	fee    int64  // same unit
}

func isBTC(asset string) bool {
	a := strings.ToUpper(asset)
	return a == "XBT" || a == "XXBT" || a == "BTC"
}

// krakenFiatCodes maps Kraken's asset codes for its supported fiat rails to
// their ISO 4217 currency code. Kraken supports far more than EUR (USD, GBP,
// CAD, JPY, CHF, AUD, ...); older exports use the "Z"-prefixed form
// (ZUSD, ZEUR, ...), newer ones sometimes drop it.
var krakenFiatCodes = map[string]string{
	"ZUSD": "USD", "USD": "USD",
	"ZEUR": "EUR", "EUR": "EUR",
	"ZGBP": "GBP", "GBP": "GBP",
	"ZCAD": "CAD", "CAD": "CAD",
	"ZJPY": "JPY", "JPY": "JPY",
	"ZCHF": "CHF", "CHF": "CHF",
	"ZAUD": "AUD", "AUD": "AUD",
}

// fiatCode returns the normalised ISO currency code for a Kraken fiat asset,
// and whether the asset is a recognised fiat rail at all.
func fiatCode(asset string) (string, bool) {
	code, ok := krakenFiatCodes[strings.ToUpper(asset)]
	return code, ok
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

	// First pass: collect all rows grouped by refid.
	type refGroup struct {
		btcRows  []krakenRow
		fiatRows []krakenRow
	}
	groups := map[string]*refGroup{}
	var order []string // preserve refid insertion order

	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		asset := get(row, "asset")
		kind := strings.ToLower(get(row, "type"))
		refid := get(row, "refid")
		if refid == "" {
			refid = get(row, "txid")
		}

		var amount, fee int64
		isFiat := false
		if isBTC(asset) {
			amount = common.ParseBTCSats(get(row, "amount"))
			fee = common.ParseBTCSats(get(row, "fee"))
		} else if _, ok := fiatCode(asset); ok {
			isFiat = true
			amount = common.ParseFiatCents(get(row, "amount"))
			fee = common.ParseFiatCents(get(row, "fee"))
		} else {
			// Non-BTC, non-fiat asset — skip (e.g., ETH)
			continue
		}

		t, parseErr := time.Parse("2006-01-02 15:04:05", get(row, "time"))
		if parseErr != nil {
			t, _ = time.Parse("2006-01-02 15:04:05.0000", get(row, "time"))
		}

		kr := krakenRow{
			txid:   get(row, "txid"),
			refid:  refid,
			t:      t.UTC(),
			kind:   kind,
			asset:  asset,
			amount: amount,
			fee:    fee,
		}

		if _, exists := groups[refid]; !exists {
			groups[refid] = &refGroup{}
			order = append(order, refid)
		}
		g := groups[refid]
		if isBTC(asset) {
			g.btcRows = append(g.btcRows, kr)
		} else if isFiat {
			g.fiatRows = append(g.fiatRows, kr)
		}
	}

	result := &importer.ImportResult{}
	now := time.Now().UTC()

	// Second pass: construct ExchangeTrades from paired rows.
	for _, refid := range order {
		g := groups[refid]

		for _, btcRow := range g.btcRows {
			sats := btcRow.amount
			feeSats := btcRow.fee
			if feeSats < 0 {
				feeSats = -feeSats
			}

			var tt domain.TradeType
			var fiatCents, feeFiat int64
			fiatCurrency := "EUR" // fallback if no paired fiat row is found at all

			switch btcRow.kind {
			case "deposit":
				tt = domain.TradeTypeDeposit
				if sats < 0 {
					sats = -sats
				}
			case "withdrawal":
				tt = domain.TradeTypeWithdrawal
				if sats < 0 {
					sats = -sats
				}
			case "trade", "spend", "receive":
				// Find the paired fiat row with the same refid. Kraken supports
				// many fiat rails (USD, GBP, CAD, ...), not just EUR — use
				// whatever currency the paired row actually reports, and
				// capture its fee (previously parsed but discarded here).
				for _, fiatRow := range g.fiatRows {
					fc := fiatRow.amount
					if fc < 0 {
						fc = -fc
					}
					fiatCents = fc
					if code, ok := fiatCode(fiatRow.asset); ok {
						fiatCurrency = code
					}
					ff := fiatRow.fee
					if ff < 0 {
						ff = -ff
					}
					feeFiat = ff
					break
				}
				if sats > 0 {
					tt = domain.TradeTypeBuy
				} else {
					tt = domain.TradeTypeSell
					sats = -sats
				}
			default:
				// Unknown type — skip.
				continue
			}

			trade := domain.ExchangeTrade{
				ID:           uuid.New().String(),
				WalletID:     walletID,
				TradeType:    tt,
				Exchange:     "kraken",
				ExternalID:   refid,
				TradedAt:     btcRow.t,
				Sats:         sats,
				FiatAmount:   fiatCents,
				FiatCurrency: fiatCurrency,
				FeeFiat:      feeFiat,
				FeeSats:      feeSats,
				CreatedAt:    now,
			}
			result.Trades = append(result.Trades, trade)
		}
	}

	return result, nil
}
