package accounting

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// RunHIFO computes cost basis records using Highest-In-First-Out.
// When a UTXO is disposed, its cost is matched against the highest-cost
// acquisition not yet matched to another disposal, minimising short-term gains.
func RunHIFO(
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

	// Build acquisition pool for all UTXOs.
	acqs := make([]acquisition, len(utxos))
	for i, u := range utxos {
		acqPrice := price(currency, u.BlockTime)
		acqs[i] = acquisition{utxo: u, costFiat: satsToFiat(u.Sats, acqPrice)}
	}

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

		// Find highest-cost unmatched acquisition.
		bestIdx := -1
		for i := range acqs {
			if !acqs[i].matched {
				if bestIdx < 0 || acqs[i].costFiat > acqs[bestIdx].costFiat {
					bestIdx = i
				}
			}
		}
		var costFiat int64
		if bestIdx >= 0 {
			costFiat = acqs[bestIdx].costFiat
			acqs[bestIdx].matched = true
		} else {
			costFiat = satsToFiat(d.Sats, price(currency, d.BlockTime))
		}

		gainFiat := proceedsFiat - costFiat
		dr := disposalResult{utxo: d, costFiat: costFiat, proceedsFiat: proceedsFiat, gainFiat: gainFiat}
		if !spendTime.IsZero() {
			dr.disposedAt = &spendTime
		}
		disposalResults = append(disposalResults, dr)
	}

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
				Method:       domain.MethodHIFO,
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
			Method:       domain.MethodHIFO,
			DisposedAt:   dr.disposedAt,
			ProceedsFiat: &proceeds,
			GainFiat:     &gain,
		})
	}
	return records
}
