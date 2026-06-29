package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

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
	"github.com/storagebirddrop/abacus/internal/sync/esplora"
	electrumbackend "github.com/storagebirddrop/abacus/internal/sync/electrum"
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

	migrationsPath, err := filepath.Abs(cfg.MigrationsPath)
	if err != nil {
		log.Fatalf("migrations path: %v", err)
	}
	if err := repository.Migrate(db, migrationsPath); err != nil {
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

	// Blockchain backend selection.
	var blockchainBackend abacussync.BlockchainBackend
	switch cfg.BlockchainBackend {
	case "electrum":
		blockchainBackend = electrumbackend.New(cfg.ElectrumHost, cfg.ElectrumPort, cfg.ElectrumTLS)
		log.Printf("Blockchain backend: electrum (%s:%d, tls=%v)", cfg.ElectrumHost, cfg.ElectrumPort, cfg.ElectrumTLS)
	default:
		blockchainBackend = esplora.New(cfg.EsploraURL, cfg.EsploraRateMS)
		log.Printf("Blockchain backend: esplora (%s)", cfg.EsploraURL)
	}

	// Services
	importSvc := importer.NewService(db, walletRepo, txRepo, labelRepo, ledgerRepo, utxoRepo, jobRepo)
	accountingSvc := accounting.NewService(db, utxoRepo, cbRepo, priceRepo, txRepo)
	syncSvc := abacussync.NewService(db, walletRepo, txRepo, ledgerRepo, utxoRepo, syncJobRepo, syncStateRepo, blockchainBackend)

	// HTTP handlers
	journalRepo := repository.NewJournalRepo(db)
	walletHandler := api.NewWalletHandler(walletRepo, txRepo, ledgerRepo, journalRepo, db, jobRepo, labelRepo, importSvc)
	accountingHandler := api.NewAccountingHandler(accountingSvc, priceRepo, cbRepo, walletRepo)
	reportHandler := api.NewReportHandler(walletRepo, txRepo, utxoRepo, cbRepo, priceRepo)
	syncHandler := api.NewSyncHandler(syncSvc, syncJobRepo)
	ledgerHandler := api.NewLedgerHandler(walletRepo, ledgerRepo, journalRepo, utxoRepo)
	portfolioHandler := api.NewPortfolioHandler(walletRepo, cbRepo, utxoRepo)
	router := api.NewRouter(cfg.Version, walletHandler, accountingHandler, reportHandler, syncHandler, ledgerHandler, portfolioHandler)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Abacus %s starting on %s", cfg.Version, addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
