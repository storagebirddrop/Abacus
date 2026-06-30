package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

func TestTransactionRepo_ListFiltered(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()

	walletID := uuid.New().String()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO wallets (id, name, descriptor, fingerprint, type, network, source, created_at, updated_at)
		 VALUES (?, 'test', '', '', 'singlesig', 'mainnet', 'manual', 0, 0)`, walletID); err != nil {
		t.Fatalf("insert wallet: %v", err)
	}

	// Three transactions: differing block_time, fee, confirmed, and txid.
	rows := []struct {
		txid      string
		blockTime int64
		fee       int64
		confirmed int
	}{
		{"aaa111", 100, 500, 1},
		{"bbb222", 300, 100, 1},
		{"ccc333", 200, 900, 0},
	}
	for _, r := range rows {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO transactions (id, wallet_id, txid, block_height, block_time, fee_sats, confirmed, created_at)
			 VALUES (?, ?, ?, 0, ?, ?, ?, 0)`,
			uuid.New().String(), walletID, r.txid, r.blockTime, r.fee, r.confirmed); err != nil {
			t.Fatalf("insert tx %s: %v", r.txid, err)
		}
	}

	repo := NewTransactionRepo(db)

	// Default: sort by date desc → bbb(300), ccc(200), aaa(100).
	txs, total, err := repo.ListFiltered(ctx, walletID, domain.TxFilter{})
	if err != nil {
		t.Fatalf("ListFiltered: %v", err)
	}
	if total != 3 || len(txs) != 3 {
		t.Fatalf("expected 3 total/returned, got total=%d len=%d", total, len(txs))
	}
	if txs[0].Txid != "bbb222" || txs[2].Txid != "aaa111" {
		t.Errorf("default date-desc order wrong: %s … %s", txs[0].Txid, txs[2].Txid)
	}

	// Search: substring on txid.
	txs, total, err = repo.ListFiltered(ctx, walletID, domain.TxFilter{Search: "bbb"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(txs) != 1 || txs[0].Txid != "bbb222" {
		t.Errorf("search bbb expected 1 match, got total=%d txs=%v", total, txs)
	}

	// Status: pending only → ccc333.
	txs, total, err = repo.ListFiltered(ctx, walletID, domain.TxFilter{Status: "pending"})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if total != 1 || len(txs) != 1 || txs[0].Txid != "ccc333" {
		t.Errorf("status pending expected ccc333, got total=%d txs=%v", total, txs)
	}

	// Sort by fee asc → bbb(100), aaa(500), ccc(900).
	txs, _, err = repo.ListFiltered(ctx, walletID, domain.TxFilter{Sort: "fee", Dir: "asc"})
	if err != nil {
		t.Fatalf("sort fee: %v", err)
	}
	if txs[0].Txid != "bbb222" || txs[2].Txid != "ccc333" {
		t.Errorf("fee-asc order wrong: %s … %s", txs[0].Txid, txs[2].Txid)
	}

	// Pagination: limit 2, offset 0 then 2; total stays 3.
	txs, total, err = repo.ListFiltered(ctx, walletID, domain.TxFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if total != 3 || len(txs) != 2 {
		t.Errorf("page 1 expected total=3 len=2, got total=%d len=%d", total, len(txs))
	}
	txs, _, err = repo.ListFiltered(ctx, walletID, domain.TxFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("paginate 2: %v", err)
	}
	if len(txs) != 1 {
		t.Errorf("page 2 expected len=1, got %d", len(txs))
	}
}
