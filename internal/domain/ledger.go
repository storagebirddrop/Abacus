package domain

import "time"

type EntryType string
type Category string

const (
	EntryTypeDebit  EntryType = "debit"
	EntryTypeCredit EntryType = "credit"

	CategoryIncome     Category = "income"
	CategoryExpense    Category = "expense"
	CategoryTransfer   Category = "transfer"
	CategoryExchange   Category = "exchange"
	CategoryMining     Category = "mining"
	CategoryDonation   Category = "donation"
	CategorySalary     Category = "salary"
	CategoryGift       Category = "gift"
	CategoryCoinJoin   Category = "coinjoin"
	CategoryLightning  Category = "lightning"
	CategoryCorrection Category = "correction"
	CategoryFee        Category = "fee"
	CategoryUnknown    Category = "unknown"
)

// LedgerEntry is immutable — never updated after creation.
type LedgerEntry struct {
	ID               string    `json:"id"`
	WalletID         string    `json:"wallet_id"`
	TransactionID    string    `json:"transaction_id"`
	Type             EntryType `json:"type"`
	Sats             int64     `json:"sats"`
	FiatAmount       int64     `json:"fiat_amount"`  // stored as cents
	FiatCurrency     string    `json:"fiat_currency"` // e.g. "EUR"
	PriceSnapshotID  string    `json:"price_snapshot_id,omitempty"`
	Category         Category  `json:"category"`
	CounterpartyID   string    `json:"counterparty_id,omitempty"`
	Note             string    `json:"note"`
	CreatedAt        time.Time `json:"created_at"`
}

// JournalEntry records every metadata change to a LedgerEntry.
type JournalEntry struct {
	ID           string    `json:"id"`
	LedgerEntryID string   `json:"ledger_entry_id"`
	FieldChanged string    `json:"field_changed"`
	OldValue     string    `json:"old_value"`
	NewValue     string    `json:"new_value"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

type Counterparty struct {
	ID       string    `json:"id"`
	WalletID string    `json:"wallet_id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"` // exchange | merchant | self | unknown
	Note     string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}
