package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type walletReader interface {
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
}

type ledgerReader interface {
	ListByWallet(ctx context.Context, walletID string, limit, offset int) ([]*domain.LedgerEntry, int, error)
	GetByID(ctx context.Context, id string) (*domain.LedgerEntry, error)
}

type journalReader interface {
	ListByLedgerEntry(ctx context.Context, ledgerEntryID string) ([]*domain.JournalEntry, error)
}

type utxoReader interface {
	ListByWallet(ctx context.Context, walletID string, unspentOnly bool) ([]*domain.UTXO, error)
}

type LedgerHandler struct {
	walletRepo  walletReader
	ledgerRepo  ledgerReader
	journalRepo journalReader
	utxoRepo    utxoReader
}

func NewLedgerHandler(walletRepo walletReader, ledgerRepo ledgerReader, journalRepo journalReader, utxoRepo utxoReader) *LedgerHandler {
	return &LedgerHandler{walletRepo: walletRepo, ledgerRepo: ledgerRepo, journalRepo: journalRepo, utxoRepo: utxoRepo}
}

func (h *LedgerHandler) ListLedger(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	if _, err := h.walletRepo.GetByID(r.Context(), walletID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("wallet not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 50
	}
	offset := (page - 1) * limit

	entries, total, err := h.ledgerRepo.ListByWallet(r.Context(), walletID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if entries == nil {
		entries = []*domain.LedgerEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  entries,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *LedgerHandler) GetLedgerEntry(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	entryID := chi.URLParam(r, "entryID")

	entry, err := h.ledgerRepo.GetByID(r.Context(), entryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("ledger entry not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if entry.WalletID != walletID {
		writeError(w, http.StatusNotFound, errors.New("ledger entry not found"))
		return
	}

	journal, err := h.journalRepo.ListByLedgerEntry(r.Context(), entryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if journal == nil {
		journal = []*domain.JournalEntry{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":               entry.ID,
		"wallet_id":        entry.WalletID,
		"transaction_id":   entry.TransactionID,
		"type":             entry.Type,
		"sats":             entry.Sats,
		"fiat_amount":      entry.FiatAmount,
		"fiat_currency":    entry.FiatCurrency,
		"price_snapshot_id": entry.PriceSnapshotID,
		"category":         entry.Category,
		"counterparty_id":  entry.CounterpartyID,
		"note":             entry.Note,
		"created_at":       entry.CreatedAt,
		"journal":          journal,
	})
}

func (h *LedgerHandler) ListUTXOs(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	if _, err := h.walletRepo.GetByID(r.Context(), walletID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("wallet not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	unspentOnly := r.URL.Query().Get("unspent") == "true"
	utxos, err := h.utxoRepo.ListByWallet(r.Context(), walletID, unspentOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if utxos == nil {
		utxos = []*domain.UTXO{}
	}
	writeJSON(w, http.StatusOK, utxos)
}
