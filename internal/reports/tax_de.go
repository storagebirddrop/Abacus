package reports

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"
)

// DETaxRow is one disposal event for the German §23 EStG report.
type DETaxRow struct {
	Txid         string
	Vout         int
	AcquiredAt   time.Time
	DisposedAt   time.Time
	HoldingDays  int
	CostFiat     int64 // cents (EUR)
	ProceedsFiat int64 // cents (EUR)
	GainFiat     int64 // cents (EUR)
	TaxFree      bool  // true when holding >= 365 days → §23 Abs. 1 S. 1 Nr. 2 EStG
}

// DETaxSummary is the aggregate summary for the §23 EStG report.
type DETaxSummary struct {
	Year                   int
	Rows                   []DETaxRow
	TaxableGainCents       int64 // sum of short-term gains only
	FreigrenzeCents        int64 // always 60_000 (€600)
	FreigreifenGilt        bool  // true if total taxable gain <= €600
	NetTaxableCents        int64 // 0 if Freigrenze applies, else TaxableGainCents
}

// BuildDESummary filters and computes all §23 EStG figures for the given calendar year.
func BuildDESummary(year int, rows []DETaxRow) DETaxSummary {
	const freigrenzeCents = 60_000 // €600

	var taxable int64
	for _, r := range rows {
		if !r.TaxFree && r.GainFiat > 0 {
			taxable += r.GainFiat
		}
	}

	greift := taxable <= freigrenzeCents
	net := taxable
	if greift {
		net = 0
	}

	return DETaxSummary{
		Year:             year,
		Rows:             rows,
		TaxableGainCents: taxable,
		FreigrenzeCents:  freigrenzeCents,
		FreigreifenGilt:  greift,
		NetTaxableCents:  net,
	}
}

// WriteTaxDECSV writes the Germany §23 EStG tax report as CSV.
func WriteTaxDECSV(w io.Writer, s DETaxSummary) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{fmt.Sprintf("Anlage SO — Kryptowährungen — Veranlagungszeitraum %d", s.Year)})
	_ = cw.Write([]string{"Rechtsgrundlage: EStG §23 Abs. 1 S. 1 Nr. 2"})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{
		"Bezeichnung",
		"Anschaffungsdatum",
		"Veräußerungsdatum",
		"Haltedauer (Tage)",
		"Anschaffungskosten (EUR)",
		"Veräußerungserlös (EUR)",
		"Gewinn/Verlust (EUR)",
		"Steuerrechtliche Behandlung",
	})

	for _, r := range s.Rows {
		treatment := "Steuerpflichtig (§23 EStG)"
		if r.TaxFree {
			treatment = "Steuerfrei (§23 Abs. 1 S. 1 Nr. 2 EStG — Haltedauer ≥ 365 Tage)"
		}
		_ = cw.Write([]string{
			fmt.Sprintf("BTC %s:%d", r.Txid, r.Vout),
			fmtDate(r.AcquiredAt),
			fmtDate(r.DisposedAt),
			fmt.Sprintf("%d", r.HoldingDays),
			fmtFiat(r.CostFiat),
			fmtFiat(r.ProceedsFiat),
			fmtFiat(r.GainFiat),
			treatment,
		})
	}

	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Zusammenfassung"})
	_ = cw.Write([]string{"Steuerpflichtiger Gesamtgewinn (EUR)", fmtFiat(s.TaxableGainCents)})
	_ = cw.Write([]string{"Freigrenze §23 EStG (EUR)", fmtFiat(s.FreigrenzeCents)})
	if s.FreigreifenGilt {
		_ = cw.Write([]string{"Freigrenze greift", "Ja — gesamter Betrag steuerfrei"})
	} else {
		_ = cw.Write([]string{"Freigrenze greift", "Nein"})
	}
	_ = cw.Write([]string{"Zu versteuern (EUR)", fmtFiat(s.NetTaxableCents)})

	cw.Flush()
	return cw.Error()
}

// WriteTaxDEPDF writes the Germany §23 EStG tax report as PDF.
func WriteTaxDEPDF(w io.Writer, s DETaxSummary) error {
	m, err := newPDF("", fmt.Sprintf("Anlage SO Kryptowährungen %d", s.Year), "EUR")
	if err != nil {
		return err
	}

	freigreiftStr := "Nein"
	if s.FreigreifenGilt {
		freigreiftStr = "Ja — gesamter steuerpflichtiger Betrag ist steuerfrei"
	}
	m.AddRows(
		summaryBox("Veranlagungszeitraum:", fmt.Sprintf("%d", s.Year)),
		summaryBox("Rechtsgrundlage:", "EStG §23 Abs. 1 S. 1 Nr. 2"),
		summaryBox("Steuerpfl. Gesamtgewinn:", fmtFiat(s.TaxableGainCents)+" EUR"),
		summaryBox("Freigrenze (§23 EStG):", fmtFiat(s.FreigrenzeCents)+" EUR"),
		summaryBox("Freigrenze greift:", freigreiftStr),
		summaryBox("Zu versteuern:", fmtFiat(s.NetTaxableCents)+" EUR"),
	)
	m.AddRow(6)

	headers := []string{"Bezeichnung", "Anschaffung", "Veräußerung", "Tage", "Kosten (€)", "Erlös (€)", "G/V (€)", "Behandlung"}
	widths := []uint{2, 1, 1, 1, 1, 1, 1, 4}
	aligns := []align.Type{align.Left, align.Left, align.Left, align.Right, align.Right, align.Right, align.Right, align.Left}

	m.AddRows(tableHeader(headers, widths))
	for i, r := range s.Rows {
		treatment := "Steuerpflichtig"
		if r.TaxFree {
			treatment = "Steuerfrei (≥365 Tage)"
		}
		label := fmt.Sprintf("BTC %s:%d", r.Txid, r.Vout)
		if len(label) > 18 {
			label = label[:18]
		}
		vals := []string{
			label,
			fmtDate(r.AcquiredAt),
			fmtDate(r.DisposedAt),
			fmt.Sprintf("%d", r.HoldingDays),
			fmtFiat(r.CostFiat),
			fmtFiat(r.ProceedsFiat),
			fmtFiat(r.GainFiat),
			treatment,
		}
		m.AddRows(tableRow(vals, widths, i%2 == 1, aligns))
	}

	doc, err := m.Generate()
	if err != nil {
		return err
	}
	_, err = w.Write(doc.GetBytes())
	return err
}
