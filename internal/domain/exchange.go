package domain

import "time"

type TradeType string

const (
	TradeTypeBuy              TradeType = "buy"
	TradeTypeSell             TradeType = "sell"
	TradeTypeFee              TradeType = "fee"
	TradeTypeDeposit          TradeType = "deposit"
	TradeTypeWithdrawal       TradeType = "withdrawal"
	TradeTypeLightningReceive TradeType = "lightning_receive"
	TradeTypeLightningSend    TradeType = "lightning_send"
)

// ExchangeTrade represents a single custodial exchange transaction where the
// fiat value is known exactly at trade time (unlike on-chain UTXOs).
type ExchangeTrade struct {
	ID           string    `json:"id"`
	WalletID     string    `json:"wallet_id"`
	TradeType    TradeType `json:"trade_type"`
	Exchange     string    `json:"exchange"`
	ExternalID   string    `json:"external_id,omitempty"` // exchange's own trade ID (dedup key)
	TradedAt     time.Time `json:"traded_at"`
	Sats         int64     `json:"sats"`          // BTC side in satoshis (positive for buys, negative for sells)
	FiatAmount   int64     `json:"fiat_amount"`   // fiat side in cents (absolute value)
	FiatCurrency string    `json:"fiat_currency"` // e.g. "EUR", "USD"
	FeeSats      int64     `json:"fee_sats"`
	FeeFiat      int64     `json:"fee_fiat"` // in cents
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

// IsAcquisition returns true for trade types that increase BTC holdings.
func (t *ExchangeTrade) IsAcquisition() bool {
	return t.TradeType == TradeTypeBuy ||
		t.TradeType == TradeTypeDeposit ||
		t.TradeType == TradeTypeLightningReceive
}

// IsDisposal returns true for trade types that decrease BTC holdings.
func (t *ExchangeTrade) IsDisposal() bool {
	return t.TradeType == TradeTypeSell ||
		t.TradeType == TradeTypeWithdrawal ||
		t.TradeType == TradeTypeLightningSend
}
