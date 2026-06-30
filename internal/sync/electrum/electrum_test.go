package electrum

import (
	"encoding/json"
	"testing"
)

func TestBtcStringToSats(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0", 0},
		{"1", 100_000_000},
		{"10", 1_000_000_000},
		{"0.00000001", 1},
		{"1.23456789", 123_456_789},
		{"1.5", 150_000_000},
		{"0.29", 29_000_000},                  // float path gives 28999999 — the bug
		{"20999999.97690000", 2_099_999_997_690_000},
		{"0.123456789", 12_345_678},           // >8 decimals truncated to sat precision
		{"", 0},
		{"-0.5", -50_000_000},
	}
	for _, c := range cases {
		if got := btcStringToSats(json.Number(c.in)); got != c.want {
			t.Errorf("btcStringToSats(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBtcStringToSats_BeatsFloatTruncation(t *testing.T) {
	// Demonstrates the defect the fix addresses. Using a runtime float64 (not a
	// compile-time constant, which Go folds exactly) the naive conversion loses
	// a satoshi on 0.29 BTC; the exact decimal conversion does not.
	btc := 0.29
	floaty := int64(btc * 1e8)
	exact := btcStringToSats(json.Number("0.29"))
	if exact != 29_000_000 {
		t.Fatalf("exact conversion = %d, want 29000000", exact)
	}
	if floaty == exact {
		t.Skipf("platform float gave %d (no loss here); exact path still correct", floaty)
	}
	t.Logf("float path lost precision: %d vs exact %d", floaty, exact)
}

func TestElectrumVout_DecodesValueExactly(t *testing.T) {
	// The verbose-tx value arrives as a JSON number; ensure it round-trips
	// through the struct into exact satoshis.
	var vout electrumVout
	if err := json.Unmarshal([]byte(`{"value":0.29,"n":0,"scriptPubKey":{}}`), &vout); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := btcStringToSats(vout.Value); got != 29_000_000 {
		t.Errorf("decoded value → %d sats, want 29000000", got)
	}
}
