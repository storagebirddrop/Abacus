package sync

import (
	"fmt"
	"strings"

	"crypto/sha256"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// addrType identifies the address script type inferred from the descriptor prefix.
type addrType int

const (
	addrP2WPKH     addrType = iota // wpkh(...)
	addrP2SHP2WPKH                 // sh(wpkh(...))
	addrP2PKH                      // pkh(...)
)

// DeriveAddresses derives the first upTo receiving and change addresses from a
// single-sig output descriptor. Supports wpkh, sh(wpkh), and pkh descriptors.
// Returns an error for multisig or unrecognised descriptor types.
func DeriveAddresses(descriptor string, network *chaincfg.Params, upTo int) (receiving, change []string, err error) {
	desc := strings.TrimSpace(descriptor)
	// Strip checksum (#...)
	if idx := strings.LastIndex(desc, "#"); idx >= 0 {
		desc = desc[:idx]
	}
	lower := strings.ToLower(desc)

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
		if len(snip) > 40 {
			snip = snip[:40]
		}
		return nil, nil, fmt.Errorf("unsupported descriptor (multisig requires Phase 7+): %s", snip)
	}

	xpubStr, err := extractXpub(desc)
	if err != nil {
		return nil, nil, err
	}

	masterKey, err := hdkeychain.NewKeyFromString(xpubStr)
	if err != nil {
		// Some wallets export zpub/ypub; normalise to xpub version bytes.
		norm, nerr := normalizeToXpub(xpubStr)
		if nerr != nil {
			return nil, nil, fmt.Errorf("parse xpub %q: %w", xpubStr[:min(20, len(xpubStr))], err)
		}
		masterKey, err = hdkeychain.NewKeyFromString(norm)
		if err != nil {
			return nil, nil, fmt.Errorf("parse normalized xpub: %w", err)
		}
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
