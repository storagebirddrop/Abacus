package accounting

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// RunAvgCost computes cost basis records using the Average Cost method.
// A running pool tracks total sats held and total fiat cost paid.
// On each disposal the average cost per sat at that moment is used.
//
// spendTimes maps spending txid → block_time so disposal prices can be looked up.
func RunAvgCost(
	walletID string,
	utxos []domain.UTXO,
	spendTimes map[string]time.Time,
	price PriceLookup,
	currency string,
) []domain.CostBasisRecord {
	// Process UTXOs chronologically so the pool evolves in time order.
	sorted := make([]domain.UTXO, len(utxos))
	copy(sorted, utxos)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BlockTime.Before(sorted[j].BlockTime)
	})

	// Running pool state.
	var poolSats int64  // total sats currently held
	var poolFiat int64  // total fiat cost of current holdings (cents)

	records := make([]domain.CostBasisRecord, 0, len(sorted))

	for _, u := range sorted {
		acqPrice := price(currency, u.BlockTime)
		costFiat := satsToFiat(u.Sats, acqPrice)

		// Add this acquisition to the pool.
		poolSats += u.Sats
		poolFiat += costFiat

		cb := domain.CostBasisRecord{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			Txid:         u.Txid,
			Vout:         u.Vout,
			AcquiredAt:   u.BlockTime,
			CostSats:     u.Sats,
			CostFiat:     costFiat,
			FiatCurrency: currency,
			Method:       domain.MethodAvgCost,
		}

		if u.Spent && u.SpentTxid != "" {
			// Average cost per sat at disposal time.
			var avgCostPerSat int64
			if poolSats > 0 {
				avgCostPerSat = poolFiat / poolSats // integer division in cents/sat
			}
			avgCostFiat := avgCostPerSat * u.Sats

			spendTime, ok := spendTimes[u.SpentTxid]
			if !ok {
				spendTime = time.Time{}
			}
			dispPrice := price(currency, spendTime)
			proceedsFiat := satsToFiat(u.Sats, dispPrice)
			gainFiat := proceedsFiat - avgCostFiat

			if !spendTime.IsZero() {
				cb.DisposedAt = &spendTime
			}
			cb.CostFiat = avgCostFiat // override with pool-average cost
			cb.ProceedsFiat = &proceedsFiat
			cb.GainFiat = &gainFiat

			// Remove disposed sats from pool.
			poolSats -= u.Sats
			poolFiat -= avgCostFiat
			if poolSats < 0 {
				poolSats = 0
			}
			if poolFiat < 0 {
				poolFiat = 0
			}
		}

		records = append(records, cb)
	}
	return records
}
