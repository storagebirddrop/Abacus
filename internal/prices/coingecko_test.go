package prices

import "testing"

// TestFetchRange documents the rounding contract for the prices/100 → cents
// conversion: it was previously an int64() truncation, biasing every price
// down by up to 1 cent. This test exercises the arithmetic directly since
// FetchRange itself requires a live HTTP call.
func TestPriceToCentsRounding(t *testing.T) {
	tests := []struct {
		price float64
		want  int64
	}{
		{51234.567, 5_123_457}, // truncation would give 5_123_456
		{51234.564, 5_123_456},
		{100.0, 10_000},
		{0.005, 1}, // rounds up, truncation would give 0
	}
	for _, tc := range tests {
		got := roundCents(tc.price)
		if got != tc.want {
			t.Errorf("roundCents(%v) = %d, want %d", tc.price, got, tc.want)
		}
	}
}
