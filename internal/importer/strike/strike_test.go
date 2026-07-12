package strike

import (
	"context"
	"strings"
	"testing"
)

const sampleCSV = `Date,Type,BTC Amount,USD Amount,Fee,Balance (BTC),Balance (USD),Description
2024-01-15T10:00:00Z,Trade Buy,0.01000000,500.00,0.00,0.01,0,Bought BTC
2024-02-20T11:00:00Z,Trade Sell,0.00500000,300.00,0.00,0.005,0,Sold BTC
`

func TestImport_BasicFields(t *testing.T) {
	imp := New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(result.Trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(result.Trades))
	}
	if result.Trades[0].FiatCurrency != "USD" {
		t.Errorf("FiatCurrency = %q, want USD", result.Trades[0].FiatCurrency)
	}
}

// TestImport_ExternalIDDeduplication verifies the fix for silent duplicate
// trades on re-import: Strike's CSV has no per-row transaction ID, and the
// exchange_trades unique index only applies when external_id is non-empty,
// so re-importing the same file previously created full duplicate rows with
// no error.
func TestImport_ExternalIDDeduplication(t *testing.T) {
	imp := New()
	first, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatalf("first Import: %v", err)
	}
	second, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatalf("second Import: %v", err)
	}
	if len(first.Trades) != len(second.Trades) {
		t.Fatalf("trade count differs between imports: %d vs %d", len(first.Trades), len(second.Trades))
	}
	for i := range first.Trades {
		if first.Trades[i].ExternalID == "" {
			t.Errorf("trade %d: ExternalID is empty; re-importing this file would create an unguarded duplicate", i)
		}
		if first.Trades[i].ExternalID != second.Trades[i].ExternalID {
			t.Errorf("trade %d: ExternalID differs between imports of the same file: %q vs %q",
				i, first.Trades[i].ExternalID, second.Trades[i].ExternalID)
		}
	}
	if first.Trades[0].ExternalID == first.Trades[1].ExternalID {
		t.Error("two genuinely different trades produced the same ExternalID")
	}
}
