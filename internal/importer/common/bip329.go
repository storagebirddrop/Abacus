package common

import (
	"bufio"
	"encoding/json"
	"io"

	"github.com/storagebirddrop/abacus/internal/domain"
)

// bip329Line mirrors a single JSON line from a BIP329 .jsonl file.
type bip329Line struct {
	Type      string  `json:"type"`
	Ref       string  `json:"ref"`
	Label     string  `json:"label"`
	Origin    string  `json:"origin,omitempty"`
	Spendable *bool   `json:"spendable,omitempty"`
}

// ParseBIP329 reads a BIP329 JSONL file and returns domain Labels.
func ParseBIP329(walletID string, r io.Reader) ([]domain.Label, []ImportError) {
	var labels []domain.Label
	var errs []ImportError

	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		var line bip329Line
		if err := json.Unmarshal(raw, &line); err != nil {
			errs = append(errs, ImportError{Line: lineNum, Message: err.Error()})
			continue
		}
		labels = append(labels, domain.Label{
			WalletID:  walletID,
			Type:      line.Type,
			Ref:       line.Ref,
			Label:     line.Label,
			Origin:    line.Origin,
			Spendable: line.Spendable,
		})
	}
	return labels, errs
}

type ImportError struct {
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Message string `json:"message"`
}
