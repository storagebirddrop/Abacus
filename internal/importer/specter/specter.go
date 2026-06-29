package specter

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/storagebirddrop/abacus/internal/importer"
)

// Specter Desktop exports a JSON file containing a "descriptor" field.
// It explicitly must NOT have "xfp" (Coldcard) or "wallet_type" (Electrum).
type Importer struct{}

func New() *Importer { return &Importer{} }

func (i *Importer) Name() string               { return "Specter Desktop" }
func (i *Importer) SupportedFormats() []string { return []string{"json"} }

func (i *Importer) Detect(filename string, r io.ReadSeeker) bool {
	if !strings.HasSuffix(strings.ToLower(filename), ".json") {
		return false
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return false
	}
	_, _ = r.Seek(0, io.SeekStart)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	_, hasDescriptor := raw["descriptor"]
	_, hasXFP := raw["xfp"]
	_, hasWalletType := raw["wallet_type"]
	return hasDescriptor && !hasXFP && !hasWalletType
}

type specterExport struct {
	Label      string `json:"label"`
	Descriptor string `json:"descriptor"`
}

func (i *Importer) Import(ctx context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	var export specterExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, err
	}

	return &importer.ImportResult{
		WalletSetup: &importer.WalletSetup{
			Descriptor:  export.Descriptor,
			Fingerprint: extractFirstFingerprint(export.Descriptor),
			Name:        export.Label,
		},
	}, nil
}

// extractFirstFingerprint pulls the first [xxxxxxxx/...] fingerprint from a descriptor.
func extractFirstFingerprint(desc string) string {
	start := strings.Index(desc, "[")
	if start < 0 {
		return ""
	}
	end := strings.IndexAny(desc[start+1:], "/]")
	if end < 0 {
		return ""
	}
	fp := desc[start+1 : start+1+end]
	if len(fp) == 8 {
		return fp
	}
	return ""
}
