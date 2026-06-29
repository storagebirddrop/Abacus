package accounting

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// RunSection104 computes cost basis using the UK HMRC Section 104 pool method.
// TCGA 1992 s.104/105/106A — mandatory for UK cryptoassets.
//
// Matching priority (per HMRC guidance):
//  1. Same-day rule (s.105): disposals matched to acquisitions on the same calendar day.
//  2. 30-day rule (s.106A, "bed and breakfast"): disposals matched to acquisitions
//     within the 30 days immediately following the disposal date.
//  3. Section 104 pool: remaining acquisitions are pooled; disposal cost is drawn
//     proportionally from the pool's total allowable cost.
func RunSection104(
	walletID string,
	utxos []domain.UTXO,
	spendTimes map[string]time.Time,
	price PriceLookup,
	currency string,
) []domain.CostBasisRecord {
	type acqEntry struct {
		utxo      domain.UTXO
		costFiat  int64 // full acquisition cost
		remaining int64 // sats not yet matched/pooled
	}
	type dispEntry struct {
		utxo      domain.UTXO
		spendTime time.Time
		proceeds  int64 // full disposal proceeds
		remaining int64 // sats not yet matched
		allCost   int64 // accumulates matched/pool cost
	}

	// Collect acquisitions and disposals.
	var acqs []acqEntry
	var disps []dispEntry

	for _, u := range utxos {
		acqPrice := price(currency, u.BlockTime)
		costFiat := satsToFiat(u.Sats, acqPrice)
		acqs = append(acqs, acqEntry{
			utxo:      u,
			costFiat:  costFiat,
			remaining: u.Sats,
		})
		if u.Spent && u.SpentTxid != "" {
			st := spendTimes[u.SpentTxid]
			dispPrice := price(currency, st)
			proceeds := satsToFiat(u.Sats, dispPrice)
			disps = append(disps, dispEntry{
				utxo:      u,
				spendTime: st,
				proceeds:  proceeds,
				remaining: u.Sats,
			})
		}
	}

	// Sort acquisitions and disposals chronologically.
	sort.Slice(acqs, func(i, j int) bool {
		return acqs[i].utxo.BlockTime.Before(acqs[j].utxo.BlockTime)
	})
	sort.Slice(disps, func(i, j int) bool {
		return disps[i].spendTime.Before(disps[j].spendTime)
	})

	calDay := func(t time.Time) time.Time {
		y, m, d := t.UTC().Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}

	// Pass 1 — same-day rule (s.105).
	for di := range disps {
		if disps[di].remaining == 0 {
			continue
		}
		dispDay := calDay(disps[di].spendTime)
		for ai := range acqs {
			if acqs[ai].remaining == 0 {
				continue
			}
			if !calDay(acqs[ai].utxo.BlockTime).Equal(dispDay) {
				continue
			}
			matchSats := min(disps[di].remaining, acqs[ai].remaining)
			// Proportional cost from this acquisition lot.
			matchCost := acqs[ai].costFiat * matchSats / acqs[ai].utxo.Sats
			disps[di].allCost += matchCost
			disps[di].remaining -= matchSats
			acqs[ai].remaining -= matchSats
			if disps[di].remaining == 0 {
				break
			}
		}
	}

	// Pass 2 — 30-day rule (s.106A, "bed and breakfast").
	// For each disposal, look for acquisitions strictly after the disposal day,
	// within 30 calendar days, in chronological order.
	for di := range disps {
		if disps[di].remaining == 0 {
			continue
		}
		dispDay := calDay(disps[di].spendTime)
		deadline := dispDay.AddDate(0, 0, 30)
		for ai := range acqs {
			if acqs[ai].remaining == 0 {
				continue
			}
			acqDay := calDay(acqs[ai].utxo.BlockTime)
			if !acqDay.After(dispDay) || acqDay.After(deadline) {
				continue
			}
			matchSats := min(disps[di].remaining, acqs[ai].remaining)
			matchCost := acqs[ai].costFiat * matchSats / acqs[ai].utxo.Sats
			disps[di].allCost += matchCost
			disps[di].remaining -= matchSats
			acqs[ai].remaining -= matchSats
			if disps[di].remaining == 0 {
				break
			}
		}
	}

	// Pass 3 — Section 104 pool.
	// Merge acquisition and disposal events by time; acquisitions before disposals
	// on the same day (pool is built before disposals draw from it).
	type event struct {
		t     time.Time
		isAcq bool
		idx   int
	}
	events := make([]event, 0, len(acqs)+len(disps))
	for i := range acqs {
		events = append(events, event{acqs[i].utxo.BlockTime, true, i})
	}
	for i := range disps {
		events = append(events, event{disps[i].spendTime, false, i})
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].t.Equal(events[j].t) {
			return events[i].isAcq && !events[j].isAcq
		}
		return events[i].t.Before(events[j].t)
	})

	var poolSats, poolCostFiat int64
	for _, ev := range events {
		if ev.isAcq {
			a := &acqs[ev.idx]
			if a.remaining > 0 {
				// Add remaining sats (unused by same-day/30-day) to pool.
				addCost := a.costFiat * a.remaining / a.utxo.Sats
				poolSats += a.remaining
				poolCostFiat += addCost
				a.remaining = 0
			}
		} else {
			d := &disps[ev.idx]
			if d.remaining > 0 && poolSats > 0 {
				matchSats := min(d.remaining, poolSats)
				poolCost := poolCostFiat * matchSats / poolSats
				d.allCost += poolCost
				d.remaining -= matchSats
				poolSats -= matchSats
				poolCostFiat -= poolCost
			}
		}
	}

	// Map disposal results by UTXO key for output.
	type dispResult struct {
		spendTime time.Time
		allCost   int64
		proceeds  int64
	}
	dispMap := make(map[string]dispResult, len(disps))
	for _, d := range disps {
		key := fmt.Sprintf("%s:%d", d.utxo.Txid, d.utxo.Vout)
		dispMap[key] = dispResult{
			spendTime: d.spendTime,
			allCost:   d.allCost,
			proceeds:  d.proceeds,
		}
	}

	// Build one CostBasisRecord per UTXO.
	records := make([]domain.CostBasisRecord, 0, len(utxos))
	for _, u := range utxos {
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
			Method:       domain.MethodSection104,
		}

		if u.Spent && u.SpentTxid != "" {
			key := fmt.Sprintf("%s:%d", u.Txid, u.Vout)
			if dr, ok := dispMap[key]; ok {
				gainFiat := dr.proceeds - dr.allCost
				cb.CostFiat = dr.allCost
				cb.DisposedAt = &dr.spendTime
				cb.ProceedsFiat = &dr.proceeds
				cb.GainFiat = &gainFiat
			}
		}

		records = append(records, cb)
	}
	return records
}
