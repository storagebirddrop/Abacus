package bitcoincore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer returns a stub JSON-RPC server that answers getblockcount,
// scantxoutset, getblockhash, getrawtransaction, and getblockheader with
// fixed responses shaped like real Bitcoin Core output for one address with
// one deposit tx. It asserts getrawtransaction is called with the block
// hash resolved from the scanned output's height, catching a regression
// back to the txindex-only (no blockhash) call shape.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		var result any
		switch req.Method {
		case "getblockcount":
			result = 800123
		case "scantxoutset":
			result = map[string]any{
				"unspents": []map[string]any{
					{"txid": "a", "vout": 0, "amount": 0.00001, "height": 800000},
				},
			}
		case "getblockhash":
			params, _ := req.Params.([]any)
			if len(params) != 1 || params[0] != float64(800000) {
				t.Fatalf("getblockhash: unexpected params %v", req.Params)
			}
			result = "hashA"
		case "getrawtransaction":
			params, _ := req.Params.([]any)
			if len(params) < 1 || params[0] != "a" {
				t.Fatalf("getrawtransaction: unexpected params %v", req.Params)
			}
			if len(params) == 3 && params[2] != "hashA" {
				t.Fatalf("getrawtransaction: expected blockhash \"hashA\", got %v", params[2])
			}
			result = map[string]any{
				"txid":          "a",
				"confirmations": 5,
				"blockhash":     "hashA",
				"blocktime":     1700000000,
				"vin":           []map[string]any{{"coinbase": "01"}},
				"vout": []map[string]any{
					{"value": 0.00001, "scriptPubKey": map[string]any{"address": "bc1qreceiver"}},
				},
			}
		case "getblockheader":
			result = map[string]any{"height": 800000}
		default:
			t.Fatalf("unexpected method %q", req.Method)
		}
		resultJSON, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(rpcResponse{Result: resultJSON})
	}))
}

func TestBackend_BlockHeight(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	b := New(srv.URL, "user", "pass")
	height, err := b.BlockHeight(context.Background())
	if err != nil {
		t.Fatalf("BlockHeight: %v", err)
	}
	if height != 800123 {
		t.Errorf("expected height 800123, got %d", height)
	}
}

func TestBackend_GetTransactions(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	b := New(srv.URL, "user", "pass")
	records, err := b.GetTransactions(context.Background(), "bc1qreceiver")
	if err != nil {
		t.Fatalf("GetTransactions: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records[0]
	if rec.Txid != "a" || !rec.Confirmed || rec.BlockHeight != 800000 {
		t.Errorf("unexpected record: %+v", rec)
	}
	if len(rec.Outputs) != 1 || rec.Outputs[0].Address != "bc1qreceiver" || rec.Outputs[0].Sats != 1000 {
		t.Errorf("unexpected outputs: %+v", rec.Outputs)
	}
	// Coinbase input is skipped, so fee cannot be computed (haveAllInputs is
	// still true here since coinbase inputs are explicitly excluded).
	if len(rec.Inputs) != 0 {
		t.Errorf("expected no inputs for a coinbase tx, got %+v", rec.Inputs)
	}
}

func TestBackend_Name(t *testing.T) {
	if (&Backend{}).Name() != "bitcoincore" {
		t.Error("expected Name() to return \"bitcoincore\"")
	}
}
