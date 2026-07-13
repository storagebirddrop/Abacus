package accounting

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/storagebirddrop/abacus/internal/domain"
)

// unrealisedGainFiat mark-to-market values a held (undisposed) lot at
// currentPriceCents and returns the gain/loss vs. its recorded cost.
// Returns 0 if the current price is unknown.
func unrealisedGainFiat(costSats, costFiat, currentPriceCents int64) int64 {
	if currentPriceCents == 0 {
		return 0
	}
	return satsToFiat(costSats, currentPriceCents) - costFiat
}

// PriceLookup returns the BTC price in fiat cents for the given currency at time t.
// Returns 0 if no price data is available (proceeds/gain will be stored as 0).
type PriceLookup func(currency string, t time.Time) int64

// AccountingSummary is the result of aggregating all cost basis records.
type AccountingSummary struct {
	WalletID           string                `json:"wallet_id"`
	Method             domain.CostBasisMethod `json:"method"`
	FiatCurrency       string                `json:"fiat_currency"`
	TotalCostSats      int64                 `json:"total_cost_sats"`
	TotalCostFiat      int64                 `json:"total_cost_fiat"`
	UnrealisedGainFiat int64                 `json:"unrealised_gain_fiat"`
	RealisedGainFiat   int64                 `json:"realised_gain_fiat"`
	ComputedAt         time.Time             `json:"computed_at"`
}

// satsToFiat converts satoshis to fiat cents given a BTC price in cents.
// Returns 0 when price is 0 (unknown).
func satsToFiat(sats, pricePerBTCCents int64) int64 {
	if pricePerBTCCents == 0 {
		return 0
	}
	// sats / 1e8 * price  →  sats * price / 1e8  (integer arithmetic, truncates)
	return sats * pricePerBTCCents / 100_000_000
}

// utxoRepo is the minimal interface the Service needs.
type utxoRepo interface {
	ListByWallet(ctx context.Context, walletID string, unspentOnly bool) ([]*domain.UTXO, error)
}

type cbRepo interface {
	UpsertWithTx(ctx context.Context, tx *sql.Tx, cb *domain.CostBasisRecord) error
	DeleteByWallet(ctx context.Context, tx *sql.Tx, walletID string) error
	ListByWallet(ctx context.Context, walletID string) ([]*domain.CostBasisRecord, error)
}

type priceRepo interface {
	GetClosest(ctx context.Context, currency string, t time.Time) (*domain.PriceSnapshot, error)
}

type txDB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type txRepo interface {
	List(ctx context.Context, walletID string, limit, offset int) ([]*domain.Transaction, int, error)
}

type exchangeTradeRepo interface {
	ListByWallet(ctx context.Context, walletID string) ([]*domain.ExchangeTrade, error)
}

// Service orchestrates cost basis calculations.
type Service struct {
	db                txDB
	utxoRepo          utxoRepo
	cbRepo            cbRepo
	priceRepo         priceRepo
	txRepo            txRepo
	exchangeTradeRepo exchangeTradeRepo
}

func NewService(db txDB, utxoRepo utxoRepo, cbRepo cbRepo, priceRepo priceRepo, txRepo txRepo, exchangeTradeRepo exchangeTradeRepo) *Service {
	return &Service{
		db:                db,
		utxoRepo:          utxoRepo,
		cbRepo:            cbRepo,
		priceRepo:         priceRepo,
		txRepo:            txRepo,
		exchangeTradeRepo: exchangeTradeRepo,
	}
}

// Run computes cost basis for all UTXOs (or exchange trades) in walletID using
// the chosen method, deletes the previous records, and persists the new ones
// atomically.
func (s *Service) Run(ctx context.Context, walletID string, method domain.CostBasisMethod, currency string) error {
	// Load exchange trades first. Exchange wallets have no UTXOs; on-chain
	// wallets have no trades. Mixed wallets use UTXO accounting only.
	trades, err := s.exchangeTradeRepo.ListByWallet(ctx, walletID)
	if err != nil {
		return fmt.Errorf("load exchange trades: %w", err)
	}

	var records []domain.CostBasisRecord

	if len(trades) > 0 {
		// Exchange wallet: use trade-based cost basis calculation.
		switch method {
		case domain.MethodFIFO, domain.MethodAvgCost, domain.MethodLIFO,
			domain.MethodHIFO, domain.MethodSpecificID, domain.MethodSection104:
			records = RunExchangeFIFO(walletID, trades, currency, method)
		default:
			return fmt.Errorf("unknown method: %s", method)
		}
	} else {
		// On-chain wallet: use UTXO-based cost basis calculation.
		all, err := s.utxoRepo.ListByWallet(ctx, walletID, false)
		if err != nil {
			return fmt.Errorf("load utxos: %w", err)
		}
		utxos := make([]domain.UTXO, len(all))
		for i, u := range all {
			utxos[i] = *u
		}

		spendTimes, err := s.buildSpendTimes(ctx, walletID)
		if err != nil {
			return fmt.Errorf("build spend times: %w", err)
		}

		priceFn := s.makePriceLookup(ctx)

		switch method {
		case domain.MethodFIFO:
			records = RunFIFO(walletID, utxos, spendTimes, priceFn, currency)
		case domain.MethodAvgCost:
			records = RunAvgCost(walletID, utxos, spendTimes, priceFn, currency)
		case domain.MethodLIFO:
			records = RunLIFO(walletID, utxos, spendTimes, priceFn, currency)
		case domain.MethodHIFO:
			records = RunHIFO(walletID, utxos, spendTimes, priceFn, currency)
		case domain.MethodSpecificID:
			records = RunSpecificID(walletID, utxos, spendTimes, priceFn, currency, nil)
		case domain.MethodSection104:
			records = RunSection104(walletID, utxos, spendTimes, priceFn, currency)
		default:
			return fmt.Errorf("unknown method: %s", method)
		}
	}

	// Persist atomically: delete old → insert new.
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := s.cbRepo.DeleteByWallet(ctx, dbTx, walletID); err != nil {
		_ = dbTx.Rollback()
		return fmt.Errorf("delete old records: %w", err)
	}
	for i := range records {
		if err := s.cbRepo.UpsertWithTx(ctx, dbTx, &records[i]); err != nil {
			_ = dbTx.Rollback()
			return fmt.Errorf("upsert record: %w", err)
		}
	}
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Summary aggregates the stored cost basis records into a portfolio summary.
func (s *Service) Summary(ctx context.Context, walletID string) (*AccountingSummary, error) {
	records, err := s.cbRepo.ListByWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &AccountingSummary{WalletID: walletID, ComputedAt: time.Now().UTC()}, nil
	}

	sum := &AccountingSummary{
		WalletID:     walletID,
		Method:       records[0].Method,
		FiatCurrency: records[0].FiatCurrency,
		ComputedAt:   time.Now().UTC(),
	}

	// Held (undisposed) lots have no GainFiat — RunFIFO/RunAvgCost/etc. only
	// set it on disposal. Mark them to market against the current price so
	// "unrealised gain" isn't silently always zero.
	var currentPrice int64
	if snap, err := s.priceRepo.GetClosest(ctx, sum.FiatCurrency, sum.ComputedAt); err == nil && snap != nil {
		currentPrice = snap.PriceFiat
	}

	for _, r := range records {
		if r.DisposedAt != nil && r.GainFiat != nil {
			sum.RealisedGainFiat += *r.GainFiat
			continue
		}
		// Still held: cost basis and unrealised gain both count against
		// current holdings only, not coins already disposed of.
		sum.TotalCostSats += r.CostSats
		sum.TotalCostFiat += r.CostFiat
		sum.UnrealisedGainFiat += unrealisedGainFiat(r.CostSats, r.CostFiat, currentPrice)
	}
	return sum, nil
}

// buildSpendTimes maps each spending txid to its block time, derived from the
// wallet's transaction list (paginated in batches).
func (s *Service) buildSpendTimes(ctx context.Context, walletID string) (map[string]time.Time, error) {
	m := map[string]time.Time{}
	offset := 0
	const batch = 500
	for {
		txs, _, err := s.txRepo.List(ctx, walletID, batch, offset)
		if err != nil {
			return nil, err
		}
		for _, tx := range txs {
			m[tx.Txid] = tx.BlockTime
		}
		if len(txs) < batch {
			break
		}
		offset += batch
	}
	return m, nil
}

// makePriceLookup returns a PriceLookup backed by the price snapshot repo.
// On a cache miss (no snapshot or error) it returns 0 so the caller stores 0.
func (s *Service) makePriceLookup(ctx context.Context) PriceLookup {
	cache := map[string]int64{} // "currency:unix" → price
	return func(currency string, t time.Time) int64 {
		if t.IsZero() {
			return 0
		}
		key := fmt.Sprintf("%s:%d", currency, t.Unix())
		if p, ok := cache[key]; ok {
			return p
		}
		snap, err := s.priceRepo.GetClosest(ctx, currency, t)
		if err != nil || snap == nil {
			cache[key] = 0
			return 0
		}
		cache[key] = snap.PriceFiat
		return snap.PriceFiat
	}
}
