package domain

import "time"

type CostBasisMethod string

const (
	MethodFIFO       CostBasisMethod = "fifo"
	MethodAvgCost    CostBasisMethod = "avgcost"
	MethodLIFO       CostBasisMethod = "lifo"
	MethodHIFO       CostBasisMethod = "hifo"
	MethodSpecificID  CostBasisMethod = "specificid"
	MethodSection104  CostBasisMethod = "section104"
)

type CostBasisRecord struct {
	ID           string          `json:"id"`
	WalletID     string          `json:"wallet_id"`
	Txid         string          `json:"txid"`
	Vout         int             `json:"vout"`
	AcquiredAt   time.Time       `json:"acquired_at"`
	CostSats     int64           `json:"cost_sats"`
	CostFiat     int64           `json:"cost_fiat"`   // cents
	FiatCurrency string          `json:"fiat_currency"`
	Method       CostBasisMethod `json:"method"`
	DisposedAt   *time.Time      `json:"disposed_at,omitempty"`
	ProceedsFiat *int64          `json:"proceeds_fiat,omitempty"` // cents
	GainFiat     *int64          `json:"gain_fiat,omitempty"`     // cents
}

type PriceSnapshot struct {
	ID         string    `json:"id"`
	Currency   string    `json:"currency"`
	PriceFiat  int64     `json:"price_fiat"` // price per BTC in cents
	Source     string    `json:"source"`     // e.g. "manual", "coingecko"
	Timestamp  time.Time `json:"timestamp"`
}
