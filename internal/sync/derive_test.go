package sync

import (
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

func TestDeriveAddresses_multisig_rejected(t *testing.T) {
	desc := "wsh(multi(2,[abcd1234/48h/0h/0h/2h]xpub.../0/*))"
	_, _, err := DeriveAddresses(desc, &chaincfg.MainNetParams, 5)
	if err == nil {
		t.Fatal("expected error for multisig descriptor, got nil")
	}
}

func TestDeriveAddresses_noXpub(t *testing.T) {
	desc := "wpkh(not-an-xpub)"
	_, _, err := DeriveAddresses(desc, &chaincfg.MainNetParams, 3)
	if err == nil {
		t.Fatal("expected error for missing xpub, got nil")
	}
}
