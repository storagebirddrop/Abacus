package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type syncService interface {
	StartSync(ctx context.Context, walletID string) (jobID string, err error)
}

type syncJobReader interface {
	Get(ctx context.Context, id string) (*domain.SyncJob, error)
	ListByWallet(ctx context.Context, walletID string) ([]*domain.SyncJob, error)
}

type syncWalletRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
}

type SyncHandler struct {
	svc        syncService
	jobRepo    syncJobReader
	walletRepo syncWalletRepo
}

func NewSyncHandler(svc syncService, jobRepo syncJobReader, walletRepo syncWalletRepo) *SyncHandler {
	return &SyncHandler{svc: svc, jobRepo: jobRepo, walletRepo: walletRepo}
}

// requireWallet writes a 404 (or 500) and returns false when the wallet is
// missing, so handlers don't conflate "no such wallet" with other errors.
func (h *SyncHandler) requireWallet(w http.ResponseWriter, r *http.Request, walletID string) bool {
	_, err := h.walletRepo.GetByID(r.Context(), walletID)
	if err == nil {
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "wallet not found"})
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return false
}

func (h *SyncHandler) StartSync(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	if !h.requireWallet(w, r, walletID) {
		return
	}
	jobID, err := h.svc.StartSync(r.Context(), walletID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *SyncHandler) ListSyncJobs(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	if !h.requireWallet(w, r, walletID) {
		return
	}
	jobs, err := h.jobRepo.ListByWallet(r.Context(), walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if jobs == nil {
		jobs = []*domain.SyncJob{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *SyncHandler) GetSyncJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	job, err := h.jobRepo.Get(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sync job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}
