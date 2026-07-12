package sync

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// addrType identifies the address script type inferred from the descriptor prefix.
type addrType int

const (
	addrP2WPKH      addrType = iota // wpkh(...)
	addrP2SHP2WPKH                  // sh(wpkh(...))
	addrP2PKH                       // pkh(...)
	addrP2WSHMulti                  // wsh(multi(k,...)) or wsh(sortedmulti(k,...))
)

// DeriveAddresses derives the first upTo receiving and change addresses from an
// output descriptor. Supports wpkh, sh(wpkh), pkh, wsh(multi), and
// wsh(sortedmulti) descriptors.
func DeriveAddresses(descriptor string, network *chaincfg.Params, upTo int) (receiving, change []string, err error) {
	desc := strings.TrimSpace(descriptor)
	// Strip checksum (#...)
	if idx := strings.LastIndex(desc, "#"); idx >= 0 {
		desc = desc[:idx]
	}
	lower := strings.ToLower(desc)

	// Multisig: wsh(multi(...)) or wsh(sortedmulti(...))
	if strings.HasPrefix(lower, "wsh(multi(") || strings.HasPrefix(lower, "wsh(sortedmulti(") {
		sorted := strings.HasPrefix(lower, "wsh(sortedmulti(")
		return deriveMultisig(desc, network, upTo, sorted)
	}

	// Singlesig
	var at addrType
	switch {
	case strings.HasPrefix(lower, "sh(wpkh("):
		at = addrP2SHP2WPKH
	case strings.HasPrefix(lower, "wpkh("):
		at = addrP2WPKH
	case strings.HasPrefix(lower, "pkh("):
		at = addrP2PKH
	default:
		snip := desc
		if len(snip) > 60 {
			snip = snip[:60]
		}
		return nil, nil, fmt.Errorf("unsupported descriptor: %s", snip)
	}

	xpubStr, err := extractXpub(desc)
	if err != nil {
		return nil, nil, err
	}

	masterKey, err := parseExtendedKey(xpubStr)
	if err != nil {
		return nil, nil, err
	}

	deriveChain := func(chainIdx uint32) ([]string, error) {
		chainKey, err := masterKey.Derive(chainIdx)
		if err != nil {
			return nil, err
		}
		addrs := make([]string, 0, upTo)
		for i := uint32(0); i < uint32(upTo); i++ {
			childKey, err := chainKey.Derive(i)
			if err != nil {
				return nil, err
			}
			addr, err := keyToAddr(childKey, at, network)
			if err != nil {
				return nil, err
			}
			addrs = append(addrs, addr)
		}
		return addrs, nil
	}

	receiving, err = deriveChain(0)
	if err != nil {
		return nil, nil, fmt.Errorf("derive receiving addresses: %w", err)
	}
	change, err = deriveChain(1)
	if err != nil {
		return nil, nil, fmt.Errorf("derive change addresses: %w", err)
	}
	return receiving, change, nil
}

// deriveMultisig handles wsh(multi(k,...)) and wsh(sortedmulti(k,...)) descriptors.
//
// Descriptor form (abbreviated):
//
//	wsh(sortedmulti(k,[fp/path]xpub1/0/*,[fp/path]xpub2/0/*,...))
//
// The last two path components after each xpub (/chain/index) map to the
// BIP32 receive (chain=0) and change (chain=1) branches. The wildcard (*)
// is the per-address index we enumerate.
func deriveMultisig(desc string, network *chaincfg.Params, upTo int, sorted bool) (receiving, change []string, err error) {
	// Extract the inner content of wsh(...)
	inner, err := extractParenContent(desc, "wsh(")
	if err != nil {
		return nil, nil, err
	}

	// Extract the inner content of multi(...) or sortedmulti(...)
	prefix := "multi("
	if sorted {
		prefix = "sortedmulti("
	}
	multiInner, err := extractParenContent(inner, prefix)
	if err != nil {
		return nil, nil, err
	}

	// Split on commas at the top level (not inside brackets or parens).
	parts := splitTopLevel(multiInner)
	if len(parts) < 2 {
		return nil, nil, fmt.Errorf("invalid multi descriptor: expected threshold and at least one key")
	}

	threshold, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid threshold %q: %w", parts[0], err)
	}
	keyExprs := parts[1:]
	n := len(keyExprs)
	if threshold < 1 || threshold > n {
		return nil, nil, fmt.Errorf("invalid threshold %d for %d keys", threshold, n)
	}
	// p2wshMultisigAddr encodes both threshold and n as single OP_N opcodes
	// (OP_1..OP_16), which only cover 1-16. Above that, the same arithmetic
	// silently lands on unrelated opcodes (e.g. n=17 -> 0x61, OP_NOP1, not a
	// number push) and produces a malformed witness script with no error.
	if threshold > 16 || n > 16 {
		return nil, nil, fmt.Errorf("multisig threshold and key count must each be <= 16 (got %d-of-%d)", threshold, n)
	}

	// Parse each key expression into an hdkeychain.ExtendedKey.
	masterKeys := make([]*hdkeychain.ExtendedKey, n)
	for i, expr := range keyExprs {
		xpubStr, err := extractXpub(strings.TrimSpace(expr))
		if err != nil {
			return nil, nil, fmt.Errorf("key %d: %w", i, err)
		}
		k, err := parseExtendedKey(xpubStr)
		if err != nil {
			return nil, nil, fmt.Errorf("key %d: %w", i, err)
		}
		masterKeys[i] = k
	}

	deriveChain := func(chainIdx uint32) ([]string, error) {
		// Derive chain-level keys for all cosigners.
		chainKeys := make([]*hdkeychain.ExtendedKey, n)
		for i, mk := range masterKeys {
			ck, err := mk.Derive(chainIdx)
			if err != nil {
				return nil, fmt.Errorf("derive chain key %d: %w", i, err)
			}
			chainKeys[i] = ck
		}

		addrs := make([]string, 0, upTo)
		for idx := uint32(0); idx < uint32(upTo); idx++ {
			pubkeys := make([][]byte, n)
			for i, ck := range chainKeys {
				child, err := ck.Derive(idx)
				if err != nil {
					return nil, fmt.Errorf("derive index %d key %d: %w", idx, i, err)
				}
				pub, err := child.ECPubKey()
				if err != nil {
					return nil, fmt.Errorf("ECPubKey index %d key %d: %w", idx, i, err)
				}
				pubkeys[i] = pub.SerializeCompressed()
			}

			if sorted {
				sort.Slice(pubkeys, func(a, b int) bool {
					return bytes.Compare(pubkeys[a], pubkeys[b]) < 0
				})
			}

			addr, err := p2wshMultisigAddr(threshold, pubkeys, network)
			if err != nil {
				return nil, fmt.Errorf("build address index %d: %w", idx, err)
			}
			addrs = append(addrs, addr)
		}
		return addrs, nil
	}

	receiving, err = deriveChain(0)
	if err != nil {
		return nil, nil, fmt.Errorf("derive receiving addresses: %w", err)
	}
	change, err = deriveChain(1)
	if err != nil {
		return nil, nil, fmt.Errorf("derive change addresses: %w", err)
	}
	return receiving, change, nil
}

// p2wshMultisigAddr builds a native SegWit P2WSH address for a k-of-n multisig.
func p2wshMultisigAddr(threshold int, pubkeys [][]byte, net *chaincfg.Params) (string, error) {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_1 - 1 + byte(threshold))
	for _, pub := range pubkeys {
		builder.AddData(pub)
	}
	builder.AddOp(txscript.OP_1 - 1 + byte(len(pubkeys)))
	builder.AddOp(txscript.OP_CHECKMULTISIG)
	witnessScript, err := builder.Script()
	if err != nil {
		return "", fmt.Errorf("build witness script: %w", err)
	}

	scriptHash := sha256.Sum256(witnessScript)
	addr, err := btcutil.NewAddressWitnessScriptHash(scriptHash[:], net)
	if err != nil {
		return "", fmt.Errorf("build P2WSH address: %w", err)
	}
	return addr.EncodeAddress(), nil
}

// extractParenContent returns the content inside prefix(...) at the outermost
// nesting level. prefix must include the opening paren, e.g. "wsh(".
func extractParenContent(s, prefix string) (string, error) {
	lower := strings.ToLower(s)
	idx := strings.Index(lower, strings.ToLower(prefix))
	if idx < 0 {
		return "", fmt.Errorf("%q not found in descriptor", prefix)
	}
	start := idx + len(prefix)
	depth := 1
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[start:i], nil
			}
		}
	}
	return "", fmt.Errorf("unbalanced parentheses in descriptor")
}

// splitTopLevel splits s by commas that are not inside brackets [] or parens ().
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// extractXpub finds the xpub/ypub/zpub (or testnet variants) in a descriptor string.
func extractXpub(desc string) (string, error) {
	// An extended public key starts with [xyztuvXYZTUV]pub followed by base58 chars.
	const prefixes = "xXyYzZtTuUvV"
	start := -1
	for i, r := range desc {
		if strings.ContainsRune(prefixes, r) && i+4 < len(desc) && desc[i+1:i+4] == "pub" {
			start = i
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("no xpub/ypub/zpub found in descriptor")
	}
	// Read until a non-base58 character.
	end := start
	for end < len(desc) {
		c := desc[end]
		if isBase58(c) {
			end++
		} else {
			break
		}
	}
	return desc[start:end], nil
}

func isBase58(c byte) bool {
	const base58chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	for i := 0; i < len(base58chars); i++ {
		if base58chars[i] == c {
			return true
		}
	}
	return false
}

// parseExtendedKey parses an xpub/ypub/zpub string, normalising to xpub version
// bytes if needed.
func parseExtendedKey(xpubStr string) (*hdkeychain.ExtendedKey, error) {
	k, err := hdkeychain.NewKeyFromString(xpubStr)
	if err == nil {
		return k, nil
	}
	norm, nerr := normalizeToXpub(xpubStr)
	if nerr != nil {
		return nil, fmt.Errorf("parse xpub %q: %w", xpubStr[:min(20, len(xpubStr))], err)
	}
	k, err = hdkeychain.NewKeyFromString(norm)
	if err != nil {
		return nil, fmt.Errorf("parse normalized xpub: %w", err)
	}
	return k, nil
}

// keyToAddr converts an extended key to a Bitcoin address string.
func keyToAddr(key *hdkeychain.ExtendedKey, at addrType, net *chaincfg.Params) (string, error) {
	pub, err := key.ECPubKey()
	if err != nil {
		return "", err
	}
	pubBytes := pub.SerializeCompressed()

	switch at {
	case addrP2WPKH:
		h160 := btcutil.Hash160(pubBytes)
		addr, err := btcutil.NewAddressWitnessPubKeyHash(h160, net)
		if err != nil {
			return "", err
		}
		return addr.EncodeAddress(), nil

	case addrP2SHP2WPKH:
		h160 := btcutil.Hash160(pubBytes)
		wpkhAddr, err := btcutil.NewAddressWitnessPubKeyHash(h160, net)
		if err != nil {
			return "", err
		}
		redeemScript, err := txscript.PayToAddrScript(wpkhAddr)
		if err != nil {
			return "", err
		}
		addr, err := btcutil.NewAddressScriptHash(redeemScript, net)
		if err != nil {
			return "", err
		}
		return addr.EncodeAddress(), nil

	case addrP2PKH:
		addr, err := key.Address(net)
		if err != nil {
			return "", err
		}
		return addr.EncodeAddress(), nil
	}
	return "", fmt.Errorf("unknown address type")
}

// normalizeToXpub converts ypub/zpub/tpub/upub/vpub to mainnet xpub version bytes.
// Extended key encoding: version(4) + depth(1) + fingerprint(4) + childnum(4) +
// chaincode(32) + key(33) + checksum(4) = 82 bytes total.
// We replace the 4-byte version prefix and recompute the checksum.
func normalizeToXpub(key string) (string, error) {
	decoded := base58.Decode(key)
	// 78 payload bytes + 4 checksum bytes = 82 total.
	if len(decoded) < 82 {
		return "", fmt.Errorf("extended key too short: %d bytes", len(decoded))
	}
	// Replace version bytes with xpub mainnet 0x0488B21E.
	decoded[0], decoded[1], decoded[2], decoded[3] = 0x04, 0x88, 0xB2, 0x1E
	// Recompute checksum over the 78-byte payload.
	payload := decoded[:78]
	h1 := sha256.Sum256(payload)
	h2 := sha256.Sum256(h1[:])
	copy(decoded[78:], h2[:4])
	return base58.Encode(decoded), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
