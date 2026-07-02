package accounting

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// RunExchangeFIFO computes cost basis records for exchange trades using
// First-In-First-Out lot matching with partial-lot splitting.
//
//   - Buy/deposit/lightning_receive trades open new lots.
//   - Sell/withdrawal/lightning_send trades consume lots chronologically.
//   - Lots are split when a disposal partially consumes one.
//   - The method parameter is recorded on each record for reporting.
//
// All monetary values are in fiat cents. If a trade's FiatCurrency differs
// from the requested currency, its fiat amount is stored as 0.
func RunExchangeFIFO(walletID string, trades []*domain.ExchangeTrade, currency string, method domain.CostBasisMethod) []domain.CostBasisRecord {
	// acquisitionLot tracks a single buy lot that may be partially consumed.
	type acquisitionLot struct {
		tradeID      string
		acquiredAt   time.Time
		remainSats   int64
		originalSats int64
		totalCost    int64 // fiat cents for originalSats
	}

	// Sort trades chronologically so acquisitions and disposals are processed in order.
	sorted := make([]*domain.ExchangeTrade, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TradedAt.Before(sorted[j].TradedAt)
	})

	var queue []acquisitionLot
	var records []domain.CostBasisRecord

	fiatFor := func(t *domain.ExchangeTrade) int64 {
		if t.FiatCurrency == currency {
			return t.FiatAmount
		}
		return 0
	}

	for _, trade := range sorted {
		switch {
		case trade.IsAcquisition():
			sats := trade.Sats
			if sats < 0 {
				sats = -sats
			}
			queue = append(queue, acquisitionLot{
				tradeID:      trade.ID,
				acquiredAt:   trade.TradedAt,
				remainSats:   sats,
				originalSats: sats,
				totalCost:    fiatFor(trade),
			})

		case trade.IsDisposal():
			sellSats := trade.Sats
			if sellSats < 0 {
				sellSats = -sellSats
			}
			totalProceeds := fiatFor(trade)
			dispTime := trade.TradedAt
			remaining := sellSats

			for remaining > 0 && len(queue) > 0 {
				front := &queue[0]
				consumed := front.remainSats
				if consumed > remaining {
					consumed = remaining
				}

				// Proportional cost: lot.totalCost × consumed / lot.originalSats
				var costForConsumed int64
				if front.originalSats > 0 {
					costForConsumed = front.totalCost * consumed / front.originalSats
				}

				// Proportional proceeds: totalProceeds × consumed / sellSats
				var proceedsForConsumed int64
				if sellSats > 0 {
					proceedsForConsumed = totalProceeds * consumed / sellSats
				}

				gainFiat := proceedsForConsumed - costForConsumed

				records = append(records, domain.CostBasisRecord{
					ID:           uuid.New().String(),
					WalletID:     walletID,
					Txid:         front.tradeID,
					Vout:         0,
					AcquiredAt:   front.acquiredAt,
					CostSats:     consumed,
					CostFiat:     costForConsumed,
					FiatCurrency: currency,
					Method:       method,
					DisposedAt:   &dispTime,
					ProceedsFiat: &proceedsForConsumed,
					GainFiat:     &gainFiat,
				})

				remaining -= consumed
				front.remainSats -= consumed
				if front.remainSats == 0 {
					queue = queue[1:]
				}
			}
			// Any remaining sats from a disposal with no matching lot are silently
			// dropped (this can happen if import history is incomplete).
		}
	}

	// Remaining open lots → unspent cost basis records.
	for i := range queue {
		l := &queue[i]
		var costForRemain int64
		if l.originalSats > 0 {
			costForRemain = l.totalCost * l.remainSats / l.originalSats
		}
		records = append(records, domain.CostBasisRecord{
			ID:           uuid.New().String(),
			WalletID:     walletID,
			Txid:         l.tradeID,
			Vout:         0,
			AcquiredAt:   l.acquiredAt,
			CostSats:     l.remainSats,
			CostFiat:     costForRemain,
			FiatCurrency: currency,
			Method:       method,
		})
	}

	return records
}
