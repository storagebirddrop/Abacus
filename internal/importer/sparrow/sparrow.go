package sparrow

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
	"github.com/storagebirddrop/abacus/internal/importer"
	"github.com/storagebirddrop/abacus/internal/importer/common"
)

// Importer handles Sparrow wallet exports:
//   - BSMS files (wallet descriptor)
//   - BIP329 label files (.jsonl)
//   - Transaction CSV export
type Importer struct{}

func New() *Importer { return &Importer{} }

func (s *Importer) Name() string { return "Sparrow" }

func (s *Importer) SupportedFormats() []string {
	return []string{"json", "jsonl", "csv", "bsms"}
}

func (s *Importer) Detect(filename string, r io.ReadSeeker) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".bsms":
		return common.IsBSMS(r)
	case ".jsonl":
		return isBIP329(r)
	case ".csv":
		return isSparrowCSV(r)
	case ".json":
		return isSparrowJSON(r)
	}
	// Try content detection without extension
	if common.IsBSMS(r) {
		return true
	}
	if isBIP329(r) {
		return true
	}
	return false
}

func (s *Importer) Import(ctx context.Context, walletID string, r io.Reader) (*importer.ImportResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	rs := io.NewSectionReader(bytes.NewReader(data), 0, int64(len(data)))

	// Try each format in order
	if common.IsBSMS(rs) {
		_, _ = rs.Seek(0, io.SeekStart)
		return parseBSMS(walletID, bytes.NewReader(data))
	}
	_, _ = rs.Seek(0, io.SeekStart)
	if isBIP329(rs) {
		_, _ = rs.Seek(0, io.SeekStart)
		return parseBIP329(walletID, bytes.NewReader(data))
	}
	_, _ = rs.Seek(0, io.SeekStart)
	if isSparrowCSV(rs) {
		_, _ = rs.Seek(0, io.SeekStart)
		return parseSparrowCSV(walletID, bytes.NewReader(data))
	}
	// Try JSON last
	return parseSparrowJSON(walletID, bytes.NewReader(data))
}

// --- BSMS ---

func parseBSMS(walletID string, r io.Reader) (*importer.ImportResult, error) {
	rec, err := common.ParseBSMS(r)
	if err != nil {
		return nil, err
	}
	fps := common.ExtractFingerprints(rec.Descriptor)
	fingerprint := ""
	if len(fps) > 0 {
		fingerprint = fps[0]
	}
	_ = fingerprint // stored on wallet, not in ImportResult
	_ = rec
	// BSMS carries descriptor + first address; no transactions
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

// --- Sparrow CSV ---
// Sparrow exports columns:
// Date, Label, Value, Fee, Balance, TXID, Type

func parseSparrowCSV(walletID string, r io.Reader) (*importer.ImportResult, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		return nil, err
	}
	// Build column index map
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	result := &importer.ImportResult{}
	lineNum := 1
	for {
		lineNum++
		row, err := cr.Read()
		if err != nil {
			break
		}
		tx, errs := sparrowCSVRowToTransaction(walletID, row, idx, lineNum)
		if len(errs) > 0 {
			result.Errors = append(result.Errors, errs...)
			continue
		}
		if tx != nil {
			result.Transactions = append(result.Transactions, *tx)
		}
	}
	return result, nil
}

func sparrowCSVRowToTransaction(walletID string, row []string, idx map[string]int, line int) (*domain.Transaction, []importer.ImportError) {
	get := func(key string) string {
		i, ok := idx[key]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	txid := get("txid")
	if txid == "" {
		txid = get("transaction id")
	}
	if txid == "" {
		return nil, nil // skip empty rows
	}

	var blockTime time.Time
	if dateStr := get("date"); dateStr != "" {
		// Sparrow uses: "2024-01-15 14:32:05" or ISO formats
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05-07:00",
			"01/02/2006 15:04:05",
		} {
			if t, err := time.Parse(layout, dateStr); err == nil {
				blockTime = t.UTC()
				break
			}
		}
	}

	tx := &domain.Transaction{
		WalletID:  walletID,
		Txid:      txid,
		BlockTime: blockTime,
		Confirmed: blockTime.Year() > 2008,
	}

	// Value field is signed satoshis in Sparrow CSV
	if valStr := get("value"); valStr != "" {
		valStr = strings.ReplaceAll(valStr, ",", "")
		valStr = strings.ReplaceAll(valStr, " ", "")
		var sats int64
		if _, err := parseIntStr(valStr, &sats); err == nil {
			_ = sats // stored in ledger layer, not transaction
		}
	}

	return tx, nil
}

// --- Sparrow JSON ---

type sparrowWalletJSON struct {
	Name        string `json:"name"`
	Descriptor  string `json:"descriptor"`
	Network     string `json:"network"`
	Fingerprint string `json:"masterFingerprint"`
	Transactions []sparrowTxJSON `json:"transactions"`
}

type sparrowTxJSON struct {
	Txid        string  `json:"txid"`
	BlockHeight int64   `json:"height"`
	BlockTime   int64   `json:"date"` // unix timestamp
	Fee         int64   `json:"fee"`
	Inputs      []sparrowIOJSON `json:"inputs"`
	Outputs     []sparrowIOJSON `json:"outputs"`
}

type sparrowIOJSON struct {
	Txid    string `json:"txid"`
	Vout    int    `json:"vout"`
	Value   int64  `json:"value"`
	Address string `json:"address"`
}

func parseSparrowJSON(walletID string, r io.Reader) (*importer.ImportResult, error) {
	var w sparrowWalletJSON
	if err := json.NewDecoder(r).Decode(&w); err != nil {
		return nil, err
	}

	result := &importer.ImportResult{}
	for _, stx := range w.Transactions {
		tx := domain.Transaction{
			WalletID:    walletID,
			Txid:        stx.Txid,
			BlockHeight: stx.BlockHeight,
			BlockTime:   time.Unix(stx.BlockTime/1000, 0).UTC(), // Sparrow uses millis
			FeeSats:     stx.Fee,
			Confirmed:   stx.BlockHeight > 0,
		}
		result.Transactions = append(result.Transactions, tx)

		for _, in := range stx.Inputs {
			result.Inputs = append(result.Inputs, domain.TransactionInput{
				PrevTxid: in.Txid,
				PrevVout: in.Vout,
				Sats:     in.Value,
				Address:  in.Address,
			})
		}
		for _, out := range stx.Outputs {
			result.Outputs = append(result.Outputs, domain.TransactionOutput{
				Vout:    out.Vout,
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
	line := strings.TrimSpace(string(buf[:n]))
	// BIP329 lines start with {"type": or similar
	firstLine := strings.Split(line, "\n")[0]
	return strings.HasPrefix(firstLine, "{") &&
		(strings.Contains(firstLine, `"type"`) || strings.Contains(firstLine, `"ref"`))
}

func isSparrowCSV(r io.ReadSeeker) bool {
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	if n == 0 {
		return false
	}
	header := strings.ToLower(string(buf[:n]))
	return strings.Contains(header, "txid") || strings.Contains(header, "transaction id")
}

func isSparrowJSON(r io.ReadSeeker) bool {
	buf := make([]byte, 64)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	if n == 0 {
		return false
	}
	trimmed := strings.TrimSpace(string(buf[:n]))
	return strings.HasPrefix(trimmed, "{")
}

func parseIntStr(s string, out *int64) (int64, error) {
	s = strings.TrimSpace(s)
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}
	var v int64
	for _, c := range s {
		if c < '0' || c > '9' {
			continue // skip separators
		}
		v = v*10 + int64(c-'0')
	}
	if negative {
		v = -v
	}
	if out != nil {
		*out = v
	}
	return v, nil
}
