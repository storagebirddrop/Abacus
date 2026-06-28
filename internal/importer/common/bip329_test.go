package common_test

import (
	"strings"
	"testing"

	"github.com/storagebirddrop/abacus/internal/importer/common"
)

const bip329Sample = `{"type":"tx","ref":"f4184fc596403b9d638783cf57adfe4c75c605f6356fbc91338530e9831e9e16","label":"Genesis tx"}
{"type":"addr","ref":"1A1zP1eP5QGefi2DMPTfTL5SLmv7Divfna","label":"Satoshi"}
{"type":"tx","ref":"abc123","label":"Coffee","origin":"m/84'/0'/0'/0/5"}
`

func TestParseBIP329(t *testing.T) {
	labels, errs := common.ParseBIP329("wallet-1", strings.NewReader(bip329Sample))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(labels) != 3 {
		t.Fatalf("got %d labels, want 3", len(labels))
	}

	if labels[0].Type != "tx" || labels[0].Ref != "f4184fc596403b9d638783cf57adfe4c75c605f6356fbc91338530e9831e9e16" {
		t.Errorf("label[0] = %+v", labels[0])
	}
	if labels[1].Type != "addr" || labels[1].Label != "Satoshi" {
		t.Errorf("label[1] = %+v", labels[1])
	}
	if labels[2].Origin != "m/84'/0'/0'/0/5" {
		t.Errorf("label[2] origin = %q", labels[2].Origin)
	}
	for _, l := range labels {
		if l.WalletID != "wallet-1" {
			t.Errorf("wallet_id not set: %q", l.WalletID)
		}
	}
}

func TestParseBIP329_InvalidLine(t *testing.T) {
	input := `{"type":"tx","ref":"abc","label":"ok"}
not valid json
{"type":"addr","ref":"xyz","label":"ok2"}
`
	labels, errs := common.ParseBIP329("w", strings.NewReader(input))
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
	if len(labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels))
	}
}
