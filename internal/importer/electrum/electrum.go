package electrum

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/importer"
)

// Electrum exports an unencrypted JSON wallet file containing "wallet_type",
// "keystore", and "transactions" fields.
type Importer struct{}

func New() *Importer { return &Importer{} }

func (i *Importer) Name() string               { return "Electrum" }
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
	_, hasWalletType := raw["wallet_type"]
	_, hasKeystore := raw["keystore"]
	return hasWalletType && hasKeystore
}

type electrumWallet struct {
	WalletType    string `json:"wallet_type"`
	UseEncryption bool   `json:"use_encryption"`
	Keystore      struct {
		Type       string `json:"type"`
		XPub       string `json:"xpub"`
		Derivation string `json:"derivation"`
	} `json:"keystore"`
	Transactions map[string]json.RawMessage `json:"transactions"`
	Addresses    struct {
		Receiving []string `json:"receiving"`
		Change    []string `json:"change"`
	} `json:"addresses"`
}

func (i *Importer) Import(ctx context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var wallet electrumWallet
	if err := json.Unmarshal(data, &wallet); err != nil {
		return nil, err
	}

	if wallet.UseEncryption {
		return nil, fmt.Errorf("encrypted Electrum wallets are not supported; disable encryption in Electrum before exporting")
	}

	result := &importer.ImportResult{}

	if wallet.Keystore.XPub != "" {
		result.WalletSetup = &importer.WalletSetup{
			Descriptor: "wpkh(" + wallet.Keystore.XPub + "/0/*)",
		}
	}

	// Electrum stores transactions as raw hex strings keyed by txid.
	epoch := time.Unix(0, 0).UTC()
	for txid, raw := range wallet.Transactions {
		var hexStr string
		if err := json.Unmarshal(raw, &hexStr); err != nil {
			continue // skip non-string entries (some versions store objects)
		}
		_ = hexStr
		result.Transactions = append(result.Transactions, domain.Transaction{
			ID:        uuid.New().String(),
			WalletID:  walletID,
			Txid:      txid,
			BlockTime: epoch,
		})
	}

	for _, addr := range wallet.Addresses.Receiving {
		result.Addresses = append(result.Addresses, domain.Address{
			WalletID: walletID,
			Address:  addr,
			Type:     "receive",
		})
	}
	for _, addr := range wallet.Addresses.Change {
		result.Addresses = append(result.Addresses, domain.Address{
			WalletID: walletID,
			Address:  addr,
			Type:     "change",
		})
	}

	return result, nil
}
