package ledger_test

import (
	"testing"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/ledger"
)

func baseTx() *domain.Transaction {
	return &domain.Transaction{
		ID:       "tx-uuid",
		WalletID: "wallet-1",
		Txid:     "aaaa",
		FeeSats:  1000,
		BlockTime: time.Unix(1_700_000_000, 0).UTC(),
	}
}

func TestBuild_Receive(t *testing.T) {
	tx := baseTx()
	tx.FeeSats = 0 // sender pays fee on receive

	outputs := []domain.TransactionOutput{
		{Vout: 0, Sats: 100_000, Address: "bc1qreceive", IsMine: true},
		{Vout: 1, Sats: 50_000, Address: "bc1qother", IsMine: false},
	}

	entries, utxos, spent := ledger.Build(tx, nil, outputs)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Type != domain.EntryTypeCredit {
		t.Errorf("type = %q, want credit", e.Type)
	}
	if e.Sats != 100_000 {
		t.Errorf("sats = %d, want 100000", e.Sats)
	}
	if e.Category != domain.CategoryUnknown {
		t.Errorf("category = %q", e.Category)
	}

	if len(utxos) != 1 {
		t.Fatalf("expected 1 utxo, got %d", len(utxos))
	}
	if utxos[0].Sats != 100_000 || utxos[0].Vout != 0 {
		t.Errorf("utxo wrong: %+v", utxos[0])
	}
	if len(spent) != 0 {
		t.Errorf("expected no spent keys on receive, got %d", len(spent))
	}
}

func TestBuild_Send(t *testing.T) {
	tx := baseTx()
	tx.FeeSats = 1_500

	inputs := []domain.TransactionInput{
		{PrevTxid: "prev", PrevVout: 0, Sats: 500_000, IsMine: true},
	}
	outputs := []domain.TransactionOutput{
		{Vout: 0, Sats: 498_500, Address: "bc1qrecipient", IsMine: false},
		{Vout: 1, Sats: 1_000, Address: "bc1qchange", IsMine: true},
	}

	// myIn=500000, myOut=1000, net=-499000
	// sent = 499000 - 1500 = 497500
	entries, utxos, spent := ledger.Build(tx, inputs, outputs)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (send + fee), got %d", len(entries))
	}
	sendEntry := entries[0]
	feeEntry := entries[1]
	if sendEntry.Type != domain.EntryTypeDebit || sendEntry.Sats != 497_500 {
		t.Errorf("send entry: %+v", sendEntry)
	}
	if feeEntry.Type != domain.EntryTypeDebit || feeEntry.Sats != 1_500 || feeEntry.Category != domain.CategoryFee {
		t.Errorf("fee entry: %+v", feeEntry)
	}

	if len(utxos) != 1 || utxos[0].Sats != 1_000 {
		t.Errorf("expected 1 change utxo of 1000 sats, got %+v", utxos)
	}
	if len(spent) != 1 || spent[0].Txid != "prev" || spent[0].Vout != 0 {
		t.Errorf("expected 1 spent key, got %+v", spent)
	}
}

func TestBuild_SelfTransfer(t *testing.T) {
	tx := baseTx()
	tx.FeeSats = 500

	inputs := []domain.TransactionInput{
		{PrevTxid: "prev", PrevVout: 0, Sats: 200_000, IsMine: true},
	}
	outputs := []domain.TransactionOutput{
		{Vout: 0, Sats: 199_500, Address: "bc1qself", IsMine: true},
	}

	// net = 199500 - 200000 = -500 == -fee → sent = 0, only fee entry
	entries, _, _ := ledger.Build(tx, inputs, outputs)

	if len(entries) != 1 {
		t.Fatalf("expected 1 fee entry for self-transfer, got %d", len(entries))
	}
	if entries[0].Category != domain.CategoryFee || entries[0].Sats != 500 {
		t.Errorf("fee entry: %+v", entries[0])
	}
}

func TestBuild_NoWalletActivity(t *testing.T) {
	tx := baseTx()
	outputs := []domain.TransactionOutput{
		{Vout: 0, Sats: 100_000, Address: "bc1qother", IsMine: false},
	}
	entries, utxos, spent := ledger.Build(tx, nil, outputs)
	if len(entries) != 0 || len(utxos) != 0 || len(spent) != 0 {
		t.Errorf("expected nothing for non-wallet tx")
	}
}
