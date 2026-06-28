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
//   - Nunchuk JSON transaction export
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
	if common.IsBSMS(r) {
		return true
	}
	return false
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
		labels, errs := common.ParseBIP329(walletID, bytes.NewReader(data))
		result := &importer.ImportResult{Labels: labels}
		for _, e := range errs {
			result.Errors = append(result.Errors, importer.ImportError{
				Line:    e.Line,
				Message: e.Message,
			})
		}
		return result, nil
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

// --- Nunchuk JSON ---
// Nunchuk exports a JSON array of transaction objects.

type nunchukExport struct {
	Transactions []nunchukTx `json:"transactions"`
}

type nunchukTx struct {
	Txid        string       `json:"txid"`
	BlockHeight int64        `json:"blockHeight"`
	BlockTime   int64        `json:"blockTime"` // unix seconds
	Fee         int64        `json:"fee"`
	Status      string       `json:"status"` // "CONFIRMED" | "PENDING"
	Inputs      []nunchukIO  `json:"inputs"`
	Outputs     []nunchukIO  `json:"outputs"`
}

type nunchukIO struct {
	Txid    string `json:"txid"`
	Vout    int    `json:"vout"`
	Value   int64  `json:"value"` // satoshis
	Address string `json:"address"`
}

func parseNunchukJSON(walletID string, r io.Reader) (*importer.ImportResult, error) {
	var export nunchukExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, err
	}

	result := &importer.ImportResult{}
	for _, ntx := range export.Transactions {
		tx := domain.Transaction{
			WalletID:    walletID,
			Txid:        ntx.Txid,
			BlockHeight: ntx.BlockHeight,
			BlockTime:   time.Unix(ntx.BlockTime, 0).UTC(),
			FeeSats:     ntx.Fee,
			Confirmed:   strings.EqualFold(ntx.Status, "confirmed"),
		}
		result.Transactions = append(result.Transactions, tx)

		for _, in := range ntx.Inputs {
			result.Inputs = append(result.Inputs, domain.TransactionInput{
				PrevTxid: in.Txid,
				PrevVout: in.Vout,
				Sats:     in.Value,
				Address:  in.Address,
			})
		}
		for i, out := range ntx.Outputs {
			result.Outputs = append(result.Outputs, domain.TransactionOutput{
				Vout:    i,
				Sats:    out.Value,
				Address: out.Address,
			})
		}
	}
	return result, nil
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

func isNunchukJSON(r io.ReadSeeker) bool {
	buf := make([]byte, 128)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	if n == 0 {
		return false
	}
	content := string(buf[:n])
	return strings.Contains(content, `"transactions"`) ||
		strings.Contains(content, `"txid"`)
}
