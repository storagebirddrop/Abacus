package accounting_test

import (
	"testing"
	"time"

	"github.com/storagebirddrop/abacus/internal/accounting"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// noPrices returns 0 for every lookup (no price data available).
func noPrices(currency string, t time.Time) int64 { return 0 }

// fixedPrice returns a constant BTC price in cents/BTC for any timestamp.
func fixedPrice(cents int64) accounting.PriceLookup {
	return func(currency string, t time.Time) int64 {
		if t.IsZero() {
			return 0
		}
		return cents
	}
}

var (
	t1 = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 = time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
)

func utxo(txid string, vout int, sats int64, acquiredAt time.Time, spent bool, spentTxid string) domain.UTXO {
	return domain.UTXO{
		WalletID:  "wallet-1",
		Txid:      txid,
		Vout:      vout,
		Sats:      sats,
		BlockTime: acquiredAt,
		Spent:     spent,
		SpentTxid: spentTxid,
	}
}

// --- FIFO tests ---

func TestFIFO_UnspentOnly(t *testing.T) {
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""), // 0.01 BTC
		utxo("tx2", 0, 2_000_000, t2, false, ""), // 0.02 BTC
	}
	// BTC at 30 000 EUR = 3 000 000 cents/BTC
	// 0.01 BTC * 30 000 EUR = 300 EUR = 30 000 cents
	// 0.02 BTC * 30 000 EUR = 600 EUR = 60 000 cents
	price := fixedPrice(3_000_000) // 30 000 EUR/BTC in cents
	records := accounting.RunFIFO("wallet-1", utxos, nil, price, "EUR")

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].CostFiat != 30_000 {
		t.Errorf("record[0].CostFiat = %d, want 30000", records[0].CostFiat)
	}
	if records[1].CostFiat != 60_000 {
		t.Errorf("record[1].CostFiat = %d, want 60000", records[1].CostFiat)
	}
	if records[0].DisposedAt != nil {
		t.Error("unspent UTXO should have nil DisposedAt")
	}
	if records[0].Method != domain.MethodFIFO {
		t.Errorf("method = %q", records[0].Method)
	}
}

func TestFIFO_SpentWithPriceData(t *testing.T) {
	spendTimes := map[string]time.Time{"spend-tx": t3}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, true, "spend-tx"), // 0.01 BTC
	}
	// Acquired at 20 000 EUR/BTC = 2 000 000 cents → cost = 200 EUR = 20 000 cents
	// Disposed at 40 000 EUR/BTC = 4 000 000 cents → proceeds = 400 EUR = 40 000 cents
	// Gain = 200 EUR = 20 000 cents
	priceByTime := func(currency string, ts time.Time) int64 {
		if ts.IsZero() {
			return 0
		}
		if !ts.After(t2) {
			return 2_000_000 // 20 000 EUR/BTC
		}
		return 4_000_000 // 40 000 EUR/BTC
	}

	records := accounting.RunFIFO("wallet-1", utxos, spendTimes, priceByTime, "EUR")
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.CostFiat != 20_000 {
		t.Errorf("CostFiat = %d, want 20000", r.CostFiat)
	}
	if r.ProceedsFiat == nil || *r.ProceedsFiat != 40_000 {
		t.Errorf("ProceedsFiat = %v, want 40000", r.ProceedsFiat)
	}
	if r.GainFiat == nil || *r.GainFiat != 20_000 {
		t.Errorf("GainFiat = %v, want 20000", r.GainFiat)
	}
	if r.DisposedAt == nil || !r.DisposedAt.Equal(t3) {
		t.Errorf("DisposedAt = %v, want %v", r.DisposedAt, t3)
	}
}

func TestFIFO_NoPriceData(t *testing.T) {
	utxos := []domain.UTXO{utxo("tx1", 0, 500_000, t1, false, "")}
	records := accounting.RunFIFO("wallet-1", utxos, nil, noPrices, "EUR")
	if len(records) != 1 {
		t.Fatalf("expected 1 record")
	}
	if records[0].CostFiat != 0 {
		t.Errorf("CostFiat should be 0 when no price data, got %d", records[0].CostFiat)
	}
}

// --- AvgCost tests ---

func TestAvgCost_TwoAcquisitions(t *testing.T) {
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""), // 0.01 BTC at 20 000 EUR → 20 000 cents cost
		utxo("tx2", 0, 1_000_000, t2, false, ""), // 0.01 BTC at 40 000 EUR → 40 000 cents cost
	}
	priceByTime := func(currency string, ts time.Time) int64 {
		if !ts.After(t1) {
			return 2_000_000 // 20 000 EUR/BTC in cents
		}
		return 4_000_000 // 40 000 EUR/BTC in cents
	}
	records := accounting.RunAvgCost("wallet-1", utxos, nil, priceByTime, "EUR")
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].CostFiat != 20_000 {
		t.Errorf("records[0].CostFiat = %d, want 20000", records[0].CostFiat)
	}
	if records[1].CostFiat != 40_000 {
		t.Errorf("records[1].CostFiat = %d, want 40000", records[1].CostFiat)
	}
}

func TestAvgCost_DisposalUsesPoolAverage(t *testing.T) {
	// Acquire 1M sats at 20 000 EUR/BTC → cost = 20 000 cents.
	// Acquire 1M sats at 40 000 EUR/BTC → cost = 40 000 cents.
	// Pool before disposal: 2M sats, 60 000 cents total.
	// Dispose second 1M sats: avg cost = 60 000 * 1M / 2M = 30 000 cents.
	// Proceeds = 1M sats at 50 000 EUR/BTC = 50 000 cents. Gain = 20 000 cents.
	spendTimes := map[string]time.Time{"spend-tx": t3}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""),
		utxo("tx2", 0, 1_000_000, t2, true, "spend-tx"),
	}
	priceByTime := func(currency string, ts time.Time) int64 {
		switch {
		case !ts.After(t1):
			return 2_000_000 // 20 000 EUR/BTC
		case !ts.After(t2):
			return 4_000_000 // 40 000 EUR/BTC
		default:
			return 5_000_000 // 50 000 EUR/BTC at disposal
		}
	}
	records := accounting.RunAvgCost("wallet-1", utxos, spendTimes, priceByTime, "EUR")
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	r := records[1] // the spent one
	if r.CostFiat != 30_000 {
		t.Errorf("CostFiat = %d, want 30000 (pool average)", r.CostFiat)
	}
	if r.GainFiat == nil {
		t.Fatal("GainFiat should be set for disposed UTXO")
	}
	if *r.GainFiat != 20_000 {
		t.Errorf("GainFiat = %d, want 20000", *r.GainFiat)
	}
	if r.DisposedAt == nil {
		t.Fatal("DisposedAt should be set for disposed UTXO")
	}
}

func TestSatsToFiat_OneSatAtHighPrice(t *testing.T) {
	// 1 sat at 1 000 000 EUR/BTC (= 100 000 000 cents/BTC)
	// = 1 * 100_000_000 / 100_000_000 = 1 cent
	got := accounting.RunFIFO("w", []domain.UTXO{
		utxo("tx1", 0, 1, t1, false, ""),
	}, nil, fixedPrice(100_000_000), "EUR")
	if len(got) != 1 || got[0].CostFiat != 1 {
		t.Errorf("1 sat at 1M EUR/BTC should cost 1 cent, got %d", got[0].CostFiat)
	}
}

// --- LIFO tests ---

func TestLIFO_DisposalMatchesNewest(t *testing.T) {
	// Two acquisitions: tx1 at t1 (cheap), tx2 at t2 (expensive).
	// LIFO: disposal should be matched against tx2 (newest).
	spendTimes := map[string]time.Time{"spend-tx": t3}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""),          // unspent, cheap
		utxo("tx2", 0, 1_000_000, t2, true, "spend-tx"),   // spent
	}
	priceByTime := func(currency string, ts time.Time) int64 {
		if ts.IsZero() {
			return 0
		}
		if !ts.After(t1) {
			return 2_000_000 // 20 000 EUR/BTC — tx1 acquisition price
		}
		if !ts.After(t2) {
			return 4_000_000 // 40 000 EUR/BTC — tx2 acquisition price (LIFO uses this)
		}
		return 5_000_000 // 50 000 EUR/BTC — disposal price
	}
	records := accounting.RunLIFO("wallet-1", utxos, spendTimes, priceByTime, "EUR")
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	// Find the disposal record.
	var disposal *domain.CostBasisRecord
	for i := range records {
		if records[i].DisposedAt != nil {
			disposal = &records[i]
		}
	}
	if disposal == nil {
		t.Fatal("no disposal record found")
	}
	// Cost should reflect tx2's acquisition price (LIFO: newest = 40 000 cents cost).
	if disposal.CostFiat != 40_000 {
		t.Errorf("LIFO disposal CostFiat = %d, want 40000 (newest acquisition)", disposal.CostFiat)
	}
	if disposal.ProceedsFiat == nil || *disposal.ProceedsFiat != 50_000 {
		t.Errorf("ProceedsFiat = %v, want 50000", disposal.ProceedsFiat)
	}
	if disposal.GainFiat == nil || *disposal.GainFiat != 10_000 {
		t.Errorf("GainFiat = %v, want 10000", disposal.GainFiat)
	}
}

func TestLIFO_UnspentOnly(t *testing.T) {
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""),
		utxo("tx2", 0, 2_000_000, t2, false, ""),
	}
	records := accounting.RunLIFO("wallet-1", utxos, nil, fixedPrice(3_000_000), "EUR")
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	for _, r := range records {
		if r.DisposedAt != nil {
			t.Error("unspent UTXOs should have nil DisposedAt")
		}
		if r.Method != domain.MethodLIFO {
			t.Errorf("method = %q, want lifo", r.Method)
		}
	}
}

// --- HIFO tests ---

func TestHIFO_DisposalMatchesHighestCost(t *testing.T) {
	// Two unspent acquisitions in pool: cheap one (20 000 EUR) and expensive one (40 000 EUR).
	// One disposal — HIFO should match expensive acquisition (higher cost = lower gain).
	spendTimes := map[string]time.Time{"spend-tx": t3}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""),         // cheap, unspent
		utxo("tx2", 0, 1_000_000, t2, false, ""),         // expensive, unspent
		utxo("tx3", 0, 1_000_000, t2, true, "spend-tx"),  // disposal
	}
	priceByTime := func(currency string, ts time.Time) int64 {
		if ts.IsZero() {
			return 0
		}
		if !ts.After(t1) {
			return 2_000_000 // 20 000 EUR — cheap
		}
		if !ts.After(t2) {
			return 4_000_000 // 40 000 EUR — expensive
		}
		return 5_000_000 // disposal price
	}
	records := accounting.RunHIFO("wallet-1", utxos, spendTimes, priceByTime, "EUR")
	var disposal *domain.CostBasisRecord
	for i := range records {
		if records[i].DisposedAt != nil {
			disposal = &records[i]
		}
	}
	if disposal == nil {
		t.Fatal("no disposal record found")
	}
	// HIFO picks highest cost in pool (40 000 cents).
	if disposal.CostFiat != 40_000 {
		t.Errorf("HIFO disposal CostFiat = %d, want 40000", disposal.CostFiat)
	}
	if disposal.GainFiat == nil || *disposal.GainFiat != 10_000 {
		t.Errorf("GainFiat = %v, want 10000", disposal.GainFiat)
	}
}

func TestHIFO_Method(t *testing.T) {
	utxos := []domain.UTXO{utxo("tx1", 0, 1_000_000, t1, false, "")}
	records := accounting.RunHIFO("wallet-1", utxos, nil, fixedPrice(3_000_000), "EUR")
	if len(records) != 1 || records[0].Method != domain.MethodHIFO {
		t.Errorf("method = %q, want hifo", records[0].Method)
	}
}

// --- SpecificID tests ---

func TestSpecificID_ExplicitSelection(t *testing.T) {
	// Two acquisitions, one disposal explicitly mapped to the expensive acquisition.
	spendTimes := map[string]time.Time{"spend-tx": t3}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""),         // cheap, unspent (acquisition)
		utxo("tx2", 0, 1_000_000, t2, false, ""),         // expensive, unspent (acquisition)
		utxo("tx3", 0, 1_000_000, t2, true, "spend-tx"),  // disposal
	}
	priceByTime := func(currency string, ts time.Time) int64 {
		if ts.IsZero() {
			return 0
		}
		if !ts.After(t1) {
			return 2_000_000
		}
		if !ts.After(t2) {
			return 4_000_000
		}
		return 5_000_000
	}
	// Explicitly select expensive acquisition (tx2:0) for disposal tx3:0.
	selections := accounting.SpecificIDSelection{"tx3:0": "tx2:0"}
	records := accounting.RunSpecificID("wallet-1", utxos, spendTimes, priceByTime, "EUR", selections)

	var disposal *domain.CostBasisRecord
	for i := range records {
		if records[i].DisposedAt != nil {
			disposal = &records[i]
		}
	}
	if disposal == nil {
		t.Fatal("no disposal record found")
	}
	if disposal.CostFiat != 40_000 {
		t.Errorf("SpecificID disposal CostFiat = %d, want 40000 (explicit selection)", disposal.CostFiat)
	}
}

func TestSpecificID_FallsBackToFIFO(t *testing.T) {
	// No selections map — should fall back to FIFO (oldest first).
	spendTimes := map[string]time.Time{"spend-tx": t3}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""),         // oldest, cheapest
		utxo("tx2", 0, 1_000_000, t2, false, ""),         // newer, expensive
		utxo("tx3", 0, 1_000_000, t2, true, "spend-tx"),  // disposal
	}
	priceByTime := func(currency string, ts time.Time) int64 {
		if ts.IsZero() {
			return 0
		}
		if !ts.After(t1) {
			return 2_000_000 // cheap — FIFO picks this
		}
		if !ts.After(t2) {
			return 4_000_000
		}
		return 5_000_000
	}
	records := accounting.RunSpecificID("wallet-1", utxos, spendTimes, priceByTime, "EUR", nil)
	var disposal *domain.CostBasisRecord
	for i := range records {
		if records[i].DisposedAt != nil {
			disposal = &records[i]
		}
	}
	if disposal == nil {
		t.Fatal("no disposal record found")
	}
	// FIFO fallback: oldest = tx1 at 20 000 EUR = 20 000 cents.
	if disposal.CostFiat != 20_000 {
		t.Errorf("SpecificID FIFO fallback CostFiat = %d, want 20000", disposal.CostFiat)
	}
}

func TestSpecificID_Method(t *testing.T) {
	utxos := []domain.UTXO{utxo("tx1", 0, 1_000_000, t1, false, "")}
	records := accounting.RunSpecificID("wallet-1", utxos, nil, fixedPrice(3_000_000), "EUR", nil)
	if len(records) != 1 || records[0].Method != domain.MethodSpecificID {
		t.Errorf("method = %q, want specificid", records[0].Method)
	}
}

// --- Section 104 tests ---

func TestSection104_UnspentOnly(t *testing.T) {
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, t1, false, ""),
		utxo("tx2", 0, 2_000_000, t2, false, ""),
	}
	price := fixedPrice(3_000_000) // 30,000 EUR/BTC
	records := accounting.RunSection104("wallet-1", utxos, nil, price, "EUR")
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	for _, r := range records {
		if r.DisposedAt != nil {
			t.Errorf("expected no disposal, got %v", r.DisposedAt)
		}
		if r.Method != domain.MethodSection104 {
			t.Errorf("expected section104 method, got %s", r.Method)
		}
	}
}

func TestSection104_Pool(t *testing.T) {
	// Acquire 1M sats then spend them — no same-day or 30-day match.
	// Pool cost = full acquisition cost.
	acqTime := t1
	dispTime := t3 // 2 years later
	spendTimes := map[string]time.Time{"spend1": dispTime}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, acqTime, true, "spend1"),
	}
	// Acquisition price: 30,000 EUR/BTC → cost = 30,000 cents
	// Disposal price: 60,000 EUR/BTC → proceeds = 60,000 cents
	// Gain = 60,000 - 30,000 = 30,000 cents
	priceFn := func(currency string, t time.Time) int64 {
		if t.Equal(acqTime) {
			return 3_000_000 // 30,000 EUR/BTC
		}
		return 6_000_000 // 60,000 EUR/BTC
	}
	records := accounting.RunSection104("wallet-1", utxos, spendTimes, priceFn, "EUR")
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.DisposedAt == nil {
		t.Fatal("expected disposal")
	}
	if *r.GainFiat != 30_000 {
		t.Errorf("GainFiat = %d, want 30000", *r.GainFiat)
	}
}

func TestSection104_SameDayRule(t *testing.T) {
	// Acquire and dispose on the same day — same-day rule applies.
	// The disposal should be matched against the same-day acquisition.
	sameDay := t1
	spendTimes := map[string]time.Time{"spend1": sameDay}
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, sameDay, true, "spend1"),
	}
	// Fixed price 50,000 EUR/BTC → cost = proceeds = 50,000 cents → gain = 0
	price := fixedPrice(5_000_000)
	records := accounting.RunSection104("wallet-1", utxos, spendTimes, price, "EUR")
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if *records[0].GainFiat != 0 {
		t.Errorf("same-day gain = %d, want 0 (cost == proceeds)", *records[0].GainFiat)
	}
}

func TestSection104_30DayRule(t *testing.T) {
	// Dispose on day 0, acquire 10 days later — 30-day rule should match.
	// Cost from the later acquisition should be used (not pool average).
	dispTime := t1
	acqTime := t1.AddDate(0, 0, 10)
	spendTimes := map[string]time.Time{"spend1": dispTime}

	// Disposal UTXO acquired well before disposal.
	oldAcq := t1.AddDate(-1, 0, 0)
	utxos := []domain.UTXO{
		utxo("tx1", 0, 1_000_000, oldAcq, true, "spend1"),  // acquired 1yr ago, disposed on day 0
		utxo("tx2", 0, 1_000_000, acqTime, false, ""),       // acquired 10 days after disposal
	}
	// Acquisition prices differ to verify which one is used.
	priceFn := func(currency string, t time.Time) int64 {
		switch {
		case t.Equal(oldAcq):
			return 2_000_000 // 20,000 EUR/BTC (old acquisition)
		case t.Equal(acqTime):
			return 4_000_000 // 40,000 EUR/BTC (30-day replacement)
		case t.Equal(dispTime):
			return 5_000_000 // 50,000 EUR/BTC (disposal proceeds)
		}
		return 0
	}
	records := accounting.RunSection104("wallet-1", utxos, spendTimes, priceFn, "EUR")

	// Find the disposal record (tx1).
	var dispRecord *domain.CostBasisRecord
	for i := range records {
		if records[i].Txid == "tx1" && records[i].DisposedAt != nil {
			dispRecord = &records[i]
		}
	}
	if dispRecord == nil {
		t.Fatal("disposal record for tx1 not found")
	}
	// 30-day rule: cost = 40,000 cents (from the 10-day-later acquisition at 40,000 EUR/BTC)
	// proceeds = 50,000 cents, gain = 10,000 cents
	if dispRecord.CostFiat != 40_000 {
		t.Errorf("30-day cost = %d, want 40000", dispRecord.CostFiat)
	}
	if *dispRecord.GainFiat != 10_000 {
		t.Errorf("30-day gain = %d, want 10000", *dispRecord.GainFiat)
	}
}
