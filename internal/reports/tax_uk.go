package reports

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"
)

// UKPoolRow records a single pool movement (acquisition or disposal into/from the pool).
type UKPoolRow struct {
	Date        time.Time
	Event       string // "Acquisition" | "Disposal (same-day)" | "Disposal (30-day)" | "Disposal (pool)"
	Sats        int64
	CostGBP     int64 // cents — the cost attributed to this event
	PoolSats    int64 // pool total after this event
	PoolCostGBP int64 // pool allowable cost after this event (cents)
}

// UKDisposalRow is one disposal event for HMRC SA108.
type UKDisposalRow struct {
	Date             time.Time
	Txid             string
	Vout             int
	ProceedsGBP      int64  // cents
	AllowableCostGBP int64  // cents
	GainGBP          int64  // cents (negative = loss)
	MatchingRule     string // "same-day" | "30-day" | "pool"
}

// UKTaxData is the full UK CGT dataset for one tax year.
type UKTaxData struct {
	TaxYear           string // e.g. "2024/25"
	YearStart         int    // e.g. 2024
	PoolRows          []UKPoolRow
	Disposals         []UKDisposalRow
	TotalGainsGBP     int64 // cents (sum of positive gains)
	TotalLossesGBP    int64 // cents (sum of negative, stored negative)
	NetGainGBP        int64 // cents
	AnnualExemptGBP   int64 // cents — £3,000 for 2024/25 onward
}

// UKTaxYear returns "YYYY/YY" and the start year for a UK tax year beginning April 6, year.
func UKTaxYear(year int) (string, int) {
	short := fmt.Sprintf("%02d", (year+1)%100)
	return fmt.Sprintf("%d/%s", year, short), year
}

// UKTaxYearBounds returns [Apr 6 year, Apr 5 year+1] as UTC times.
func UKTaxYearBounds(year int) (time.Time, time.Time) {
	start := time.Date(year, 4, 6, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, 4, 5, 23, 59, 59, 0, time.UTC)
	return start, end
}

// WriteTaxUKCSV writes the UK HMRC CGT Section 104 report as CSV.
func WriteTaxUKCSV(w io.Writer, d UKTaxData) error {
	cw := csv.NewWriter(w)

	_ = cw.Write([]string{fmt.Sprintf("Capital Gains Tax — Cryptoassets — Tax Year %s", d.TaxYear)})
	_ = cw.Write([]string{"For use with HMRC Self Assessment SA108"})
	_ = cw.Write([]string{"Statutory basis: TCGA 1992 s.104/105/106A (Section 104 Pool)"})
	_ = cw.Write([]string{})

	// Disposals section.
	_ = cw.Write([]string{"--- DISPOSALS ---"})
	_ = cw.Write([]string{"Date", "UTXO", "Proceeds (GBP)", "Allowable Cost (GBP)", "Gain/Loss (GBP)", "Matching Rule"})
	for _, r := range d.Disposals {
		_ = cw.Write([]string{
			fmtDate(r.Date),
			fmt.Sprintf("%s:%d", r.Txid, r.Vout),
			fmtFiat(r.ProceedsGBP),
			fmtFiat(r.AllowableCostGBP),
			fmtFiat(r.GainGBP),
			r.MatchingRule,
		})
	}

	_ = cw.Write([]string{})
	_ = cw.Write([]string{"--- SECTION 104 POOL MOVEMENTS ---"})
	_ = cw.Write([]string{"Date", "Event", "Sats", "Cost (GBP)", "Pool Sats After", "Pool Cost After (GBP)"})
	for _, r := range d.PoolRows {
		_ = cw.Write([]string{
			fmtDate(r.Date),
			r.Event,
			fmt.Sprintf("%d", r.Sats),
			fmtFiat(r.CostGBP),
			fmt.Sprintf("%d", r.PoolSats),
			fmtFiat(r.PoolCostGBP),
		})
	}

	_ = cw.Write([]string{})
	_ = cw.Write([]string{"--- SUMMARY ---"})
	_ = cw.Write([]string{"Total gains (GBP)", fmtFiat(d.TotalGainsGBP)})
	_ = cw.Write([]string{"Total losses (GBP)", fmtFiat(d.TotalLossesGBP)})
	_ = cw.Write([]string{"Net gain/loss (GBP)", fmtFiat(d.NetGainGBP)})
	_ = cw.Write([]string{"Annual exempt amount (GBP)", fmtFiat(d.AnnualExemptGBP)})
	taxable := d.NetGainGBP - d.AnnualExemptGBP
	if taxable < 0 {
		taxable = 0
	}
	_ = cw.Write([]string{"Taxable gain (GBP)", fmtFiat(taxable)})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Note: Enter net gain on SA108 box 7 (or box 8 if a net loss). Enter total proceeds in box 6."})

	cw.Flush()
	return cw.Error()
}

// WriteTaxUKPDF writes the UK HMRC CGT Section 104 report as PDF.
func WriteTaxUKPDF(w io.Writer, d UKTaxData) error {
	m, err := newPDF("", fmt.Sprintf("CGT Cryptoassets %s", d.TaxYear), "GBP")
	if err != nil {
		return err
	}

	taxable := d.NetGainGBP - d.AnnualExemptGBP
	if taxable < 0 {
		taxable = 0
	}
	m.AddRows(
		summaryBox("Tax year:", d.TaxYear),
		summaryBox("Statutory basis:", "TCGA 1992 s.104/105/106A"),
		summaryBox("Total gains:", fmtFiat(d.TotalGainsGBP)+" GBP"),
		summaryBox("Total losses:", fmtFiat(d.TotalLossesGBP)+" GBP"),
		summaryBox("Net gain/loss:", fmtFiat(d.NetGainGBP)+" GBP"),
		summaryBox("Annual exempt amount:", fmtFiat(d.AnnualExemptGBP)+" GBP"),
		summaryBox("Taxable gain:", fmtFiat(taxable)+" GBP"),
	)
	m.AddRow(6)

	// Disposals table.
	m.AddRows(tableHeader(
		[]string{"Date", "UTXO", "Proceeds (£)", "Cost (£)", "Gain/Loss (£)", "Rule"},
		[]uint{2, 3, 2, 2, 2, 1},
	))
	for i, r := range d.Disposals {
		label := fmt.Sprintf("%s:%d", r.Txid, r.Vout)
		if len(label) > 24 {
			label = label[:12] + "…"
		}
		m.AddRows(tableRow(
			[]string{fmtDate(r.Date), label, fmtFiat(r.ProceedsGBP), fmtFiat(r.AllowableCostGBP), fmtFiat(r.GainGBP), r.MatchingRule},
			[]uint{2, 3, 2, 2, 2, 1},
			i%2 == 1,
			[]align.Type{align.Left, align.Left, align.Right, align.Right, align.Right, align.Center},
		))
	}

	m.AddRow(6)

	// Pool movements table.
	m.AddRows(summaryBox("Section 104 Pool Movements", ""))
	m.AddRows(tableHeader(
		[]string{"Date", "Event", "Sats", "Cost (£)", "Pool Sats", "Pool Cost (£)"},
		[]uint{2, 3, 2, 1, 2, 2},
	))
	for i, r := range d.PoolRows {
		m.AddRows(tableRow(
			[]string{
				fmtDate(r.Date),
				r.Event,
				fmt.Sprintf("%d", r.Sats),
				fmtFiat(r.CostGBP),
				fmt.Sprintf("%d", r.PoolSats),
				fmtFiat(r.PoolCostGBP),
			},
			[]uint{2, 3, 2, 1, 2, 2},
			i%2 == 1,
			[]align.Type{align.Left, align.Left, align.Right, align.Right, align.Right, align.Right},
		))
	}

	m.AddRow(8)
	m.AddRows(summaryBox("Note:", "Enter net gain on SA108 box 7 (box 8 if a net loss). Enter total proceeds in box 6."))

	doc, err := m.Generate()
	if err != nil {
		return err
	}
	_, err = w.Write(doc.GetBytes())
	return err
}
