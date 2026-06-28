package sparrow

import (
	"context"
	"io"

	"github.com/storagebirddrop/abacus/internal/importer"
)

// Importer handles Sparrow wallet exports:
//   - Sparrow JSON wallet export
//   - BIP329 label files (.jsonl)
//   - Transaction CSV export
type Importer struct{}

func New() *Importer { return &Importer{} }

func (s *Importer) Name() string { return "Sparrow" }

func (s *Importer) SupportedFormats() []string {
	return []string{"json", "jsonl", "csv"}
}

func (s *Importer) Detect(filename string, r io.ReadSeeker) bool {
	// Phase 1 implementation: detect Sparrow JSON structure
	// or BIP329 JSONL format
	return false
}

func (s *Importer) Import(_ context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	// Phase 1 implementation
	return &importer.ImportResult{}, nil
}
