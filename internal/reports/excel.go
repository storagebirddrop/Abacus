package reports

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

const (
	headerFill  = "1E293B" // dark slate
	headerFont  = "FFFFFF" // white
	altRowFill  = "F8FAFC" // very light blue-gray
	gainColor   = "16A34A" // green
	lossColor   = "DC2626" // red
	accentColor = "3B82F6" // blue for summary cells
)

func newHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: headerFont, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{headerFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "bottom", Color: "334155", Style: 1},
		},
	})
}

func newAltRowStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{altRowFill}, Pattern: 1},
	})
}

func newSummaryLabelStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "334155"},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
}

func newSummaryValueStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: accentColor},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
}

func newGainStyle(f *excelize.File, positive bool) (int, error) {
	color := gainColor
	if !positive {
		color = lossColor
	}
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: color},
	})
}

func setColWidths(f *excelize.File, sheet string, widths map[string]float64) {
	for col, w := range widths {
		_ = f.SetColWidth(sheet, col, col, w)
	}
}

// WriteTransactionsExcel writes the transactions report as an .xlsx workbook.
func WriteTransactionsExcel(w io.Writer, rows []TransactionRow, walletName string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Transactions"
	f.SetSheetName("Sheet1", sheet)

	hStyle, _ := newHeaderStyle(f)
	altStyle, _ := newAltRowStyle(f)

	headers := []string{"Date", "Txid", "Fee (sats)", "Confirmed"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, hStyle)
	}
	_ = f.SetRowHeight(sheet, 1, 22)

	for i, r := range rows {
		row := i + 2
		confirmed := "No"
		if r.Confirmed {
			confirmed = "Yes"
		}
		vals := []interface{}{fmtDate(r.Date), r.Txid, r.FeeSats, confirmed}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
		if i%2 == 1 {
			start, _ := excelize.CoordinatesToCellName(1, row)
			end, _ := excelize.CoordinatesToCellName(len(headers), row)
			_ = f.SetCellStyle(sheet, start, end, altStyle)
		}
	}

	_ = f.AutoFilter(sheet, fmt.Sprintf("A1:%s1", colLetter(len(headers))), []excelize.AutoFilterOptions{})
	setColWidths(f, sheet, map[string]float64{"A": 14, "B": 68, "C": 12, "D": 12})

	return f.Write(w)
}

// WritePnLExcel writes the P&L report as an .xlsx workbook with two sheets.
func WritePnLExcel(w io.Writer, rows []PnLRow, currency, walletName string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: Summary
	sumSheet := "Summary"
	f.SetSheetName("Sheet1", sumSheet)
	_, _ = f.NewSheet("Disposals")

	lblStyle, _ := newSummaryLabelStyle(f)
	valStyle, _ := newSummaryValueStyle(f)

	var totalGain, totalLoss int64
	disposals := 0
	for _, r := range rows {
		disposals++
		if r.GainFiat >= 0 {
			totalGain += r.GainFiat
		} else {
			totalLoss += r.GainFiat
		}
	}

	summaryRows := [][]interface{}{
		{"Wallet", walletName},
		{"Currency", currency},
		{"Total disposals", disposals},
		{"Total realised gains", fmtFiat(totalGain) + " " + currency},
		{"Total realised losses", fmtFiat(totalLoss) + " " + currency},
		{"Net gain/loss", fmtFiat(totalGain+totalLoss) + " " + currency},
	}
	for i, sr := range summaryRows {
		lCell, _ := excelize.CoordinatesToCellName(1, i+2)
		vCell, _ := excelize.CoordinatesToCellName(2, i+2)
		_ = f.SetCellValue(sumSheet, lCell, sr[0])
		_ = f.SetCellValue(sumSheet, vCell, sr[1])
		_ = f.SetCellStyle(sumSheet, lCell, lCell, lblStyle)
		_ = f.SetCellStyle(sumSheet, vCell, vCell, valStyle)
	}
	setColWidths(f, sumSheet, map[string]float64{"A": 28, "B": 30})

	// Sheet 2: Disposals
	dispSheet := "Disposals"
	hStyle, _ := newHeaderStyle(f)
	altStyle, _ := newAltRowStyle(f)

	headers := []string{
		"UTXO", "Acquired", "Disposed", "Method",
		"Cost (" + currency + ")", "Proceeds (" + currency + ")", "Gain/Loss (" + currency + ")",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(dispSheet, cell, h)
		_ = f.SetCellStyle(dispSheet, cell, cell, hStyle)
	}
	_ = f.SetRowHeight(dispSheet, 1, 22)

	for i, r := range rows {
		row := i + 2
		vals := []interface{}{
			fmt.Sprintf("%s:%d", r.Txid, r.Vout),
			fmtDate(r.AcquiredAt),
			fmtDate(r.DisposedAt),
			r.Method,
			fmtFiat(r.CostFiat),
			fmtFiat(r.ProceedsFiat),
			fmtFiat(r.GainFiat),
		}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(dispSheet, cell, v)
		}
		// Color gain/loss cell
		gainCell, _ := excelize.CoordinatesToCellName(7, row)
		gainStyle, _ := newGainStyle(f, r.GainFiat >= 0)
		_ = f.SetCellStyle(dispSheet, gainCell, gainCell, gainStyle)

		if i%2 == 1 {
			start, _ := excelize.CoordinatesToCellName(1, row)
			end, _ := excelize.CoordinatesToCellName(6, row)
			_ = f.SetCellStyle(dispSheet, start, end, altStyle)
		}
	}
	_ = f.AutoFilter(dispSheet, fmt.Sprintf("A1:%s1", colLetter(len(headers))), []excelize.AutoFilterOptions{})
	setColWidths(f, dispSheet, map[string]float64{"A": 22, "B": 13, "C": 13, "D": 12, "E": 14, "F": 14, "G": 16})

	if idx, err := f.GetSheetIndex(sumSheet); err == nil {
		f.SetActiveSheet(idx)
	}
	return f.Write(w)
}

// WriteBalanceSheetExcel writes the balance-sheet as an .xlsx workbook.
func WriteBalanceSheetExcel(w io.Writer, rows []BalanceRow, currency, walletName string) error {
	f := excelize.NewFile()
	defer f.Close()

	sumSheet := "Summary"
	f.SetSheetName("Sheet1", sumSheet)
	_, _ = f.NewSheet("Holdings")

	lblStyle, _ := newSummaryLabelStyle(f)
	valStyle, _ := newSummaryValueStyle(f)

	var totalSats, totalCost int64
	for _, r := range rows {
		totalSats += r.Sats
		totalCost += r.CostFiat
	}

	summaryRows := [][]interface{}{
		{"Wallet", walletName},
		{"Currency", currency},
		{"UTXOs held", len(rows)},
		{"Total BTC", fmtBTC(totalSats)},
		{"Total cost basis", fmtFiat(totalCost) + " " + currency},
	}
	for i, sr := range summaryRows {
		lCell, _ := excelize.CoordinatesToCellName(1, i+2)
		vCell, _ := excelize.CoordinatesToCellName(2, i+2)
		_ = f.SetCellValue(sumSheet, lCell, sr[0])
		_ = f.SetCellValue(sumSheet, vCell, sr[1])
		_ = f.SetCellStyle(sumSheet, lCell, lCell, lblStyle)
		_ = f.SetCellStyle(sumSheet, vCell, vCell, valStyle)
	}
	setColWidths(f, sumSheet, map[string]float64{"A": 24, "B": 28})

	holdSheet := "Holdings"
	hStyle, _ := newHeaderStyle(f)
	altStyle, _ := newAltRowStyle(f)

	headers := []string{"UTXO", "Address", "Acquired", "Amount (BTC)", "Cost Basis (" + currency + ")"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(holdSheet, cell, h)
		_ = f.SetCellStyle(holdSheet, cell, cell, hStyle)
	}
	_ = f.SetRowHeight(holdSheet, 1, 22)

	for i, r := range rows {
		row := i + 2
		vals := []interface{}{
			fmt.Sprintf("%s:%d", r.Txid, r.Vout),
			r.Address,
			fmtDate(r.AcquiredAt),
			fmtBTC(r.Sats),
			fmtFiat(r.CostFiat),
		}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(holdSheet, cell, v)
		}
		if i%2 == 1 {
			start, _ := excelize.CoordinatesToCellName(1, row)
			end, _ := excelize.CoordinatesToCellName(len(headers), row)
			_ = f.SetCellStyle(holdSheet, start, end, altStyle)
		}
	}
	_ = f.AutoFilter(holdSheet, fmt.Sprintf("A1:%s1", colLetter(len(headers))), []excelize.AutoFilterOptions{})
	setColWidths(f, holdSheet, map[string]float64{"A": 22, "B": 46, "C": 13, "D": 16, "E": 18})

	if idx, err := f.GetSheetIndex(sumSheet); err == nil {
		f.SetActiveSheet(idx)
	}
	return f.Write(w)
}

// colLetter converts a 1-based column index to a letter (1→A, 7→G).
func colLetter(n int) string {
	name, _ := excelize.ColumnNumberToName(n)
	return name
}
