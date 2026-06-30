package esplora

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestServer returns an Esplora-shaped stub: a tx list for /address/.../txs
// and a tip height for /blocks/tip/height.
func newTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/txs"):
			_, _ = w.Write([]byte(`[{"txid":"a","fee":120,"status":{"confirmed":true,"block_height":800000,"block_time":1700000000},"vin":[],"vout":[{"scriptpubkey_address":"bc1q","value":1000}]}]`))
		case strings.HasSuffix(r.URL.Path, "/blocks/tip/height"):
			_, _ = w.Write([]byte(`800123`))
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestBackend_ConcurrentRequests exercises the rate-limiter under concurrency.
// Run with -race; it fails if lastReq is accessed without synchronization.
func TestBackend_ConcurrentRequests(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	b := New(srv.URL, 1) // 1ms spacing so the test stays fast
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := b.GetTransactions(ctx, "bc1qexample"); err != nil {
				t.Errorf("GetTransactions: %v", err)
			}
			if _, err := b.BlockHeight(ctx); err != nil {
				t.Errorf("BlockHeight: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestBackend_ParsesTransaction(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	recs, err := New(srv.URL, 1).GetTransactions(context.Background(), "bc1qexample")
	if err != nil {
		t.Fatalf("GetTransactions: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	r := recs[0]
	if r.Txid != "a" || r.FeeSats != 120 || !r.Confirmed || r.BlockHeight != 800000 {
		t.Errorf("record = %+v", r)
	}
	if len(r.Outputs) != 1 || r.Outputs[0].Sats != 1000 || r.Outputs[0].Address != "bc1q" {
		t.Errorf("outputs = %+v", r.Outputs)
	}
}

// TestBackend_ThrottleRespectsCancellation ensures a cancelled context aborts
// the inter-request wait instead of sleeping the full delay.
func TestBackend_ThrottleRespectsCancellation(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	b := New(srv.URL, 10_000) // 10s spacing
	// First call primes lastReq.
	if _, err := b.BlockHeight(context.Background()); err != nil {
		t.Fatalf("priming call: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := b.BlockHeight(ctx)
	if err == nil {
		t.Fatal("expected context deadline error during throttle wait")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("throttle ignored cancellation: waited %v", elapsed)
	}
}
