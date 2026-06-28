package domain

import "time"

// Label is BIP329-compatible.
type Label struct {
	ID        string    `json:"id"`
	WalletID  string    `json:"wallet_id"`
	Type      string    `json:"type"` // tx | addr | xpub | input | output
	Ref       string    `json:"ref"`
	Label     string    `json:"label"`
	Origin    string    `json:"origin,omitempty"`
	Spendable *bool     `json:"spendable,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ImportJob struct {
	ID              string     `json:"id"`
	WalletID        string     `json:"wallet_id"`
	Source          string     `json:"source"` // sparrow | nunchuk | bsms | bip329
	Filename        string     `json:"filename"`
	Status          string     `json:"status"` // pending | running | done | failed
	RecordsImported int        `json:"records_imported"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}
