package accounting

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// SpecificIDSelection maps a disposal UTXO key ("txid:vout") to the acquisition
// UTXO key ("txid:vout") that should be matched against it.
// Disposals without an explicit mapping fall back to FIFO (oldest unmatched acquisition).
type SpecificIDSelection map[string]string

// RunSpecificID computes cost basis using Specific Identification.
// The caller provides an explicit mapping of disposal→acquisition via selections.
// Disposals without a mapping fall back to FIFO ordering.
func RunSpecificID(
	walletID string,
	utxos []domain.UTXO,
	spendTimes map[string]time.Time,
	price PriceLookup,
	currency string,
	selections SpecificIDSelection,
) []domain.CostBasisRecord {
	utxoKey := func(u domain.UTXO) string {
		return fmt.Sprintf("%s:%d", u.Txid, u.Vout)
	}

	type acquisition struct {
		utxo     domain.UTXO
		key      string
		costFiat int64
		matched  bool
	}

	// Build acquisition pool — all UTXOs, FIFO order (oldest first) for fallback.
	acqs := make([]acquisition, len(utxos))
	for i, u := range utxos {
		acqPrice := price(currency, u.BlockTime)
		acqs[i] = acquisition{
			utxo:     u,
			key:      utxoKey(u),
			costFiat: satsToFiat(u.Sats, acqPrice),
		}
	}
	sort.Slice(acqs, func(i, j int) bool {
		return acqs[i].utxo.BlockTime.Before(acqs[j].utxo.BlockTime)
	})
	acqByKey := make(map[string]*acquisition, len(acqs))
	for i := range acqs {
		acqByKey[acqs[i].key] = &acqs[i]
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

		var costFiat int64
		dispKey := utxoKey(d)

		// Try explicit selection first.
		if selections != nil {
			if acqKey, explicit := selections[dispKey]; explicit {
				if a, found := acqByKey[acqKey]; found && !a.matched {
					costFiat = a.costFiat
					a.matched = true
				}
			}
		}

		// Fallback: oldest unmatched acquisition (FIFO).
		if costFiat == 0 {
			for i := range acqs {
				if !acqs[i].matched {
					costFiat = acqs[i].costFiat
					acqs[i].matched = true
					break
				}
			}
		}
		if costFiat == 0 {
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
				Method:       domain.MethodSpecificID,
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
			Method:       domain.MethodSpecificID,
			DisposedAt:   dr.disposedAt,
			ProceedsFiat: &proceeds,
			GainFiat:     &gain,
		})
	}
	return records
}
