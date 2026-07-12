package kraken

import (
	"context"
	"strings"
	"testing"
)

const header = "txid,refid,time,type,subtype,aclass,asset,amount,fee,balance\n"

// TestImport_EURTrade is the original, previously-supported case: a EUR-quoted
// buy pairs an XBT receive row with a ZEUR spend row under the same refid.
func TestImport_EURTrade(t *testing.T) {
	csv := header +
		"T1,REF1,2024-01-15 10:00:00,trade,,currency,XXBT,0.01000000,0.0000000000,0.01\n" +
		"T2,REF1,2024-01-15 10:00:00,trade,,currency,ZEUR,-500.0000,1.5000,1000.00\n"

	imp := New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	tr := result.Trades[0]
	if tr.FiatCurrency != "EUR" {
		t.Errorf("FiatCurrency = %q, want EUR", tr.FiatCurrency)
	}
	if tr.FiatAmount != 50000 {
		t.Errorf("FiatAmount = %d, want 50000 (500.00 EUR)", tr.FiatAmount)
	}
}

// TestImport_USDTrade is the fix under test: before it, Kraken only recognised
// EUR/ZEUR fiat rows, so a USD-quoted trade (extremely common — Kraken is a
// global, multi-currency exchange) found zero matching fiat rows and silently
// recorded FiatAmount=0 for every trade, with the import reporting success.
func TestImport_USDTrade(t *testing.T) {
	csv := header +
		"T1,REF1,2024-01-15 10:00:00,trade,,currency,XXBT,0.01000000,0.0000000000,0.01\n" +
		"T2,REF1,2024-01-15 10:00:00,trade,,currency,ZUSD,-500.0000,1.5000,1000.00\n"

	imp := New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	tr := result.Trades[0]
	if tr.FiatCurrency != "USD" {
		t.Fatalf("FiatCurrency = %q, want USD (previously this would default to EUR and FiatAmount would be 0)", tr.FiatCurrency)
	}
	if tr.FiatAmount != 50000 {
		t.Errorf("FiatAmount = %d, want 50000 (500.00 USD) — got the pre-fix bug (0) if this fails", tr.FiatAmount)
	}
}

// TestImport_GBPTrade spot-checks a third currency to confirm the fix isn't
// EUR/USD-special-cased.
func TestImport_GBPTrade(t *testing.T) {
	csv := header +
		"T1,REF1,2024-01-15 10:00:00,trade,,currency,XXBT,0.01000000,0.0000000000,0.01\n" +
		"T2,REF1,2024-01-15 10:00:00,trade,,currency,ZGBP,-400.0000,1.0000,1000.00\n"

	imp := New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].FiatCurrency != "GBP" {
		t.Errorf("FiatCurrency = %q, want GBP", result.Trades[0].FiatCurrency)
	}
}

// TestImport_FiatFeeCaptured verifies the fiat-side "fee" column, previously
// parsed and then discarded, is now stored on the resulting trade.
func TestImport_FiatFeeCaptured(t *testing.T) {
	csv := header +
		"T1,REF1,2024-01-15 10:00:00,trade,,currency,XXBT,0.01000000,0.0000000000,0.01\n" +
		"T2,REF1,2024-01-15 10:00:00,trade,,currency,ZEUR,-500.0000,1.5000,1000.00\n"

	imp := New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].FeeFiat != 150 {
		t.Errorf("FeeFiat = %d, want 150 (1.50 EUR fee, previously discarded)", result.Trades[0].FeeFiat)
	}
}

// TestImport_UnrecognisedFiatSkipped confirms a genuinely non-fiat,
// non-BTC asset (e.g. an altcoin leg) is still ignored rather than
// mis-detected as a fiat rail.
func TestImport_UnrecognisedFiatSkipped(t *testing.T) {
	csv := header +
		"T1,REF1,2024-01-15 10:00:00,trade,,currency,XETH,-1.0000000000,0.0000000000,0.00\n" +
		"T2,REF1,2024-01-15 10:00:00,trade,,currency,XXBT,0.01000000,0.0000000000,0.01\n"

	imp := New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	// No fiat leg was found at all, so FiatAmount falls back to 0 with the
	// default EUR label -- this is a pre-existing, separate limitation (no
	// fiat row present at all, vs. this fix's bug which was a supported fiat
	// row being ignored). Documenting current behavior, not asserting it's ideal.
	if result.Trades[0].FiatAmount != 0 {
		t.Errorf("FiatAmount = %d, want 0 (no fiat leg in this refid group)", result.Trades[0].FiatAmount)
	}
}
