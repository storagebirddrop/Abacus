package reports

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

func mustParseCSV(t *testing.T, b []byte) [][]string {
	t.Helper()
	// The tax CSVs contain ragged rows (title lines, blank lines), so disable
	// the field-count check.
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v", err)
	}
	return recs
}

// findRow returns the first CSV record whose first cell equals key, or nil.
func findRow(recs [][]string, key string) []string {
	for _, rec := range recs {
		if len(rec) > 0 && rec[0] == key {
			return rec
		}
	}
	return nil
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// --- formatting helpers ------------------------------------------------------

func TestFormatHelpers(t *testing.T) {
	if got := fmtBTC(123_456_789); got != "1.23456789" {
		t.Errorf("fmtBTC(123456789) = %q, want 1.23456789", got)
	}
	if got := fmtBTC(0); got != "0.00000000" {
		t.Errorf("fmtBTC(0) = %q, want 0.00000000", got)
	}
	if got := fmtFiat(60_000); got != "600.00" {
		t.Errorf("fmtFiat(60000) = %q, want 600.00", got)
	}
	if got := fmtFiat(-12_345); got != "-123.45" {
		t.Errorf("fmtFiat(-12345) = %q, want -123.45", got)
	}
	if got := fmtDate(date(2024, time.March, 9)); got != "2024-03-09" {
		t.Errorf("fmtDate = %q, want 2024-03-09", got)
	}
	if got := fmtDate(time.Time{}); got != "" {
		t.Errorf("fmtDate(zero) = %q, want empty", got)
	}
}

// --- generic CSV reports -----------------------------------------------------

func TestWriteTransactionsCSV(t *testing.T) {
	var buf bytes.Buffer
	rows := []TransactionRow{
		{Date: date(2024, time.January, 2), Txid: "abc", FeeSats: 250, Confirmed: true},
		{Date: date(2024, time.February, 3), Txid: "def", FeeSats: 0, Confirmed: false},
	}
	if err := WriteTransactionsCSV(&buf, rows); err != nil {
		t.Fatalf("WriteTransactionsCSV: %v", err)
	}
	recs := mustParseCSV(t, buf.Bytes())
	if got := recs[0]; got[0] != "Date" || got[2] != "Fee (sats)" {
		t.Errorf("header = %v", got)
	}
	if recs[1][1] != "abc" || recs[1][2] != "250" || recs[1][3] != "Yes" {
		t.Errorf("row1 = %v", recs[1])
	}
	if recs[2][3] != "No" {
		t.Errorf("row2 confirmed = %v, want No", recs[2][3])
	}
}

func TestWritePnLCSV(t *testing.T) {
	var buf bytes.Buffer
	rows := []PnLRow{{
		Txid: "tx1", Vout: 0,
		AcquiredAt: date(2023, time.January, 1), DisposedAt: date(2024, time.January, 1),
		CostFiat: 30_000, ProceedsFiat: 50_000, GainFiat: 20_000,
		Method: "fifo", Currency: "EUR",
	}}
	if err := WritePnLCSV(&buf, rows, "EUR"); err != nil {
		t.Fatalf("WritePnLCSV: %v", err)
	}
	recs := mustParseCSV(t, buf.Bytes())
	if !strings.Contains(recs[0][4], "EUR") {
		t.Errorf("header missing currency: %v", recs[0])
	}
	got := recs[1]
	if got[0] != "tx1:0" || got[3] != "fifo" || got[4] != "300.00" || got[6] != "200.00" {
		t.Errorf("pnl row = %v", got)
	}
}

func TestWriteBalanceSheetCSV(t *testing.T) {
	var buf bytes.Buffer
	rows := []BalanceRow{{
		Txid: "tx1", Vout: 1, Sats: 100_000_000, Address: "bc1qxyz",
		AcquiredAt: date(2024, time.June, 1), CostFiat: 6_000_000, Currency: "EUR",
	}}
	if err := WriteBalanceSheetCSV(&buf, rows, "EUR"); err != nil {
		t.Fatalf("WriteBalanceSheetCSV: %v", err)
	}
	recs := mustParseCSV(t, buf.Bytes())
	got := recs[1]
	if got[0] != "tx1:1" || got[1] != "bc1qxyz" || got[3] != "1.00000000" || got[4] != "60000.00" {
		t.Errorf("balance row = %v", got)
	}
}

// --- Netherlands Box 3 -------------------------------------------------------

func TestNLPeildatum(t *testing.T) {
	p := NLPeildatum(2024)
	if p.Year() != 2024 || p.Month() != time.January || p.Day() != 1 {
		t.Errorf("NLPeildatum(2024) = %v, want 2024-01-01", p)
	}
}

func TestWriteTaxNLCSV_Value(t *testing.T) {
	var buf bytes.Buffer
	d := NLTaxData{
		Year:         2024,
		PeildatumBTC: 100_000_000, // 1 BTC
		PriceEUR:     6_000_000,   // €60,000.00
		ValueEUR:     6_000_000,
		Holdings:     []NLHoldingRow{{Txid: "tx1", Vout: 0, Sats: 100_000_000}},
	}
	if err := WriteTaxNLCSV(&buf, d); err != nil {
		t.Fatalf("WriteTaxNLCSV: %v", err)
	}
	recs := mustParseCSV(t, buf.Bytes())
	total := findRow(recs, "Totaal")
	if total == nil {
		t.Fatalf("no Totaal row in %v", recs)
	}
	// Totaal | Aantal BTC | Koers | Waarde
	if total[1] != "1.00000000" || total[3] != "60000.00" {
		t.Errorf("Totaal row = %v, want BTC 1.0 / value 60000.00", total)
	}
}

func TestWriteTaxNLCSV_UnknownPrice(t *testing.T) {
	var buf bytes.Buffer
	d := NLTaxData{
		Year:         2024,
		PeildatumBTC: 100_000_000,
		PriceEUR:     0, // unknown
		Holdings:     []NLHoldingRow{{Txid: "tx1", Vout: 0, Sats: 100_000_000}},
	}
	if err := WriteTaxNLCSV(&buf, d); err != nil {
		t.Fatalf("WriteTaxNLCSV: %v", err)
	}
	if !strings.Contains(buf.String(), "onbekend") {
		t.Errorf("expected 'onbekend' when price unknown, got:\n%s", buf.String())
	}
}

// --- Germany §23 EStG --------------------------------------------------------

func TestBuildDESummary_FreigrenzeBoundary(t *testing.T) {
	// The Freigrenze is year-dependent: €600 through 2023, €1,000 from 2024.
	cases := []struct {
		name        string
		year        int
		gain        int64
		wantGreift  bool
		wantNet     int64
	}{
		{"2023 one cent under €600 → exempt", 2023, 59_999, true, 0},
		{"2023 exactly at €600 → taxable (weniger als)", 2023, 60_000, false, 60_000},
		{"2023 one cent over €600 → taxable", 2023, 60_001, false, 60_001},
		{"2024 €600 now below €1,000 → exempt", 2024, 60_000, true, 0},
		{"2024 one cent under €1,000 → exempt", 2024, 99_999, true, 0},
		{"2024 exactly at €1,000 → taxable (weniger als)", 2024, 100_000, false, 100_000},
		{"2024 one cent over €1,000 → taxable", 2024, 100_001, false, 100_001},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := BuildDESummary(c.year, []DETaxRow{{GainFiat: c.gain, HoldingDays: 100}})
			if s.FreigreifenGilt != c.wantGreift || s.NetTaxableCents != c.wantNet {
				t.Errorf("greift=%v net=%d, want greift=%v net=%d",
					s.FreigreifenGilt, s.NetTaxableCents, c.wantGreift, c.wantNet)
			}
		})
	}
}

func TestBuildDESummary_TaxFreeExcluded(t *testing.T) {
	// A long-held (tax-free) row with a huge gain must not count toward the
	// taxable total; only the short-term row does. Use 2023 + a gain above the
	// €600 threshold so it remains taxable.
	s := BuildDESummary(2023, []DETaxRow{
		{GainFiat: 5_000_000, HoldingDays: 800, TaxFree: true}, // excluded
		{GainFiat: 70_000, HoldingDays: 30, TaxFree: false},    // counted, > €600
	})
	if s.TaxableGainCents != 70_000 {
		t.Errorf("TaxableGainCents = %d, want 70000 (tax-free row excluded)", s.TaxableGainCents)
	}
	if s.FreigreifenGilt || s.NetTaxableCents != 70_000 {
		t.Errorf("net = %d greift=%v, want 70000/false", s.NetTaxableCents, s.FreigreifenGilt)
	}
}

func TestBuildDESummary_LossesOffsetGains(t *testing.T) {
	// §23 permits offsetting losses against gains within the same type.
	// 2024 threshold is €1,000; net 160_000 (€1,600) stays taxable.
	s := BuildDESummary(2024, []DETaxRow{
		{GainFiat: 200_000, HoldingDays: 10, TaxFree: false},
		{GainFiat: -40_000, HoldingDays: 10, TaxFree: false},
	})
	if s.TaxableGainCents != 160_000 {
		t.Errorf("TaxableGainCents = %d, want 160000 (loss offsets gain)", s.TaxableGainCents)
	}
	if s.NetTaxableCents != 160_000 {
		t.Errorf("NetTaxableCents = %d, want 160000", s.NetTaxableCents)
	}
}

func TestBuildDESummary_NetLossNotTaxable(t *testing.T) {
	// A net loss across short-term disposals yields zero taxable, and the
	// Freigrenze is not reported as "applying" (it's a loss, not an exemption).
	s := BuildDESummary(2024, []DETaxRow{
		{GainFiat: 30_000, HoldingDays: 10, TaxFree: false},
		{GainFiat: -90_000, HoldingDays: 10, TaxFree: false},
	})
	if s.TaxableGainCents != -60_000 {
		t.Errorf("TaxableGainCents = %d, want -60000", s.TaxableGainCents)
	}
	if s.NetTaxableCents != 0 || s.FreigreifenGilt {
		t.Errorf("net=%d greift=%v, want net=0 greift=false", s.NetTaxableCents, s.FreigreifenGilt)
	}
}

// --- United States Form 8949 -------------------------------------------------

func TestUsHoldingDays(t *testing.T) {
	acq := date(2023, time.January, 1)
	cases := []struct {
		name       string
		disposed   time.Time
		wantDays   int
		wantLong   bool
	}{
		{"exactly 365 days is short-term", acq.AddDate(0, 0, 365), 365, false},
		{"366 days is long-term", acq.AddDate(0, 0, 366), 366, true},
		{"same day", acq, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			days, long := UsHoldingDays(acq, c.disposed)
			if days != c.wantDays || long != c.wantLong {
				t.Errorf("UsHoldingDays = (%d,%v), want (%d,%v)", days, long, c.wantDays, c.wantLong)
			}
		})
	}
}

func TestBuildUSRows_SplitAndNet(t *testing.T) {
	st, lt, stNet, ltNet := BuildUSRows([]USTaxRow{
		{Description: "a", GainUSD: 10_000, LongTerm: false},
		{Description: "b", GainUSD: -3_000, LongTerm: false},
		{Description: "c", GainUSD: 50_000, LongTerm: true},
	})
	if len(st) != 2 || len(lt) != 1 {
		t.Fatalf("split = %d short / %d long, want 2/1", len(st), len(lt))
	}
	if stNet != 7_000 {
		t.Errorf("stNet = %d, want 7000 (10000 - 3000)", stNet)
	}
	if ltNet != 50_000 {
		t.Errorf("ltNet = %d, want 50000", ltNet)
	}
}

// --- United Kingdom CGT / Section 104 ----------------------------------------

func TestUKTaxYear(t *testing.T) {
	s, start := UKTaxYear(2024)
	if s != "2024/25" || start != 2024 {
		t.Errorf("UKTaxYear(2024) = (%q,%d), want (2024/25,2024)", s, start)
	}
}

func TestUKTaxYearBounds(t *testing.T) {
	start, end := UKTaxYearBounds(2024)
	if !start.Equal(date(2024, time.April, 6)) {
		t.Errorf("start = %v, want 2024-04-06", start)
	}
	if end.Year() != 2025 || end.Month() != time.April || end.Day() != 5 {
		t.Errorf("end = %v, want 2025-04-05", end)
	}
}

func TestWriteTaxUKCSV_TaxableFlooredAtZero(t *testing.T) {
	// Net gain below the annual exempt amount → taxable gain floored at 0.
	var buf bytes.Buffer
	d := UKTaxData{
		TaxYear: "2024/25", YearStart: 2024,
		NetGainGBP: 50_000, AnnualExemptGBP: 300_000, // £500 net, £3,000 exempt
	}
	if err := WriteTaxUKCSV(&buf, d); err != nil {
		t.Fatalf("WriteTaxUKCSV: %v", err)
	}
	recs := mustParseCSV(t, buf.Bytes())
	row := findRow(recs, "Taxable gain (GBP)")
	if row == nil || row[1] != "0.00" {
		t.Errorf("taxable gain row = %v, want 0.00", row)
	}
}

func TestWriteTaxUKCSV_TaxableAboveExempt(t *testing.T) {
	var buf bytes.Buffer
	d := UKTaxData{
		TaxYear: "2024/25", YearStart: 2024,
		NetGainGBP: 1_000_000, AnnualExemptGBP: 300_000, // £10,000 net, £3,000 exempt
	}
	if err := WriteTaxUKCSV(&buf, d); err != nil {
		t.Fatalf("WriteTaxUKCSV: %v", err)
	}
	recs := mustParseCSV(t, buf.Bytes())
	row := findRow(recs, "Taxable gain (GBP)")
	if row == nil || row[1] != "7000.00" {
		t.Errorf("taxable gain row = %v, want 7000.00", row)
	}
}

// --- PDF smoke tests ---------------------------------------------------------

func assertPDF(t *testing.T, b []byte, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("PDF generation error: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("PDF is empty")
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Errorf("output does not start with %%PDF header: %q", b[:min(8, len(b))])
	}
}

func TestTaxPDFsGenerate(t *testing.T) {
	t.Run("NL", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteTaxNLPDF(&buf, NLTaxData{
			Year: 2024, PeildatumBTC: 100_000_000, PriceEUR: 6_000_000, ValueEUR: 6_000_000,
			Holdings: []NLHoldingRow{{Txid: "tx1", Vout: 0, Sats: 100_000_000}},
		})
		assertPDF(t, buf.Bytes(), err)
	})
	t.Run("DE", func(t *testing.T) {
		var buf bytes.Buffer
		s := BuildDESummary(2024, []DETaxRow{
			{Txid: "tx1", Vout: 0, AcquiredAt: date(2023, 1, 1), DisposedAt: date(2024, 6, 1),
				HoldingDays: 30, CostFiat: 30_000, ProceedsFiat: 50_000, GainFiat: 20_000},
		})
		err := WriteTaxDEPDF(&buf, s)
		assertPDF(t, buf.Bytes(), err)
	})
	t.Run("UK", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteTaxUKPDF(&buf, UKTaxData{
			TaxYear: "2024/25", YearStart: 2024,
			Disposals: []UKDisposalRow{{Date: date(2024, 6, 1), Txid: "tx1", Vout: 0,
				ProceedsGBP: 50_000, AllowableCostGBP: 30_000, GainGBP: 20_000, MatchingRule: "pool"}},
			NetGainGBP: 20_000, AnnualExemptGBP: 300_000,
		})
		assertPDF(t, buf.Bytes(), err)
	})
	t.Run("US", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteTaxUSPDF(&buf, USTaxData{
			Year: 2024, Currency: "USD",
			ShortTerm: []USTaxRow{{Description: "BTC tx1:0", AcquiredAt: date(2024, 1, 1),
				DisposedAt: date(2024, 6, 1), HoldingDays: 152, ProceedsUSD: 50_000, CostUSD: 30_000, GainUSD: 20_000}},
			STNetGain: 20_000, TotalNetGain: 20_000,
		})
		assertPDF(t, buf.Bytes(), err)
	})
}
