package descriptor

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/storagebirddrop/abacus/internal/importer"
)

// Generic descriptor fallback: any JSON containing a "descriptor" or "desc" field.
// Registered last so higher-priority importers (Coldcard, Specter, Electrum) run first.
// Covers: Jade, Passport, SeedSigner, and future devices using standard descriptor exports.
type Importer struct{}

func New() *Importer { return &Importer{} }

func (i *Importer) Name() string               { return "Generic Descriptor" }
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
	_, hasDesc := raw["desc"]
	return hasDescriptor || hasDesc
}

type genericExport struct {
	Descriptor string `json:"descriptor"`
	Desc       string `json:"desc"`
	Label      string `json:"label"`
	Name       string `json:"name"`
}

func (i *Importer) Import(ctx context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	var export genericExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, err
	}

	desc := export.Descriptor
	if desc == "" {
		desc = export.Desc
	}
	name := export.Label
	if name == "" {
		name = export.Name
	}

	return &importer.ImportResult{
		WalletSetup: &importer.WalletSetup{
			Descriptor: desc,
			Name:       name,
		},
	}, nil
}
