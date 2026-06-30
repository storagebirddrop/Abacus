package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SecurityConfig configures the optional API auth + rate-limiting middleware.
type SecurityConfig struct {
	// APIToken, when non-empty, requires `Authorization: Bearer <token>` on all
	// /api/v1 routes except health/version. Empty disables auth (default).
	APIToken string
	// RateLimitRPM caps requests per client IP per minute. <= 0 disables it.
	RateLimitRPM int
	// TrustProxy, when true, derives the client IP from the X-Forwarded-For /
	// X-Real-IP headers instead of the TCP peer address. Enable this ONLY when
	// Abacus sits behind a trusted reverse proxy that sets these headers —
	// otherwise a client can spoof them to evade per-IP rate limiting.
	TrustProxy bool
}

// passthrough is a no-op middleware used when a feature is disabled.
func passthrough(next http.Handler) http.Handler { return next }

// tokenAuth requires a bearer token on every request except health/version.
// Returns a no-op middleware when token is empty.
func tokenAuth(token string) func(http.Handler) http.Handler {
	if token == "" {
		return passthrough
	}
	expected := []byte("Bearer " + token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exact match: a suffix check would let a future route such as
			// /wallets/health silently bypass auth.
			if p := r.URL.Path; p == "/api/v1/health" || p == "/api/v1/version" {
				next.ServeHTTP(w, r)
				return
			}
			got := []byte(r.Header.Get("Authorization"))
			if subtle.ConstantTimeCompare(got, expected) == 1 {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		})
	}
}

type rateBucket struct {
	count       int
	windowStart time.Time
}

// rateLimiter applies a fixed-window per-IP request cap. Returns a no-op
// middleware when rpm <= 0.
func rateLimiter(rpm int, trustProxy bool) func(http.Handler) http.Handler {
	if rpm <= 0 {
		return passthrough
	}
	const window = time.Minute
	var (
		mu        sync.Mutex
		buckets   = map[string]*rateBucket{}
		lastSweep = time.Now()
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, trustProxy)
			now := time.Now()

			mu.Lock()
			// Opportunistic prune of stale buckets to bound memory.
			if now.Sub(lastSweep) >= window {
				for k, b := range buckets {
					if now.Sub(b.windowStart) >= window {
						delete(buckets, k)
					}
				}
				lastSweep = now
			}
			b := buckets[ip]
			if b == nil || now.Sub(b.windowStart) >= window {
				b = &rateBucket{windowStart: now}
				buckets[ip] = b
			}
			b.count++
			count := b.count
			reset := b.windowStart.Add(window)
			mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rpm))
			if count > rpm {
				retry := int(time.Until(reset).Seconds()) + 1
				w.Header().Set("Retry-After", strconv.Itoa(retry))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client host. When trustProxy is set it prefers the
// first hop in X-Forwarded-For (the original client), falling back to X-Real-IP,
// then the TCP peer address. Without trustProxy it always uses the peer address
// so spoofed headers cannot influence per-IP rate limiting.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Leftmost entry is the original client; the rest are proxies.
			first := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
			if first != "" {
				return first
			}
		}
		if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
			return xr
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
