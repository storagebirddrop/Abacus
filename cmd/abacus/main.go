package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	abacus "github.com/storagebirddrop/abacus"
	"github.com/storagebirddrop/abacus/internal/accounting"
	"github.com/storagebirddrop/abacus/internal/api"
	"github.com/storagebirddrop/abacus/internal/config"
	"github.com/storagebirddrop/abacus/internal/importer"
	"github.com/storagebirddrop/abacus/internal/importer/coldcard"
	"github.com/storagebirddrop/abacus/internal/importer/descriptor"
	"github.com/storagebirddrop/abacus/internal/importer/electrum"
	"github.com/storagebirddrop/abacus/internal/importer/nunchuk"
	"github.com/storagebirddrop/abacus/internal/importer/sparrow"
	"github.com/storagebirddrop/abacus/internal/importer/specter"
	"github.com/storagebirddrop/abacus/internal/repository"
	abacussync "github.com/storagebirddrop/abacus/internal/sync"
	electrumbackend "github.com/storagebirddrop/abacus/internal/sync/electrum"
	"github.com/storagebirddrop/abacus/internal/sync/esplora"
)

func main() {
	cfg := config.Load()

	// Register importers in priority order.
	// Nunchuk first: claims BSMS files before Sparrow.
	// Coldcard: JSON with "xfp" field (hardware signing device).
	// Electrum: JSON with "wallet_type" + "keystore".
	// Specter: JSON with "descriptor" but no "xfp" or "wallet_type".
	// Sparrow: broad JSON/CSV catch-all.
	// Generic descriptor: last-resort fallback for any JSON with a descriptor field.
	importer.Register(nunchuk.New())
	importer.Register(coldcard.New())
	importer.Register(electrum.New())
	importer.Register(specter.New())
	importer.Register(sparrow.New())
	importer.Register(descriptor.New())

	// Database
	db, err := repository.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Migrations: use embedded FS (works in both binary and AppImage).
	migrationsFS, err := fs.Sub(abacus.Migrations, "migrations")
	if err != nil {
		log.Fatalf("migrations fs: %v", err)
	}
	if err := repository.Migrate(db, migrationsFS); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("Database migrations applied")

	// Repositories
	walletRepo := repository.NewWalletRepo(db)
	txRepo := repository.NewTransactionRepo(db)
	labelRepo := repository.NewLabelRepo(db)
	ledgerRepo := repository.NewLedgerRepo(db)
	utxoRepo := repository.NewUTXORepo(db)
	jobRepo := repository.NewImportJobRepo(db)
	priceRepo := repository.NewPriceSnapshotRepo(db)
	cbRepo := repository.NewCostBasisRepo(db)
	syncJobRepo := repository.NewSyncJobRepo(db)
	syncStateRepo := repository.NewSyncStateRepo(db)
	settingsRepo := repository.NewSettingsRepo(db)

	// BackendFactory: builds the active BlockchainBackend from DB settings at call time.
	// Sync is disabled by default; set sync_enabled=true in Settings to activate.
	backendFactory := func(ctx context.Context) (abacussync.BlockchainBackend, error) {
		m, err := settingsRepo.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("load settings: %w", err)
		}
		if m["sync_enabled"] != "true" {
			return nil, fmt.Errorf("blockchain sync is disabled — enable it in Settings")
		}
		get := func(key, def string) string {
			if v, ok := m[key]; ok && v != "" {
				return v
			}
			return def
		}
		switch get("blockchain_backend", "esplora") {
		case "electrum":
			port := 50002
			if v := m["electrum_port"]; v != "" {
				if n, err := fmt.Sscanf(v, "%d", &port); n == 0 || err != nil {
					port = 50002
				}
			}
			return electrumbackend.New(get("electrum_host", "electrum.blockstream.info"), port, get("electrum_tls", "true") == "true"), nil
		default:
			rateMS := 100
			if v := m["esplora_rate_ms"]; v != "" {
				fmt.Sscanf(v, "%d", &rateMS)
			}
			return esplora.New(get("esplora_url", "https://mempool.space/api"), rateMS), nil
		}
	}

	// Services
	importSvc := importer.NewService(db, walletRepo, txRepo, labelRepo, ledgerRepo, utxoRepo, jobRepo)
	accountingSvc := accounting.NewService(db, utxoRepo, cbRepo, priceRepo, txRepo)
	syncSvc := abacussync.NewService(db, walletRepo, txRepo, ledgerRepo, utxoRepo, syncJobRepo, syncStateRepo, backendFactory)

	// Frontend FS: use FRONTEND_DIR env var for dev (disk); otherwise embedded.
	var frontendFS fs.FS
	if dir := cfg.FrontendDir; dir != "" {
		if _, err := os.Stat(dir); err == nil {
			frontendFS = os.DirFS(dir)
			log.Printf("Serving frontend from disk: %s", dir)
		}
	}
	if frontendFS == nil {
		frontendFS, err = fs.Sub(abacus.Frontend, "web/dist")
		if err != nil {
			log.Fatalf("frontend fs: %v", err)
		}
		log.Println("Serving frontend from embedded FS")
	}

	// HTTP handlers
	journalRepo := repository.NewJournalRepo(db)
	walletHandler := api.NewWalletHandler(walletRepo, txRepo, ledgerRepo, journalRepo, db, jobRepo, labelRepo, importSvc)
	accountingHandler := api.NewAccountingHandler(accountingSvc, priceRepo, cbRepo, walletRepo)
	reportHandler := api.NewReportHandler(walletRepo, txRepo, utxoRepo, cbRepo, priceRepo)
	syncHandler := api.NewSyncHandler(syncSvc, syncJobRepo)
	ledgerHandler := api.NewLedgerHandler(walletRepo, ledgerRepo, journalRepo, utxoRepo)
	portfolioHandler := api.NewPortfolioHandler(walletRepo, cbRepo, utxoRepo)
	settingsHandler := api.NewSettingsHandler(settingsRepo)
	router := api.NewRouter(cfg.Version, walletHandler, accountingHandler, reportHandler, syncHandler, ledgerHandler, portfolioHandler, settingsHandler, frontendFS)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Abacus %s starting on %s", cfg.Version, addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
