package domain

import "time"

type UTXO struct {
	ID          string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	Txid        string    `json:"txid"`
	Vout        int       `json:"vout"`
	Sats        int64     `json:"sats"`
	Address     string    `json:"address"`
	BlockHeight int64     `json:"block_height"`
	BlockTime   time.Time `json:"block_time"`
	Spent       bool      `json:"spent"`
	SpentTxid   string    `json:"spent_txid,omitempty"`
	Label       string    `json:"label"`
}
