package importer

import (
	"context"
	"io"

	"github.com/storagebirddrop/abacus/internal/domain"
)

// WalletImporter is the plugin interface for all wallet importers.
// Adding support for a new wallet requires only implementing this interface.
type WalletImporter interface {
	// Name returns the human-readable name of this importer.
	Name() string

	// SupportedFormats returns file extensions this importer can handle.
	SupportedFormats() []string

	// Detect returns true if this importer recognizes the given file.
	// The reader is positioned at the start; implementors must not consume it.
	Detect(filename string, r io.ReadSeeker) bool

	// Import reads the file and returns normalized data.
	Import(ctx context.Context, walletID string, r io.Reader) (*ImportResult, error)
}

// WalletSetup carries optional wallet metadata from setup-only importers
// (hardware signing devices that export descriptor + xpub, not transaction history).
type WalletSetup struct {
	Descriptor  string
	Fingerprint string
	Name        string
}

type ImportResult struct {
	WalletSetup  *WalletSetup
	Transactions []domain.Transaction
	Inputs       []domain.TransactionInput
	Outputs      []domain.TransactionOutput
	UTXOs        []domain.UTXO
	Labels       []domain.Label
	Addresses    []domain.Address
	Errors       []ImportError
}

type ImportError struct {
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Registry holds all registered importers.
var Registry []WalletImporter

// Register adds an importer to the registry.
func Register(imp WalletImporter) {
	Registry = append(Registry, imp)
}

// Detect returns the first importer that recognizes the file, or nil.
func Detect(filename string, r io.ReadSeeker) WalletImporter {
	for _, imp := range Registry {
		if imp.Detect(filename, r) {
			_, _ = r.Seek(0, io.SeekStart)
			return imp
		}
		_, _ = r.Seek(0, io.SeekStart)
	}
	return nil
}
