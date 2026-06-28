package common_test

import (
	"strings"
	"testing"

	"github.com/storagebirddrop/abacus/internal/importer/common"
)

const realBSMS = `BSMS 1.0
wsh(sortedmulti(3,[b1631dce/48'/0'/0'/2']xpub6FK6pskKTaCVxrhwuUrWSmZBPs3YU4pEFvwxjkGu8iPuufwsqGydkWatjX5zNG7AyCkqsAXd7HyNhGmr9NGiqdvZJRpkMANpgBm1gjStcdr/*,[5624c2be/48'/0'/0'/2']xpub6EirFr47HqwrZT1dTdvLk5J5ZZsRPKfSrTvtm9y13K9H3DE9Btd16f3MfhneT5y8VjPKnHNjtRjNwsvsm2E4pFW9FcwvzgE9n24gWsRzAcX/*,[787baec8/48'/0'/0'/2']xpub6F8Ld31MaLCq39j7bfmwrN4LQqMgo1Uvx8mLSNWL9wJ4EEHb8KStCrGT1tDKs6sBD46XpzJ95ou7Cwrr56gjRj7MZZGauaoZoGn6cSm22vv/*,[524c7c5d/48'/0'/0'/2']xpub6DyAobAESpGHZbgSP9TqZFAuTNN11tLXPBaq5drjhAi8rPBNYZprsYLNfptFT3thAdiTUTMb7FoGhCfPHEMj3FSL123s9Bj1AuyhFrV3WhV/*,[154caf25/48'/0'/0'/2']xpub6F7JnTXuWqXzNpx1q9ZL88pc4kiHpsmvZpiatkZSdDmXKqv3x6ZgTcxvuhiYUsJNPxixgWDVjVWtZuM3kWHih6sMdyYCZghM2KCFSqn4ivb/*))#xy0rh9xl
No path restrictions
bc1qgwdq2tu76k4afak3erhwx7wzr722ht7fk8hw9xe3p6ws8l988qtsv4mcaw`

func TestParseBSMS(t *testing.T) {
	rec, err := common.ParseBSMS(strings.NewReader(realBSMS))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Version != "BSMS 1.0" {
		t.Errorf("version = %q, want %q", rec.Version, "BSMS 1.0")
	}
	if !strings.HasPrefix(rec.Descriptor, "wsh(sortedmulti") {
		t.Errorf("descriptor missing wsh(sortedmulti prefix")
	}
	if rec.PathRestrictions != "No path restrictions" {
		t.Errorf("path restrictions = %q", rec.PathRestrictions)
	}
	if rec.FirstAddress != "bc1qgwdq2tu76k4afak3erhwx7wzr722ht7fk8hw9xe3p6ws8l988qtsv4mcaw" {
		t.Errorf("first address = %q", rec.FirstAddress)
	}
}

func TestExtractFingerprints(t *testing.T) {
	fps := common.ExtractFingerprints(realBSMS)
	expected := []string{"b1631dce", "5624c2be", "787baec8", "524c7c5d", "154caf25"}
	if len(fps) != len(expected) {
		t.Fatalf("got %d fingerprints, want %d: %v", len(fps), len(expected), fps)
	}
	for i, fp := range fps {
		if fp != expected[i] {
			t.Errorf("fingerprint[%d] = %q, want %q", i, fp, expected[i])
		}
	}
}

func TestIsBSMS(t *testing.T) {
	if !common.IsBSMS(strings.NewReader(realBSMS)) {
		t.Error("should detect BSMS file")
	}
	if common.IsBSMS(strings.NewReader(`{"type":"tx"}`)) {
		t.Error("should not detect JSON as BSMS")
	}
}

func TestExtractDescriptor(t *testing.T) {
	d := `wsh(sortedmulti(3,...))#xy0rh9xl`
	got := common.ExtractDescriptor(d)
	if got != "wsh(sortedmulti(3,...))" {
		t.Errorf("got %q", got)
	}
}
