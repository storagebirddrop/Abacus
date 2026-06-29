package accounting

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// RunFIFO computes cost basis records for all UTXOs using First-In-First-Out.
// UTXOs are sorted by acquisition time; each UTXO maps to one CostBasisRecord.
//
// spendTimes maps spending txid → block_time so disposal prices can be looked up.
// If a spent UTXO's txid is absent from spendTimes, disposal time is zero and
// proceeds/gain are stored as 0 (backfilled when the data arrives).
func RunFIFO(
	walletID string,
	utxos []domain.UTXO,
	spendTimes map[string]time.Time,
	price PriceLookup,
	currency string,
) []domain.CostBasisRecord {
	sorted := make([]domain.UTXO, len(utxos))
	copy(sorted, utxos)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BlockTime.Before(sorted[j].BlockTime)
	})

	records := make([]domain.CostBasisRecord, 0, len(sorted))
	for _, u := range sorted {
		acqPrice := price(currency, u.BlockTime)
		costFiat := satsToFiat(u.Sats, acqPrice)

		cb := domain.CostBasisRecord{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			Txid:         u.Txid,
			Vout:         u.Vout,
			AcquiredAt:   u.BlockTime,
			CostSats:     u.Sats,
			CostFiat:     costFiat,
			FiatCurrency: currency,
			Method:       domain.MethodFIFO,
		}

		if u.Spent && u.SpentTxid != "" {
			spendTime, ok := spendTimes[u.SpentTxid]
			if !ok {
				spendTime = time.Time{} // unknown; zero value
			}
			dispPrice := price(currency, spendTime)
			proceedsFiat := satsToFiat(u.Sats, dispPrice)
			gainFiat := proceedsFiat - costFiat
			if !spendTime.IsZero() {
				cb.DisposedAt = &spendTime
			}
			cb.ProceedsFiat = &proceedsFiat
			cb.GainFiat = &gainFiat
		}

		records = append(records, cb)
	}
	return records
}
