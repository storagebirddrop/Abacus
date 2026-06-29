package reports

import (
	"fmt"
	"time"
)

// TransactionRow holds one row of the transactions report.
type TransactionRow struct {
	Date      time.Time
	Txid      string
	FeeSats   int64
	Confirmed bool
}

// PnLRow holds one disposal event for the P&L report.
type PnLRow struct {
	Txid         string
	Vout         int
	AcquiredAt   time.Time
	DisposedAt   time.Time
	CostSats     int64
	CostFiat     int64 // cents
	ProceedsFiat int64 // cents
	GainFiat     int64 // cents
	Method       string
	Currency     string
}

// BalanceRow holds one unspent UTXO for the balance-sheet report.
type BalanceRow struct {
	Txid       string
	Vout       int
	Sats       int64
	Address    string
	AcquiredAt time.Time
	CostFiat   int64 // cents; 0 if no accounting run done yet
	Currency   string
}

func fmtBTC(sats int64) string {
	return fmt.Sprintf("%.8f", float64(sats)/1e8)
}

func fmtFiat(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

func fmtDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
