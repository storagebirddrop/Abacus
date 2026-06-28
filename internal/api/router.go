package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(version string, wh *WalletHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(jsonContentType)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handleHealth(version))
		r.Get("/version", handleVersion(version))

		// Wallets
		r.Get("/wallets", wh.List)
		r.Post("/wallets", wh.Create)
		r.Get("/wallets/{walletID}", wh.Get)
		r.Delete("/wallets/{walletID}", wh.Delete)

		// Import
		r.Post("/wallets/{walletID}/import", wh.Import)
		r.Get("/wallets/{walletID}/import-jobs", wh.ListImportJobs)
		r.Get("/import-jobs/{jobID}", wh.GetImportJob)

		// Transactions
		r.Get("/wallets/{walletID}/transactions", wh.ListTransactions)
		r.Get("/wallets/{walletID}/transactions/{txid}", wh.GetTransaction)
		r.Patch("/wallets/{walletID}/transactions/{txid}", notImplemented)

		// Labels
		r.Get("/wallets/{walletID}/labels", wh.ListLabels)
		r.Post("/wallets/{walletID}/labels", notImplemented)

		// Ledger (Phase 2)
		r.Get("/wallets/{walletID}/ledger", notImplemented)
		r.Get("/wallets/{walletID}/ledger/{entryID}", notImplemented)

		// Accounting (Phase 3)
		r.Post("/wallets/{walletID}/accounting/run", notImplemented)
		r.Get("/wallets/{walletID}/accounting/summary", notImplemented)
		r.Get("/wallets/{walletID}/accounting/cost-basis", notImplemented)

		// UTXOs (Phase 2)
		r.Get("/wallets/{walletID}/utxos", notImplemented)

		// Prices (Phase 3)
		r.Get("/prices", notImplemented)
		r.Post("/prices", notImplemented)

		// Reports (Phase 4)
		r.Get("/wallets/{walletID}/reports/balance-sheet", notImplemented)
		r.Get("/wallets/{walletID}/reports/pnl", notImplemented)
		r.Get("/wallets/{walletID}/reports/transactions", notImplemented)
	})

	return r
}

func handleHealth(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"version": version,
		})
	}
}

func handleVersion(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"version": version,
		})
	}
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"error": "not implemented",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
