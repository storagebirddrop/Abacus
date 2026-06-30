// Package electrum implements the BlockchainBackend interface using the Electrum TCP JSON-RPC protocol.
// Connects to a public or self-hosted Electrum server (stratum protocol).
package electrum

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	btcsync "github.com/storagebirddrop/abacus/internal/sync"
)

// Backend implements BlockchainBackend via Electrum stratum TCP protocol.
type Backend struct {
	host string
	port int
	tls  bool

	mu      sync.Mutex
	conn    net.Conn
	reader  *bufio.Reader
	nextID  int
}

func New(host string, port int, useTLS bool) *Backend {
	return &Backend{host: host, port: port, tls: useTLS}
}

func (b *Backend) Name() string { return "electrum" }

func (b *Backend) connect() error {
	addr := net.JoinHostPort(b.host, fmt.Sprintf("%d", b.port))
	var conn net.Conn
	var err error
	if b.tls {
		conn, err = tls.Dial("tcp", addr, &tls.Config{MinVersion: tls.VersionTLS12})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	b.conn = conn
	b.reader = bufio.NewReader(conn)
	return nil
}

func (b *Backend) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn == nil {
		if err := b.connect(); err != nil {
			return nil, fmt.Errorf("electrum connect: %w", err)
		}
	}

	b.nextID++
	id := b.nextID

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(req)
	data = append(data, '\n')

	if dl, ok := ctx.Deadline(); ok {
		_ = b.conn.SetDeadline(dl)
	} else {
		_ = b.conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	if _, err := b.conn.Write(data); err != nil {
		b.conn.Close()
		b.conn = nil
		return nil, fmt.Errorf("electrum write: %w", err)
	}

	line, err := b.reader.ReadBytes('\n')
	if err != nil {
		b.conn.Close()
		b.conn = nil
		return nil, fmt.Errorf("electrum read: %w", err)
	}

	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("electrum parse: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("electrum error: %s", resp.Error.Message)
	}
	return resp.Result, nil
}

func (b *Backend) BlockHeight(ctx context.Context) (int64, error) {
	raw, err := b.call(ctx, "blockchain.headers.subscribe", []any{})
	if err != nil {
		return 0, err
	}
	var resp struct {
		Height int64 `json:"height"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return 0, err
	}
	return resp.Height, nil
}

func (b *Backend) GetTransactions(ctx context.Context, address string) ([]btcsync.TxRecord, error) {
	scripthash, err := addressToScripthash(address)
	if err != nil {
		return nil, err
	}

	raw, err := b.call(ctx, "blockchain.scripthash.get_history", []any{scripthash})
	if err != nil {
		return nil, err
	}
	var history []struct {
		TxHash string `json:"tx_hash"`
		Height int64  `json:"height"`
	}
	if err := json.Unmarshal(raw, &history); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var records []btcsync.TxRecord
	for _, h := range history {
		if seen[h.TxHash] {
			continue
		}
		seen[h.TxHash] = true

		txRaw, err := b.call(ctx, "blockchain.transaction.get", []any{h.TxHash, true})
		if err != nil {
			continue
		}
		var tx electrumTx
		if err := json.Unmarshal(txRaw, &tx); err != nil {
			continue
		}
		rec := btcsync.TxRecord{
			Txid:        tx.Txid,
			FeeSats:     0, // electrum verbose doesn't always include fee
			Confirmed:   h.Height > 0,
			BlockHeight: h.Height,
			BlockTime:   tx.Blocktime,
		}
		for _, vin := range tx.Vin {
			rec.Inputs = append(rec.Inputs, btcsync.TxInput{
				PrevTxid: vin.Txid,
				PrevVout: vin.Vout,
			})
		}
		for _, vout := range tx.Vout {
			addr := ""
			if len(vout.ScriptPubKey.Addresses) > 0 {
				addr = vout.ScriptPubKey.Addresses[0]
			} else if vout.ScriptPubKey.Address != "" {
				addr = vout.ScriptPubKey.Address
			}
			rec.Outputs = append(rec.Outputs, btcsync.TxOutput{
				Vout:    vout.N,
				Sats:    btcStringToSats(vout.Value),
				Address: addr,
			})
		}
		records = append(records, rec)
	}
	return records, nil
}

// addressToScripthash converts a Bitcoin address to an Electrum scripthash.
// The scripthash is SHA256(scriptPubKey) with bytes reversed, in hex.
func addressToScripthash(address string) (string, error) {
	script, err := addressToScript(address)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(script)
	// Reverse bytes.
	for i, j := 0, len(h)-1; i < j; i, j = i+1, j-1 {
		h[i], h[j] = h[j], h[i]
	}
	return hex.EncodeToString(h[:]), nil
}

// addressToScript converts a Bitcoin address string to its scriptPubKey bytes.
// Supports P2WPKH (bc1q...), P2SH (3...), and P2PKH (1...).
func addressToScript(address string) ([]byte, error) {
	// Detect address type by prefix / length.
	switch {
	case strings.HasPrefix(address, "bc1q") || strings.HasPrefix(address, "tb1q"):
		// P2WPKH: OP_0 <20-byte-hash>
		witProg, err := decodeBech32(address)
		if err != nil {
			return nil, err
		}
		return append([]byte{0x00, 0x14}, witProg...), nil

	case strings.HasPrefix(address, "3") || strings.HasPrefix(address, "2"):
		// P2SH: OP_HASH160 <20-byte-hash> OP_EQUAL
		hash, err := decodeBase58Check(address)
		if err != nil {
			return nil, err
		}
		return append([]byte{0xa9, 0x14}, append(hash, 0x87)...), nil

	case strings.HasPrefix(address, "1") || strings.HasPrefix(address, "m") || strings.HasPrefix(address, "n"):
		// P2PKH: OP_DUP OP_HASH160 <20-byte-hash> OP_EQUALVERIFY OP_CHECKSIG
		hash, err := decodeBase58Check(address)
		if err != nil {
			return nil, err
		}
		return append([]byte{0x76, 0xa9, 0x14}, append(hash, 0x88, 0xac)...), nil
	}
	return nil, fmt.Errorf("unsupported address format: %s", address[:min(10, len(address))])
}

// decodeBech32 decodes a bech32 segwit address and returns the witness program bytes.
// This is a minimal implementation for P2WPKH (version 0, 20 bytes).
func decodeBech32(addr string) ([]byte, error) {
	// Split at the last '1' separator.
	sep := strings.LastIndex(addr, "1")
	if sep < 2 {
		return nil, fmt.Errorf("invalid bech32: %s", addr)
	}
	data := addr[sep+1:]
	// Strip 6-char checksum.
	if len(data) < 8 {
		return nil, fmt.Errorf("bech32 too short")
	}
	// First char is witness version; skip it.
	data = data[1 : len(data)-6]

	// Convert from 5-bit groups to 8-bit bytes (base32 → base256).
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	bits := 0
	value := 0
	var out []byte
	for _, c := range data {
		idx := strings.IndexRune(charset, c)
		if idx < 0 {
			return nil, fmt.Errorf("invalid bech32 char")
		}
		value = (value << 5) | idx
		bits += 5
		for bits >= 8 {
			bits -= 8
			out = append(out, byte(value>>bits))
			value &= (1 << bits) - 1
		}
	}
	if len(out) != 20 {
		return nil, fmt.Errorf("unexpected witness program length: %d", len(out))
	}
	return out, nil
}

// decodeBase58Check decodes a base58check address and returns the 20-byte hash.
func decodeBase58Check(addr string) ([]byte, error) {
	const base58Chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	var decoded [25]byte
	var bigInt [4]uint32
	for _, c := range addr {
		idx := strings.IndexRune(base58Chars, c)
		if idx < 0 {
			return nil, fmt.Errorf("invalid base58 char")
		}
		carry := uint64(idx)
		for j := 3; j >= 0; j-- {
			carry += uint64(bigInt[j]) * 58
			bigInt[j] = uint32(carry & 0xffffffff)
			carry >>= 32
		}
	}
	binary.BigEndian.PutUint32(decoded[0:], bigInt[0])
	binary.BigEndian.PutUint32(decoded[4:], bigInt[1])
	binary.BigEndian.PutUint32(decoded[8:], bigInt[2])
	binary.BigEndian.PutUint32(decoded[12:], bigInt[3])
	// decoded[0] = version byte, decoded[1:21] = hash, decoded[21:25] = checksum
	if len(decoded) < 21 {
		return nil, fmt.Errorf("decoded address too short")
	}
	return decoded[1:21], nil
}

// btcStringToSats converts a decimal BTC amount (as the raw JSON number string,
// e.g. "0.29") to satoshis exactly, avoiding float rounding. The old float path
// `int64(btc * 1e8)` truncates — 0.29 BTC became 28999999, losing a satoshi.
func btcStringToSats(v json.Number) int64 {
	s := string(v)
	if s == "" {
		return 0
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	intPart, fracPart, _ := strings.Cut(s, ".")
	// Normalise the fractional part to exactly 8 digits (satoshi precision).
	if len(fracPart) > 8 {
		fracPart = fracPart[:8]
	}
	for len(fracPart) < 8 {
		fracPart += "0"
	}

	var sats int64
	if intPart != "" {
		if n, err := strconv.ParseInt(intPart, 10, 64); err == nil {
			sats = n * 100_000_000
		}
	}
	if f, err := strconv.ParseInt(fracPart, 10, 64); err == nil {
		sats += f
	}
	if neg {
		sats = -sats
	}
	return sats
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------- JSON shapes ----------

type electrumTx struct {
	Txid      string         `json:"txid"`
	Blocktime int64          `json:"blocktime"`
	Vin       []electrumVin  `json:"vin"`
	Vout      []electrumVout `json:"vout"`
}

type electrumVin struct {
	Txid string `json:"txid"`
	Vout int    `json:"vout"`
}

type electrumVout struct {
	Value       json.Number       `json:"value"`
	N           int                `json:"n"`
	ScriptPubKey electrumScriptKey `json:"scriptPubKey"`
}

type electrumScriptKey struct {
	Address   string   `json:"address"`
	Addresses []string `json:"addresses"`
}
