package reports

import (
	"fmt"
	"io"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

var (
	darkSlate = props.Color{Red: 30, Green: 41, Blue: 59}   // #1E293B
	lightGray = props.Color{Red: 248, Green: 250, Blue: 252} // #F8FAFC
	gainGreen = props.Color{Red: 22, Green: 163, Blue: 74}   // #16A34A
	lossRed   = props.Color{Red: 220, Green: 38, Blue: 38}   // #DC2626
	white     = props.Color{Red: 255, Green: 255, Blue: 255}
	black     = props.Color{Red: 0, Green: 0, Blue: 0}
	slate600  = props.Color{Red: 71, Green: 85, Blue: 105}
)

func newPDF(walletName, reportTitle, currency string) (core.Maroto, error) {
	cfg := config.NewBuilder().
		WithPageNumber(props.PageNumber{
			Pattern: "{current} / {total}",
			Size:    8,
		}).
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(15).
		WithBottomMargin(20).
		Build()

	m := maroto.New(cfg)

	// Header bar
	m.AddRows(
		row.New(14).Add(
			col.New(12).Add(
				text.New("Abacus  ·  "+reportTitle, props.Text{
					Size:  14,
					Style: fontstyle.Bold,
					Color: &white,
					Align: align.Left,
					Top:   3,
					Left:  2,
				}),
			),
		).WithStyle(&props.Cell{
			BackgroundColor: &darkSlate,
			BorderType:      border.None,
		}),
	)

	// Subtitle row: wallet name + generated date
	m.AddRows(
		row.New(8).Add(
			col.New(8).Add(
				text.New("Wallet: "+walletName, props.Text{
					Size:  9,
					Color: &slate600,
					Align: align.Left,
					Top:   1,
				}),
			),
			col.New(4).Add(
				text.New("Generated: "+time.Now().Format("2006-01-02"), props.Text{
					Size:  9,
					Color: &slate600,
					Align: align.Right,
					Top:   1,
				}),
			),
		),
	)

	// Thin divider
	m.AddRows(
		row.New(1).Add(
			col.New(12).Add(
				text.New("", props.Text{}),
			),
		).WithStyle(&props.Cell{
			BackgroundColor: &darkSlate,
		}),
	)

	// Spacer
	m.AddRow(4)

	return m, nil
}

func tableHeader(cols []string, widths []uint) core.Row {
	cells := make([]core.Col, len(cols))
	for i, h := range cols {
		w := uint(2)
		if i < len(widths) {
			w = widths[i]
		}
		cells[i] = col.New(int(w)).Add(
			text.New(h, props.Text{
				Size:  8,
				Style: fontstyle.Bold,
				Color: &white,
				Align: align.Center,
				Top:   1.5,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: &darkSlate,
			BorderType:      border.None,
		})
	}
	return row.New(10).Add(cells...)
}

func tableRow(vals []string, widths []uint, alt bool, alignments []align.Type) core.Row {
	cells := make([]core.Col, len(vals))
	bg := &white
	if alt {
		bg = &lightGray
	}
	for i, v := range vals {
		w := uint(2)
		if i < len(widths) {
			w = widths[i]
		}
		al := align.Left
		if i < len(alignments) {
			al = alignments[i]
		}
		cells[i] = col.New(int(w)).Add(
			text.New(v, props.Text{
				Size:  8,
				Color: &black,
				Align: al,
				Top:   1.5,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: bg,
			BorderType:      border.None,
		})
	}
	return row.New(9).Add(cells...)
}

func summaryBox(label, value string) core.Row {
	return row.New(10).Add(
		col.New(4).Add(
			text.New(label, props.Text{
				Size:  9,
				Style: fontstyle.Bold,
				Color: &slate600,
				Align: align.Right,
				Top:   1,
			}),
		),
		col.New(8).Add(
			text.New(value, props.Text{
				Size:  10,
				Style: fontstyle.Bold,
				Color: &darkSlate,
				Align: align.Left,
				Top:   1,
				Left:  2,
			}),
		),
	)
}

// WriteTransactionsPDF writes the transactions report as a PDF to w.
func WriteTransactionsPDF(w io.Writer, rows []TransactionRow, walletName string) error {
	m, err := newPDF(walletName, "Transaction History", "")
	if err != nil {
		return err
	}

	// Summary
	m.AddRows(summaryBox("Total transactions:", fmt.Sprintf("%d", len(rows))))
	m.AddRow(6)

	// Table
	cols := []string{"Date", "Txid", "Fee (sats)", "Confirmed"}
	widths := []uint{2, 7, 2, 1}
	aligns := []align.Type{align.Left, align.Left, align.Right, align.Center}

	m.AddRows(tableHeader(cols, widths))

	for i, r := range rows {
		confirmed := "No"
		if r.Confirmed {
			confirmed = "Yes"
		}
		vals := []string{fmtDate(r.Date), r.Txid[:min(len(r.Txid), 40)], fmt.Sprintf("%d", r.FeeSats), confirmed}
		m.AddRows(tableRow(vals, widths, i%2 == 1, aligns))
	}

	doc, err := m.Generate()
	if err != nil {
		return err
	}
	_, err = w.Write(doc.GetBytes())
	return err
}

// WritePnLPDF writes the P&L report as a PDF to w.
func WritePnLPDF(w io.Writer, rows []PnLRow, currency, walletName string) error {
	m, err := newPDF(walletName, "Profit & Loss Report", currency)
	if err != nil {
		return err
	}

	// Summary stats
	var totalGain, totalLoss int64
	for _, r := range rows {
		if r.GainFiat >= 0 {
			totalGain += r.GainFiat
		} else {
			totalLoss += r.GainFiat
		}
	}
	m.AddRows(
		summaryBox("Currency:", currency),
		summaryBox("Disposals:", fmt.Sprintf("%d", len(rows))),
		summaryBox("Total gains:", fmtFiat(totalGain)+" "+currency),
		summaryBox("Total losses:", fmtFiat(totalLoss)+" "+currency),
		summaryBox("Net gain/loss:", fmtFiat(totalGain+totalLoss)+" "+currency),
	)
	m.AddRow(6)

	// Table
	cols := []string{"UTXO", "Acquired", "Disposed", "Method", "Cost", "Proceeds", "Gain/Loss"}
	widths := []uint{3, 2, 2, 1, 1, 1, 2}
	aligns := []align.Type{align.Left, align.Left, align.Left, align.Center, align.Right, align.Right, align.Right}

	m.AddRows(tableHeader(cols, widths))

	for i, r := range rows {
		gainStr := fmtFiat(r.GainFiat) + " " + r.Currency
		vals := []string{
			fmt.Sprintf("%s:%d", r.Txid[:min(len(r.Txid), 16)], r.Vout),
			fmtDate(r.AcquiredAt),
			fmtDate(r.DisposedAt),
			r.Method,
			fmtFiat(r.CostFiat),
			fmtFiat(r.ProceedsFiat),
			gainStr,
		}

		// Color last cell based on gain/loss
		alt := i%2 == 1
		bg := &white
		if alt {
			bg = &lightGray
		}

		cells := make([]core.Col, len(vals))
		for j, v := range vals {
			w := uint(2)
			if j < len(widths) {
				w = widths[j]
			}
			al := align.Left
			if j < len(aligns) {
				al = aligns[j]
			}
			color := &black
			if j == len(vals)-1 {
				if r.GainFiat >= 0 {
					color = &gainGreen
				} else {
					color = &lossRed
				}
			}
			cells[j] = col.New(int(w)).Add(
				text.New(v, props.Text{Size: 8, Color: color, Align: al, Top: 1.5, Style: func() fontstyle.Type {
					if j == len(vals)-1 {
						return fontstyle.Bold
					}
					return fontstyle.Normal
				}()}),
			).WithStyle(&props.Cell{BackgroundColor: bg, BorderType: border.None})
		}
		m.AddRows(row.New(9).Add(cells...))
	}

	doc, err := m.Generate()
	if err != nil {
		return err
	}
	_, err = w.Write(doc.GetBytes())
	return err
}

// WriteBalanceSheetPDF writes the balance-sheet report as a PDF to w.
func WriteBalanceSheetPDF(w io.Writer, rows []BalanceRow, currency, walletName string) error {
	m, err := newPDF(walletName, "Balance Sheet", currency)
	if err != nil {
		return err
	}

	var totalSats, totalCost int64
	for _, r := range rows {
		totalSats += r.Sats
		totalCost += r.CostFiat
	}
	m.AddRows(
		summaryBox("Currency:", currency),
		summaryBox("UTXOs held:", fmt.Sprintf("%d", len(rows))),
		summaryBox("Total BTC:", fmtBTC(totalSats)),
		summaryBox("Total cost basis:", fmtFiat(totalCost)+" "+currency),
	)
	m.AddRow(6)

	cols := []string{"UTXO", "Address", "Acquired", "Amount (BTC)", "Cost Basis"}
	widths := []uint{3, 4, 2, 2, 1}
	aligns := []align.Type{align.Left, align.Left, align.Left, align.Right, align.Right}

	m.AddRows(tableHeader(cols, widths))

	for i, r := range rows {
		addrTrunc := r.Address
		if len(addrTrunc) > 30 {
			addrTrunc = addrTrunc[:14] + "…" + addrTrunc[len(addrTrunc)-10:]
		}
		vals := []string{
			fmt.Sprintf("%s:%d", r.Txid[:min(len(r.Txid), 12)], r.Vout),
			addrTrunc,
			fmtDate(r.AcquiredAt),
			fmtBTC(r.Sats),
			fmtFiat(r.CostFiat),
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
