package sync

import "context"

// TxInput is a simplified input for sync purposes.
type TxInput struct {
	PrevTxid string
	PrevVout int
	Sats     int64
	Address  string
}

// TxOutput is a simplified output for sync purposes.
type TxOutput struct {
	Vout    int
	Sats    int64
	Address string
}

// TxRecord holds all data about a single transaction fetched from the blockchain.
type TxRecord struct {
	Txid        string
	BlockHeight int64
	BlockTime   int64 // unix epoch; 0 if unconfirmed
	Confirmed   bool
	FeeSats     int64
	Inputs      []TxInput
	Outputs     []TxOutput
}

// BlockchainBackend is the interface that sync backends must implement.
type BlockchainBackend interface {
	Name() string
	// GetTransactions returns all tx history for a single address.
	GetTransactions(ctx context.Context, address string) ([]TxRecord, error)
	// BlockHeight returns the current tip height.
	BlockHeight(ctx context.Context) (int64, error)
}
