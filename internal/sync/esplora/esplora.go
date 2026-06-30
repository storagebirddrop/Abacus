// Package esplora implements the BlockchainBackend interface using the Esplora REST API.
// Compatible with Blockstream.info (https://blockstream.info/api) and
// Mempool.space (https://mempool.space/api).
package esplora

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	stdsync "sync"
	"time"

	"github.com/storagebirddrop/abacus/internal/sync"
)

// Backend queries an Esplora-compatible REST API.
// It is safe for concurrent use: the rate-limiter state is mutex-guarded.
type Backend struct {
	baseURL    string
	httpClient *http.Client
	rateDelay  time.Duration

	mu      stdsync.Mutex
	lastReq time.Time
}

func New(baseURL string, rateDelayMS int) *Backend {
	if rateDelayMS <= 0 {
		rateDelayMS = 100
	}
	return &Backend{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		rateDelay:  time.Duration(rateDelayMS) * time.Millisecond,
	}
}

func (b *Backend) Name() string { return "esplora" }

func (b *Backend) BlockHeight(ctx context.Context) (int64, error) {
	var height int64
	if err := b.get(ctx, "/blocks/tip/height", &height); err != nil {
		return 0, err
	}
	return height, nil
}

func (b *Backend) GetTransactions(ctx context.Context, address string) ([]sync.TxRecord, error) {
	var raw []esploraTransaction
	if err := b.get(ctx, "/address/"+address+"/txs", &raw); err != nil {
		return nil, err
	}
	records := make([]sync.TxRecord, 0, len(raw))
	for _, tx := range raw {
		records = append(records, tx.toRecord())
	}
	return records, nil
}

// throttle enforces the configured minimum spacing between requests. The lock
// is held across the wait so concurrent callers are paced correctly rather than
// racing on lastReq.
func (b *Backend) throttle(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if delay := b.rateDelay - time.Since(b.lastReq); delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	b.lastReq = time.Now()
	return nil
}

func (b *Backend) get(ctx context.Context, path string, out any) error {
	if err := b.throttle(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("esplora %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ---------- JSON shapes ----------

type esploraTransaction struct {
	Txid   string        `json:"txid"`
	Fee    int64         `json:"fee"`
	Status esploraStatus `json:"status"`
	Vin    []esploraVin  `json:"vin"`
	Vout   []esploraVout `json:"vout"`
}

type esploraStatus struct {
	Confirmed   bool  `json:"confirmed"`
	BlockHeight int64 `json:"block_height"`
	BlockTime   int64 `json:"block_time"`
}

type esploraVin struct {
	Txid    string      `json:"txid"`
	Vout    int         `json:"vout"`
	Prevout esploraVout `json:"prevout"`
}

type esploraVout struct {
	ScriptpubkeyAddress string `json:"scriptpubkey_address"`
	Value               int64  `json:"value"`
}

func (t *esploraTransaction) toRecord() sync.TxRecord {
	rec := sync.TxRecord{
		Txid:        t.Txid,
		FeeSats:     t.Fee,
		Confirmed:   t.Status.Confirmed,
		BlockHeight: t.Status.BlockHeight,
		BlockTime:   t.Status.BlockTime,
	}
	for _, vin := range t.Vin {
		rec.Inputs = append(rec.Inputs, sync.TxInput{
			PrevTxid: vin.Txid,
			PrevVout: vin.Vout,
			Sats:     vin.Prevout.Value,
			Address:  vin.Prevout.ScriptpubkeyAddress,
		})
	}
	for i, vout := range t.Vout {
		rec.Outputs = append(rec.Outputs, sync.TxOutput{
			Vout:    i,
			Sats:    vout.Value,
			Address: vout.ScriptpubkeyAddress,
		})
	}
	return rec
}
