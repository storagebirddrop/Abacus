package api

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(version string, wh *WalletHandler, ah *AccountingHandler, rh *ReportHandler, sh *SyncHandler, lh *LedgerHandler, ph *PortfolioHandler, sth *SettingsHandler, frontendFS fs.FS, sec SecurityConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(jsonContentType)
		// Rate limit first (cheap reject), then optional bearer auth. Both are
		// no-ops unless configured; health/version stay reachable for probes.
		r.Use(rateLimiter(sec.RateLimitRPM))
		r.Use(tokenAuth(sec.APIToken))
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
		r.Patch("/wallets/{walletID}/transactions/{txid}", wh.PatchTransaction)

		// Labels
		r.Get("/wallets/{walletID}/labels", wh.ListLabels)
		r.Post("/wallets/{walletID}/labels", wh.CreateLabel)
		r.Get("/wallets/{walletID}/labels/export", wh.ExportLabels)

		// Ledger (Phase 2)
		r.Get("/wallets/{walletID}/ledger", lh.ListLedger)
		r.Get("/wallets/{walletID}/ledger/{entryID}", lh.GetLedgerEntry)

		// Accounting (Phase 3)
		r.Post("/wallets/{walletID}/accounting/run", ah.RunAccounting)
		r.Get("/wallets/{walletID}/accounting/summary", ah.GetSummary)
		r.Get("/wallets/{walletID}/accounting/cost-basis", ah.ListCostBasis)

		// UTXOs (Phase 2)
		r.Get("/wallets/{walletID}/utxos", lh.ListUTXOs)

		// Prices (Phase 3)
		r.Get("/prices", ah.ListPrices)
		r.Post("/prices", ah.CreatePrice)

		// Reports (Phase 5)
		r.Get("/wallets/{walletID}/reports/balance-sheet", rh.BalanceSheet)
		r.Get("/wallets/{walletID}/reports/pnl", rh.PnL)
		r.Get("/wallets/{walletID}/reports/transactions", rh.Transactions)
			r.Get("/wallets/{walletID}/reports/tax", rh.TaxReport)

		// Blockchain sync (Phase 7)
		r.Post("/wallets/{walletID}/sync", sh.StartSync)
		r.Get("/wallets/{walletID}/sync-jobs", sh.ListSyncJobs)
		r.Get("/sync-jobs/{jobID}", sh.GetSyncJob)

		// Settings
		r.Get("/settings", sth.GetSettings)
		r.Patch("/settings", sth.UpdateSettings)

		// Portfolio (cross-wallet)
		r.Get("/portfolio/summary", ph.GetPortfolioSummary)
	})

	// Serve frontend SPA — all non-API routes fall through to index.html
	if frontendFS != nil {
		r.NotFound(spaHandler(frontendFS))
		r.Get("/*", spaHandler(frontendFS))
	}

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

// spaHandler serves static files from frontendFS; falls back to index.html for
// unknown paths so React Router handles client-side navigation.
func spaHandler(frontendFS fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(frontendFS))
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		if path == "" {
			path = "index.html"
		}
		if _, err := frontendFS.Open(path); err != nil {
			// File not found — serve index.html for SPA routing
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}
