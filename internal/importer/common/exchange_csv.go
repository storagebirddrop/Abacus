// Package common provides shared parsing utilities for exchange CSV importers.
package common

import "strings"

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
