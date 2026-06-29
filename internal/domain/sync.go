package domain

import "time"

type SyncJob struct {
	ID               string     `json:"id"`
	WalletID         string     `json:"wallet_id"`
	Backend          string     `json:"backend"`
	Status           string     `json:"status"` // pending | running | done | failed
	AddressesScanned int        `json:"addresses_scanned"`
	TxFound          int        `json:"tx_found"`
	ErrorMsg         string     `json:"error_message,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

type SyncState struct {
	WalletID        string    `json:"wallet_id"`
	LastSyncedAt    time.Time `json:"last_synced_at"`
	ReceiveGapStart int       `json:"receive_gap_start"`
	ChangeGapStart  int       `json:"change_gap_start"`
	BlockHeight     int64     `json:"block_height"`
}
