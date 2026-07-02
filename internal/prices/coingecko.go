// Package prices provides external price-feed integrations.
package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const coinGeckoBase = "https://api.coingecko.com/api/v3"

// FetchRange fetches historical daily BTC prices from CoinGecko for the given
// date range and fiat currency. It makes a single HTTP request regardless of
// how many days are in the range.
//
// Returns a map keyed by UTC calendar date (always midnight) to price in cents.
// Dates with no CoinGecko data are absent from the map; the caller decides how
// to handle gaps.
func FetchRange(ctx context.Context, currency string, from, to time.Time) (map[time.Time]int64, error) {
	url := fmt.Sprintf(
		"%s/coins/bitcoin/market_chart/range?vs_currency=%s&from=%d&to=%d",
		coinGeckoBase,
		strings.ToLower(currency),
		from.Unix(),
		to.Unix(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Abacus-Bitcoin-Accounting")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("CoinGecko request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("CoinGecko rate limit reached — please wait a minute and try again")
	case http.StatusOK:
		// continue below
	default:
		return nil, fmt.Errorf("CoinGecko returned HTTP %d", resp.StatusCode)
	}

	// Response: {"prices": [[unix_ms, price_float], ...], "market_caps": [...], "total_volumes": [...]}
	var body struct {
		Prices [][2]float64 `json:"prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode CoinGecko response: %w", err)
	}

	// Normalise to UTC calendar days. CoinGecko may return multiple data points
	// per day (hourly for short ranges). We keep the last entry per day so the
	// retained price is close to the end-of-day close.
	result := make(map[time.Time]int64, len(body.Prices))
	for _, entry := range body.Prices {
		ts := time.Unix(int64(entry[0])/1000, 0).UTC()
		day := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
		cents := int64(entry[1] * 100)
		if cents > 0 {
			result[day] = cents // later entries for the same day overwrite earlier ones
		}
	}
	return result, nil
}
