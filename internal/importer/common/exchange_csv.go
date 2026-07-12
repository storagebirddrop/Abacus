// Package common provides shared parsing utilities for exchange CSV importers.
package common

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ParseBTCSats converts a BTC decimal string (e.g. "0.05000000" or "-0.01")
// to integer satoshis. Returns 0 on empty input, not an error.
func ParseBTCSats(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	whole := parseDigits(parts[0]) * 100_000_000
	var frac int64
	if len(parts) == 2 {
		fs := parts[1]
		if len(fs) > 8 {
			fs = fs[:8]
		}
		for len(fs) < 8 {
			fs += "0"
		}
		frac = parseDigits(fs)
	}
	sats := whole + frac
	if negative {
		sats = -sats
	}
	return sats
}

// ParseFiatCents converts a fiat decimal string (e.g. "1234.56" or "1,234.56")
// to integer cents. Returns 0 on empty input.
func ParseFiatCents(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0
	}
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	major := parseDigits(parts[0]) * 100
	var minor int64
	if len(parts) == 2 {
		fs := parts[1]
		if len(fs) > 2 {
			fs = fs[:2]
		}
		for len(fs) < 2 {
			fs += "0"
		}
		minor = parseDigits(fs)
	}
	result := major + minor
	if negative {
		result = -result
	}
	return result
}

// SyntheticExternalID builds a stable, deterministic dedup key for exchange
// CSV rows that carry no native per-trade ID (e.g. Coinbase, Strike). The
// exchange_trades table has a unique index on (wallet_id, external_id) that
// only applies when external_id is non-NULL — an importer that always leaves
// ExternalID empty gets no dedup protection at all, and re-importing the same
// file (or an updated export that overlaps previously-imported history, the
// normal workflow for anyone tracking an ongoing account) silently inserts
// full duplicate trades.
//
// Callers should pass enough stable, row-identifying fields (raw CSV values,
// not yet parsed to sats/cents) that the same source row always produces the
// same ID on re-import. This is best-effort, not a true unique ID: two
// genuinely distinct trades with identical values in every hashed field
// (same timestamp down to the precision given, same type, same amounts) will
// collide and be treated as duplicates. In practice this requires two trades
// executed in the same second with identical amounts, which is rare enough
// to accept given the alternative (no dedup at all).
func SyntheticExternalID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0}) // separator so "ab"+"c" can't collide with "a"+"bc"
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// parseDigits converts a non-negative digit-only string to int64.
func parseDigits(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
