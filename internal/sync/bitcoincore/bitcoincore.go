// Package bitcoincore implements the BlockchainBackend interface against a
// self-hosted Bitcoin Core node's JSON-RPC interface.
//
// It uses scantxoutset to discover unspent outputs for an address (no
// -txindex or wallet import required — a synced node with a recent UTXO set
// snapshot is enough) and getrawtransaction/getblockheader to fetch full
// transaction and input details. Reconstructing full history for an address
// that has since had all its outputs spent elsewhere in the wallet requires
// the node to be running with -txindex=1, since getrawtransaction otherwise
// only resolves transactions already indexed by wallet or mempool.
package bitcoincore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/storagebirddrop/abacus/internal/sync"
)

// Backend queries a Bitcoin Core node via JSON-RPC.
type Backend struct {
	rpcURL     string
	user       string
	pass       string
	httpClient *http.Client
}

func New(rpcURL, user, pass string) *Backend {
	return &Backend{
		rpcURL:     rpcURL,
		user:       user,
		pass:       pass,
		httpClient: &http.Client{},
	}
}

func (b *Backend) Name() string { return "bitcoincore" }

func (b *Backend) BlockHeight(ctx context.Context) (int64, error) {
	var height int64
	if err := b.call(ctx, "getblockcount", nil, &height); err != nil {
		return 0, err
	}
	return height, nil
}

func (b *Backend) GetTransactions(ctx context.Context, address string) ([]sync.TxRecord, error) {
	var scan scanTxOutSetResult
	desc := fmt.Sprintf("addr(%s)", address)
	if err := b.call(ctx, "scantxoutset", []any{"start", []string{desc}}, &scan); err != nil {
		return nil, fmt.Errorf("scantxoutset: %w", err)
	}

	seen := make(map[string]bool)
	records := make([]sync.TxRecord, 0, len(scan.Unspents))
	for _, u := range scan.Unspents {
		if seen[u.Txid] {
			continue
		}
		seen[u.Txid] = true
		rec, err := b.fetchTxRecord(ctx, u.Txid)
		if err != nil {
			return nil, fmt.Errorf("fetch tx %s: %w", u.Txid, err)
		}
		records = append(records, rec)
	}
	return records, nil
}

func (b *Backend) fetchTxRecord(ctx context.Context, txid string) (sync.TxRecord, error) {
	var raw rawTransaction
	if err := b.call(ctx, "getrawtransaction", []any{txid, true}, &raw); err != nil {
		return sync.TxRecord{}, err
	}

	rec := sync.TxRecord{
		Txid:      raw.Txid,
		Confirmed: raw.Confirmations > 0,
		BlockTime: raw.BlockTime,
	}
	if raw.BlockHash != "" {
		var header blockHeader
		if err := b.call(ctx, "getblockheader", []any{raw.BlockHash}, &header); err == nil {
			rec.BlockHeight = header.Height
		}
	}

	var totalIn, totalOut int64
	haveAllInputs := true
	for _, vin := range raw.Vin {
		if vin.Coinbase != "" {
			continue
		}
		var prev rawTransaction
		if err := b.call(ctx, "getrawtransaction", []any{vin.Txid, true}, &prev); err != nil || vin.Vout >= len(prev.Vout) {
			haveAllInputs = false
			continue
		}
		out := prev.Vout[vin.Vout]
		sats := btcToSats(out.Value)
		totalIn += sats
		rec.Inputs = append(rec.Inputs, sync.TxInput{
			PrevTxid: vin.Txid,
			PrevVout: vin.Vout,
			Sats:     sats,
			Address:  out.addressOrEmpty(),
		})
	}
	for i, vout := range raw.Vout {
		sats := btcToSats(vout.Value)
		totalOut += sats
		rec.Outputs = append(rec.Outputs, sync.TxOutput{
			Vout:    i,
			Sats:    sats,
			Address: vout.addressOrEmpty(),
		})
	}
	if haveAllInputs && totalIn > totalOut {
		rec.FeeSats = totalIn - totalOut
	}
	return rec, nil
}

func btcToSats(btc float64) int64 {
	return int64(btc*1e8 + 0.5)
}

// ---------- JSON-RPC plumbing ----------

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (b *Backend) call(ctx context.Context, method string, params any, out any) error {
	if params == nil {
		params = []any{}
	}
	body, err := json.Marshal(rpcRequest{JSONRPC: "1.0", ID: "abacus", Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.user != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(b.user + ":" + b.pass))
		req.Header.Set("Authorization", "Basic "+auth)
	}
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		return fmt.Errorf("bitcoincore %s: HTTP %d", method, resp.StatusCode)
	}
	var rr rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return err
	}
	if rr.Error != nil {
		return fmt.Errorf("bitcoincore %s: %s (code %d)", method, rr.Error.Message, rr.Error.Code)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(rr.Result, out)
}

// ---------- JSON shapes ----------

type scanTxOutSetResult struct {
	Unspents []scanUnspent `json:"unspents"`
}

type scanUnspent struct {
	Txid   string  `json:"txid"`
	Vout   int     `json:"vout"`
	Amount float64 `json:"amount"`
	Height int64   `json:"height"`
}

type rawTransaction struct {
	Txid          string    `json:"txid"`
	Confirmations int64     `json:"confirmations"`
	BlockHash     string    `json:"blockhash"`
	BlockTime     int64     `json:"blocktime"`
	Vin           []rawVin  `json:"vin"`
	Vout          []rawVout `json:"vout"`
}

type rawVin struct {
	Txid     string `json:"txid"`
	Vout     int    `json:"vout"`
	Coinbase string `json:"coinbase"`
}

type rawVout struct {
	Value        float64         `json:"value"`
	ScriptPubKey rawScriptPubKey `json:"scriptPubKey"`
}

type rawScriptPubKey struct {
	Address   string   `json:"address"`
	Addresses []string `json:"addresses"`
}

func (v rawVout) addressOrEmpty() string {
	if v.ScriptPubKey.Address != "" {
		return v.ScriptPubKey.Address
	}
	if len(v.ScriptPubKey.Addresses) > 0 {
		return v.ScriptPubKey.Addresses[0]
	}
	return ""
}

type blockHeader struct {
	Height int64 `json:"height"`
}
