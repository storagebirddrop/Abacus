package ledger

import (
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

// SpentKey identifies a UTXO that this transaction spends.
type SpentKey struct {
	Txid string
	Vout int
}

// Build derives immutable LedgerEntries and UTXOs from a transaction and its
// wallet-annotated inputs/outputs.
//
// Accounting logic:
//   - Receive  (net > 0): one credit entry for the net sats received.
//   - Send     (net < 0): one debit for sats sent to recipient(s),
//     plus a separate fee debit when fee > 0.
//   - Self-transfer / consolidation (net == 0): fee debit only (if any).
func Build(
	tx *domain.Transaction,
	inputs []domain.TransactionInput,
	outputs []domain.TransactionOutput,
) (entries []domain.LedgerEntry, utxos []domain.UTXO, spent []SpentKey) {

	var myIn, myOut int64
	for _, in := range inputs {
		if in.IsMine {
			myIn += in.Sats
		}
	}
	for _, out := range outputs {
		if out.IsMine {
			myOut += out.Sats
		}
	}

	net := myOut - myIn
	now := time.Now().UTC()

	switch {
	case net > 0:
		entries = append(entries, domain.LedgerEntry{
			ID:            uuid.New().String(),
			WalletID:      tx.WalletID,
			TransactionID: tx.ID,
			Type:          domain.EntryTypeCredit,
			Sats:          net,
			Category:      domain.CategoryUnknown,
			CreatedAt:     now,
		})

	case net < 0:
		sent := -net - tx.FeeSats
		if sent < 0 {
			sent = 0
		}
		if sent > 0 {
			entries = append(entries, domain.LedgerEntry{
				ID:            uuid.New().String(),
				WalletID:      tx.WalletID,
				TransactionID: tx.ID,
				Type:          domain.EntryTypeDebit,
				Sats:          sent,
				Category:      domain.CategoryUnknown,
				CreatedAt:     now,
			})
		}
		if tx.FeeSats > 0 {
			entries = append(entries, domain.LedgerEntry{
				ID:            uuid.New().String(),
				WalletID:      tx.WalletID,
				TransactionID: tx.ID,
				Type:          domain.EntryTypeDebit,
				Sats:          tx.FeeSats,
				Category:      domain.CategoryFee,
				CreatedAt:     now,
			})
		}
	}

	// Fee-only for self-transfers (consolidation: net == 0, but wallet paid fee).
	if net == 0 && tx.FeeSats > 0 && myIn > 0 {
		entries = append(entries, domain.LedgerEntry{
			ID:            uuid.New().String(),
			WalletID:      tx.WalletID,
			TransactionID: tx.ID,
			Type:          domain.EntryTypeDebit,
			Sats:          tx.FeeSats,
			Category:      domain.CategoryFee,
			CreatedAt:     now,
		})
	}

	// New UTXO for every wallet output.
	for _, out := range outputs {
		if !out.IsMine {
			continue
		}
		utxos = append(utxos, domain.UTXO{
			ID:          uuid.New().String(),
			WalletID:    tx.WalletID,
			Txid:        tx.Txid,
			Vout:        out.Vout,
			Sats:        out.Sats,
			Address:     out.Address,
			BlockHeight: tx.BlockHeight,
			BlockTime:   tx.BlockTime,
			Spent:       false,
		})
	}

	// Record which UTXOs this transaction spends.
	for _, in := range inputs {
		if in.IsMine {
			spent = append(spent, SpentKey{Txid: in.PrevTxid, Vout: in.PrevVout})
		}
	}

	return entries, utxos, spent
}
