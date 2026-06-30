package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type stubImportSvc struct{ called bool }

func (s *stubImportSvc) Run(_ context.Context, _, _ string, _ []byte) (*domain.ImportJob, error) {
	s.called = true
	return &domain.ImportJob{ID: "job1", Status: "running"}, nil
}

func importRouter(h *WalletHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/wallets/{walletID}/import", h.Import)
	return r
}

// multipartUpload builds a multipart body with a single "file" part of the given
// byte size and returns the body plus its Content-Type header.
func multipartUpload(t *testing.T, filename string, size int) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(make([]byte, size)); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &buf, mw.FormDataContentType()
}

func TestImport_RejectsOversizedUpload(t *testing.T) {
	svc := &stubImportSvc{}
	h := NewWalletHandler(
		&stubWalletStore{wallet: testWallet()},
		nil, nil, nil, nil, nil, nil, svc,
	)

	body, ct := multipartUpload(t, "huge.json", maxUploadBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/wallets/w1/import", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()

	importRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d (%s)", rec.Code, rec.Body.String())
	}
	if svc.called {
		t.Error("import service should not be invoked for an oversized upload")
	}
}

func TestImport_AcceptsSmallUpload(t *testing.T) {
	svc := &stubImportSvc{}
	h := NewWalletHandler(
		&stubWalletStore{wallet: testWallet()},
		nil, nil, nil, nil, nil, nil, svc,
	)

	body, ct := multipartUpload(t, "wallet.json", 64)
	req := httptest.NewRequest(http.MethodPost, "/wallets/w1/import", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()

	importRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !svc.called {
		t.Error("import service should be invoked for a valid upload")
	}
}

func TestImport_WalletNotFound(t *testing.T) {
	svc := &stubImportSvc{}
	h := NewWalletHandler(
		&stubWalletStore{wallet: nil}, // GetByID → sql.ErrNoRows
		nil, nil, nil, nil, nil, nil, svc,
	)

	body, ct := multipartUpload(t, "wallet.json", 64)
	req := httptest.NewRequest(http.MethodPost, "/wallets/missing/import", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()

	importRouter(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if svc.called {
		t.Error("import service should not be invoked when wallet is missing")
	}
}
