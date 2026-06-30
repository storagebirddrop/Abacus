package domain

import "time"

type Transaction struct {
	ID          string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	Txid        string    `json:"txid"`
	BlockHeight int64     `json:"block_height"`
	BlockHash   string    `json:"block_hash"`
	BlockTime   time.Time `json:"block_time"`
	FeeSats     int64     `json:"fee_sats"`
	Confirmed   bool      `json:"confirmed"`
	CreatedAt   time.Time `json:"created_at"`
}

// TxFilter holds the optional search/sort/filter parameters for a paginated
// transaction listing. Zero values mean "no filter" / sensible defaults.
type TxFilter struct {
	Search string // case-insensitive substring match on txid; "" = no search
	Status string // "confirmed" | "pending"; "" = any
	Sort   string // "date" (block_time) | "fee" (fee_sats); "" = date
	Dir    string // "asc" | "desc"; "" = desc
	Limit  int
	Offset int
}

type TransactionInput struct {
	ID            string `json:"id"`
	TransactionID string `json:"transaction_id"`
	PrevTxid      string `json:"prev_txid"`
	PrevVout      int    `json:"prev_vout"`
	Sats          int64  `json:"sats"`
	Address       string `json:"address"`
	Sequence      uint32 `json:"sequence"`
	IsMine        bool   `json:"is_mine"`

	// ParentTxid is set during import for linking; not persisted directly.
	ParentTxid string `json:"-"`
}

type TransactionOutput struct {
	ID            string `json:"id"`
	TransactionID string `json:"transaction_id"`
	Vout          int    `json:"vout"`
	Sats          int64  `json:"sats"`
	Address       string `json:"address"`
	ScriptPubkey  string `json:"script_pubkey"`
	IsMine        bool   `json:"is_mine"`

	// ParentTxid is set during import for linking; not persisted directly.
	ParentTxid string `json:"-"`
}
