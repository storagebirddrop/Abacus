package nunchuk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/importer"
	"github.com/storagebirddrop/abacus/internal/importer/common"
)

// Importer handles Nunchuk wallet exports:
//   - BSMS files for multisig descriptor import (BIP-129)
//   - Nunchuk wallet JSON export (wallet config + co-signers)
//   - Nunchuk transaction history JSON export
//   - BIP329 label files (.jsonl)
type Importer struct{}

func New() *Importer { return &Importer{} }

func (n *Importer) Name() string { return "Nunchuk" }

func (n *Importer) SupportedFormats() []string {
	return []string{"json", "jsonl", "bsms"}
}

func (n *Importer) Detect(filename string, r io.ReadSeeker) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".bsms":
		return common.IsBSMS(r)
	case ".jsonl":
		return isBIP329(r)
	case ".json":
		return isNunchukJSON(r)
	}
	// Content-based detection without extension
	if common.IsBSMS(r) {
		return true
	}
	_, _ = r.Seek(0, io.SeekStart)
	return isNunchukJSON(r)
}

func (n *Importer) Import(ctx context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	rs := bytes.NewReader(data)

	if common.IsBSMS(rs) {
		_, _ = rs.Seek(0, io.SeekStart)
		return parseBSMS(walletID, bytes.NewReader(data))
	}
	_, _ = rs.Seek(0, io.SeekStart)
	if isBIP329(rs) {
		_, _ = rs.Seek(0, io.SeekStart)
		return parseBIP329(walletID, bytes.NewReader(data))
	}
	return parseNunchukJSON(walletID, bytes.NewReader(data))
}

// --- BSMS ---

func parseBSMS(walletID string, r io.Reader) (*importer.ImportResult, error) {
	rec, err := common.ParseBSMS(r)
	if err != nil {
		return nil, err
	}
	result := &importer.ImportResult{}
	if rec.FirstAddress != "" {
		result.Addresses = append(result.Addresses, domain.Address{
			WalletID: walletID,
			Address:  rec.FirstAddress,
			Type:     "receive",
		})
	}
	return result, nil
}

// --- BIP329 ---

func parseBIP329(walletID string, r io.Reader) (*importer.ImportResult, error) {
	labels, errs := common.ParseBIP329(walletID, r)
	result := &importer.ImportResult{Labels: labels}
	for _, e := range errs {
		result.Errors = append(result.Errors, importer.ImportError{
			Line:    e.Line,
			Message: e.Message,
		})
	}
	return result, nil
}

// --- Nunchuk JSON ---
//
// Nunchuk exports two JSON formats:
//
// 1. Wallet config (contains "wallet_type" or "signers"):
//    {"name":"My Vault","wallet_type":"MULTI_SIG","descriptor":"wsh(...)","signers":[...]}
//
// 2. Transaction history (contains "transactions"):
//    {"transactions":[{"txid":"...","height":800000,...}]}
//
// Both may appear combined in one file.

type nunchukFile struct {
	// Wallet config fields
	Name        string         `json:"name"`
	WalletType  string         `json:"wallet_type"`  // "MULTI_SIG" | "SINGLE_SIG"
	AddressType string         `json:"address_type"` // "NATIVE_SEGWIT" | "TAPROOT" | ...
	M           int            `json:"m"`
	N           int            `json:"n"`
	Descriptor  string         `json:"descriptor"`
	Signers     []nunchukSigner `json:"signers"`

	// Transaction history
	Transactions []nunchukTx `json:"transactions"`
}

type nunchukSigner struct {
	Name           string `json:"name"`
	Xfp            string `json:"xfp"`             // master key fingerprint
	Xpub           string `json:"xpub"`
	DerivationPath string `json:"derivation_path"`
	Type           string `json:"type"` // "HARDWARE" | "SOFTWARE" | "AIRGAP" | "NFC"
}

type nunchukTx struct {
	Txid        string      `json:"txid"`
	Height      int64       `json:"height"`     // block height; 0 = unconfirmed
	BlockTime   int64       `json:"block_time"` // unix seconds; alt: "time"
	Time        int64       `json:"time"`       // alias used in some versions
	Fee         int64       `json:"fee"`        // satoshis
	Memo        string      `json:"memo"`       // user note / label
	Status      string      `json:"status"`     // "CONFIRMED" | "PENDING_CONFIRMATION" | "NETWORK_REJECTED"
	Inputs      []nunchukIO `json:"inputs"`
	Outputs     []nunchukIO `json:"outputs"`
}

type nunchukIO struct {
	Txid    string `json:"txid"`    // only on inputs
	Vout    int    `json:"vout"`    // only on inputs
	Value   int64  `json:"value"`   // satoshis
	Address string `json:"address"`
	Mine    bool   `json:"is_mine"` // belongs to this wallet
}

func parseNunchukJSON(walletID string, r io.Reader) (*importer.ImportResult, error) {
	var f nunchukFile
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		return nil, err
	}

	result := &importer.ImportResult{}

	// Import transaction history
	for i := range f.Transactions {
		ntx := &f.Transactions[i]
		blockTime := ntx.BlockTime
		if blockTime == 0 {
			blockTime = ntx.Time
		}

		tx := domain.Transaction{
			WalletID:    walletID,
			Txid:        ntx.Txid,
			BlockHeight: ntx.Height,
			BlockTime:   time.Unix(blockTime, 0).UTC(),
			FeeSats:     ntx.Fee,
			Confirmed:   isConfirmed(ntx.Status, ntx.Height),
		}
		result.Transactions = append(result.Transactions, tx)

		// Inputs
		for _, in := range ntx.Inputs {
			result.Inputs = append(result.Inputs, domain.TransactionInput{
				PrevTxid: in.Txid,
				PrevVout: in.Vout,
				Sats:     in.Value,
				Address:  in.Address,
			})
		}
		// Outputs
		for idx, out := range ntx.Outputs {
			result.Outputs = append(result.Outputs, domain.TransactionOutput{
				Vout:    idx,
				Sats:    out.Value,
				Address: out.Address,
			})
		}

		// Nunchuk memos become BIP329 tx labels
		if ntx.Memo != "" {
			result.Labels = append(result.Labels, domain.Label{
				WalletID: walletID,
				Type:     "tx",
				Ref:      ntx.Txid,
				Label:    ntx.Memo,
				Origin:   "nunchuk",
			})
		}
	}

	return result, nil
}

// isConfirmed determines confirmation from Nunchuk status and block height.
func isConfirmed(status string, height int64) bool {
	if status != "" {
		return strings.EqualFold(status, "confirmed")
	}
	return height > 0
}

// --- Detection helpers ---

func isBIP329(r io.ReadSeeker) bool {
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	if n == 0 {
		return false
	}
	firstLine := strings.Split(strings.TrimSpace(string(buf[:n])), "\n")[0]
	return strings.HasPrefix(firstLine, "{") &&
		(strings.Contains(firstLine, `"type"`) || strings.Contains(firstLine, `"ref"`))
}

// isNunchukJSON checks for Nunchuk-specific JSON keys.
// Nunchuk files contain "wallet_type", "signers", or "transactions" with "height"/"memo".
func isNunchukJSON(r io.ReadSeeker) bool {
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	if n == 0 {
		return false
	}
	s := string(buf[:n])
	return strings.Contains(s, `"wallet_type"`) ||
		strings.Contains(s, `"signers"`) ||
		strings.Contains(s, `"is_mine"`) ||
		(strings.Contains(s, `"transactions"`) && strings.Contains(s, `"height"`))
}
