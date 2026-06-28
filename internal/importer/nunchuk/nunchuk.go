package nunchuk

import (
	"context"
	"io"

	"github.com/storagebirddrop/abacus/internal/importer"
)

// Importer handles Nunchuk wallet exports:
//   - Nunchuk JSON transaction export
//   - BSMS files for multisig descriptor import (BIP-129)
//   - BIP329 label files (.jsonl)
type Importer struct{}

func New() *Importer { return &Importer{} }

func (n *Importer) Name() string { return "Nunchuk" }

func (n *Importer) SupportedFormats() []string {
	return []string{"json", "jsonl", "bsms"}
}

func (n *Importer) Detect(filename string, r io.ReadSeeker) bool {
	// Phase 1 implementation: detect Nunchuk JSON structure
	// or BSMS header ("BSMS 1.0")
	return false
}

func (n *Importer) Import(_ context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	// Phase 1 implementation
	return &importer.ImportResult{}, nil
}
