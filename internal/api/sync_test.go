package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type stubSyncService struct {
	jobID string
	err   error
}

func (s *stubSyncService) StartSync(_ context.Context, _ string) (string, error) {
	return s.jobID, s.err
}

type stubSyncJobReader struct {
	jobs []*domain.SyncJob
}

func (s *stubSyncJobReader) Get(_ context.Context, _ string) (*domain.SyncJob, error) {
	return nil, sql.ErrNoRows
}
func (s *stubSyncJobReader) ListByWallet(_ context.Context, _ string) ([]*domain.SyncJob, error) {
	return s.jobs, nil
}

type stubSyncWalletRepo struct {
	exists bool
}

func (s *stubSyncWalletRepo) GetByID(_ context.Context, _ string) (*domain.Wallet, error) {
	if s.exists {
		return &domain.Wallet{ID: "w1"}, nil
	}
	return nil, sql.ErrNoRows
}

func syncReq(h http.HandlerFunc, method, walletID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/wallets/"+walletID+"/sync", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("walletID", walletID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestStartSync_404ForMissingWallet(t *testing.T) {
	h := NewSyncHandler(&stubSyncService{jobID: "job1"}, &stubSyncJobReader{}, &stubSyncWalletRepo{exists: false})
	if rec := syncReq(h.StartSync, "POST", "nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing wallet: got %d, want 404", rec.Code)
	}
}

func TestStartSync_AcceptedWhenWalletExists(t *testing.T) {
	h := NewSyncHandler(&stubSyncService{jobID: "job1"}, &stubSyncJobReader{}, &stubSyncWalletRepo{exists: true})
	if rec := syncReq(h.StartSync, "POST", "w1"); rec.Code != http.StatusAccepted {
		t.Fatalf("existing wallet: got %d, want 202", rec.Code)
	}
}

func TestListSyncJobs_404ForMissingWallet(t *testing.T) {
	h := NewSyncHandler(&stubSyncService{}, &stubSyncJobReader{}, &stubSyncWalletRepo{exists: false})
	if rec := syncReq(h.ListSyncJobs, "GET", "nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing wallet: got %d, want 404", rec.Code)
	}
}

func TestListSyncJobs_OKWhenWalletExists(t *testing.T) {
	h := NewSyncHandler(&stubSyncService{}, &stubSyncJobReader{}, &stubSyncWalletRepo{exists: true})
	if rec := syncReq(h.ListSyncJobs, "GET", "w1"); rec.Code != http.StatusOK {
		t.Fatalf("existing wallet: got %d, want 200", rec.Code)
	}
}
