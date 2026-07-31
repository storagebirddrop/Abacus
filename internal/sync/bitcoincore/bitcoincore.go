// Package bitcoincore implements the BlockchainBackend interface against a
// self-hosted Bitcoin Core node's JSON-RPC interface.
//
// It uses scantxoutset to discover an address's current unspent outputs (no
// -txindex or wallet import required — a synced node with a recent UTXO set
// snapshot is enough) and getrawtransaction/getblockheader to fetch full
// transaction and input details, resolving each found transaction's block
// hash via getblockhash so getrawtransaction works without -txindex.
//
// This only discovers currently-unspent outputs: an address whose outputs
// have all since been spent elsewhere returns no history at all, since
// scantxoutset is a UTXO-set scan, not a full transaction index. A node
// running with -txindex=1 does not change this — it only helps resolve the
// previous transaction for an already-found input's spend details.
package bitcoincore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		httpClient: &http.Client{Timeout: 5 * time.Minute}, // scantxoutset can run long
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
		rec, err := b.fetchTxRecord(ctx, u.Txid, u.Height)
		if err != nil {
			return nil, fmt.Errorf("fetch tx %s: %w", u.Txid, err)
		}
		records = append(records, rec)
	}
	return records, nil
}

// fetchTxRecord fetches full transaction detail for txid, known to have been
// confirmed at the given height (0 if unconfirmed, per scantxoutset). Passing
// the resolved block hash to getrawtransaction lets it find the transaction
// on a node without -txindex — without a block hash, getrawtransaction can
// only resolve mempool/wallet-known transactions.
func (b *Backend) fetchTxRecord(ctx context.Context, txid string, height int64) (sync.TxRecord, error) {
	var blockHash string
	if height > 0 {
		if err := b.call(ctx, "getblockhash", []any{height}, &blockHash); err != nil {
			return sync.TxRecord{}, fmt.Errorf("getblockhash: %w", err)
		}
	}

	params := []any{txid, true}
	if blockHash != "" {
		params = append(params, blockHash)
	}
	var raw rawTransaction
	if err := b.call(ctx, "getrawtransaction", params, &raw); err != nil {
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
		// Best-effort: the previous tx's block hash isn't known here, so this
		// lookup only succeeds if the node can resolve it without one
		// (mempool/wallet-known, or a -txindex node).
		var prev rawTransaction
		if err := b.call(ctx, "getrawtransaction", []any{vin.Txid, true}, &prev); err != nil || vin.Vout >= len(prev.Vout) {
			haveAllInputs = false
			continue
		}
		out := prev.Vout[vin.Vout]
		sats, err := parseSats(out.Value)
		if err != nil {
			return sync.TxRecord{}, fmt.Errorf("parse input value: %w", err)
		}
		totalIn += sats
		rec.Inputs = append(rec.Inputs, sync.TxInput{
			PrevTxid: vin.Txid,
			PrevVout: vin.Vout,
			Sats:     sats,
			Address:  out.addressOrEmpty(),
		})
	}
	for i, vout := range raw.Vout {
		sats, err := parseSats(vout.Value)
		if err != nil {
			return sync.TxRecord{}, fmt.Errorf("parse output value: %w", err)
		}
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

// parseSats converts a Bitcoin Core JSON-RPC decimal BTC amount (e.g.
// "0.00001234") to integer satoshis via exact string arithmetic — no
// float64 round-trip, since json.Number preserves the literal decimal text
// Bitcoin Core emits (always exactly 8 fraction digits).
func parseSats(n json.Number) (int64, error) {
	s := n.String()
	if s == "" {
		return 0, nil
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	whole, frac, _ := strings.Cut(s, ".")
	if len(frac) > 8 {
		frac = frac[:8]
	}
	for len(frac) < 8 {
		frac += "0"
	}
	if whole == "" {
		whole = "0"
	}
	wholeSats, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", n, err)
	}
	fracSats, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", n, err)
	}
	sats := wholeSats*1e8 + fracSats
	if neg {
		sats = -sats
	}
	return sats, nil
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
	Txid   string      `json:"txid"`
	Vout   int         `json:"vout"`
	Amount json.Number `json:"amount"`
	Height int64       `json:"height"`
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
	Value        json.Number     `json:"value"`
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
