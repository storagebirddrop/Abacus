package nunchuk_test

import (
	"context"
	"strings"
	"testing"

	"github.com/storagebirddrop/abacus/internal/importer/nunchuk"
)

// realBSMS is the actual Nunchuk-exported BSMS file provided for testing.
const realBSMS = `BSMS 1.0
wsh(sortedmulti(3,[b1631dce/48'/0'/0'/2']xpub6FK6pskKTaCVxrhwuUrWSmZBPs3YU4pEFvwxjkGu8iPuufwsqGydkWatjX5zNG7AyCkqsAXd7HyNhGmr9NGiqdvZJRpkMANpgBm1gjStcdr/*,[5624c2be/48'/0'/0'/2']xpub6EirFr47HqwrZT1dTdvLk5J5ZZsRPKfSrTvtm9y13K9H3DE9Btd16f3MfhneT5y8VjPKnHNjtRjNwsvsm2E4pFW9FcwvzgE9n24gWsRzAcX/*,[787baec8/48'/0'/0'/2']xpub6F8Ld31MaLCq39j7bfmwrN4LQqMgo1Uvx8mLSNWL9wJ4EEHb8KStCrGT1tDKs6sBD46XpzJ95ou7Cwrr56gjRj7MZZGauaoZoGn6cSm22vv/*,[524c7c5d/48'/0'/0'/2']xpub6DyAobAESpGHZbgSP9TqZFAuTNN11tLXPBaq5drjhAi8rPBNYZprsYLNfptFT3thAdiTUTMb7FoGhCfPHEMj3FSL123s9Bj1AuyhFrV3WhV/*,[154caf25/48'/0'/0'/2']xpub6F7JnTXuWqXzNpx1q9ZL88pc4kiHpsmvZpiatkZSdDmXKqv3x6ZgTcxvuhiYUsJNPxixgWDVjVWtZuM3kWHih6sMdyYCZghM2KCFSqn4ivb/*))#xy0rh9xl
No path restrictions
bc1qgwdq2tu76k4afak3erhwx7wzr722ht7fk8hw9xe3p6ws8l988qtsv4mcaw`

func TestDetectBSMS(t *testing.T) {
	imp := nunchuk.New()
	if !imp.Detect("cjpf6vrk.bsms", strings.NewReader(realBSMS)) {
		t.Error("should detect Nunchuk BSMS file by extension + content")
	}
	if !imp.Detect("wallet.bsms", strings.NewReader(realBSMS)) {
		t.Error("should detect .bsms extension")
	}
}

func TestImportBSMS_FirstAddress(t *testing.T) {
	imp := nunchuk.New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(realBSMS))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(result.Addresses))
	}
	addr := result.Addresses[0]
	if addr.Address != "bc1qgwdq2tu76k4afak3erhwx7wzr722ht7fk8hw9xe3p6ws8l988qtsv4mcaw" {
		t.Errorf("wrong address: %s", addr.Address)
	}
	if addr.Type != "receive" {
		t.Errorf("type = %q, want receive", addr.Type)
	}
	if addr.WalletID != "wallet-1" {
		t.Errorf("wallet_id = %q", addr.WalletID)
	}
}

func TestImportBSMS_NoTransactions(t *testing.T) {
	imp := nunchuk.New()
	result, err := imp.Import(context.Background(), "wallet-1", strings.NewReader(realBSMS))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Transactions) != 0 {
		t.Errorf("expected 0 transactions from BSMS, got %d", len(result.Transactions))
	}
}

func TestImportTransactionHistory(t *testing.T) {
	const txJSON = `{
		"transactions": [
			{
				"txid": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"height": 840000,
				"block_time": 1713000000,
				"fee": 1500,
				"memo": "Hardware aankoop",
				"status": "CONFIRMED",
				"inputs": [
					{"txid": "dead","vout": 0,"value": 500000,"address": "bc1qinput","is_mine": true}
				],
				"outputs": [
					{"value": 498500,"address": "bc1qoutput","is_mine": false},
					{"value": 1000,"address": "bc1qchange","is_mine": true}
				]
			},
			{
				"txid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				"height": 0,
				"fee": 800,
				"status": "PENDING_CONFIRMATION",
				"inputs": [],
				"outputs": [{"value": 100000,"address": "bc1qreceive","is_mine": true}]
			}
		]
	}`

	imp := nunchuk.New()
	result, err := imp.Import(context.Background(), "wallet-x", strings.NewReader(txJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(result.Transactions))
	}

	tx0 := result.Transactions[0]
	if tx0.Txid != "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2" {
		t.Errorf("wrong txid: %s", tx0.Txid)
	}
	if tx0.BlockHeight != 840000 {
		t.Errorf("block_height = %d", tx0.BlockHeight)
	}
	if tx0.FeeSats != 1500 {
		t.Errorf("fee = %d, want 1500", tx0.FeeSats)
	}
	if !tx0.Confirmed {
		t.Error("tx0 should be confirmed")
	}

	tx1 := result.Transactions[1]
	if tx1.Confirmed {
		t.Error("tx1 should be unconfirmed (height=0, status=PENDING)")
	}

	// Memo → BIP329 label
	if len(result.Labels) != 1 {
		t.Fatalf("expected 1 label (from memo), got %d", len(result.Labels))
	}
	lbl := result.Labels[0]
	if lbl.Label != "Hardware aankoop" {
		t.Errorf("label = %q", lbl.Label)
	}
	if lbl.Type != "tx" || lbl.Ref != tx0.Txid {
		t.Errorf("label type/ref wrong: %s/%s", lbl.Type, lbl.Ref)
	}
	if lbl.Origin != "nunchuk" {
		t.Errorf("origin = %q, want nunchuk", lbl.Origin)
	}
}

func TestImportWalletConfig(t *testing.T) {
	const walletJSON = `{
		"name": "Vogelnestje",
		"wallet_type": "MULTI_SIG",
		"address_type": "NATIVE_SEGWIT",
		"m": 3,
		"n": 5,
		"descriptor": "wsh(sortedmulti(3,...))",
		"signers": [
			{"name": "Jade 1","xfp": "b1631dce","xpub": "xpub6FK...","derivation_path": "m/48h/0h/0h/2h","type": "HARDWARE"},
			{"name": "Jade 2","xfp": "5624c2be","xpub": "xpub6Ei...","derivation_path": "m/48h/0h/0h/2h","type": "HARDWARE"}
		]
	}`

	imp := nunchuk.New()
	if !imp.Detect("wallet.json", strings.NewReader(walletJSON)) {
		t.Error("should detect Nunchuk wallet config JSON")
	}

	result, err := imp.Import(context.Background(), "wallet-z", strings.NewReader(walletJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Wallet config alone has no transactions or labels
	if len(result.Transactions) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(result.Transactions))
	}
}

func TestDetectBIP329(t *testing.T) {
	const bip329 = `{"type":"tx","ref":"abc","label":"Coffee"}
{"type":"addr","ref":"bc1q","label":"Mine"}`

	imp := nunchuk.New()
	if !imp.Detect("labels.jsonl", strings.NewReader(bip329)) {
		t.Error("should detect BIP329 .jsonl")
	}
}

func TestImportBIP329_Labels(t *testing.T) {
	const bip329 = `{"type":"tx","ref":"txid1","label":"Salary"}
{"type":"addr","ref":"bc1qabc","label":"Work wallet"}
`
	imp := nunchuk.New()
	result, err := imp.Import(context.Background(), "wallet-q", strings.NewReader(bip329))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(result.Labels))
	}
	if result.Labels[0].Label != "Salary" {
		t.Errorf("label[0] = %q", result.Labels[0].Label)
	}
}

func TestImport_TimeField(t *testing.T) {
	// Some Nunchuk versions use "time" instead of "block_time"
	const txJSON = `{"transactions":[{"txid":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","height":830000,"time":1700000000,"fee":500,"status":"CONFIRMED","inputs":[],"outputs":[]}]}`
	imp := nunchuk.New()
	result, err := imp.Import(context.Background(), "w", strings.NewReader(txJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Transactions) != 1 {
		t.Fatalf("expected 1 tx, got %d", len(result.Transactions))
	}
	if result.Transactions[0].BlockTime.Unix() != 1700000000 {
		t.Errorf("block_time via 'time' field not parsed: %v", result.Transactions[0].BlockTime)
	}
}
