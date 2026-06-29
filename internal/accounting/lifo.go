package accounting

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// RunLIFO computes cost basis records using Last-In-First-Out.
// Every UTXO is an acquisition; spent UTXOs are also disposals.
// When a UTXO is disposed, the cost is matched against the most recently
// acquired holding not yet matched to another disposal.
func RunLIFO(
	walletID string,
	utxos []domain.UTXO,
	spendTimes map[string]time.Time,
	price PriceLookup,
	currency string,
) []domain.CostBasisRecord {
	type acquisition struct {
		utxo     domain.UTXO
		costFiat int64
		matched  bool
	}

	// Build acquisition list, newest first (LIFO stack: pop from front).
	acqs := make([]acquisition, len(utxos))
	for i, u := range utxos {
		acqPrice := price(currency, u.BlockTime)
		acqs[i] = acquisition{utxo: u, costFiat: satsToFiat(u.Sats, acqPrice)}
	}
	sort.Slice(acqs, func(i, j int) bool {
		return acqs[i].utxo.BlockTime.After(acqs[j].utxo.BlockTime)
	})

	// Collect disposals in chronological order.
	var disposals []domain.UTXO
	for _, u := range utxos {
		if u.Spent && u.SpentTxid != "" {
			disposals = append(disposals, u)
		}
	}
	sort.Slice(disposals, func(i, j int) bool {
		t1, _ := spendTimes[disposals[i].SpentTxid]
		t2, _ := spendTimes[disposals[j].SpentTxid]
		return t1.Before(t2)
	})

	// Match each disposal to the newest unmatched acquisition (LIFO).
	type disposalResult struct {
		utxo         domain.UTXO
		costFiat     int64
		proceedsFiat int64
		gainFiat     int64
		disposedAt   *time.Time
	}
	disposalResults := make([]disposalResult, 0, len(disposals))
	for _, d := range disposals {
		spendTime, ok := spendTimes[d.SpentTxid]
		if !ok {
			spendTime = time.Time{}
		}
		dispPrice := price(currency, spendTime)
		proceedsFiat := satsToFiat(d.Sats, dispPrice)

		var costFiat int64
		for i := range acqs {
			if !acqs[i].matched {
				costFiat = acqs[i].costFiat
				acqs[i].matched = true
				break
			}
		}
		if costFiat == 0 && proceedsFiat == 0 {
			costFiat = satsToFiat(d.Sats, price(currency, d.BlockTime))
		}

		gainFiat := proceedsFiat - costFiat
		dr := disposalResult{utxo: d, costFiat: costFiat, proceedsFiat: proceedsFiat, gainFiat: gainFiat}
		if !spendTime.IsZero() {
			dr.disposedAt = &spendTime
		}
		disposalResults = append(disposalResults, dr)
	}

	// Build records: unmatched acquisitions first (unspent holdings), then disposals.
	records := make([]domain.CostBasisRecord, 0, len(utxos))
	for _, a := range acqs {
		if !a.matched {
			records = append(records, domain.CostBasisRecord{
				ID:           uuid.New().String(),
				WalletID:     walletID,
				Txid:         a.utxo.Txid,
				Vout:         a.utxo.Vout,
				AcquiredAt:   a.utxo.BlockTime,
				CostSats:     a.utxo.Sats,
				CostFiat:     a.costFiat,
				FiatCurrency: currency,
				Method:       domain.MethodLIFO,
			})
		}
	}
	for _, dr := range disposalResults {
		proceeds := dr.proceedsFiat
		gain := dr.gainFiat
		records = append(records, domain.CostBasisRecord{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			Txid:         dr.utxo.Txid,
			Vout:         dr.utxo.Vout,
			AcquiredAt:   dr.utxo.BlockTime,
			CostSats:     dr.utxo.Sats,
			CostFiat:     dr.costFiat,
			FiatCurrency: currency,
			Method:       domain.MethodLIFO,
			DisposedAt:   dr.disposedAt,
			ProceedsFiat: &proceeds,
			GainFiat:     &gain,
		})
	}
	return records
}
