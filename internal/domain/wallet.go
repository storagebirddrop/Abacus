package domain

import "time"

type WalletType string
type Network string
type WalletSource string

const (
	WalletTypeSinglesig WalletType = "singlesig"
	WalletTypeMultisig  WalletType = "multisig"

	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
	NetworkSignet  Network = "signet"

	WalletSourceSparrow WalletSource = "sparrow"
	WalletSourceNunchuk WalletSource = "nunchuk"
	WalletSourceBSMS    WalletSource = "bsms"
	WalletSourceManual  WalletSource = "manual"
)

type Wallet struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Descriptor  string       `json:"descriptor"`
	Fingerprint string       `json:"fingerprint"`
	Type        WalletType   `json:"type"`
	Network     Network      `json:"network"`
	Source      WalletSource `json:"source"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Address struct {
	ID             string    `json:"id"`
	WalletID       string    `json:"wallet_id"`
	Address        string    `json:"address"`
	DerivationPath string    `json:"derivation_path"`
	Type           string    `json:"type"` // receive | change
	Label          string    `json:"label"`
	CreatedAt      time.Time `json:"created_at"`
}
