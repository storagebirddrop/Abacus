package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/importer"
)

type walletStore interface {
	Create(ctx context.Context, w *domain.Wallet) error
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
	List(ctx context.Context) ([]*domain.Wallet, error)
	Delete(ctx context.Context, id string) error
}

type txStore interface {
	List(ctx context.Context, walletID string, limit, offset int) ([]*domain.Transaction, int, error)
	GetByTxid(ctx context.Context, walletID, txid string) (*domain.Transaction, error)
	GetInputsByTransactionID(ctx context.Context, txID string) ([]*domain.TransactionInput, error)
	GetOutputsByTransactionID(ctx context.Context, txID string) ([]*domain.TransactionOutput, error)
}

type txLedgerStore interface {
	ListByTransaction(ctx context.Context, walletID, transactionID string) ([]*domain.LedgerEntry, error)
	UpdateMetadata(ctx context.Context, tx *sql.Tx, id string, category domain.Category, note, counterpartyID string) error
}

type txJournalStore interface {
	Insert(ctx context.Context, tx *sql.Tx, e *domain.JournalEntry) error
}

type txDBStore interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type jobStore interface {
	GetByID(ctx context.Context, id string) (*domain.ImportJob, error)
	ListByWallet(ctx context.Context, walletID string) ([]*domain.ImportJob, error)
}

type labelStore interface {
	ListByWallet(ctx context.Context, walletID string) ([]*domain.Label, error)
	UpsertWithTx(ctx context.Context, tx *sql.Tx, l *domain.Label) error
}

type importService interface {
	Run(ctx context.Context, walletID, filename string, data []byte) (*domain.ImportJob, error)
}

type WalletHandler struct {
	wallets   walletStore
	txs       txStore
	ledger    txLedgerStore
	journal   txJournalStore
	db        txDBStore
	jobs      jobStore
	labels    labelStore
	importSvc importService
}

func NewWalletHandler(w walletStore, tx txStore, ledger txLedgerStore, journal txJournalStore, db txDBStore, j jobStore, l labelStore, svc importService) *WalletHandler {
	return &WalletHandler{wallets: w, txs: tx, ledger: ledger, journal: journal, db: db, jobs: j, labels: l, importSvc: svc}
}

func (h *WalletHandler) List(w http.ResponseWriter, r *http.Request) {
	wallets, err := h.wallets.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if wallets == nil {
		wallets = []*domain.Wallet{}
	}
	writeJSON(w, http.StatusOK, wallets)
}

func (h *WalletHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Descriptor  string `json:"descriptor"`
		Fingerprint string `json:"fingerprint"`
		Network     string `json:"network"`
		Source      string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" || req.Descriptor == "" {
		writeError(w, http.StatusBadRequest, errors.New("name and descriptor are required"))
		return
	}
	network := domain.Network(req.Network)
	if network == "" {
		network = domain.NetworkMainnet
	}
	source := domain.WalletSource(req.Source)
	if source == "" {
		source = domain.WalletSourceManual
	}

	walletType := domain.WalletTypeSinglesig
	if strings.Contains(strings.ToLower(req.Descriptor), "sortedmulti") ||
		strings.Contains(strings.ToLower(req.Descriptor), "multi(") {
		walletType = domain.WalletTypeMultisig
	}

	wallet := &domain.Wallet{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Descriptor:  req.Descriptor,
		Fingerprint: req.Fingerprint,
		Type:        walletType,
		Network:     network,
		Source:      source,
	}
	if err := h.wallets.Create(r.Context(), wallet); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, wallet)
}

func (h *WalletHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "walletID")
	wallet, err := h.wallets.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, errors.New("wallet not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, wallet)
}

func (h *WalletHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "walletID")
	if err := h.wallets.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WalletHandler) Import(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")

	// Verify wallet exists
	if _, err := h.wallets.GetByID(r.Context(), walletID); err != nil {
		writeError(w, http.StatusNotFound, errors.New("wallet not found"))
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid multipart form"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("file field required"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	job, err := h.importSvc.Run(r.Context(), walletID, header.Filename, data)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func (h *WalletHandler) ListImportJobs(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	jobs, err := h.jobs.ListByWallet(r.Context(), walletID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if jobs == nil {
		jobs = []*domain.ImportJob{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *WalletHandler) GetImportJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")
	job, err := h.jobs.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, errors.New("job not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *WalletHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 50
	}
	offset := (page - 1) * limit

	txs, total, err := h.txs.List(r.Context(), walletID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if txs == nil {
		txs = []*domain.Transaction{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  txs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *WalletHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	txid := chi.URLParam(r, "txid")

	tx, err := h.txs.GetByTxid(r.Context(), walletID, txid)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, errors.New("transaction not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	inputs, _ := h.txs.GetInputsByTransactionID(r.Context(), tx.ID)
	outputs, _ := h.txs.GetOutputsByTransactionID(r.Context(), tx.ID)

	writeJSON(w, http.StatusOK, map[string]any{
		"transaction": tx,
		"inputs":      inputs,
		"outputs":     outputs,
	})
}

func (h *WalletHandler) ListLabels(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	labels, err := h.labels.ListByWallet(r.Context(), walletID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if labels == nil {
		labels = []*domain.Label{}
	}
	writeJSON(w, http.StatusOK, labels)
}

func (h *WalletHandler) CreateLabel(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")

	var req struct {
		Type      string `json:"type"`
		Ref       string `json:"ref"`
		Label     string `json:"label"`
		Origin    string `json:"origin,omitempty"`
		Spendable *bool  `json:"spendable,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid JSON"))
		return
	}
	switch req.Type {
	case "tx", "addr", "xpub", "input", "output":
	default:
		writeError(w, http.StatusBadRequest, errors.New("type must be tx, addr, xpub, input, or output"))
		return
	}
	if req.Ref == "" || req.Label == "" {
		writeError(w, http.StatusBadRequest, errors.New("ref and label are required"))
		return
	}

	l := &domain.Label{
		WalletID:  walletID,
		Type:      req.Type,
		Ref:       req.Ref,
		Label:     req.Label,
		Origin:    req.Origin,
		Spendable: req.Spendable,
	}

	dbTx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer dbTx.Rollback()

	if err := h.labels.UpsertWithTx(r.Context(), dbTx, l); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := dbTx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (h *WalletHandler) ExportLabels(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	labels, err := h.labels.ListByWallet(r.Context(), walletID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/x-jsonlines")
	w.Header().Set("Content-Disposition", `attachment; filename="labels.jsonl"`)
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	for _, lbl := range labels {
		obj := map[string]any{
			"type":  lbl.Type,
			"ref":   lbl.Ref,
			"label": lbl.Label,
		}
		if lbl.Origin != "" {
			obj["origin"] = lbl.Origin
		}
		if lbl.Spendable != nil {
			obj["spendable"] = *lbl.Spendable
		}
		_ = enc.Encode(obj)
	}
}

func (h *WalletHandler) PatchTransaction(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	txid := chi.URLParam(r, "txid")

	tx, err := h.txs.GetByTxid(r.Context(), walletID, txid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("transaction not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	var body struct {
		Category       *string `json:"category"`
		Note           *string `json:"note"`
		CounterpartyID *string `json:"counterparty_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid JSON"))
		return
	}
	if body.Category == nil && body.Note == nil && body.CounterpartyID == nil {
		writeError(w, http.StatusBadRequest, errors.New("no fields to update"))
		return
	}

	entries, err := h.ledger.ListByTransaction(r.Context(), walletID, tx.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	dbTx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer dbTx.Rollback() //nolint:errcheck

	now := time.Now().UTC()
	for _, entry := range entries {
		newCategory := entry.Category
		newNote := entry.Note
		newCounterpartyID := entry.CounterpartyID

		if body.Category != nil && string(entry.Category) != *body.Category {
			j := &domain.JournalEntry{
				ID:            uuid.New().String(),
				LedgerEntryID: entry.ID,
				FieldChanged:  "category",
				OldValue:      string(entry.Category),
				NewValue:      *body.Category,
				Reason:        "user edit",
				CreatedAt:     now,
			}
			if err := h.journal.Insert(r.Context(), dbTx, j); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			newCategory = domain.Category(*body.Category)
		}
		if body.Note != nil && entry.Note != *body.Note {
			j := &domain.JournalEntry{
				ID:            uuid.New().String(),
				LedgerEntryID: entry.ID,
				FieldChanged:  "note",
				OldValue:      entry.Note,
				NewValue:      *body.Note,
				Reason:        "user edit",
				CreatedAt:     now,
			}
			if err := h.journal.Insert(r.Context(), dbTx, j); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			newNote = *body.Note
		}
		if body.CounterpartyID != nil && entry.CounterpartyID != *body.CounterpartyID {
			j := &domain.JournalEntry{
				ID:            uuid.New().String(),
				LedgerEntryID: entry.ID,
				FieldChanged:  "counterparty_id",
				OldValue:      entry.CounterpartyID,
				NewValue:      *body.CounterpartyID,
				Reason:        "user edit",
				CreatedAt:     now,
			}
			if err := h.journal.Insert(r.Context(), dbTx, j); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			newCounterpartyID = *body.CounterpartyID
		}
		if err := h.ledger.UpdateMetadata(r.Context(), dbTx, entry.ID, newCategory, newNote, newCounterpartyID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	if err := dbTx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// keep importer import used
var _ = importer.Registry
