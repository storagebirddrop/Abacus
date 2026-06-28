package reports

import (
	"encoding/csv"
	"fmt"
	"io"
)

// WriteTransactionsCSV writes the transactions report as CSV to w.
func WriteTransactionsCSV(w io.Writer, rows []TransactionRow) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"Date", "Txid", "Fee (sats)", "Confirmed"}); err != nil {
		return err
	}
	for _, r := range rows {
		confirmed := "No"
		if r.Confirmed {
			confirmed = "Yes"
		}
		if err := cw.Write([]string{
			fmtDate(r.Date),
			r.Txid,
			fmt.Sprintf("%d", r.FeeSats),
			confirmed,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WritePnLCSV writes the P&L report as CSV to w.
func WritePnLCSV(w io.Writer, rows []PnLRow, currency string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"UTXO (txid:vout)", "Acquired", "Disposed",
		"Method", "Cost (" + currency + ")", "Proceeds (" + currency + ")", "Gain/Loss (" + currency + ")",
	}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write([]string{
			fmt.Sprintf("%s:%d", r.Txid, r.Vout),
			fmtDate(r.AcquiredAt),
			fmtDate(r.DisposedAt),
			r.Method,
			fmtFiat(r.CostFiat),
			fmtFiat(r.ProceedsFiat),
			fmtFiat(r.GainFiat),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteBalanceSheetCSV writes the balance-sheet report as CSV to w.
func WriteBalanceSheetCSV(w io.Writer, rows []BalanceRow, currency string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"UTXO (txid:vout)", "Address", "Acquired", "Amount (BTC)", "Cost Basis (" + currency + ")",
	}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write([]string{
			fmt.Sprintf("%s:%d", r.Txid, r.Vout),
			r.Address,
			fmtDate(r.AcquiredAt),
			fmtBTC(r.Sats),
			fmtFiat(r.CostFiat),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
