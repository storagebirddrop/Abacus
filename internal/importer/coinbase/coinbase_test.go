package coinbase

import (
	"context"
	"strings"
	"testing"
)

const sampleCSV = `Timestamp,Transaction Type,Asset,Quantity Transacted,Spot Price Currency,Spot Price at Transaction,Subtotal,Total (inclusive of fees and/or spread),Fees and/or Spread,Notes
2024-01-15T10:00:00Z,Buy,BTC,0.01000000,USD,50000.00,500.00,501.50,1.50,Bought BTC
2024-02-20T11:00:00Z,Sell,BTC,0.00500000,USD,60000.00,300.00,298.50,1.50,Sold BTC
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

	buy := result.Trades[0]
	if buy.FiatCurrency != "USD" {
		t.Errorf("FiatCurrency = %q, want USD", buy.FiatCurrency)
	}
	if buy.FiatAmount != 50150 { // "Total (inclusive of fees and/or spread)" = 501.50
		t.Errorf("FiatAmount = %d, want 50150 (fee-inclusive total)", buy.FiatAmount)
	}
	if buy.FeeFiat != 150 {
		t.Errorf("FeeFiat = %d, want 150", buy.FeeFiat)
	}
	if buy.Sats != 1_000_000 {
		t.Errorf("Sats = %d, want 1000000", buy.Sats)
	}
}

// TestImport_ExternalIDDeduplication verifies the fix for silent duplicate
// trades on re-import: Coinbase's CSV has no native order ID, and the
// exchange_trades unique index only applies when external_id is non-empty,
// so re-importing the same file previously created full duplicate rows with
// no error. Each row must now get a stable, non-empty synthetic ID, and
// re-importing the same file must produce identical IDs.
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
	// The two distinct rows (different day, type, and amounts) must not
	// collide with each other.
	if first.Trades[0].ExternalID == first.Trades[1].ExternalID {
		t.Error("two genuinely different trades produced the same ExternalID")
	}
}
