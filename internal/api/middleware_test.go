package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func doReq(h http.Handler, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// --- tokenAuth ---------------------------------------------------------------

func TestTokenAuth_DisabledWhenEmpty(t *testing.T) {
	h := tokenAuth("")(okHandler())
	if rec := doReq(h, "GET", "/api/v1/wallets", ""); rec.Code != http.StatusOK {
		t.Fatalf("no token configured should pass through, got %d", rec.Code)
	}
}

func TestTokenAuth_RejectsMissingAndWrong(t *testing.T) {
	h := tokenAuth("s3cret")(okHandler())
	if rec := doReq(h, "GET", "/api/v1/wallets", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("missing token: got %d, want 401", rec.Code)
	}
	if rec := doReq(h, "GET", "/api/v1/wallets", "Bearer nope"); rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", rec.Code)
	}
}

func TestTokenAuth_AcceptsCorrect(t *testing.T) {
	h := tokenAuth("s3cret")(okHandler())
	if rec := doReq(h, "GET", "/api/v1/wallets", "Bearer s3cret"); rec.Code != http.StatusOK {
		t.Errorf("correct token: got %d, want 200", rec.Code)
	}
}

func TestTokenAuth_HealthAndVersionExempt(t *testing.T) {
	h := tokenAuth("s3cret")(okHandler())
	for _, p := range []string{"/api/v1/health", "/api/v1/version"} {
		if rec := doReq(h, "GET", p, ""); rec.Code != http.StatusOK {
			t.Errorf("%s should be exempt from auth, got %d", p, rec.Code)
		}
	}
}

func TestTokenAuth_ExemptionIsExactNotSuffix(t *testing.T) {
	// A route merely ending in "/health" must still require auth.
	h := tokenAuth("s3cret")(okHandler())
	if rec := doReq(h, "GET", "/api/v1/wallets/health", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("/wallets/health must not bypass auth, got %d", rec.Code)
	}
}

// --- clientIP ----------------------------------------------------------------

func TestClientIP_IgnoresProxyHeadersByDefault(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/wallets", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := clientIP(req, false); got != "10.0.0.1" {
		t.Errorf("without trustProxy got %q, want peer 10.0.0.1", got)
	}
}

func TestClientIP_UsesForwardedForWhenTrusted(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/wallets", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 10.0.0.9")
	if got := clientIP(req, true); got != "1.2.3.4" {
		t.Errorf("with trustProxy got %q, want original client 1.2.3.4", got)
	}
}

func TestClientIP_FallsBackToRealIPThenPeer(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/wallets", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "5.6.7.8")
	if got := clientIP(req, true); got != "5.6.7.8" {
		t.Errorf("X-Real-IP fallback got %q, want 5.6.7.8", got)
	}
	req2 := httptest.NewRequest("GET", "/api/v1/wallets", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	if got := clientIP(req2, true); got != "10.0.0.1" {
		t.Errorf("no proxy headers got %q, want peer 10.0.0.1", got)
	}
}

func TestRateLimiter_SpoofedHeaderSharesBucketWithoutProxy(t *testing.T) {
	// Without trustProxy, two requests from the same peer but different
	// X-Forwarded-For values must share one bucket (no spoofed evasion).
	h := rateLimiter(1, false)(okHandler())
	mk := func(xff string) int {
		req := httptest.NewRequest("GET", "/api/v1/wallets", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := mk("1.1.1.1"); c != http.StatusOK {
		t.Fatalf("first request got %d, want 200", c)
	}
	if c := mk("2.2.2.2"); c != http.StatusTooManyRequests {
		t.Fatalf("spoofed-XFF second request should still be rate-limited, got %d", c)
	}
}

// --- rateLimiter -------------------------------------------------------------

func TestRateLimiter_DisabledWhenZero(t *testing.T) {
	h := rateLimiter(0, false)(okHandler())
	for i := 0; i < 50; i++ {
		if rec := doReq(h, "GET", "/api/v1/wallets", ""); rec.Code != http.StatusOK {
			t.Fatalf("disabled limiter should always pass, got %d on req %d", rec.Code, i)
		}
	}
}

func TestRateLimiter_AllowsUpToLimitThen429(t *testing.T) {
	h := rateLimiter(3, false)(okHandler())
	for i := 1; i <= 3; i++ {
		if rec := doReq(h, "GET", "/api/v1/wallets", ""); rec.Code != http.StatusOK {
			t.Fatalf("request %d should be allowed, got %d", i, rec.Code)
		}
	}
	rec := doReq(h, "GET", "/api/v1/wallets", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request should be 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 response should set Retry-After")
	}
	if rec.Header().Get("X-RateLimit-Limit") != "3" {
		t.Errorf("X-RateLimit-Limit = %q, want 3", rec.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	h := rateLimiter(1, false)(okHandler())
	// First IP uses its single allowance.
	req1 := httptest.NewRequest("GET", "/api/v1/wallets", nil)
	req1.RemoteAddr = "10.0.0.1:1111"
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("IP1 first request: got %d", rec1.Code)
	}
	// A different IP must not be affected by IP1's usage.
	req2 := httptest.NewRequest("GET", "/api/v1/wallets", nil)
	req2.RemoteAddr = "10.0.0.2:2222"
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("IP2 should have its own bucket, got %d", rec2.Code)
	}
}
