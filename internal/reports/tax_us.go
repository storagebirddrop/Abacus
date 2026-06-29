package reports

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"
)

// USTaxRow is one line on IRS Form 8949.
type USTaxRow struct {
	Description string    // e.g. "BTC txid:vout"
	AcquiredAt  time.Time
	DisposedAt  time.Time
	HoldingDays int
	ProceedsUSD int64 // cents
	CostUSD     int64 // cents
	GainUSD     int64 // cents (negative = loss)
	LongTerm    bool  // held > 365 days → Part II
}

// USTaxData is the full Form 8949 dataset for one calendar year.
type USTaxData struct {
	Year          int
	Currency      string // currency of stored records (warn if not USD)
	ShortTerm     []USTaxRow // Part I — held ≤ 365 days
	LongTerm      []USTaxRow // Part II — held > 365 days
	STNetGain     int64      // cents — net short-term gain/loss
	LTNetGain     int64      // cents — net long-term gain/loss
	TotalNetGain  int64      // cents — combined
}

// WriteTaxUSCSV writes the US IRS Form 8949 report as CSV.
func WriteTaxUSCSV(w io.Writer, d USTaxData) error {
	cw := csv.NewWriter(w)

	_ = cw.Write([]string{fmt.Sprintf("Form 8949 — Sales and Other Dispositions of Capital Assets — %d", d.Year)})
	_ = cw.Write([]string{"IRS Notice 2014-21: Bitcoin is property. Each disposal is a capital gain/loss event."})
	if d.Currency != "" && d.Currency != "USD" {
		_ = cw.Write([]string{fmt.Sprintf("WARNING: Cost basis records are in %s. IRS requires USD amounts. Convert using the exchange rate on each transaction date.", d.Currency)})
	}
	_ = cw.Write([]string{})

	writeF8949Part := func(label string, rows []USTaxRow) {
		_ = cw.Write([]string{label})
		_ = cw.Write([]string{
			"(a) Description",
			"(b) Date Acquired",
			"(c) Date Sold",
			"(d) Proceeds",
			"(e) Cost or Other Basis",
			"(f) Adjustment Code",
			"(h) Gain or (Loss)",
		})
		for _, r := range rows {
			_ = cw.Write([]string{
				r.Description,
				fmtDate(r.AcquiredAt),
				fmtDate(r.DisposedAt),
				fmtFiat(r.ProceedsUSD),
				fmtFiat(r.CostUSD),
				"",
				fmtFiat(r.GainUSD),
			})
		}
		_ = cw.Write([]string{})
	}

	writeF8949Part(
		fmt.Sprintf("Part I — Short-Term (held ≤ 365 days) — %d disposals", len(d.ShortTerm)),
		d.ShortTerm,
	)
	writeF8949Part(
		fmt.Sprintf("Part II — Long-Term (held > 365 days) — %d disposals", len(d.LongTerm)),
		d.LongTerm,
	)

	_ = cw.Write([]string{"--- Schedule D Summary ---"})
	_ = cw.Write([]string{"Short-term net gain/loss", fmtFiat(d.STNetGain)})
	_ = cw.Write([]string{"Long-term net gain/loss", fmtFiat(d.LTNetGain)})
	_ = cw.Write([]string{"Combined net gain/loss", fmtFiat(d.TotalNetGain)})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Note: Report short-term amounts on Schedule D Part I; long-term on Part II."})
	_ = cw.Write([]string{"Wash sale rules do not currently apply to cryptocurrency."})

	cw.Flush()
	return cw.Error()
}

// WriteTaxUSPDF writes the US IRS Form 8949 report as PDF.
func WriteTaxUSPDF(w io.Writer, d USTaxData) error {
	m, err := newPDF("", fmt.Sprintf("Form 8949 — %d", d.Year), d.Currency)
	if err != nil {
		return err
	}

	m.AddRows(
		summaryBox("Tax year:", fmt.Sprintf("%d (calendar year)", d.Year)),
		summaryBox("Authority:", "IRS Notice 2014-21; Rev. Rul. 2023-14"),
		summaryBox("Short-term disposals:", fmt.Sprintf("%d", len(d.ShortTerm))),
		summaryBox("Long-term disposals:", fmt.Sprintf("%d", len(d.LongTerm))),
		summaryBox("ST net gain/loss:", fmtFiat(d.STNetGain)+" "+d.Currency),
		summaryBox("LT net gain/loss:", fmtFiat(d.LTNetGain)+" "+d.Currency),
		summaryBox("Combined net gain/loss:", fmtFiat(d.TotalNetGain)+" "+d.Currency),
	)

	if d.Currency != "" && d.Currency != "USD" {
		m.AddRows(summaryBox(
			"Warning:",
			fmt.Sprintf("Records are in %s. Convert to USD using the exchange rate on each transaction date (IRS requirement).", d.Currency),
		))
	}

	m.AddRow(6)

	writePartTable := func(label string, rows []USTaxRow) {
		m.AddRows(summaryBox(label, fmt.Sprintf("%d disposals", len(rows))))
		headers := []string{"Description", "Acquired", "Sold", "Proceeds", "Cost Basis", "Gain/(Loss)"}
		widths := []uint{3, 2, 2, 2, 2, 1}
		aligns := []align.Type{align.Left, align.Left, align.Left, align.Right, align.Right, align.Right}
		m.AddRows(tableHeader(headers, widths))
		for i, r := range rows {
			desc := r.Description
			if len(desc) > 22 {
				desc = desc[:10] + "…"
			}
			m.AddRows(tableRow(
				[]string{
					desc,
					fmtDate(r.AcquiredAt),
					fmtDate(r.DisposedAt),
					fmtFiat(r.ProceedsUSD),
					fmtFiat(r.CostUSD),
					fmtFiat(r.GainUSD),
				},
				widths, i%2 == 1, aligns,
			))
		}
		m.AddRow(4)
	}

	writePartTable("Part I — Short-Term Capital Gains (held ≤ 365 days)", d.ShortTerm)
	writePartTable("Part II — Long-Term Capital Gains (held > 365 days)", d.LongTerm)

	m.AddRows(summaryBox("Note:", "Report amounts on Schedule D. Wash sale rules do not apply to crypto."))

	doc, err := m.Generate()
	if err != nil {
		return err
	}
	_, err = w.Write(doc.GetBytes())
	return err
}

// buildUSRows converts disposed cost basis records into Part I / Part II rows.
// holdingThreshold is typically 365 days.
func BuildUSRows(rows []USTaxRow) (shortTerm, longTerm []USTaxRow, stNet, ltNet int64) {
	for _, r := range rows {
		if r.LongTerm {
			longTerm = append(longTerm, r)
			ltNet += r.GainUSD
		} else {
			shortTerm = append(shortTerm, r)
			stNet += r.GainUSD
		}
	}
	return
}

// usHoldingDays computes the holding period and whether it is long-term (> 365 days).
// IRS counts the date acquired exclusive and the date sold inclusive.
func UsHoldingDays(acquired, disposed time.Time) (days int, longTerm bool) {
	days = int(disposed.Sub(acquired).Hours() / 24)
	return days, days > 365
}
