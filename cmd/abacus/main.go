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
	"github.com/storagebirddrop/abacus/internal/importer/nunchuk"
	"github.com/storagebirddrop/abacus/internal/importer/sparrow"
	"github.com/storagebirddrop/abacus/internal/repository"
)

func main() {
	cfg := config.Load()

	// Register importers — Nunchuk first so it claims BSMS files before Sparrow.
	importer.Register(nunchuk.New())
	importer.Register(sparrow.New())

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

	// Services
	importSvc := importer.NewService(db, txRepo, labelRepo, ledgerRepo, utxoRepo, jobRepo)
	accountingSvc := accounting.NewService(db, utxoRepo, cbRepo, priceRepo, txRepo)

	// HTTP handlers
	walletHandler := api.NewWalletHandler(walletRepo, txRepo, jobRepo, labelRepo, importSvc)
	accountingHandler := api.NewAccountingHandler(accountingSvc, priceRepo, cbRepo)
	reportHandler := api.NewReportHandler(walletRepo, txRepo, utxoRepo, cbRepo)
	router := api.NewRouter(cfg.Version, walletHandler, accountingHandler, reportHandler)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Abacus %s starting on %s", cfg.Version, addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
