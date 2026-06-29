package coldcard

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/importer"
)

// Coldcard exports a JSON file called "coldcard-export.json" that contains
// an "xfp" (extended fingerprint) field plus BIP84/BIP48 descriptor sections.
// It contains NO transaction history — only wallet setup.
type Importer struct{}

func New() *Importer { return &Importer{} }

func (i *Importer) Name() string               { return "Coldcard" }
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
	_, hasXFP := raw["xfp"]
	return hasXFP
}

type coldcardExport struct {
	Chain   string `json:"chain"`
	XFP     string `json:"xfp"`
	Account int    `json:"account"`
	BIP84   *struct {
		Deriv string `json:"deriv"`
		XPub  string `json:"xpub"`
		Desc  string `json:"desc"`
		First string `json:"first"`
	} `json:"bip84"`
	BIP48_2 *struct {
		Deriv string `json:"deriv"`
		Desc  string `json:"desc"`
		First string `json:"first"`
	} `json:"bip48_2"`
}

func (i *Importer) Import(ctx context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	var export coldcardExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, err
	}

	// Prefer native segwit (BIP84), fall back to multisig (BIP48/2).
	var desc, firstAddr string
	if export.BIP84 != nil && export.BIP84.Desc != "" {
		desc = export.BIP84.Desc
		firstAddr = export.BIP84.First
	} else if export.BIP48_2 != nil && export.BIP48_2.Desc != "" {
		desc = export.BIP48_2.Desc
		firstAddr = export.BIP48_2.First
	}

	result := &importer.ImportResult{
		WalletSetup: &importer.WalletSetup{
			Descriptor:  desc,
			Fingerprint: export.XFP,
		},
	}

	if firstAddr != "" {
		result.Addresses = []domain.Address{{
			WalletID: walletID,
			Address:  firstAddr,
			Type:     "receive",
		}}
	}

	return result, nil
}
