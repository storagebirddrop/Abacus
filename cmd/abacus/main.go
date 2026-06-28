package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/storagebirddrop/abacus/internal/api"
	"github.com/storagebirddrop/abacus/internal/config"
	"github.com/storagebirddrop/abacus/internal/importer"
	"github.com/storagebirddrop/abacus/internal/importer/nunchuk"
	"github.com/storagebirddrop/abacus/internal/importer/sparrow"
)

func main() {
	cfg := config.Load()

	// Register importers
	importer.Register(sparrow.New())
	importer.Register(nunchuk.New())

	router := api.NewRouter(cfg.Version)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Abacus %s starting on %s", cfg.Version, addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
