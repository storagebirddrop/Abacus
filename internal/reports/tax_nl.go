package reports

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"
)

// NLHoldingRow is one UTXO held on the Box 3 peildatum (1 januari).
type NLHoldingRow struct {
	Txid string
	Vout int
	Sats int64
}

// NLTaxData is the complete dataset for the Netherlands Box 3 report.
type NLTaxData struct {
	Year         int
	PeildatumBTC int64 // total sats held on 1 januari
	PriceEUR     int64 // cents per BTC on 1 januari (0 = unknown)
	ValueEUR     int64 // cents total
	Holdings     []NLHoldingRow
}

// NLPeildatum returns January 1 of the given year (UTC) — the Box 3 reference date.
func NLPeildatum(year int) time.Time {
	return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
}

// WriteTaxNLCSV writes the Netherlands Box 3 tax report as CSV.
func WriteTaxNLCSV(w io.Writer, d NLTaxData) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{fmt.Sprintf("Opgave Box 3 Cryptovaluta — %d", d.Year)})
	_ = cw.Write([]string{fmt.Sprintf("Peildatum: 1 januari %d", d.Year)})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"UTXO", "Aantal BTC", "Koers EUR/BTC", "Waarde EUR"})

	priceStr := fmtFiat(d.PriceEUR)
	if d.PriceEUR == 0 {
		priceStr = "onbekend"
	}
	for _, h := range d.Holdings {
		valStr := fmtFiat(h.Sats * d.PriceEUR / 100_000_000)
		if d.PriceEUR == 0 {
			valStr = "onbekend"
		}
		_ = cw.Write([]string{
			fmt.Sprintf("%s:%d", h.Txid, h.Vout),
			fmtBTC(h.Sats),
			priceStr,
			valStr,
		})
	}
	_ = cw.Write([]string{})
	totalStr := fmtFiat(d.ValueEUR)
	if d.PriceEUR == 0 {
		totalStr = "onbekend"
	}
	_ = cw.Write([]string{"Totaal", fmtBTC(d.PeildatumBTC), priceStr, totalStr})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Opmerking: voer de totale waarde in als 'Overige bezittingen' in uw aangifte inkomstenbelasting."})
	cw.Flush()
	return cw.Error()
}

// WriteTaxNLPDF writes the Netherlands Box 3 tax report as PDF.
func WriteTaxNLPDF(w io.Writer, d NLTaxData) error {
	m, err := newPDF("", fmt.Sprintf("Box 3 Cryptovaluta %d", d.Year), "EUR")
	if err != nil {
		return err
	}

	priceLabel := fmtFiat(d.PriceEUR) + " EUR/BTC"
	totalLabel := fmtFiat(d.ValueEUR) + " EUR"
	if d.PriceEUR == 0 {
		priceLabel = "onbekend"
		totalLabel = "onbekend"
	}

	m.AddRows(
		summaryBox("Peildatum:", fmt.Sprintf("1 januari %d", d.Year)),
		summaryBox("Totaal BTC:", fmtBTC(d.PeildatumBTC)),
		summaryBox("Koers op peildatum:", priceLabel),
		summaryBox("Totale waarde (Box 3):", totalLabel),
	)
	m.AddRow(6)

	headers := []string{"UTXO", "Aantal BTC", "Koers (EUR/BTC)", "Waarde (EUR)"}
	widths := []uint{5, 3, 2, 2}
	aligns := []align.Type{align.Left, align.Right, align.Right, align.Right}

	m.AddRows(tableHeader(headers, widths))
	for i, h := range d.Holdings {
		valStr := fmtFiat(h.Sats * d.PriceEUR / 100_000_000)
		if d.PriceEUR == 0 {
			valStr = "onbekend"
		}
		label := fmt.Sprintf("%s:%d", h.Txid, h.Vout)
		if len(label) > 36 {
			label = label[:16] + "…" + label[len(label)-14:]
		}
		pStr := fmtFiat(d.PriceEUR)
		if d.PriceEUR == 0 {
			pStr = "onbekend"
		}
		vals := []string{label, fmtBTC(h.Sats), pStr, valStr}
		m.AddRows(tableRow(vals, widths, i%2 == 1, aligns))
	}

	m.AddRow(8)
	m.AddRows(summaryBox(
		"Opmerking:",
		"Voer de totale waarde in als 'Overige bezittingen' in uw aangifte inkomstenbelasting (IB).",
	))

	doc, err := m.Generate()
	if err != nil {
		return err
	}
	_, err = w.Write(doc.GetBytes())
	return err
}
