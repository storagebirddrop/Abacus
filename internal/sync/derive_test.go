package sync

import (
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

// Known test vector: a standard mainnet wpkh descriptor.
// xpub from BIP32 test vector 1 (chain m).
func TestDeriveAddresses_wpkh(t *testing.T) {
	// Real BIP32 test vector xpub (chain m/0): this is a well-known public value.
	// xpub from BIP32 test vector 1, chain m (well-known, publicly documented).
	desc := "wpkh(xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8)"
	net := &chaincfg.MainNetParams

	receiving, change, err := DeriveAddresses(desc, net, 5)
	if err != nil {
		t.Fatalf("DeriveAddresses: %v", err)
	}
	if len(receiving) != 5 {
		t.Errorf("expected 5 receiving addresses, got %d", len(receiving))
	}
	if len(change) != 5 {
		t.Errorf("expected 5 change addresses, got %d", len(change))
	}
	// P2WPKH mainnet addresses start with bc1q.
	for i, addr := range receiving {
		if len(addr) < 4 || addr[:4] != "bc1q" {
			t.Errorf("receiving[%d] = %q: expected bc1q prefix", i, addr)
		}
	}
}

// TestDeriveAddresses_sortedmulti verifies that wsh(sortedmulti(...)) descriptors
// produce the correct native SegWit P2WSH address.
//
// The two xpubs below are BIP32 test vectors 1 and 2 (chain m), both well-known
// public values. The expected address was independently verified by
// reconstructing the BIP67-sorted witness script from scratch (manual pubkey
// derivation + manual sort.Slice + manual OP_2 <keys> OP_2 OP_CHECKMULTISIG
// script assembly) in a separate code path from deriveMultisig/
// p2wshMultisigAddr, then confirming both produce the same address.
func TestDeriveAddresses_sortedmulti(t *testing.T) {
	// BIP32 test vector xpubs (publicly documented, no private key exposure):
	//   TV1: xpub661MyMwAqRbcFtXgS5... (chain m)
	//   TV2: xpub661MyMwAqRbcFW31Y... (chain m)
	xpub1 := "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8"
	xpub2 := "xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB"

	desc := "wsh(sortedmulti(2," + xpub1 + "/0/*," + xpub2 + "/0/*))"
	net := &chaincfg.MainNetParams

	receiving, change, err := DeriveAddresses(desc, net, 3)
	if err != nil {
		t.Fatalf("DeriveAddresses sortedmulti: %v", err)
	}
	if len(receiving) != 3 {
		t.Errorf("expected 3 receiving, got %d", len(receiving))
	}
	if len(change) != 3 {
		t.Errorf("expected 3 change, got %d", len(change))
	}
	// Independently reconstructed expected value for receiving[0] (index 0).
	const wantFirstReceiving = "bc1qpsnmhwhv2cvnyp25efl8xvpyyenc8qr7zt0fj47dklj0txpc4e0q7vlzgy"
	if receiving[0] != wantFirstReceiving {
		t.Errorf("receiving[0] = %q, want %q (independently reconstructed BIP67 script)", receiving[0], wantFirstReceiving)
	}
	// P2WSH mainnet addresses: bech32, start with "bc1q", 62 chars.
	for i, addr := range receiving {
		if !strings.HasPrefix(addr, "bc1q") || len(addr) != 62 {
			t.Errorf("receiving[%d] = %q: expected bc1q P2WSH address (62 chars)", i, addr)
		}
	}
	for i, addr := range change {
		if !strings.HasPrefix(addr, "bc1q") || len(addr) != 62 {
			t.Errorf("change[%d] = %q: expected bc1q P2WSH address (62 chars)", i, addr)
		}
	}
	// Receiving addresses must all be distinct.
	seen := make(map[string]bool)
	for _, addr := range receiving {
		if seen[addr] {
			t.Errorf("duplicate receiving address: %s", addr)
		}
		seen[addr] = true
	}
}

// TestDeriveAddresses_multi_unsorted verifies that wsh(multi(...)) preserves
// descriptor key order (no BIP67 sort), producing a different address than
// wsh(sortedmulti(...)) for the same two keys -- proving the `sorted` flag
// actually changes behavior rather than being a no-op.
func TestDeriveAddresses_multi_unsorted(t *testing.T) {
	xpub1 := "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8"
	xpub2 := "xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB"

	descSorted := "wsh(sortedmulti(2," + xpub1 + "/0/*," + xpub2 + "/0/*))"
	descUnsorted := "wsh(multi(2," + xpub1 + "/0/*," + xpub2 + "/0/*))"
	net := &chaincfg.MainNetParams

	recvSorted, _, err := DeriveAddresses(descSorted, net, 1)
	if err != nil {
		t.Fatalf("sortedmulti: %v", err)
	}
	recvUnsorted, _, err := DeriveAddresses(descUnsorted, net, 1)
	if err != nil {
		t.Fatalf("multi: %v", err)
	}

	const wantUnsortedFirstReceiving = "bc1q008ztx6a24axhk6srvfg2f48nzczjhl6u5xr3nwl9a2kcd3q6xtq632a7g"
	if recvUnsorted[0] != wantUnsortedFirstReceiving {
		t.Errorf("unsorted receiving[0] = %q, want %q", recvUnsorted[0], wantUnsortedFirstReceiving)
	}
	// For these two test-vector keys, BIP67 sorting actually swaps their
	// order, so sorted and unsorted must diverge. If they ever produced the
	// same address, the sort would no longer be provably applied.
	if recvSorted[0] == recvUnsorted[0] {
		t.Fatal("sortedmulti and multi produced the same address for keys that BIP67-sort differently — the sort may not be applied")
	}
}

// TestDeriveAddresses_multisig_thresholdTooLarge verifies that a multisig
// descriptor with more than 16 keys or a threshold above 16 is rejected.
// p2wshMultisigAddr encodes threshold/n as single OP_N opcodes (OP_1..OP_16
// only); above 16 the same arithmetic lands on unrelated opcodes (e.g. n=17
// computes to OP_NOP1, not a number push) and would otherwise silently
// produce a malformed witness script instead of erroring.
func TestDeriveAddresses_multisig_thresholdTooLarge(t *testing.T) {
	xpub := "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8"
	keyExpr := xpub + "/0/*"

	desc := "wsh(sortedmulti(17"
	for i := 0; i < 17; i++ {
		desc += "," + keyExpr
	}
	desc += "))"

	_, _, err := DeriveAddresses(desc, &chaincfg.MainNetParams, 1)
	if err == nil {
		t.Fatal("expected an error for a 17-of-17 descriptor (exceeds OP_16), got nil")
	}
}

// TestDeriveAddresses_unsupported verifies that truly unsupported descriptors
// return an error.
func TestDeriveAddresses_unsupported(t *testing.T) {
	desc := "tr(xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8)"
	_, _, err := DeriveAddresses(desc, &chaincfg.MainNetParams, 5)
	if err == nil {
		t.Fatal("expected error for taproot descriptor, got nil")
	}
}

func TestDeriveAddresses_noXpub(t *testing.T) {
	desc := "wpkh(not-an-xpub)"
	_, _, err := DeriveAddresses(desc, &chaincfg.MainNetParams, 3)
	if err == nil {
		t.Fatal("expected error for missing xpub, got nil")
	}
}
