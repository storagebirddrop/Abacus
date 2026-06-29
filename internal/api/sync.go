package api

import (
	"context"
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

type SyncHandler struct {
	svc     syncService
	jobRepo syncJobReader
}

func NewSyncHandler(svc syncService, jobRepo syncJobReader) *SyncHandler {
	return &SyncHandler{svc: svc, jobRepo: jobRepo}
}

func (h *SyncHandler) StartSync(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	jobID, err := h.svc.StartSync(r.Context(), walletID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *SyncHandler) ListSyncJobs(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
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
